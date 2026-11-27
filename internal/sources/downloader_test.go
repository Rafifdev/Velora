package sources

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"velora/internal/db"
)

func TestDownloaderChunkedRangeBypassesNonRange403(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open error: %v", err)
	}
	defer store.Close()

	payload := bytes.Repeat([]byte("VELORA_AUDIO_DATA_"), 70*1024) // ~1.26 MB (> 512KB chunk)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rng := r.Header.Get("Range")
		if rng == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var start, end int64
		if _, err := fmt.Sscanf(rng, "bytes=%d-%d", &start, &end); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if end >= int64(len(payload)) {
			end = int64(len(payload)) - 1
		}
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start : end+1])
	}))
	defer srv.Close()

	yt := &YTMusicClient{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("offline test")
			}),
		},
		cache: map[string]cachedStream{},
	}
	yt.mu.Lock()
	yt.cache["vidRange01:high"] = cachedStream{
		URL:           srv.URL + "/audio",
		MimeType:      "audio/mp4",
		UserAgent:     "TestAgent/1.0",
		ContentLength: int64(len(payload)),
		DurationSec:   120,
		ExpiresAt:     time.Now().Add(time.Hour),
	}
	yt.mu.Unlock()

	tr, err := store.UpsertTrack(db.Track{
		SourceType:  "ytmusic",
		ExternalID:  "vidRange01",
		Title:       "Range Song",
		ArtistName:  "Test Artist",
		AlbumTitle:  "Test Album",
		DurationSec: 120,
		Format:      "M4A",
	})
	if err != nil {
		t.Fatalf("SaveYTMusicTrack error: %v", err)
	}

	doneCh := make(chan DownloadProgress, 16)
	dm := NewDownloadManager(store, yt, filepath.Join(tmpDir, "downloads"), func(_ string, data any) {
		if p, ok := data.(DownloadProgress); ok && (p.Status == "completed" || p.Status == "failed") {
			doneCh <- p
		}
	})

	if _, err := dm.Enqueue(tr); err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}

	select {
	case p := <-doneCh:
		if p.Status != "completed" {
			t.Fatalf("expected completed, got %s (%s)", p.Status, p.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for chunked download")
	}

	updated, err := store.GetTrackByID(tr.ID)
	if err != nil {
		t.Fatalf("GetTrackByID error: %v", err)
	}
	if !updated.IsDownloaded || updated.FilePath == "" {
		t.Fatalf("expected track to be marked downloaded, got %+v", updated)
	}
	gotBytes, err := os.ReadFile(updated.FilePath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !bytes.Equal(gotBytes, payload) {
		t.Fatalf("downloaded bytes mismatch: got %d want %d", len(gotBytes), len(payload))
	}
}

func TestDownloadManagerEnqueueBatchGrouping(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open error: %v", err)
	}
	defer store.Close()

	payload := bytes.Repeat([]byte("A"), 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "audio/webm")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(payload)-1, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	yt := &YTMusicClient{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("offline test")
			}),
		},
		cache: map[string]cachedStream{},
	}
	yt.mu.Lock()
	yt.cache["vidGrp01:high"] = cachedStream{URL: srv.URL + "/1", MimeType: "audio/webm", ContentLength: int64(len(payload)), ExpiresAt: time.Now().Add(time.Hour)}
	yt.cache["vidGrp02:high"] = cachedStream{URL: srv.URL + "/2", MimeType: "audio/webm", ContentLength: int64(len(payload)), ExpiresAt: time.Now().Add(time.Hour)}
	yt.mu.Unlock()

	t1, _ := store.UpsertTrack(db.Track{SourceType: "ytmusic", ExternalID: "vidGrp01", Title: "Song 1", ArtistName: "Artist", AlbumTitle: "Album", DurationSec: 60, Format: "OPUS"})
	t2, _ := store.UpsertTrack(db.Track{SourceType: "ytmusic", ExternalID: "vidGrp02", Title: "Song 2", ArtistName: "Artist", AlbumTitle: "Album", DurationSec: 60, Format: "OPUS"})

	var mu sync.Mutex
	var events []DownloadProgress
	doneCh := make(chan struct{})
	dm := NewDownloadManager(store, yt, filepath.Join(tmpDir, "downloads"), func(_ string, data any) {
		p, ok := data.(DownloadProgress)
		if !ok {
			return
		}
		mu.Lock()
		events = append(events, p)
		completed := 0
		for _, e := range events {
			if e.Status == "completed" {
				completed++
			}
		}
		mu.Unlock()
		if completed >= 2 {
			select {
			case <-doneCh:
			default:
				close(doneCh)
			}
		}
	})

	dm.EnqueueBatch([]db.Track{t1, t2}, DownloadGroup{
		Key:   "playlist:10",
		Title: "My Mix",
		Cover: "https://example.com/cover.jpg",
		Kind:  "playlist",
	})

	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for batch download")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) == 0 {
		t.Fatal("expected progress events")
	}
	last := events[len(events)-1]
	if last.GroupKey != "playlist:10" || last.GroupTotal != 2 || last.GroupCompleted != 2 {
		t.Fatalf("unexpected group final state: %+v", last)
	}
}

