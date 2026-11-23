package sources

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
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

var (
	lrcTagRegex      = regexp.MustCompile(`\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?\]`)
	lrcOffsetRegex   = regexp.MustCompile(`(?i)\[offset:([+-]?\d+)\]`)
	titleParenRegex  = regexp.MustCompile(`\s*[\(\[](?:official\s*(?:music\s*)?(?:video|audio)|lyrics?|lyric\s*video|visualizer|hd|4k|remaster(?:ed)?|live(?:\s+at\s+[^\)\]]+)?|feat\.[^\)\]]+|ft\.[^\)\]]+)[\)\]]`)
	titleFeatRegex   = regexp.MustCompile(`(?i)\s+(?:feat\.|ft\.)\s+.*$`)
	artistSplitRegex = regexp.MustCompile(`(?i)\s*(?:,|&|feat\.|ft\.)\s*`)
)

func isLyricCreditLine(text string, timeSec float64) bool {
	if timeSec > 18.0 {
		return false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	low := strings.ToLower(trimmed)
	creditPrefixes := []string{
		"作词", "作曲", "编曲", "制作人", "制作", "混音", "母带", "录音", "监制", "出品", "演唱",
		"词：", "词:", "曲：", "曲:", "编：", "编:",
		"lyrics by", "lyricist", "composed by", "composer", "arranged by", "arrangement by",
		"produced by", "producer", "written by", "mixed by", "mastered by",
		"lirik oleh", "pencipta", "cipt.",
	}
	for _, prefix := range creditPrefixes {
		if strings.HasPrefix(low, prefix) {
			return true
		}
	}
	return false
}

func parseLRC(raw string) []LyricLine {
	offsetSec := 0.0
	rows := strings.Split(raw, "\n")
	for _, row := range rows {
		if m := lrcOffsetRegex.FindStringSubmatch(row); len(m) == 2 {
			if msVal, err := strconv.Atoi(m[1]); err == nil {
				offsetSec = float64(msVal) / 1000.0
			}
		}
	}

	var rawLines []LyricLine
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		matches := lrcTagRegex.FindAllStringSubmatchIndex(row, -1)
		if len(matches) == 0 {
			continue
		}
		lastEnd := matches[len(matches)-1][1]
		text := strings.TrimSpace(row[lastEnd:])
		for _, loc := range matches {
			sub := lrcTagRegex.FindStringSubmatch(row[loc[0]:loc[1]])
			if len(sub) != 4 {
				continue
			}
			mins, _ := strconv.Atoi(sub[1])
			secs, _ := strconv.Atoi(sub[2])
			ms := 0
			if sub[3] != "" {
				msVal, _ := strconv.Atoi(sub[3])
				switch len(sub[3]) {
				case 1:
					ms = msVal * 100
				case 2:
					ms = msVal * 10
				default:
					ms = msVal
				}
			}
			totalSec := float64(mins*60+secs) + float64(ms)/1000.0 + offsetSec
			if totalSec < 0 {
				totalSec = 0
			}
			rawLines = append(rawLines, LyricLine{
				TimeSec: totalSec,
				Text:    text,
			})
		}
	}
	sort.SliceStable(rawLines, func(i, j int) bool {
		return rawLines[i].TimeSec < rawLines[j].TimeSec
	})

	lines := make([]LyricLine, 0, len(rawLines))
	seenVocal := false
	lastTime := -10.0
	for _, entry := range rawLines {
		if entry.Text == "" {
			if seenVocal && entry.TimeSec-lastTime >= 4.0 {
				lines = append(lines, LyricLine{
					TimeSec: entry.TimeSec,
					Text:    "♪",
				})
				lastTime = entry.TimeSec
			}
			continue
		}
		if isLyricCreditLine(entry.Text, entry.TimeSec) {
			continue
		}
		seenVocal = true
		lines = append(lines, entry)
		lastTime = entry.TimeSec
	}
	return lines
}

func alignIntroOffset(lines []LyricLine, lrcDurationSec float64, trackDurationSec int) []LyricLine {
	if len(lines) == 0 || trackDurationSec <= 0 || lrcDurationSec <= 0 {
		return lines
	}
	delta := float64(trackDurationSec) - lrcDurationSec
	if delta >= 2.5 && delta <= 90.0 && lines[0].TimeSec < 12.0 && lines[len(lines)-1].TimeSec+delta <= float64(trackDurationSec)+2.0 {
		for i := range lines {
			lines[i].TimeSec += delta
		}
	}
	return lines
}

