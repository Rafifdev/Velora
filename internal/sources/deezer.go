package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DeezerClient struct {
	httpClient    *http.Client
	mu            sync.RWMutex
	resolvedCache map[string]string
	regions       []string
}

func NewDeezerClient() *DeezerClient {
	return &DeezerClient{
		httpClient:    &http.Client{Timeout: 12 * time.Second},
		resolvedCache: map[string]string{},
		regions:       []string{"ID"},
	}
}

func (c *DeezerClient) SetRegions(regions []string) {
	cleaned := make([]string, 0, len(regions))
	seen := map[string]struct{}{}
	for _, r := range regions {
		code := strings.ToUpper(strings.TrimSpace(r))
		if len(code) != 2 {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		cleaned = append(cleaned, code)
	}
	if len(cleaned) == 0 {
		cleaned = []string{"ID"}
	}
	c.mu.Lock()
	c.regions = cleaned
	c.mu.Unlock()
}

func (c *DeezerClient) getRegions() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.regions) == 0 {
		return []string{"ID"}
	}
	out := make([]string, len(c.regions))
	copy(out, c.regions)
	return out
}

func (c *DeezerClient) countryNameForCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "ID":
		return "Indonesia"
	case "US":
		return "USA"
	case "GB":
		return "UK"
	case "JP":
		return "Japan"
	case "KR":
		return "Korea"
	case "IN":
		return "India"
	case "BR":
		return "Brazil"
	case "DE":
		return "Germany"
	case "FR":
		return "France"
	case "MX":
		return "Mexico"
	case "AU":
		return "Australia"
	case "MY":
		return "Malaysia"
	default:
		return "Indonesia"
	}
}

func (c *DeezerClient) allowsNonLatin(regions []string) bool {
	for _, r := range regions {
		switch strings.ToUpper(strings.TrimSpace(r)) {
		case "JP", "KR", "IN":
			return true
		}
	}
	return false
}

func (c *DeezerClient) apiGet(ctx context.Context, path string, out any) error {
	endpoint := "https://api.deezer.com" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Velora/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("deezer API HTTP %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

type dzArtist struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PictureXL  string `json:"picture_xl"`
	PictureBig string `json:"picture_big"`
	PictureMed string `json:"picture_medium"`
}

func (a dzArtist) bestImage() string {
	for _, u := range []string{a.PictureXL, a.PictureBig, a.PictureMed} {
		if strings.TrimSpace(u) != "" {
			return u
		}
	}
	return ""
}

type dzAlbum struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	CoverXL     string   `json:"cover_xl"`
	CoverBig    string   `json:"cover_big"`
	CoverMedium string   `json:"cover_medium"`
	ReleaseDate string   `json:"release_date"`
	Artist      dzArtist `json:"artist"`
}

func (a dzAlbum) bestCover() string {
	for _, u := range []string{a.CoverXL, a.CoverBig, a.CoverMedium} {
		if strings.TrimSpace(u) != "" {
			return u
		}
	}
	return ""
}

type dzTrack struct {
	ID             int64    `json:"id"`
	Title          string   `json:"title"`
	Duration       int      `json:"duration"`
	ExplicitLyrics bool     `json:"explicit_lyrics"`
	Artist         dzArtist `json:"artist"`
	Album          dzAlbum  `json:"album"`
}

func (t dzTrack) toYTMusicTrack(fallbackAlbum, fallbackCover string) YTMusicTrack {
	album := strings.TrimSpace(t.Album.Title)
	if album == "" {
		album = fallbackAlbum
	}
	if album == "" {
		album = "Single"
	}
	cover := t.Album.bestCover()
	if cover == "" {
		cover = fallbackCover
	}
	artist := strings.TrimSpace(t.Artist.Name)
	if artist == "" {
		artist = "Unknown Artist"
	}
	dur := t.Duration
	if dur <= 0 {
		dur = 180
	}
	return YTMusicTrack{
		VideoID:     "dz:track:" + strconv.FormatInt(t.ID, 10),
		Title:       t.Title,
		Artist:      artist,
		Album:       album,
		DurationSec: dur,
		CoverURL:    cover,
		Format:      "AAC",
		Explicit:    t.ExplicitLyrics,
	}
}

type dzPlaylist struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	PictureXL  string `json:"picture_xl"`
	PictureBig string `json:"picture_big"`
	PictureMed string `json:"picture_medium"`
	User       struct {
		Name string `json:"name"`
	} `json:"user"`
}

func (p dzPlaylist) bestImage() string {
	for _, u := range []string{p.PictureXL, p.PictureBig, p.PictureMed} {
		if strings.TrimSpace(u) != "" {
			return u
		}
	}
	return ""
}

func isLatinText(s string) bool {
	for _, r := range s {
		if r > 0x024F && (r < 0x1E00 || r > 0x1EFF) && (r < 0x2000 || r > 0x206F) {
			return false
		}
	}
	return true
}

