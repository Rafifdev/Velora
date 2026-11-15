CREATE TABLE IF NOT EXISTS sources (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	path TEXT DEFAULT '',
	config_json TEXT DEFAULT '{}',
	track_count INTEGER DEFAULT 0,
	enabled INTEGER DEFAULT 1,
	last_synced_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS artists (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	avatar_url TEXT DEFAULT '',
	bio TEXT DEFAULT '',
	source_type TEXT DEFAULT 'local',
	external_id TEXT DEFAULT '',
	is_favorite INTEGER DEFAULT 0,
	created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS albums (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	artist_id INTEGER DEFAULT 0,
	artist_name TEXT NOT NULL,
	cover_url TEXT DEFAULT '',
	release_year INTEGER DEFAULT 2024,
	source_type TEXT DEFAULT 'local',
	external_id TEXT DEFAULT '',
	is_favorite INTEGER DEFAULT 0,
	created_at TEXT DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(title, artist_name)
);

CREATE TABLE IF NOT EXISTS tracks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	artist_id INTEGER DEFAULT 0,
	artist_name TEXT NOT NULL,
	album_id INTEGER DEFAULT 0,
	album_title TEXT DEFAULT 'Single',
	duration_sec INTEGER DEFAULT 180,
	format TEXT DEFAULT 'MP3',
	bitrate INTEGER DEFAULT 320,
	source_type TEXT NOT NULL DEFAULT 'local',
	file_path TEXT DEFAULT '',
	external_id TEXT DEFAULT '',
	cover_url TEXT DEFAULT '',
	is_favorite INTEGER DEFAULT 0,
	play_count INTEGER DEFAULT 0,
	created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlists (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	description TEXT DEFAULT '',
	cover_url TEXT DEFAULT '',
	cover_color TEXT DEFAULT '#1DB954',
	is_local INTEGER DEFAULT 1,
	source_type TEXT DEFAULT 'local',
	external_id TEXT DEFAULT '',
	created_at TEXT DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlist_tracks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	playlist_id INTEGER NOT NULL,
	track_id INTEGER NOT NULL,
	position INTEGER NOT NULL DEFAULT 0,
	added_at TEXT DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(playlist_id, track_id)
);

CREATE TABLE IF NOT EXISTS play_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	track_id INTEGER NOT NULL,
	played_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	duration_listened_sec INTEGER NOT NULL DEFAULT 0,
	completed INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS daily_streaks (
	date_str TEXT PRIMARY KEY,
	seconds_listened INTEGER NOT NULL DEFAULT 0,
	tracks_count INTEGER NOT NULL DEFAULT 0,
	streak_met INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS preferences (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS track_lyrics (
	track_id INTEGER PRIMARY KEY,
	plain_lyrics TEXT NOT NULL DEFAULT '',
	synced_json TEXT NOT NULL DEFAULT '[]',
	source TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tracks_source_type ON tracks(source_type);
CREATE INDEX IF NOT EXISTS idx_tracks_external_id ON tracks(external_id);
CREATE INDEX IF NOT EXISTS idx_tracks_file_path ON tracks(file_path);
CREATE INDEX IF NOT EXISTS idx_play_history_played_at ON play_history(played_at);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_pid ON playlist_tracks(playlist_id, position);