func cleanLyricTitle(title string) string {
	s := titleParenRegex.ReplaceAllString(title, "")
	s = titleFeatRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func primaryLyricArtist(artist string) string {
	parts := artistSplitRegex.Split(strings.TrimSpace(artist), 2)
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0])
	}
	return strings.TrimSpace(artist)
}

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
	if strings.EqualFold(cleanArtist, "Unknown Artist") {
		cleanArtist = ""
	}
	if cleanTitle == "" {
		return out
	}

	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 3500 * time.Millisecond}

	var plainFallback *TrackLyrics
	var syncedFallback *TrackLyrics
	savePlain := func(candidate TrackLyrics) {
		if plainFallback == nil && strings.TrimSpace(candidate.PlainLyrics) != "" {
			cp := candidate
			plainFallback = &cp
		}
	}
	saveSynced := func(candidate TrackLyrics) {
		if syncedFallback == nil && len(candidate.SyncedLyrics) > 0 {
			cp := candidate
			syncedFallback = &cp
		}
	}

	params := url.Values{}
	params.Set("track_name", cleanTitle)
	if cleanArtist != "" {
		params.Set("artist_name", cleanArtist)
	}
	if album != "" && !strings.EqualFold(album, "Single") && !strings.EqualFold(album, "YouTube Music") {
		params.Set("album_name", album)
	}
	if durationSec > 0 {
		params.Set("duration", strconv.Itoa(durationSec))
	}
	if res, ok := queryLRCLIBGet(reqCtx, client, "https://lrclib.net/api/get?"+params.Encode()); ok {
		populated := populateLyrics(out, res, durationSec)
		durMatch := durationSec <= 0 || res.Duration <= 0 || math.Abs(res.Duration-float64(durationSec)) <= 2.5
		if populated.Source == "instrumental" || (len(populated.SyncedLyrics) > 0 && durMatch) {
			return populated
		}
		saveSynced(populated)
		savePlain(populated)
	}

	shortTitle := cleanLyricTitle(cleanTitle)
	shortArtist := primaryLyricArtist(cleanArtist)
	searchQueries := []string{strings.TrimSpace(cleanArtist + " " + cleanTitle)}
	if alt := strings.TrimSpace(shortArtist + " " + shortTitle); alt != "" && alt != searchQueries[0] {
		searchQueries = append(searchQueries, alt)
	}
	for _, sq := range searchQueries {
		searchURL := "https://lrclib.net/api/search?q=" + url.QueryEscape(sq)
		if list, ok := queryLRCLIBSearch(reqCtx, client, searchURL); ok && len(list) > 0 {
			best := pickBestLRCLIB(list, cleanTitle, cleanArtist, durationSec)
			populated := populateLyrics(out, best, durationSec)
			populated.Source = "lrclib-search"
			durMatch := durationSec <= 0 || best.Duration <= 0 || math.Abs(best.Duration-float64(durationSec)) <= 2.5
			if populated.Source == "instrumental" || (len(populated.SyncedLyrics) > 0 && durMatch) {
				return populated
			}
			saveSynced(populated)
			savePlain(populated)
		}
	}

	if ne, ok := queryNetEaseLyrics(reqCtx, client, out, shortTitle, shortArtist, durationSec); ok {
		if len(ne.SyncedLyrics) > 0 {
			return ne
		}
		savePlain(ne)
	}

	if kg, ok := queryKuGouLyrics(reqCtx, client, out, shortTitle, shortArtist, durationSec); ok {
		if len(kg.SyncedLyrics) > 0 {
			return kg
		}
		savePlain(kg)
	}

	if syncedFallback != nil {
		return *syncedFallback
	}

	if plainFallback == nil && shortArtist != "" && shortTitle != "" {
		if ovh, ok := queryLyricsOVH(reqCtx, client, out, shortTitle, shortArtist); ok {
			savePlain(ovh)
		}
	}

	if plainFallback != nil {
		return *plainFallback
	}

	out.Source = "fallback"
	out.PlainLyrics = fmt.Sprintf("%s — %s\n\nLirik belum tersedia secara otomatis untuk lagu ini.\nKamu tetap dapat menikmati pemutaran audio dan kontrol antrean secara penuh.", cleanTitle, cleanArtist)
	return out
}

