package discord

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultClientID is the fallback Discord Application Client ID used when no custom ID is configured.
	DefaultClientID  = "1121632048155742288"
	FallbackCoverURL = "https://raw.githubusercontent.com/Rafifdev/Velora/main/build/appicon.png"

	opHandshake uint32 = 0
	opFrame     uint32 = 1
	opClose     uint32 = 2
	opPing      uint32 = 3
	opPong      uint32 = 4

	maxFrameBytes       = 1 << 20
	ipcTimeout          = 3 * time.Second
	minSendInterval     = 1500 * time.Millisecond
	reconnectBackoff    = 10 * time.Second
	heartbeatInterval   = 30 * time.Second
	pauseClearThreshold = 10 * time.Minute
)

var (
	errDiscordUnavailable = errors.New("discord ipc unavailable")
	ytImgPathRe           = regexp.MustCompile(`(?i)^/(?:vi|vi_webp)/([^/]+)/`)
	googleSizeRe          = regexp.MustCompile(`(?i)=w\d+-h\d+[^&?]*`)
)

// TrackPresence describes the current track state synced from the player.
type TrackPresence struct {
	Title       string
	Artist      string
	Album       string
	CoverURL    string
	ExternalID  string
	IsPlaying   bool
	PositionSec float64
	DurationSec float64
}

type presenceSig struct {
	clientID   string
	title      string
	artist     string
	album      string
	coverURL   string
	externalID string
	isPlaying  bool
	startEpoch int64
	endEpoch   int64
}

type deadlineSetter interface {
	SetDeadline(t time.Time) error
}

// Client manages a background Discord IPC connection and updates Rich Presence asynchronously.
type Client struct {
	mu          sync.Mutex
	enabled     bool
	clientID    string
	desired     TrackPresence
	hasTrack    bool
	pausedSince time.Time
	lastPosAt   time.Time
	lastPosSec  float64
	dial        func() (io.ReadWriteCloser, error)

	wakeCh   chan struct{}
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
}

// NewClient creates and starts a background Discord Rich Presence client.
func NewClient(enabled bool, customClientID string) *Client {
	c := &Client{
		enabled:  enabled,
		clientID: resolveClientID(customClientID),
		dial:     openDiscordIPC,
		wakeCh:   make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	go c.loop()
	return c
}

func resolveClientID(custom string) string {
	if s := strings.TrimSpace(custom); s != "" {
		return s
	}
	if s := strings.TrimSpace(os.Getenv("VELORA_DISCORD_CLIENT_ID")); s != "" {
		return s
	}
	return DefaultClientID
}

func (c *Client) wake() {
	select {
	case c.wakeCh <- struct{}{}:
	default:
	}
}

// SetEnabled enables or disables Discord Rich Presence updates.
func (c *Client) SetEnabled(enabled bool) {
	c.mu.Lock()
	changed := c.enabled != enabled
	c.enabled = enabled
	c.mu.Unlock()
	if changed {
		c.wake()
	}
}

// SetClientID updates the Discord Application Client ID at runtime.
func (c *Client) SetClientID(clientID string) {
	resolved := resolveClientID(clientID)
	c.mu.Lock()
	changed := c.clientID != resolved
	c.clientID = resolved
	c.mu.Unlock()
	if changed {
		c.wake()
	}
}

// UpdatePresence updates the desired track and playback state. Non-blocking.
func (c *Client) UpdatePresence(p TrackPresence) {
	c.mu.Lock()
	hasTrack := strings.TrimSpace(p.Title) != ""
	if !hasTrack {
		c.pausedSince = time.Time{}
	} else if !p.IsPlaying {
		if c.pausedSince.IsZero() || c.desired.IsPlaying || c.desired.Title != p.Title {
			c.pausedSince = time.Now()
		}
	} else {
		c.pausedSince = time.Time{}
	}
	c.desired = p
	c.hasTrack = hasTrack
	c.lastPosSec = p.PositionSec
	c.lastPosAt = time.Now()
	c.mu.Unlock()
	c.wake()
}

// UpdateProgress updates playback position and wakes Discord RPC only on a seek jump or newly available duration.
func (c *Client) UpdateProgress(positionSec, durationSec float64) {
	c.mu.Lock()
	if !c.hasTrack {
		c.mu.Unlock()
		return
	}
	now := time.Now()
	durationChanged := durationSec > 0 && math.Abs(durationSec-c.desired.DurationSec) >= 2.0
	seekJumped := false
	if c.desired.IsPlaying && !c.lastPosAt.IsZero() {
		expected := c.lastPosSec + now.Sub(c.lastPosAt).Seconds()
		if math.Abs(positionSec-expected) >= 3.0 {
			seekJumped = true
		}
	}
	c.desired.PositionSec = positionSec
	if durationSec > 0 {
		c.desired.DurationSec = durationSec
	}
	c.lastPosSec = positionSec
	c.lastPosAt = now
	c.mu.Unlock()

	if durationChanged || seekJumped {
		c.wake()
	}
}

// Shutdown clears any active presence, closes the IPC pipe, and stops the worker.
func (c *Client) Shutdown() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	select {
	case <-c.doneCh:
	case <-time.After(800 * time.Millisecond):
	}
}

