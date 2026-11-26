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

	yt := NewYTMusicClient()
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

	yt := NewYTMusicClient()
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
