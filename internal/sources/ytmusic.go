package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type YTMusicTrack struct {
	VideoID       string `json:"videoId"`
	Title         string `json:"title"`
	Artist        string `json:"artist"`
	Album         string `json:"album"`
	DurationSec   int    `json:"durationSec"`
	CoverURL      string `json:"coverUrl"`
	Format        string `json:"format"`
	PlayCountText string `json:"playCountText,omitempty"`
	Explicit      bool   `json:"explicit,omitempty"`
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
	Explicit    bool   `json:"explicit,omitempty"`
}

type YTMusicShelf struct {
	Title string        `json:"title"`
	Items []YTMusicCard `json:"items"`
}

type YTMusicHomeFeed struct {
	Shelves      []YTMusicShelf `json:"shelves"`
	Tracks       []YTMusicTrack `json:"tracks"`
	Continuation string         `json:"continuation,omitempty"`
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

const (
	ytWebRemixClientName    = "WEB_REMIX"
	ytWebRemixClientVersion = "1.20260707.12.00"
	ytWebRemixClientID      = "67"
	ytWebRemixUserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0"
	ytChartsParams          = "ggMGCgQIgAQ%3D"
)

var ytVisitorDataRegex = regexp.MustCompile(`"(?:VISITOR_DATA|visitorData)"\s*:\s*"([^"]+)"`)

type YTMusicClient struct {
	httpClient      *http.Client
	mu              sync.RWMutex
	cache           map[string]cachedStream
	top100Cache     []YTMusicTrack
	top100Expires   time.Time
	regions         []string
	language        string
	regionOverride  bool
	homeVisitorData string
	visitorMu       sync.Mutex
	visitorData     string
	visitorLocale   string
	lastVisitorTry  time.Time
}

func NewYTMusicClient() *YTMusicClient {
	return &YTMusicClient{
		httpClient: &http.Client{Timeout: 14 * time.Second},
		cache:      map[string]cachedStream{},
		regions:    []string{"ID"},
		language:   "id",
	}
}

func normalizeYTLang(hl string) string {
	code := strings.ToLower(strings.TrimSpace(hl))
	if idx := strings.IndexAny(code, "-_"); idx > 0 {
		code = code[:idx]
	}
	switch code {
	case "id", "en", "ja", "ko", "hi", "pt", "de", "fr", "es", "ms":
		return code
	default:
		if len(code) == 2 {
			return code
		}
		return "id"
	}
}

func ytRegionDisplayName(gl string) string {
	switch strings.ToUpper(strings.TrimSpace(gl)) {
	case "ID":
		return "Indonesia"
	case "US":
		return "United States"
	case "GB":
		return "United Kingdom"
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
		return "Global"
	}
}

func ytAcceptLanguageHeader(hl, gl string) string {
	hl = normalizeYTLang(hl)
	gl = strings.ToUpper(strings.TrimSpace(gl))
	if len(gl) == 2 {
		return fmt.Sprintf("%s-%s,%s;q=0.9", hl, gl, hl)
	}
	return hl
}

func ytPrefCookieHeader(hl, gl string) string {
	hl = normalizeYTLang(hl)
	gl = strings.ToUpper(strings.TrimSpace(gl))
	if len(gl) != 2 {
		gl = "ID"
	}
	return fmt.Sprintf("PREF=hl=%s&gl=%s", hl, gl)
}

func (c *YTMusicClient) SetRegions(regions []string) {
	c.mu.RLock()
	hl := c.language
	c.mu.RUnlock()
	c.SetLocale(regions, hl)
}

func (c *YTMusicClient) SetLanguage(hl string) {
	c.mu.RLock()
	regions := make([]string, len(c.regions))
	copy(regions, c.regions)
	c.mu.RUnlock()
	c.SetLocale(regions, hl)
}

func (c *YTMusicClient) SetLocale(regions []string, hl string) {
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
	normHL := normalizeYTLang(hl)

	c.mu.Lock()
	changed := c.language != normHL || len(c.regions) != len(cleaned)
	if !changed {
		for i := range cleaned {
			if c.regions[i] != cleaned[i] {
				changed = true
				break
			}
		}
	}
	c.regions = cleaned
	c.language = normHL
	// Always activate region override mode so InnerTube browse/search honor gl/hl + PREF cookie
	// instead of pinning responses to the IP-minted visitorData geo (matching Auralys/InnerTubeX).
	c.regionOverride = true
	if changed {
		c.top100Cache = nil
		c.top100Expires = time.Time{}
		c.homeVisitorData = ""
	}
	c.mu.Unlock()

	if changed {
		c.visitorMu.Lock()
		c.visitorData = ""
		c.visitorLocale = ""
		c.lastVisitorTry = time.Time{}
		c.visitorMu.Unlock()
	}
}

func (c *YTMusicClient) getRegions() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.regions) == 0 {
		return []string{"ID"}
	}
	out := make([]string, len(c.regions))
	copy(out, c.regions)
	return out
}