func isBackgroundMusicName(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "" {
		return false
	}
	for _, kw := range []string{
		"lofi", "lo-fi", "study", "sleep", "piano", "ambient", "rain",
		"relax", "meditation", "instrumental", "classical", "white noise",
		"asmr", "chill beats", "yoga", "spa", "nature sound", "exam",
		"workout", "running", "gym", "fitness", "cardio", "bpm",
		"dj mix", "nonstop", "compilation", "best selling", "karaoke",
		"tribute", "ringtone",
	} {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}

func spreadAndShuffleTracks(tracks []YTMusicTrack, maxPerArtist int) []YTMusicTrack {
	if len(tracks) <= 1 {
		return tracks
	}
	if maxPerArtist <= 0 {
		maxPerArtist = 2
	}

	shuffled := make([]YTMusicTrack, len(tracks))
	copy(shuffled, tracks)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	seenID := make(map[string]struct{})
	seenCover := make(map[string]struct{})
	seenAlbum := make(map[string]struct{})
	artistCount := make(map[string]int)
	byArtist := make(map[string][]YTMusicTrack)
	var artists []string

	for _, tr := range shuffled {
		if strings.TrimSpace(tr.VideoID) == "" || strings.TrimSpace(tr.Title) == "" {
			continue
		}
		if !isLatinText(tr.Title) || !isLatinText(tr.Artist) {
			continue
		}
		if isBackgroundMusicName(tr.Title) || isBackgroundMusicName(tr.Artist) || isBackgroundMusicName(tr.Album) {
			continue
		}
		if _, dup := seenID[tr.VideoID]; dup {
			continue
		}
		covKey := strings.TrimSpace(tr.CoverURL)
		if covKey != "" {
			if _, dup := seenCover[covKey]; dup {
				continue
			}
		}
		albKey := strings.ToLower(strings.TrimSpace(tr.Album))
		if albKey != "" && albKey != "single" {
			if _, dup := seenAlbum[albKey]; dup {
				continue
			}
		}
		arKey := strings.ToLower(strings.TrimSpace(tr.Artist))
		if arKey == "" {
			arKey = tr.VideoID
		}
		if artistCount[arKey] >= maxPerArtist {
			continue
		}

		seenID[tr.VideoID] = struct{}{}
		if covKey != "" {
			seenCover[covKey] = struct{}{}
		}
		if albKey != "" && albKey != "single" {
			seenAlbum[albKey] = struct{}{}
		}
		artistCount[arKey]++
		if _, exists := byArtist[arKey]; !exists {
			artists = append(artists, arKey)
		}
		byArtist[arKey] = append(byArtist[arKey], tr)
	}

	total := 0
	for _, list := range byArtist {
		total += len(list)
	}
	out := make([]YTMusicTrack, 0, total)
	recent := make([]string, 0, 3)

	for len(out) < total {
		bestArtist := ""
		bestCount := -1

		for _, ar := range artists {
			rem := len(byArtist[ar])
			if rem == 0 {
				continue
			}
			inRecent := false
			for _, r := range recent {
				if r == ar {
					inRecent = true
					break
				}
			}
			if !inRecent && rem > bestCount {
				bestCount = rem
				bestArtist = ar
			}
		}
		if bestArtist == "" {
			last := ""
			if len(recent) > 0 {
				last = recent[len(recent)-1]
			}
			for _, ar := range artists {
				rem := len(byArtist[ar])
				if rem > 0 && ar != last && rem > bestCount {
					bestCount = rem
					bestArtist = ar
				}
			}
		}
		if bestArtist == "" {
			for _, ar := range artists {
				if len(byArtist[ar]) > 0 {
					bestArtist = ar
					break
				}
			}
		}
		if bestArtist == "" {
			break
		}

		out = append(out, byArtist[bestArtist][0])
		byArtist[bestArtist] = byArtist[bestArtist][1:]
		recent = append(recent, bestArtist)
		if len(recent) > 3 {
			recent = recent[1:]
		}
	}
	return out
}

func (c *DeezerClient) fetchTopPlaylistTracks(ctx context.Context, queries []string, maxPlaylists int) []dzTrack {
	if len(queries) == 0 || maxPlaylists <= 0 {
		return nil
	}
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		pls []dzPlaylist
	)
	seenPL := make(map[int64]struct{})
	for _, q := range queries {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()
			var res struct {
				Data []dzPlaylist `json:"data"`
			}
			if err := c.apiGet(ctx, "/search/playlist?q="+url.QueryEscape(query)+"&limit=4", &res); err == nil {
				mu.Lock()
				for _, pl := range res.Data {
					if pl.ID == 0 {
						continue
					}
					if _, ok := seenPL[pl.ID]; ok {
						continue
					}
					seenPL[pl.ID] = struct{}{}
					pls = append(pls, pl)
				}
				mu.Unlock()
			}
		}(q)
	}
	wg.Wait()

	if len(pls) > maxPlaylists {
		pls = pls[:maxPlaylists]
	}

	buckets := make([][]dzTrack, len(pls))
	var twg sync.WaitGroup
	for i, pl := range pls {
		twg.Add(1)
		go func(idx int, pid int64) {
			defer twg.Done()
			var res struct {
				Data []dzTrack `json:"data"`
			}
			if err := c.apiGet(ctx, fmt.Sprintf("/playlist/%d/tracks?limit=50", pid), &res); err == nil {
				buckets[idx] = res.Data
			}
		}(i, pl.ID)
	}
	twg.Wait()

	var out []dzTrack
	maxLen := 0
	for _, b := range buckets {
		if len(b) > maxLen {
			maxLen = len(b)
		}
	}
	for i := 0; i < maxLen; i++ {
		for _, b := range buckets {
			if i < len(b) {
				out = append(out, b[i])
			}
		}
	}
	return out
}

func (c *DeezerClient) hydrateArtistImages(ctx context.Context, cards []YTMusicCard) {
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i := range cards {
		if strings.TrimSpace(cards[i].CoverURL) != "" || !strings.HasPrefix(cards[i].BrowseID, "dz:artist:") {
			continue
		}
		wg.Add(1)
		go func(idx int, idStr string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var ar dzArtist
			if err := c.apiGet(ctx, "/artist/"+url.PathEscape(idStr), &ar); err == nil {
				if img := ar.bestImage(); img != "" {
					cards[idx].CoverURL = img
				}
			}
		}(i, strings.TrimPrefix(cards[i].BrowseID, "dz:artist:"))
	}
	wg.Wait()
}

func (c *DeezerClient) SearchTracks(ctx context.Context, query string) ([]YTMusicTrack, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		country := c.countryNameForCode(c.getRegions()[0])
		var (
			wg        sync.WaitGroup
			plTracks  []dzTrack
			id1Tracks []dzTrack
			chtTracks []dzTrack
			firstErr  error
		)
		wg.Add(3)
		go func() {
			defer wg.Done()
			plTracks = c.fetchTopPlaylistTracks(ctx, []string{"Top Hits " + country, "Today's Top Hits"}, 4)
		}()
		go func() {
			defer wg.Done()
			var res struct {
				Data []dzTrack `json:"data"`
			}
			if err := c.apiGet(ctx, "/search/track?q="+url.QueryEscape("Top Hits "+country)+"&limit=28", &res); err == nil {
				id1Tracks = res.Data
			} else if firstErr == nil {
				firstErr = err
			}
		}()
		go func() {
			defer wg.Done()
			var res struct {
				Data []dzTrack `json:"data"`
			}
			if err := c.apiGet(ctx, "/chart/0/tracks?limit=40", &res); err == nil {
				chtTracks = res.Data
			} else if firstErr == nil {
				firstErr = err
			}
		}()
		wg.Wait()

		buckets := [][]dzTrack{plTracks, id1Tracks, chtTracks}
		var interleaved []dzTrack
		maxLen := 0
		for _, b := range buckets {
			if len(b) > maxLen {
				maxLen = len(b)
			}
		}
		for i := 0; i < maxLen; i++ {
			for _, b := range buckets {
				if i < len(b) {
					interleaved = append(interleaved, b[i])
				}
			}
		}

		seenID := make(map[int64]struct{})
		seenCover := make(map[string]struct{})
		seenAlbum := make(map[string]struct{})
		artistCount := make(map[string]int)
		var out []YTMusicTrack
		for _, it := range interleaved {
			if it.ID == 0 || strings.TrimSpace(it.Title) == "" {
				continue
			}
			if !isLatinText(it.Title) || !isLatinText(it.Artist.Name) || isBackgroundMusicName(it.Title) || isBackgroundMusicName(it.Artist.Name) {
				continue
			}
			if _, dup := seenID[it.ID]; dup {
				continue
			}
			yt := it.toYTMusicTrack("", "")
			covKey := strings.TrimSpace(yt.CoverURL)
			albKey := strings.ToLower(strings.TrimSpace(yt.Album))
			arKey := strings.ToLower(strings.TrimSpace(yt.Artist))
			if covKey != "" {
				if _, dup := seenCover[covKey]; dup {
					continue
				}
			}
			if albKey != "" && albKey != "single" {
				if _, dup := seenAlbum[albKey]; dup {
					continue
				}
			}
			if artistCount[arKey] >= 2 {
				continue
			}
			seenID[it.ID] = struct{}{}
			if covKey != "" {
				seenCover[covKey] = struct{}{}
			}
			if albKey != "" {
				seenAlbum[albKey] = struct{}{}
			}
			artistCount[arKey]++
			out = append(out, yt)
			if len(out) >= 35 {
				break
			}
		}
		if len(out) == 0 && firstErr != nil {
			return nil, firstErr
		}
		return spreadAndShuffleTracks(out, 2), nil
	}

	var res struct {
		Data []dzTrack `json:"data"`
	}
	if err := c.apiGet(ctx, "/search/track?q="+url.QueryEscape(q)+"&limit=35", &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicTrack, 0, len(res.Data))
	for _, it := range res.Data {
		if it.ID == 0 || strings.TrimSpace(it.Title) == "" {
			continue
		}
		out = append(out, it.toYTMusicTrack("", ""))
	}
	return spreadAndShuffleTracks(out, 2), nil
}

