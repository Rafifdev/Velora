package stats

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"velora/internal/db"
)

type HourBucket struct {
	Hour    int     `json:"hour"`
	Label   string  `json:"label"`
	Minutes float64 `json:"minutes"`
	Plays   int     `json:"plays"`
}

type DayBucket struct {
	DayIndex int     `json:"dayIndex"`
	Label    string  `json:"label"`
	FullName string  `json:"fullName"`
	Minutes  float64 `json:"minutes"`
	Plays    int     `json:"plays"`
}

type TopStatItem struct {
	Name     string  `json:"name"`
	SubText  string  `json:"subText"`
	ImageURL string  `json:"imageUrl"`
	Plays    int     `json:"plays"`
	Minutes  float64 `json:"minutes"`
}

type ListeningStats struct {
	TimeOfDay     []HourBucket  `json:"timeOfDay"`
	DayOfWeek     []DayBucket   `json:"dayOfWeek"`
	TotalMinutes  float64       `json:"totalMinutes"`
	TotalPlays    int           `json:"totalPlays"`
	UniqueTracks  int           `json:"uniqueTracks"`
	PeakHourLabel string        `json:"peakHourLabel"`
	PeakPersona   string        `json:"peakPersona"`
	MostActiveDay string        `json:"mostActiveDay"`
	TopTracks     []TopStatItem `json:"topTracks"`
	TopArtists    []TopStatItem `json:"topArtists"`
}

type FireStreakStatus struct {
	CurrentStreak   int                 `json:"currentStreak"`
	LongestStreak   int                 `json:"longestStreak"`
	TodaySeconds    int                 `json:"todaySeconds"`
	TodayMinutes    float64             `json:"todayMinutes"`
	TodayTracks     int                 `json:"todayTracks"`
	GoalMinutes     int                 `json:"goalMinutes"`
	GoalTracks      int                 `json:"goalTracks"`
	StreakMetToday  bool                `json:"streakMetToday"`
	ProgressPercent int                 `json:"progressPercent"`
	Last30Days      []db.DailyStreakRow `json:"last30Days"`
}

type Service struct {
	store *db.Store
}

func NewService(store *db.Store) *Service {
	return &Service{store: store}
}

func (s *Service) getGoalTargets() (int, int) {
	goalMinutes := 10
	goalTracks := 3
	prefs, err := s.store.GetPreferences()
	if err == nil {
		if v, ok := prefs["streak_goal_minutes"]; ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				goalMinutes = n
			}
		}
		if v, ok := prefs["streak_goal_tracks"]; ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				goalTracks = n
			}
		}
	}
	return goalMinutes, goalTracks
}

func (s *Service) RecordListen(trackID int64, secondsListened int, completed bool, playedAt time.Time) error {
	if trackID <= 0 || secondsListened <= 0 {
		return nil
	}
	if playedAt.IsZero() {
		playedAt = time.Now()
	}

	compInt := 0
	if completed {
		compInt = 1
	}
	ts := playedAt.Format("2006-01-02 15:04:05")
	if _, err := s.store.DB().Exec(
		`INSERT INTO play_history (track_id, played_at, duration_listened_sec, completed) VALUES (?, ?, ?, ?)`,
		trackID, ts, secondsListened, compInt,
	); err != nil {
		return err
	}

	_, _ = s.store.DB().Exec(`UPDATE tracks SET play_count = play_count + 1 WHERE id = ?`, trackID)

	dateStr := playedAt.Format("2006-01-02")
	var curSec, curTracks int
	err := s.store.DB().QueryRow(`SELECT seconds_listened, tracks_count FROM daily_streaks WHERE date_str = ?`, dateStr).Scan(&curSec, &curTracks)
	if err != nil {
		curSec = 0
		curTracks = 0
	}
	newSec := curSec + secondsListened
	newTracks := curTracks + 1

	goalMin, goalTracks := s.getGoalTargets()
	met := 0
	if newSec >= goalMin*60 || newTracks >= goalTracks {
		met = 1
	}

	_, err = s.store.DB().Exec(`
		INSERT INTO daily_streaks (date_str, seconds_listened, tracks_count, streak_met)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(date_str) DO UPDATE SET
			seconds_listened = excluded.seconds_listened,
			tracks_count = excluded.tracks_count,
			streak_met = MAX(daily_streaks.streak_met, excluded.streak_met)`,
		dateStr, newSec, newTracks, met,
	)
	return err
}