func (c *YTMusicClient) primaryRegion() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.regions) == 0 || strings.TrimSpace(c.regions[0]) == "" {
		return "ID"
	}
	return c.regions[0]
}

func (c *YTMusicClient) primaryLanguage() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return normalizeYTLang(c.language)
}

func (c *YTMusicClient) isRegionOverrideActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.regionOverride
}

func (c *YTMusicClient) ytContext() map[string]any {
	return ytWebRemixContextForLocale(c.primaryRegion(), c.primaryLanguage())
}

func (c *YTMusicClient) ensureVisitorData(ctx context.Context, gl, hl string) string {
	localeKey := strings.ToUpper(gl) + "|" + normalizeYTLang(hl)
	c.visitorMu.Lock()
	if c.visitorData != "" && c.visitorLocale == localeKey {
		v := c.visitorData
		c.visitorMu.Unlock()
		return v
	}
	if !c.lastVisitorTry.IsZero() && time.Since(c.lastVisitorTry) < 30*time.Second && c.visitorLocale == localeKey {
		c.visitorMu.Unlock()
		return ""
	}
	c.lastVisitorTry = time.Now()
	c.visitorLocale = localeKey
	c.visitorMu.Unlock()

	vd := c.fetchServiceWorkerVisitorData(ctx, gl, hl)
	if vd == "" {
		vd = c.fetchHomepageVisitorData(ctx, gl, hl)
	}
	if vd != "" {
		c.visitorMu.Lock()
		c.visitorData = vd
		c.visitorLocale = localeKey
		c.visitorMu.Unlock()
	}
	return vd
}

func (c *YTMusicClient) fetchServiceWorkerVisitorData(ctx context.Context, gl, hl string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://www.youtube.com/sw.js_data", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", ytWebRemixUserAgent)
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	req.Header.Set("Accept-Language", ytAcceptLanguageHeader(hl, gl))
	req.Header.Set("Cookie", ytPrefCookieHeader(hl, gl))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(raw))
	if idx := strings.Index(text, "["); idx > 0 && idx < 16 {
		text = text[idx:]
	}
	var arr []any
	if err := json.Unmarshal([]byte(text), &arr); err == nil {
		if a0, ok := indexSlice(arr, 0).([]any); ok {
			if a2, ok := indexSlice(a0, 2).([]any); ok {
				if a00, ok := indexSlice(a2, 0).([]any); ok {
					if a000, ok := indexSlice(a00, 0).([]any); ok {
						if s, ok := indexSlice(a000, 13).(string); ok && strings.TrimSpace(s) != "" {
							return strings.TrimSpace(s)
						}
					}
				}
			}
		}
	}
	if m := ytVisitorDataRegex.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func (c *YTMusicClient) fetchHomepageVisitorData(ctx context.Context, gl, hl string) string {
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://music.youtube.com", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", ytWebRemixUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", ytAcceptLanguageHeader(hl, gl))
	req.Header.Set("Cookie", ytPrefCookieHeader(hl, gl))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return ""
	}
	if m := ytVisitorDataRegex.FindStringSubmatch(string(raw)); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func indexSlice(s []any, idx int) any {
	if idx < 0 || idx >= len(s) {
		return nil
	}
	return s[idx]
}

func (c *YTMusicClient) postInnerTube(ctx context.Context, endpoint string, payload map[string]any) (any, error) {
	gl := c.primaryRegion()
	hl := c.primaryLanguage()
	if ctxMap, ok := payload["context"].(map[string]any); ok {
		if cl, ok := ctxMap["client"].(map[string]any); ok {
			if g, ok := cl["gl"].(string); ok && len(strings.TrimSpace(g)) == 2 {
				gl = strings.ToUpper(strings.TrimSpace(g))
			}
			if h, ok := cl["hl"].(string); ok && strings.TrimSpace(h) != "" {
				hl = normalizeYTLang(h)
			}
		}
	}
	return c.postInnerTubeFor(ctx, endpoint, payload, gl, hl, nil)
}

func (c *YTMusicClient) postInnerTubeFor(ctx context.Context, endpoint string, payload map[string]any, gl, hl string, query url.Values) (any, error) {
	return c.postInnerTubeWithVisitor(ctx, endpoint, payload, gl, hl, query, "")
}

