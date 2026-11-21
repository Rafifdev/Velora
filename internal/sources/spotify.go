package sources

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	spotifyPlaylistIDRe = regexp.MustCompile(`(?:open\.spotify\.com/(?:embed/|intl-[a-z-]+/)?playlist/|spotify:playlist:)([A-Za-z0-9]+)`)
	spotifyNextDataRe   = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)
)

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

func ResolveSpotifyTracksToYTMusic(ctx context.Context, yt *YTMusicClient, spTracks []SpotifyTrack) []YTMusicTrack {
	out := make([]YTMusicTrack, len(spTracks))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup

	for i, st := range spTracks {
		wg.Add(1)
		go func(idx int, tr SpotifyTrack) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			subCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			firstArtist := tr.Artist
			if comma := strings.Index(firstArtist, ","); comma > 0 {
				firstArtist = strings.TrimSpace(firstArtist[:comma])
			}

			queries := []string{strings.TrimSpace(tr.Title + " " + tr.Artist)}
			if firstArtist != "" && firstArtist != tr.Artist {
				queries = append(queries, strings.TrimSpace(tr.Title+" "+firstArtist))
			}

			for _, q := range queries {
				if q == "" {
					continue
				}
				if hits, err := yt.Search(subCtx, q); err == nil && len(hits) > 0 {
					for _, h := range hits {
						if strings.TrimSpace(h.VideoID) == "" {
							continue
						}
						best := h
						if best.Album == "" || best.Album == "YouTube Music" {
							best.Album = best.Title
						}
						out[idx] = best
						return
					}
				}
			}

			out[idx] = YTMusicTrack{
				VideoID:     fmt.Sprintf("sp:import:%d:%s", idx, strings.TrimSpace(tr.Title)),
				Title:       tr.Title,
				Artist:      tr.Artist,
				Album:       "Imported from Spotify",
				DurationSec: tr.DurationSec,
				Format:      "OPUS",
			}
		}(i, st)
	}
	wg.Wait()

	filtered := make([]YTMusicTrack, 0, len(out))
	for _, t := range out {
		if t.VideoID != "" && strings.TrimSpace(t.Title) != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

type SpotifyClient struct {
	httpClient    *http.Client
	mu            sync.RWMutex
	clientID      string
	clientSecret  string
	accessToken   string
	tokenExpiry   time.Time
	resolvedCache map[string]string
	region        string
}

func NewSpotifyClient() *SpotifyClient {
	return &SpotifyClient{
		httpClient:    &http.Client{Timeout: 12 * time.Second},
		resolvedCache: map[string]string{},
		region:        "ID",
	}
}

func (c *SpotifyClient) SetRegion(gl string) {
	code := strings.ToUpper(strings.TrimSpace(gl))
	if len(code) != 2 {
		code = "ID"
	}
	c.mu.Lock()
	c.region = code
	c.mu.Unlock()
}

func (c *SpotifyClient) getRegion() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.region) != 2 {
		return "ID"
	}
	return c.region
}

func (c *SpotifyClient) SetCredentials(clientID, clientSecret string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := strings.TrimSpace(clientID)
	sec := strings.TrimSpace(clientSecret)
	if id != c.clientID || sec != c.clientSecret {
		c.clientID = id
		c.clientSecret = sec
		c.accessToken = ""
		c.tokenExpiry = time.Time{}
	}
}

func (c *SpotifyClient) credentials() (string, string) {
	c.mu.RLock()
	id := c.clientID
	sec := c.clientSecret
	c.mu.RUnlock()
	if id == "" {
		id = strings.TrimSpace(os.Getenv("VELORA_SPOTIFY_CLIENT_ID"))
	}
	if sec == "" {
		sec = strings.TrimSpace(os.Getenv("VELORA_SPOTIFY_CLIENT_SECRET"))
	}
	return id, sec
}

func (c *SpotifyClient) IsConfigured() bool {
	id, sec := c.credentials()
	return id != "" && sec != ""
}

func (c *SpotifyClient) getAccessToken(ctx context.Context) (string, error) {
	id, sec := c.credentials()
	if id == "" || sec == "" {
		return "", fmt.Errorf("kredensial Spotify API belum dikonfigurasi")
	}

	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		tok := c.accessToken
		c.mu.RUnlock()
		return tok, nil
	}
	c.mu.RUnlock()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(id + ":" + sec))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("spotify token error HTTP %d", resp.StatusCode)
	}

	var tokResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokResp); err != nil {
		return "", err
	}
	if tokResp.AccessToken == "" {
		return "", fmt.Errorf("spotify access token kosong")
	}
	ttl := time.Duration(tokResp.ExpiresIn-60) * time.Second
	if ttl < time.Minute {
		ttl = 5 * time.Minute
	}

	c.mu.Lock()
	c.accessToken = tokResp.AccessToken
	c.tokenExpiry = time.Now().Add(ttl)
	c.mu.Unlock()
	return tokResp.AccessToken, nil
}

