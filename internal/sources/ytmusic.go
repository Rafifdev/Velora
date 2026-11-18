package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type YTMusicTrack struct {
	VideoID     string `json:"videoId"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	DurationSec int    `json:"durationSec"`
	CoverURL      string `json:"coverUrl"`
	Format        string `json:"format"`
	PlayCountText string `json:"playCountText,omitempty"`
}

type YTMusicCard struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	CoverURL    string `json:"coverUrl"`
	Kind        string `json:"kind"` // "playlist" | "album" | "song"
	VideoID     string `json:"videoId"`
	BrowseID    string `json:"browseId"`
	PlaylistID  string `json:"playlistId"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	DurationSec int    `json:"durationSec"`
}

type YTMusicShelf struct {
	Title string        `json:"title"`
	Items []YTMusicCard `json:"items"`
}

type YTMusicHomeFeed struct {
	Shelves []YTMusicShelf `json:"shelves"`
	Tracks  []YTMusicTrack `json:"tracks"`
}

type YTMusicAlbumDetail struct {
	BrowseID       string         `json:"browseId"`
	Title          string         `json:"title"`
	Artist         string         `json:"artist"`
	ArtistBrowseID string         `json:"artistBrowseId"`
	Year           string         `json:"year"`
	CoverURL       string         `json:"coverUrl"`
	Subtitle       string         `json:"subtitle"`
	Tracks         []YTMusicTrack `json:"tracks"`
	ArtistAlbums   []YTMusicCard  `json:"artistAlbums"`
}

type YTMusicArtistDetail struct {
	BrowseID       string         `json:"browseId"`
	Name           string         `json:"name"`
	AvatarURL      string         `json:"avatarUrl"`
	FansCount      int64          `json:"fansCount"`
	TopTracks      []YTMusicTrack `json:"topTracks"`
	Albums         []YTMusicCard  `json:"albums"`
	RelatedArtists []YTMusicCard  `json:"relatedArtists"`
}

type StreamInfo struct {
	URL           string
	MimeType      string
	UserAgent     string
	ContentLength int64
	DurationSec   int
}

type cachedStream struct {
	URL           string
	MimeType      string
	UserAgent     string
	ContentLength int64
	DurationSec   int
	ExpiresAt     time.Time
}

type YTMusicClient struct {
	httpClient    *http.Client
	mu            sync.RWMutex
	cache         map[string]cachedStream
	top100Cache   []YTMusicTrack
	top100Expires time.Time
}

func NewYTMusicClient() *YTMusicClient {
	return &YTMusicClient{
		httpClient: &http.Client{Timeout: 12 * time.Second},
		cache:      map[string]cachedStream{},
	}
}

func (c *YTMusicClient) postInnerTube(ctx context.Context, endpoint string, payload map[string]any) (any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://music.youtube.com/youtubei/v1/"+endpoint+"?prettyPrint=false",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://music.youtube.com")
	req.Header.Set("Referer", "https://music.youtube.com/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("innertube %s status %d", endpoint, resp.StatusCode)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func ytWebRemixContext() map[string]any {
	return map[string]any{
		"client": map[string]any{
			"clientName":    "WEB_REMIX",
			"clientVersion": "1.20241113.01.00",
			"hl":            "id",
			"gl":            "ID",
		},
	}
}

func (c *YTMusicClient) Search(ctx context.Context, query string) ([]YTMusicTrack, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		feed, err := c.GetHomeFeed(ctx)
		return feed.Tracks, err
	}

	payload := map[string]any{
		"context": ytWebRemixContext(),
		"query":   query,
		"params":  "EgWKAQIIAWoMEAMQBBAJEAoQBRAV",
	}

	tree, err := c.postInnerTube(ctx, "search", payload)
	if err != nil {
		return []YTMusicTrack{}, err
	}

	var results []YTMusicTrack
	seen := map[string]struct{}{}
	extractResponsiveItems(tree, &results, seen)

	if len(results) == 0 {
		delete(payload, "params")
		if tree2, err2 := c.postInnerTube(ctx, "search", payload); err2 == nil {
			extractResponsiveItems(tree2, &results, seen)
		}
	}

	if len(results) > 24 {
		results = results[:24]
	}
	if results == nil {
		results = []YTMusicTrack{}
	}
	return results, nil
}

// SearchPlaylists searches YouTube Music for playlists (featured & community).
func (c *YTMusicClient) SearchPlaylists(ctx context.Context, query string) ([]YTMusicCard, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		query = "Top Hits Indonesia Playlist"
	}

	var results []YTMusicCard
	seen := map[string]struct{}{}

	for _, params := range []string{
		"Eg-KAQwIABAAGAAgACgBMABqChAEEAMQCRAFEAo%3D", // Featured playlists
		"EgeKAQQoAEABagoQAxAEEAkQChAF",                 // Community playlists
	} {
		payload := map[string]any{
			"context": ytWebRemixContext(),
			"query":   query,
			"params":  params,
		}
		tree, err := c.postInnerTube(ctx, "search", payload)
		if err != nil {
			continue
		}
		extractPlaylistSearchItems(tree, &results, seen)
		if len(results) >= 24 {
			break
		}
	}

	if len(results) > 24 {
		results = results[:24]
	}
	if results == nil {
		results = []YTMusicCard{}
	}
	return results, nil
}

// SearchAlbums searches YouTube Music for albums (plus new releases when query is empty/default).
func (c *YTMusicClient) SearchAlbums(ctx context.Context, query string) ([]YTMusicCard, error) {
	query = strings.TrimSpace(query)
	var results []YTMusicCard
	seen := map[string]struct{}{}

	if query == "" {
		if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  ytWebRemixContext(),
			"browseId": "FEmusic_new_releases_albums",
		}); err == nil {
			extractAlbumSearchItems(tree, &results, seen)
		}
		query = "Top Pop Indonesia Album"
	}

	if len(results) < 28 {
		payload := map[string]any{
			"context": ytWebRemixContext(),
			"query":   query,
			"params":  "EgWKAQIYAWoMEAMQBBAJEAoQBRAV",
		}
		if tree, err := c.postInnerTube(ctx, "search", payload); err == nil {
			extractAlbumSearchItems(tree, &results, seen)
		}
	}

	if len(results) > 28 {
		results = results[:28]
	}
	if results == nil {
		results = []YTMusicCard{}
	}
	return results, nil
}

// SearchArtists searches YouTube Music for artists (defaulting to Indonesian & English popular artists).
func (c *YTMusicClient) SearchArtists(ctx context.Context, query string) ([]YTMusicCard, error) {
	query = strings.TrimSpace(query)
	queries := []string{query}
	if query == "" {
		queries = []string{"Penyanyi Pop Indonesia", "Popular Pop Artists"}
	}

	var results []YTMusicCard
	seen := map[string]struct{}{}

	for _, q := range queries {
		payload := map[string]any{
			"context": ytWebRemixContext(),
			"query":   q,
			"params":  "EgWKAQIgAWoMEAMQBBAJEAoQBRAV",
		}
		if tree, err := c.postInnerTube(ctx, "search", payload); err == nil {
			extractArtistSearchItems(tree, &results, seen)
		}
	}

	if len(results) > 35 {
		results = results[:35]
	}
	if results == nil {
		results = []YTMusicCard{}
	}
	return results, nil
}

func extractArtistSearchItems(node any, out *[]YTMusicCard, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		if mr, ok := v["musicResponsiveListItemRenderer"].(map[string]any); ok {
			if card, ok := parseResponsiveArtistItem(mr); ok {
				if _, exists := seen[card.ID]; !exists {
					seen[card.ID] = struct{}{}
					*out = append(*out, card)
				}
			}
		}
		for _, child := range v {
			extractArtistSearchItems(child, out, seen)
		}
	case []any:
		for _, child := range v {
			extractArtistSearchItems(child, out, seen)
		}
	}
}

