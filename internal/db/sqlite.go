package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

type Source struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Path         string `json:"path"`
	ConfigJSON   string `json:"configJson"`
	TrackCount   int    `json:"trackCount"`
	Enabled      bool   `json:"enabled"`
	LastSyncedAt string `json:"lastSyncedAt"`
}

type Artist struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	Bio        string `json:"bio"`
	SourceType string `json:"sourceType"`
	ExternalID string `json:"externalId"`
	IsFavorite bool   `json:"isFavorite"`
	TrackCount int    `json:"trackCount"`
}

type Album struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	ArtistID    int64  `json:"artistId"`
	ArtistName  string `json:"artistName"`
	CoverURL    string `json:"coverUrl"`
	ReleaseYear int    `json:"releaseYear"`
	SourceType  string `json:"sourceType"`
	ExternalID  string `json:"externalId"`
	IsFavorite  bool   `json:"isFavorite"`
	TrackCount  int    `json:"trackCount"`
}

type Track struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	ArtistID    int64  `json:"artistId"`
	ArtistName  string `json:"artistName"`
	AlbumID     int64  `json:"albumId"`
	AlbumTitle  string `json:"albumTitle"`
	DurationSec int    `json:"durationSec"`
	Format      string `json:"format"`
	Bitrate     int    `json:"bitrate"`
	SourceType  string `json:"sourceType"`
	FilePath    string `json:"filePath"`
	ExternalID  string `json:"externalId"`
	CoverURL    string `json:"coverUrl"`
	IsFavorite   bool   `json:"isFavorite"`
	PlayCount    int    `json:"playCount"`
	IsDownloaded bool   `json:"isDownloaded"`
	DownloadedAt string `json:"downloadedAt"`
	GlobalPlays  string `json:"globalPlays,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type Playlist struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CoverURL    string `json:"coverUrl"`
	CoverColor  string `json:"coverColor"`
	IsLocal     bool   `json:"isLocal"`
	SourceType  string `json:"sourceType"`
	ExternalID  string `json:"externalId"`
	TrackCount  int    `json:"trackCount"`
	TotalSec    int    `json:"totalSec"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type PlaylistDetail struct {
	Playlist Playlist `json:"playlist"`
	Tracks   []Track  `json:"tracks"`
}

type PlayHistoryItem struct {
	ID                  int64  `json:"id"`
	TrackID             int64  `json:"trackId"`
	Title               string `json:"title"`
	ArtistName          string `json:"artistName"`
	AlbumTitle          string `json:"albumTitle"`
	CoverURL            string `json:"coverUrl"`
	SourceType          string `json:"sourceType"`
	Format              string `json:"format"`
	DurationSec         int    `json:"durationSec"`
	PlayedAt            string `json:"playedAt"`
	DurationListenedSec int    `json:"durationListenedSec"`
	Completed           bool   `json:"completed"`
}

