package sources

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type LyricWord struct {
	TimeSec     float64 `json:"timeSec"`
	DurationSec float64 `json:"durationSec"`
	Text        string  `json:"text"`
}

type LyricLine struct {
	TimeSec float64     `json:"timeSec"`
	Text    string      `json:"text"`
	Words   []LyricWord `json:"words,omitempty"`
}

type TrackLyrics struct {
	TrackID            int64       `json:"trackId"`
	Title              string      `json:"title"`
	Artist             string      `json:"artist"`
	PlainLyrics        string      `json:"plainLyrics"`
	SyncedLyrics       []LyricLine `json:"syncedLyrics"`
	Source             string      `json:"source"`
	MatchedDurationSec float64     `json:"-"`
	NeedsVideoVerify   bool        `json:"-"`
}

type SponsorBlockInterval struct {
	Start float64
	End   float64
}

type sbCacheEntry struct {
	intervals []SponsorBlockInterval
	found     bool
	expiresAt time.Time
}

var (
	sbCacheMu sync.Mutex
	sbCache   = map[string]sbCacheEntry{}
)

const sbCacheMaxEntries = 256

type lrclibResponse struct {
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
}

var (
	lrcTagRegex            = regexp.MustCompile(`\[(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?\]`)
	elrcTagRegex           = regexp.MustCompile(`<(\d{1,2}):(\d{2})(?:[.:](\d{1,3}))?>`)
	yrcLineRegex           = regexp.MustCompile(`^\[(\d+),(\d+)\](.*)$`)
	yrcWordRegex           = regexp.MustCompile(`\((\d+),(\d+),\d+\)([^(\r\n]*)`)
	lrcOffsetRegex         = regexp.MustCompile(`(?i)\[offset:([+-]?\d+)\]`)
	titleParenRegex        = regexp.MustCompile(`(?i)\s*[\(\[](?:official\s*(?:music\s*)?(?:video|audio)|lyrics?|lyric\s*video|visualizer|audio|explicit|hd|4k|remaster(?:ed)?(?:\s+\d+)?|live(?:\s+at\s+[^\)\]]+)?|(?:feat\.|ft\.|with)\s+[^\)\]]+)[\)\]]`)
	titleDashSuffixRegex   = regexp.MustCompile(`(?i)\s+-\s+(?:remaster(?:ed)?(?:\s+\d{4})?|\d{4}\s+remaster(?:ed)?|single\s+version|radio\s+edit|album\s+version|acoustic(?:\s+version)?|live(?:\s+.*)?|bonus\s+track|from\s+.*|explicit|mono|stereo)$`)
	titleFeatRegex         = regexp.MustCompile(`(?i)\s+(?:feat\.|ft\.|with)\s+.*$`)
	artistSplitRegex       = regexp.MustCompile(`(?i)\s*(?:,|&|/|feat\.|ft\.|\bx\b)\s*`)
	artistTopicSuffixRegex = regexp.MustCompile(`(?i)\s*-\s*topic$|\s*vevo$`)
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

func parseLRCTimestampMatch(sub []string) (float64, bool) {
	if len(sub) != 4 {
		return 0, false
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
	return float64(mins*60+secs) + float64(ms)/1000.0, true
}

func parseEnhancedWords(rawText string, offsetSec float64) (string, []LyricWord) {
	locs := elrcTagRegex.FindAllStringSubmatchIndex(rawText, -1)
	if len(locs) == 0 {
		return strings.TrimSpace(rawText), nil
	}
	type rawWord struct {
		t    float64
		text string
	}
	var rw []rawWord
	var cleanParts []string
	for i, loc := range locs {
		sub := elrcTagRegex.FindStringSubmatch(rawText[loc[0]:loc[1]])
		tSec, ok := parseLRCTimestampMatch(sub)
		if !ok {
			continue
		}
		tSec = math.Max(0, tSec+offsetSec)
		segEnd := len(rawText)
		if i+1 < len(locs) {
			segEnd = locs[i+1][0]
		}
		seg := strings.TrimSpace(rawText[loc[1]:segEnd])
		if seg == "" {
			continue
		}
		for _, tok := range strings.Fields(seg) {
			cleanParts = append(cleanParts, tok)
			rw = append(rw, rawWord{t: tSec, text: tok})
		}
	}
	if len(rw) == 0 {
		return strings.TrimSpace(elrcTagRegex.ReplaceAllString(rawText, "")), nil
	}
	words := make([]LyricWord, len(rw))
	for i, w := range rw {
		dur := 0.45
		if i+1 < len(rw) {
			d := rw[i+1].t - w.t
			if d > 0.08 {
				dur = d
			} else {
				dur = 0.25
			}
		}
		words[i] = LyricWord{
			TimeSec:     w.t,
			DurationSec: dur,
			Text:        w.text,
		}
	}
	return strings.Join(cleanParts, " "), words
}

func parseNetEaseYRC(raw string) []LyricLine {
	var lines []LyricLine
	for _, row := range strings.Split(raw, "\n") {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		lm := yrcLineRegex.FindStringSubmatch(row)
		if len(lm) != 4 {
			continue
		}
		lineStartMs, err1 := strconv.Atoi(lm[1])
		if err1 != nil {
			continue
		}
		lineStartSec := float64(lineStartMs) / 1000.0
		body := lm[3]
		wMatches := yrcWordRegex.FindAllStringSubmatch(body, -1)
		if len(wMatches) == 0 {
			continue
		}
		var words []LyricWord
		var textBuilder strings.Builder
		for _, wm := range wMatches {
			wStartMs, _ := strconv.Atoi(wm[1])
			wDurMs, _ := strconv.Atoi(wm[2])
			wRaw := wm[3]
			wTrim := strings.TrimSpace(wRaw)
			if wTrim == "" {
				continue
			}
			wStartSec := float64(wStartMs) / 1000.0
			wDurSec := math.Max(0.08, float64(wDurMs)/1000.0)
			if textBuilder.Len() > 0 {
				textBuilder.WriteByte(' ')
			}
			textBuilder.WriteString(wTrim)
			words = append(words, LyricWord{
				TimeSec:     wStartSec,
				DurationSec: wDurSec,
				Text:        wTrim,
			})
		}
		fullText := strings.TrimSpace(textBuilder.String())
		if fullText == "" || isLyricCreditLine(fullText, lineStartSec) {
			continue
		}
		lines = append(lines, LyricLine{
			TimeSec: lineStartSec,
			Text:    fullText,
			Words:   words,
		})
	}
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].TimeSec < lines[j].TimeSec
	})
	return lines
}

