package jam

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

//go:embed remote.html
var remoteHTML []byte

const DefaultPort = 8899

type QueueItem struct {
	TrackID     int64   `json:"trackId"`
	ExternalID  string  `json:"externalId"`
	Title       string  `json:"title"`
	Artist      string  `json:"artist"`
	DurationSec float64 `json:"durationSec"`
	CoverURL    string  `json:"coverUrl"`
}

type PlaybackSnapshot struct {
	TrackID     int64       `json:"trackId"`
	ExternalID  string      `json:"externalId"`
	Title       string      `json:"title"`
	Artist      string      `json:"artist"`
	Album       string      `json:"album"`
	CoverURL    string      `json:"coverUrl"`
	IsPlaying   bool        `json:"isPlaying"`
	PositionSec float64     `json:"positionSec"`
	DurationSec float64     `json:"durationSec"`
	Volume      float64     `json:"volume"`
	Muted       bool        `json:"muted"`
	Shuffle     bool        `json:"shuffle"`
	RepeatMode  string      `json:"repeatMode"`
	Queue       []QueueItem `json:"queue"`
}

type Command struct {
	Action     string  `json:"action"`
	Value      float64 `json:"value"`
	TrackID    int64   `json:"trackId,omitempty"`
	ExternalID string  `json:"externalId,omitempty"`
	Title      string  `json:"title,omitempty"`
	Artist     string  `json:"artist,omitempty"`
	Album      string  `json:"album,omitempty"`
	CoverURL   string  `json:"coverUrl,omitempty"`
	Duration   int     `json:"durationSec,omitempty"`
}