func parseResponsiveArtistItem(r map[string]any) (YTMusicCard, bool) {
	browseID := ""
	pageType := ""
	if nav, ok := r["navigationEndpoint"].(map[string]any); ok {
		if be, ok := nav["browseEndpoint"].(map[string]any); ok {
			browseID, _ = be["browseId"].(string)
			if cfg, ok := be["browseEndpointContextSupportedConfigs"].(map[string]any); ok {
				if mc, ok := cfg["browseEndpointContextMusicConfig"].(map[string]any); ok {
					pageType, _ = mc["pageType"].(string)
				}
			}
		}
	}
	if browseID == "" || (!strings.HasPrefix(browseID, "UC") && pageType != "MUSIC_PAGE_TYPE_ARTIST") {
		return YTMusicCard{}, false
	}

	title := ""
	subtitle := "Artis"
	if cols, ok := r["flexColumns"].([]any); ok {
		if len(cols) > 0 {
			title = extractColumnText(cols[0])
		}
		if len(cols) > 1 {
			if sub := strings.TrimSpace(extractColumnText(cols[1])); sub != "" {
				subtitle = sub
			}
		}
	}
	if title == "" {
		return YTMusicCard{}, false
	}
	cover := findLastThumbnailURL(r)
	return YTMusicCard{
		ID:       browseID,
		Title:    title,
		Subtitle: subtitle,
		CoverURL: cover,
		Kind:     "artist",
		BrowseID: browseID,
		Artist:   title,
	}, true
}

func extractAlbumSearchItems(node any, out *[]YTMusicCard, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		if mr, ok := v["musicResponsiveListItemRenderer"].(map[string]any); ok {
			if card, ok := parseResponsiveAlbumItem(mr); ok {
				if _, dup := seen[card.BrowseID]; !dup {
					seen[card.BrowseID] = struct{}{}
					*out = append(*out, card)
				}
			}
			return
		}
		if tr, ok := v["musicTwoRowItemRenderer"].(map[string]any); ok {
			if card, ok := parseTwoRowItem(tr); ok && card.Kind == "album" && card.BrowseID != "" {
				if _, dup := seen[card.BrowseID]; !dup {
					seen[card.BrowseID] = struct{}{}
					*out = append(*out, card)
				}
			}
			return
		}
		for _, child := range v {
			extractAlbumSearchItems(child, out, seen)
		}
	case []any:
		for _, child := range v {
			extractAlbumSearchItems(child, out, seen)
		}
	}
}

func parseResponsiveAlbumItem(r map[string]any) (YTMusicCard, bool) {
	browseID := ""
	pageType := ""
	playlistID := ""

	if nav, ok := r["navigationEndpoint"].(map[string]any); ok {
		if be, ok := nav["browseEndpoint"].(map[string]any); ok {
			browseID, _ = be["browseId"].(string)
			if cfg, ok := be["browseEndpointContextSupportedConfigs"].(map[string]any); ok {
				if mc, ok := cfg["browseEndpointContextMusicConfig"].(map[string]any); ok {
					pageType, _ = mc["pageType"].(string)
				}
			}
		}
	}
	if overlay, ok := r["overlay"].(map[string]any); ok {
		if to, ok := overlay["musicItemThumbnailOverlayRenderer"].(map[string]any); ok {
			if content, ok := to["content"].(map[string]any); ok {
				if btn, ok := content["musicPlayButtonRenderer"].(map[string]any); ok {
					if pnav, ok := btn["playNavigationEndpoint"].(map[string]any); ok {
						if wpe, ok := pnav["watchPlaylistEndpoint"].(map[string]any); ok {
							if pid, ok := wpe["playlistId"].(string); ok && pid != "" {
								playlistID = pid
							}
						}
					}
				}
			}
		}
	}

	if browseID == "" || (!strings.HasPrefix(browseID, "MPREb_") && pageType != "MUSIC_PAGE_TYPE_ALBUM") {
		return YTMusicCard{}, false
	}

	title := ""
	subtitle := ""
	if cols, ok := r["flexColumns"].([]any); ok {
		if len(cols) > 0 {
			title = extractColumnText(cols[0])
		}
		if len(cols) > 1 {
			subtitle = extractColumnText(cols[1])
		}
	}
	if title == "" {
		return YTMusicCard{}, false
	}
	cover := findLastThumbnailURL(r)
	return YTMusicCard{
		ID:         browseID,
		Title:      title,
		Subtitle:   subtitle,
		CoverURL:   cover,
		Kind:       "album",
		BrowseID:   browseID,
		PlaylistID: playlistID,
	}, true
}

func extractPlaylistSearchItems(node any, out *[]YTMusicCard, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		if mr, ok := v["musicResponsiveListItemRenderer"].(map[string]any); ok {
			if card, ok := parseResponsivePlaylistItem(mr); ok {
				if _, exists := seen[card.ID]; !exists {
					seen[card.ID] = struct{}{}
					*out = append(*out, card)
				}
			}
		}
		if tr, ok := v["musicTwoRowItemRenderer"].(map[string]any); ok {
			if card, ok := parseTwoRowItem(tr); ok && card.Kind == "playlist" {
				if _, exists := seen[card.ID]; !exists {
					seen[card.ID] = struct{}{}
					*out = append(*out, card)
				}
			}
		}
		for _, child := range v {
			extractPlaylistSearchItems(child, out, seen)
		}
	case []any:
		for _, child := range v {
			extractPlaylistSearchItems(child, out, seen)
		}
	}
}

func parseResponsivePlaylistItem(r map[string]any) (YTMusicCard, bool) {
	browseID := ""
	pageType := ""
	playlistID := ""

	if nav, ok := r["navigationEndpoint"].(map[string]any); ok {
		if be, ok := nav["browseEndpoint"].(map[string]any); ok {
			browseID, _ = be["browseId"].(string)
			if cfg, ok := be["browseEndpointContextSupportedConfigs"].(map[string]any); ok {
				if mc, ok := cfg["browseEndpointContextMusicConfig"].(map[string]any); ok {
					pageType, _ = mc["pageType"].(string)
				}
			}
		}
	}

	if overlay, ok := r["overlay"].(map[string]any); ok {
		if to, ok := overlay["musicItemThumbnailOverlayRenderer"].(map[string]any); ok {
			if content, ok := to["content"].(map[string]any); ok {
				if btn, ok := content["musicPlayButtonRenderer"].(map[string]any); ok {
					if pnav, ok := btn["playNavigationEndpoint"].(map[string]any); ok {
						if wpe, ok := pnav["watchPlaylistEndpoint"].(map[string]any); ok {
							if pid, ok := wpe["playlistId"].(string); ok && pid != "" {
								playlistID = pid
							}
						}
					}
				}
			}
		}
	}

	if browseID == "" && playlistID == "" {
		return YTMusicCard{}, false
	}
	if pageType != "" && pageType != "MUSIC_PAGE_TYPE_PLAYLIST" && !strings.HasPrefix(browseID, "VL") && playlistID == "" {
		return YTMusicCard{}, false
	}

	title := ""
	subtitle := ""
	if cols, ok := r["flexColumns"].([]any); ok {
		if len(cols) > 0 {
			title = extractColumnText(cols[0])
		}
		if len(cols) > 1 {
			subtitle = extractColumnText(cols[1])
		}
	}
	if title == "" {
		return YTMusicCard{}, false
	}

	cover := findLastThumbnailURL(r)
	idVal := browseID
	if idVal == "" {
		idVal = playlistID
	}

	artist := cleanCollectionSubtitle(subtitle)
	if artist == "" {
		artist = "YouTube Music"
	}

	return YTMusicCard{
		ID:         "playlist:" + idVal,
		Title:      title,
		Subtitle:   subtitle,
		CoverURL:   cover,
		Kind:       "playlist",
		BrowseID:   browseID,
		PlaylistID: playlistID,
		Artist:     artist,
		Album:      title,
	}, true
}

func (c *YTMusicClient) GetHomeFeed(ctx context.Context) (YTMusicHomeFeed, error) {
	feed := YTMusicHomeFeed{
		Shelves: []YTMusicShelf{},
		Tracks:  []YTMusicTrack{},
	}

	browseIDs := []string{"FEmusic_home", "FEmusic_explore"}
	results := make([][]YTMusicShelf, len(browseIDs))
	var wg sync.WaitGroup
	for i, bid := range browseIDs {
		wg.Add(1)
		go func(idx int, browseID string) {
			defer wg.Done()
			tree, err := c.postInnerTube(ctx, "browse", map[string]any{
				"context":  ytWebRemixContext(),
				"browseId": browseID,
			})
			if err != nil {
				return
			}
			var shelves []YTMusicShelf
			extractCarouselShelves(tree, &shelves)
			results[idx] = shelves
		}(i, bid)
	}
	wg.Wait()

	seenShelf := map[string]struct{}{}
	seenTrack := map[string]struct{}{}
	for _, list := range results {
		for _, sh := range list {
			key := strings.ToLower(strings.TrimSpace(sh.Title))
			if key == "" || len(sh.Items) == 0 {
				continue
			}
			if _, dup := seenShelf[key]; dup {
				continue
			}
			seenShelf[key] = struct{}{}
			feed.Shelves = append(feed.Shelves, sh)

			for _, item := range sh.Items {
				if item.VideoID != "" {
					if _, dup := seenTrack[item.VideoID]; !dup {
						seenTrack[item.VideoID] = struct{}{}
						dur := item.DurationSec
						if dur <= 0 {
							dur = 200
						}
						artist := item.Artist
						if artist == "" {
							artist = item.Subtitle
						}
						album := item.Album
						if album == "" {
							album = "YouTube Music"
						}
						feed.Tracks = append(feed.Tracks, YTMusicTrack{
							VideoID:     item.VideoID,
							Title:       item.Title,
							Artist:      artist,
							Album:       album,
							DurationSec: dur,
							CoverURL:    item.CoverURL,
							Format:      "OPUS",
						})
					}
				}
			}
		}
	}

	return feed, nil
}

