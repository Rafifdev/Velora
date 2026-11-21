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
}