type SearchResult struct {
	TrackID     int64  `json:"trackId"`
	ExternalID  string `json:"externalId"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	DurationSec int    `json:"durationSec"`
	CoverURL    string `json:"coverUrl"`
	SourceType  string `json:"sourceType"`
}

type LyricLine struct {
	TimeSec float64 `json:"timeSec"`
	Text    string  `json:"text"`
}

type LyricsPayload struct {
	PlainLyrics  string      `json:"plainLyrics"`
	SyncedLyrics []LyricLine `json:"syncedLyrics"`
}

type Status struct {
	Enabled          bool   `json:"enabled"`
	Running          bool   `json:"running"`
	Port             int    `json:"port"`
	LanIP            string `json:"lanIp"`
	RemoteURL        string `json:"remoteUrl"`
	APIURL           string `json:"apiUrl"`
	QRDataURL        string `json:"qrDataUrl"`
	ConnectedClients int    `json:"connectedClients"`
	Error            string `json:"error,omitempty"`
}

type Server struct {
	mu           sync.RWMutex
	running      bool
	port         int
	lanIP        string
	remoteURL    string
	apiURL       string
	qrDataURL    string
	mediaBaseURL string
	lastErr      string
	snapshot     PlaybackSnapshot
	srv          *http.Server
	listener     net.Listener
	clients      map[chan []byte]struct{}
	onCommand    func(Command)
	onClients    func(int)
	onSearch     func(context.Context, string) ([]SearchResult, error)
	onLyrics     func(context.Context, PlaybackSnapshot) (LyricsPayload, error)
	onRPC        func(string, []json.RawMessage) (any, error)
	frontendFS   fs.FS
	httpClient   *http.Client
}

func NewServer(mediaBaseURL string, onCommand func(Command), onClients func(int)) *Server {
	return &Server{
		mediaBaseURL: strings.TrimRight(mediaBaseURL, "/"),
		clients:      make(map[chan []byte]struct{}),
		onCommand:    onCommand,
		onClients:    onClients,
		snapshot: PlaybackSnapshot{
			Volume:     80,
			RepeatMode: "off",
			Queue:      []QueueItem{},
		},
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

func (s *Server) SetSearchHandler(fn func(context.Context, string) ([]SearchResult, error)) {
	s.mu.Lock()
	s.onSearch = fn
	s.mu.Unlock()
}

func (s *Server) SetLyricsHandler(fn func(context.Context, PlaybackSnapshot) (LyricsPayload, error)) {
	s.mu.Lock()
	s.onLyrics = fn
	s.mu.Unlock()
}

func (s *Server) SetRPCHandler(fn func(string, []json.RawMessage) (any, error)) {
	s.mu.Lock()
	s.onRPC = fn
	s.mu.Unlock()
}

func (s *Server) SetFrontendFS(fsys fs.FS) {
	s.mu.Lock()
	s.frontendFS = fsys
	s.mu.Unlock()
}

func (s *Server) SetMediaBaseURL(baseURL string) {
	s.mu.Lock()
	s.mediaBaseURL = strings.TrimRight(baseURL, "/")
	s.mu.Unlock()
}

func (s *Server) Start() (Status, error) {
	s.mu.Lock()
	if s.running {
		st := s.statusLocked()
		s.mu.Unlock()
		return st, nil
	}
	s.mu.Unlock()

	ln, port, err := listenJamPort()
	if err != nil {
		s.mu.Lock()
		s.lastErr = err.Error()
		st := s.statusLocked()
		s.mu.Unlock()
		return st, err
	}

	lanIP := detectLanIP()
	remoteURL := fmt.Sprintf("http://%s:%d/jam", lanIP, port)
	apiURL := fmt.Sprintf("http://%s:%d/api", lanIP, port)
	qrDataURL := BuildQRDataURL(remoteURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRemoteUI)
	mux.HandleFunc("/api/state", s.withCORS(s.handleGetState))
	mux.HandleFunc("/api/events", s.withCORS(s.handleSSE))
	mux.HandleFunc("/api/cover", s.withCORS(s.handleCover))
	mux.HandleFunc("/api/search", s.withCORS(s.handleSearch))
	mux.HandleFunc("/api/lyrics", s.withCORS(s.handleLyrics))
	mux.HandleFunc("/api/rpc/", s.withCORS(s.handleRPC))
	mux.HandleFunc("/api/playback/", s.withCORS(s.handlePlaybackCommand))
	mux.HandleFunc("/media/", s.withCORS(s.handleMediaProxy))

	httpSrv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.mu.Lock()
	s.running = true
	s.port = port
	s.lanIP = lanIP
	s.remoteURL = remoteURL
	s.apiURL = apiURL
	s.qrDataURL = qrDataURL
	s.lastErr = ""
	s.listener = ln
	s.srv = httpSrv
	st := s.statusLocked()
	s.mu.Unlock()

	go func() {
		if serveErr := httpSrv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			s.mu.Lock()
			s.running = false
			s.lastErr = serveErr.Error()
			s.mu.Unlock()
		}
	}()

	return st, nil
}

func (s *Server) Stop() Status {
	s.mu.Lock()
	srv := s.srv
	ln := s.listener
	s.running = false
	s.srv = nil
	s.listener = nil
	for ch := range s.clients {
		close(ch)
		delete(s.clients, ch)
	}
	st := s.statusLocked()
	onClients := s.onClients
	s.mu.Unlock()

	if onClients != nil {
		onClients(0)
	}
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
	} else if ln != nil {
		_ = ln.Close()
	}
	return st
}

func (s *Server) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statusLocked()
}

func (s *Server) statusLocked() Status {
	remote := s.remoteURL
	api := s.apiURL
	if remote == "" {
		remote = fmt.Sprintf("http://127.0.0.1:%d/jam", DefaultPort)
	}
	if api == "" {
		api = fmt.Sprintf("http://127.0.0.1:%d/api", DefaultPort)
	}
	return Status{
		Enabled:          s.running,
		Running:          s.running,
		Port:             s.port,
		LanIP:            s.lanIP,
		RemoteURL:        remote,
		APIURL:           api,
		QRDataURL:        s.qrDataURL,
		ConnectedClients: len(s.clients),
		Error:            s.lastErr,
	}
}

func (s *Server) UpdateState(snap PlaybackSnapshot) {
	if snap.Queue == nil {
		snap.Queue = []QueueItem{}
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.snapshot = snap
	clients := make([]chan []byte, 0, len(s.clients))
	for ch := range s.clients {
		clients = append(clients, ch)
	}
	s.mu.Unlock()

	for _, ch := range clients {
		select {
		case ch <- payload:
		default:
		}
	}
}

func (s *Server) handleRemoteUI(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	fsys := s.frontendFS
	s.mu.RUnlock()

	if fsys == nil {
		if r.URL.Path != "/" && r.URL.Path != "/jam" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(remoteHTML)
		return
	}

	reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if reqPath == "" || reqPath == "jam" || reqPath == "." {
		reqPath = "index.html"
	}
	if _, err := fs.Stat(fsys, reqPath); err != nil {
		if strings.Contains(path.Base(reqPath), ".") {
			http.NotFound(w, r)
			return
		}
		reqPath = "index.html"
	}
	if reqPath == "index.html" {
		w.Header().Set("Cache-Control", "no-store")
	}
	http.ServeFileFS(w, r, fsys, reqPath)
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	method := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/rpc/"), "/")
	if method == "" {
		http.Error(w, "missing method", http.StatusBadRequest)
		return
	}

	var req struct {
		Args []json.RawMessage `json:"args"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(io.LimitReader(r.Body, 2*1024*1024)).Decode(&req)
	}

	s.mu.RLock()
	fn := s.onRPC
	s.mu.RUnlock()
	if fn == nil {
		http.Error(w, "rpc unavailable", http.StatusServiceUnavailable)
		return
	}

	res, err := fn(method, req.Args)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if res == nil {
		_, _ = w.Write([]byte("null"))
		return
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleMediaProxy(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mediaBase := s.mediaBaseURL
	s.mu.RUnlock()
	if mediaBase == "" {
		http.NotFound(w, r)
		return
	}
	target := mediaBase + r.URL.RequestURI()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		http.Error(w, "bad media url", http.StatusBadRequest)
		return
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		http.Error(w, "media unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Cache-Control"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) handleGetState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	snap := s.snapshot
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(snap)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []byte, 8)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	count := len(s.clients)
	snap := s.snapshot
	onClients := s.onClients
	s.mu.Unlock()

	if onClients != nil {
		onClients(count)
	}

	defer func() {
		s.mu.Lock()
		if _, exists := s.clients[ch]; exists {
			delete(s.clients, ch)
			close(ch)
		}
		remaining := len(s.clients)
		s.mu.Unlock()
		if onClients != nil {
			onClients(remaining)
		}
	}()

	if initData, err := json.Marshal(snap); err == nil {
		_, _ = fmt.Fprintf(w, "event: state\ndata: %s\n\n", initData)
		flusher.Flush()
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, open := <-ch:
			if !open {
				return
			}
			_, _ = fmt.Fprintf(w, "event: state\ndata: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}
}

func (s *Server) handleCover(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	coverURL := strings.TrimSpace(s.snapshot.CoverURL)
	mediaBase := s.mediaBaseURL
	s.mu.RUnlock()

	if coverURL == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(coverURL, "/") && mediaBase != "" {
		coverURL = mediaBase + coverURL
	}
	if !strings.HasPrefix(coverURL, "http://") && !strings.HasPrefix(coverURL, "https://") {
		http.NotFound(w, r)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, coverURL, nil)
	if err != nil {
		http.Error(w, "invalid cover url", http.StatusBadRequest)
		return
	}
	resp, err := s.httpClient.Do(req)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp != nil {
			_ = resp.Body.Close()
		}
		http.Error(w, "cover unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 8*1024*1024))
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
		return
	}
	s.mu.RLock()
	fn := s.onSearch
	s.mu.RUnlock()
	if fn == nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()
	results, err := fn(ctx, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []SearchResult{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

func (s *Server) handleLyrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	snap := s.snapshot
	fn := s.onLyrics
	s.mu.RUnlock()

	if fn == nil || snap.Title == "" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plainLyrics":"","syncedLyrics":[]}`))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	res, err := fn(ctx, snap)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plainLyrics":"","syncedLyrics":[]}`))
		return
	}
	if res.SyncedLyrics == nil {
		res.SyncedLyrics = []LyricLine{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handlePlaybackCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	action := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/playback/"), "/")
	switch action {
	case "play", "pause", "toggle", "next", "prev", "seek", "volume", "mute", "shuffle", "repeat", "play-index", "remove-index", "play-track", "add-track":
	default:
		http.Error(w, "unknown action", http.StatusNotFound)
		return
	}

	var body Command
	if r.Body != nil {
		_ = json.NewDecoder(io.LimitReader(r.Body, 8192)).Decode(&body)
	}
	if qVal := r.URL.Query().Get("value"); qVal != "" {
		if parsed, err := strconv.ParseFloat(qVal, 64); err == nil {
			body.Value = parsed
		}
	}

	body.Action = action
	s.mu.RLock()
	cb := s.onCommand
	s.mu.RUnlock()
	if cb != nil {
		cb(body)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "action": action, "value": body.Value})
}

func (s *Server) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func listenJamPort() (net.Listener, int, error) {
	for port := DefaultPort; port <= DefaultPort+10; port++ {
		if ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port)); err == nil {
			return ln, port, nil
		}
	}
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, 0, err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return ln, DefaultPort, nil
	}
	return ln, addr.Port, nil
}

func detectLanIP() string {
	if conn, err := net.Dial("udp4", "8.8.8.8:80"); err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil && !addr.IP.IsLoopback() {
			if ip4 := addr.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, _ := iface.Addrs()
			for _, a := range addrs {
				var ip net.IP
				switch v := a.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				ip4 := ip.To4()
				if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
					continue
				}
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

func BuildQRDataURL(content string) string {
	png, err := qrcode.Encode(content, qrcode.Medium, 280)
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}
