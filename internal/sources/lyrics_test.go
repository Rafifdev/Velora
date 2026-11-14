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