func (c *DeezerClient) SearchAlbums(ctx context.Context, query string) ([]YTMusicCard, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		country := c.countryNameForCode(c.getRegions()[0])
		var (
			wg        sync.WaitGroup
			plTracks  []dzTrack
			idAlbums  []dzAlbum
			chtAlbums []dzAlbum
			firstErr  error
		)
		wg.Add(3)
		go func() {
			defer wg.Done()
			plTracks = c.fetchTopPlaylistTracks(ctx, []string{"Top Hits " + country, "Pop " + country}, 4)
		}()
		go func() {
			defer wg.Done()
			var res struct {
				Data []dzAlbum `json:"data"`
			}
			if err := c.apiGet(ctx, "/search/album?q="+url.QueryEscape("Pop "+country)+"&limit=28", &res); err == nil {
				idAlbums = res.Data
			} else if firstErr == nil {
				firstErr = err
			}
		}()
		go func() {
			defer wg.Done()
			var res struct {
				Data []dzAlbum `json:"data"`
			}
			if err := c.apiGet(ctx, "/chart/0/albums?limit=40", &res); err == nil {
				chtAlbums = res.Data
			} else if firstErr == nil {
				firstErr = err
			}
		}()
		wg.Wait()

		var plAlbums []dzAlbum
		for _, tr := range plTracks {
			if tr.Album.ID > 0 && strings.TrimSpace(tr.Album.Title) != "" {
				al := tr.Album
				if strings.TrimSpace(al.Artist.Name) == "" {
					al.Artist = tr.Artist
				}
				plAlbums = append(plAlbums, al)
			}
		}

		seen := make(map[int64]struct{})
		seenCover := make(map[string]struct{})
		var out []YTMusicCard
		allAlbums := append(plAlbums, idAlbums...)
		allAlbums = append(allAlbums, chtAlbums...)
		for _, al := range allAlbums {
			if al.ID == 0 || strings.TrimSpace(al.Title) == "" {
				continue
			}
			if !isLatinText(al.Title) || !isLatinText(al.Artist.Name) || isBackgroundMusicName(al.Title) || isBackgroundMusicName(al.Artist.Name) {
				continue
			}
			if _, dup := seen[al.ID]; dup {
				continue
			}
			cov := al.bestCover()
			if cov != "" {
				if _, dup := seenCover[cov]; dup {
					continue
				}
				seenCover[cov] = struct{}{}
			}
			seen[al.ID] = struct{}{}
			bid := "dz:album:" + strconv.FormatInt(al.ID, 10)
			artist := strings.TrimSpace(al.Artist.Name)
			out = append(out, YTMusicCard{
				ID:       bid,
				Title:    al.Title,
				Subtitle: artist,
				CoverURL: cov,
				Kind:     "album",
				BrowseID: bid,
				Artist:   artist,
				Album:    al.Title,
			})
		}
		if len(out) == 0 && firstErr != nil {
			return nil, firstErr
		}
		return out, nil
	}

	var res struct {
		Data []dzAlbum `json:"data"`
	}
	if err := c.apiGet(ctx, "/search/album?q="+url.QueryEscape(q)+"&limit=28", &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicCard, 0, len(res.Data))
	for _, al := range res.Data {
		if al.ID == 0 || strings.TrimSpace(al.Title) == "" {
			continue
		}
		bid := "dz:album:" + strconv.FormatInt(al.ID, 10)
		artist := strings.TrimSpace(al.Artist.Name)
		out = append(out, YTMusicCard{
			ID:       bid,
			Title:    al.Title,
			Subtitle: artist,
			CoverURL: al.bestCover(),
			Kind:     "album",
			BrowseID: bid,
			Artist:   artist,
			Album:    al.Title,
		})
	}
	return out, nil
}

