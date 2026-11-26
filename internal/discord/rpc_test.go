package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNormalizeCoverURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", FallbackCoverURL},
		{"local_media", "http://127.0.0.1:49210/media/cover/abc.jpg", FallbackCoverURL},
		{"data_uri", "data:image/png;base64,AAAA", FallbackCoverURL},
		{
			"wrapped_ytimg",
			"http://127.0.0.1:5000/media/cover?url=https%3A%2F%2Fi.ytimg.com%2Fvi%2FdQw4w9WgXcQ%2Fdefault.jpg",
			"https://i.ytimg.com/vi/dQw4w9WgXcQ/mqdefault.jpg",
		},
		{
			"wrapped_lh3",
			"http://127.0.0.1:5000/media/cover?url=https%3A%2F%2Flh3.googleusercontent.com%2Fabc%3Dw120-h120-c-l85-rj",
			"https://lh3.googleusercontent.com/abc=w512-h512-c-l90-rj",
		},
		{
			"direct_https",
			"https://cdn-images.dzcdn.net/images/cover/1234/500x500.jpg",
			"https://cdn-images.dzcdn.net/images/cover/1234/500x500.jpg",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeCoverURL(tc.in); got != tc.want {
				t.Fatalf("NormalizeCoverURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestClampDiscordText(t *testing.T) {
	if got := clampDiscordText("", "Fallback"); got != "Fallback" {
		t.Fatalf("empty got %q", got)
	}
	if got := clampDiscordText("A", "Fallback"); got != "A " {
		t.Fatalf("single char got %q", got)
	}
	long := strings.Repeat("x", 140)
	got := clampDiscordText(long, "Fallback")
	if len([]rune(got)) != 128 || !strings.HasSuffix(got, "...") {
		t.Fatalf("long clamp got len=%d %q", len([]rune(got)), got)
	}
}

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := map[string]any{"v": 1, "client_id": DefaultClientID}
	if err := writeFrame(&buf, opHandshake, in); err != nil {
		t.Fatalf("writeFrame error: %v", err)
	}
	op, body, err := readFrame(&buf)
	if err != nil {
		t.Fatalf("readFrame error: %v", err)
	}
	if op != opHandshake {
		t.Fatalf("opcode = %d, want %d", op, opHandshake)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if out["client_id"] != DefaultClientID {
		t.Fatalf("client_id = %v, want %s", out["client_id"], DefaultClientID)
	}
}

func TestBuildSetActivityPayload(t *testing.T) {
	now := int64(1750000000)
	playing := buildSetActivityPayload(42, TrackPresence{
		Title:       "Starlight",
		Artist:      "Muse",
		Album:       "Black Holes and Revelations",
		CoverURL:    "https://i.ytimg.com/vi/dQw4w9WgXcQ/mqdefault.jpg",
		ExternalID:  "dQw4w9WgXcQ",
		IsPlaying:   true,
		PositionSec: 30,
		DurationSec: 240,
	}, now, "n-1")

	args := playing["args"].(map[string]any)
	act := args["activity"].(map[string]any)
	if act["name"] != "Velora" {
		t.Fatalf("activity name = %v, want Velora", act["name"])
	}
	if act["type"] != 2 {
		t.Fatalf("activity type = %v, want 2", act["type"])
	}
	if act["status_display_type"] != 0 {
		t.Fatalf("status_display_type = %v, want 0", act["status_display_type"])
	}
	if _, hasBtns := act["buttons"]; hasBtns {
		t.Fatalf("did not expect buttons in Nuclear-style layout")
	}
	assets := act["assets"].(map[string]any)
	if assets["small_image"] != FallbackCoverURL {
		t.Fatalf("expected small_image corner badge %q, got %v", FallbackCoverURL, assets["small_image"])
	}
	ts, ok := act["timestamps"].(map[string]any)
	if !ok {
		t.Fatalf("expected timestamps when playing")
	}
	if ts["start"] != now-30 || ts["end"] != now+210 {
		t.Fatalf("unexpected timestamps: %+v", ts)
	}

	rawJSON, err := json.Marshal(buildSetActivityPayload(42, TrackPresence{
		Title:       "Starlight",
		Artist:      "Muse",
		Album:       "YouTube Music",
		CoverURL:    "https://i.ytimg.com/vi/dQw4w9WgXcQ/mqdefault.jpg",
		ExternalID:  "dQw4w9WgXcQ",
		IsPlaying:   true,
		PositionSec: 30,
		DurationSec: 240,
	}, now, "n-yt"))
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if strings.Contains(strings.ToLower(string(rawJSON)), "youtube music") {
		t.Fatalf("payload should not contain YouTube Music: %s", string(rawJSON))
	}

	paused := buildSetActivityPayload(42, TrackPresence{
		Title:       "Starlight",
		Artist:      "Muse",
		IsPlaying:   false,
		PositionSec: 30,
		DurationSec: 240,
	}, now, "n-2")
	pausedAct := paused["args"].(map[string]any)["activity"].(map[string]any)
	if _, hasTS := pausedAct["timestamps"]; hasTS {
		t.Fatalf("did not expect timestamps when paused")
	}
}

func TestClientMockIPCExchange(t *testing.T) {
	clientPipe, serverPipe := net.Pipe()
	received := make(chan map[string]any, 4)

	go func() {
		defer serverPipe.Close()
		for {
			op, body, err := readFrame(serverPipe)
			if err != nil {
				return
			}
			var msg map[string]any
			_ = json.Unmarshal(body, &msg)
			received <- msg
			if op == opHandshake {
				_ = writeFrame(serverPipe, opFrame, map[string]any{"evt": "READY"})
			} else {
				_ = writeFrame(serverPipe, opFrame, map[string]any{"cmd": "SET_ACTIVITY"})
			}
		}
	}()

	c := &Client{
		enabled:  true,
		clientID: DefaultClientID,
		dial:     func() (io.ReadWriteCloser, error) { return clientPipe, nil },
		wakeCh:   make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	go c.loop()

	c.UpdatePresence(TrackPresence{
		Title:       "Midnight City",
		Artist:      "M83",
		Album:       "Hurry Up, We're Dreaming",
		IsPlaying:   true,
		PositionSec: 12,
		DurationSec: 243,
	})

	var gotHandshake, gotActivity bool
	deadline := time.After(2 * time.Second)
	for !gotHandshake || !gotActivity {
		select {
		case msg := <-received:
			if msg["client_id"] == DefaultClientID {
				gotHandshake = true
			}
			if msg["cmd"] == "SET_ACTIVITY" {
				gotActivity = true
			}
		case <-deadline:
			c.Shutdown()
			t.Fatalf("timed out waiting for mock IPC messages (handshake=%v activity=%v)", gotHandshake, gotActivity)
		}
	}
	c.Shutdown()
}