// GetTop100Tracks fetches up to 100 top songs from YouTube Music (cached for 30 minutes).
func (c *YTMusicClient) GetTop100Tracks(ctx context.Context) ([]YTMusicTrack, error) {
	c.mu.RLock()
	if len(c.top100Cache) > 0 && time.Now().Before(c.top100Expires) {
		out := make([]YTMusicTrack, len(c.top100Cache))
		copy(out, c.top100Cache)
		c.mu.RUnlock()
		return out, nil
	}
	c.mu.RUnlock()

	results, err := c.GetCollectionTracks(
		ctx,
		"VLPL4fGSI1pDJn6puJdseH2Rt9sMvt9E2M4i",
		"PL4fGSI1pDJn6puJdseH2Rt9sMvt9E2M4i",
		"",
		"",
		"",
	)
	if err != nil || len(results) < 10 {
		if tree, chartErr := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  ytWebRemixContext(),
			"browseId": "FEmusic_charts",
		}); chartErr == nil {
			seen := map[string]struct{}{}
			for _, t := range results {
				seen[t.VideoID] = struct{}{}
			}
			extractResponsiveItems(tree, &results, seen)
		}
	}
	if len(results) < 10 {
		if feed, feedErr := c.GetHomeFeed(ctx); feedErr == nil && len(feed.Tracks) > 0 {
			seen := map[string]struct{}{}
			for _, t := range results {
				seen[t.VideoID] = struct{}{}
			}
			for _, t := range feed.Tracks {
				if _, ok := seen[t.VideoID]; !ok {
					seen[t.VideoID] = struct{}{}
					results = append(results, t)
				}
			}
		}
	}
	if len(results) > 100 {
		results = results[:100]
	}
	if len(results) > 0 {
		c.mu.Lock()
		c.top100Cache = make([]YTMusicTrack, len(results))
		copy(c.top100Cache, results)
		c.top100Expires = time.Now().Add(30 * time.Minute)
		c.mu.Unlock()
	}
	return results, nil
}

// GetTop100Recommended picks 10 random tracks from Top 100 YouTube Music and enriches their album metadata.
func (c *YTMusicClient) GetTop100Recommended(ctx context.Context, excludeVideoIDs []string) ([]YTMusicTrack, error) {
	all, err := c.GetTop100Tracks(ctx)
	if err != nil {
		return nil, err
	}
	excluded := make(map[string]struct{}, len(excludeVideoIDs))
	for _, id := range excludeVideoIDs {
		if id != "" {
			excluded[id] = struct{}{}
		}
	}
	pool := make([]YTMusicTrack, 0, len(all))
	for _, t := range all {
		if _, skip := excluded[t.VideoID]; !skip {
			pool = append(pool, t)
		}
	}
	if len(pool) == 0 {
		return []YTMusicTrack{}, nil
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})
	if len(pool) > 10 {
		pool = pool[:10]
	}

	enriched := c.enrichTracksAlbum(ctx, pool)
	if len(enriched) > 0 {
		c.mu.Lock()
		for i := range c.top100Cache {
			if upd, ok := enriched[c.top100Cache[i].VideoID]; ok {
				c.top100Cache[i].Album = upd.Album
				c.top100Cache[i].CoverURL = upd.CoverURL
			}
		}
		c.mu.Unlock()
	}

	return pool, nil
}

type HistorySeed struct {
	VideoID string
	Title   string
	Artist  string
}

// GetHistoryBasedRecommended finds 10 genre-similar tracks using YouTube Music Radio ("next" RDAMVM) seeded from play history.
func (c *YTMusicClient) GetHistoryBasedRecommended(ctx context.Context, seeds []HistorySeed, excludeVideoIDs []string) ([]YTMusicTrack, error) {
	if len(seeds) == 0 {
		return c.GetTop100Recommended(ctx, excludeVideoIDs)
	}

	excluded := make(map[string]struct{}, len(excludeVideoIDs)+len(seeds))
	for _, id := range excludeVideoIDs {
		if id != "" {
			excluded[id] = struct{}{}
		}
	}
	for _, s := range seeds {
		if s.VideoID != "" {
			excluded[s.VideoID] = struct{}{}
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffledSeeds := make([]HistorySeed, len(seeds))
	copy(shuffledSeeds, seeds)
	r.Shuffle(len(shuffledSeeds), func(i, j int) {
		shuffledSeeds[i], shuffledSeeds[j] = shuffledSeeds[j], shuffledSeeds[i]
	})
	if len(shuffledSeeds) > 4 {
		shuffledSeeds = shuffledSeeds[:4]
	}

	var wg sync.WaitGroup
	var poolMu sync.Mutex
	seenPool := map[string]struct{}{}
	var pool []YTMusicTrack

	for _, seed := range shuffledSeeds {
		wg.Add(1)
		go func(s HistorySeed) {
			defer wg.Done()
			subCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
			defer cancel()

			vid := strings.TrimSpace(s.VideoID)
			if vid == "" {
				q := strings.TrimSpace(s.Title + " " + s.Artist)
				if q != "" {
					if hits, err := c.Search(subCtx, q); err == nil && len(hits) > 0 {
						vid = hits[0].VideoID
					}
				}
			}

			var related []YTMusicTrack
			relSeen := map[string]struct{}{}
			if vid != "" {
				if tree, err := c.postInnerTube(subCtx, "next", map[string]any{
					"context":     ytWebRemixContext(),
					"videoId":     vid,
					"playlistId":  "RDAMVM" + vid,
					"isAudioOnly": true,
				}); err == nil {
					extractPlaylistPanelItems(tree, &related, relSeen)
				}
			}
			if len(related) < 4 && strings.TrimSpace(s.Artist) != "" {
				if hits, err := c.Search(subCtx, s.Artist); err == nil {
					for _, h := range hits {
						if _, dup := relSeen[h.VideoID]; !dup {
							relSeen[h.VideoID] = struct{}{}
							related = append(related, h)
						}
					}
				}
			}

			poolMu.Lock()
			for _, tr := range related {
				if tr.VideoID == "" || tr.VideoID == vid {
					continue
				}
				if _, skip := excluded[tr.VideoID]; skip {
					continue
				}
				if _, dup := seenPool[tr.VideoID]; dup {
					continue
				}
				seenPool[tr.VideoID] = struct{}{}
				pool = append(pool, tr)
			}
			poolMu.Unlock()
		}(seed)
	}
	wg.Wait()

	if len(pool) == 0 {
		return c.GetTop100Recommended(ctx, excludeVideoIDs)
	}

	r.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})
	if len(pool) > 10 {
		pool = pool[:10]
	}

	c.enrichTracksAlbum(ctx, pool)
	return pool, nil
}

func (c *YTMusicClient) enrichTracksAlbum(ctx context.Context, pool []YTMusicTrack) map[string]YTMusicTrack {
	var wg sync.WaitGroup
	enriched := make(map[string]YTMusicTrack)
	var enrichMu sync.Mutex

	for i := range pool {
		if pool[i].Album != "" && pool[i].Album != "YouTube Music" {
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tr := pool[idx]
			subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			q := strings.TrimSpace(tr.Title + " " + tr.Artist)
			if hits, sErr := c.Search(subCtx, q); sErr == nil {
				for _, h := range hits {
					if h.Album != "" && h.Album != "YouTube Music" {
						tr.Album = h.Album
						if h.CoverURL != "" {
							tr.CoverURL = h.CoverURL
						}
						if tr.PlayCountText == "" && h.PlayCountText != "" {
							tr.PlayCountText = h.PlayCountText
						}
						break
					}
				}
			}
			if tr.Album == "" || tr.Album == "YouTube Music" {
				if tree, nErr := c.postInnerTube(subCtx, "next", map[string]any{
					"context": ytWebRemixContext(),
					"videoId": tr.VideoID,
				}); nErr == nil {
					_, albTitle, _, _, _ := extractAlbumArtistFromNext(tree)
					if albTitle != "" && albTitle != "YouTube Music" {
						tr.Album = albTitle
					}
				}
			}
			if tr.Album == "" || tr.Album == "YouTube Music" || tr.Album == "Single" {
				tr.Album = tr.Title
			}
			pool[idx] = tr
			enrichMu.Lock()
			enriched[tr.VideoID] = tr
			enrichMu.Unlock()
		}(i)
	}
	wg.Wait()
	return enriched
}

