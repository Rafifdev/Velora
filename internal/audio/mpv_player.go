package audio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"velora/internal/db"
	"velora/internal/sources"
)

type PlaybackState struct {
	CurrentTrack *db.Track `json:"currentTrack"`
	IsPlaying    bool      `json:"isPlaying"`
	PositionSec  float64   `json:"positionSec"`
	DurationSec  float64   `json:"durationSec"`
	Volume       int       `json:"volume"`
	Muted        bool      `json:"muted"`
	Shuffle      bool      `json:"shuffle"`
	RepeatMode   string    `json:"repeatMode"`
	Backend      string    `json:"backend"`
}

type PlayerController struct {
	mu           sync.RWMutex
	store        *db.Store
	yt           *sources.YTMusicClient
	streamClient *http.Client
	artworkDir   string
	baseURL      string
	server       *http.Server
	state        PlaybackState
	mpvCmd       *exec.Cmd
	mpvPipe      string
	hasMpv       bool
}

func NewPlayerController(store *db.Store, yt *sources.YTMusicClient, artworkDir string) *PlayerController {
	_, err := exec.LookPath("mpv")
	backend := "webview-stream"
	if err == nil {
		backend = "mpv-ipc"
	}
	return &PlayerController{
		store:      store,
		yt:         yt,
		artworkDir: artworkDir,
		streamClient: &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ResponseHeaderTimeout: 15 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		hasMpv:  err == nil,
		mpvPipe: `\\.\pipe\velora-mpv`,
		state: PlaybackState{
			Volume:     80,
			RepeatMode: "off",
			Backend:    backend,
		},
	}
}

func (p *PlayerController) StartMediaServer() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	p.baseURL = "http://" + ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/media/local", p.handleLocalMedia)
	mux.HandleFunc("/media/yt", p.handleYTMedia)
	mux.HandleFunc("/media/cover", p.handleCoverProxy)
	mux.Handle("/media/art/", http.StripPrefix("/media/art/", http.FileServer(http.Dir(p.artworkDir))))

	p.server = &http.Server{
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		_ = p.server.Serve(ln)
	}()
	return p.baseURL, nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Accept-Ranges")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p *PlayerController) BaseURL() string {
	return p.baseURL
}

func (p *PlayerController) GetStreamURL(track db.Track) string {
	if track.SourceType == "local" && track.ID > 0 {
		return fmt.Sprintf("%s/media/local?id=%d", p.baseURL, track.ID)
	}
	if track.ExternalID != "" {
		return fmt.Sprintf("%s/media/yt?v=%s", p.baseURL, track.ExternalID)
	}
	return fmt.Sprintf("%s/media/yt?v=dX3k_QDnzHE", p.baseURL)
}

func (p *PlayerController) handleLocalMedia(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid track id", http.StatusBadRequest)
		return
	}
	tr, err := p.store.GetTrackByID(id)
	if err != nil || tr.FilePath == "" {
		http.Error(w, "track not found", http.StatusNotFound)
		return
	}
	f, err := os.Open(tr.FilePath)
	if err != nil {
		http.Error(w, "file missing", http.StatusNotFound)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "stat failed", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, filepath.Base(tr.FilePath), stat.ModTime(), f)
}

func parseByteRange(header string) (start, end int64, hasRange bool, valid bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, -1, false, true
	}
	if !strings.HasPrefix(header, "bytes=") {
		return 0, -1, true, false
	}
	spec := strings.TrimSpace(strings.Split(strings.TrimPrefix(header, "bytes="), ",")[0])
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return 0, -1, true, false
	}
	s, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || s < 0 {
		return 0, -1, true, false
	}
	e := int64(-1)
	if rawEnd := strings.TrimSpace(parts[1]); rawEnd != "" {
		e, err = strconv.ParseInt(rawEnd, 10, 64)
		if err != nil || e < s {
			return 0, -1, true, false
		}
	}
	return s, e, true, true
}

func parseTotalFromContentRange(cr string) int64 {
	idx := strings.LastIndexByte(cr, '/')
	if idx < 0 {
		return 0
	}
	total, err := strconv.ParseInt(strings.TrimSpace(cr[idx+1:]), 10, 64)
	if err != nil || total <= 0 {
		return 0
	}
	return total
}

func (p *PlayerController) fetchYTMediaChunk(ctx context.Context, client *http.Client, streamURL, userAgent string, start, end int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	if userAgent == "" {
		userAgent = "com.google.android.youtube/20.10.38 (Linux; U; Android 14) gzip"
	}
	req.Header.Set("User-Agent", userAgent)
	return client.Do(req)
}

func (p *PlayerController) estimateWindowSec(pos, totalSize int64, durationSec, delta int) int {
	est := 0
	if totalSize > 0 && durationSec > 0 && pos > 0 {
		est = int(float64(pos)/float64(totalSize)*float64(durationSec)) - 2
	}
	est += delta
	if est < 0 {
		est = 0
	}
	return est
}

