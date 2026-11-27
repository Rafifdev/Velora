package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"velora/internal/db"
)

type DownloadGroup struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Cover string `json:"cover"`
	Kind  string `json:"kind"` // "playlist" | "album"
}

type DownloadProgress struct {
	TrackID              int64   `json:"trackId"`
	ExternalID           string  `json:"externalId"`
	Title                string  `json:"title"`
	Artist               string  `json:"artist"`
	Status               string  `json:"status"` // queued | downloading | completed | failed | cancelled
	Progress             float64 `json:"progress"`
	Error                string  `json:"error,omitempty"`
	GroupKey             string  `json:"groupKey,omitempty"`
	GroupTitle           string  `json:"groupTitle,omitempty"`
	GroupCover           string  `json:"groupCover,omitempty"`
	GroupKind            string  `json:"groupKind,omitempty"`
	GroupTotal           int     `json:"groupTotal,omitempty"`
	GroupCompleted       int     `json:"groupCompleted,omitempty"`
	GroupProgress        float64 `json:"groupProgress,omitempty"`
	DownloadedBytes      int64   `json:"downloadedBytes,omitempty"`
	TotalBytes           int64   `json:"totalBytes,omitempty"`
	SpeedBps             int64   `json:"speedBps,omitempty"`
	GroupDownloadedBytes int64   `json:"groupDownloadedBytes,omitempty"`
	GroupTotalBytes      int64   `json:"groupTotalBytes,omitempty"`
	GroupSpeedBps        int64   `json:"groupSpeedBps,omitempty"`
}

type downloadGroupState struct {
	DownloadGroup
	tracks      map[int64]float64
	trackBytes  map[int64]int64
	trackTotals map[int64]int64
	trackSpeeds map[int64]int64
	enqueuing   bool
	cancelling  bool
}

type DownloadManager struct {
	store        *db.Store
	ytClient     *YTMusicClient
	downloadsDir string
	emit         func(eventName string, data any)
	httpClient   *http.Client
	sem          chan struct{}
	resolveTrack func(ctx context.Context, tr db.Track) (db.Track, error)

	mu           sync.Mutex
	states       map[int64]DownloadProgress
	cancels      map[int64]context.CancelFunc
	paused       map[int64]bool
	pausedTracks map[int64]db.Track
	groups       map[string]*downloadGroupState
	trackGroup   map[int64]string
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
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				MaxIdleConns:          64,
				MaxIdleConnsPerHost:   16,
				ResponseHeaderTimeout: 20 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		sem:          make(chan struct{}, 6),
		states:       make(map[int64]DownloadProgress),
		cancels:      make(map[int64]context.CancelFunc),
		paused:       make(map[int64]bool),
		pausedTracks: make(map[int64]db.Track),
		groups:       make(map[string]*downloadGroupState),
		trackGroup:   make(map[int64]string),
	}
}

func (m *DownloadManager) SetTrackResolver(fn func(ctx context.Context, tr db.Track) (db.Track, error)) {
	m.mu.Lock()
	m.resolveTrack = fn
	m.mu.Unlock()
}

