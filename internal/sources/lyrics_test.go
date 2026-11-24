package sources

import (
	"testing"
)

func TestParseLRC(t *testing.T) {
	sample := `
[00:12.50] Midnight city lights
[01:05.2] Waiting in the dark
[02:10]
Invalid line ignored
`
	lines := parseLRC(sample)
	if len(lines) != 3 {
		t.Fatalf("expected 3 parsed lines, got %d", len(lines))
	}
	if lines[0].TimeSec != 12.5 || lines[0].Text != "Midnight city lights" {
		t.Fatalf("unexpected line 0: %+v", lines[0])
	}
	if lines[1].TimeSec != 65.2 || lines[1].Text != "Waiting in the dark" {
		t.Fatalf("unexpected line 1: %+v", lines[1])
	}
	if lines[2].TimeSec != 130.0 || lines[2].Text != "♪" {
		t.Fatalf("unexpected line 2: %+v", lines[2])
	}
}

func TestParseLRCIntroCreditsAndOffset(t *testing.T) {
	sample := `
[offset:1500]
[00:00.00]
[00:01.20] 作词 : Test Writer
[00:02.50] Lyrics by Test Artist
[00:24.00] First vocal line
[00:30.00] Second vocal line
`
	lines := parseLRC(sample)
	if len(lines) != 2 {
		t.Fatalf("expected 2 vocal lines after filtering intro credits/empty, got %d: %+v", len(lines), lines)
	}
	if lines[0].TimeSec != 25.5 || lines[0].Text != "First vocal line" {
		t.Fatalf("expected line 0 shifted by offset +1.5s to 25.5, got %+v", lines[0])
	}
	if lines[1].TimeSec != 31.5 || lines[1].Text != "Second vocal line" {
		t.Fatalf("expected line 1 at 31.5, got %+v", lines[1])
	}
}

func TestAlignIntroOffset(t *testing.T) {
	lines := []LyricLine{
		{TimeSec: 2.0, Text: "Start"},
		{TimeSec: 180.0, Text: "End"},
	}
	aligned := alignIntroOffset(lines, 190.0, 205)
	if aligned[0].TimeSec != 17.0 || aligned[1].TimeSec != 195.0 {
		t.Fatalf("expected +15s intro alignment, got %+v", aligned)
	}

	mvLines := []LyricLine{
		{TimeSec: 5.04, Text: "Intro"},
		{TimeSec: 271.33, Text: "Outro"},
	}
	alignedMV := alignIntroOffset(mvLines, 278.0, 339)
	if alignedMV[0].TimeSec != 66.04 || alignedMV[1].TimeSec != 332.33 {
		t.Fatalf("expected +61s MV intro alignment, got %+v", alignedMV)
	}
}

func TestCleanLyricMetadataAndTrackPicker(t *testing.T) {
	if got := cleanLyricTitle("Bohemian Rhapsody (Official Music Video) - Remastered 2011"); got != "Bohemian Rhapsody" {
		t.Fatalf("unexpected cleanLyricTitle: %q", got)
	}
	if got := cleanLyricArtist("NIKI - Topic"); got != "NIKI" {
		t.Fatalf("unexpected cleanLyricArtist: %q", got)
	}
	if got := primaryLyricArtist("Tulus, NIKI"); got != "Tulus" {
		t.Fatalf("unexpected primaryLyricArtist: %q", got)
	}
	if got := cleanLyricAlbum("Imported from Spotify", "Lowkey", "Lowkey"); got != "" {
		t.Fatalf("expected placeholder album cleared, got %q", got)
	}

	hits := []YTMusicTrack{
		{VideoID: "vidMV", Title: "lowkey (Official Video)", Artist: "NIKI", Album: "YouTube Music", DurationSec: 215},
		{VideoID: "vidATV", Title: "lowkey", Artist: "NIKI", Album: "lowkey", DurationSec: 171},
	}
	best, ok := PickBestYTMusicTrack(hits, "lowkey", "NIKI", 171)
	if !ok || best.VideoID != "vidATV" {
		t.Fatalf("expected studio ATV hit vidATV, got ok=%v hit=%+v", ok, best)
	}
}

func TestSponsorBlockPiecewiseAndOutroProtection(t *testing.T) {
	// MV with intro [0, 38.89], mid-song skit [201.26, 233.21], and outro [320.78, 330.52]
	intervals := []SponsorBlockInterval{
		{Start: 0, End: 38.89},
		{Start: 201.26, End: 233.21},
		{Start: 320.78, End: 330.52},
	}
	lines := []LyricLine{
		{TimeSec: 0.50, Text: "Pre-skit line"},
		{TimeSec: 163.00, Text: "Post-skit line"},
	}
	aligned, heuristic := AlignLyricsTimeline(lines, 207.0, 331, intervals, true)
	if heuristic {
		t.Fatal("expected verified SponsorBlock alignment, not heuristic")
	}
	if aligned[0].TimeSec < 39.38 || aligned[0].TimeSec > 39.40 {
		t.Fatalf("expected pre-skit line ~39.39s, got %.2f", aligned[0].TimeSec)
	}
	// 163.00 + 38.89 = 201.89 (>= 201.26 - 0.35), so + 31.95 = 233.84
	if aligned[1].TimeSec < 233.83 || aligned[1].TimeSec > 233.85 {
		t.Fatalf("expected post-skit line ~233.84s, got %.2f", aligned[1].TimeSec)
	}

	// Outro-only MV [184.0, 205.0] must NOT shift lyrics at the beginning
	outroOnly := []SponsorBlockInterval{{Start: 184.0, End: 205.0}}
	mamaLines := []LyricLine{
		{TimeSec: 9.76, Text: "Where should we run to?"},
		{TimeSec: 176.0, Text: "Outro vocal"},
	}
	alignedOutro, _ := AlignLyricsTimeline(mamaLines, 184.0, 205, outroOnly, true)
	if alignedOutro[0].TimeSec != 9.76 {
		t.Fatalf("expected outro-only MV to leave first line at 9.76s, got %.2f", alignedOutro[0].TimeSec)
	}

	// Unknown trackDurationSec (0) must never shift
	unknownDur, _ := AlignLyricsTimeline(mamaLines, 184.0, 0, nil, false)
	if unknownDur[0].TimeSec != 9.76 {
		t.Fatalf("expected unknown duration to leave first line at 9.76s, got %.2f", unknownDur[0].TimeSec)
	}
}

func TestParseYTMusicSubtitleAndATVPreference(t *testing.T) {
	art, alb, dur, _ := ParseYTMusicSubtitle(nil, "Jonas Blue & William Singe \u2022 Mama \u2022 3:05")
	if art != "Jonas Blue & William Singe" || alb != "Mama" || dur != 185 {
		t.Fatalf("unexpected ParseYTMusicSubtitle result: art=%q alb=%q dur=%d", art, alb, dur)
	}

	hits := []YTMusicTrack{
		{VideoID: "vidOMV", Title: "Mama", Artist: "Jonas Blue", Album: "Mama", DurationSec: 184, VideoType: "MUSIC_VIDEO_TYPE_OMV"},
		{VideoID: "vidATV", Title: "Mama", Artist: "Jonas Blue", Album: "Mama", DurationSec: 184, VideoType: "MUSIC_VIDEO_TYPE_ATV"},
	}
	best, ok := PickBestYTMusicTrack(hits, "Mama", "Jonas Blue", 184)
	if !ok || best.VideoID != "vidATV" {
		t.Fatalf("expected ATV hit vidATV over OMV, got ok=%v hit=%+v", ok, best)
	}
}