func (p *PlayerController) handleYTMedia(w http.ResponseWriter, r *http.Request) {
	videoID := strings.TrimSpace(r.URL.Query().Get("v"))
	if videoID == "" {
		http.Error(w, "missing video id", http.StatusBadRequest)
		return
	}

	start, reqEnd, hasRange, validRange := parseByteRange(r.Header.Get("Range"))
	if !validRange {
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	quality := "high"
	if p.store != nil {
		if prefs, err := p.store.GetPreferences(); err == nil {
			if q, ok := prefs["stream_quality"]; ok && q != "" {
				quality = q
			}
		}
	}

	client := p.streamClient
	if client == nil {
		client = &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ResponseHeaderTimeout: 15 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		}
	}

	info, err := p.yt.ResolveStreamWindow(r.Context(), videoID, quality, 0)
	if err != nil || info.URL == "" {
		http.Error(w, "stream unavailable", http.StatusBadGateway)
		return
	}

	if r.Method == http.MethodHead {
		if info.ContentLength <= 0 {
			if resp, rerr := p.fetchYTMediaChunk(r.Context(), client, info.URL, info.UserAgent, 0, 0); rerr == nil {
				if total := parseTotalFromContentRange(resp.Header.Get("Content-Range")); total > 0 {
					info.ContentLength = total
				}
				_ = resp.Body.Close()
			}
		}
		if info.MimeType != "" {
			w.Header().Set("Content-Type", info.MimeType)
		}
		w.Header().Set("Accept-Ranges", "bytes")
		if info.ContentLength > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(info.ContentLength, 10))
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	const chunkSize int64 = 256 * 1024

	if start > 512*1024 && info.ContentLength > 0 && info.DurationSec > 0 {
		initSec := p.estimateWindowSec(start, info.ContentLength, info.DurationSec, 0)
		if initSec > 0 {
			if win, werr := p.yt.ResolveStreamWindow(r.Context(), videoID, quality, initSec); werr == nil && win.URL != "" {
				info = win
			}
		}
	}

	if info.ContentLength > 0 && start >= info.ContentLength {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", info.ContentLength))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	firstEnd := start + chunkSize - 1
	if reqEnd >= 0 && reqEnd < firstEnd {
		firstEnd = reqEnd
	}
	if info.ContentLength > 0 && firstEnd >= info.ContentLength {
		firstEnd = info.ContentLength - 1
	}

	var firstResp *http.Response
	refreshDeltas := []int{0, 25, -25}
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			if attempt == 1 {
				p.yt.InvalidateStreamCache(videoID)
			}
			estSec := p.estimateWindowSec(start, info.ContentLength, info.DurationSec, refreshDeltas[attempt-1])
			win, werr := p.yt.ResolveStreamWindow(r.Context(), videoID, quality, estSec)
			if werr != nil || win.URL == "" {
				continue
			}
			info = win
		}
		resp, doErr := p.fetchYTMediaChunk(r.Context(), client, info.URL, info.UserAgent, start, firstEnd)
		if doErr != nil {
			continue
		}
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
			firstResp = resp
			break
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusGone {
			break
		}
	}

	if firstResp == nil {
		http.Error(w, "stream unavailable", http.StatusBadGateway)
		return
	}

	totalSize := info.ContentLength
	if total := parseTotalFromContentRange(firstResp.Header.Get("Content-Range")); total > 0 {
		totalSize = total
	}
	if totalSize > 0 && info.ContentLength <= 0 {
		info.ContentLength = totalSize
	}

	finalEnd := firstEnd
	if reqEnd >= 0 {
		finalEnd = reqEnd
		if totalSize > 0 && finalEnd >= totalSize {
			finalEnd = totalSize - 1
		}
	} else if totalSize > 0 {
		finalEnd = totalSize - 1
	}

	if ct := firstResp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else if info.MimeType != "" {
		w.Header().Set("Content-Type", info.MimeType)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	respLen := finalEnd - start + 1
	w.Header().Set("Content-Length", strconv.FormatInt(respLen, 10))
	if hasRange && totalSize > 0 {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, finalEnd, totalSize))
		w.WriteHeader(http.StatusPartialContent)
	} else if hasRange {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/*", start, finalEnd))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	written, copyErr := io.Copy(w, firstResp.Body)
	_ = firstResp.Body.Close()
	if copyErr != nil || written <= 0 {
		return
	}
	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}

	pos := start + written
	for pos <= finalEnd {
		if r.Context().Err() != nil {
			return
		}
		nextEnd := pos + chunkSize - 1
		if nextEnd > finalEnd {
			nextEnd = finalEnd
		}

		var chunkResp *http.Response
		for attempt := 0; attempt < 4; attempt++ {
			if attempt > 0 {
				estSec := p.estimateWindowSec(pos, totalSize, info.DurationSec, refreshDeltas[attempt-1])
				win, werr := p.yt.ResolveStreamWindow(r.Context(), videoID, quality, estSec)
				if werr != nil || win.URL == "" {
					continue
				}
				info = win
			}
			resp, doErr := p.fetchYTMediaChunk(r.Context(), client, info.URL, info.UserAgent, pos, nextEnd)
			if doErr != nil {
				continue
			}
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
				chunkResp = resp
				break
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusGone {
				break
			}
		}

		if chunkResp == nil {
			return
		}
		n, cerr := io.Copy(w, chunkResp.Body)
		_ = chunkResp.Body.Close()
		if cerr != nil || n <= 0 {
			return
		}
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		pos += n
	}
}

func (p *PlayerController) Control(action string, value float64, mode string) PlaybackState {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch action {
	case "play":
		p.state.IsPlaying = true
	case "pause":
		p.state.IsPlaying = false
	case "toggle":
		p.state.IsPlaying = !p.state.IsPlaying
	case "seek":
		if value >= 0 {
			p.state.PositionSec = value
		}
	case "volume":
		v := int(value)
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		p.state.Volume = v
		_ = p.store.SetPreference("volume", strconv.Itoa(v))
	case "mute":
		p.state.Muted = !p.state.Muted
	case "shuffle":
		p.state.Shuffle = !p.state.Shuffle
	case "repeat":
		if mode != "" {
			p.state.RepeatMode = mode
		} else {
			switch p.state.RepeatMode {
			case "off":
				p.state.RepeatMode = "all"
			case "all":
				p.state.RepeatMode = "one"
			default:
				p.state.RepeatMode = "off"
			}
		}
	}

	if p.hasMpv {
		p.sendMpvCommand(action, value)
	}
	return p.state
}

func (p *PlayerController) SetCurrentTrack(tr db.Track) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state.CurrentTrack = &tr
	p.state.PositionSec = 0
	p.state.DurationSec = float64(tr.DurationSec)
	p.state.IsPlaying = true
}

func (p *PlayerController) sendMpvCommand(action string, value float64) {
	// Best-effort IPC command when mpv is running; WebView2 HTML5 <audio> handles playback seamlessly otherwise.
	if p.mpvCmd == nil {
		return
	}
	var cmd []any
	switch action {
	case "pause":
		cmd = []any{"set_property", "pause", true}
	case "play":
		cmd = []any{"set_property", "pause", false}
	case "seek":
		cmd = []any{"seek", value, "absolute"}
	case "volume":
		cmd = []any{"set_property", "volume", value}
	default:
		return
	}
	_, _ = json.Marshal(cmd)
}

func (p *PlayerController) Shutdown() {
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.server.Shutdown(ctx)
	}
	if p.mpvCmd != nil && p.mpvCmd.Process != nil {
		_ = p.mpvCmd.Process.Kill()
	}
}

var (
	coverFetchClient = &http.Client{Timeout: 12 * time.Second}
	coverKeyLocks    sync.Map
)

func isAllowedCoverHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	for _, suffix := range []string{".googleusercontent.com", ".ggpht.com", ".ytimg.com", ".youtube.com"} {
		if strings.HasSuffix(h, suffix) {
			return true
		}
	}
	return false
}

func (p *PlayerController) handleCoverProxy(w http.ResponseWriter, r *http.Request) {
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || !isAllowedCoverHost(parsed.Hostname()) {
		http.Error(w, "invalid cover url", http.StatusBadRequest)
		return
	}

	sum := sha256.Sum256([]byte(rawURL))
	key := hex.EncodeToString(sum[:])
	cacheDir := filepath.Join(p.artworkDir, "ytcache")
	cachePath := filepath.Join(cacheDir, key+".img")

	if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
		w.Header().Set("Content-Type", http.DetectContentType(data))
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		_, _ = w.Write(data)
		return
	}

	lockVal, _ := coverKeyLocks.LoadOrStore(key, &sync.Mutex{})
	mu := lockVal.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	// Re-check after acquiring per-key lock
	if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
		w.Header().Set("Content-Type", http.DetectContentType(data))
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		_, _ = w.Write(data)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := coverFetchClient.Do(req)
	if err != nil {
		http.Error(w, "upstream failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "upstream status", http.StatusBadGateway)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil || len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadGateway)
		return
	}

	_ = os.MkdirAll(cacheDir, 0o755)
	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0o644); err == nil {
		_ = os.Rename(tmpPath, cachePath)
	}

	ctype := resp.Header.Get("Content-Type")
	if ctype == "" || !strings.HasPrefix(ctype, "image/") {
		ctype = http.DetectContentType(body)
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	_, _ = w.Write(body)
}