func (c *YTMusicClient) GetCollectionTracks(ctx context.Context, browseID, playlistID, fallbackArtist, fallbackAlbum, fallbackCover string) ([]YTMusicTrack, error) {
	targetID := strings.TrimSpace(browseID)
	pid := strings.TrimSpace(playlistID)
	pidClean := strings.TrimPrefix(pid, "VL")
	if targetID == "" && pid != "" {
		if strings.HasPrefix(pid, "VL") {
			targetID = pid
		} else {
			targetID = "VL" + pid
		}
	}

	var results []YTMusicTrack
	seen := map[string]struct{}{}

	if targetID != "" {
		if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  ytWebRemixContext(),
			"browseId": targetID,
		}); err == nil {
			extractResponsiveItems(tree, &results, seen)
		}
	}

	if len(results) == 0 && pidClean != "" {
		vlID := "VL" + pidClean
		if vlID != targetID {
			if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
				"context":  ytWebRemixContext(),
				"browseId": vlID,
			}); err == nil {
				extractResponsiveItems(tree, &results, seen)
			}
		}
	}

	if len(results) == 0 && pidClean != "" {
		if tree, err := c.postInnerTube(ctx, "next", map[string]any{
			"context":    ytWebRemixContext(),
			"playlistId": pidClean,
		}); err == nil {
			extractPlaylistPanelItems(tree, &results, seen)
		}
	}

	cleanArtist := cleanCollectionSubtitle(fallbackArtist)
	if len(results) == 0 {
		q := strings.TrimSpace(fallbackAlbum + " " + cleanArtist)
		if q != "" {
			return c.Search(ctx, q)
		}
		return []YTMusicTrack{}, nil
	}

	for i := range results {
		if fallbackCover != "" && (results[i].CoverURL == "" || strings.Contains(results[i].CoverURL, "i.ytimg.com/vi/")) {
			results[i].CoverURL = fallbackCover
		}
		if (results[i].Artist == "" || results[i].Artist == "YouTube Music Artist") && cleanArtist != "" {
			results[i].Artist = cleanArtist
		}
		if (results[i].Album == "" || results[i].Album == "YouTube Music") && fallbackAlbum != "" {
			results[i].Album = fallbackAlbum
		}
	}
	if len(results) > 40 {
		results = results[:40]
	}
	return results, nil
}

var yearRegex = regexp.MustCompile(`\b(19|20)\d{2}\b`)

// GetTrackAlbumDetail resolves the official YouTube Music album (MPREb_...) for a track
// or browseId, returning its full tracklist and other albums by the same artist.
func (c *YTMusicClient) GetTrackAlbumDetail(ctx context.Context, videoID, browseID, fallbackTitle, fallbackArtist, fallbackAlbum, fallbackCover string) (YTMusicAlbumDetail, error) {
	albumBrowseID := strings.TrimSpace(browseID)
	artistBrowseID := ""
	artistName := cleanCollectionSubtitle(fallbackArtist)
	albumTitle := strings.TrimSpace(fallbackAlbum)
	cover := strings.TrimSpace(fallbackCover)

	if !strings.HasPrefix(albumBrowseID, "MPREb_") && strings.TrimSpace(videoID) != "" {
		if tree, err := c.postInnerTube(ctx, "next", map[string]any{
			"context": ytWebRemixContext(),
			"videoId": strings.TrimSpace(videoID),
		}); err == nil {
			abid, atitle, arid, aname, acover := extractAlbumArtistFromNext(tree)
			if abid != "" {
				albumBrowseID = abid
			}
			if atitle != "" {
				albumTitle = atitle
			}
			if arid != "" {
				artistBrowseID = arid
			}
			if aname != "" {
				artistName = aname
			}
			if cover == "" && acover != "" {
				cover = acover
			}
		}
	}

	if !strings.HasPrefix(albumBrowseID, "MPREb_") {
		q := strings.TrimSpace(fallbackTitle + " " + artistName)
		if q == "" {
			q = strings.TrimSpace(albumTitle + " " + artistName)
		}
		if q != "" {
			if cards, err := c.SearchPlaylists(ctx, q); err == nil {
				for _, card := range cards {
					if card.Kind == "album" && strings.HasPrefix(card.BrowseID, "MPREb_") {
						albumBrowseID = card.BrowseID
						if albumTitle == "" {
							albumTitle = card.Title
						}
						if cover == "" {
							cover = card.CoverURL
						}
						break
					}
				}
			}
			if !strings.HasPrefix(albumBrowseID, "MPREb_") {
				if tracks, err := c.Search(ctx, q); err == nil && len(tracks) > 0 && tracks[0].VideoID != "" && tracks[0].VideoID != videoID {
					if tree, err := c.postInnerTube(ctx, "next", map[string]any{
						"context": ytWebRemixContext(),
						"videoId": tracks[0].VideoID,
					}); err == nil {
						abid, atitle, arid, aname, acover := extractAlbumArtistFromNext(tree)
						if abid != "" {
							albumBrowseID = abid
						}
						if atitle != "" {
							albumTitle = atitle
						}
						if arid != "" && artistBrowseID == "" {
							artistBrowseID = arid
						}
						if aname != "" && artistName == "" {
							artistName = aname
						}
						if cover == "" && acover != "" {
							cover = acover
						}
					}
				}
			}
		}
	}

	detail := YTMusicAlbumDetail{
		BrowseID:       albumBrowseID,
		Title:          albumTitle,
		Artist:         artistName,
		ArtistBrowseID: artistBrowseID,
		CoverURL:       cover,
	}

	if strings.HasPrefix(albumBrowseID, "MPREb_") {
		if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  ytWebRemixContext(),
			"browseId": albumBrowseID,
		}); err == nil {
			extractAlbumHeaderMeta(tree, &detail)
			seen := map[string]struct{}{}
			extractResponsiveItems(tree, &detail.Tracks, seen)
		}
	}

	if detail.Title == "" {
		detail.Title = fallbackTitle
	}
	if detail.Artist == "" {
		detail.Artist = "YouTube Music"
	}

	for i := range detail.Tracks {
		if detail.CoverURL != "" && (detail.Tracks[i].CoverURL == "" || strings.Contains(detail.Tracks[i].CoverURL, "i.ytimg.com/vi/")) {
			detail.Tracks[i].CoverURL = detail.CoverURL
		}
		if detail.Tracks[i].Artist == "" || detail.Tracks[i].Artist == "YouTube Music Artist" {
			detail.Tracks[i].Artist = detail.Artist
		}
		if detail.Tracks[i].Album == "" || detail.Tracks[i].Album == "YouTube Music" {
			detail.Tracks[i].Album = detail.Title
		}
	}

	// Fetch other albums from the same artist
	var otherAlbums []YTMusicCard
	seenAlbums := map[string]struct{}{}
	if detail.BrowseID != "" {
		seenAlbums[detail.BrowseID] = struct{}{}
	}
	if detail.ArtistBrowseID != "" {
		if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  ytWebRemixContext(),
			"browseId": detail.ArtistBrowseID,
		}); err == nil {
			extractArtistAlbumCards(tree, &otherAlbums, seenAlbums)
		}
	}
	if len(otherAlbums) == 0 && detail.Artist != "" && detail.Artist != "YouTube Music" {
		if cards, err := c.SearchPlaylists(ctx, detail.Artist); err == nil {
			for _, card := range cards {
				if card.Kind != "album" || card.BrowseID == "" {
					continue
				}
				if _, dup := seenAlbums[card.BrowseID]; dup {
					continue
				}
				seenAlbums[card.BrowseID] = struct{}{}
				otherAlbums = append(otherAlbums, card)
				if len(otherAlbums) >= 12 {
					break
				}
			}
		}
	}
	detail.ArtistAlbums = otherAlbums
	return detail, nil
}