func shiftLyricLine(line *LyricLine, delta float64) {
	line.TimeSec = math.Max(0, line.TimeSec+delta)
	for j := range line.Words {
		line.Words[j].TimeSec = math.Max(0, line.Words[j].TimeSec+delta)
	}
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
		rawBody := strings.TrimSpace(row[lastEnd:])
		cleanText, words := parseEnhancedWords(rawBody, offsetSec)
		for _, loc := range matches {
			sub := lrcTagRegex.FindStringSubmatch(row[loc[0]:loc[1]])
			baseSec, ok := parseLRCTimestampMatch(sub)
			if !ok {
				continue
			}
			totalSec := math.Max(0, baseSec+offsetSec)
			var clonedWords []LyricWord
			if len(words) > 0 && len(matches) == 1 {
				clonedWords = words
			}
			rawLines = append(rawLines, LyricLine{
				TimeSec: totalSec,
				Text:    cleanText,
				Words:   clonedWords,
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
	if delta >= 2.5 && delta <= 90.0 && lines[0].TimeSec < 45.0 && lines[len(lines)-1].TimeSec+delta <= float64(trackDurationSec)+2.5 {
		for i := range lines {
			shiftLyricLine(&lines[i], delta)
		}
	}
	return lines
}

func cleanLyricTitle(title string) string {
	s := titleParenRegex.ReplaceAllString(title, "")
	s = titleDashSuffixRegex.ReplaceAllString(s, "")
	s = titleFeatRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func cleanLyricArtist(artist string) string {
	s := strings.TrimSpace(strings.ReplaceAll(artist, "\u00a0", " "))
	if idx := strings.IndexAny(s, "\u2022\u00b7"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	s = artistTopicSuffixRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func primaryLyricArtist(artist string) string {
	cleaned := cleanLyricArtist(artist)
	parts := artistSplitRegex.Split(cleaned, 2)
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		return cleanLyricArtist(parts[0])
	}
	return cleaned
}

func cleanLyricAlbum(album, cleanTitle, shortTitle string) string {
	a := strings.TrimSpace(album)
	if a == "" ||
		strings.EqualFold(a, "Single") ||
		strings.EqualFold(a, "YouTube Music") ||
		strings.EqualFold(a, "Imported from Spotify") ||
		strings.EqualFold(a, "Unknown Album") ||
		(cleanTitle != "" && strings.EqualFold(a, cleanTitle)) ||
		(shortTitle != "" && strings.EqualFold(a, shortTitle)) {
		return ""
	}
	return a
}

func FetchSponsorBlockMusicOfftopic(ctx context.Context, client *http.Client, videoID string) ([]SponsorBlockInterval, bool) {
	vid := strings.TrimSpace(videoID)
	if vid == "" || isHybridVideoID(vid) {
		return nil, false
	}

	now := time.Now()
	sbCacheMu.Lock()
	if ent, ok := sbCache[vid]; ok && now.Before(ent.expiresAt) {
		out := make([]SponsorBlockInterval, len(ent.intervals))
		copy(out, ent.intervals)
		found := ent.found
		sbCacheMu.Unlock()
		return out, found
	}
	sbCacheMu.Unlock()

	if client == nil {
		client = &http.Client{Timeout: 1800 * time.Millisecond}
	}
	sbCtx, cancel := context.WithTimeout(ctx, 1800*time.Millisecond)
	defer cancel()

	endpoint := "https://sponsor.ajay.app/api/skipSegments?videoID=" + url.QueryEscape(vid) + "&category=music_offtopic"
	req, err := http.NewRequestWithContext(sbCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", "VeloraMusicPlayer/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		sbCacheMu.Lock()
		if len(sbCache) >= sbCacheMaxEntries {
			for k := range sbCache {
				delete(sbCache, k)
				break
			}
		}
		sbCache[vid] = sbCacheEntry{found: false, expiresAt: now.Add(2 * time.Hour)}
		sbCacheMu.Unlock()
		return nil, false
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return nil, false
	}

	var rows []struct {
		Category   string     `json:"category"`
		ActionType string     `json:"actionType"`
		Segment    [2]float64 `json:"segment"`
		Votes      int        `json:"votes"`
		Locked     int        `json:"locked"`
	}
	if json.Unmarshal(raw, &rows) != nil {
		return nil, false
	}

	type sbCand struct {
		start  float64
		end    float64
		locked int
		votes  int
	}
	var cands []sbCand
	for _, r := range rows {
		if r.Category != "music_offtopic" {
			continue
		}
		if r.ActionType != "" && r.ActionType != "skip" {
			continue
		}
		if r.Votes < 0 && r.Locked <= 0 {
			continue
		}
		s, e := r.Segment[0], r.Segment[1]
		if s < 0 || e-s < 1.5 {
			continue
		}
		cands = append(cands, sbCand{start: s, end: e, locked: r.Locked, votes: r.Votes})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].locked != cands[j].locked {
			return cands[i].locked > cands[j].locked
		}
		if cands[i].votes != cands[j].votes {
			return cands[i].votes > cands[j].votes
		}
		return (cands[i].end - cands[i].start) > (cands[j].end - cands[j].start)
	})

	var accepted []SponsorBlockInterval
	for _, c := range cands {
		overlaps := false
		for _, a := range accepted {
			if !(c.end <= a.Start+0.5 || c.start >= a.End-0.5) {
				overlaps = true
				break
			}
		}
		if !overlaps {
			accepted = append(accepted, SponsorBlockInterval{Start: c.start, End: c.end})
		}
	}
	sort.Slice(accepted, func(i, j int) bool {
		return accepted[i].Start < accepted[j].Start
	})

	sbCacheMu.Lock()
	if len(sbCache) >= sbCacheMaxEntries {
		for k := range sbCache {
			delete(sbCache, k)
			break
		}
	}
	sbCache[vid] = sbCacheEntry{
		intervals: accepted,
		found:     len(accepted) > 0,
		expiresAt: now.Add(2 * time.Hour),
	}
	sbCacheMu.Unlock()

	out := make([]SponsorBlockInterval, len(accepted))
	copy(out, accepted)
	return out, len(accepted) > 0
}

// AlignLyricsTimeline shifts studio LRC timestamps to follow MV intros & mid-video skits using SponsorBlock intervals (ignoring outros).
func AlignLyricsTimeline(lines []LyricLine, lrcDurationSec float64, trackDurationSec int, sbIntervals []SponsorBlockInterval, hasSB bool) ([]LyricLine, bool) {
	if len(lines) == 0 || trackDurationSec <= 0 {
		return lines, false
	}
	if lrcDurationSec > 0 && math.Abs(float64(trackDurationSec)-lrcDurationSec) < 2.5 {
		return lines, false
	}
	if hasSB && len(sbIntervals) > 0 {
		shifted := make([]LyricLine, len(lines))
		copy(shifted, lines)
		for _, iv := range sbIntervals {
			segDur := iv.End - iv.Start
			if segDur < 1.5 {
				continue
			}
			if iv.Start <= 1.5 && shifted[0].TimeSec >= iv.End-1.5 {
				continue
			}
			if iv.Start >= float64(trackDurationSec)-2.0 {
				continue
			}
			hasAfter := false
			for i := range shifted {
				if shifted[i].TimeSec >= iv.Start-0.35 {
					hasAfter = true
					break
				}
			}
			if !hasAfter {
				continue
			}
			if shifted[len(shifted)-1].TimeSec+segDur > float64(trackDurationSec)+3.5 {
				continue
			}
			for i := range shifted {
				if shifted[i].TimeSec >= iv.Start-0.35 {
					shiftLyricLine(&shifted[i], segDur)
				}
			}
		}
		return shifted, false
	}
	aligned := alignIntroOffset(lines, lrcDurationSec, trackDurationSec)
	heuristicUsed := lrcDurationSec > 0 && math.Abs(float64(trackDurationSec)-lrcDurationSec) > 4.0
	return aligned, heuristicUsed
}

// AlignLyricsWithVideo aligns lyrics using YouTube Captions/ASR first, then SponsorBlock, then intro offset.
func AlignLyricsWithVideo(ctx context.Context, videoID string, lines []LyricLine, lrcDurationSec float64, trackDurationSec int) []LyricLine {
	if len(lines) == 0 {
		return lines
	}
	vid := strings.TrimSpace(videoID)
	if vid != "" && !isHybridVideoID(vid) {
		client := &http.Client{Timeout: 3200 * time.Millisecond}
		if cueTracks, ok := FetchYouTubeCaptionCues(ctx, client, vid); ok {
			if aligned, okCap := AlignLyricsWithCaptions(lines, cueTracks, trackDurationSec); okCap {
				return aligned
			}
		}
		if trackDurationSec > 0 {
			if ivs, ok := FetchSponsorBlockMusicOfftopic(ctx, client, vid); ok {
				aligned, _ := AlignLyricsTimeline(lines, lrcDurationSec, trackDurationSec, ivs, true)
				return aligned
			}
		}
	}
	if trackDurationSec <= 0 {
		return lines
	}
	aligned, _ := AlignLyricsTimeline(lines, lrcDurationSec, trackDurationSec, nil, false)
	return aligned
}

type CaptionCue struct {
	StartSec float64
	Text     string
	Words    []string
}

type captionCacheEntry struct {
	cueTracks [][]CaptionCue
	found     bool
	expiresAt time.Time
}

var (
	captionCacheMu sync.Mutex
	captionCache   = map[string]captionCacheEntry{}
	bracketTagRe   = regexp.MustCompile(`\[[^\]]*\]`)
)

const captionCacheMaxEntries = 128

func tokenizeLyricWords(text string) []string {
	cleaned := bracketTagRe.ReplaceAllString(text, " ")
	cleaned = strings.ToLower(cleaned)
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			w := cur.String()
			if len([]rune(w)) >= 2 {
				words = append(words, w)
			}
			cur.Reset()
		}
	}
	for _, r := range cleaned {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return words
}

// FetchYouTubeCaptionCues retrieves normalized manual/ASR caption tracks from YouTube's IOS player endpoint.
func FetchYouTubeCaptionCues(ctx context.Context, client *http.Client, videoID string) ([][]CaptionCue, bool) {
	vid := strings.TrimSpace(videoID)
	if vid == "" || isHybridVideoID(vid) {
		return nil, false
	}

	now := time.Now()
	captionCacheMu.Lock()
	if ent, ok := captionCache[vid]; ok && now.Before(ent.expiresAt) {
		captionCacheMu.Unlock()
		return ent.cueTracks, ent.found
	}
	captionCacheMu.Unlock()

	if client == nil {
		client = &http.Client{Timeout: 3200 * time.Millisecond}
	}
	pCtx, cancel := context.WithTimeout(ctx, 3200*time.Millisecond)
	defer cancel()

	bodyBytes, err := json.Marshal(map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "IOS",
				"clientVersion": "20.10.4",
				"hl":            "id",
				"gl":            "ID",
			},
		},
		"videoId": vid,
	})
	if err != nil {
		return nil, false
	}
	req, err := http.NewRequestWithContext(pCtx, http.MethodPost, "https://www.youtube.com/youtubei/v1/player", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "com.google.ios.youtube/20.10.4 (iPhone16,2; U; CPU iOS 18_2 like Mac OS X;)")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, false
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	resp.Body.Close()
	if err != nil {
		return nil, false
	}

	var playerResp struct {
		Captions struct {
			PlayerCaptionsTracklistRenderer struct {
				CaptionTracks []struct {
					BaseURL      string `json:"baseUrl"`
					LanguageCode string `json:"languageCode"`
					Kind         string `json:"kind"`
				} `json:"captionTracks"`
			} `json:"playerCaptionsTracklistRenderer"`
		} `json:"captions"`
	}
	if json.Unmarshal(raw, &playerResp) != nil {
		return nil, false
	}

	tracks := playerResp.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks
	if len(tracks) == 0 {
		captionCacheMu.Lock()
		if len(captionCache) >= captionCacheMaxEntries {
			for k := range captionCache {
				delete(captionCache, k)
				break
			}
		}
		captionCache[vid] = captionCacheEntry{found: false, expiresAt: now.Add(2 * time.Hour)}
		captionCacheMu.Unlock()
		return nil, false
	}

	type capTrack struct {
		url  string
		lang string
		kind string
	}
	var manualNonEn, manualEn, asrTracks []capTrack
	for _, t := range tracks {
		if strings.TrimSpace(t.BaseURL) == "" {
			continue
		}
		ct := capTrack{
			url:  t.BaseURL,
			lang: strings.ToLower(strings.TrimSpace(t.LanguageCode)),
			kind: strings.ToLower(strings.TrimSpace(t.Kind)),
		}
		if ct.kind == "asr" {
			asrTracks = append(asrTracks, ct)
		} else if !strings.HasPrefix(ct.lang, "en") {
			manualNonEn = append(manualNonEn, ct)
		} else {
			manualEn = append(manualEn, ct)
		}
	}

	var chosen []capTrack
	if len(manualNonEn) > 0 {
		chosen = append(chosen, manualNonEn[0])
	} else if len(manualEn) > 0 {
		chosen = append(chosen, manualEn[0])
	}
	if len(asrTracks) > 0 {
		chosen = append(chosen, asrTracks[0])
	}

	var cueTracks [][]CaptionCue
	for _, ct := range chosen {
		cURL := ct.url
		if !strings.Contains(cURL, "fmt=") {
			if strings.Contains(cURL, "?") {
				cURL += "&fmt=json3"
			} else {
				cURL += "?fmt=json3"
			}
		}
		cCtx, cCancel := context.WithTimeout(ctx, 2500*time.Millisecond)
		creq, err := http.NewRequestWithContext(cCtx, http.MethodGet, cURL, nil)
		if err != nil {
			cCancel()
			continue
		}
		creq.Header.Set("User-Agent", "Mozilla/5.0")
		cresp, err := client.Do(creq)
		if err != nil || cresp.StatusCode != http.StatusOK {
			if cresp != nil {
				cresp.Body.Close()
			}
			cCancel()
			continue
		}
		craw, err := io.ReadAll(io.LimitReader(cresp.Body, 2<<20))
		cresp.Body.Close()
		cCancel()
		if err != nil {
			continue
		}

		var json3 struct {
			Events []struct {
				TStartMs int `json:"tStartMs"`
				Segs     []struct {
					UTF8      string `json:"utf8"`
					TOffsetMs int    `json:"tOffsetMs"`
				} `json:"segs"`
			} `json:"events"`
		}
		if json.Unmarshal(craw, &json3) != nil {
			continue
		}

		var cues []CaptionCue
		for _, ev := range json3.Events {
			if len(ev.Segs) == 0 {
				continue
			}
			var sb strings.Builder
			firstOffsetMs := -1
			for _, seg := range ev.Segs {
				txt := strings.ReplaceAll(seg.UTF8, "\n", " ")
				sb.WriteString(txt)
				if firstOffsetMs < 0 && len(tokenizeLyricWords(txt)) > 0 {
					firstOffsetMs = seg.TOffsetMs
				}
			}
			fullText := strings.TrimSpace(sb.String())
			words := tokenizeLyricWords(fullText)
			if len(words) == 0 {
				continue
			}
			startMs := ev.TStartMs
			if firstOffsetMs > 0 {
				startMs += firstOffsetMs
			}
			cues = append(cues, CaptionCue{
				StartSec: float64(startMs) / 1000.0,
				Text:     fullText,
				Words:    words,
			})
		}
		if len(cues) > 0 {
			cueTracks = append(cueTracks, cues)
		}
	}

	found := len(cueTracks) > 0
	captionCacheMu.Lock()
	if len(captionCache) >= captionCacheMaxEntries {
		for k := range captionCache {
			delete(captionCache, k)
			break
		}
	}
	captionCache[vid] = captionCacheEntry{
		cueTracks: cueTracks,
		found:     found,
		expiresAt: now.Add(2 * time.Hour),
	}
	captionCacheMu.Unlock()

	return cueTracks, found
}