func (c *SpotifyClient) apiGet(ctx context.Context, path string, out any) error {
	tok, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	endpoint := "https://api.spotify.com/v1" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		c.mu.Lock()
		c.accessToken = ""
		c.tokenExpiry = time.Time{}
		c.mu.Unlock()
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify API HTTP %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

type spImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type spArtistSimple struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Images []spImage `json:"images"`
}

type spAlbumSimple struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	ReleaseDate string           `json:"release_date"`
	Images      []spImage        `json:"images"`
	Artists     []spArtistSimple `json:"artists"`
}

type spTrackItem struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	DurationMS int              `json:"duration_ms"`
	Artists    []spArtistSimple `json:"artists"`
	Album      spAlbumSimple    `json:"album"`
}

func bestSPImage(imgs []spImage) string {
	if len(imgs) == 0 {
		return ""
	}
	best := imgs[0].URL
	bestW := imgs[0].Width
	for _, im := range imgs[1:] {
		if im.Width > bestW && im.URL != "" {
			best = im.URL
			bestW = im.Width
		}
	}
	return best
}

func joinSPArtists(artists []spArtistSimple) string {
	names := make([]string, 0, len(artists))
	for _, a := range artists {
		if n := strings.TrimSpace(a.Name); n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return "Unknown Artist"
	}
	return strings.Join(names, ", ")
}

func spYear(releaseDate string) string {
	if len(releaseDate) >= 4 {
		return releaseDate[:4]
	}
	return ""
}

func (t spTrackItem) toYTMusicTrack(fallbackAlbum, fallbackCover string) YTMusicTrack {
	album := strings.TrimSpace(t.Album.Name)
	if album == "" {
		album = fallbackAlbum
	}
	if album == "" {
		album = "Single"
	}
	cover := bestSPImage(t.Album.Images)
	if cover == "" {
		cover = fallbackCover
	}
	dur := t.DurationMS / 1000
	if dur <= 0 {
		dur = 180
	}
	return YTMusicTrack{
		VideoID:     "sp:track:" + t.ID,
		Title:       t.Name,
		Artist:      joinSPArtists(t.Artists),
		Album:       album,
		DurationSec: dur,
		CoverURL:    cover,
		Format:      "AAC",
	}
}

func (c *SpotifyClient) SearchTracks(ctx context.Context, query string) ([]YTMusicTrack, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Top Hits Indonesia"
	}
	var res struct {
		Tracks struct {
			Items []spTrackItem `json:"items"`
		} `json:"tracks"`
	}
	path := fmt.Sprintf("/search?q=%s&type=track&limit=28&market=ID", url.QueryEscape(q))
	if err := c.apiGet(ctx, path, &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicTrack, 0, len(res.Tracks.Items))
	for _, it := range res.Tracks.Items {
		if it.ID == "" || it.Name == "" {
			continue
		}
		out = append(out, it.toYTMusicTrack("", ""))
	}
	return out, nil
}

func (c *SpotifyClient) SearchAlbums(ctx context.Context, query string) ([]YTMusicCard, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Pop Indonesia"
	}
	var res struct {
		Albums struct {
			Items []spAlbumSimple `json:"items"`
		} `json:"albums"`
	}
	path := fmt.Sprintf("/search?q=%s&type=album&limit=28&market=ID", url.QueryEscape(q))
	if err := c.apiGet(ctx, path, &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicCard, 0, len(res.Albums.Items))
	for _, al := range res.Albums.Items {
		if al.ID == "" || al.Name == "" {
			continue
		}
		artist := joinSPArtists(al.Artists)
		sub := artist
		if yr := spYear(al.ReleaseDate); yr != "" {
			sub = artist + " • " + yr
		}
		out = append(out, YTMusicCard{
			ID:       "sp:album:" + al.ID,
			Title:    al.Name,
			Subtitle: sub,
			CoverURL: bestSPImage(al.Images),
			Kind:     "album",
			BrowseID: "sp:album:" + al.ID,
			Artist:   artist,
			Album:    al.Name,
		})
	}
	return out, nil
}

