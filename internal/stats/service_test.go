package stats

import (
	"path/filepath"
	"testing"
	"time"

	"velora/internal/db"
)

func TestFireStreakAndListeningStats(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "stats_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	_ = store.SetPreference("streak_goal_minutes", "10")
	_ = store.SetPreference("streak_goal_tracks", "3")

	tr1, err := store.UpsertTrack(db.Track{
		Title:       "Midnight City",
		ArtistName:  "M83",
		AlbumTitle:  "Hurry Up",
		DurationSec: 240,
		Format:      "FLAC",
		SourceType:  "local",
		FilePath:    "C:/music/midnight.flac",
	})
	if err != nil {
		t.Fatalf("upsert track: %v", err)
	}

	svc := NewService(store)
	now := time.Date(2026, 11, 14, 21, 30, 0, 0, time.Local)

	// Day -4 (broken gap before day -2)
	if err := svc.RecordListen(tr1.ID, 660, true, now.AddDate(0, 0, -4)); err != nil {
		t.Fatalf("record day-4: %v", err)
	}
	// Day -2: 3 tracks -> meets track goal (3)
	for i := 0; i < 3; i++ {
		if err := svc.RecordListen(tr1.ID, 60, true, now.AddDate(0, 0, -2)); err != nil {
			t.Fatalf("record day-2: %v", err)
		}
	}
	// Day -1: 650s -> meets minute goal (10m = 600s)
	if err := svc.RecordListen(tr1.ID, 650, true, now.AddDate(0, 0, -1)); err != nil {
		t.Fatalf("record day-1: %v", err)
	}
	// Today: 620s -> meets minute goal -> current streak should be 3, longest should be 3
	if err := svc.RecordListen(tr1.ID, 620, true, now); err != nil {
		t.Fatalf("record today: %v", err)
	}

	streak, err := svc.GetFireStreak(now)
	if err != nil {
		t.Fatalf("get fire streak: %v", err)
	}
	if streak.CurrentStreak != 3 || streak.LongestStreak != 3 || !streak.StreakMetToday {
		t.Fatalf("unexpected streak status: %+v", streak)
	}
	if len(streak.Last30Days) != 30 {
		t.Fatalf("expected 30 calendar days, got %d", len(streak.Last30Days))
	}

	// Test ListeningStats aggregation
	ls, err := svc.GetListeningStats()
	if err != nil {
		t.Fatalf("get listening stats: %v", err)
	}
	if len(ls.TimeOfDay) != 24 || len(ls.DayOfWeek) != 7 {
		t.Fatalf("unexpected buckets: tod=%d dow=%d", len(ls.TimeOfDay), len(ls.DayOfWeek))
	}
	if ls.TimeOfDay[21].Plays != 6 {
		t.Fatalf("expected 6 plays at 21:00, got %d", ls.TimeOfDay[21].Plays)
	}
	if ls.TotalPlays != 6 || ls.UniqueTracks != 1 {
		t.Fatalf("unexpected summary totals: %+v", ls)
	}
	if ls.PeakHourLabel != "21:00" || ls.PeakPersona != "Evening Vibes" {
		t.Fatalf("unexpected peak hour/persona: %s / %s", ls.PeakHourLabel, ls.PeakPersona)
	}
}