func TestDownloadManagerPauseResumeAndCancelGroupRollback(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open error: %v", err)
	}
	defer store.Close()

	payload := bytes.Repeat([]byte("VELORA_STREAM_"), 180*1024) // ~2.58 MB
	holdCh := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var start, end int64
		if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end); err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if end >= int64(len(payload)) {
			end = int64(len(payload)) - 1
		}
		if start > 0 && r.URL.Path != "/audio/1" {
			select {
			case <-r.Context().Done():
				return
			case <-holdCh:
			}
		}
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start : end+1])
	}))
	defer srv.Close()
	defer func() {
		select {
		case <-holdCh:
		default:
			close(holdCh)
		}
	}()

	yt := &YTMusicClient{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("offline test")
			}),
		},
		cache: map[string]cachedStream{},
	}
	var tracks []db.Track
	// Create 1 fast track + 7 multi-chunk tracks (> sem capacity 6)
	for i := 1; i <= 8; i++ {
		vid := fmt.Sprintf("vidPR%02d", i)
		yt.mu.Lock()
		yt.cache[vid+":high"] = cachedStream{
			URL:           srv.URL + fmt.Sprintf("/audio/%d", i),
			MimeType:      "audio/mp4",
			UserAgent:     "TestAgent/1.0",
			ContentLength: int64(len(payload)),
			DurationSec:   100,
			ExpiresAt:     time.Now().Add(time.Hour),
		}
		yt.mu.Unlock()
		tr, uErr := store.UpsertTrack(db.Track{
			SourceType:  "ytmusic",
			ExternalID:  vid,
			Title:       fmt.Sprintf("Song %d", i),
			ArtistName:  "Artist",
			DurationSec: 100,
		})
		if uErr != nil {
			t.Fatal(uErr)
		}
		tracks = append(tracks, tr)
	}

	firstDone := make(chan struct{}, 1)
	dm := NewDownloadManager(store, yt, filepath.Join(tmpDir, "downloads"), func(_ string, data any) {
		if p, ok := data.(DownloadProgress); ok && p.Status == "completed" {
			select {
			case firstDone <- struct{}{}:
			default:
			}
		}
	})

	dm.EnqueueBatch(tracks, DownloadGroup{
		Key:   "playlist:99",
		Title: "New Release Mix",
		Kind:  "playlist",
	})

	select {
	case <-firstDone:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for track 1 to complete")
	}

	// Pause the group while tracks 2..6 are downloading/queued
	dm.PauseGroup("playlist:99")
	time.Sleep(120 * time.Millisecond)

	states := dm.GetStates()
	if len(states) != 7 {
		t.Fatalf("expected 7 remaining paused tracks in states, got %d: %+v", len(states), states)
	}
	for _, st := range states {
		if st.Status != "paused" {
			t.Fatalf("expected every remaining track to be paused, got track %d status=%q", st.TrackID, st.Status)
		}
	}

	// Resume the group and verify states switch back out of paused
	dm.ResumeGroup("playlist:99")
	time.Sleep(60 * time.Millisecond)
	resumedStates := dm.GetStates()
	hasActive := false
	for _, st := range resumedStates {
		if st.Status == "downloading" || st.Status == "queued" {
			hasActive = true
		}
	}
	if !hasActive {
		t.Fatalf("expected resumed group to have downloading/queued tracks, got %+v", resumedStates)
	}

	// Now CancelGroup: track 1 (already completed!) and tracks 2..8 (in-flight/part) must ALL be rolled back!
	dm.CancelGroup("playlist:99")

	if got := dm.GetStates(); len(got) != 0 {
		t.Fatalf("expected 0 active states after CancelGroup, got %+v", got)
	}
	dlTracks, err := store.GetDownloadedTracks()
	if err != nil {
		t.Fatal(err)
	}
	if len(dlTracks) != 0 {
		t.Fatalf("expected 0 downloaded tracks stuck in DB after CancelGroup, got %d: %+v", len(dlTracks), dlTracks)
	}
	deadline := time.Now().Add(2 * time.Second)
	var entries []os.DirEntry
	for {
		entries, _ = os.ReadDir(filepath.Join(tmpDir, "downloads"))
		if len(entries) == 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 files left in downloads dir after CancelGroup, got %d", len(entries))
	}
}