func (s *Service) GetFireStreak(now time.Time) (FireStreakStatus, error) {
	if now.IsZero() {
		now = time.Now()
	}
	goalMin, goalTracks := s.getGoalTargets()

	rows, err := s.store.DB().Query(`SELECT date_str, seconds_listened, tracks_count, streak_met FROM daily_streaks ORDER BY date_str ASC`)
	if err != nil {
		return FireStreakStatus{}, err
	}
	defer rows.Close()

	byDate := map[string]db.DailyStreakRow{}
	var metDates []time.Time
	for rows.Next() {
		var r db.DailyStreakRow
		var met int
		if err := rows.Scan(&r.DateStr, &r.SecondsListened, &r.TracksCount, &met); err != nil {
			return FireStreakStatus{}, err
		}
		r.StreakMet = met == 1
		byDate[r.DateStr] = r
		if r.StreakMet {
			if dt, err := time.ParseInLocation("2006-01-02", r.DateStr, now.Location()); err == nil {
				metDates = append(metDates, dt)
			}
		}
	}

	todayKey := now.Format("2006-01-02")
	yesterdayKey := now.AddDate(0, 0, -1).Format("2006-01-02")

	longest := 0
	run := 0
	var prev time.Time
	for i, dt := range metDates {
		if i == 0 {
			run = 1
		} else {
			diffDays := int(math.Round(dt.Sub(prev).Hours() / 24))
			if diffDays == 1 {
				run++
			} else if diffDays > 1 {
				run = 1
			}
		}
		if run > longest {
			longest = run
		}
		prev = dt
	}

	current := 0
	cursor := now
	if r, ok := byDate[todayKey]; ok && r.StreakMet {
		cursor = now
	} else if r, ok := byDate[yesterdayKey]; ok && r.StreakMet {
		cursor = now.AddDate(0, 0, -1)
	} else {
		cursor = time.Time{}
	}

	if !cursor.IsZero() {
		for {
			k := cursor.Format("2006-01-02")
			if r, ok := byDate[k]; ok && r.StreakMet {
				current++
				cursor = cursor.AddDate(0, 0, -1)
			} else {
				break
			}
		}
	}

	todayRow := byDate[todayKey]
	todayMin := math.Round((float64(todayRow.SecondsListened)/60.0)*10) / 10
	pctSec := 0
	if goalMin > 0 {
		pctSec = int(math.Round((float64(todayRow.SecondsListened) / float64(goalMin*60)) * 100))
	}
	pctTrk := 0
	if goalTracks > 0 {
		pctTrk = int(math.Round((float64(todayRow.TracksCount) / float64(goalTracks)) * 100))
	}
	pct := pctSec
	if pctTrk > pct {
		pct = pctTrk
	}
	if pct > 100 || todayRow.StreakMet {
		pct = 100
	}

	last30 := make([]db.DailyStreakRow, 0, 30)
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		if r, ok := byDate[d]; ok {
			last30 = append(last30, r)
		} else {
			last30 = append(last30, db.DailyStreakRow{DateStr: d})
		}
	}

	return FireStreakStatus{
		CurrentStreak:   current,
		LongestStreak:   longest,
		TodaySeconds:    todayRow.SecondsListened,
		TodayMinutes:    todayMin,
		TodayTracks:     todayRow.TracksCount,
		GoalMinutes:     goalMin,
		GoalTracks:      goalTracks,
		StreakMetToday:  todayRow.StreakMet,
		ProgressPercent: pct,
		Last30Days:      last30,
	}, nil
}

func (s *Service) GetMonthlyStreak(year int, month int, now time.Time) ([]db.DailyStreakRow, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if year <= 0 || month < 1 || month > 12 {
		year = now.Year()
		month = int(now.Month())
	}
	if year > now.Year() || (year == now.Year() && month > int(now.Month())) {
		year = now.Year()
		month = int(now.Month())
	}

	prefix := fmt.Sprintf("%04d-%02d-", year, month)
	rows, err := s.store.DB().Query(
		`SELECT date_str, seconds_listened, tracks_count, streak_met FROM daily_streaks WHERE date_str LIKE ? ORDER BY date_str ASC`,
		prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byDate := map[string]db.DailyStreakRow{}
	for rows.Next() {
		var r db.DailyStreakRow
		var met int
		if err := rows.Scan(&r.DateStr, &r.SecondsListened, &r.TracksCount, &met); err != nil {
			return nil, err
		}
		r.StreakMet = met == 1
		byDate[r.DateStr] = r
	}

	firstOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, now.Location())
	daysInMonth := firstOfMonth.AddDate(0, 1, -1).Day()
	out := make([]db.DailyStreakRow, 0, daysInMonth)
	for day := 1; day <= daysInMonth; day++ {
		key := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		if row, ok := byDate[key]; ok {
			out = append(out, row)
		} else {
			out = append(out, db.DailyStreakRow{DateStr: key})
		}
	}
	return out, nil
}