type captionAnchor struct {
	lineIdx int
	lrcSec  float64
	capSec  float64
	delta   float64
}

func countWordMatches(lineWords, cueWords []string) (matched int, longMatched int) {
	if len(lineWords) == 0 || len(cueWords) == 0 {
		return 0, 0
	}
	cueSet := make(map[string]struct{}, len(cueWords))
	for _, cw := range cueWords {
		cueSet[cw] = struct{}{}
	}
	seen := make(map[string]struct{}, len(lineWords))
	for _, lw := range lineWords {
		if _, dup := seen[lw]; dup {
			continue
		}
		seen[lw] = struct{}{}
		if _, ok := cueSet[lw]; ok {
			matched++
			if len([]rune(lw)) >= 4 {
				longMatched++
			}
		}
	}
	return matched, longMatched
}

func medianDelta(anchors []captionAnchor) float64 {
	if len(anchors) == 0 {
		return 0
	}
	vals := make([]float64, len(anchors))
	for i, a := range anchors {
		vals[i] = a.delta
	}
	sort.Float64s(vals)
	return vals[len(vals)/2]
}

// AlignLyricsWithCaptions matches LRC lyric lines against YouTube manual/ASR captions
// to automatically shift timestamps for MV intros, outros, and mid-video skits.
func AlignLyricsWithCaptions(lines []LyricLine, cueTracks [][]CaptionCue, trackDurationSec int) ([]LyricLine, bool) {
	if len(lines) == 0 || len(cueTracks) == 0 {
		return lines, false
	}

	var bestAnchors []captionAnchor
	for _, cues := range cueTracks {
		if len(cues) < 2 {
			continue
		}
		var anchors []captionAnchor
		lastCueTime := -10.0
		for i, line := range lines {
			lw := tokenizeLyricWords(line.Text)
			if len(lw) < 2 {
				continue
			}
			bestScore := 0.0
			bestCapSec := 0.0
			matchedCue := false
			for k := 0; k < len(cues); k++ {
				if cues[k].StartSec < lastCueTime-1.0 {
					continue
				}
				if math.Abs(cues[k].StartSec-line.TimeSec) > 120.0 {
					continue
				}
				m1, lm1 := countWordMatches(lw, cues[k].Words)
				ratio1 := float64(m1) / float64(len(lw))
				if m1 >= 2 && (ratio1 >= 0.5 || lm1 >= 2) {
					sc := float64(m1)*10.0 + ratio1*5.0
					if sc > bestScore {
						bestScore = sc
						bestCapSec = cues[k].StartSec
						matchedCue = true
					}
				}
				if k+1 < len(cues) && cues[k+1].StartSec-cues[k].StartSec <= 4.5 {
					merged := make([]string, 0, len(cues[k].Words)+len(cues[k+1].Words))
					merged = append(merged, cues[k].Words...)
					merged = append(merged, cues[k+1].Words...)
					m2, lm2 := countWordMatches(lw, merged)
					ratio2 := float64(m2) / float64(len(lw))
					if m2 > m1 && m2 >= 2 && (ratio2 >= 0.5 || lm2 >= 2) {
						sc := float64(m2)*10.0 + ratio2*5.0 - 0.5
						if sc > bestScore {
							bestScore = sc
							bestCapSec = cues[k].StartSec
							matchedCue = true
						}
					}
				}
			}
			if matchedCue {
				anchors = append(anchors, captionAnchor{
					lineIdx: i,
					lrcSec:  line.TimeSec,
					capSec:  bestCapSec,
					delta:   bestCapSec - line.TimeSec,
				})
				lastCueTime = bestCapSec
			}
		}
		if len(anchors) > len(bestAnchors) {
			bestAnchors = anchors
		}
	}

	if len(bestAnchors) < 2 {
		return lines, false
	}

	// Keep anchors that belong to consistent (>= 2 anchors within 2.2s delta) clusters
	var consistent []captionAnchor
	for i, a := range bestAnchors {
		peers := 0
		for j, b := range bestAnchors {
			if i != j && math.Abs(a.delta-b.delta) <= 2.2 {
				peers++
			}
		}
		if peers >= 1 {
			consistent = append(consistent, a)
		}
	}
	if len(consistent) < 2 {
		return lines, false
	}

	// Detect piecewise mid-video skit split if early & late clusters diverge by >= 3.0s
	splitIdx := -1
	var earlyDelta, lateDelta float64
	for k := 2; k <= len(consistent)-2; k++ {
		d1 := medianDelta(consistent[:k])
		d2 := medianDelta(consistent[k:])
		if d2-d1 >= 3.0 {
			earlyConsistent := true
			for _, a := range consistent[:k] {
				if math.Abs(a.delta-d1) > 2.2 {
					earlyConsistent = false
					break
				}
			}
			lateConsistent := true
			for _, a := range consistent[k:] {
				if math.Abs(a.delta-d2) > 2.2 {
					lateConsistent = false
					break
				}
			}
			if earlyConsistent && lateConsistent {
				splitIdx = consistent[k].lineIdx
				earlyDelta = d1
				lateDelta = d2
				break
			}
		}
	}

	if splitIdx >= 0 {
		shifted := make([]LyricLine, len(lines))
		copy(shifted, lines)
		for i := range shifted {
			if len(shifted[i].Words) > 0 {
				cpW := make([]LyricWord, len(shifted[i].Words))
				copy(cpW, shifted[i].Words)
				shifted[i].Words = cpW
			}
			d := earlyDelta
			if i >= splitIdx {
				d = lateDelta
			}
			shiftLyricLine(&shifted[i], d)
		}
		return shifted, true
	}

	deltaMed := medianDelta(consistent)
	if math.Abs(deltaMed) < 0.8 {
		return lines, true
	}
	if trackDurationSec > 0 && lines[len(lines)-1].TimeSec+deltaMed > float64(trackDurationSec)+8.0 {
		return lines, false
	}
	shifted := make([]LyricLine, len(lines))
	copy(shifted, lines)
	for i := range shifted {
		if len(shifted[i].Words) > 0 {
			cpW := make([]LyricWord, len(shifted[i].Words))
			copy(cpW, shifted[i].Words)
			shifted[i].Words = cpW
		}
		shiftLyricLine(&shifted[i], deltaMed)
	}
	return shifted, true
}