func (c *SpotifyClient) SearchArtists(ctx context.Context, query string) ([]YTMusicCard, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Pop Indonesia"
	}
	var res struct {
		Artists struct {
			Items []spArtistSimple `json:"items"`
		} `json:"artists"`
	}
	path := fmt.Sprintf("/search?q=%s&type=artist&limit=28&market=ID", url.QueryEscape(q))
	if err := c.apiGet(ctx, path, &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicCard, 0, len(res.Artists.Items))
	for _, ar := range res.Artists.Items {
		if ar.ID == "" || ar.Name == "" {
			continue
		}
		out = append(out, YTMusicCard{
			ID:       "sp:artist:" + ar.ID,
			Title:    ar.Name,
			Subtitle: "Artis",
			CoverURL: bestSPImage(ar.Images),
			Kind:     "artist",
			BrowseID: "sp:artist:" + ar.ID,
			Artist:   ar.Name,
		})
	}
	return out, nil
}

func (c *SpotifyClient) SearchPlaylists(ctx context.Context, query string) ([]YTMusicCard, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Top Hits Indonesia"
	}
	var res struct {
		Playlists struct {
			Items []*struct {
				ID     string    `json:"id"`
				Name   string    `json:"name"`
				Images []spImage `json:"images"`
				Owner  struct {
					DisplayName string `json:"display_name"`
				} `json:"owner"`
			} `json:"items"`
		} `json:"playlists"`
	}
	path := fmt.Sprintf("/search?q=%s&type=playlist&limit=16&market=ID", url.QueryEscape(q))
	if err := c.apiGet(ctx, path, &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicCard, 0, len(res.Playlists.Items))
	for _, pl := range res.Playlists.Items {
		if pl == nil || pl.ID == "" || pl.Name == "" {
			continue
		}
		sub := strings.TrimSpace(pl.Owner.DisplayName)
		if sub == "" {
			sub = "Spotify Playlist"
		}
		out = append(out, YTMusicCard{
			ID:         "sp:playlist:" + pl.ID,
			Title:      pl.Name,
			Subtitle:   sub,
			CoverURL:   bestSPImage(pl.Images),
			Kind:       "playlist",
			BrowseID:   "sp:playlist:" + pl.ID,
			PlaylistID: "sp:playlist:" + pl.ID,
		})
	}
	return out, nil
}

func (c *SpotifyClient) GetHomeFeed(ctx context.Context) (YTMusicHomeFeed, error) {
	var (
		wg        sync.WaitGroup
		newAlbums []YTMusicCard
		playlists []YTMusicCard
		popAlbums []YTMusicCard
		tracks    []YTMusicTrack
		firstErr  error
		errMu     sync.Mutex
	)
	setErr := func(err error) {
		if err != nil {
			errMu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			errMu.Unlock()
		}
	}

	wg.Add(4)
	go func() {
		defer wg.Done()
		var res struct {
			Albums struct {
				Items []spAlbumSimple `json:"items"`
			} `json:"albums"`
		}
		if err := c.apiGet(ctx, "/browse/new-releases?limit=14&country="+c.getRegion(), &res); err != nil {
			setErr(err)
			return
		}
		for _, al := range res.Albums.Items {
			if al.ID == "" || al.Name == "" {
				continue
			}
			artist := joinSPArtists(al.Artists)
			newAlbums = append(newAlbums, YTMusicCard{
				ID:       "sp:album:" + al.ID,
				Title:    al.Name,
				Subtitle: artist,
				CoverURL: bestSPImage(al.Images),
				Kind:     "album",
				BrowseID: "sp:album:" + al.ID,
				Artist:   artist,
				Album:    al.Name,
			})
		}
	}()

	go func() {
		defer wg.Done()
		pls, err := c.SearchPlaylists(ctx, "Top Hits Indonesia")
		if err != nil {
			setErr(err)
			return
		}
		playlists = pls
	}()

	go func() {
		defer wg.Done()
		als, err := c.SearchAlbums(ctx, "Pop Indonesia")
		if err != nil {
			setErr(err)
			return
		}
		popAlbums = als
	}()

	go func() {
		defer wg.Done()
		trs, err := c.SearchTracks(ctx, "Top Pop Indonesia")
		if err != nil {
			setErr(err)
			return
		}
		tracks = trs
	}()

	wg.Wait()

	var shelves []YTMusicShelf
	if len(newAlbums) > 0 {
		shelves = append(shelves, YTMusicShelf{Title: "Rilis Terbaru", Items: newAlbums})
	}
	if len(playlists) > 0 {
		shelves = append(shelves, YTMusicShelf{Title: "Playlist Populer", Items: playlists})
	}
	if len(popAlbums) > 0 {
		shelves = append(shelves, YTMusicShelf{Title: "Album Pilihan", Items: popAlbums})
	}
	if len(shelves) == 0 && len(tracks) == 0 {
		if firstErr != nil {
			return YTMusicHomeFeed{}, firstErr
		}
		return YTMusicHomeFeed{}, fmt.Errorf("gagal memuat feed Spotify")
	}
	return YTMusicHomeFeed{Shelves: shelves, Tracks: tracks}, nil
}