func pickBestLRCLIB(list []lrclibResponse, title, artist string, durationSec int) lrclibResponse {
	best := list[0]
	bestScore := -1e9
	lowTitle := strings.ToLower(strings.TrimSpace(cleanLyricTitle(title)))
	lowArtist := strings.ToLower(strings.TrimSpace(primaryLyricArtist(artist)))
	for _, item := range list {
		score := 0.0
		if strings.TrimSpace(item.SyncedLyrics) != "" {
			score += 1000.0
		} else if strings.TrimSpace(item.PlainLyrics) != "" {
			score += 100.0
		}
		if durationSec > 0 && item.Duration > 0 {
			diff := math.Abs(item.Duration - float64(durationSec))
			if diff <= 2.0 {
				score += 300.0
			}
			score -= diff * 25.0
		}
		itemTitle := strings.ToLower(strings.TrimSpace(cleanLyricTitle(item.TrackName)))
		if lowTitle != "" && itemTitle != "" {
			if itemTitle == lowTitle {
				score += 120.0
			} else if strings.Contains(itemTitle, lowTitle) || strings.Contains(lowTitle, itemTitle) {
				score += 50.0
			}
		}
		itemArtist := strings.ToLower(strings.TrimSpace(item.ArtistName))
		if lowArtist != "" && itemArtist != "" && strings.Contains(itemArtist, lowArtist) {
			score += 80.0
		}
		if score > bestScore {
			bestScore = score
			best = item
		}
	}
	return best
}

