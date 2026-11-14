package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LyricLine struct {
	TimeSec float64 `json:"timeSec"`
	Text    string  `json:"text"`
}

type TrackLyrics struct {
	TrackID      int64       `json:"trackId"`
	Title        string      `json:"title"`
	Artist       string      `json:"artist"`
	PlainLyrics  string      `json:"plainLyrics"`
	SyncedLyrics []LyricLine `json:"syncedLyrics"`
	Source       string      `json:"source"`
}

type lrclibResponse struct {
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
}

var lrcLineRegex = regexp.MustCompile(`^\[(\d{1,2}):(\d{2})(?:\.(\d{1,3}))?\]\s*(.*)$`)

func parseLRC(raw string) []LyricLine {
	var lines []LyricLine
	for _, row := range strings.Split(raw, "\n") {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		m := lrcLineRegex.FindStringSubmatch(row)
		if len(m) != 5 {
			continue
		}
		mins, _ := strconv.Atoi(m[1])
		secs, _ := strconv.Atoi(m[2])
		ms := 0
		if m[3] != "" {
			msVal, _ := strconv.Atoi(m[3])
			switch len(m[3]) {
			case 1:
				ms = msVal * 100
			case 2:
				ms = msVal * 10
			default:
				ms = msVal
			}
		}
		totalSec := float64(mins*60+secs) + float64(ms)/1000.0
		text := strings.TrimSpace(m[4])
		if text == "" {
			text = "♪"
		}
		lines = append(lines, LyricLine{
			TimeSec: totalSec,
			Text:    text,
		})
	}
	return lines
}

// FetchLyrics queries LRCLIB for synchronized and plain lyrics with graceful offline fallback.
func FetchLyrics(ctx context.Context, trackID int64, title, artist, album string, durationSec int) TrackLyrics {
	out := TrackLyrics{
		TrackID:      trackID,
		Title:        title,
		Artist:       artist,
		SyncedLyrics: []LyricLine{},
		Source:       "lrclib",
	}

	cleanTitle := strings.TrimSpace(title)
	cleanArtist := strings.TrimSpace(artist)
	if cleanTitle == "" {
		return out
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}

	params := url.Values{}
	params.Set("track_name", cleanTitle)
	if cleanArtist != "" && !strings.EqualFold(cleanArtist, "Unknown Artist") {
		params.Set("artist_name", cleanArtist)
	}
	if album != "" && !strings.EqualFold(album, "Single") {
		params.Set("album_name", album)
	}
	if durationSec > 0 {
		params.Set("duration", strconv.Itoa(durationSec))
	}

	if res, ok := queryLRCLIBGet(reqCtx, client, "https://lrclib.net/api/get?"+params.Encode()); ok {
		return populateLyrics(out, res)
	}

	searchQuery := strings.TrimSpace(cleanArtist + " " + cleanTitle)
	searchURL := "https://lrclib.net/api/search?q=" + url.QueryEscape(searchQuery)
	if list, ok := queryLRCLIBSearch(reqCtx, client, searchURL); ok && len(list) > 0 {
		return populateLyrics(out, list[0])
	}

	out.Source = "fallback"
	out.PlainLyrics = fmt.Sprintf("%s — %s\n\nLirik belum tersedia secara otomatis untuk lagu ini.\nKamu tetap dapat menikmati pemutaran audio dan kontrol antrean secara penuh.", cleanTitle, cleanArtist)
	return out
}

func populateLyrics(out TrackLyrics, res lrclibResponse) TrackLyrics {
	if res.Instrumental && strings.TrimSpace(res.PlainLyrics) == "" && strings.TrimSpace(res.SyncedLyrics) == "" {
		out.Source = "instrumental"
		out.PlainLyrics = "[Musik Instrumental]"
		return out
	}
	if strings.TrimSpace(res.SyncedLyrics) != "" {
		out.SyncedLyrics = parseLRC(res.SyncedLyrics)
	}
	if strings.TrimSpace(res.PlainLyrics) != "" {
		out.PlainLyrics = strings.TrimSpace(res.PlainLyrics)
	} else if len(out.SyncedLyrics) > 0 {
		var parts []string
		for _, l := range out.SyncedLyrics {
			parts = append(parts, l.Text)
		}
		out.PlainLyrics = strings.Join(parts, "\n")
	}
	return out
}

func queryLRCLIBGet(ctx context.Context, client *http.Client, endpoint string) (lrclibResponse, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return lrclibResponse{}, false
	}
	req.Header.Set("User-Agent", "VeloraMusicPlayer/1.0 (https://github.com/velora)")
	resp, err := client.Do(req)
	if err != nil {
		return lrclibResponse{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return lrclibResponse{}, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return lrclibResponse{}, false
	}
	var data lrclibResponse
	if json.Unmarshal(raw, &data) != nil {
		return lrclibResponse{}, false
	}
	return data, true
}

func queryLRCLIBSearch(ctx context.Context, client *http.Client, endpoint string) ([]lrclibResponse, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", "VeloraMusicPlayer/1.0 (https://github.com/velora)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, false
	}
	var list []lrclibResponse
	if json.Unmarshal(raw, &list) != nil {
		return nil, false
	}
	return list, true
}