func (c *DeezerClient) SearchArtists(ctx context.Context, query string) ([]YTMusicCard, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		var (
			wg         sync.WaitGroup
			plTracks   []dzTrack
			chtTracks  []dzTrack
			popArtists []dzArtist
			chtArtists []dzArtist
			firstErr   error
		)
		wg.Add(4)
		go func() {
			defer wg.Done()
			plTracks = c.fetchTopPlaylistTracks(ctx, []string{"Top Hits Indonesia", "Today's Top Hits", "Pop Hits"}, 5)
		}()
		go func() {
			defer wg.Done()
			var res struct {
				Data []dzTrack `json:"data"`
			}
			if err := c.apiGet(ctx, "/chart/0/tracks?limit=50", &res); err == nil {
				chtTracks = res.Data
			}
		}()
		go func() {
			defer wg.Done()
			var res struct {
				Data []dzArtist `json:"data"`
			}
			if err := c.apiGet(ctx, "/search/artist?q="+url.QueryEscape("Pop Indonesia")+"&limit=28", &res); err == nil {
				popArtists = res.Data
			} else if firstErr == nil {
				firstErr = err
			}
		}()
		go func() {
			defer wg.Done()
			var res struct {
				Data []dzArtist `json:"data"`
			}
			if err := c.apiGet(ctx, "/chart/0/artists?limit=40", &res); err == nil {
				chtArtists = res.Data
			} else if firstErr == nil {
				firstErr = err
			}
		}()
		wg.Wait()

		seen := make(map[int64]struct{})
		var out []YTMusicCard
		var all []dzArtist
		for _, tr := range plTracks {
			if tr.Artist.ID > 0 {
				all = append(all, tr.Artist)
			}
		}
		for _, tr := range chtTracks {
			if tr.Artist.ID > 0 {
				all = append(all, tr.Artist)
			}
		}
		all = append(all, popArtists...)
		all = append(all, chtArtists...)

		for _, ar := range all {
			if ar.ID == 0 || strings.TrimSpace(ar.Name) == "" {
				continue
			}
			if !isLatinText(ar.Name) || isBackgroundMusicName(ar.Name) {
				continue
			}
			if _, dup := seen[ar.ID]; dup {
				continue
			}
			seen[ar.ID] = struct{}{}
			bid := "dz:artist:" + strconv.FormatInt(ar.ID, 10)
			out = append(out, YTMusicCard{
				ID:       bid,
				Title:    ar.Name,
				Subtitle: "Artis",
				CoverURL: ar.bestImage(),
				Kind:     "artist",
				BrowseID: bid,
				Artist:   ar.Name,
			})
			if len(out) >= 35 {
				break
			}
		}
		if len(out) == 0 && firstErr != nil {
			return nil, firstErr
		}
		c.hydrateArtistImages(ctx, out)
		return out, nil
	}

	var res struct {
		Data []dzArtist `json:"data"`
	}
	if err := c.apiGet(ctx, "/search/artist?q="+url.QueryEscape(q)+"&limit=28", &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicCard, 0, len(res.Data))
	for _, ar := range res.Data {
		if ar.ID == 0 || strings.TrimSpace(ar.Name) == "" {
			continue
		}
		bid := "dz:artist:" + strconv.FormatInt(ar.ID, 10)
		out = append(out, YTMusicCard{
			ID:       bid,
			Title:    ar.Name,
			Subtitle: "Artis",
			CoverURL: ar.bestImage(),
			Kind:     "artist",
			BrowseID: bid,
			Artist:   ar.Name,
		})
	}
	return out, nil
}

func (c *DeezerClient) GetMoreArtists(ctx context.Context, offset int, excludeBrowseIDs []string) ([]YTMusicCard, error) {
	seen := make(map[string]struct{}, len(excludeBrowseIDs))
	for _, id := range excludeBrowseIDs {
		if k := strings.TrimSpace(id); k != "" {
			seen[k] = struct{}{}
		}
	}

	var out []YTMusicCard
	appendArtists := func(list []dzArtist) {
		for _, ar := range list {
			if ar.ID == 0 || strings.TrimSpace(ar.Name) == "" {
				continue
			}
			if !isLatinText(ar.Name) || isBackgroundMusicName(ar.Name) {
				continue
			}
			bid := "dz:artist:" + strconv.FormatInt(ar.ID, 10)
			if _, dup := seen[bid]; dup {
				continue
			}
			seen[bid] = struct{}{}
			out = append(out, YTMusicCard{
				ID:       bid,
				Title:    ar.Name,
				Subtitle: "Artis",
				CoverURL: ar.bestImage(),
				Kind:     "artist",
				BrowseID: bid,
				Artist:   ar.Name,
			})
		}
	}

	if offset < 0 {
		offset = 0
	}
	var chartTracksRes struct {
		Data []dzTrack `json:"data"`
	}
	if err := c.apiGet(ctx, fmt.Sprintf("/chart/0/tracks?index=%d&limit=40", offset), &chartTracksRes); err == nil {
		var trArtists []dzArtist
		for _, tr := range chartTracksRes.Data {
			if tr.Artist.ID > 0 {
				trArtists = append(trArtists, tr.Artist)
			}
		}
		appendArtists(trArtists)
	}

	var chartRes struct {
		Data []dzArtist `json:"data"`
	}
	if err := c.apiGet(ctx, fmt.Sprintf("/chart/0/artists?index=%d&limit=28", offset), &chartRes); err == nil {
		appendArtists(chartRes.Data)
	}

	genreIDs := []int{132, 165, 85, 116, 152, 113}
	for i := 0; i < len(genreIDs) && len(out) < 14; i++ {
		gid := genreIDs[(offset/28+i)%len(genreIDs)]
		var gRes struct {
			Data []dzArtist `json:"data"`
		}
		if err := c.apiGet(ctx, fmt.Sprintf("/genre/%d/artists", gid), &gRes); err == nil {
			appendArtists(gRes.Data)
		}
	}

	c.hydrateArtistImages(ctx, out)
	return out, nil
}

func (c *DeezerClient) SearchPlaylists(ctx context.Context, query string) ([]YTMusicCard, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "Top Hits " + c.countryNameForCode(c.getRegions()[0])
	}
	var res struct {
		Data []dzPlaylist `json:"data"`
	}
	if err := c.apiGet(ctx, "/search/playlist?q="+url.QueryEscape(q)+"&limit=16", &res); err != nil {
		return nil, err
	}
	out := make([]YTMusicCard, 0, len(res.Data))
	for _, pl := range res.Data {
		if pl.ID == 0 || strings.TrimSpace(pl.Title) == "" {
			continue
		}
		bid := "dz:playlist:" + strconv.FormatInt(pl.ID, 10)
		sub := strings.TrimSpace(pl.User.Name)
		if sub == "" {
			sub = "Deezer Playlist"
		}
		out = append(out, YTMusicCard{
			ID:         bid,
			Title:      pl.Title,
			Subtitle:   sub,
			CoverURL:   pl.bestImage(),
			Kind:       "playlist",
			BrowseID:   bid,
			PlaylistID: bid,
		})
	}
	return out, nil
}