func (c *YTMusicClient) postInnerTubeWithVisitor(ctx context.Context, endpoint string, payload map[string]any, gl, hl string, query url.Values, explicitVisitor string) (any, error) {
	gl = strings.ToUpper(strings.TrimSpace(gl))
	if len(gl) != 2 {
		gl = "ID"
	}
	hl = normalizeYTLang(hl)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("prettyPrint", "false")
	for k, vals := range query {
		for _, v := range vals {
			q.Set(k, v)
		}
	}
	endpointURL := "https://music.youtube.com/youtubei/v1/" + endpoint + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", ytWebRemixUserAgent)
	req.Header.Set("X-Goog-Api-Format-Version", "1")
	req.Header.Set("X-YouTube-Client-Name", ytWebRemixClientID)
	req.Header.Set("X-YouTube-Client-Version", ytWebRemixClientVersion)
	req.Header.Set("Origin", "https://music.youtube.com")
	req.Header.Set("X-Origin", "https://music.youtube.com")
	req.Header.Set("Referer", "https://music.youtube.com/")
	req.Header.Set("Accept-Language", ytAcceptLanguageHeader(hl, gl))
	req.Header.Set("Cookie", ytPrefCookieHeader(hl, gl))

	if explicitVisitor != "" {
		req.Header.Set("X-Goog-Visitor-Id", explicitVisitor)
	} else {
		// Omit X-Goog-Visitor-Id on browse/search when region override is active so YouTube
		// honors the selected gl/hl + PREF cookie instead of IP-baked visitorData geo.
		omitVisitor := c.isRegionOverrideActive() && (endpoint == "browse" || endpoint == "search")
		if !omitVisitor {
			if vd := c.ensureVisitorData(ctx, gl, hl); vd != "" {
				req.Header.Set("X-Goog-Visitor-Id", vd)
			}
		}
	}

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

func extractResponseVisitorData(tree any) string {
	m, ok := tree.(map[string]any)
	if !ok {
		return ""
	}
	rc, ok := m["responseContext"].(map[string]any)
	if !ok {
		return ""
	}
	vd, _ := rc["visitorData"].(string)
	return strings.TrimSpace(vd)
}

func ytWebRemixContextFor(gl string) map[string]any {
	return ytWebRemixContextForLocale(gl, "id")
}

func ytWebRemixContextForLocale(gl, hl string) map[string]any {
	gl = strings.ToUpper(strings.TrimSpace(gl))
	if len(gl) != 2 {
		gl = "ID"
	}
	hl = normalizeYTLang(hl)
	return map[string]any{
		"client": map[string]any{
			"clientName":    ytWebRemixClientName,
			"clientVersion": ytWebRemixClientVersion,
			"hl":            hl,
			"gl":            gl,
		},
	}
}

func ytWebRemixContext() map[string]any {
	return ytWebRemixContextFor("ID")
}

func PickBestYTMusicVideoID(hits []YTMusicTrack, title, artist string, targetDurationSec int) string {
	bestID := ""
	bestScore := -1e9
	reqTitle := strings.ToLower(strings.TrimSpace(title))
	reqArtist := strings.ToLower(strings.TrimSpace(primaryLyricArtist(artist)))
	for _, h := range hits {
		vid := strings.TrimSpace(h.VideoID)
		if vid == "" {
			continue
		}
		score := 0.0
		if targetDurationSec > 0 && h.DurationSec > 0 {
			diff := math.Abs(float64(h.DurationSec - targetDurationSec))
			score += 250.0 - diff*20.0
		}
		if h.Album != "" && !strings.EqualFold(strings.TrimSpace(h.Album), "YouTube Music") {
			score += 80.0
		}
		hitTitle := strings.ToLower(strings.TrimSpace(h.Title))
		for _, tag := range []string{"official video", "music video", "lyric video", "mv", "live", "extended"} {
			if strings.Contains(hitTitle, tag) && !strings.Contains(reqTitle, tag) {
				score -= 150.0
			}
		}
		if reqTitle != "" && (strings.Contains(hitTitle, reqTitle) || strings.Contains(reqTitle, hitTitle)) {
			score += 30.0
		}
		if reqArtist != "" && strings.Contains(strings.ToLower(h.Artist), reqArtist) {
			score += 20.0
		}
		if bestID == "" || score > bestScore {
			bestScore = score
			bestID = vid
		}
	}
	return bestID
}

func (c *YTMusicClient) Search(ctx context.Context, query string) ([]YTMusicTrack, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		feed, err := c.GetHomeFeed(ctx)
		return feed.Tracks, err
	}

	payload := map[string]any{
		"context": c.ytContext(),
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

func (c *YTMusicClient) SearchPlaylists(ctx context.Context, query string) ([]YTMusicCard, error) {
	query = strings.TrimSpace(query)
	var results []YTMusicCard
	seen := map[string]struct{}{}

	if query == "" {
		if feed, err := c.GetHomeFeed(ctx); err == nil {
			for _, sh := range feed.Shelves {
				for _, item := range sh.Items {
					if item.Kind == "playlist" && (item.BrowseID != "" || item.PlaylistID != "") {
						if _, dup := seen[item.ID]; !dup {
							seen[item.ID] = struct{}{}
							results = append(results, item)
						}
					}
				}
			}
		}
		query = "Top Hits " + ytRegionDisplayName(c.primaryRegion()) + " Playlist"
	}

	if len(results) < 24 {
		for _, params := range []string{
			"Eg-KAQwIABAAGAAgACgBMABqChAEEAMQCRAFEAo%3D", // Featured playlists
			"EgeKAQQoAEABagoQAxAEEAkQChAF",                 // Community playlists
		} {
			payload := map[string]any{
				"context": c.ytContext(),
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
	}

	if len(results) > 24 {
		results = results[:24]
	}
	if results == nil {
		results = []YTMusicCard{}
	}
	return results, nil
}

func (c *YTMusicClient) SearchAlbums(ctx context.Context, query string) ([]YTMusicCard, error) {
	query = strings.TrimSpace(query)
	var results []YTMusicCard
	seen := map[string]struct{}{}

	if query == "" {
		if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  c.ytContext(),
			"browseId": "FEmusic_new_releases_albums",
		}); err == nil {
			extractAlbumSearchItems(tree, &results, seen)
		}
		query = "Top " + ytRegionDisplayName(c.primaryRegion()) + " Album"
	}

	if len(results) < 28 {
		payload := map[string]any{
			"context": c.ytContext(),
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

func (c *YTMusicClient) SearchArtists(ctx context.Context, query string) ([]YTMusicCard, error) {
	query = strings.TrimSpace(query)
	var results []YTMusicCard
	seen := map[string]struct{}{}

	queries := []string{query}
	if query == "" {
		// Pull strictly country-localized top artists from FEmusic_charts using formData.selectedValues.
		gl := c.primaryRegion()
		if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  c.ytContext(),
			"browseId": "FEmusic_charts",
			"formData": map[string]any{"selectedValues": []string{gl}},
		}); err == nil {
			var chartShelves []YTMusicShelf
			extractCarouselShelves(tree, &chartShelves)
			for _, sh := range chartShelves {
				for _, item := range sh.Items {
					if item.Kind == "artist" && item.BrowseID != "" {
						if _, dup := seen[item.BrowseID]; !dup {
							seen[item.BrowseID] = struct{}{}
							results = append(results, item)
						}
					}
				}
			}
		}
		regionName := ytRegionDisplayName(gl)
		queries = []string{"Top " + regionName + " Artists", "Popular " + regionName + " Singers"}
	}

	for _, q := range queries {
		if len(results) >= 28 {
			break
		}
		payload := map[string]any{
			"context": c.ytContext(),
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

func isPodcastShelfTitle(title string) bool {
	low := strings.ToLower(strings.TrimSpace(title))
	return strings.Contains(low, "podcast") || strings.Contains(low, "episode") || strings.Contains(low, "ポッドキャスト") || strings.Contains(low, "팟캐스트")
}

func isExcludedHomeShelfTitle(title string) bool {
	if isPodcastShelfTitle(title) {
		return true
	}
	low := strings.ToLower(strings.TrimSpace(title))
	return strings.Contains(low, "covers & remixes") ||
		strings.Contains(low, "covers and remixes") ||
		strings.Contains(low, "cover & remix")
}

func homeShelfPriority(title string) int {
	low := strings.ToLower(strings.TrimSpace(title))
	switch {
	case strings.Contains(low, "new release") || strings.Contains(low, "rilis baru"):
		return 0
	case strings.Contains(low, "today's indonesian hits") || strings.Contains(low, "today indonesian hits") || strings.Contains(low, "today indonesia hits") || strings.Contains(low, "hits indonesia hari ini"):
		return 1
	case strings.Contains(low, "all-time hits") || strings.Contains(low, "all time hits") || strings.Contains(low, "hits sepanjang masa"):
		return 2
	case strings.Contains(low, "trending songs for you") || strings.Contains(low, "lagu trending untuk anda"):
		return 3
	case strings.Contains(low, "trending community playlist") || strings.Contains(low, "playlist komunitas trending"):
		return 4
	case strings.Contains(low, "song mixes") || strings.Contains(low, "songs mixes") || strings.Contains(low, "mix lagu"):
		return 5
	case strings.Contains(low, "rain + feels"):
		return 6
	case strings.Contains(low, "monsoon melancholy"):
		return 7
	default:
		return 1000
	}
}

func hasAllPriorityHomeShelves(shelves []YTMusicShelf) bool {
	found := [8]bool{}
	for _, sh := range shelves {
		p := homeShelfPriority(sh.Title)
		if p >= 0 && p < 8 {
			found[p] = true
		}
	}
	for _, ok := range found {
		if !ok {
			return false
		}
	}
	return true
}

func sortHomeShelves(shelves []YTMusicShelf) []YTMusicShelf {
	if len(shelves) <= 1 {
		return shelves
	}
	out := make([]YTMusicShelf, len(shelves))
	copy(out, shelves)
	sort.SliceStable(out, func(i, j int) bool {
		return homeShelfPriority(out[i].Title) < homeShelfPriority(out[j].Title)
	})
	return out
}

func isSongShelf(sh YTMusicShelf) bool {
	if len(sh.Items) == 0 {
		return false
	}
	songs := 0
	for _, it := range sh.Items {
		if it.Kind == "song" && it.VideoID != "" {
			songs++
		}
	}
	return songs*2 >= len(sh.Items)
}

func ytLocalizedRegionName(hl, gl string) string {
	if normalizeYTLang(hl) == "id" {
		switch strings.ToUpper(strings.TrimSpace(gl)) {
		case "ID":
			return "Indonesia"
		case "US":
			return "Amerika Serikat"
		case "GB":
			return "Britania Raya"
		case "JP":
			return "Jepang"
		case "KR":
			return "Korea Selatan"
		case "IN":
			return "India"
		case "BR":
			return "Brasil"
		case "DE":
			return "Jerman"
		case "FR":
			return "Prancis"
		case "MX":
			return "Meksiko"
		case "AU":
			return "Australia"
		case "MY":
			return "Malaysia"
		}
	}
	return ytRegionDisplayName(gl)
}

func localizedTopSongsTitle(hl, gl string) string {
	region := ytLocalizedRegionName(hl, gl)
	switch normalizeYTLang(hl) {
	case "id":
		return "Lagu Teratas " + region
	case "ms":
		return "Lagu Teratas " + region
	case "ja":
		return region + "のトップソング"
	case "ko":
		return region + " 인기 곡"
	default:
		return "Top Songs " + region
	}
}

func localizedAlbumsTitle(hl, gl string) string {
	region := ytLocalizedRegionName(hl, gl)
	switch normalizeYTLang(hl) {
	case "id":
		return "Album Populer " + region
	case "ms":
		return "Album Popular " + region
	case "ja":
		return region + "の人気アルバム"
	case "ko":
		return region + " 인기 앨범"
	default:
		return "Popular Albums " + region
	}
}

func pickPrimaryChartPlaylist(shelves []YTMusicShelf) *YTMusicCard {
	var fallback *YTMusicCard
	for _, sh := range shelves {
		if isPodcastShelfTitle(sh.Title) {
			continue
		}
		for i := range sh.Items {
			it := sh.Items[i]
			if it.Kind != "playlist" || (it.BrowseID == "" && it.PlaylistID == "") {
				continue
			}
			low := strings.ToLower(it.Title)
			if fallback == nil {
				cp := it
				fallback = &cp
			}
			// Prefer Trending 20 / Daily Top Music Videos / Top 100 Songs over Live Performances
			if strings.Contains(low, "trending") || strings.Contains(low, "harian") || strings.Contains(low, "daily") || strings.Contains(low, "급상승") || strings.Contains(low, "일간") {
				cp := it
				return &cp
			}
		}
	}
	return fallback
}

func (c *YTMusicClient) GetHomeFeed(ctx context.Context) (YTMusicHomeFeed, error) {
	feed := YTMusicHomeFeed{
		Shelves: []YTMusicShelf{},
		Tracks:  []YTMusicTrack{},
	}

	gl := c.primaryRegion()
	hl := c.primaryLanguage()

	// 1:1 with Auralys (YouTube.home -> FEmusic_home + homeContinuation):
	// Fetch page 1 of FEmusic_home directly, capture responseContext.visitorData & continuation,
	// then seamlessly fetch page 2 continuation so the initial Home screen shows authentic
	// YouTube Music Home shelves (Quick picks / Mix lagu, Moods, Trending, Hits, etc.)
	// and exposes the remaining continuation token for "Muat lebih banyak".
	tree1, err := c.postInnerTubeFor(ctx, "browse", map[string]any{
		"context":  ytWebRemixContextForLocale(gl, hl),
		"browseId": "FEmusic_home",
	}, gl, hl, nil)
	if err != nil {
		return feed, err
	}

	var shelves1 []YTMusicShelf
	extractCarouselShelves(tree1, &shelves1)
	cont := extractContinuationToken(tree1)
	vd := extractResponseVisitorData(tree1)
	if vd != "" {
		c.mu.Lock()
		c.homeVisitorData = vd
		c.mu.Unlock()
	}

	shelfIdxByKey := map[string]int{}
	seenShelfItem := map[string]map[string]struct{}{}
	seenTrack := map[string]struct{}{}

	filterHomeShelves := func(in []YTMusicShelf) []YTMusicShelf {
		out := make([]YTMusicShelf, 0, len(in))
		for _, sh := range in {
			if !isExcludedHomeShelfTitle(sh.Title) {
				out = append(out, sh)
			}
		}
		return out
	}

	appendShelvesToFeed(&feed, filterHomeShelves(shelves1), shelfIdxByKey, seenShelfItem, seenTrack)

	// Automatically fetch continuation pages of FEmusic_home until we have the prioritized home shelves
	// (or up to 3 continuation pages) so initial Home displays them right away.
	for page := 0; page < 3 && cont != ""; page++ {
		if page > 0 && hasAllPriorityHomeShelves(feed.Shelves) {
			break
		}
		pageNext, errNext := c.GetHomeContinuation(ctx, cont)
		if errNext != nil {
			break
		}
		appendShelvesToFeed(&feed, pageNext.Shelves, shelfIdxByKey, seenShelfItem, seenTrack)
		cont = pageNext.Continuation
	}
	feed.Shelves = sortHomeShelves(feed.Shelves)
	feed.Continuation = cont

	return feed, nil
}

func (c *YTMusicClient) GetHomeContinuation(ctx context.Context, continuation string) (YTMusicHomeFeed, error) {
	feed := YTMusicHomeFeed{
		Shelves: []YTMusicShelf{},
		Tracks:  []YTMusicTrack{},
	}
	token := strings.TrimSpace(continuation)
	if token == "" {
		return feed, nil
	}
	gl := c.primaryRegion()
	hl := c.primaryLanguage()
	c.mu.RLock()
	vd := c.homeVisitorData
	c.mu.RUnlock()

	q := url.Values{}
	q.Set("ctoken", token)
	q.Set("continuation", token)
	q.Set("type", "next")

	ctxObj := ytWebRemixContextForLocale(gl, hl)
	if vd != "" {
		if cl, ok := ctxObj["client"].(map[string]any); ok {
			cl["visitorData"] = vd
		}
	}

	tree, err := c.postInnerTubeWithVisitor(ctx, "browse", map[string]any{
		"context":      ctxObj,
		"continuation": token,
	}, gl, hl, q, vd)
	if err != nil {
		return feed, err
	}

	var shelves []YTMusicShelf
	extractCarouselShelves(tree, &shelves)
	filtered := make([]YTMusicShelf, 0, len(shelves))
	for _, sh := range shelves {
		if !isExcludedHomeShelfTitle(sh.Title) {
			filtered = append(filtered, sh)
		}
	}
	feed.Continuation = extractContinuationToken(tree)
	appendShelvesToFeed(&feed, filtered, map[string]int{}, map[string]map[string]struct{}{}, map[string]struct{}{})
	feed.Shelves = sortHomeShelves(feed.Shelves)
	return feed, nil
}

func appendShelvesToFeed(
	feed *YTMusicHomeFeed,
	shelves []YTMusicShelf,
	shelfIdxByKey map[string]int,
	seenShelfItem map[string]map[string]struct{},
	seenTrack map[string]struct{},
) {
	for _, sh := range shelves {
		key := strings.ToLower(strings.TrimSpace(sh.Title))
		if key == "" || len(sh.Items) == 0 {
			continue
		}
		targetIdx, exists := shelfIdxByKey[key]
		if !exists {
			targetIdx = len(feed.Shelves)
			shelfIdxByKey[key] = targetIdx
			seenShelfItem[key] = map[string]struct{}{}
			feed.Shelves = append(feed.Shelves, YTMusicShelf{
				Title: sh.Title,
				Items: make([]YTMusicCard, 0, len(sh.Items)),
			})
		}

		for _, item := range sh.Items {
			itemKey := item.ID
			if itemKey == "" {
				itemKey = item.VideoID + "|" + strings.ToLower(item.Title)
			}
			if _, dup := seenShelfItem[key][itemKey]; dup {
				continue
			}
			seenShelfItem[key][itemKey] = struct{}{}
			feed.Shelves[targetIdx].Items = append(feed.Shelves[targetIdx].Items, item)

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
						Explicit:    item.Explicit,
					})
				}
			}
		}
	}
}

func (c *YTMusicClient) GetTop100Tracks(ctx context.Context) ([]YTMusicTrack, error) {
	c.mu.RLock()
	if len(c.top100Cache) > 0 && time.Now().Before(c.top100Expires) {
		out := make([]YTMusicTrack, len(c.top100Cache))
		copy(out, c.top100Cache)
		c.mu.RUnlock()
		return out, nil
	}
	c.mu.RUnlock()

	var results []YTMusicTrack
	seen := map[string]struct{}{}

	// Fetch strictly country-localized charts first using formData.selectedValues.
	gl := c.primaryRegion()
	if tree, chartErr := c.postInnerTube(ctx, "browse", map[string]any{
		"context":  c.ytContext(),
		"browseId": "FEmusic_charts",
		"formData": map[string]any{"selectedValues": []string{gl}},
	}); chartErr == nil {
		extractResponsiveItems(tree, &results, seen)
		var chartShelves []YTMusicShelf
		extractCarouselShelves(tree, &chartShelves)
		chartPlaylist := pickPrimaryChartPlaylist(chartShelves)
		for _, sh := range chartShelves {
			if isPodcastShelfTitle(sh.Title) {
				continue
			}
			for _, item := range sh.Items {
				if item.VideoID != "" {
					if _, dup := seen[item.VideoID]; !dup {
						seen[item.VideoID] = struct{}{}
						dur := item.DurationSec
						if dur <= 0 {
							dur = 200
						}
						results = append(results, YTMusicTrack{
							VideoID:     item.VideoID,
							Title:       item.Title,
							Artist:      item.Artist,
							Album:       item.Album,
							DurationSec: dur,
							CoverURL:    item.CoverURL,
							Format:      "OPUS",
						})
					}
				}
			}
		}
		if len(results) < 25 && chartPlaylist != nil {
			if plTracks, pErr := c.GetCollectionTracks(ctx, chartPlaylist.BrowseID, chartPlaylist.PlaylistID, chartPlaylist.Title, chartPlaylist.Artist, chartPlaylist.CoverURL); pErr == nil {
				for _, t := range plTracks {
					if _, dup := seen[t.VideoID]; !dup {
						seen[t.VideoID] = struct{}{}
						results = append(results, t)
					}
				}
			}
		}
	}

	if len(results) < 10 {
		if globalTracks, err := c.GetCollectionTracks(
			ctx,
			"VLPL4fGSI1pDJn6puJdseH2Rt9sMvt9E2M4i",
			"PL4fGSI1pDJn6puJdseH2Rt9sMvt9E2M4i",
			"",
			"",
			"",
		); err == nil {
			for _, t := range globalTracks {
				if _, dup := seen[t.VideoID]; !dup {
					seen[t.VideoID] = struct{}{}
					results = append(results, t)
				}
			}
		}
	}
	if len(results) < 10 {
		if feed, feedErr := c.GetHomeFeed(ctx); feedErr == nil && len(feed.Tracks) > 0 {
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
					"context":     c.ytContext(),
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
					"context": c.ytContext(),
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
			"context":  c.ytContext(),
			"browseId": targetID,
		}); err == nil {
			extractResponsiveItems(tree, &results, seen)
		}
	}

	if len(results) == 0 && pidClean != "" {
		vlID := "VL" + pidClean
		if vlID != targetID {
			if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
				"context":  c.ytContext(),
				"browseId": vlID,
			}); err == nil {
				extractResponsiveItems(tree, &results, seen)
			}
		}
	}

	if len(results) == 0 && pidClean != "" {
		if tree, err := c.postInnerTube(ctx, "next", map[string]any{
			"context":    c.ytContext(),
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

	isAlbumCollection := strings.HasPrefix(targetID, "MPREb_")
	for i := range results {
		if fallbackCover != "" && (results[i].CoverURL == "" || (isAlbumCollection && strings.Contains(results[i].CoverURL, "i.ytimg.com/vi/"))) {
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

func (c *YTMusicClient) GetTrackAlbumDetail(ctx context.Context, videoID, browseID, fallbackTitle, fallbackArtist, fallbackAlbum, fallbackCover string) (YTMusicAlbumDetail, error) {
	albumBrowseID := strings.TrimSpace(browseID)
	artistBrowseID := ""
	artistName := cleanCollectionSubtitle(fallbackArtist)
	albumTitle := strings.TrimSpace(fallbackAlbum)
	cover := strings.TrimSpace(fallbackCover)

	isPlaylistBrowse := albumBrowseID != "" && !strings.HasPrefix(albumBrowseID, "MPREb_")

	if !strings.HasPrefix(albumBrowseID, "MPREb_") && !isPlaylistBrowse && strings.TrimSpace(videoID) != "" {
		if tree, err := c.postInnerTube(ctx, "next", map[string]any{
			"context": c.ytContext(),
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

	if !strings.HasPrefix(albumBrowseID, "MPREb_") && !isPlaylistBrowse {
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
			if !strings.HasPrefix(albumBrowseID, "MPREb_") && !isPlaylistBrowse {
				if tracks, err := c.Search(ctx, q); err == nil && len(tracks) > 0 && tracks[0].VideoID != "" && tracks[0].VideoID != videoID {
					if tree, err := c.postInnerTube(ctx, "next", map[string]any{
						"context": c.ytContext(),
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
			"context":  c.ytContext(),
			"browseId": albumBrowseID,
		}); err == nil {
			extractAlbumHeaderMeta(tree, &detail)
			seen := map[string]struct{}{}
			extractResponsiveItems(tree, &detail.Tracks, seen)
		}
	} else if isPlaylistBrowse {
		if tracks, err := c.GetCollectionTracks(ctx, albumBrowseID, "", artistName, albumTitle, cover); err == nil {
			detail.Tracks = tracks
		}
	}

	if detail.Title == "" {
		detail.Title = fallbackTitle
	}
	if detail.Artist == "" {
		detail.Artist = "YouTube Music"
	}

	for i := range detail.Tracks {
		if detail.CoverURL != "" && (detail.Tracks[i].CoverURL == "" || (!isPlaylistBrowse && strings.Contains(detail.Tracks[i].CoverURL, "i.ytimg.com/vi/"))) {
			detail.Tracks[i].CoverURL = detail.CoverURL
		}
		if detail.Tracks[i].Artist == "" || detail.Tracks[i].Artist == "YouTube Music Artist" {
			detail.Tracks[i].Artist = detail.Artist
		}
		if detail.Tracks[i].Album == "" || detail.Tracks[i].Album == "YouTube Music" {
			detail.Tracks[i].Album = detail.Title
		}
	}

	var otherAlbums []YTMusicCard
	seenAlbums := map[string]struct{}{}
	if detail.BrowseID != "" {
		seenAlbums[detail.BrowseID] = struct{}{}
	}
	if detail.ArtistBrowseID != "" {
		if tree, err := c.postInnerTube(ctx, "browse", map[string]any{
			"context":  c.ytContext(),
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
	for _, prefix := range []string{"Album • ", "Single • ", "EP • ", "Playlist • "} {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
	}
	return s
}

func extractContinuationToken(node any) string {
	switch v := node.(type) {
	case map[string]any:
		if ncd, ok := v["nextContinuationData"].(map[string]any); ok {
			if tok, ok := ncd["continuation"].(string); ok && strings.TrimSpace(tok) != "" {
				return strings.TrimSpace(tok)
			}
		}
		if cir, ok := v["continuationItemRenderer"].(map[string]any); ok {
			if ep, ok := cir["continuationEndpoint"].(map[string]any); ok {
				if cmd, ok := ep["continuationCommand"].(map[string]any); ok {
					if tok, ok := cmd["token"].(string); ok && strings.TrimSpace(tok) != "" {
						return strings.TrimSpace(tok)
					}
				}
			}
		}
		for _, child := range v {
			if tok := extractContinuationToken(child); tok != "" {
				return tok
			}
		}
	case []any:
		for _, child := range v {
			if tok := extractContinuationToken(child); tok != "" {
				return tok
			}
		}
	}
	return ""
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
									Explicit:    tr.Explicit,
								})
							}
						} else if artistCard, ok := parseResponsiveArtistItem(respItem); ok {
							if _, dup := seen[artistCard.ID]; !dup {
								seen[artistCard.ID] = struct{}{}
								cards = append(cards, artistCard)
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
		if grid, ok := v["gridRenderer"].(map[string]any); ok {
			title := ""
			if hdr, ok := grid["header"].(map[string]any); ok {
				if ghr, ok := hdr["gridHeaderRenderer"].(map[string]any); ok {
					title = strings.TrimSuffix(strings.TrimSpace(extractRunsText(ghr["title"])), ".")
				}
			}
			var cards []YTMusicCard
			seen := map[string]struct{}{}
			if items, ok := grid["items"].([]any); ok {
				for _, entry := range items {
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
	} else if strings.HasPrefix(browseID, "UC") || pageType == "MUSIC_PAGE_TYPE_ARTIST" {
		kind = "artist"
	} else if strings.HasPrefix(browseID, "MPREb_") || pageType == "MUSIC_PAGE_TYPE_ALBUM" {
		kind = "album"
	}

	explicit := false
	if badges, ok := r["subtitleBadges"].([]any); ok {
		for _, b := range badges {
			if bm, ok := b.(map[string]any); ok {
				if ib, ok := bm["musicInlineBadgeRenderer"].(map[string]any); ok {
					if ic, ok := ib["icon"].(map[string]any); ok {
						if it, _ := ic["iconType"].(string); it == "MUSIC_EXPLICIT_BADGE" {
							explicit = true
						}
					}
				}
			}
		}
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
	if kind == "artist" {
		artist = title
	} else if artist == "" {
		artist = "YouTube Music"
	}

	albumTitle := title
	if kind == "song" {
		albumTitle = "YouTube Music"
		// Check subtitle runs for linked album browseId (MPREb_), matching HomePage.Section.fromMusicTwoRowItemRenderer.
		if subObj, ok := r["subtitle"].(map[string]any); ok {
			if runs, ok := subObj["runs"].([]any); ok {
				for _, runItem := range runs {
					rm, ok := runItem.(map[string]any)
					if !ok {
						continue
					}
					if rnav, ok := rm["navigationEndpoint"].(map[string]any); ok {
						if rbe, ok := rnav["browseEndpoint"].(map[string]any); ok {
							if rbid, _ := rbe["browseId"].(string); strings.HasPrefix(rbid, "MPREb_") {
								browseID = rbid
								if rt, _ := rm["text"].(string); strings.TrimSpace(rt) != "" {
									albumTitle = strings.TrimSpace(rt)
								}
							}
						}
					}
				}
			}
		}
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
		Album:      albumTitle,
		Explicit:   explicit,
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
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	if strings.HasPrefix(u, "//") {
		u = "https:" + u
	}
	if m := ytImgVideoRe.FindStringSubmatch(u); len(m) == 2 {
		return "https://i.ytimg.com/vi/" + m[1] + "/hqdefault.jpg"
	}
	if strings.Contains(u, "googleusercontent.com") || strings.Contains(u, "yt3.ggpht.com") {
		return ytThumbSizeRe.ReplaceAllString(u, "=w544-h544-l90-rj")
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

func (c *YTMusicClient) ClearAllStreamCache() {
	c.mu.Lock()
	clear(c.cache)
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
			} else if quality == "normal" && f.Bitrate >= 64000 && f.Bitrate <= 135000 {
				score += 500_000
				if f.Itag == 140 {
					score += 40_000
				}
			} else {
				// Prefer Opus 160kbps (itag 251) or AAC 256kbps (itag 141)
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

func (c *YTMusicClient) GetLyricsByVideoID(ctx context.Context, videoID string) (string, bool) {
	vid := strings.TrimSpace(videoID)
	if vid == "" {
		return "", false
	}
	nextTree, err := c.postInnerTube(ctx, "next", map[string]any{
		"context": c.ytContext(),
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
		"context":  c.ytContext(),
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
