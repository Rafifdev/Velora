package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type SpotifyTrack struct {
	Title       string
	Artist      string
	DurationSec int
}

type SpotifyPlaylistData struct {
	ID       string
	Name     string
	CoverURL string
	Tracks   []SpotifyTrack
}

var (
	spotifyPlaylistIDRe = regexp.MustCompile(`(?:open\.spotify\.com/(?:embed/)?playlist/|spotify:playlist:)([A-Za-z0-9]+)`)
	spotifyNextDataRe   = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)
)

// ExtractSpotifyPlaylistID resolves a Spotify playlist URL (including spotify.link redirects) to its playlist ID.
func ExtractSpotifyPlaylistID(ctx context.Context, rawURL string) (string, error) {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return "", fmt.Errorf("URL playlist Spotify tidak boleh kosong")
	}
	if m := spotifyPlaylistIDRe.FindStringSubmatch(u); len(m) > 1 {
		return m[1], nil
	}
	if strings.Contains(u, "spotify.link") || strings.Contains(u, "spoti.fi") {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err == nil {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
			if resp, rErr := client.Do(req); rErr == nil {
				finalURL := resp.Request.URL.String()
				_ = resp.Body.Close()
				if m := spotifyPlaylistIDRe.FindStringSubmatch(finalURL); len(m) > 1 {
					return m[1], nil
				}
			}
		}
	}
	return "", fmt.Errorf("link playlist Spotify tidak valid (contoh: https://open.spotify.com/playlist/...)")
}

// FetchSpotifyPlaylist fetches public playlist metadata and tracks from Spotify's embed page.
func FetchSpotifyPlaylist(ctx context.Context, rawURL string) (SpotifyPlaylistData, error) {
	pid, err := ExtractSpotifyPlaylistID(ctx, rawURL)
	if err != nil {
		return SpotifyPlaylistData{}, err
	}

	embedURL := fmt.Sprintf("https://open.spotify.com/embed/playlist/%s", pid)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, embedURL, nil)
	if err != nil {
		return SpotifyPlaylistData{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return SpotifyPlaylistData{}, fmt.Errorf("gagal mengambil playlist Spotify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SpotifyPlaylistData{}, fmt.Errorf("playlist Spotify tidak ditemukan atau bersifat privat (HTTP %d)", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return SpotifyPlaylistData{}, err
	}

	m := spotifyNextDataRe.FindSubmatch(bodyBytes)
	if len(m) < 2 {
		return SpotifyPlaylistData{}, fmt.Errorf("data playlist Spotify tidak dapat dibaca")
	}

	var payload struct {
		Props struct {
			PageProps struct {
				State struct {
					Data struct {
						Entity struct {
							Name     string `json:"name"`
							Title    string `json:"title"`
							CoverArt struct {
								Sources []struct {
									URL string `json:"url"`
								} `json:"sources"`
							} `json:"coverArt"`
							TrackList []struct {
								Title    string `json:"title"`
								Subtitle string `json:"subtitle"`
								Duration int    `json:"duration"`
							} `json:"trackList"`
						} `json:"entity"`
					} `json:"data"`
				} `json:"state"`
			} `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &payload); err != nil {
		return SpotifyPlaylistData{}, fmt.Errorf("gagal mengurai data Spotify: %w", err)
	}

	ent := payload.Props.PageProps.State.Data.Entity
	name := strings.TrimSpace(ent.Name)
	if name == "" {
		name = strings.TrimSpace(ent.Title)
	}
	if name == "" {
		name = "Spotify Playlist"
	}

	cover := ""
	if len(ent.CoverArt.Sources) > 0 {
		cover = ent.CoverArt.Sources[0].URL
	}

	tracks := make([]SpotifyTrack, 0, len(ent.TrackList))
	for _, t := range ent.TrackList {
		title := strings.TrimSpace(t.Title)
		if title == "" {
			continue
		}
		artist := strings.ReplaceAll(strings.TrimSpace(t.Subtitle), "\u00a0", " ")
		durSec := t.Duration / 1000
		if durSec <= 0 {
			durSec = 200
		}
		tracks = append(tracks, SpotifyTrack{
			Title:       title,
			Artist:      artist,
			DurationSec: durSec,
		})
	}
	if len(tracks) == 0 {
		return SpotifyPlaylistData{}, fmt.Errorf("playlist Spotify kosong atau tidak dapat diakses")
	}

	return SpotifyPlaylistData{
		ID:       pid,
		Name:     name,
		CoverURL: cover,
		Tracks:   tracks,
	}, nil
}

// ResolveSpotifyTracksToYTMusic matches Spotify tracks onto playable YouTube Music tracks concurrently.
func ResolveSpotifyTracksToYTMusic(ctx context.Context, yt *YTMusicClient, spTracks []SpotifyTrack) []YTMusicTrack {
	out := make([]YTMusicTrack, len(spTracks))
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup

	for i, st := range spTracks {
		wg.Add(1)
		go func(idx int, tr SpotifyTrack) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			subCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
			defer cancel()

			q := strings.TrimSpace(tr.Title + " " + tr.Artist)
			if hits, err := yt.Search(subCtx, q); err == nil && len(hits) > 0 {
				best := hits[0]
				if best.Album == "" || best.Album == "YouTube Music" {
					best.Album = best.Title
				}
				out[idx] = best
			}
		}(i, st)
	}
	wg.Wait()

	filtered := make([]YTMusicTrack, 0, len(out))
	for _, t := range out {
		if t.VideoID != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}