func (c *SpotifyClient) GetTrackAlbumID(ctx context.Context, spTrackID string) (string, error) {
	id := strings.TrimPrefix(strings.TrimSpace(spTrackID), "sp:track:")
	if id == "" {
		return "", fmt.Errorf("invalid spotify track id")
	}
	var tr spTrackItem
	if err := c.apiGet(ctx, "/tracks/"+url.PathEscape(id)+"?market=ID", &tr); err != nil {
		return "", err
	}
	if tr.Album.ID == "" {
		return "", fmt.Errorf("album tidak ditemukan")
	}
	return "sp:album:" + tr.Album.ID, nil
}

func (c *SpotifyClient) GetAlbumDetail(ctx context.Context, spAlbumID string) (YTMusicAlbumDetail, error) {
	id := strings.TrimPrefix(strings.TrimSpace(spAlbumID), "sp:album:")
	if id == "" {
		return YTMusicAlbumDetail{}, fmt.Errorf("invalid spotify album id")
	}
	var al struct {
		ID          string           `json:"id"`
		Name        string           `json:"name"`
		ReleaseDate string           `json:"release_date"`
		Images      []spImage        `json:"images"`
		Artists     []spArtistSimple `json:"artists"`
		Tracks      struct {
			Items []spTrackItem `json:"items"`
		} `json:"tracks"`
	}
	if err := c.apiGet(ctx, "/albums/"+url.PathEscape(id)+"?market=ID", &al); err != nil {
		return YTMusicAlbumDetail{}, err
	}

	cover := bestSPImage(al.Images)
	artist := joinSPArtists(al.Artists)
	artistBrowseID := ""
	if len(al.Artists) > 0 && al.Artists[0].ID != "" {
		artistBrowseID = "sp:artist:" + al.Artists[0].ID
	}

	tracks := make([]YTMusicTrack, 0, len(al.Tracks.Items))
	for _, it := range al.Tracks.Items {
		if it.ID == "" || it.Name == "" {
			continue
		}
		tracks = append(tracks, it.toYTMusicTrack(al.Name, cover))
	}

	var artistAlbums []YTMusicCard
	if len(al.Artists) > 0 && al.Artists[0].ID != "" {
		var other struct {
			Items []spAlbumSimple `json:"items"`
		}
		ep := fmt.Sprintf("/artists/%s/albums?include_groups=album,single&limit=10&market=ID", url.PathEscape(al.Artists[0].ID))
		if err := c.apiGet(ctx, ep, &other); err == nil {
			for _, oa := range other.Items {
				if oa.ID == "" || oa.ID == al.ID {
					continue
				}
				artistAlbums = append(artistAlbums, YTMusicCard{
					ID:       "sp:album:" + oa.ID,
					Title:    oa.Name,
					Subtitle: spYear(oa.ReleaseDate),
					CoverURL: bestSPImage(oa.Images),
					Kind:     "album",
					BrowseID: "sp:album:" + oa.ID,
					Artist:   artist,
					Album:    oa.Name,
				})
			}
		}
	}

	year := spYear(al.ReleaseDate)
	sub := artist
	if year != "" {
		sub = artist + " • " + year
	}

	return YTMusicAlbumDetail{
		BrowseID:       "sp:album:" + al.ID,
		Title:          al.Name,
		Artist:         artist,
		ArtistBrowseID: artistBrowseID,
		Year:           year,
		CoverURL:       cover,
		Subtitle:       sub,
		Tracks:         tracks,
		ArtistAlbums:   artistAlbums,
	}, nil
}