func (m *DownloadManager) emitState(p DownloadProgress) {
	m.mu.Lock()
	if m.paused[p.TrackID] && p.Status != "paused" && p.Status != "cancelled" {
		if prev, ok := m.states[p.TrackID]; ok {
			if p.Progress <= 0 && prev.Progress > 0 {
				p.Progress = prev.Progress
			}
			if p.DownloadedBytes <= 0 && prev.DownloadedBytes > 0 {
				p.DownloadedBytes = prev.DownloadedBytes
			}
			if p.TotalBytes <= 0 && prev.TotalBytes > 0 {
				p.TotalBytes = prev.TotalBytes
			}
		}
		p.Status = "paused"
		p.SpeedBps = 0
		p.Error = ""
	}
	gk := m.trackGroup[p.TrackID]
	if gk != "" {
		if g := m.groups[gk]; g != nil {
			if g.trackBytes == nil {
				g.trackBytes = make(map[int64]int64)
			}
			if g.trackTotals == nil {
				g.trackTotals = make(map[int64]int64)
			}
			if g.trackSpeeds == nil {
				g.trackSpeeds = make(map[int64]int64)
			}
			switch p.Status {
			case "cancelled":
				if !g.cancelling {
					delete(g.tracks, p.TrackID)
					delete(g.trackBytes, p.TrackID)
					delete(g.trackTotals, p.TrackID)
					delete(g.trackSpeeds, p.TrackID)
					delete(m.trackGroup, p.TrackID)
				} else {
					g.trackSpeeds[p.TrackID] = 0
				}
			case "completed":
				g.tracks[p.TrackID] = 100
				if p.DownloadedBytes > 0 {
					g.trackBytes[p.TrackID] = p.DownloadedBytes
					g.trackTotals[p.TrackID] = p.DownloadedBytes
				}
				g.trackSpeeds[p.TrackID] = 0
			case "downloading":
				g.tracks[p.TrackID] = p.Progress
				if p.DownloadedBytes > 0 {
					g.trackBytes[p.TrackID] = p.DownloadedBytes
				}
				if p.TotalBytes > 0 {
					g.trackTotals[p.TrackID] = p.TotalBytes
				}
				g.trackSpeeds[p.TrackID] = p.SpeedBps
			case "paused":
				g.trackSpeeds[p.TrackID] = 0
			case "failed":
				g.trackSpeeds[p.TrackID] = 0
			case "queued":
				if _, ok := g.tracks[p.TrackID]; !ok {
					g.tracks[p.TrackID] = 0
				}
			}

			total := len(g.tracks)
			if total > 0 {
				completed := 0
				sum := 0.0
				for _, pct := range g.tracks {
					if pct >= 100 {
						completed++
					}
					sum += pct
				}
				var grpBytes, grpTotalBytes, grpSpeed int64
				for _, b := range g.trackBytes {
					grpBytes += b
				}
				for _, tb := range g.trackTotals {
					grpTotalBytes += tb
				}
				for _, sp := range g.trackSpeeds {
					grpSpeed += sp
				}
				p.GroupKey = g.Key
				p.GroupTitle = g.Title
				p.GroupCover = g.Cover
				p.GroupKind = g.Kind
				p.GroupTotal = total
				p.GroupCompleted = completed
				p.GroupProgress = sum / float64(total)
				p.GroupDownloadedBytes = grpBytes
				p.GroupTotalBytes = grpTotalBytes
				p.GroupSpeedBps = grpSpeed

				for tid, st := range m.states {
					if m.trackGroup[tid] == gk {
						st.GroupKey = p.GroupKey
						st.GroupTitle = p.GroupTitle
						st.GroupCover = p.GroupCover
						st.GroupKind = p.GroupKind
						st.GroupTotal = p.GroupTotal
						st.GroupCompleted = p.GroupCompleted
						st.GroupProgress = p.GroupProgress
						st.GroupDownloadedBytes = p.GroupDownloadedBytes
						st.GroupTotalBytes = p.GroupTotalBytes
						st.GroupSpeedBps = p.GroupSpeedBps
						m.states[tid] = st
					}
				}
			}
			if p.Status == "failed" && !g.cancelling {
				delete(g.tracks, p.TrackID)
				delete(g.trackBytes, p.TrackID)
				delete(g.trackTotals, p.TrackID)
				delete(g.trackSpeeds, p.TrackID)
				delete(m.trackGroup, p.TrackID)
			}
		}
	}

	if p.Status == "completed" || p.Status == "cancelled" || p.Status == "failed" {
		delete(m.states, p.TrackID)
		delete(m.pausedTracks, p.TrackID)
	} else {
		m.states[p.TrackID] = p
	}

	if gk != "" {
		stillActive := false
		for tid := range m.states {
			if m.trackGroup[tid] == gk {
				stillActive = true
				break
			}
		}
		if !stillActive && (m.groups[gk] == nil || !m.groups[gk].enqueuing) {
			delete(m.groups, gk)
			for tid, gKey := range m.trackGroup {
				if gKey == gk {
					delete(m.trackGroup, tid)
				}
			}
		}
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

func isTrackAlreadyDownloaded(tr db.Track) bool {
	if !tr.IsDownloaded || tr.FilePath == "" {
		return false
	}
	st, err := os.Stat(tr.FilePath)
	return err == nil && !st.IsDir()
}

func (m *DownloadManager) Enqueue(tr db.Track) (db.Track, error) {
	if tr.ID <= 0 {
		return db.Track{}, fmt.Errorf("invalid track id")
	}
	if isTrackAlreadyDownloaded(tr) {
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

	m.mu.Lock()
	if cur, ok := m.states[tr.ID]; ok && (cur.Status == "queued" || cur.Status == "downloading") {
		m.mu.Unlock()
		return tr, nil
	}
	delete(m.paused, tr.ID)
	m.pausedTracks[tr.ID] = tr
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

func (m *DownloadManager) EnqueueBatch(tracks []db.Track, group DownloadGroup) {
	if len(tracks) == 0 {
		return
	}
	key := strings.TrimSpace(group.Key)
	if key == "" {
		for _, tr := range tracks {
			_, _ = m.Enqueue(tr)
		}
		return
	}

	var allIDs []int64
	var pending []db.Track
	m.mu.Lock()
	g := m.groups[key]
	if g == nil {
		g = &downloadGroupState{
			DownloadGroup: group,
			tracks:        make(map[int64]float64, len(tracks)),
		}
		m.groups[key] = g
	} else {
		if strings.TrimSpace(group.Title) != "" {
			g.Title = group.Title
		}
		if strings.TrimSpace(group.Cover) != "" {
			g.Cover = group.Cover
		}
	}
	for _, tr := range tracks {
		if tr.ID <= 0 {
			continue
		}
		allIDs = append(allIDs, tr.ID)
		m.trackGroup[tr.ID] = key
		if isTrackAlreadyDownloaded(tr) {
			g.tracks[tr.ID] = 100
			continue
		}
		if _, exists := g.tracks[tr.ID]; !exists {
			g.tracks[tr.ID] = 0
		}
		pending = append(pending, tr)
	}
	if len(pending) == 0 {
		delete(m.groups, key)
		for tid, gk := range m.trackGroup {
			if gk == key {
				delete(m.trackGroup, tid)
			}
		}
		m.mu.Unlock()
		return
	}
	g.enqueuing = true
	m.mu.Unlock()

	if m.store != nil && len(allIDs) > 0 {
		m.store.SetTracksDownloadGroup(allIDs, key, group.Title, group.Cover, group.Kind)
	}

	for _, tr := range pending {
		_, _ = m.Enqueue(tr)
	}

	m.mu.Lock()
	if curGroup := m.groups[key]; curGroup != nil {
		curGroup.enqueuing = false
		stillActive := false
		for tid := range m.states {
			if m.trackGroup[tid] == key {
				stillActive = true
				break
			}
		}
		if !stillActive {
			delete(m.groups, key)
			for tid, gk := range m.trackGroup {
				if gk == key {
					delete(m.trackGroup, tid)
				}
			}
		}
	}
	m.mu.Unlock()
}

func (m *DownloadManager) isPaused(trackID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paused[trackID]
}

func (m *DownloadManager) Pause(trackID int64) {
	m.mu.Lock()
	m.paused[trackID] = true
	cancel, ok := m.cancels[trackID]
	st, hasState := m.states[trackID]
	if ok {
		delete(m.cancels, trackID)
	}
	m.mu.Unlock()

	if ok && cancel != nil {
		cancel()
	}
	if hasState && st.Status != "completed" && st.Status != "cancelled" {
		st.Status = "paused"
		st.SpeedBps = 0
		m.emitState(st)
	}
}

func (m *DownloadManager) Resume(trackID int64) {
	m.mu.Lock()
	delete(m.paused, trackID)
	tr, ok := m.pausedTracks[trackID]
	delete(m.pausedTracks, trackID)
	m.mu.Unlock()

	if ok {
		_, _ = m.Enqueue(tr)
	}
}

func (m *DownloadManager) PauseGroup(groupKey string) {
	key := strings.TrimSpace(groupKey)
	if key == "" {
		return
	}
	m.mu.Lock()
	var trackIDs []int64
	var cancels []context.CancelFunc
	var toEmit []DownloadProgress
	for tid, gk := range m.trackGroup {
		if gk == key {
			if st, ok := m.states[tid]; ok {
				m.paused[tid] = true
				if c, hasC := m.cancels[tid]; hasC && c != nil {
					cancels = append(cancels, c)
					delete(m.cancels, tid)
				}
				if st.Status != "completed" && st.Status != "cancelled" {
					st.Status = "paused"
					st.SpeedBps = 0
					m.states[tid] = st
					toEmit = append(toEmit, st)
				}
				trackIDs = append(trackIDs, tid)
			}
		}
	}
	m.mu.Unlock()

	for _, c := range cancels {
		c()
	}
	for _, st := range toEmit {
		m.emitState(st)
	}
	_ = trackIDs
}

func (m *DownloadManager) ResumeGroup(groupKey string) {
	key := strings.TrimSpace(groupKey)
	if key == "" {
		return
	}
	m.mu.Lock()
	var trackIDs []int64
	for tid, gk := range m.trackGroup {
		if gk == key && m.paused[tid] {
			trackIDs = append(trackIDs, tid)
		}
	}
	m.mu.Unlock()

	for _, tid := range trackIDs {
		m.Resume(tid)
	}
}

func (m *DownloadManager) emitAbortedOrPaused(tr db.Track, tmpPath string, written, totalBytes int64) {
	if m.isPaused(tr.ID) {
		m.mu.Lock()
		prev := m.states[tr.ID]
		m.mu.Unlock()
		if written <= 0 && prev.DownloadedBytes > 0 {
			written = prev.DownloadedBytes
		}
		if totalBytes <= 0 && prev.TotalBytes > 0 {
			totalBytes = prev.TotalBytes
		}
		pct := prev.Progress
		if totalBytes > 0 && written > 0 {
			pct = float64(written) * 100 / float64(totalBytes)
			if pct > 99 {
				pct = 99
			}
		}
		m.emitState(DownloadProgress{
			TrackID:         tr.ID,
			ExternalID:      tr.ExternalID,
			Title:           tr.Title,
			Artist:          tr.ArtistName,
			Status:          "paused",
			Progress:        pct,
			DownloadedBytes: written,
			TotalBytes:      totalBytes,
			SpeedBps:        0,
		})
		return
	}
	if tmpPath != "" {
		_ = os.Remove(tmpPath)
	}
	m.emitState(DownloadProgress{
		TrackID:    tr.ID,
		ExternalID: tr.ExternalID,
		Title:      tr.Title,
		Artist:     tr.ArtistName,
		Status:     "cancelled",
	})
}

func (m *DownloadManager) cleanupTrackFilesAndDB(trackID int64) {
	if trackID <= 0 {
		return
	}
	prefix := fmt.Sprintf("%d_", trackID)
	if entries, err := os.ReadDir(m.downloadsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
				_ = os.Remove(filepath.Join(m.downloadsDir, e.Name()))
			}
		}
	}
	if m.store != nil {
		if tr, err := m.store.GetTrackByID(trackID); err == nil && tr.FilePath != "" {
			cleanDownloads := filepath.Clean(m.downloadsDir)
			cleanFile := filepath.Clean(tr.FilePath)
			if strings.HasPrefix(strings.ToLower(cleanFile), strings.ToLower(cleanDownloads+string(filepath.Separator))) {
				_ = os.Remove(cleanFile)
			}
		}
		_, _ = m.store.ClearTrackDownloaded(trackID)
		m.store.ClearTrackDownloadGroup(trackID)
	}
}

func (m *DownloadManager) Cancel(trackID int64) {
	m.mu.Lock()
	delete(m.paused, trackID)
	delete(m.pausedTracks, trackID)
	cancel, ok := m.cancels[trackID]
	st, hasState := m.states[trackID]
	if ok {
		delete(m.cancels, trackID)
	}
	m.mu.Unlock()

	if ok && cancel != nil {
		cancel()
	}
	m.cleanupTrackFilesAndDB(trackID)
	if hasState {
		st.Status = "cancelled"
		m.emitState(st)
	}
}

func (m *DownloadManager) CancelGroup(groupKey string) {
	key := strings.TrimSpace(groupKey)
	if key == "" {
		return
	}
	m.mu.Lock()
	idSet := make(map[int64]struct{})
	if g := m.groups[key]; g != nil {
		g.cancelling = true
		for tid := range g.tracks {
			idSet[tid] = struct{}{}
		}
	}
	for tid, gk := range m.trackGroup {
		if gk == key {
			idSet[tid] = struct{}{}
		}
	}
	var trackIDs []int64
	var cancels []context.CancelFunc
	for tid := range idSet {
		trackIDs = append(trackIDs, tid)
		delete(m.paused, tid)
		delete(m.pausedTracks, tid)
		if c, ok := m.cancels[tid]; ok && c != nil {
			cancels = append(cancels, c)
			delete(m.cancels, tid)
		}
	}
	m.mu.Unlock()

	for _, c := range cancels {
		c()
	}

	for _, tid := range trackIDs {
		m.cleanupTrackFilesAndDB(tid)
		m.mu.Lock()
		st, hasState := m.states[tid]
		m.mu.Unlock()
		if hasState {
			st.Status = "cancelled"
			m.emitState(st)
		}
	}

	m.mu.Lock()
	delete(m.groups, key)
	for tid := range idSet {
		delete(m.states, tid)
		if m.trackGroup[tid] == key {
			delete(m.trackGroup, tid)
		}
	}
	m.mu.Unlock()
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

func (m *DownloadManager) RemoveAll() (int64, error) {
	m.mu.Lock()
	for _, fn := range m.cancels {
		if fn != nil {
			fn()
		}
	}
	m.cancels = make(map[int64]context.CancelFunc)
	m.mu.Unlock()

	var freedBytes int64
	if entries, err := os.ReadDir(m.downloadsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				p := filepath.Join(m.downloadsDir, e.Name())
				var sz int64
				if info, iErr := e.Info(); iErr == nil {
					sz = info.Size()
				}
				if os.Remove(p) == nil {
					freedBytes += sz
				}
			}
		}
	}

	if err := m.store.ClearAllDownloadedTracks(); err != nil {
		return freedBytes, err
	}
	return freedBytes, nil
}

func (m *DownloadManager) failDownload(tr db.Track, errMsg string) {
	m.emitState(DownloadProgress{
		TrackID:    tr.ID,
		ExternalID: tr.ExternalID,
		Title:      tr.Title,
		Artist:     tr.ArtistName,
		Status:     "failed",
		Error:      errMsg,
	})
}

func (m *DownloadManager) fetchChunk(ctx context.Context, streamURL, userAgent string, start, end int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	if userAgent == "" {
		userAgent = "com.google.android.youtube/20.10.38 (Linux; U; Android 14) gzip"
	}
	req.Header.Set("User-Agent", userAgent)
	return m.httpClient.Do(req)
}

func parseTotalBytesFromContentRange(cr string) int64 {
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

func (m *DownloadManager) resolveDownloadStream(ctx context.Context, tr *db.Track) (StreamInfo, error) {
	info, err := m.ytClient.ResolveStreamWindow(ctx, tr.ExternalID, "high", 0)
	if err == nil && strings.TrimSpace(info.URL) != "" {
		return info, nil
	}
	if strings.TrimSpace(tr.Title) != "" {
		q := strings.TrimSpace(tr.Title + " " + tr.ArtistName)
		if hits, sErr := m.ytClient.Search(ctx, q); sErr == nil {
			tried := 0
			for _, h := range hits {
				candID := strings.TrimSpace(h.VideoID)
				if candID == "" || candID == tr.ExternalID {
					continue
				}
				if candInfo, cErr := m.ytClient.ResolveStreamWindow(ctx, candID, "high", 0); cErr == nil && strings.TrimSpace(candInfo.URL) != "" {
					tr.ExternalID = candID
					if m.store != nil && tr.ID > 0 {
						_ = m.store.UpdateTrackExternalID(tr.ID, candID)
					}
					return candInfo, nil
				}
				tried++
				if tried >= 3 {
					break
				}
			}
		}
	}
	if err != nil {
		return StreamInfo{}, err
	}
	return StreamInfo{}, fmt.Errorf("failed to resolve audio stream")
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
		m.emitAbortedOrPaused(tr, "", 0, 0)
		return
	}

	if ctx.Err() != nil || m.isPaused(tr.ID) {
		m.emitAbortedOrPaused(tr, "", 0, 0)
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
				m.failDownload(tr, err.Error())
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

	m.mu.Lock()
	m.pausedTracks[tr.ID] = tr
	resolver := m.resolveTrack
	m.mu.Unlock()
	if resolver != nil {
		origID := tr.ID
		if resolved, rErr := resolver(ctx, tr); rErr == nil && resolved.ID > 0 {
			if resolved.ID != origID {
				var groupMeta *downloadGroupState
				m.mu.Lock()
				if m.paused[origID] {
					delete(m.paused, origID)
					m.paused[resolved.ID] = true
				}
				delete(m.pausedTracks, origID)
				m.pausedTracks[resolved.ID] = resolved
				if gk := m.trackGroup[origID]; gk != "" {
					delete(m.trackGroup, origID)
					m.trackGroup[resolved.ID] = gk
					if g := m.groups[gk]; g != nil {
						pct := g.tracks[origID]
						delete(g.tracks, origID)
						g.tracks[resolved.ID] = pct
						cp := *g
						groupMeta = &cp
					}
				}
				if cancelFn, ok := m.cancels[origID]; ok {
					delete(m.cancels, origID)
					m.cancels[resolved.ID] = cancelFn
				}
				if st, ok := m.states[origID]; ok {
					delete(m.states, origID)
					st.TrackID = resolved.ID
					st.ExternalID = resolved.ExternalID
					m.states[resolved.ID] = st
				}
				m.mu.Unlock()
				if groupMeta != nil && m.store != nil {
					m.store.ClearTrackDownloadGroup(origID)
					m.store.SetTracksDownloadGroup([]int64{resolved.ID}, groupMeta.Key, groupMeta.Title, groupMeta.Cover, groupMeta.Kind)
				}
			}
			tr = resolved
		}
	}

	if ctx.Err() != nil || m.isPaused(tr.ID) {
		m.emitAbortedOrPaused(tr, "", 0, 0)
		return
	}

	if strings.TrimSpace(tr.ExternalID) == "" {
		m.failDownload(tr, "track has no downloadable source")
		return
	}

	info, err := m.resolveDownloadStream(ctx, &tr)
	if err != nil {
		if ctx.Err() != nil || m.isPaused(tr.ID) {
			m.emitAbortedOrPaused(tr, "", 0, 0)
			return
		}
		m.failDownload(tr, err.Error())
		return
	}

	ext, formatLabel := pickDownloadExt(info.MimeType)
	safeName := sanitizeDownloadName(fmt.Sprintf("%d_%s_%s", tr.ID, tr.ArtistName, tr.Title))
	finalPath := filepath.Join(m.downloadsDir, safeName+ext)
	tmpPath := finalPath + ".part"

	totalBytes := info.ContentLength
	var written int64
	if st, statErr := os.Stat(tmpPath); statErr == nil && !st.IsDir() && st.Size() > 0 && (totalBytes <= 0 || st.Size() < totalBytes) {
		written = st.Size()
	}
	var outFile *os.File
	if written > 0 {
		outFile, err = os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	} else {
		outFile, err = os.Create(tmpPath)
	}
	if err != nil {
		m.failDownload(tr, err.Error())
		return
	}

	const chunkSize int64 = 256 * 1024
	lastEmit := time.Now()
	lastPct := 2.0
	lastEmitBytes := written
	var lastSpeedBps int64
	buf := make([]byte, 64*1024)
	refreshDeltas := []int{0, 15, -15, 30, -30}
	searchedAlternate := false

	for {
		if ctx.Err() != nil {
			_ = outFile.Close()
			if m.isPaused(tr.ID) {
				var pct float64
				if totalBytes > 0 {
					pct = float64(written) * 100 / float64(totalBytes)
					if pct > 99 {
						pct = 99
					}
				}
				m.emitState(DownloadProgress{
					TrackID:         tr.ID,
					ExternalID:      tr.ExternalID,
					Title:           tr.Title,
					Artist:          tr.ArtistName,
					Status:          "paused",
					Progress:        pct,
					DownloadedBytes: written,
					TotalBytes:      totalBytes,
					SpeedBps:        0,
				})
				return
			}
			_ = os.Remove(tmpPath)
			m.emitState(DownloadProgress{
				TrackID:    tr.ID,
				ExternalID: tr.ExternalID,
				Title:      tr.Title,
				Artist:     tr.ArtistName,
				Status:     "cancelled",
			})
			return
		}

		nextEnd := written + chunkSize - 1
		if totalBytes > 0 && nextEnd >= totalBytes {
			nextEnd = totalBytes - 1
		}
		if totalBytes > 0 && written >= totalBytes {
			break
		}

		var chunkResp *http.Response
		var lastStatus int
		var lastReqErr error

		for attempt := 0; attempt <= len(refreshDeltas); attempt++ {
			if attempt > 0 {
				if attempt == 1 {
					m.ytClient.InvalidateStreamCache(tr.ExternalID)
				}
				estSec := 0
				if totalBytes > 0 && info.DurationSec > 0 && written > 0 {
					estSec = int(float64(written)/float64(totalBytes)*float64(info.DurationSec)) - 2
				}
				estSec += refreshDeltas[attempt-1]
				if estSec < 0 {
					estSec = 0
				}
				win, wErr := m.ytClient.ResolveStreamWindow(ctx, tr.ExternalID, "high", estSec)
				if wErr == nil && strings.TrimSpace(win.URL) != "" {
					info = win
					if totalBytes <= 0 && info.ContentLength > 0 {
						totalBytes = info.ContentLength
					}
				}
			}

			resp, doErr := m.fetchChunk(ctx, info.URL, info.UserAgent, written, nextEnd)
			if doErr != nil {
				lastReqErr = doErr
				if ctx.Err() != nil {
					break
				}
				continue
			}
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
				chunkResp = resp
				break
			}
			lastStatus = resp.StatusCode
			_ = resp.Body.Close()

			if (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusGone) &&
				attempt == 2 && written == 0 && !searchedAlternate && strings.TrimSpace(tr.Title) != "" {
				searchedAlternate = true
				q := strings.TrimSpace(tr.Title + " " + tr.ArtistName)
				if hits, sErr := m.ytClient.Search(ctx, q); sErr == nil {
					for _, h := range hits {
						candID := strings.TrimSpace(h.VideoID)
						if candID == "" || candID == tr.ExternalID {
							continue
						}
						if candInfo, cErr := m.ytClient.ResolveStreamWindow(ctx, candID, "high", 0); cErr == nil && strings.TrimSpace(candInfo.URL) != "" {
							tr.ExternalID = candID
							info = candInfo
							if totalBytes <= 0 && info.ContentLength > 0 {
								totalBytes = info.ContentLength
							}
							if m.store != nil && tr.ID > 0 {
								_ = m.store.UpdateTrackExternalID(tr.ID, candID)
							}
							break
						}
					}
				}
			}

			if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusGone {
				break
			}
		}

		if chunkResp == nil {
			_ = outFile.Close()
			if ctx.Err() != nil || m.isPaused(tr.ID) {
				m.emitAbortedOrPaused(tr, tmpPath, written, totalBytes)
				return
			}
			_ = os.Remove(tmpPath)
			if lastStatus > 0 {
				m.failDownload(tr, fmt.Sprintf("download HTTP %d", lastStatus))
			} else if lastReqErr != nil {
				m.failDownload(tr, lastReqErr.Error())
			} else {
				m.failDownload(tr, "download failed")
			}
			return
		}

		if crTotal := parseTotalBytesFromContentRange(chunkResp.Header.Get("Content-Range")); crTotal > 0 {
			totalBytes = crTotal
		} else if totalBytes <= 0 && chunkResp.ContentLength > 0 && chunkResp.StatusCode == http.StatusOK {
			totalBytes = chunkResp.ContentLength
		}

		var chunkRead int64
		for {
			n, readErr := chunkResp.Body.Read(buf)
			if n > 0 {
				if _, wErr := outFile.Write(buf[:n]); wErr != nil {
					_ = chunkResp.Body.Close()
					_ = outFile.Close()
					_ = os.Remove(tmpPath)
					m.failDownload(tr, wErr.Error())
					return
				}
				written += int64(n)
				chunkRead += int64(n)
				if totalBytes > 0 {
					pct := float64(written) * 100 / float64(totalBytes)
					if pct > 99 {
						pct = 99
					}
					if pct-lastPct >= 2.0 || time.Since(lastEmit) >= 250*time.Millisecond {
						if m.isPaused(tr.ID) || ctx.Err() != nil {
							continue
						}
						now := time.Now()
						elapsed := now.Sub(lastEmit).Seconds()
						if elapsed > 0.05 && written >= lastEmitBytes {
							lastSpeedBps = int64(float64(written-lastEmitBytes) / elapsed)
						}
						lastEmitBytes = written
						lastPct = pct
						lastEmit = now
						m.emitState(DownloadProgress{
							TrackID:         tr.ID,
							ExternalID:      tr.ExternalID,
							Title:           tr.Title,
							Artist:          tr.ArtistName,
							Status:          "downloading",
							Progress:        pct,
							DownloadedBytes: written,
							TotalBytes:      totalBytes,
							SpeedBps:        lastSpeedBps,
						})
					}
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				_ = chunkResp.Body.Close()
				_ = outFile.Close()
				if ctx.Err() != nil || m.isPaused(tr.ID) {
					m.emitAbortedOrPaused(tr, tmpPath, written, totalBytes)
					return
				}
				_ = os.Remove(tmpPath)
				m.failDownload(tr, readErr.Error())
				return
			}
		}
		_ = chunkResp.Body.Close()

		if totalBytes > 0 {
			if written >= totalBytes {
				break
			}
		} else if chunkRead < chunkSize {
			break
		}
		if chunkRead == 0 {
			break
		}
	}

	_ = outFile.Close()
	if written <= 0 {
		_ = os.Remove(tmpPath)
		m.failDownload(tr, "empty audio response")
		return
	}

	if ctx.Err() != nil || m.isPaused(tr.ID) {
		m.emitAbortedOrPaused(tr, tmpPath, written, totalBytes)
		return
	}

	_ = os.Remove(finalPath)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		m.failDownload(tr, err.Error())
		return
	}

	if _, err := m.store.MarkTrackDownloaded(tr.ID, finalPath, formatLabel); err != nil {
		m.failDownload(tr, err.Error())
		return
	}
	if info.HasLoudness && m.store != nil {
		m.store.SaveTrackLoudness(tr.ID, tr.ExternalID, info.LoudnessDb)
	}

	m.emitState(DownloadProgress{
		TrackID:         tr.ID,
		ExternalID:      tr.ExternalID,
		Title:           tr.Title,
		Artist:          tr.ArtistName,
		Status:          "completed",
		Progress:        100,
		DownloadedBytes: written,
		TotalBytes:      written,
		SpeedBps:        0,
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