func extractAlbumArtistFromNext(node any) (albumBrowseID, albumTitle, artistBrowseID, artistName, coverURL string) {
	var walk func(n any) bool
	walk = func(n any) bool {
		switch v := n.(type) {
		case map[string]any:
			if pvr, ok := v["playlistPanelVideoRenderer"].(map[string]any); ok {
				if byline, ok := pvr["longBylineText"].(map[string]any); ok {
					if runs, ok := byline["runs"].([]any); ok {
						for _, item := range runs {
							rm, ok := item.(map[string]any)
							if !ok {
								continue
							}
							txt, _ := rm["text"].(string)
							nav, _ := rm["navigationEndpoint"].(map[string]any)
							be, _ := nav["browseEndpoint"].(map[string]any)
							bid, _ := be["browseId"].(string)
							if strings.HasPrefix(bid, "MPREb_") && albumBrowseID == "" {
								albumBrowseID = bid
								albumTitle = strings.TrimSpace(txt)
							} else if strings.HasPrefix(bid, "UC") && artistBrowseID == "" {
								artistBrowseID = bid
								artistName = strings.TrimSpace(txt)
							}
						}
					}
				}
				if t, ok := pvr["thumbnail"].(map[string]any); ok {
					if list, ok := t["thumbnails"].([]any); ok && len(list) > 0 {
						if last, ok := list[len(list)-1].(map[string]any); ok {
							if u, ok := last["url"].(string); ok && u != "" {
								coverURL = upgradeThumbURL(u)
							}
						}
					}
				}
				if menu, ok := pvr["menu"].(map[string]any); ok {
					if mr, ok := menu["menuRenderer"].(map[string]any); ok {
						if items, ok := mr["items"].([]any); ok {
							for _, item := range items {
								im, ok := item.(map[string]any)
								if !ok {
									continue
								}
								mni, ok := im["menuNavigationItemRenderer"].(map[string]any)
								if !ok {
									continue
								}
								nav, _ := mni["navigationEndpoint"].(map[string]any)
								be, _ := nav["browseEndpoint"].(map[string]any)
								bid, _ := be["browseId"].(string)
								if strings.HasPrefix(bid, "MPREb_") && albumBrowseID == "" {
									albumBrowseID = bid
								} else if strings.HasPrefix(bid, "UC") && artistBrowseID == "" {
									artistBrowseID = bid
								}
							}
						}
					}
				}
				if albumBrowseID != "" || artistBrowseID != "" {
					return true
				}
			}
			for _, child := range v {
				if walk(child) {
					return true
				}
			}
		case []any:
			for _, child := range v {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	walk(node)
	return
}

func extractAlbumHeaderMeta(node any, detail *YTMusicAlbumDetail) {
	var walk func(n any) bool
	walk = func(n any) bool {
		switch v := n.(type) {
		case map[string]any:
			for _, key := range []string{"musicResponsiveHeaderRenderer", "musicDetailHeaderRenderer"} {
				if hdr, ok := v[key].(map[string]any); ok {
					if t := strings.TrimSpace(extractRunsText(hdr["title"])); t != "" {
						detail.Title = t
					}
					if sub := strings.TrimSpace(extractRunsText(hdr["subtitle"])); sub != "" {
						detail.Subtitle = sub
						if m := yearRegex.FindString(sub); m != "" {
							detail.Year = m
						}
					}
					for _, field := range []string{"straplineTextOne", "subtitle"} {
						if nodeMap, ok := hdr[field].(map[string]any); ok {
							if runs, ok := nodeMap["runs"].([]any); ok {
								for _, item := range runs {
									rm, ok := item.(map[string]any)
									if !ok {
										continue
									}
									txt, _ := rm["text"].(string)
									nav, _ := rm["navigationEndpoint"].(map[string]any)
									be, _ := nav["browseEndpoint"].(map[string]any)
									bid, _ := be["browseId"].(string)
									if strings.HasPrefix(bid, "UC") && detail.ArtistBrowseID == "" {
										detail.ArtistBrowseID = bid
										if strings.TrimSpace(txt) != "" {
											detail.Artist = strings.TrimSpace(txt)
										}
									}
								}
							}
						}
					}
					if thumb, ok := hdr["thumbnail"].(map[string]any); ok {
						if u := extractThumbRendererURL(thumb); u != "" {
							detail.CoverURL = u
						}
					}
					return true
				}
			}
			for _, child := range v {
				if walk(child) {
					return true
				}
			}
		case []any:
			for _, child := range v {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	walk(node)
}

func extractArtistAlbumCards(node any, out *[]YTMusicCard, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		if r, ok := v["musicTwoRowItemRenderer"].(map[string]any); ok {
			if card, ok := parseTwoRowItem(r); ok && card.Kind == "album" && strings.HasPrefix(card.BrowseID, "MPREb_") {
				if _, dup := seen[card.BrowseID]; !dup {
					seen[card.BrowseID] = struct{}{}
					*out = append(*out, card)
					if len(*out) >= 12 {
						return
					}
				}
			}
		}
		for _, child := range v {
			extractArtistAlbumCards(child, out, seen)
			if len(*out) >= 12 {
				return
			}
		}
	case []any:
		for _, child := range v {
			extractArtistAlbumCards(child, out, seen)
			if len(*out) >= 12 {
				return
			}
		}
	}
}

func extractPlaylistPanelItems(node any, out *[]YTMusicTrack, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		if pvr, ok := v["playlistPanelVideoRenderer"].(map[string]any); ok {
			if tr, ok := parsePlaylistPanelItem(pvr); ok {
				if _, dup := seen[tr.VideoID]; !dup {
					seen[tr.VideoID] = struct{}{}
					*out = append(*out, tr)
				}
			}
		}
		for _, child := range v {
			extractPlaylistPanelItems(child, out, seen)
		}
	case []any:
		for _, child := range v {
			extractPlaylistPanelItems(child, out, seen)
		}
	}
}

func parsePlaylistPanelItem(r map[string]any) (YTMusicTrack, bool) {
	videoID, _ := r["videoId"].(string)
	title := extractRunsText(r["title"])
	if videoID == "" || title == "" {
		return YTMusicTrack{}, false
	}
	artist := "YouTube Music Artist"
	album := "YouTube Music"
	if byline, ok := r["longBylineText"].(map[string]any); ok {
		if runs, ok := byline["runs"].([]any); ok {
			var artistParts []string
			var fallbackParts []string
			for _, item := range runs {
				rm, ok := item.(map[string]any)
				if !ok {
					continue
				}
				txt, _ := rm["text"].(string)
				txt = strings.TrimSpace(txt)
				if txt == "" || txt == "\u2022" || txt == "\u00b7" || txt == "," || txt == "&" {
					continue
				}
				bid := ""
				pageType := ""
				if nav, ok := rm["navigationEndpoint"].(map[string]any); ok {
					if be, ok := nav["browseEndpoint"].(map[string]any); ok {
						bid, _ = be["browseId"].(string)
						if cfg, ok := be["browseEndpointContextSupportedConfigs"].(map[string]any); ok {
							if mc, ok := cfg["browseEndpointContextMusicConfig"].(map[string]any); ok {
								pageType, _ = mc["pageType"].(string)
							}
						}
					}
				}
				if strings.HasPrefix(bid, "MPREb_") || pageType == "MUSIC_PAGE_TYPE_ALBUM" {
					album = txt
				} else if strings.HasPrefix(bid, "UC") || pageType == "MUSIC_PAGE_TYPE_ARTIST" {
					artistParts = append(artistParts, txt)
				} else {
					fallbackParts = append(fallbackParts, txt)
				}
			}
			if len(artistParts) > 0 {
				artist = strings.Join(artistParts, ", ")
			} else if len(fallbackParts) > 0 {
				artist = fallbackParts[0]
			}
		}
	}
	durationSec := 200
	if d := parseDurationString(extractRunsText(r["lengthText"])); d > 0 {
		durationSec = d
	}
	cover := fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", videoID)
	if t, ok := r["thumbnail"].(map[string]any); ok {
		if list, ok := t["thumbnails"].([]any); ok && len(list) > 0 {
			if last, ok := list[len(list)-1].(map[string]any); ok {
				if u, ok := last["url"].(string); ok && u != "" {
					cover = upgradeThumbURL(u)
				}
			}
		}
	}
	return YTMusicTrack{
		VideoID:     videoID,
		Title:       title,
		Artist:      artist,
		Album:       album,
		DurationSec: durationSec,
		CoverURL:    cover,
		Format:      "OPUS",
	}, true
}

func cleanCollectionSubtitle(sub string) string {
	s := strings.TrimSpace(sub)
	for _, prefix := range []string{"Album â€¢ ", "Single â€¢ ", "EP â€¢ ", "Playlist â€¢ "} {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
	}
	return s
}

func extractCarouselShelves(node any, out *[]YTMusicShelf) {
	switch v := node.(type) {
	case map[string]any:
		if shelf, ok := v["musicCarouselShelfRenderer"].(map[string]any); ok {
			title := extractShelfTitle(shelf)
			var cards []YTMusicCard
			seen := map[string]struct{}{}
			if contents, ok := shelf["contents"].([]any); ok {
				for _, entry := range contents {
					m, ok := entry.(map[string]any)
					if !ok {
						continue
					}
					if twoRow, ok := m["musicTwoRowItemRenderer"].(map[string]any); ok {
						if card, ok := parseTwoRowItem(twoRow); ok {
							if _, dup := seen[card.ID]; !dup {
								seen[card.ID] = struct{}{}
								cards = append(cards, card)
							}
						}
					} else if respItem, ok := m["musicResponsiveListItemRenderer"].(map[string]any); ok {
						if tr, ok := parseResponsiveListItem(respItem); ok {
							cardID := "song:" + tr.VideoID
							if _, dup := seen[cardID]; !dup {
								seen[cardID] = struct{}{}
								cards = append(cards, YTMusicCard{
									ID:          cardID,
									Title:       tr.Title,
									Subtitle:    tr.Artist,
									CoverURL:    tr.CoverURL,
									Kind:        "song",
									VideoID:     tr.VideoID,
									Artist:      tr.Artist,
									Album:       tr.Album,
									DurationSec: tr.DurationSec,
								})
							}
						}
					}
				}
			}
			if title != "" && len(cards) > 0 {
				*out = append(*out, YTMusicShelf{
					Title: title,
					Items: cards,
				})
			}
			return
		}
		for _, child := range v {
			extractCarouselShelves(child, out)
		}
	case []any:
		for _, child := range v {
			extractCarouselShelves(child, out)
		}
	}
}

func extractShelfTitle(shelf map[string]any) string {
	hdr, ok := shelf["header"].(map[string]any)
	if !ok {
		return ""
	}
	basic, ok := hdr["musicCarouselShelfBasicHeaderRenderer"].(map[string]any)
	if !ok {
		return ""
	}
	title := extractRunsText(basic["title"])
	return strings.TrimSuffix(strings.TrimSpace(title), ".")
}

func extractRunsText(node any) string {
	m, ok := node.(map[string]any)
	if !ok {
		return ""
	}
	runs, ok := m["runs"].([]any)
	if !ok {
		return ""
	}
	var sb strings.Builder
	for _, r := range runs {
		if rm, ok := r.(map[string]any); ok {
			if txt, ok := rm["text"].(string); ok {
				sb.WriteString(txt)
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

func parseTwoRowItem(r map[string]any) (YTMusicCard, bool) {
	title := extractRunsText(r["title"])
	if title == "" {
		return YTMusicCard{}, false
	}
	subtitle := extractRunsText(r["subtitle"])

	cover := ""
	if thumbWrap, ok := r["thumbnailRenderer"].(map[string]any); ok {
		cover = extractThumbRendererURL(thumbWrap)
	}

	browseID := ""
	pageType := ""
	videoID := ""
	playlistID := ""

	if nav, ok := r["navigationEndpoint"].(map[string]any); ok {
		if be, ok := nav["browseEndpoint"].(map[string]any); ok {
			browseID, _ = be["browseId"].(string)
			if cfg, ok := be["browseEndpointContextSupportedConfigs"].(map[string]any); ok {
				if mc, ok := cfg["browseEndpointContextMusicConfig"].(map[string]any); ok {
					pageType, _ = mc["pageType"].(string)
				}
			}
		}
		if we, ok := nav["watchEndpoint"].(map[string]any); ok {
			videoID, _ = we["videoId"].(string)
			playlistID, _ = we["playlistId"].(string)
		}
	}

	if overlay, ok := r["thumbnailOverlay"].(map[string]any); ok {
		if to, ok := overlay["musicItemThumbnailOverlayRenderer"].(map[string]any); ok {
			if content, ok := to["content"].(map[string]any); ok {
				if btn, ok := content["musicPlayButtonRenderer"].(map[string]any); ok {
					if pnav, ok := btn["playNavigationEndpoint"].(map[string]any); ok {
						if wpe, ok := pnav["watchPlaylistEndpoint"].(map[string]any); ok {
							if pid, ok := wpe["playlistId"].(string); ok && pid != "" {
								playlistID = pid
							}
						}
						if we, ok := pnav["watchEndpoint"].(map[string]any); ok {
							if vid, ok := we["videoId"].(string); ok && vid != "" && videoID == "" {
								videoID = vid
							}
							if pid, ok := we["playlistId"].(string); ok && pid != "" && playlistID == "" {
								playlistID = pid
							}
						}
					}
				}
			}
		}
	}

	kind := "playlist"
	if videoID != "" && browseID == "" {
		kind = "song"
	} else if strings.HasPrefix(browseID, "MPREb_") || pageType == "MUSIC_PAGE_TYPE_ALBUM" {
		kind = "album"
	}

	if videoID == "" && browseID == "" && playlistID == "" {
		return YTMusicCard{}, false
	}

	if cover == "" && videoID != "" {
		cover = fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", videoID)
	}

	idVal := browseID
	if kind == "song" {
		idVal = videoID
	} else if idVal == "" {
		idVal = playlistID
	}

	artist := cleanCollectionSubtitle(subtitle)
	if artist == "" {
		artist = "YouTube Music"
	}

	return YTMusicCard{
		ID:         kind + ":" + idVal,
		Title:      title,
		Subtitle:   subtitle,
		CoverURL:   cover,
		Kind:       kind,
		VideoID:    videoID,
		BrowseID:   browseID,
		PlaylistID: playlistID,
		Artist:     artist,
		Album:      title,
	}, true
}

func extractResponsiveItems(node any, out *[]YTMusicTrack, seen map[string]struct{}) {
	switch v := node.(type) {
	case map[string]any:
		if renderer, ok := v["musicResponsiveListItemRenderer"].(map[string]any); ok {
			if tr, ok := parseResponsiveListItem(renderer); ok {
				if _, dup := seen[tr.VideoID]; !dup {
					seen[tr.VideoID] = struct{}{}
					*out = append(*out, tr)
				}
			}
		}
		for _, child := range v {
			extractResponsiveItems(child, out, seen)
		}
	case []any:
		for _, child := range v {
			extractResponsiveItems(child, out, seen)
		}
	}
}

func parseResponsiveListItem(r map[string]any) (YTMusicTrack, bool) {
	videoID := ""
	if pid, ok := r["playlistItemData"].(map[string]any); ok {
		if id, ok := pid["videoId"].(string); ok {
			videoID = id
		}
	}
	if videoID == "" {
		if overlay, ok := r["overlay"].(map[string]any); ok {
			videoID = findVideoIDDeep(overlay)
		}
	}
	if videoID == "" {
		return YTMusicTrack{}, false
	}

	title := ""
	artist := "YouTube Music Artist"
	album := "YouTube Music"
	playCountText := ""
	durationSec := 200

	isPlayCount := func(lower string) bool {
		return strings.Contains(lower, "pemutaran") ||
			strings.Contains(lower, "ditonton") ||
			strings.HasSuffix(lower, "plays") ||
			strings.HasSuffix(lower, "views")
	}

	if cols, ok := r["flexColumns"].([]any); ok {
		if len(cols) > 0 {
			title = extractColumnText(cols[0])
		}
		if len(cols) > 2 {
			// Playlist / chart table layout: col 1 = artist(s), col 2 = album or play count
			col1Text := strings.TrimSpace(extractColumnText(cols[1]))
			if col1Text != "" {
				artist = col1Text
			}
			col2Text := strings.TrimSpace(extractColumnText(cols[2]))
			col2Lower := strings.ToLower(col2Text)
			if col2Text != "" && parseDurationString(col2Text) == 0 {
				if isPlayCount(col2Lower) {
					playCountText = col2Text
				} else {
					album = col2Text
				}
			}
		} else if len(cols) > 1 {
			var artistParts []string
			var fallbackParts []string
			if colMap, ok := cols[1].(map[string]any); ok {
				if rend, ok := colMap["musicResponsiveListItemFlexColumnRenderer"].(map[string]any); ok {
					if txtObj, ok := rend["text"].(map[string]any); ok {
						if runs, ok := txtObj["runs"].([]any); ok {
							for _, item := range runs {
								rm, ok := item.(map[string]any)
								if !ok {
									continue
								}
								p, _ := rm["text"].(string)
								p = strings.TrimSpace(p)
								pLower := strings.ToLower(p)
								if p == "" || p == "\u2022" || p == "\u00b7" || p == "," || p == "&" || p == "Song" || p == "Video" || p == "Lagu" {
									continue
								}
								if isPlayCount(pLower) {
									if playCountText == "" {
										playCountText = p
									}
									continue
								}
								if d := parseDurationString(p); d > 0 {
									durationSec = d
									continue
								}
								bid := ""
								pageType := ""
								if nav, ok := rm["navigationEndpoint"].(map[string]any); ok {
									if be, ok := nav["browseEndpoint"].(map[string]any); ok {
										bid, _ = be["browseId"].(string)
										if cfg, ok := be["browseEndpointContextSupportedConfigs"].(map[string]any); ok {
											if mc, ok := cfg["browseEndpointContextMusicConfig"].(map[string]any); ok {
												pageType, _ = mc["pageType"].(string)
											}
										}
									}
								}
								if strings.HasPrefix(bid, "MPREb_") || pageType == "MUSIC_PAGE_TYPE_ALBUM" {
									album = p
								} else if strings.HasPrefix(bid, "UC") || pageType == "MUSIC_PAGE_TYPE_ARTIST" {
									artistParts = append(artistParts, p)
								} else if !yearRegex.MatchString(p) {
									fallbackParts = append(fallbackParts, p)
								}
							}
						}
					}
				}
			}
			if len(artistParts) > 0 {
				artist = strings.Join(artistParts, ", ")
			} else if len(fallbackParts) > 0 {
				artist = fallbackParts[0]
			}
			if album == "YouTube Music" && len(fallbackParts) > 1 {
				album = fallbackParts[1]
			}
		}
	}

	if fixedCols, ok := r["fixedColumns"].([]any); ok {
		for _, fc := range fixedCols {
			fm, ok := fc.(map[string]any)
			if !ok {
				continue
			}
			fcr, ok := fm["musicResponsiveListItemFixedColumnRenderer"].(map[string]any)
			if !ok {
				continue
			}
			txt := extractRunsText(fcr["text"])
			if d := parseDurationString(txt); d > 0 {
				durationSec = d
				break
			}
		}
	}

	if title == "" {
		return YTMusicTrack{}, false
	}

	cover := fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", videoID)
	if thumb := findLastThumbnailURL(r); thumb != "" {
		cover = thumb
	}

	return YTMusicTrack{
		VideoID:       videoID,
		Title:         title,
		Artist:        artist,
		Album:         album,
		DurationSec:   durationSec,
		CoverURL:      cover,
		Format:        "OPUS",
		PlayCountText: playCountText,
	}, true
}

func findVideoIDDeep(node any) string {
	switch v := node.(type) {
	case map[string]any:
		if id, ok := v["videoId"].(string); ok && id != "" {
			return id
		}
		for _, child := range v {
			if found := findVideoIDDeep(child); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range v {
			if found := findVideoIDDeep(child); found != "" {
				return found
			}
		}
	}
	return ""
}

func extractColumnText(col any) string {
	runs := extractColumnRuns(col)
	return strings.TrimSpace(strings.Join(runs, ""))
}

func extractColumnRuns(col any) []string {
	m, ok := col.(map[string]any)
	if !ok {
		return nil
	}
	renderer, ok := m["musicResponsiveListItemFlexColumnRenderer"].(map[string]any)
	if !ok {
		return nil
	}
	textObj, ok := renderer["text"].(map[string]any)
	if !ok {
		return nil
	}
	runs, ok := textObj["runs"].([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, r := range runs {
		if rm, ok := r.(map[string]any); ok {
			if txt, ok := rm["text"].(string); ok {
				out = append(out, txt)
			}
		}
	}
	return out
}

var (
	ytThumbSizeRe  = regexp.MustCompile(`=w\d+-h\d+[^&?]*`)
	ytImgVideoRe   = regexp.MustCompile(`https?://i\.ytimg\.com/(?:vi|vi_webp)/([^/?#]+)/[^?#]+(?:\?.*)?$`)
)

func upgradeThumbURL(u string) string {
	if u == "" {
		return u
	}
	if m := ytImgVideoRe.FindStringSubmatch(u); len(m) == 2 {
		return "https://i.ytimg.com/vi/" + m[1] + "/mqdefault.jpg"
	}
	if strings.Contains(u, "googleusercontent.com") || strings.Contains(u, "yt3.ggpht.com") {
		return ytThumbSizeRe.ReplaceAllString(u, "=w544-h544-p-l90-rj")
	}
	return u
}

func extractThumbRendererURL(thumbObj map[string]any) string {
	mtr, ok := thumbObj["musicThumbnailRenderer"].(map[string]any)
	if !ok {
		return ""
	}
	t, ok := mtr["thumbnail"].(map[string]any)
	if !ok {
		return ""
	}
	list, ok := t["thumbnails"].([]any)
	if !ok || len(list) == 0 {
		return ""
	}
	last, ok := list[len(list)-1].(map[string]any)
	if !ok {
		return ""
	}
	u, _ := last["url"].(string)
	return upgradeThumbURL(u)
}

func findLastThumbnailURL(r map[string]any) string {
	thumbObj, ok := r["thumbnail"].(map[string]any)
	if !ok {
		return ""
	}
	return extractThumbRendererURL(thumbObj)
}

var durRe = regexp.MustCompile(`^(\d+:)?\d+:\d{2}$`)

func parseDurationString(s string) int {
	if !durRe.MatchString(s) {
		return 0
	}
	parts := strings.Split(s, ":")
	total := 0
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

func (c *YTMusicClient) InvalidateStreamCache(videoID string) {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return
	}
	prefix := videoID + ":"
	c.mu.Lock()
	for k := range c.cache {
		if strings.HasPrefix(k, prefix) {
			delete(c.cache, k)
		}
	}
	c.mu.Unlock()
}

func (c *YTMusicClient) ResolveStreamURL(ctx context.Context, videoID, quality string) (string, string, error) {
	info, err := c.ResolveStreamWindow(ctx, videoID, quality, 0)
	return info.URL, info.MimeType, err
}

func (c *YTMusicClient) ResolveStreamInfo(ctx context.Context, videoID, quality string) (string, string, string, error) {
	info, err := c.ResolveStreamWindow(ctx, videoID, quality, 0)
	return info.URL, info.MimeType, info.UserAgent, err
}

func (c *YTMusicClient) ResolveStreamWindow(ctx context.Context, videoID, quality string, startTimeSecs int) (StreamInfo, error) {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return StreamInfo{}, fmt.Errorf("empty videoId")
	}

	cacheKey := videoID + ":" + quality
	if startTimeSecs <= 0 {
		c.mu.RLock()
		if hit, ok := c.cache[cacheKey]; ok && time.Now().Before(hit.ExpiresAt) {
			c.mu.RUnlock()
			return StreamInfo{
				URL:           hit.URL,
				MimeType:      hit.MimeType,
				UserAgent:     hit.UserAgent,
				ContentLength: hit.ContentLength,
				DurationSec:   hit.DurationSec,
			}, nil
		}
		c.mu.RUnlock()
	}

	// Try InnerTube ANDROID / IOS / ANDROID_VR player clients first
	if info, err := c.resolveViaInnerTube(ctx, videoID, quality, startTimeSecs); err == nil && info.URL != "" {
		if startTimeSecs <= 0 {
			c.mu.Lock()
			c.cache[cacheKey] = cachedStream{
				URL:           info.URL,
				MimeType:      info.MimeType,
				UserAgent:     info.UserAgent,
				ContentLength: info.ContentLength,
				DurationSec:   info.DurationSec,
				ExpiresAt:     time.Now().Add(4 * time.Hour),
			}
			c.mu.Unlock()
		}
		return info, nil
	}

	// Fallback to yt-dlp if installed on PATH
	if ytdlpPath, err := exec.LookPath("yt-dlp"); err == nil {
		cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(cmdCtx, ytdlpPath, "-g", "-f", "bestaudio[ext=webm]/bestaudio[ext=m4a]/bestaudio", "https://music.youtube.com/watch?v="+videoID).Output()
		if err == nil {
			line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
			if strings.HasPrefix(line, "http") {
				mime := "audio/webm"
				if strings.Contains(line, "mime=audio%2Fmp4") || strings.Contains(line, "mime=audio/mp4") {
					mime = "audio/mp4"
				}
				ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
				info := StreamInfo{
					URL:         line,
					MimeType:    mime,
					UserAgent:   ua,
					DurationSec: 200,
				}
				if startTimeSecs <= 0 {
					c.mu.Lock()
					c.cache[cacheKey] = cachedStream{
						URL:         line,
						MimeType:    mime,
						UserAgent:   ua,
						DurationSec: 200,
						ExpiresAt:   time.Now().Add(4 * time.Hour),
					}
					c.mu.Unlock()
				}
				return info, nil
			}
		}
	}

	return StreamInfo{}, fmt.Errorf("could not resolve stream for %s", videoID)
}

type innerTubePlayerClient struct {
	Payload   map[string]any
	UserAgent string
}

func (c *YTMusicClient) resolveViaInnerTube(ctx context.Context, videoID, quality string, startTimeSecs int) (StreamInfo, error) {
	clients := []innerTubePlayerClient{
		{
			Payload: map[string]any{
				"clientName":        "ANDROID",
				"clientVersion":     "20.10.38",
				"osName":            "Android",
				"osVersion":         "14",
				"androidSdkVersion": 34,
				"hl":                "en",
				"gl":                "US",
			},
			UserAgent: "com.google.android.youtube/20.10.38 (Linux; U; Android 14) gzip",
		},
		{
			Payload: map[string]any{
				"clientName":    "IOS",
				"clientVersion": "20.10.4",
				"deviceMake":    "Apple",
				"deviceModel":   "iPhone16,2",
				"osName":        "iPhone",
				"osVersion":     "18.3.2.22D82",
				"hl":            "en",
				"gl":            "US",
			},
			UserAgent: "com.google.ios.youtube/20.10.4 (iPhone16,2; U; CPU iOS 18_3_2 like Mac OS X;)",
		},
		{
			Payload: map[string]any{
				"clientName":        "ANDROID_VR",
				"clientVersion":     "1.60.19",
				"deviceMake":        "Oculus",
				"deviceModel":       "Quest 3",
				"osName":            "Android",
				"osVersion":         "12L",
				"androidSdkVersion": 32,
				"hl":                "en",
				"gl":                "US",
			},
			UserAgent: "com.google.android.apps.youtube.vr.oculus/1.60.19 (Linux; U; Android 12L; eureka-user Build/SQ3A.220605.009.A1) gzip",
		},
	}

	var lastErr error
	for _, cl := range clients {
		payload := map[string]any{
			"videoId":        videoID,
			"contentCheckOk": true,
			"racyCheckOk":    true,
			"context": map[string]any{
				"client": cl.Payload,
			},
		}
		if startTimeSecs > 0 {
			payload["startTimeSecs"] = startTimeSecs
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.youtube.com/youtubei/v1/player?prettyPrint=false", bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", cl.UserAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("innertube status %d", resp.StatusCode)
			continue
		}

		var data struct {
			PlayabilityStatus struct {
				Status string `json:"status"`
			} `json:"playabilityStatus"`
			VideoDetails struct {
				LengthSeconds string `json:"lengthSeconds"`
			} `json:"videoDetails"`
			StreamingData struct {
				AdaptiveFormats []struct {
					Itag            int    `json:"itag"`
					URL             string `json:"url"`
					MimeType        string `json:"mimeType"`
					Bitrate         int    `json:"bitrate"`
					AudioSampleRate string `json:"audioSampleRate"`
					AudioChannels   int    `json:"audioChannels"`
					ContentLength   string `json:"contentLength"`
				} `json:"adaptiveFormats"`
			} `json:"streamingData"`
		}
		decErr := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&data)
		_ = resp.Body.Close()
		if decErr != nil {
			lastErr = decErr
			continue
		}
		if data.PlayabilityStatus.Status != "OK" {
			lastErr = fmt.Errorf("playability %s", data.PlayabilityStatus.Status)
			continue
		}

		durSec, _ := strconv.Atoi(data.VideoDetails.LengthSeconds)
		if durSec <= 0 {
			durSec = 200
		}

		bestURL := ""
		bestMime := "audio/mp4"
		var bestLen int64
		bestScore := -1

		for _, f := range data.StreamingData.AdaptiveFormats {
			if f.URL == "" || !strings.HasPrefix(f.MimeType, "audio/") {
				continue
			}
			isOpus := strings.Contains(f.MimeType, "opus") || strings.Contains(f.MimeType, "webm")
			isAAC := strings.Contains(f.MimeType, "mp4a.40.2") || strings.Contains(f.MimeType, "audio/mp4")

			score := f.Bitrate
			if quality == "low" && f.Bitrate > 0 {
				score = 1_000_000 - f.Bitrate
			} else if quality == "normal" && f.Bitrate >= 96000 && f.Bitrate <= 160000 {
				score += 500_000
			} else {
				// High quality: prioritize Opus (itag 251 / ~160kbps VBR 48kHz) or high-bitrate AAC (itag 141)
				// Opus at 160kbps VBR provides superior high-frequency clarity & stereo imaging over 128kbps AAC.
				if isOpus {
					score += 120_000
					if f.Itag == 251 {
						score += 40_000
					}
				} else if isAAC {
					score += 60_000
					if f.Itag == 141 {
						score += 150_000
					}
				}
			}
			if f.AudioChannels == 0 || f.AudioChannels >= 2 {
				score += 30_000
			}
			if sr, _ := strconv.Atoi(f.AudioSampleRate); sr >= 48000 {
				score += 20_000
			}
			if score > bestScore {
				bestScore = score
				bestURL = f.URL
				bestMime = strings.Split(f.MimeType, ";")[0]
				bestLen, _ = strconv.ParseInt(f.ContentLength, 10, 64)
			}
		}

		if bestURL != "" {
			return StreamInfo{
				URL:           bestURL,
				MimeType:      bestMime,
				UserAgent:     cl.UserAgent,
				ContentLength: bestLen,
				DurationSec:   durSec,
			}, nil
		}
		lastErr = fmt.Errorf("no direct audio stream found")
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no direct audio stream found")
	}
	return StreamInfo{}, lastErr
}

// GetLyricsByVideoID fetches official YouTube Music lyrics via InnerTube next -> browse (MPLYt_...).
func (c *YTMusicClient) GetLyricsByVideoID(ctx context.Context, videoID string) (string, bool) {
	vid := strings.TrimSpace(videoID)
	if vid == "" {
		return "", false
	}
	nextTree, err := c.postInnerTube(ctx, "next", map[string]any{
		"context": ytWebRemixContext(),
		"videoId": vid,
	})
	if err != nil {
		return "", false
	}

	var lyricsBrowseID string
	var findLyricsID func(n any) bool
	findLyricsID = func(n any) bool {
		switch v := n.(type) {
		case map[string]any:
			if bid, ok := v["browseId"].(string); ok && strings.HasPrefix(bid, "MPLYt_") {
				lyricsBrowseID = bid
				return true
			}
			for _, child := range v {
				if findLyricsID(child) {
					return true
				}
			}
		case []any:
			for _, child := range v {
				if findLyricsID(child) {
					return true
				}
			}
		}
		return false
	}
	findLyricsID(nextTree)
	if lyricsBrowseID == "" {
		return "", false
	}

	browseTree, err := c.postInnerTube(ctx, "browse", map[string]any{
		"context":  ytWebRemixContext(),
		"browseId": lyricsBrowseID,
	})
	if err != nil {
		return "", false
	}

	var text string
	var findDesc func(n any) bool
	findDesc = func(n any) bool {
		switch v := n.(type) {
		case map[string]any:
			if mdr, ok := v["musicDescriptionShelfRenderer"].(map[string]any); ok {
				if t := strings.TrimSpace(extractRunsText(mdr["description"])); t != "" {
					text = t
					return true
				}
			}
			for _, child := range v {
				if findDesc(child) {
					return true
				}
			}
		case []any:
			for _, child := range v {
				if findDesc(child) {
					return true
				}
			}
		}
		return false
	}
	findDesc(browseTree)
	return text, text != ""
}