func TestDownloader256KBChunkAvoidsLargeRange403AndClearsFailedState(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open error: %v", err)
	}
	defer store.Close()

	payload := bytes.Repeat([]byte("VELORA_256K_"), 75*1024) // ~900 KB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/always403" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var start, end int64
		if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &start, &end); err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if (end - start + 1) > 256*1024 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if end >= int64(len(payload)) {
			end = int64(len(payload)) - 1
		}
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start : end+1])
	}))
	defer srv.Close()

	yt := &YTMusicClient{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("offline test")
			}),
		},
		cache: map[string]cachedStream{},
	}
	yt.mu.Lock()
	yt.cache["vid256Ok:high"] = cachedStream{
		URL:           srv.URL + "/ok",
		MimeType:      "audio/mp4",
		UserAgent:     "TestAgent/1.0",
		ContentLength: int64(len(payload)),
		DurationSec:   90,
		ExpiresAt:     time.Now().Add(time.Hour),
	}
	yt.cache["vid403Fail:high"] = cachedStream{
		URL:           srv.URL + "/always403",
		MimeType:      "audio/mp4",
		UserAgent:     "TestAgent/1.0",
		ContentLength: int64(len(payload)),
		DurationSec:   90,
		ExpiresAt:     time.Now().Add(time.Hour),
	}
	yt.mu.Unlock()

	trOk, _ := store.UpsertTrack(db.Track{
		SourceType:  "ytmusic",
		ExternalID:  "vid256Ok",
		Title:       "Chunk 256K OK",
		ArtistName:  "Artist",
		DurationSec: 90,
	})
	trFail, _ := store.UpsertTrack(db.Track{
		SourceType:  "ytmusic",
		ExternalID:  "vid403Fail",
		Title:       "",
		ArtistName:  "Artist",
		DurationSec: 90,
	})

	events := make(chan DownloadProgress, 32)
	dm := NewDownloadManager(store, yt, filepath.Join(tmpDir, "downloads"), func(_ string, data any) {
		if p, ok := data.(DownloadProgress); ok && (p.Status == "completed" || p.Status == "failed") {
			events <- p
		}
	})

	if _, err := dm.Enqueue(trOk); err != nil {
		t.Fatalf("Enqueue ok error: %v", err)
	}
	select {
	case p := <-events:
		if p.Status != "completed" {
			t.Fatalf("expected completed for 256KB chunks, got %s (%s)", p.Status, p.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for 256KB chunk download")
	}

	if _, err := dm.Enqueue(trFail); err != nil {
		t.Fatalf("Enqueue fail error: %v", err)
	}
	select {
	case p := <-events:
		if p.Status != "failed" {
			t.Fatalf("expected failed for always403, got %+v", p)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for always403 failure")
	}

	if states := dm.GetStates(); len(states) != 0 {
		t.Fatalf("expected 0 active states after failure, got %+v", states)
	}
}