func FetchLyrics(ctx context.Context, trackID int64, title, artist, album string, durationSec int) TrackLyrics {
	return FetchLyricsWithVideoOpts(ctx, trackID, "", title, artist, album, durationSec, false)
}

func FetchLyricsWithVideo(ctx context.Context, trackID int64, videoID, title, artist, album string, durationSec int) TrackLyrics {
	return FetchLyricsWithVideoOpts(ctx, trackID, videoID, title, artist, album, durationSec, false)
}

func FetchLyricsWithVideoOpts(ctx context.Context, trackID int64, videoID, title, artist, album string, durationSec int, forceCaptionAlign bool) TrackLyrics {
	out := TrackLyrics{
		TrackID:      trackID,
		Title:        title,
		Artist:       artist,
		SyncedLyrics: []LyricLine{},
		Source:       "lrclib",
	}

	cleanTitle := strings.TrimSpace(title)
	cleanArtist := cleanLyricArtist(artist)
	if strings.EqualFold(cleanArtist, "Unknown Artist") {
		cleanArtist = ""
	}
	if cleanTitle == "" {
		return out
	}

	reqCtx, cancel := context.WithTimeout(ctx, 9*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 3500 * time.Millisecond}

	type sbResult struct {
		intervals []SponsorBlockInterval
		ok        bool
	}
	sbCh := make(chan sbResult, 1)
	if strings.TrimSpace(videoID) != "" && durationSec > 0 {
		go func() {
			ivs, ok := FetchSponsorBlockMusicOfftopic(reqCtx, client, videoID)
			sbCh <- sbResult{intervals: ivs, ok: ok}
		}()
	} else {
		sbCh <- sbResult{}
	}

	type capResult struct {
		cueTracks [][]CaptionCue
		ok        bool
	}
	capCh := make(chan capResult, 1)
	if strings.TrimSpace(videoID) != "" && !isHybridVideoID(videoID) {
		go func() {
			cues, ok := FetchYouTubeCaptionCues(reqCtx, client, videoID)
			capCh <- capResult{cueTracks: cues, ok: ok}
		}()
	} else {
		capCh <- capResult{}
	}

	var sbOnce sync.Once
	var sbIntervals []SponsorBlockInterval
	var hasSB bool
	var capOnce sync.Once
	var capCueTracks [][]CaptionCue
	var hasCap bool
	musicDurSec := durationSec
	ensureSB := func() {
		sbOnce.Do(func() {
			res := <-sbCh
			sbIntervals = res.intervals
			hasSB = res.ok
			if hasSB && durationSec > 0 {
				sbTotal := 0.0
				for _, iv := range sbIntervals {
					if iv.Start < float64(durationSec)-2.0 {
						sbTotal += iv.End - iv.Start
					}
				}
				if est := durationSec - int(math.Round(sbTotal)); est >= 30 {
					musicDurSec = est
				}
			}
		})
	}
	ensureCap := func() {
		capOnce.Do(func() {
			res := <-capCh
			capCueTracks = res.cueTracks
			hasCap = res.ok
		})
	}

	alignPopulated := func(res lrclibResponse, srcName string) (TrackLyrics, bool) {
		ensureSB()
		populated := populateLyricsWithSB(out, res, durationSec, sbIntervals, hasSB)
		populated.Source = srcName
		if populated.Source == "instrumental" {
			return populated, true
		}
		rawLines := parseLRC(res.SyncedLyrics)
		durMismatch := durationSec <= 0 || res.Duration <= 0 || math.Abs(float64(durationSec)-res.Duration) >= 1.5
		if len(rawLines) > 0 && (forceCaptionAlign || durMismatch) {
			ensureCap()
			if hasCap {
				if aligned, okCap := AlignLyricsWithCaptions(rawLines, capCueTracks, durationSec); okCap {
					populated.SyncedLyrics = aligned
					populated.NeedsVideoVerify = false
					return populated, true
				}
			}
		}
		return populated, false
	}

	matchesTargetDur := func(lrcDur float64) bool {
		if durationSec <= 0 || lrcDur <= 0 {
			return true
		}
		if math.Abs(lrcDur-float64(durationSec)) <= 4.0 {
			return true
		}
		return hasSB && math.Abs(lrcDur-float64(musicDurSec)) <= 4.5
	}

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

	shortTitle := cleanLyricTitle(cleanTitle)
	shortArtist := primaryLyricArtist(cleanArtist)
	cleanAlbum := cleanLyricAlbum(album, cleanTitle, shortTitle)

	params := url.Values{}
	params.Set("track_name", cleanTitle)
	if cleanArtist != "" {
		params.Set("artist_name", cleanArtist)
	}
	if cleanAlbum != "" {
		params.Set("album_name", cleanAlbum)
	}
	if durationSec > 0 {
		params.Set("duration", strconv.Itoa(durationSec))
	}
	if res, ok := queryLRCLIBGet(reqCtx, client, "https://lrclib.net/api/get?"+params.Encode()); ok {
		populated, capAligned := alignPopulated(res, "lrclib")
		if populated.Source == "instrumental" || (len(populated.SyncedLyrics) > 0 && (capAligned || matchesTargetDur(res.Duration))) {
			return populated
		}
		saveSynced(populated)
		savePlain(populated)
	}

	ensureSB()

	// Build unique LRCLIB search endpoints (structured track+artist first, then free-text q variants).
	searchURLs := make([]string, 0, 4)
	if shortTitle != "" && shortArtist != "" {
		sp := url.Values{}
		sp.Set("track_name", shortTitle)
		sp.Set("artist_name", shortArtist)
		searchURLs = append(searchURLs, "https://lrclib.net/api/search?"+sp.Encode())
	}
	seenQ := map[string]struct{}{}
	for _, sq := range []string{
		strings.TrimSpace(cleanArtist + " " + cleanTitle),
		strings.TrimSpace(shortArtist + " " + shortTitle),
	} {
		if sq == "" {
			continue
		}
		if _, exists := seenQ[sq]; exists {
			continue
		}
		seenQ[sq] = struct{}{}
		searchURLs = append(searchURLs, "https://lrclib.net/api/search?q="+url.QueryEscape(sq))
	}

	for _, searchURL := range searchURLs {
		if list, ok := queryLRCLIBSearch(reqCtx, client, searchURL); ok && len(list) > 0 {
			best := pickBestLRCLIB(list, cleanTitle, cleanArtist, musicDurSec)
			populated, capAligned := alignPopulated(best, "lrclib-search")
			if populated.Source == "instrumental" || (len(populated.SyncedLyrics) > 0 && (capAligned || matchesTargetDur(best.Duration))) {
				return populated
			}
			saveSynced(populated)
			savePlain(populated)
		}
	}

	if qq, ok := queryQQMusicLyrics(reqCtx, client, out, shortTitle, shortArtist, musicDurSec); ok {
		if len(qq.SyncedLyrics) > 0 {
			ensureCap()
			if hasCap {
				if aligned, okCap := AlignLyricsWithCaptions(qq.SyncedLyrics, capCueTracks, durationSec); okCap {
					qq.SyncedLyrics = aligned
					qq.NeedsVideoVerify = false
					return qq
				}
			}
			aligned, heuristic := AlignLyricsTimeline(qq.SyncedLyrics, qq.MatchedDurationSec, durationSec, sbIntervals, hasSB)
			qq.SyncedLyrics = aligned
			qq.NeedsVideoVerify = heuristic
			return qq
		}
		savePlain(qq)
	}

	if ne, ok := queryNetEaseLyrics(reqCtx, client, out, shortTitle, shortArtist, musicDurSec); ok {
		if len(ne.SyncedLyrics) > 0 {
			ensureCap()
			if hasCap {
				if aligned, okCap := AlignLyricsWithCaptions(ne.SyncedLyrics, capCueTracks, durationSec); okCap {
					ne.SyncedLyrics = aligned
					ne.NeedsVideoVerify = false
					return ne
				}
			}
			aligned, heuristic := AlignLyricsTimeline(ne.SyncedLyrics, ne.MatchedDurationSec, durationSec, sbIntervals, hasSB)
			ne.SyncedLyrics = aligned
			ne.NeedsVideoVerify = heuristic
			return ne
		}
		savePlain(ne)
	}

	if kg, ok := queryKuGouLyrics(reqCtx, client, out, shortTitle, shortArtist, musicDurSec); ok {
		if len(kg.SyncedLyrics) > 0 {
			ensureCap()
			if hasCap {
				if aligned, okCap := AlignLyricsWithCaptions(kg.SyncedLyrics, capCueTracks, durationSec); okCap {
					kg.SyncedLyrics = aligned
					kg.NeedsVideoVerify = false
					return kg
				}
			}
			aligned, heuristic := AlignLyricsTimeline(kg.SyncedLyrics, kg.MatchedDurationSec, durationSec, sbIntervals, hasSB)
			kg.SyncedLyrics = aligned
			kg.NeedsVideoVerify = heuristic
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

func lyricArtistMatches(expectedArtist, candidateArtist string) bool {
	exp := strings.ToLower(strings.TrimSpace(cleanLyricArtist(expectedArtist)))
	cand := strings.ToLower(strings.TrimSpace(cleanLyricArtist(candidateArtist)))
	if exp == "" || cand == "" {
		return true
	}
	if strings.Contains(cand, exp) || strings.Contains(exp, cand) {
		return true
	}
	for _, tok := range artistSplitRegex.Split(exp, -1) {
		t := strings.TrimSpace(tok)
		if len([]rune(t)) >= 3 && strings.Contains(cand, t) {
			return true
		}
	}
	return false
}

func lyricTitleMatches(expectedTitle, candidateTitle string) bool {
	exp := strings.ToLower(strings.TrimSpace(cleanLyricTitle(expectedTitle)))
	cand := strings.ToLower(strings.TrimSpace(cleanLyricTitle(candidateTitle)))
	if exp == "" || cand == "" {
		return true
	}
	return cand == exp || strings.Contains(cand, exp) || strings.Contains(exp, cand)
}

func pickBestLRCLIB(list []lrclibResponse, title, artist string, durationSec int) lrclibResponse {
	best := lrclibResponse{}
	bestScore := -1e9
	lowTitle := strings.ToLower(strings.TrimSpace(cleanLyricTitle(title)))
	lowArtist := strings.ToLower(strings.TrimSpace(primaryLyricArtist(artist)))
	for _, item := range list {
		if lowArtist != "" && strings.TrimSpace(item.ArtistName) != "" && !lyricArtistMatches(artist, item.ArtistName) {
			continue
		}
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

func queryQQMusicLyrics(ctx context.Context, client *http.Client, base TrackLyrics, title, artist string, durationSec int) (TrackLyrics, bool) {
	q := strings.TrimSpace(artist + " " + title)
	if q == "" {
		return base, false
	}
	searchURL := "https://c.y.qq.com/soso/fcgi-bin/client_search_cp?w=" + url.QueryEscape(q) + "&format=json&n=6&p=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return base, false
	}
	req.Header.Set("Referer", "https://y.qq.com/")
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
		Data struct {
			Song struct {
				List []struct {
					SongMID  string `json:"songmid"`
					SongName string `json:"songname"`
					Interval int    `json:"interval"`
					Singer   []struct {
						Name string `json:"name"`
					} `json:"singer"`
				} `json:"list"`
			} `json:"song"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &sr) != nil || len(sr.Data.Song.List) == 0 {
		return base, false
	}

	lowTitle := strings.ToLower(strings.TrimSpace(title))
	lowArtist := strings.ToLower(strings.TrimSpace(artist))
	candidates := sr.Data.Song.List
	sort.SliceStable(candidates, func(i, j int) bool {
		score := func(idx int) float64 {
			c := candidates[idx]
			sc := 0.0
			if durationSec > 0 && c.Interval > 0 {
				sc -= math.Abs(float64(c.Interval-durationSec)) * 20.0
			}
			cn := strings.ToLower(strings.TrimSpace(cleanLyricTitle(c.SongName)))
			if lowTitle != "" && cn != "" {
				if cn == lowTitle {
					sc += 120.0
				} else if strings.Contains(cn, lowTitle) || strings.Contains(lowTitle, cn) {
					sc += 50.0
				}
			}
			if lowArtist != "" {
				for _, sg := range c.Singer {
					if strings.Contains(strings.ToLower(sg.Name), lowArtist) {
						sc += 60.0
						break
					}
				}
			}
			return sc
		}
		return score(i) > score(j)
	})

	for _, c := range candidates {
		if strings.TrimSpace(c.SongMID) == "" {
			continue
		}
		if !lyricTitleMatches(title, c.SongName) {
			continue
		}
		if lowArtist != "" && len(c.Singer) > 0 {
			singerMatched := false
			for _, sg := range c.Singer {
				if lyricArtistMatches(artist, sg.Name) {
					singerMatched = true
					break
				}
			}
			if !singerMatched {
				continue
			}
		}
		if durationSec > 0 && c.Interval > 0 && math.Abs(float64(c.Interval-durationSec)) > 18.0 {
			continue
		}
		lyricURL := fmt.Sprintf("https://c.y.qq.com/lyric/fcgi-bin/fcg_query_lyric_new.fcg?songmid=%s&format=json&nobase64=1", url.QueryEscape(c.SongMID))
		lreq, err := http.NewRequestWithContext(ctx, http.MethodGet, lyricURL, nil)
		if err != nil {
			continue
		}
		lreq.Header.Set("Referer", "https://y.qq.com/portal/player.html")
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
			Retcode int    `json:"retcode"`
			Lyric   string `json:"lyric"`
		}
		if json.Unmarshal(lraw, &lr) != nil || lr.Retcode != 0 || strings.TrimSpace(lr.Lyric) == "" {
			continue
		}
		decoded := strings.ReplaceAll(html.UnescapeString(lr.Lyric), "\r\n", "\n")
		parsed := parseLRC(decoded)
		if len(parsed) > 0 {
			base.Source = "qqmusic"
			base.MatchedDurationSec = float64(c.Interval)
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
				ID       int64  `json:"id"`
				Name     string `json:"name"`
				Duration int    `json:"duration"`
				Artists  []struct {
					Name string `json:"name"`
				} `json:"artists"`
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
		if !lyricTitleMatches(title, s.Name) {
			continue
		}
		if strings.TrimSpace(artist) != "" && len(s.Artists) > 0 {
			artOk := false
			for _, a := range s.Artists {
				if lyricArtistMatches(artist, a.Name) {
					artOk = true
					break
				}
			}
			if !artOk {
				continue
			}
		}
		lyricURL := fmt.Sprintf("https://music.163.com/api/song/lyric?id=%d&lv=1&yv=1&tv=-1", s.ID)
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
			YRC struct {
				Lyric string `json:"lyric"`
			} `json:"yrc"`
		}
		if json.Unmarshal(lraw, &lr) != nil {
			continue
		}
		var parsed []LyricLine
		if strings.TrimSpace(lr.YRC.Lyric) != "" {
			parsed = parseNetEaseYRC(lr.YRC.Lyric)
		}
		if len(parsed) == 0 && strings.TrimSpace(lr.LRC.Lyric) != "" {
			parsed = parseLRC(lr.LRC.Lyric)
		}
		if len(parsed) > 0 {
			base.Source = "netease"
			base.MatchedDurationSec = float64(s.Duration) / 1000.0
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
		parsed := parseLRC(string(decoded))
		if len(parsed) > 0 {
			base.Source = "kugou"
			base.MatchedDurationSec = float64(c.Duration) / 1000.0
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
	return populateLyricsWithSB(out, res, trackDurationSec, nil, false)
}

func populateLyricsWithSB(out TrackLyrics, res lrclibResponse, trackDurationSec int, sbIntervals []SponsorBlockInterval, hasSB bool) TrackLyrics {
	if res.Instrumental && strings.TrimSpace(res.PlainLyrics) == "" && strings.TrimSpace(res.SyncedLyrics) == "" {
		out.Source = "instrumental"
		out.PlainLyrics = "[Musik Instrumental]"
		return out
	}
	out.MatchedDurationSec = res.Duration
	if strings.TrimSpace(res.SyncedLyrics) != "" {
		aligned, heuristic := AlignLyricsTimeline(parseLRC(res.SyncedLyrics), res.Duration, trackDurationSec, sbIntervals, hasSB)
		out.SyncedLyrics = aligned
		out.NeedsVideoVerify = heuristic
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