func (c *DeezerClient) GetHomeFeed(ctx context.Context) (YTMusicHomeFeed, error) {
	regions := c.getRegions()
	allowNonLatin := c.allowsNonLatin(regions)

	var chart struct {
		Tracks struct {
			Data []dzTrack `json:"data"`
		} `json:"tracks"`
		Albums struct {
			Data []dzAlbum `json:"data"`
		} `json:"albums"`
		Artists struct {
			Data []dzArtist `json:"data"`
		} `json:"artists"`
		Playlists struct {
			Data []dzPlaylist `json:"data"`
		} `json:"playlists"`
	}
	var extraArtists struct {
		Data []dzArtist `json:"data"`
	}
	type regionData struct {
		playlists []dzPlaylist
		tracks    []dzTrack
	}
	regResults := make([]regionData, len(regions))

	var (
		wg       sync.WaitGroup
		chartErr error
	)
	wg.Add(2 + len(regions)*2)
	go func() {
		defer wg.Done()
		chartErr = c.apiGet(ctx, "/chart/0", &chart)
	}()
	go func() {
		defer wg.Done()
		_ = c.apiGet(ctx, "/chart/0/artists?limit=28", &extraArtists)
	}()
	for i, reg := range regions {
		country := c.countryNameForCode(reg)
		go func(idx int, countryName string) {
			defer wg.Done()
			var res struct {
				Data []dzPlaylist `json:"data"`
			}
			if err := c.apiGet(ctx, "/search/playlist?q="+url.QueryEscape("Top 50 "+countryName)+"&limit=6", &res); err == nil {
				regResults[idx].playlists = res.Data
			}
		}(i, country)
		go func(idx int, countryName string) {
			defer wg.Done()
			var res struct {
				Data []dzTrack `json:"data"`
			}
			if err := c.apiGet(ctx, "/search/track?q="+url.QueryEscape("top hits "+countryName)+"&limit=14", &res); err == nil {
				regResults[idx].tracks = res.Data
			}
		}(i, country)
	}
	wg.Wait()
	if chartErr != nil && len(extraArtists.Data) == 0 {
		return YTMusicHomeFeed{}, chartErr
	}

	var shelves []YTMusicShelf

	if len(chart.Albums.Data) > 0 {
		items := make([]YTMusicCard, 0, len(chart.Albums.Data))
		for _, al := range chart.Albums.Data {
			if al.ID == 0 || strings.TrimSpace(al.Title) == "" {
				continue
			}
			if (!allowNonLatin && (!isLatinText(al.Title) || !isLatinText(al.Artist.Name))) || isBackgroundMusicName(al.Title) || isBackgroundMusicName(al.Artist.Name) {
				continue
			}
			bid := "dz:album:" + strconv.FormatInt(al.ID, 10)
			artist := strings.TrimSpace(al.Artist.Name)
			items = append(items, YTMusicCard{
				ID:       bid,
				Title:    al.Title,
				Subtitle: artist,
				CoverURL: al.bestCover(),
				Kind:     "album",
				BrowseID: bid,
				Artist:   artist,
				Album:    al.Title,
			})
		}
		if len(items) > 0 {
			shelves = append(shelves, YTMusicShelf{Title: "Album Populer", Items: items})
		}
	}

	var playlistPool []dzPlaylist
	for _, rr := range regResults {
		playlistPool = append(playlistPool, rr.playlists...)
	}
	playlistPool = append(playlistPool, chart.Playlists.Data...)
	if len(playlistPool) > 0 {
		seenPl := make(map[int64]struct{}, len(playlistPool))
		items := make([]YTMusicCard, 0, len(playlistPool))
		for _, pl := range playlistPool {
			if pl.ID == 0 || strings.TrimSpace(pl.Title) == "" {
				continue
			}
			if _, dup := seenPl[pl.ID]; dup {
				continue
			}
			seenPl[pl.ID] = struct{}{}
			bid := "dz:playlist:" + strconv.FormatInt(pl.ID, 10)
			sub := strings.TrimSpace(pl.User.Name)
			if sub == "" {
				sub = "Deezer Playlist"
			}
			items = append(items, YTMusicCard{
				ID:         bid,
				Title:      pl.Title,
				Subtitle:   sub,
				CoverURL:   pl.bestImage(),
				Kind:       "playlist",
				BrowseID:   bid,
				PlaylistID: bid,
			})
		}
		if len(items) > 0 {
			shelves = append(shelves, YTMusicShelf{Title: "Playlist Pilihan", Items: items})
		}
	}

	var artistPool []dzArtist
	for _, rr := range regResults {
		for _, tr := range rr.tracks {
			if tr.Artist.ID > 0 {
				artistPool = append(artistPool, tr.Artist)
			}
		}
	}
	for _, tr := range chart.Tracks.Data {
		if tr.Artist.ID > 0 {
			artistPool = append(artistPool, tr.Artist)
		}
	}
	artistPool = append(artistPool, extraArtists.Data...)
	artistPool = append(artistPool, chart.Artists.Data...)
	if len(artistPool) > 0 {
		seenAr := make(map[int64]struct{})
		items := make([]YTMusicCard, 0, 28)
		for _, ar := range artistPool {
			if ar.ID == 0 || strings.TrimSpace(ar.Name) == "" {
				continue
			}
			if (!allowNonLatin && !isLatinText(ar.Name)) || isBackgroundMusicName(ar.Name) {
				continue
			}
			if _, dup := seenAr[ar.ID]; dup {
				continue
			}
			seenAr[ar.ID] = struct{}{}
			bid := "dz:artist:" + strconv.FormatInt(ar.ID, 10)
			items = append(items, YTMusicCard{
				ID:       bid,
				Title:    ar.Name,
				Subtitle: "Artis",
				CoverURL: ar.bestImage(),
				Kind:     "artist",
				BrowseID: bid,
				Artist:   ar.Name,
			})
			if len(items) >= 28 {
				break
			}
		}
		if len(items) > 0 {
			c.hydrateArtistImages(ctx, items)
			shelves = append(shelves, YTMusicShelf{Title: "Artis Populer", Items: items})
		}
	}

	var trackPool []dzTrack
	for _, rr := range regResults {
		trackPool = append(trackPool, rr.tracks...)
	}
	trackPool = append(trackPool, chart.Tracks.Data...)

	seenTr := make(map[int64]struct{}, len(trackPool))
	tracks := make([]YTMusicTrack, 0, len(trackPool))
	for _, tr := range trackPool {
		if tr.ID == 0 || strings.TrimSpace(tr.Title) == "" {
			continue
		}
		if _, dup := seenTr[tr.ID]; dup {
			continue
		}
		seenTr[tr.ID] = struct{}{}
		tracks = append(tracks, tr.toYTMusicTrack("", ""))
	}
	tracks = spreadAndShuffleTracks(tracks, 2)

	if len(shelves) == 0 && len(tracks) == 0 {
		return YTMusicHomeFeed{}, fmt.Errorf("gagal memuat chart Deezer")
	}
	return YTMusicHomeFeed{Shelves: shelves, Tracks: tracks}, nil
}