func (c *Client) loop() {
	defer close(c.doneCh)

	var (
		conn        io.ReadWriteCloser
		connectedID string
		lastSent    presenceSig
		hasSent     bool
		lastAttempt time.Time
		lastSendAt  time.Time
		retryTimer  *time.Timer
	)

	stopTimer := func() {
		if retryTimer != nil {
			retryTimer.Stop()
			retryTimer = nil
		}
	}
	defer stopTimer()

	scheduleWake := func(d time.Duration) {
		if d <= 0 {
			c.wake()
			return
		}
		if retryTimer == nil {
			retryTimer = time.NewTimer(d)
		} else {
			retryTimer.Reset(d)
		}
	}

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		c.mu.Lock()
		enabled := c.enabled
		clientID := c.clientID
		desired := c.desired
		hasTrack := c.hasTrack
		pausedSince := c.pausedSince
		lastPosAt := c.lastPosAt
		c.mu.Unlock()

		shouldClear := !enabled || !hasTrack || (!desired.IsPlaying && !pausedSince.IsZero() && time.Since(pausedSince) > pauseClearThreshold)

		if shouldClear {
			if conn != nil {
				if hasSent {
					_ = clearActivity(conn)
					hasSent = false
					lastSent = presenceSig{}
				}
				if !enabled {
					_ = conn.Close()
					conn = nil
					connectedID = ""
				}
			}
		} else {
			// Extrapolate current position when playing so timestamps stay accurate across rate-limit waits.
			effPos := desired.PositionSec
			if desired.IsPlaying && !lastPosAt.IsZero() {
				effPos += time.Since(lastPosAt).Seconds()
				if desired.DurationSec > 0 && effPos > desired.DurationSec {
					effPos = desired.DurationSec
				}
			}
			desired.PositionSec = effPos

			nowSec := time.Now().Unix()
			sig := buildSignature(clientID, desired, nowSec)
			needSend := !hasSent || sigChanged(lastSent, sig) || conn == nil

			if needSend {
				if conn != nil && connectedID != clientID {
					_ = clearActivity(conn)
					_ = conn.Close()
					conn = nil
					connectedID = ""
					hasSent = false
				}

				if conn == nil {
					if !lastAttempt.IsZero() && time.Since(lastAttempt) < reconnectBackoff {
						scheduleWake(reconnectBackoff - time.Since(lastAttempt))
					} else {
						lastAttempt = time.Now()
						newConn, err := c.dialAndHandshake(clientID)
						if err == nil {
							conn = newConn
							connectedID = clientID
						} else {
							scheduleWake(reconnectBackoff)
						}
					}
				}

				if conn != nil {
					if !lastSendAt.IsZero() && time.Since(lastSendAt) < minSendInterval {
						scheduleWake(minSendInterval - time.Since(lastSendAt))
					} else {
						if err := sendSetActivity(conn, clientID, desired, nowSec); err != nil {
							_ = conn.Close()
							conn = nil
							connectedID = ""
							hasSent = false
							lastAttempt = time.Now()
							scheduleWake(reconnectBackoff)
						} else {
							lastSent = sig
							hasSent = true
							lastSendAt = time.Now()
						}
					}
				}
			}
		}

		var timerCh <-chan time.Time
		if retryTimer != nil {
			timerCh = retryTimer.C
		}

		select {
		case <-c.stopCh:
			if conn != nil {
				if hasSent {
					_ = clearActivity(conn)
				}
				_ = conn.Close()
			}
			return
		case <-c.wakeCh:
		case <-timerCh:
			retryTimer = nil
		case <-heartbeat.C:
		}
	}
}