func queryNetEaseLyrics(ctx context.Context, client *http.Client, base TrackLyrics, title, artist string, durationSec int) (TrackLyrics, bool) {
	q := strings.TrimSpace(artist + " " + title)
	if q == "" {
		return base, false
	}
	searchURL := "https://music.163.com/api/search/get/web?s=" + url.QueryEscape(q) + "&type=1&offset=0&total=true&limit=8"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return base, false
	}
	req.Header.Set("Referer", "https://music.163.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return base, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	resp.Body.Close()
	if err != nil {
		return base, false
	}
	var sr struct {
		Result struct {
			Songs []struct {
				ID       int64 `json:"id"`
				Duration int   `json:"duration"`
			} `json:"songs"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &sr) != nil || len(sr.Result.Songs) == 0 {
		return base, false
	}

	if durationSec > 0 && len(sr.Result.Songs) > 1 {
		target := float64(durationSec)
		sort.SliceStable(sr.Result.Songs, func(i, j int) bool {
			di := math.Abs(float64(sr.Result.Songs[i].Duration)/1000.0 - target)
			dj := math.Abs(float64(sr.Result.Songs[j].Duration)/1000.0 - target)
			return di < dj
		})
	}

	for _, s := range sr.Result.Songs {
		lyricURL := fmt.Sprintf("https://music.163.com/api/song/lyric?id=%d&lv=1&tv=-1", s.ID)
		lreq, err := http.NewRequestWithContext(ctx, http.MethodGet, lyricURL, nil)
		if err != nil {
			continue
		}
		lreq.Header.Set("Referer", "https://music.163.com/")
		lreq.Header.Set("User-Agent", "Mozilla/5.0")
		lresp, err := client.Do(lreq)
		if err != nil || lresp.StatusCode != http.StatusOK {
			if lresp != nil {
				lresp.Body.Close()
			}
			continue
		}
		lraw, err := io.ReadAll(io.LimitReader(lresp.Body, 512<<10))
		lresp.Body.Close()
		if err != nil {
			continue
		}
		var lr struct {
			LRC struct {
				Lyric string `json:"lyric"`
			} `json:"lrc"`
		}
		if json.Unmarshal(lraw, &lr) != nil || strings.TrimSpace(lr.LRC.Lyric) == "" {
			continue
		}
		parsed := alignIntroOffset(parseLRC(lr.LRC.Lyric), float64(s.Duration)/1000.0, durationSec)
		if len(parsed) > 0 {
			base.Source = "netease"
			base.SyncedLyrics = parsed
			var parts []string
			for _, l := range parsed {
				parts = append(parts, l.Text)
			}
			base.PlainLyrics = strings.Join(parts, "\n")
			return base, true
		}
	}
	return base, false
}

func queryKuGouLyrics(ctx context.Context, client *http.Client, base TrackLyrics, title, artist string, durationSec int) (TrackLyrics, bool) {
	q := strings.TrimSpace(artist + " " + title)
	if q == "" {
		return base, false
	}
	durMs := durationSec * 1000
	if durMs <= 0 {
		durMs = 180000
	}
	searchURL := fmt.Sprintf(
		"https://krcs.kugou.com/search?ver=1&man=yes&client=mobi&keyword=%s&duration=%d&hash=",
		url.QueryEscape(q),
		durMs,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return base, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return base, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	resp.Body.Close()
	if err != nil {
		return base, false
	}
	var sr struct {
		Candidates []struct {
			ID        string `json:"id"`
			AccessKey string `json:"accesskey"`
			Duration  int    `json:"duration"`
		} `json:"candidates"`
	}
	if json.Unmarshal(raw, &sr) != nil || len(sr.Candidates) == 0 {
		return base, false
	}

	if durationSec > 0 && len(sr.Candidates) > 1 {
		target := float64(durationSec)
		sort.SliceStable(sr.Candidates, func(i, j int) bool {
			di := math.Abs(float64(sr.Candidates[i].Duration)/1000.0 - target)
			dj := math.Abs(float64(sr.Candidates[j].Duration)/1000.0 - target)
			return di < dj
		})
	}

	for _, c := range sr.Candidates {
		if c.ID == "" || c.AccessKey == "" {
			continue
		}
		dlURL := fmt.Sprintf(
			"https://lyrics.kugou.com/download?ver=1&client=pc&id=%s&accesskey=%s&fmt=lrc&charset=utf8",
			url.QueryEscape(c.ID),
			url.QueryEscape(c.AccessKey),
		)
		dreq, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
		if err != nil {
			continue
		}
		dreq.Header.Set("User-Agent", "Mozilla/5.0")
		dresp, err := client.Do(dreq)
		if err != nil || dresp.StatusCode != http.StatusOK {
			if dresp != nil {
				dresp.Body.Close()
			}
			continue
		}
		draw, err := io.ReadAll(io.LimitReader(dresp.Body, 512<<10))
		dresp.Body.Close()
		if err != nil {
			continue
		}
		var dr struct {
			Content string `json:"content"`
		}
		if json.Unmarshal(draw, &dr) != nil || dr.Content == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(dr.Content)
		if err != nil {
			continue
		}
		parsed := alignIntroOffset(parseLRC(string(decoded)), float64(c.Duration)/1000.0, durationSec)
		if len(parsed) > 0 {
			base.Source = "kugou"
			base.SyncedLyrics = parsed
			var parts []string
			for _, l := range parsed {
				parts = append(parts, l.Text)
			}
			base.PlainLyrics = strings.Join(parts, "\n")
			return base, true
		}
	}
	return base, false
}

func queryLyricsOVH(ctx context.Context, client *http.Client, base TrackLyrics, title, artist string) (TrackLyrics, bool) {
	endpoint := fmt.Sprintf(
		"https://api.lyrics.ovh/v1/%s/%s",
		url.PathEscape(artist),
		url.PathEscape(title),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return base, false
	}
	req.Header.Set("User-Agent", "VeloraMusicPlayer/1.0")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return base, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	resp.Body.Close()
	if err != nil {
		return base, false
	}
	var lr struct {
		Lyrics string `json:"lyrics"`
	}
	if json.Unmarshal(raw, &lr) != nil || strings.TrimSpace(lr.Lyrics) == "" {
		return base, false
	}
	base.Source = "lyrics.ovh"
	base.PlainLyrics = strings.TrimSpace(lr.Lyrics)
	return base, true
}

func populateLyrics(out TrackLyrics, res lrclibResponse, trackDurationSec int) TrackLyrics {
	if res.Instrumental && strings.TrimSpace(res.PlainLyrics) == "" && strings.TrimSpace(res.SyncedLyrics) == "" {
		out.Source = "instrumental"
		out.PlainLyrics = "[Musik Instrumental]"
		return out
	}
	if strings.TrimSpace(res.SyncedLyrics) != "" {
		out.SyncedLyrics = alignIntroOffset(parseLRC(res.SyncedLyrics), res.Duration, trackDurationSec)
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