func (c *DeezerClient) GetMoreTopSongs(ctx context.Context, offset int, excludeIDs []string, targetCount int) ([]YTMusicTrack, error) {
	if targetCount <= 0 {
		targetCount = 35
	}
	if offset < 0 {
		offset = 0
	}

	excluded := make(map[string]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		if k := strings.TrimSpace(id); k != "" {
			excluded[k] = struct{}{}
		}
	}

	playlistPools := [][]string{
		{"Today's Top Hits", "Top Hits Indonesia"},
		{"Global Top 50", "Hot Hits Indonesia"},
		{"Pop Rising", "Indo Pop"},
		{"Viral Hits", "R&B Hits"},
		{"Pop Rock Hits", "Throwback Hits"},
	}
	plQueries := playlistPools[(offset/20)%len(playlistPools)]
	chartIndex := (offset * 2) % 150
	artistIndex := (offset) % 80

	var (
		wg          sync.WaitGroup
		chartTracks []dzTrack
		plTracks    []dzTrack
		artistTop   []YTMusicTrack
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		var res struct {
			Data []dzTrack `json:"data"`
		}
		if err := c.apiGet(ctx, fmt.Sprintf("/chart/0/tracks?limit=50&index=%d", chartIndex), &res); err == nil {
			chartTracks = res.Data
		}
	}()
	go func() {
		defer wg.Done()
		plTracks = c.fetchTopPlaylistTracks(ctx, plQueries, 4)
	}()
	go func() {
		defer wg.Done()
		var arRes struct {
			Data []dzArtist `json:"data"`
		}
		if err := c.apiGet(ctx, fmt.Sprintf("/chart/0/artists?limit=20&index=%d", artistIndex), &arRes); err != nil {
			return
		}
		var awg sync.WaitGroup
		var amu sync.Mutex
		sem := make(chan struct{}, 6)
		for _, ar := range arRes.Data {
			if ar.ID == 0 || !isLatinText(ar.Name) || isBackgroundMusicName(ar.Name) {
				continue
			}
			awg.Add(1)
			go func(aid int64) {
				defer awg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				var topRes struct {
					Data []dzTrack `json:"data"`
				}
				if err := c.apiGet(ctx, fmt.Sprintf("/artist/%d/top?limit=2", aid), &topRes); err == nil {
					amu.Lock()
					for _, tr := range topRes.Data {
						if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
							artistTop = append(artistTop, tr.toYTMusicTrack("", ""))
						}
					}
					amu.Unlock()
				}
			}(ar.ID)
		}
		awg.Wait()
	}()
	wg.Wait()

	var pool []YTMusicTrack
	for _, tr := range plTracks {
		if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
			yt := tr.toYTMusicTrack("", "")
			if _, skip := excluded[yt.VideoID]; !skip {
				pool = append(pool, yt)
			}
		}
	}
	for _, yt := range artistTop {
		if _, skip := excluded[yt.VideoID]; !skip {
			pool = append(pool, yt)
		}
	}
	for _, tr := range chartTracks {
		if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
			yt := tr.toYTMusicTrack("", "")
			if _, skip := excluded[yt.VideoID]; !skip {
				pool = append(pool, yt)
			}
		}
	}

	spread := spreadAndShuffleTracks(pool, 2)
	if len(spread) > targetCount {
		spread = spread[:targetCount]
	}
	return spread, nil
}

