package jam

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestJamServerLifecycleAndEndpoints(t *testing.T) {
	cmdCh := make(chan Command, 4)
	srv := NewServer("http://127.0.0.1:12345", func(c Command) {
		cmdCh <- c
	}, nil)

	st, err := srv.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Stop()

	if !st.Running || st.Port <= 0 {
		t.Fatalf("unexpected status: %+v", st)
	}
	if !strings.HasPrefix(st.QRDataURL, "data:image/png;base64,") {
		t.Fatalf("expected QR data URL, got %q", st.QRDataURL)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", st.Port)

	// 1. GET /jam
	resp, err := http.Get(baseURL + "/jam")
	if err != nil {
		t.Fatalf("GET /jam failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /jam status = %d", resp.StatusCode)
	}

	// 2. UpdateState + GET /api/state
	srv.UpdateState(PlaybackSnapshot{
		TrackID:     42,
		Title:       "Test Song",
		Artist:      "Test Artist",
		IsPlaying:   true,
		PositionSec: 12,
		DurationSec: 180,
		Volume:      75,
		RepeatMode:  "off",
		Queue:       []QueueItem{{TrackID: 42, Title: "Test Song", Artist: "Test Artist", DurationSec: 180}},
	})

	stateResp, err := http.Get(baseURL + "/api/state")
	if err != nil {
		t.Fatalf("GET /api/state failed: %v", err)
	}
	var got PlaybackSnapshot
	if err := json.NewDecoder(stateResp.Body).Decode(&got); err != nil {
		_ = stateResp.Body.Close()
		t.Fatalf("decode state failed: %v", err)
	}
	_ = stateResp.Body.Close()
	if got.TrackID != 42 || got.Title != "Test Song" || !got.IsPlaying {
		t.Fatalf("unexpected state: %+v", got)
	}

	// 3. POST /api/playback/seek
	seekResp, err := http.Post(baseURL+"/api/playback/seek", "application/json", bytes.NewBufferString(`{"value":64}`))
	if err != nil {
		t.Fatalf("POST /api/playback/seek failed: %v", err)
	}
	_ = seekResp.Body.Close()
	if seekResp.StatusCode != http.StatusOK {
		t.Fatalf("seek status = %d", seekResp.StatusCode)
	}

	select {
	case cmd := <-cmdCh:
		if cmd.Action != "seek" || cmd.Value != 64 {
			t.Fatalf("unexpected command: %+v", cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for seek command")
	}

	// 4. GET /api/events (SSE)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	sseReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/events", nil)
	sseResp, err := http.DefaultClient.Do(sseReq)
	if err != nil {
		t.Fatalf("GET /api/events failed: %v", err)
	}
	defer sseResp.Body.Close()

	reader := bufio.NewReader(sseResp.Body)
	line1, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line1, "event: state") {
		t.Fatalf("expected SSE event: state, got %q (err=%v)", line1, err)
	}
}