func (c *SpotifyClient) GetArtistDetail(ctx context.Context, spArtistID string) (YTMusicArtistDetail, error) {
	id := strings.TrimPrefix(strings.TrimSpace(spArtistID), "sp:artist:")
	if id == "" {
		return YTMusicArtistDetail{}, fmt.Errorf("invalid spotify artist id")
	}
	var (
		wg        sync.WaitGroup
		info      struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Images    []spImage `json:"images"`
			Followers struct {
				Total int64 `json:"total"`
			} `json:"followers"`
		}
		infoErr   error
		topTracks []YTMusicTrack
		albums    []YTMusicCard
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		infoErr = c.apiGet(ctx, "/artists/"+url.PathEscape(id), &info)
	}()
	go func() {
		defer wg.Done()
		var res struct {
			Tracks []spTrackItem `json:"tracks"`
		}
		if err := c.apiGet(ctx, "/artists/"+url.PathEscape(id)+"/top-tracks?market=ID", &res); err == nil {
			for _, it := range res.Tracks {
				if it.ID == "" || it.Name == "" {
					continue
				}
				topTracks = append(topTracks, it.toYTMusicTrack("", ""))
			}
		}
	}()
	go func() {
		defer wg.Done()
		var res struct {
			Items []spAlbumSimple `json:"items"`
		}
		ep := fmt.Sprintf("/artists/%s/albums?include_groups=album,single&limit=16&market=ID", url.PathEscape(id))
		if err := c.apiGet(ctx, ep, &res); err == nil {
			for _, oa := range res.Items {
				if oa.ID == "" || oa.Name == "" {
					continue
				}
				sub := spYear(oa.ReleaseDate)
				if sub == "" {
					sub = "Album"
				}
				albums = append(albums, YTMusicCard{
					ID:       "sp:album:" + oa.ID,
					Title:    oa.Name,
					Subtitle: sub,
					CoverURL: bestSPImage(oa.Images),
					Kind:     "album",
					BrowseID: "sp:album:" + oa.ID,
					Album:    oa.Name,
				})
			}
		}
	}()
	wg.Wait()

	if infoErr != nil && len(topTracks) == 0 && len(albums) == 0 {
		return YTMusicArtistDetail{}, infoErr
	}
	for i := range albums {
		albums[i].Artist = info.Name
	}
	return YTMusicArtistDetail{
		BrowseID:  "sp:artist:" + id,
		Name:      info.Name,
		AvatarURL: bestSPImage(info.Images),
		FansCount: info.Followers.Total,
		TopTracks: topTracks,
		Albums:    albums,
	}, nil
}

func (c *SpotifyClient) GetCollectionTracks(ctx context.Context, spBrowseID string) ([]YTMusicTrack, error) {
	raw := strings.TrimSpace(spBrowseID)
	switch {
	case strings.HasPrefix(raw, "sp:album:"):
		detail, err := c.GetAlbumDetail(ctx, raw)
		if err != nil {
			return nil, err
		}
		return detail.Tracks, nil

	case strings.HasPrefix(raw, "sp:artist:"):
		id := strings.TrimPrefix(raw, "sp:artist:")
		var res struct {
			Tracks []spTrackItem `json:"tracks"`
		}
		if err := c.apiGet(ctx, "/artists/"+url.PathEscape(id)+"/top-tracks?market=ID", &res); err != nil {
			return nil, err
		}
		out := make([]YTMusicTrack, 0, len(res.Tracks))
		for _, it := range res.Tracks {
			if it.ID == "" || it.Name == "" {
				continue
			}
			out = append(out, it.toYTMusicTrack("", ""))
		}
		return out, nil

	case strings.HasPrefix(raw, "sp:playlist:"):
		id := strings.TrimPrefix(raw, "sp:playlist:")
		var res struct {
			Items []struct {
				Track *spTrackItem `json:"track"`
			} `json:"items"`
		}
		if err := c.apiGet(ctx, "/playlists/"+url.PathEscape(id)+"/tracks?limit=30&market=ID", &res); err != nil {
			return nil, err
		}
		out := make([]YTMusicTrack, 0, len(res.Items))
		for _, item := range res.Items {
			if item.Track == nil || item.Track.ID == "" || item.Track.Name == "" {
				continue
			}
			out = append(out, item.Track.toYTMusicTrack("", ""))
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported spotify browseId: %s", raw)
}

func (c *SpotifyClient) ResolveTrackToYTMusic(ctx context.Context, yt *YTMusicClient, spTrackID, title, artist string) (string, error) {
	if yt == nil {
		return "", fmt.Errorf("youtube music client unavailable")
	}
	key := strings.TrimSpace(spTrackID)
	if key != "" {
		c.mu.RLock()
		cached := c.resolvedCache[key]
		c.mu.RUnlock()
		if cached != "" {
			return cached, nil
		}
	}

	q := strings.TrimSpace(title + " " + artist)
	if q == "" {
		return "", fmt.Errorf("judul lagu kosong untuk resolve audio")
	}
	hits, err := yt.Search(ctx, q)
	if err != nil || len(hits) == 0 || strings.TrimSpace(hits[0].VideoID) == "" {
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("audio YouTube Music tidak ditemukan untuk %s", q)
	}
	vid := strings.TrimSpace(hits[0].VideoID)
	if key != "" {
		c.mu.Lock()
		c.resolvedCache[key] = vid
		c.mu.Unlock()
	}
	return vid, nil
}