func (s *Service) GetListeningStats() (ListeningStats, error) {
	hours := make([]HourBucket, 24)
	for h := 0; h < 24; h++ {
		hours[h] = HourBucket{
			Hour:  h,
			Label: fmt.Sprintf("%02d:00", h),
		}
	}

	dayNames := []struct {
		Short string
		Full  string
	}{
		{"Sen", "Senin"},
		{"Sel", "Selasa"},
		{"Rab", "Rabu"},
		{"Kam", "Kamis"},
		{"Jum", "Jumat"},
		{"Sab", "Sabtu"},
		{"Min", "Minggu"},
	}
	days := make([]DayBucket, 7)
	for d := 0; d < 7; d++ {
		days[d] = DayBucket{
			DayIndex: d,
			Label:    dayNames[d].Short,
			FullName: dayNames[d].Full,
		}
	}

	rows, err := s.store.DB().Query(`SELECT track_id, played_at, duration_listened_sec FROM play_history`)
	if err != nil {
		return ListeningStats{}, err
	}
	defer rows.Close()

	uniqueTracks := map[int64]struct{}{}
	totalSec := 0
	totalPlays := 0

	for rows.Next() {
		var trackID int64
		var playedAtStr string
		var durSec int
		if err := rows.Scan(&trackID, &playedAtStr, &durSec); err != nil {
			return ListeningStats{}, err
		}
		uniqueTracks[trackID] = struct{}{}
		totalSec += durSec
		totalPlays++

		dt, err := parseTimestamp(playedAtStr)
		if err != nil {
			continue
		}
		h := dt.Hour()
		hours[h].Plays++
		hours[h].Minutes += float64(durSec) / 60.0

		// Map Sunday=0..Saturday=6 to Monday=0..Sunday=6
		wd := (int(dt.Weekday()) + 6) % 7
		days[wd].Plays++
		days[wd].Minutes += float64(durSec) / 60.0
	}

	peakHour := 0
	maxHourMin := -1.0
	for i := range hours {
		hours[i].Minutes = math.Round(hours[i].Minutes*10) / 10
		if hours[i].Minutes > maxHourMin || (hours[i].Minutes == maxHourMin && hours[i].Plays > hours[peakHour].Plays) {
			maxHourMin = hours[i].Minutes
			peakHour = i
		}
	}

	peakDay := 0
	maxDayMin := -1.0
	for i := range days {
		days[i].Minutes = math.Round(days[i].Minutes*10) / 10
		if days[i].Minutes > maxDayMin || (days[i].Minutes == maxDayMin && days[i].Plays > days[peakDay].Plays) {
			maxDayMin = days[i].Minutes
			peakDay = i
		}
	}

	peakLabel := hours[peakHour].Label
	mostActiveDay := days[peakDay].FullName
	if totalPlays == 0 {
		peakLabel = "-"
		mostActiveDay = "-"
	}

	persona := "Belum Ada Data"
	if totalPlays > 0 {
		switch {
		case peakHour >= 5 && peakHour < 12:
			persona = "Morning Listener"
		case peakHour >= 12 && peakHour < 17:
			persona = "Afternoon Cruiser"
		case peakHour >= 17 && peakHour < 22:
			persona = "Evening Vibes"
		default:
			persona = "Night Owl"
		}
	}

	topTracks := []TopStatItem{}
	tRows, err := s.store.DB().Query(`
		SELECT t.title, t.artist_name, COALESCE(t.cover_url, ''), COUNT(h.id) AS plays, COALESCE(SUM(h.duration_listened_sec), 0) AS secs
		FROM play_history h
		JOIN tracks t ON t.id = h.track_id
		GROUP BY h.track_id
		ORDER BY plays DESC, secs DESC
		LIMIT 5`)
	if err == nil {
		defer tRows.Close()
		for tRows.Next() {
			var it TopStatItem
			var secs int
			if err := tRows.Scan(&it.Name, &it.SubText, &it.ImageURL, &it.Plays, &secs); err == nil {
				it.Minutes = math.Round((float64(secs)/60.0)*10) / 10
				topTracks = append(topTracks, it)
			}
		}
	}

	topArtists := []TopStatItem{}
	aRows, err := s.store.DB().Query(`
		SELECT t.artist_name, COALESCE(MAX(NULLIF(a.avatar_url, '')), MAX(NULLIF(t.cover_url, '')), ''), COUNT(h.id) AS plays, COALESCE(SUM(h.duration_listened_sec), 0) AS secs
		FROM play_history h
		JOIN tracks t ON t.id = h.track_id
		LEFT JOIN artists a ON a.id = t.artist_id
		GROUP BY t.artist_name
		ORDER BY plays DESC, secs DESC
		LIMIT 5`)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var it TopStatItem
			var secs int
			if err := aRows.Scan(&it.Name, &it.ImageURL, &it.Plays, &secs); err == nil {
				it.SubText = fmt.Sprintf("%d putar", it.Plays)
				it.Minutes = math.Round((float64(secs)/60.0)*10) / 10
				topArtists = append(topArtists, it)
			}
		}
	}

	return ListeningStats{
		TimeOfDay:     hours,
		DayOfWeek:     days,
		TotalMinutes:  math.Round((float64(totalSec)/60.0)*10) / 10,
		TotalPlays:    totalPlays,
		UniqueTracks:  len(uniqueTracks),
		PeakHourLabel: peakLabel,
		PeakPersona:   persona,
		MostActiveDay: mostActiveDay,
		TopTracks:     topTracks,
		TopArtists:    topArtists,
	}, nil
}

func parseTimestamp(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if dt, err := time.ParseInLocation(layout, strings.TrimSpace(s), time.Local); err == nil {
			return dt, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", s)
}