func (c *DeezerClient) GetRecommendedTracks(ctx context.Context, seeds []HistorySeed, excludeIDs []string, targetCount int) ([]YTMusicTrack, error) {
	if targetCount <= 0 {
		targetCount = 28
	}

	excluded := make(map[string]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		if k := strings.TrimSpace(id); k != "" {
			excluded[k] = struct{}{}
		}
	}

	var seedArtists []string
	seenSeedArtist := make(map[string]struct{})
	for _, s := range seeds {
		ar := strings.TrimSpace(s.Artist)
		if ar == "" || strings.EqualFold(ar, "Unknown Artist") {
			continue
		}
		if idx := strings.Index(ar, ","); idx > 0 {
			ar = strings.TrimSpace(ar[:idx])
		}
		lk := strings.ToLower(ar)
		if _, ok := seenSeedArtist[lk]; ok {
			continue
		}
		seenSeedArtist[lk] = struct{}{}
		seedArtists = append(seedArtists, ar)
		if len(seedArtists) >= 4 {
			break
		}
	}

	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		seedBuckets = make([][]YTMusicTrack, len(seedArtists))
		chartPool  []YTMusicTrack
		idPool     []YTMusicTrack
		popPool    []YTMusicTrack
	)

	for i, artistName := range seedArtists {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()
			var searchRes struct {
				Data []dzArtist `json:"data"`
			}
			if err := c.apiGet(ctx, "/search/artist?q="+url.QueryEscape(name)+"&limit=1", &searchRes); err != nil || len(searchRes.Data) == 0 {
				return
			}
			aid := searchRes.Data[0].ID
			var bucket []YTMusicTrack

			var topRes struct {
				Data []dzTrack `json:"data"`
			}
			if err := c.apiGet(ctx, fmt.Sprintf("/artist/%d/top?limit=5", aid), &topRes); err == nil {
				for _, tr := range topRes.Data {
					if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
						bucket = append(bucket, tr.toYTMusicTrack("", ""))
					}
				}
			}

			var relRes struct {
				Data []dzArtist `json:"data"`
			}
			if err := c.apiGet(ctx, fmt.Sprintf("/artist/%d/related?limit=3", aid), &relRes); err == nil {
				for _, rel := range relRes.Data {
					var relTop struct {
						Data []dzTrack `json:"data"`
					}
					if err := c.apiGet(ctx, fmt.Sprintf("/artist/%d/top?limit=3", rel.ID), &relTop); err == nil {
						for _, tr := range relTop.Data {
							if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
								bucket = append(bucket, tr.toYTMusicTrack("", ""))
							}
						}
					}
				}
			}

			mu.Lock()
			seedBuckets[idx] = bucket
			mu.Unlock()
		}(i, artistName)
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		var res struct {
			Data []dzTrack `json:"data"`
		}
		if err := c.apiGet(ctx, "/chart/0/tracks?limit=50", &res); err == nil {
			var list []YTMusicTrack
			for _, tr := range res.Data {
				if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
					list = append(list, tr.toYTMusicTrack("", ""))
				}
			}
			mu.Lock()
			chartPool = list
			mu.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		var res struct {
			Data []dzTrack `json:"data"`
		}
		if err := c.apiGet(ctx, "/playlist/1116188761/tracks?limit=50", &res); err == nil && len(res.Data) > 0 {
			var list []YTMusicTrack
			for _, tr := range res.Data {
				if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
					list = append(list, tr.toYTMusicTrack("", ""))
				}
			}
			mu.Lock()
			idPool = list
			mu.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		popArtists := []string{"Bernadya", "Hindia", "Tulus", "Juicy Luicy", "Mahalini", "Nadin Amizah", "LANY", "Bruno Mars", "Billie Eilish", "Sabrina Carpenter", "Taylor Swift", "The Weeknd"}
		var list []YTMusicTrack
		var pMu sync.Mutex
		var pWg sync.WaitGroup
		for _, pa := range popArtists {
			pWg.Add(1)
			go func(aName string) {
				defer pWg.Done()
				var sRes struct {
					Data []dzArtist `json:"data"`
				}
				if err := c.apiGet(ctx, "/search/artist?q="+url.QueryEscape(aName)+"&limit=1", &sRes); err != nil || len(sRes.Data) == 0 {
					return
				}
				var tRes struct {
					Data []dzTrack `json:"data"`
				}
				if err := c.apiGet(ctx, fmt.Sprintf("/artist/%d/top?limit=3", sRes.Data[0].ID), &tRes); err == nil {
					pMu.Lock()
					for _, tr := range tRes.Data {
						if tr.ID > 0 && strings.TrimSpace(tr.Title) != "" {
							list = append(list, tr.toYTMusicTrack("", ""))
						}
					}
					pMu.Unlock()
				}
			}(pa)
		}
		pWg.Wait()
		mu.Lock()
		popPool = list
		mu.Unlock()
	}()
	wg.Wait()

	var allBuckets [][]YTMusicTrack
	for _, b := range seedBuckets {
		if len(b) > 0 {
			allBuckets = append(allBuckets, b)
		}
	}
	if len(idPool) > 0 {
		allBuckets = append(allBuckets, idPool)
	}
	if len(chartPool) > 0 {
		allBuckets = append(allBuckets, chartPool)
	}
	if len(popPool) > 0 {
		allBuckets = append(allBuckets, popPool)
	}

	var interleaved []YTMusicTrack
	maxLen := 0
	for _, b := range allBuckets {
		if len(b) > maxLen {
			maxLen = len(b)
		}
	}
	for i := 0; i < maxLen; i++ {
		for _, b := range allBuckets {
			if i < len(b) {
				interleaved = append(interleaved, b[i])
			}
		}
	}

	out := make([]YTMusicTrack, 0, targetCount)
	seenTitleArtist := make(map[string]struct{})
	albumCount := make(map[string]int)
	artistCount := make(map[string]int)

	tryAdd := func(tr YTMusicTrack, strict bool) bool {
		if len(out) >= targetCount {
			return false
		}
		if _, ex := excluded[tr.VideoID]; ex {
			return false
		}
		taKey := strings.ToLower(strings.TrimSpace(tr.Title) + "|" + strings.TrimSpace(tr.Artist))
		if _, dup := seenTitleArtist[taKey]; dup {
			return false
		}
		arKey := strings.ToLower(strings.TrimSpace(tr.Artist))
		alKey := strings.ToLower(strings.TrimSpace(tr.Album))
		if strict {
			if artistCount[arKey] >= 2 {
				return false
			}
			if alKey != "" && alKey != "single" && albumCount[alKey] >= 1 {
				return false
			}
		}
		excluded[tr.VideoID] = struct{}{}
		seenTitleArtist[taKey] = struct{}{}
		artistCount[arKey]++
		if alKey != "" {
			albumCount[alKey]++
		}
		out = append(out, tr)
		return true
	}

	for _, tr := range interleaved {
		if len(out) >= targetCount {
			break
		}
		tryAdd(tr, true)
	}
	if len(out) < targetCount {
		for _, tr := range interleaved {
			if len(out) >= targetCount {
				break
			}
			tryAdd(tr, false)
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("tidak ada rekomendasi lagu ditemukan")
	}
	return out, nil
}

func (c *DeezerClient) GetTrackAlbumID(ctx context.Context, dzTrackID string) (string, error) {
	id := strings.TrimPrefix(strings.TrimSpace(dzTrackID), "dz:track:")
	if id == "" {
		return "", fmt.Errorf("invalid deezer track id")
	}
	var tr dzTrack
	if err := c.apiGet(ctx, "/track/"+url.PathEscape(id), &tr); err != nil {
		return "", err
	}
	if tr.Album.ID == 0 {
		return "", fmt.Errorf("album Deezer tidak ditemukan")
	}
	return "dz:album:" + strconv.FormatInt(tr.Album.ID, 10), nil
}

func (c *DeezerClient) GetAlbumDetail(ctx context.Context, dzAlbumID string) (YTMusicAlbumDetail, error) {
	id := strings.TrimPrefix(strings.TrimSpace(dzAlbumID), "dz:album:")
	if id == "" {
		return YTMusicAlbumDetail{}, fmt.Errorf("invalid deezer album id")
	}
	var al struct {
		ID          int64    `json:"id"`
		Title       string   `json:"title"`
		CoverXL     string   `json:"cover_xl"`
		CoverBig    string   `json:"cover_big"`
		CoverMedium string   `json:"cover_medium"`
		ReleaseDate string   `json:"release_date"`
		Artist      dzArtist `json:"artist"`
		Tracks      struct {
			Data []dzTrack `json:"data"`
		} `json:"tracks"`
	}
	if err := c.apiGet(ctx, "/album/"+url.PathEscape(id), &al); err != nil {
		return YTMusicAlbumDetail{}, err
	}

	cover := (dzAlbum{CoverXL: al.CoverXL, CoverBig: al.CoverBig, CoverMedium: al.CoverMedium}).bestCover()
	artist := strings.TrimSpace(al.Artist.Name)
	if artist == "" {
		artist = "Unknown Artist"
	}
	artistBrowseID := ""
	if al.Artist.ID > 0 {
		artistBrowseID = "dz:artist:" + strconv.FormatInt(al.Artist.ID, 10)
	}

	tracks := make([]YTMusicTrack, 0, len(al.Tracks.Data))
	for _, it := range al.Tracks.Data {
		if it.ID == 0 || strings.TrimSpace(it.Title) == "" {
			continue
		}
		tracks = append(tracks, it.toYTMusicTrack(al.Title, cover))
	}

	var artistAlbums []YTMusicCard
	if al.Artist.ID > 0 {
		var other struct {
			Data []dzAlbum `json:"data"`
		}
		ep := fmt.Sprintf("/artist/%d/albums?limit=10", al.Artist.ID)
		if err := c.apiGet(ctx, ep, &other); err == nil {
			for _, oa := range other.Data {
				if oa.ID == 0 || oa.ID == al.ID {
					continue
				}
				bid := "dz:album:" + strconv.FormatInt(oa.ID, 10)
				artistAlbums = append(artistAlbums, YTMusicCard{
					ID:       bid,
					Title:    oa.Title,
					Subtitle: spYear(oa.ReleaseDate),
					CoverURL: oa.bestCover(),
					Kind:     "album",
					BrowseID: bid,
					Artist:   artist,
					Album:    oa.Title,
				})
			}
		}
	}

	year := spYear(al.ReleaseDate)
	sub := artist
	if year != "" {
		sub = artist + " • " + year
	}

	bid := "dz:album:" + strconv.FormatInt(al.ID, 10)
	return YTMusicAlbumDetail{
		BrowseID:       bid,
		Title:          al.Title,
		Artist:         artist,
		ArtistBrowseID: artistBrowseID,
		Year:           year,
		CoverURL:       cover,
		Subtitle:       sub,
		Tracks:         tracks,
		ArtistAlbums:   artistAlbums,
	}, nil
}

