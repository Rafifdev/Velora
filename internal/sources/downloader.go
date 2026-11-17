package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"velora/internal/db"
)

type DownloadProgress struct {
	TrackID    int64   `json:"trackId"`
	ExternalID string  `json:"externalId"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Status     string  `json:"status"` // queued | downloading | completed | failed | cancelled
	Progress   float64 `json:"progress"`
	Error      string  `json:"error,omitempty"`
}

type DownloadManager struct {
	store        *db.Store
	ytClient     *YTMusicClient
	downloadsDir string
	emit         func(eventName string, data any)
	httpClient   *http.Client
	sem          chan struct{}

	mu      sync.Mutex
	states  map[int64]DownloadProgress
	cancels map[int64]context.CancelFunc
}

func NewDownloadManager(store *db.Store, ytClient *YTMusicClient, downloadsDir string, emit func(string, any)) *DownloadManager {
	_ = os.MkdirAll(downloadsDir, 0o755)
	return &DownloadManager{
		store:        store,
		ytClient:     ytClient,
		downloadsDir: downloadsDir,
		emit:         emit,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
		sem:     make(chan struct{}, 2),
		states:  make(map[int64]DownloadProgress),
		cancels: make(map[int64]context.CancelFunc),
	}
}

func (m *DownloadManager) emitState(p DownloadProgress) {
	m.mu.Lock()
	if p.Status == "completed" || p.Status == "cancelled" {
		delete(m.states, p.TrackID)
	} else {
		m.states[p.TrackID] = p
	}
	m.mu.Unlock()

	if m.emit != nil {
		m.emit("download:progress", p)
	}
}

func (m *DownloadManager) GetStates() []DownloadProgress {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]DownloadProgress, 0, len(m.states))
	for _, st := range m.states {
		out = append(out, st)
	}
	return out
}

func (m *DownloadManager) Enqueue(tr db.Track) (db.Track, error) {
	if tr.ID <= 0 {
		return db.Track{}, fmt.Errorf("invalid track id")
	}
	if tr.IsDownloaded && tr.FilePath != "" {
		if st, err := os.Stat(tr.FilePath); err == nil && !st.IsDir() {
			m.emitState(DownloadProgress{
				TrackID:    tr.ID,
				ExternalID: tr.ExternalID,
				Title:      tr.Title,
				Artist:     tr.ArtistName,
				Status:     "completed",
				Progress:   100,
			})
			return tr, nil
		}
	}

	m.mu.Lock()
	if cur, ok := m.states[tr.ID]; ok && (cur.Status == "queued" || cur.Status == "downloading") {
		m.mu.Unlock()
		return tr, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancels[tr.ID] = cancel
	m.mu.Unlock()

	m.emitState(DownloadProgress{
		TrackID:    tr.ID,
		ExternalID: tr.ExternalID,
		Title:      tr.Title,
		Artist:     tr.ArtistName,
		Status:     "queued",
		Progress:   0,
	})

	go m.runDownload(ctx, tr)
	return tr, nil
}

func (m *DownloadManager) Cancel(trackID int64) {
	m.mu.Lock()
	cancel, ok := m.cancels[trackID]
	st := m.states[trackID]
	if ok {
		delete(m.cancels, trackID)
	}
	m.mu.Unlock()

	if ok && cancel != nil {
		cancel()
	}
	if st.TrackID == trackID {
		st.Status = "cancelled"
		m.emitState(st)
	}
}

func (m *DownloadManager) Remove(trackID int64) (db.Track, error) {
	m.Cancel(trackID)
	tr, err := m.store.GetTrackByID(trackID)
	if err != nil {
		return db.Track{}, err
	}
	if tr.FilePath != "" {
		cleanDownloads := filepath.Clean(m.downloadsDir)
		cleanFile := filepath.Clean(tr.FilePath)
		if strings.HasPrefix(strings.ToLower(cleanFile), strings.ToLower(cleanDownloads+string(filepath.Separator))) {
			_ = os.Remove(cleanFile)
		}
	}
	return m.store.ClearTrackDownloaded(trackID)
}

func (m *DownloadManager) runDownload(ctx context.Context, tr db.Track) {
	defer func() {
		m.mu.Lock()
		delete(m.cancels, tr.ID)
		m.mu.Unlock()
	}()

	select {
	case m.sem <- struct{}{}:
		defer func() { <-m.sem }()
	case <-ctx.Done():
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "cancelled",
		})
		return
	}

	m.emitState(DownloadProgress{
		TrackID:    tr.ID,
		ExternalID: tr.ExternalID,
		Title:      tr.Title,
		Artist:     tr.ArtistName,
		Status:     "downloading",
		Progress:   2,
	})

	if tr.SourceType == "local" && tr.FilePath != "" {
		if st, err := os.Stat(tr.FilePath); err == nil && !st.IsDir() {
			ext := strings.TrimPrefix(strings.ToUpper(filepath.Ext(tr.FilePath)), ".")
			if ext == "" {
				ext = tr.Format
			}
			if _, err := m.store.MarkTrackDownloaded(tr.ID, tr.FilePath, ext); err != nil {
				m.emitState(DownloadProgress{
					TrackID:    tr.ID,
					ExternalID: tr.ExternalID,
					Title:      tr.Title,
					Artist:     tr.ArtistName,
					Status:     "failed",
					Error:      err.Error(),
				})
				return
			}
			m.emitState(DownloadProgress{
				TrackID:    tr.ID,
				ExternalID: tr.ExternalID,
				Title:      tr.Title,
				Artist:     tr.ArtistName,
				Status:     "completed",
				Progress:   100,
			})
			return
		}
	}

	if strings.TrimSpace(tr.ExternalID) == "" {
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      "track has no downloadable source",
		})
		return
	}

	info, err := m.ytClient.ResolveStreamWindow(ctx, tr.ExternalID, "high", 0)
	if err != nil || strings.TrimSpace(info.URL) == "" {
		msg := "failed to resolve audio stream"
		if err != nil {
			msg = err.Error()
		}
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      msg,
		})
		return
	}

	ext, formatLabel := pickDownloadExt(info.MimeType)
	safeName := sanitizeDownloadName(fmt.Sprintf("%d_%s_%s", tr.ID, tr.ArtistName, tr.Title))
	finalPath := filepath.Join(m.downloadsDir, safeName+ext)
	tmpPath := finalPath + ".part"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      err.Error(),
		})
		return
	}
	if info.UserAgent != "" {
		req.Header.Set("User-Agent", info.UserAgent)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      fmt.Sprintf("download HTTP %d", resp.StatusCode),
		})
		return
	}

	outFile, err := os.Create(tmpPath)
	if err != nil {
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      err.Error(),
		})
		return
	}

	totalBytes := resp.ContentLength
	if totalBytes <= 0 && info.ContentLength > 0 {
		totalBytes = info.ContentLength
	}

	buf := make([]byte, 32*1024)
	var written int64
	lastEmit := time.Now()
	lastPct := 2.0

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := outFile.Write(buf[:n]); wErr != nil {
				_ = outFile.Close()
				_ = os.Remove(tmpPath)
				m.emitState(DownloadProgress{
					TrackID:    tr.ID,
					ExternalID: tr.ExternalID,
					Title:      tr.Title,
					Artist:     tr.ArtistName,
					Status:     "failed",
					Error:      wErr.Error(),
				})
				return
			}
			written += int64(n)
			if totalBytes > 0 {
				pct := float64(written) * 100 / float64(totalBytes)
				if pct > 99 {
					pct = 99
				}
				if pct-lastPct >= 2.0 || time.Since(lastEmit) >= 250*time.Millisecond {
					lastPct = pct
					lastEmit = time.Now()
					m.emitState(DownloadProgress{
						TrackID:    tr.ID,
						ExternalID: tr.ExternalID,
						Title:      tr.Title,
						Artist:     tr.ArtistName,
						Status:     "downloading",
						Progress:   pct,
					})
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = outFile.Close()
			_ = os.Remove(tmpPath)
			status := "failed"
			errMsg := readErr.Error()
			if ctx.Err() != nil {
				status = "cancelled"
				errMsg = ""
			}
			m.emitState(DownloadProgress{
				TrackID:    tr.ID,
				ExternalID: tr.ExternalID,
				Title:      tr.Title,
				Artist:     tr.ArtistName,
				Status:     status,
				Error:      errMsg,
			})
			return
		}
	}

	_ = outFile.Close()
	_ = os.Remove(finalPath)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      err.Error(),
		})
		return
	}

	if _, err := m.store.MarkTrackDownloaded(tr.ID, finalPath, formatLabel); err != nil {
		m.emitState(DownloadProgress{
			TrackID:    tr.ID,
			ExternalID: tr.ExternalID,
			Title:      tr.Title,
			Artist:     tr.ArtistName,
			Status:     "failed",
			Error:      err.Error(),
		})
		return
	}

	m.emitState(DownloadProgress{
		TrackID:    tr.ID,
		ExternalID: tr.ExternalID,
		Title:      tr.Title,
		Artist:     tr.ArtistName,
		Status:     "completed",
		Progress:   100,
	})
}

func pickDownloadExt(mimeType string) (string, string) {
	lower := strings.ToLower(mimeType)
	switch {
	case strings.Contains(lower, "audio/mp4"), strings.Contains(lower, "mp4a"), strings.Contains(lower, "m4a"):
		return ".m4a", "M4A"
	case strings.Contains(lower, "audio/ogg"), strings.Contains(lower, "ogg"):
		return ".ogg", "OGG"
	case strings.Contains(lower, "audio/mpeg"), strings.Contains(lower, "mp3"):
		return ".mp3", "MP3"
	default:
		return ".webm", "OPUS"
	}
}

func sanitizeDownloadName(raw string) string {
	replacer := strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", "\"", "_",
		"/", "_", "\\", "_", "|", "_", "?", "_", "*", "_",
	)
	cleaned := strings.TrimSpace(replacer.Replace(raw))
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if len(cleaned) > 90 {
		cleaned = strings.TrimSpace(cleaned[:90])
	}
	if cleaned == "" {
		return "track"
	}
	return cleaned
}