type DailyStreakRow struct {
	DateStr         string `json:"date"`
	SecondsListened int    `json:"secondsListened"`
	TracksCount     int    `json:"tracksCount"`
	StreakMet       bool   `json:"streakMet"`
}

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("pragma sqlite: %w", err)
	}
	if _, err := conn.Exec(schemaSQL); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	_, _ = conn.Exec(`ALTER TABLE tracks ADD COLUMN is_downloaded INTEGER NOT NULL DEFAULT 0;`)
	_, _ = conn.Exec(`ALTER TABLE tracks ADD COLUMN downloaded_at TEXT NOT NULL DEFAULT '';`)
	return &Store{db: conn}, nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) ensureArtist(name, avatarURL, sourceType, externalID string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Unknown Artist"
	}
	var id int64
	err := s.db.QueryRow(`SELECT id FROM artists WHERE lower(name) = lower(?) LIMIT 1`, name).Scan(&id)
	if err == nil {
		if avatarURL != "" {
			_, _ = s.db.Exec(`UPDATE artists SET avatar_url = ? WHERE id = ? AND avatar_url = ''`, avatarURL, id)
		}
		return id, nil
	}
	res, err := s.db.Exec(
		`INSERT INTO artists (name, avatar_url, source_type, external_id) VALUES (?, ?, ?, ?)`,
		name, avatarURL, sourceType, externalID,
	)
	if err != nil {
		err2 := s.db.QueryRow(`SELECT id FROM artists WHERE lower(name) = lower(?) LIMIT 1`, name).Scan(&id)
		if err2 == nil {
			return id, nil
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ensureAlbum(title string, artistID int64, artistName, coverURL string, releaseYear int, sourceType, externalID string) (int64, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Single"
	}
	artistName = strings.TrimSpace(artistName)
	if artistName == "" {
		artistName = "Unknown Artist"
	}
	if releaseYear <= 0 {
		releaseYear = time.Now().Year()
	}
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM albums WHERE lower(title) = lower(?) AND lower(artist_name) = lower(?) LIMIT 1`,
		title, artistName,
	).Scan(&id)
	if err == nil {
		if coverURL != "" {
			_, _ = s.db.Exec(`UPDATE albums SET cover_url = ? WHERE id = ? AND cover_url = ''`, coverURL, id)
		}
		return id, nil
	}
	res, err := s.db.Exec(
		`INSERT INTO albums (title, artist_id, artist_name, cover_url, release_year, source_type, external_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		title, artistID, artistName, coverURL, releaseYear, sourceType, externalID,
	)
	if err != nil {
		err2 := s.db.QueryRow(
			`SELECT id FROM albums WHERE lower(title) = lower(?) AND lower(artist_name) = lower(?) LIMIT 1`,
			title, artistName,
		).Scan(&id)
		if err2 == nil {
			return id, nil
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpsertTrack(t Track) (Track, error) {
	if strings.TrimSpace(t.Title) == "" {
		t.Title = "Untitled Track"
	}
	if strings.TrimSpace(t.ArtistName) == "" {
		t.ArtistName = "Unknown Artist"
	}
	if strings.TrimSpace(t.AlbumTitle) == "" {
		t.AlbumTitle = "Single"
	}
	if strings.TrimSpace(t.SourceType) == "" {
		t.SourceType = "local"
	}
	if strings.TrimSpace(t.Format) == "" {
		t.Format = "MP3"
	}
	if t.DurationSec <= 0 {
		t.DurationSec = 180
	}

	artistID, err := s.ensureArtist(t.ArtistName, t.CoverURL, t.SourceType, t.ExternalID)
	if err != nil {
		return Track{}, err
	}
	albumID, err := s.ensureAlbum(t.AlbumTitle, artistID, t.ArtistName, t.CoverURL, 2024, t.SourceType, t.ExternalID)
	if err != nil {
		return Track{}, err
	}
	t.ArtistID = artistID
	t.AlbumID = albumID

	var existingID int64
	if t.ExternalID != "" {
		_ = s.db.QueryRow(`SELECT id FROM tracks WHERE external_id = ? AND source_type = ? LIMIT 1`, t.ExternalID, t.SourceType).Scan(&existingID)
	} else if t.FilePath != "" {
		_ = s.db.QueryRow(`SELECT id FROM tracks WHERE file_path = ? LIMIT 1`, t.FilePath).Scan(&existingID)
	}

	if existingID > 0 {
		_, err = s.db.Exec(
			`UPDATE tracks SET title = ?, artist_id = ?, artist_name = ?, album_id = ?, album_title = ?, duration_sec = ?, format = ?, bitrate = ?, cover_url = CASE WHEN ? != '' THEN ? ELSE cover_url END WHERE id = ?`,
			t.Title, artistID, t.ArtistName, albumID, t.AlbumTitle, t.DurationSec, strings.ToUpper(t.Format), t.Bitrate, t.CoverURL, t.CoverURL, existingID,
		)
		if err != nil {
			return Track{}, err
		}
		return s.GetTrackByID(existingID)
	}

	favInt := 0
	if t.IsFavorite {
		favInt = 1
	}
	res, err := s.db.Exec(
		`INSERT INTO tracks (title, artist_id, artist_name, album_id, album_title, duration_sec, format, bitrate, source_type, file_path, external_id, cover_url, is_favorite)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Title, artistID, t.ArtistName, albumID, t.AlbumTitle, t.DurationSec, strings.ToUpper(t.Format), t.Bitrate, t.SourceType, t.FilePath, t.ExternalID, t.CoverURL, favInt,
	)
	if err != nil {
		return Track{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Track{}, err
	}
	return s.GetTrackByID(id)
}

func scanTrack(scanner interface{ Scan(...any) error }) (Track, error) {
	var t Track
	var fav, downloaded int
	err := scanner.Scan(
		&t.ID, &t.Title, &t.ArtistID, &t.ArtistName, &t.AlbumID, &t.AlbumTitle,
		&t.DurationSec, &t.Format, &t.Bitrate, &t.SourceType, &t.FilePath, &t.ExternalID,
		&t.CoverURL, &fav, &t.PlayCount, &downloaded, &t.DownloadedAt, &t.CreatedAt,
	)
	t.IsFavorite = fav == 1
	t.IsDownloaded = downloaded == 1
	return t, err
}

func (s *Store) GetTrackByID(id int64) (Track, error) {
	row := s.db.QueryRow(`SELECT id, title, artist_id, artist_name, album_id, album_title, duration_sec, format, bitrate, source_type, file_path, external_id, cover_url, is_favorite, play_count, is_downloaded, downloaded_at, created_at FROM tracks WHERE id = ?`, id)
	return scanTrack(row)
}

func (s *Store) GetTrackByExternalID(externalID string) (Track, error) {
	ext := strings.TrimSpace(externalID)
	if ext == "" {
		return Track{}, fmt.Errorf("empty external id")
	}
	row := s.db.QueryRow(`SELECT id, title, artist_id, artist_name, album_id, album_title, duration_sec, format, bitrate, source_type, file_path, external_id, cover_url, is_favorite, play_count, is_downloaded, downloaded_at, created_at FROM tracks WHERE external_id = ? LIMIT 1`, ext)
	return scanTrack(row)
}

func (s *Store) UpdateTrackExternalID(trackID int64, externalID string) error {
	if trackID <= 0 || strings.TrimSpace(externalID) == "" {
		return nil
	}
	_, err := s.db.Exec(`UPDATE tracks SET external_id = ? WHERE id = ?`, strings.TrimSpace(externalID), trackID)
	return err
}

func (s *Store) GetTracks(filter, query string) ([]Track, error) {
	base := `SELECT id, title, artist_id, artist_name, album_id, album_title, duration_sec, format, bitrate, source_type, file_path, external_id, cover_url, is_favorite, play_count, is_downloaded, downloaded_at, created_at FROM tracks`
	var where []string
	var args []any

	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "local":
		where = append(where, `source_type = 'local'`)
	case "ytmusic":
		where = append(where, `source_type = 'ytmusic'`)
	case "favorites":
		where = append(where, `is_favorite = 1`)
	case "downloaded":
		where = append(where, `is_downloaded = 1`)
	}

	if q := strings.TrimSpace(query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = append(where, `(lower(title) LIKE ? OR lower(artist_name) LIKE ? OR lower(album_title) LIKE ? OR lower(format) LIKE ?)`)
		args = append(args, like, like, like, like)
	}

	if len(where) > 0 {
		base += " WHERE " + strings.Join(where, " AND ")
	}
	base += " ORDER BY id DESC"

	rows, err := s.db.Query(base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (s *Store) DeleteTrack(id int64) error {
	_, _ = s.db.Exec(`DELETE FROM playlist_tracks WHERE track_id = ?`, id)
	_, err := s.db.Exec(`DELETE FROM tracks WHERE id = ?`, id)
	return err
}

func (s *Store) ToggleFavoriteTrack(id int64) (bool, error) {
	var cur int
	if err := s.db.QueryRow(`SELECT is_favorite FROM tracks WHERE id = ?`, id).Scan(&cur); err != nil {
		return false, err
	}
	next := 1
	if cur == 1 {
		next = 0
	}
	if _, err := s.db.Exec(`UPDATE tracks SET is_favorite = ? WHERE id = ?`, next, id); err != nil {
		return false, err
	}
	return next == 1, nil
}

func (s *Store) GetAlbums(favoritesOnly bool) ([]Album, error) {
	query := `
		SELECT a.id, a.title, a.artist_id, a.artist_name, a.cover_url, a.release_year, a.source_type, a.external_id, a.is_favorite,
		       (SELECT COUNT(*) FROM tracks t WHERE t.album_id = a.id) AS track_count
		FROM albums a`
	if favoritesOnly {
		query += ` WHERE a.is_favorite = 1`
	}
	query += ` ORDER BY a.is_favorite DESC, a.id DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	albums := []Album{}
	for rows.Next() {
		var al Album
		var fav int
		if err := rows.Scan(&al.ID, &al.Title, &al.ArtistID, &al.ArtistName, &al.CoverURL, &al.ReleaseYear, &al.SourceType, &al.ExternalID, &fav, &al.TrackCount); err != nil {
			return nil, err
		}
		al.IsFavorite = fav == 1
		albums = append(albums, al)
	}
	return albums, rows.Err()
}

func (s *Store) GetAlbumTracks(albumID int64) ([]Track, error) {
	rows, err := s.db.Query(`SELECT id, title, artist_id, artist_name, album_id, album_title, duration_sec, format, bitrate, source_type, file_path, external_id, cover_url, is_favorite, play_count, is_downloaded, downloaded_at, created_at FROM tracks WHERE album_id = ? ORDER BY id ASC`, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tracks := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (s *Store) ToggleFavoriteAlbum(id int64) (bool, error) {
	var cur int
	if err := s.db.QueryRow(`SELECT is_favorite FROM albums WHERE id = ?`, id).Scan(&cur); err != nil {
		return false, err
	}
	next := 1
	if cur == 1 {
		next = 0
	}
	if _, err := s.db.Exec(`UPDATE albums SET is_favorite = ? WHERE id = ?`, next, id); err != nil {
		return false, err
	}
	return next == 1, nil
}

func (s *Store) ToggleFavoriteYTMusicAlbum(browseID, title, artistName, coverURL string, releaseYear int) (Album, error) {
	browseID = strings.TrimSpace(browseID)
	title = strings.TrimSpace(title)
	artistName = strings.TrimSpace(artistName)
	if title == "" {
		title = "Unknown Album"
	}
	if artistName == "" {
		artistName = "YouTube Music"
	}

	var id int64
	if browseID != "" {
		_ = s.db.QueryRow(
			`SELECT id FROM albums WHERE source_type = 'ytmusic' AND external_id = ? LIMIT 1`,
			browseID,
		).Scan(&id)
	}
	if id == 0 {
		artistID, err := s.ensureArtist(artistName, coverURL, "ytmusic", "")
		if err != nil {
			return Album{}, err
		}
		id, err = s.ensureAlbum(title, artistID, artistName, coverURL, releaseYear, "ytmusic", browseID)
		if err != nil {
			return Album{}, err
		}
	}
	if browseID != "" {
		_, _ = s.db.Exec(`UPDATE albums SET external_id = ? WHERE id = ? AND external_id = ''`, browseID, id)
	}
	if coverURL != "" {
		_, _ = s.db.Exec(`UPDATE albums SET cover_url = ? WHERE id = ? AND cover_url = ''`, coverURL, id)
	}

	if _, err := s.ToggleFavoriteAlbum(id); err != nil {
		return Album{}, err
	}

	var al Album
	var fav int
	err := s.db.QueryRow(`
		SELECT a.id, a.title, a.artist_id, a.artist_name, a.cover_url, a.release_year, a.source_type, a.external_id, a.is_favorite,
		       (SELECT COUNT(*) FROM tracks t WHERE t.album_id = a.id) AS track_count
		FROM albums a WHERE a.id = ?`, id,
	).Scan(&al.ID, &al.Title, &al.ArtistID, &al.ArtistName, &al.CoverURL, &al.ReleaseYear, &al.SourceType, &al.ExternalID, &fav, &al.TrackCount)
	if err != nil {
		return Album{}, err
	}
	al.IsFavorite = fav == 1
	return al, nil
}

func (s *Store) ToggleFavoriteYTMusicArtist(externalID, name, avatarURL string) (Artist, error) {
	externalID = strings.TrimSpace(externalID)
	name = strings.TrimSpace(name)
	avatarURL = strings.TrimSpace(avatarURL)
	if name == "" {
		name = "Unknown Artist"
	}

	var id int64
	if externalID != "" {
		_ = s.db.QueryRow(
			`SELECT id FROM artists WHERE external_id = ? LIMIT 1`,
			externalID,
		).Scan(&id)
	}
	if id == 0 {
		_ = s.db.QueryRow(
			`SELECT id FROM artists WHERE LOWER(TRIM(name)) = LOWER(TRIM(?)) LIMIT 1`,
			name,
		).Scan(&id)
	}
	if id == 0 {
		var err error
		id, err = s.ensureArtist(name, avatarURL, "ytmusic", externalID)
		if err != nil {
			return Artist{}, err
		}
	}
	if externalID != "" {
		_, _ = s.db.Exec(`UPDATE artists SET external_id = ? WHERE id = ? AND external_id = ''`, externalID, id)
	}
	if avatarURL != "" {
		_, _ = s.db.Exec(`UPDATE artists SET avatar_url = ? WHERE id = ? AND avatar_url = ''`, avatarURL, id)
	}

	if _, err := s.ToggleFavoriteArtist(id); err != nil {
		return Artist{}, err
	}

	var ar Artist
	var fav int
	err := s.db.QueryRow(`
		SELECT ar.id, ar.name, ar.avatar_url, ar.bio, ar.source_type, ar.external_id, ar.is_favorite,
		       (SELECT COUNT(*) FROM tracks t WHERE t.artist_id = ar.id) AS track_count
		FROM artists ar WHERE ar.id = ?`, id,
	).Scan(&ar.ID, &ar.Name, &ar.AvatarURL, &ar.Bio, &ar.SourceType, &ar.ExternalID, &fav, &ar.TrackCount)
	if err != nil {
		return Artist{}, err
	}
	ar.IsFavorite = fav == 1
	return ar, nil
}

func (s *Store) GetArtists(favoritesOnly bool) ([]Artist, error) {
	query := `
		SELECT ar.id, ar.name, ar.avatar_url, ar.bio, ar.source_type, ar.external_id, ar.is_favorite,
		       (SELECT COUNT(*) FROM tracks t WHERE t.artist_id = ar.id) AS track_count
		FROM artists ar`
	if favoritesOnly {
		query += ` WHERE ar.is_favorite = 1`
	}
	query += ` ORDER BY ar.is_favorite DESC, ar.id DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artists := []Artist{}
	for rows.Next() {
		var ar Artist
		var fav int
		if err := rows.Scan(&ar.ID, &ar.Name, &ar.AvatarURL, &ar.Bio, &ar.SourceType, &ar.ExternalID, &fav, &ar.TrackCount); err != nil {
			return nil, err
		}
		ar.IsFavorite = fav == 1
		artists = append(artists, ar)
	}
	return artists, rows.Err()
}

func (s *Store) GetArtistTracks(artistID int64) ([]Track, error) {
	rows, err := s.db.Query(`SELECT id, title, artist_id, artist_name, album_id, album_title, duration_sec, format, bitrate, source_type, file_path, external_id, cover_url, is_favorite, play_count, is_downloaded, downloaded_at, created_at FROM tracks WHERE artist_id = ? ORDER BY play_count DESC, id ASC`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tracks := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (s *Store) ToggleFavoriteArtist(id int64) (bool, error) {
	var cur int
	if err := s.db.QueryRow(`SELECT is_favorite FROM artists WHERE id = ?`, id).Scan(&cur); err != nil {
		return false, err
	}
	next := 1
	if cur == 1 {
		next = 0
	}
	if _, err := s.db.Exec(`UPDATE artists SET is_favorite = ? WHERE id = ?`, next, id); err != nil {
		return false, err
	}
	return next == 1, nil
}

func (s *Store) GetPlaylists(localOnly bool) ([]Playlist, error) {
	query := `
		SELECT p.id, p.name, p.description,
		       COALESCE(NULLIF(p.cover_url, ''), (
		         SELECT t2.cover_url
		         FROM playlist_tracks pt2
		         JOIN tracks t2 ON t2.id = pt2.track_id
		         WHERE pt2.playlist_id = p.id AND COALESCE(t2.cover_url, '') != ''
		         ORDER BY pt2.position ASC, pt2.id ASC
		         LIMIT 1
		       ), '') AS cover_url,
		       p.cover_color, p.is_local, p.source_type, p.external_id, p.created_at, p.updated_at,
		       COUNT(pt.id) AS track_count,
		       COALESCE(SUM(t.duration_sec), 0) AS total_sec
		FROM playlists p
		LEFT JOIN playlist_tracks pt ON pt.playlist_id = p.id
		LEFT JOIN tracks t ON t.id = pt.track_id`
	if localOnly {
		query += ` WHERE p.is_local = 1`
	}
	query += ` GROUP BY p.id ORDER BY p.id DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Playlist{}
	for rows.Next() {
		var p Playlist
		var isLocal int
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CoverURL, &p.CoverColor, &isLocal, &p.SourceType, &p.ExternalID, &p.CreatedAt, &p.UpdatedAt, &p.TrackCount, &p.TotalSec); err != nil {
			return nil, err
		}
		p.IsLocal = isLocal == 1
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *Store) GetPlaylistDetail(id int64) (PlaylistDetail, error) {
	var p Playlist
	var isLocal int
	err := s.db.QueryRow(`
		SELECT p.id, p.name, p.description, p.cover_url, p.cover_color, p.is_local, p.source_type, p.external_id, p.created_at, p.updated_at,
		       COUNT(pt.id) AS track_count,
		       COALESCE(SUM(t.duration_sec), 0) AS total_sec
		FROM playlists p
		LEFT JOIN playlist_tracks pt ON pt.playlist_id = p.id
		LEFT JOIN tracks t ON t.id = pt.track_id
		WHERE p.id = ?
		GROUP BY p.id`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.CoverURL, &p.CoverColor, &isLocal, &p.SourceType, &p.ExternalID, &p.CreatedAt, &p.UpdatedAt, &p.TrackCount, &p.TotalSec,
	)
	if err != nil {
		return PlaylistDetail{}, err
	}
	p.IsLocal = isLocal == 1

	rows, err := s.db.Query(`
		SELECT t.id, t.title, t.artist_id, t.artist_name, t.album_id, t.album_title, t.duration_sec, t.format, t.bitrate, t.source_type, t.file_path, t.external_id, t.cover_url, t.is_favorite, t.play_count, t.is_downloaded, t.downloaded_at, COALESCE(pt.added_at, t.created_at)
		FROM playlist_tracks pt
		JOIN tracks t ON t.id = pt.track_id
		WHERE pt.playlist_id = ?
		ORDER BY pt.position ASC, pt.id ASC`, id)
	if err != nil {
		return PlaylistDetail{}, err
	}
	defer rows.Close()

	tracks := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return PlaylistDetail{}, err
		}
		tracks = append(tracks, t)
	}
	return PlaylistDetail{Playlist: p, Tracks: tracks}, rows.Err()
}

func (s *Store) CreatePlaylist(name, description string, isLocal bool, coverColor string) (Playlist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Playlist Baru"
	}
	if strings.TrimSpace(coverColor) == "" {
		coverColor = "#1DB954"
	}
	localInt := 0
	sourceType := "ytmusic"
	if isLocal {
		localInt = 1
		sourceType = "local"
	}
	res, err := s.db.Exec(
		`INSERT INTO playlists (name, description, cover_color, is_local, source_type) VALUES (?, ?, ?, ?, ?)`,
		name, strings.TrimSpace(description), coverColor, localInt, sourceType,
	)
	if err != nil {
		return Playlist{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Playlist{}, err
	}
	detail, err := s.GetPlaylistDetail(id)
	if err != nil {
		return Playlist{}, err
	}
	return detail.Playlist, nil
}

func (s *Store) UpdatePlaylist(id int64, name, description, coverColor string) (Playlist, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Playlist{}, fmt.Errorf("playlist name cannot be empty")
	}
	if strings.TrimSpace(coverColor) == "" {
		coverColor = "#1DB954"
	}
	_, err := s.db.Exec(
		`UPDATE playlists SET name = ?, description = ?, cover_color = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		name, strings.TrimSpace(description), coverColor, id,
	)
	if err != nil {
		return Playlist{}, err
	}
	detail, err := s.GetPlaylistDetail(id)
	return detail.Playlist, err
}

func (s *Store) RenamePlaylist(id int64, name string) (Playlist, error) {
	detail, err := s.GetPlaylistDetail(id)
	if err != nil {
		return Playlist{}, err
	}
	return s.UpdatePlaylist(id, name, detail.Playlist.Description, detail.Playlist.CoverColor)
}

func (s *Store) UpdatePlaylistCover(id int64, coverURL string) {
	if strings.TrimSpace(coverURL) == "" {
		return
	}
	_, _ = s.db.Exec(`UPDATE playlists SET cover_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, strings.TrimSpace(coverURL), id)
}

func (s *Store) DeletePlaylist(id int64) error {
	_, _ = s.db.Exec(`DELETE FROM playlist_tracks WHERE playlist_id = ?`, id)
	_, err := s.db.Exec(`DELETE FROM playlists WHERE id = ?`, id)
	return err
}

func (s *Store) AddTrackToPlaylist(playlistID, trackID int64) error {
	var maxPos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM playlist_tracks WHERE playlist_id = ?`, playlistID).Scan(&maxPos)
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id, position) VALUES (?, ?, ?)`,
		playlistID, trackID, maxPos+1,
	)
	if err == nil {
		_, _ = s.db.Exec(`UPDATE playlists SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, playlistID)
	}
	return err
}

func (s *Store) RemoveTrackFromPlaylist(playlistID, trackID int64) error {
	_, err := s.db.Exec(`DELETE FROM playlist_tracks WHERE playlist_id = ? AND track_id = ?`, playlistID, trackID)
	if err == nil {
		_, _ = s.db.Exec(`UPDATE playlists SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, playlistID)
	}
	return err
}

func (s *Store) ReorderPlaylistTracks(playlistID int64, trackIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	for idx, tid := range trackIDs {
		if _, err = tx.Exec(`UPDATE playlist_tracks SET position = ? WHERE playlist_id = ? AND track_id = ?`, idx, playlistID, tid); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`UPDATE playlists SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, playlistID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetSources() ([]Source, error) {
	rows, err := s.db.Query(`SELECT id, name, type, path, config_json, track_count, enabled, last_synced_at FROM sources ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Source{}
	for rows.Next() {
		var src Source
		var enabled int
		if err := rows.Scan(&src.ID, &src.Name, &src.Type, &src.Path, &src.ConfigJSON, &src.TrackCount, &enabled, &src.LastSyncedAt); err != nil {
			return nil, err
		}
		src.Enabled = enabled == 1
		out = append(out, src)
	}
	return out, rows.Err()
}

func (s *Store) AddSource(name, srcType, path string, trackCount int) (Source, error) {
	var existingID int64
	if path != "" {
		_ = s.db.QueryRow(`SELECT id FROM sources WHERE path = ? AND type = ? LIMIT 1`, path, srcType).Scan(&existingID)
	} else {
		_ = s.db.QueryRow(`SELECT id FROM sources WHERE type = ? AND path = '' LIMIT 1`, srcType).Scan(&existingID)
	}
	if existingID > 0 {
		_, err := s.db.Exec(`UPDATE sources SET name = ?, track_count = ?, last_synced_at = CURRENT_TIMESTAMP WHERE id = ?`, name, trackCount, existingID)
		if err != nil {
			return Source{}, err
		}
		return s.GetSourceByID(existingID)
	}
	res, err := s.db.Exec(
		`INSERT INTO sources (name, type, path, track_count, enabled) VALUES (?, ?, ?, ?, 1)`,
		name, srcType, path, trackCount,
	)
	if err != nil {
		return Source{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Source{}, err
	}
	return s.GetSourceByID(id)
}

func (s *Store) GetSourceByID(id int64) (Source, error) {
	var src Source
	var enabled int
	err := s.db.QueryRow(`SELECT id, name, type, path, config_json, track_count, enabled, last_synced_at FROM sources WHERE id = ?`, id).Scan(
		&src.ID, &src.Name, &src.Type, &src.Path, &src.ConfigJSON, &src.TrackCount, &enabled, &src.LastSyncedAt,
	)
	src.Enabled = enabled == 1
	return src, err
}

func (s *Store) UpdateSourceCount(id int64, trackCount int) error {
	_, err := s.db.Exec(`UPDATE sources SET track_count = ?, last_synced_at = CURRENT_TIMESTAMP WHERE id = ?`, trackCount, id)
	return err
}

func (s *Store) RemoveSource(id int64) error {
	_, err := s.db.Exec(`DELETE FROM sources WHERE id = ?`, id)
	return err
}

func (s *Store) GetPreferences() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM preferences`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prefs := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		prefs[k] = v
	}
	return prefs, rows.Err()
}

func (s *Store) SetPreference(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO preferences (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Store) SavePreferences(prefs map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	for k, v := range prefs {
		if _, err = tx.Exec(`INSERT INTO preferences (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetPlayHistory(limit int) ([]PlayHistoryItem, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT h.id, h.track_id, t.title, t.artist_name, t.album_title, t.cover_url, t.source_type, t.format, t.duration_sec, h.played_at, h.duration_listened_sec, h.completed
		FROM play_history h
		JOIN tracks t ON t.id = h.track_id
		ORDER BY h.id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PlayHistoryItem{}
	for rows.Next() {
		var it PlayHistoryItem
		var comp int
		if err := rows.Scan(&it.ID, &it.TrackID, &it.Title, &it.ArtistName, &it.AlbumTitle, &it.CoverURL, &it.SourceType, &it.Format, &it.DurationSec, &it.PlayedAt, &it.DurationListenedSec, &comp); err != nil {
			return nil, err
		}
		it.Completed = comp == 1
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetRecentSeedTracks returns distinct recently played tracks ordered by most recent play.
func (s *Store) GetRecentSeedTracks(limit int) ([]Track, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		SELECT t.id, t.title, t.artist_id, t.artist_name, t.album_id, t.album_title, t.duration_sec, t.format, t.bitrate, t.source_type, t.file_path, t.external_id, t.cover_url, t.is_favorite, t.play_count, t.is_downloaded, t.downloaded_at, t.created_at
		FROM tracks t
		JOIN (
			SELECT track_id, MAX(id) AS last_play_id
			FROM play_history
			GROUP BY track_id
		) h ON h.track_id = t.id
		ORDER BY h.last_play_id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tracks := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (s *Store) ClearPlayHistory() error {
	_, err := s.db.Exec(`DELETE FROM play_history`)
	return err
}

func (s *Store) SeedDefaults() error {
	defaults := map[string]string{
		"stream_quality":      "high",
		"streak_goal_minutes": "10",
		"streak_goal_tracks":  "3",
		"accent_color":        "#FF9E64",
		"gapless_playback":    "true",
		"volume":              "80",
	}
	for k, v := range defaults {
		var exists int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM preferences WHERE key = ?`, k).Scan(&exists)
		if exists == 0 {
			if err := s.SetPreference(k, v); err != nil {
				return err
			}
		}
	}

	var hasCustomSeed int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM preferences WHERE key = 'theme.seed' AND value != '' AND value != 'monolith'`).Scan(&hasCustomSeed)
	if hasCustomSeed == 0 {
		_, _ = s.db.Exec(`UPDATE preferences SET value = '#FF9E64' WHERE key = 'accent_color' AND UPPER(value) = '#D0BCFF'`)
	}

	var srcCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sources`).Scan(&srcCount); err == nil && srcCount == 0 {
		_, _ = s.AddSource("YouTube Music (InnerTube)", "ytmusic", "", 0)
		_, _ = s.AddSource("Uploaded Local Files", "local_upload", "", 0)
	}

	// Clean up any legacy seeded dummy tracks, playlists, and orphaned albums/artists
	_, _ = s.db.Exec(`DELETE FROM playlist_tracks WHERE track_id IN (SELECT id FROM tracks WHERE external_id IN ('dX3k_QDnzHE', '8GW6sLrK40k', '4NRXx6U8ABQ', '5NV6Rdv1a3I', 'MV_3Dpw-BRY', 'a5uQMwRMHcs'))`)
	_, _ = s.db.Exec(`DELETE FROM play_history WHERE track_id IN (SELECT id FROM tracks WHERE external_id IN ('dX3k_QDnzHE', '8GW6sLrK40k', '4NRXx6U8ABQ', '5NV6Rdv1a3I', 'MV_3Dpw-BRY', 'a5uQMwRMHcs'))`)
	_, _ = s.db.Exec(`DELETE FROM tracks WHERE external_id IN ('dX3k_QDnzHE', '8GW6sLrK40k', '4NRXx6U8ABQ', '5NV6Rdv1a3I', 'MV_3Dpw-BRY', 'a5uQMwRMHcs')`)
	_, _ = s.db.Exec(`DELETE FROM playlist_tracks WHERE playlist_id IN (SELECT id FROM playlists WHERE name IN ('Local Favorites', 'Late Night Synthwave') AND description IN ('Koleksi playlist lokal offline Velora', 'Mix pilihan dari YouTube Music'))`)
	_, _ = s.db.Exec(`DELETE FROM playlists WHERE name IN ('Local Favorites', 'Late Night Synthwave') AND description IN ('Koleksi playlist lokal offline Velora', 'Mix pilihan dari YouTube Music')`)
	_, _ = s.db.Exec(`DELETE FROM albums WHERE id NOT IN (SELECT DISTINCT album_id FROM tracks WHERE album_id IS NOT NULL AND album_id > 0)`)
	_, _ = s.db.Exec(`DELETE FROM artists WHERE id NOT IN (SELECT DISTINCT artist_id FROM tracks WHERE artist_id IS NOT NULL AND artist_id > 0)`)
	_, _ = s.db.Exec(`UPDATE sources SET track_count = (SELECT COUNT(*) FROM tracks WHERE source_type = 'ytmusic') WHERE type = 'ytmusic'`)

	return nil
}

func (s *Store) GetCachedLyrics(trackID int64) (plainLyrics, syncedJSON, source string, ok bool) {
	if trackID <= 0 {
		return "", "", "", false
	}
	err := s.db.QueryRow(
		`SELECT plain_lyrics, synced_json, source FROM track_lyrics WHERE track_id = ?`,
		trackID,
	).Scan(&plainLyrics, &syncedJSON, &source)
	if err != nil {
		return "", "", "", false
	}
	return plainLyrics, syncedJSON, source, source != "" && source != "fallback"
}

func (s *Store) SaveCachedLyrics(trackID int64, plainLyrics, syncedJSON, source string) {
	if trackID <= 0 || source == "" || source == "fallback" {
		return
	}
	_, _ = s.db.Exec(
		`INSERT INTO track_lyrics (track_id, plain_lyrics, synced_json, source, updated_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(track_id) DO UPDATE SET
		   plain_lyrics = excluded.plain_lyrics,
		   synced_json = excluded.synced_json,
		   source = excluded.source,
		   updated_at = CURRENT_TIMESTAMP`,
		trackID, plainLyrics, syncedJSON, source,
	)
}

func (s *Store) MarkTrackDownloaded(trackID int64, filePath, format string) (Track, error) {
	if trackID <= 0 {
		return Track{}, fmt.Errorf("invalid track id")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(format) != "" {
		_, err := s.db.Exec(
			`UPDATE tracks SET file_path = ?, format = ?, is_downloaded = 1, downloaded_at = ? WHERE id = ?`,
			filePath, strings.ToUpper(strings.TrimSpace(format)), now, trackID,
		)
		if err != nil {
			return Track{}, err
		}
	} else {
		_, err := s.db.Exec(
			`UPDATE tracks SET file_path = ?, is_downloaded = 1, downloaded_at = ? WHERE id = ?`,
			filePath, now, trackID,
		)
		if err != nil {
			return Track{}, err
		}
	}
	return s.GetTrackByID(trackID)
}

func (s *Store) ClearTrackDownloaded(trackID int64) (Track, error) {
	if trackID <= 0 {
		return Track{}, fmt.Errorf("invalid track id")
	}
	_, err := s.db.Exec(
		`UPDATE tracks
		 SET is_downloaded = 0,
		     downloaded_at = '',
		     file_path = CASE WHEN source_type = 'ytmusic' THEN '' ELSE file_path END
		 WHERE id = ?`,
		trackID,
	)
	if err != nil {
		return Track{}, err
	}
	return s.GetTrackByID(trackID)
}

func (s *Store) GetDownloadedTracks() ([]Track, error) {
	rows, err := s.db.Query(
		`SELECT id, title, artist_id, artist_name, album_id, album_title, duration_sec, format, bitrate, source_type, file_path, external_id, cover_url, is_favorite, play_count, is_downloaded, downloaded_at, created_at
		 FROM tracks
		 WHERE is_downloaded = 1
		 ORDER BY downloaded_at DESC, id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tracks := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}