func (c *Client) dialAndHandshake(clientID string) (io.ReadWriteCloser, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	if err := handshake(conn, clientID); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func buildSignature(clientID string, p TrackPresence, nowSec int64) presenceSig {
	var startEpoch, endEpoch int64
	if p.IsPlaying && p.DurationSec > 0 {
		startEpoch = nowSec - int64(math.Floor(math.Max(0, p.PositionSec)))
		endEpoch = startEpoch + int64(math.Ceil(p.DurationSec))
	}
	return presenceSig{
		clientID:   clientID,
		title:      strings.TrimSpace(p.Title),
		artist:     strings.TrimSpace(p.Artist),
		album:      strings.TrimSpace(p.Album),
		coverURL:   NormalizeCoverURL(p.CoverURL),
		externalID: strings.TrimSpace(p.ExternalID),
		isPlaying:  p.IsPlaying,
		startEpoch: startEpoch,
		endEpoch:   endEpoch,
	}
}

func sigChanged(a, b presenceSig) bool {
	if a.clientID != b.clientID ||
		a.title != b.title ||
		a.artist != b.artist ||
		a.album != b.album ||
		a.coverURL != b.coverURL ||
		a.externalID != b.externalID ||
		a.isPlaying != b.isPlaying {
		return true
	}
	// Ignore 1-second rounding jitter on start/end epochs during normal playback.
	if a.isPlaying {
		if abs64(a.startEpoch-b.startEpoch) >= 2 || abs64(a.endEpoch-b.endEpoch) >= 2 {
			return true
		}
	}
	return false
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func setConnDeadline(rw io.ReadWriteCloser, d time.Duration) {
	if ds, ok := rw.(deadlineSetter); ok {
		if d > 0 {
			_ = ds.SetDeadline(time.Now().Add(d))
		} else {
			_ = ds.SetDeadline(time.Time{})
		}
	}
}

func handshake(rw io.ReadWriteCloser, clientID string) error {
	setConnDeadline(rw, ipcTimeout)
	defer setConnDeadline(rw, 0)

	payload := map[string]any{
		"v":         1,
		"client_id": clientID,
	}
	if err := writeFrame(rw, opHandshake, payload); err != nil {
		return err
	}
	op, body, err := readFrame(rw)
	if err != nil {
		return err
	}
	if op == opClose {
		return fmt.Errorf("discord closed handshake: %s", string(body))
	}
	return nil
}

func sendSetActivity(rw io.ReadWriteCloser, _ string, p TrackPresence, nowSec int64) error {
	setConnDeadline(rw, ipcTimeout)
	defer setConnDeadline(rw, 0)

	payload := buildSetActivityPayload(os.Getpid(), p, nowSec, uuid.NewString())
	if err := writeFrame(rw, opFrame, payload); err != nil {
		return err
	}
	return drainResponse(rw)
}

func clearActivity(rw io.ReadWriteCloser) error {
	setConnDeadline(rw, ipcTimeout)
	defer setConnDeadline(rw, 0)

	payload := map[string]any{
		"cmd": "SET_ACTIVITY",
		"args": map[string]any{
			"pid":      os.Getpid(),
			"activity": nil,
		},
		"nonce": uuid.NewString(),
	}
	if err := writeFrame(rw, opFrame, payload); err != nil {
		return err
	}
	return drainResponse(rw)
}

func drainResponse(rw io.ReadWriteCloser) error {
	for i := 0; i < 3; i++ {
		op, body, err := readFrame(rw)
		if err != nil {
			return err
		}
		switch op {
		case opPing:
			_ = writeFrame(rw, opPong, json.RawMessage(body))
		case opClose:
			return fmt.Errorf("discord closed connection: %s", string(body))
		default:
			return nil
		}
	}
	return nil
}

func sanitizeDiscordMeta(s, fallback string) string {
	t := strings.TrimSpace(s)
	if t == "" || strings.EqualFold(t, "YouTube Music") {
		return fallback
	}
	return t
}

func buildSetActivityPayload(pid int, p TrackPresence, nowSec int64, nonce string) map[string]any {
	cleanArtist := sanitizeDiscordMeta(p.Artist, "Unknown Artist")
	albumText := sanitizeDiscordMeta(p.Album, "Velora")
	cover := NormalizeCoverURL(p.CoverURL)

	assets := map[string]any{
		"large_image": cover,
		"large_text":  clampDiscordText(albumText, "Velora"),
	}
	if cover != FallbackCoverURL {
		smallText := "Velora"
		if !p.IsPlaying {
			smallText = "Velora (Paused)"
		}
		assets["small_image"] = FallbackCoverURL
		assets["small_text"] = smallText
	}

	activity := map[string]any{
		"name":                "Velora",
		"type":                2, // Listening
		"status_display_type": 0, // App Name ("Velora")
		"details":             clampDiscordText(p.Title, "Unknown Track"),
		"state":               clampDiscordText(cleanArtist, "Unknown Artist"),
		"assets":              assets,
	}

	if p.IsPlaying && p.DurationSec > 0 {
		start := nowSec - int64(math.Floor(math.Max(0, p.PositionSec)))
		end := start + int64(math.Ceil(p.DurationSec))
		activity["timestamps"] = map[string]any{
			"start": start,
			"end":   end,
		}
	}

	return map[string]any{
		"cmd": "SET_ACTIVITY",
		"args": map[string]any{
			"pid":      pid,
			"activity": activity,
		},
		"nonce": nonce,
	}
}

func writeFrame(w io.Writer, opcode uint32, payload any) error {
	var data []byte
	switch v := payload.(type) {
	case json.RawMessage:
		data = v
	case []byte:
		data = v
	default:
		var err error
		data, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	var hdr [8]byte
	binary.LittleEndian.PutUint32(hdr[0:4], opcode)
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(data)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	if len(data) > 0 {
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func readFrame(r io.Reader) (uint32, []byte, error) {
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	op := binary.LittleEndian.Uint32(hdr[0:4])
	length := binary.LittleEndian.Uint32(hdr[4:8])
	if length > maxFrameBytes {
		return 0, nil, fmt.Errorf("discord ipc frame too large: %d", length)
	}
	body := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, body); err != nil {
			return 0, nil, err
		}
	}
	return op, body, nil
}

// NormalizeCoverURL unwraps local Velora media proxy URLs and returns a public HTTPS cover URL for Discord.
func NormalizeCoverURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return FallbackCoverURL
	}

	if strings.Contains(s, "/media/cover") {
		if u, err := url.Parse(s); err == nil {
			if inner := strings.TrimSpace(u.Query().Get("url")); inner != "" {
				s = inner
			}
		}
	}

	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return FallbackCoverURL
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || host == "127.0.0.1" || host == "localhost" || host == "::1" {
		return FallbackCoverURL
	}

	if host == "i.ytimg.com" {
		if m := ytImgPathRe.FindStringSubmatch(u.Path); len(m) == 2 {
			return "https://i.ytimg.com/vi/" + m[1] + "/mqdefault.jpg"
		}
	}
	if strings.Contains(host, "googleusercontent.com") || strings.Contains(host, "ggpht.com") {
		s = googleSizeRe.ReplaceAllString(s, "=w512-h512-c-l90-rj")
	}
	if len(s) > 256 {
		return FallbackCoverURL
	}
	return s
}

func clampDiscordText(s, fallback string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		t = fallback
	}
	runes := []rune(t)
	if len(runes) == 1 {
		return t + " "
	}
	if len(runes) > 128 {
		return string(runes[:125]) + "..."
	}
	return t
}