func (c *DeezerClient) GetArtistDetail(ctx context.Context, dzArtistID string) (YTMusicArtistDetail, error) {
	id := strings.TrimPrefix(strings.TrimSpace(dzArtistID), "dz:artist:")
	if id == "" {
		return YTMusicArtistDetail{}, fmt.Errorf("invalid deezer artist id")
	}

	var (
		wg        sync.WaitGroup
		info      struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			PictureXL  string `json:"picture_xl"`
			PictureBig string `json:"picture_big"`
			PictureMed string `json:"picture_medium"`
			NbFan      int64  `json:"nb_fan"`
		}
		infoErr   error
		topTracks []YTMusicTrack
		albums    []YTMusicCard
		related   []YTMusicCard
	)

	wg.Add(4)
	go func() {
		defer wg.Done()
		infoErr = c.apiGet(ctx, "/artist/"+url.PathEscape(id), &info)
	}()
	go func() {
		defer wg.Done()
		var res struct {
			Data []dzTrack `json:"data"`
		}
		if err := c.apiGet(ctx, "/artist/"+url.PathEscape(id)+"/top?limit=15", &res); err == nil {
			for _, it := range res.Data {
				if it.ID == 0 || strings.TrimSpace(it.Title) == "" {
					continue
				}
				topTracks = append(topTracks, it.toYTMusicTrack("", ""))
			}
		}
	}()
	go func() {
		defer wg.Done()
		var res struct {
			Data []dzAlbum `json:"data"`
		}
		if err := c.apiGet(ctx, "/artist/"+url.PathEscape(id)+"/albums?limit=16", &res); err == nil {
			for _, al := range res.Data {
				if al.ID == 0 || strings.TrimSpace(al.Title) == "" {
					continue
				}
				bid := "dz:album:" + strconv.FormatInt(al.ID, 10)
				sub := spYear(al.ReleaseDate)
				if sub == "" {
					sub = "Album"
				}
				albums = append(albums, YTMusicCard{
					ID:       bid,
					Title:    al.Title,
					Subtitle: sub,
					CoverURL: al.bestCover(),
					Kind:     "album",
					BrowseID: bid,
					Album:    al.Title,
				})
			}
		}
	}()
	go func() {
		defer wg.Done()
		var res struct {
			Data []dzArtist `json:"data"`
		}
		if err := c.apiGet(ctx, "/artist/"+url.PathEscape(id)+"/related?limit=12", &res); err == nil {
			for _, ar := range res.Data {
				if ar.ID == 0 || strings.TrimSpace(ar.Name) == "" {
					continue
				}
				bid := "dz:artist:" + strconv.FormatInt(ar.ID, 10)
				related = append(related, YTMusicCard{
					ID:       bid,
					Title:    ar.Name,
					Subtitle: "Artis",
					CoverURL: ar.bestImage(),
					Kind:     "artist",
					BrowseID: bid,
					Artist:   ar.Name,
				})
			}
		}
	}()
	wg.Wait()

	if infoErr != nil && len(topTracks) == 0 && len(albums) == 0 {
		return YTMusicArtistDetail{}, infoErr
	}
	avatar := (dzArtist{PictureXL: info.PictureXL, PictureBig: info.PictureBig, PictureMed: info.PictureMed}).bestImage()
	name := strings.TrimSpace(info.Name)
	for i := range albums {
		if albums[i].Artist == "" {
			albums[i].Artist = name
		}
	}
	return YTMusicArtistDetail{
		BrowseID:       "dz:artist:" + id,
		Name:           name,
		AvatarURL:      avatar,
		FansCount:      info.NbFan,
		TopTracks:      topTracks,
		Albums:         albums,
		RelatedArtists: related,
	}, nil
}

func (c *DeezerClient) GetCollectionTracks(ctx context.Context, dzBrowseID string) ([]YTMusicTrack, error) {
	raw := strings.TrimSpace(dzBrowseID)
	switch {
	case strings.HasPrefix(raw, "dz:album:"):
		detail, err := c.GetAlbumDetail(ctx, raw)
		if err != nil {
			return nil, err
		}
		return detail.Tracks, nil

	case strings.HasPrefix(raw, "dz:artist:"):
		id := strings.TrimPrefix(raw, "dz:artist:")
		var res struct {
			Data []dzTrack `json:"data"`
		}
		if err := c.apiGet(ctx, "/artist/"+url.PathEscape(id)+"/top?limit=25", &res); err != nil {
			return nil, err
		}
		out := make([]YTMusicTrack, 0, len(res.Data))
		for _, it := range res.Data {
			if it.ID == 0 || strings.TrimSpace(it.Title) == "" {
				continue
			}
			out = append(out, it.toYTMusicTrack("", ""))
		}
		return out, nil

	case strings.HasPrefix(raw, "dz:playlist:"):
		id := strings.TrimPrefix(raw, "dz:playlist:")
		var res struct {
			Data []dzTrack `json:"data"`
		}
		if err := c.apiGet(ctx, "/playlist/"+url.PathEscape(id)+"/tracks?limit=30", &res); err != nil {
			return nil, err
		}
		out := make([]YTMusicTrack, 0, len(res.Data))
		for _, it := range res.Data {
			if it.ID == 0 || strings.TrimSpace(it.Title) == "" {
				continue
			}
			out = append(out, it.toYTMusicTrack("", ""))
		}
		return out, nil
	}
	return nil, fmt.Errorf("unsupported deezer browseId: %s", raw)
}

func (c *DeezerClient) ResolveTrackToYTMusic(ctx context.Context, yt *YTMusicClient, dzTrackID, title, artist string) (string, error) {
	if yt == nil {
		return "", fmt.Errorf("youtube music client unavailable")
	}
	key := strings.TrimSpace(dzTrackID)
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
