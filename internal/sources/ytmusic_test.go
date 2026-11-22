package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestResolveStreamWindowAndStartTime(t *testing.T) {
	var seenStartTimes []float64
	yt := &YTMusicClient{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				bodyBytes, _ := io.ReadAll(req.Body)
				var payload map[string]any
				_ = json.Unmarshal(bodyBytes, &payload)
				if st, ok := payload["startTimeSecs"].(float64); ok {
					seenStartTimes = append(seenStartTimes, st)
				} else {
					seenStartTimes = append(seenStartTimes, 0)
				}
				respJSON := `{"playabilityStatus":{"status":"OK"},"videoDetails":{"lengthSeconds":"213"},"streamingData":{"adaptiveFormats":[{"url":"https://example.com/audio.webm","mimeType":"audio/webm; codecs=\"opus\"","bitrate":150000,"contentLength":"3614812"}]}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(respJSON)),
					Header:     make(http.Header),
				}, nil
			}),
		},
		cache: map[string]cachedStream{},
	}

	ctx := context.Background()
	info0, err := yt.ResolveStreamWindow(ctx, "5F28ye50-Kc", "high", 0)
	if err != nil {
		t.Fatalf("ResolveStreamWindow(0) error: %v", err)
	}
	if info0.ContentLength != 3614812 || info0.DurationSec != 213 || info0.MimeType != "audio/webm" {
		t.Fatalf("unexpected StreamInfo: %+v", info0)
	}

	if _, err := yt.ResolveStreamWindow(ctx, "5F28ye50-Kc", "high", 0); err != nil {
		t.Fatalf("cached ResolveStreamWindow(0) error: %v", err)
	}
	if len(seenStartTimes) != 1 {
		t.Fatalf("expected 1 upstream call after cached hit, got %d", len(seenStartTimes))
	}

	if _, err := yt.ResolveStreamWindow(ctx, "5F28ye50-Kc", "high", 60); err != nil {
		t.Fatalf("ResolveStreamWindow(60) error: %v", err)
	}
	if len(seenStartTimes) != 2 || seenStartTimes[1] != 60 {
		t.Fatalf("expected second call with startTimeSecs=60, got %v", seenStartTimes)
	}
}

func TestInvalidateStreamCache(t *testing.T) {
	yt := NewYTMusicClient()
	yt.mu.Lock()
	yt.cache["vid1:high"] = cachedStream{URL: "https://example.com/1", ExpiresAt: time.Now().Add(time.Hour)}
	yt.cache["vid1:low"] = cachedStream{URL: "https://example.com/2", ExpiresAt: time.Now().Add(time.Hour)}
	yt.cache["vid2:high"] = cachedStream{URL: "https://example.com/3", ExpiresAt: time.Now().Add(time.Hour)}
	yt.mu.Unlock()

	yt.InvalidateStreamCache("vid1")

	yt.mu.Lock()
	defer yt.mu.Unlock()
	if _, ok := yt.cache["vid1:high"]; ok {
		t.Fatal("expected vid1:high to be evicted")
	}
	if _, ok := yt.cache["vid1:low"]; ok {
		t.Fatal("expected vid1:low to be evicted")
	}
	if _, ok := yt.cache["vid2:high"]; !ok {
		t.Fatal("expected vid2:high to remain cached")
	}
}

func TestLiveResolveStreamFromHomeFeed(t *testing.T) {
	if os.Getenv("VELORA_LIVE_STREAM_CHECK") != "1" {
		t.Skip("set VELORA_LIVE_STREAM_CHECK=1 to run live InnerTube check")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	yt := NewYTMusicClient()
	feed, err := yt.GetHomeFeed(ctx)
	if err != nil {
		t.Fatalf("GetHomeFeed error: %v", err)
	}

	var vid, title string
	for _, s := range feed.Shelves {
		for _, item := range s.Items {
			if item.VideoID != "" {
				vid = item.VideoID
				title = item.Title
				break
			}
		}
		if vid != "" {
			break
		}
	}
	if vid == "" {
		vid = "5F28ye50-Kc"
		title = "Fallback Home Track"
	}

	info, err := yt.ResolveStreamWindow(ctx, vid, "high", 0)
	if err != nil {
		t.Fatalf("ResolveStreamWindow(%s, 0) error: %v", vid, err)
	}
	t.Logf("Resolved Home videoId=%s (%s) mime=%s len=%d dur=%ds ua=%s", vid, title, info.MimeType, info.ContentLength, info.DurationSec, info.UserAgent)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}
	if info.UserAgent != "" {
		req.Header.Set("User-Agent", info.UserAgent)
	}
	req.Header.Set("Range", "bytes=0-65535")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP Range error: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Stream HTTP Status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected HTTP 200 or 206, got %d", resp.StatusCode)
	}
}

func TestLocaleHeadersAndRegionOverride(t *testing.T) {
	var gotCookie, gotAcceptLang, gotClientName, gotVisitor, gotGL, gotHL string
	yt := &YTMusicClient{
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				gotCookie = req.Header.Get("Cookie")
				gotAcceptLang = req.Header.Get("Accept-Language")
				gotClientName = req.Header.Get("X-YouTube-Client-Name")
				gotVisitor = req.Header.Get("X-Goog-Visitor-Id")
				bodyBytes, _ := io.ReadAll(req.Body)
				var payload map[string]any
				_ = json.Unmarshal(bodyBytes, &payload)
				if ctxMap, ok := payload["context"].(map[string]any); ok {
					if cl, ok := ctxMap["client"].(map[string]any); ok {
						gotGL, _ = cl["gl"].(string)
						gotHL, _ = cl["hl"].(string)
					}
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"contents":{}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		},
		cache:       map[string]cachedStream{},
		visitorData: "CgtfakeVisitorData",
	}

	yt.SetLocale([]string{"JP"}, "ja")
	_, err := yt.postInnerTube(context.Background(), "browse", map[string]any{
		"context":  yt.ytContext(),
		"browseId": "FEmusic_home",
	})
	if err != nil {
		t.Fatalf("postInnerTube error: %v", err)
	}
	if gotCookie != "PREF=hl=ja&gl=JP" {
		t.Fatalf("expected PREF=hl=ja&gl=JP, got %q", gotCookie)
	}
	if gotAcceptLang != "ja-JP,ja;q=0.9" {
		t.Fatalf("expected ja-JP,ja;q=0.9, got %q", gotAcceptLang)
	}
	if gotClientName != "67" {
		t.Fatalf("expected X-YouTube-Client-Name 67, got %q", gotClientName)
	}
	if gotVisitor != "" {
		t.Fatalf("expected empty X-Goog-Visitor-Id when regionOverrideActive, got %q", gotVisitor)
	}
	if gotGL != "JP" || gotHL != "ja" {
		t.Fatalf("expected gl=JP hl=ja, got gl=%q hl=%q", gotGL, gotHL)
	}
}

func TestLiveHomeFeedByCountry(t *testing.T) {
	if os.Getenv("VELORA_LIVE_HOME_CHECK") != "1" {
		t.Skip("set VELORA_LIVE_HOME_CHECK=1 to run live country check")
	}
	for _, pair := range [][2]string{{"ID", "id"}, {"JP", "id"}, {"US", "en"}} {
		yt := NewYTMusicClient()
		yt.SetLocale([]string{pair[0]}, pair[1])
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		feed, err := yt.GetHomeFeed(ctx)
		cancel()
		if err != nil {
			t.Fatalf("GetHomeFeed(%s,%s) error: %v", pair[0], pair[1], err)
		}
		t.Logf("=== COUNTRY %s (hl=%s): %d shelves, %d tracks, cont=%v ===", pair[0], pair[1], len(feed.Shelves), len(feed.Tracks), feed.Continuation != "")
		for i, sh := range feed.Shelves {
			if i >= 6 {
				break
			}
			sample := ""
			for j, it := range sh.Items {
				if j >= 3 {
					break
				}
				if j > 0 {
					sample += " | "
				}
				sample += it.Kind + ":" + it.Title
			}
			t.Logf("  [%d] %s (%d items): %s", i+1, sh.Title, len(sh.Items), sample)
		}
	}
}
