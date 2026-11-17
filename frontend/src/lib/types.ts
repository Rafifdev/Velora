export interface Track {
  id: number;
  title: string;
  artistId: number;
  artistName: string;
  albumId: number;
  albumTitle: string;
  durationSec: number;
  format: string;
  bitrate: number;
  sourceType: 'local' | 'ytmusic' | string;
  filePath: string;
  externalId: string;
  coverUrl: string;
  isFavorite: boolean;
  playCount: number;
  isDownloaded?: boolean;
  downloadedAt?: string;
  globalPlays?: string;
  createdAt: string;
}

export interface DownloadProgress {
  trackId: number;
  externalId: string;
  title: string;
  artist: string;
  status: 'queued' | 'downloading' | 'completed' | 'failed' | 'cancelled' | string;
  progress: number;
  error?: string;
}

export interface Album {
  id: number;
  title: string;
  artistId: number;
  artistName: string;
  coverUrl: string;
  releaseYear: number;
  sourceType: string;
  externalId: string;
  isFavorite: boolean;
  trackCount: number;
}

export interface Artist {
  id: number;
  name: string;
  avatarUrl: string;
  bio: string;
  sourceType: string;
  externalId: string;
  isFavorite: boolean;
  trackCount: number;
}

export interface Playlist {
  id: number;
  name: string;
  description: string;
  coverUrl: string;
  coverColor: string;
  isLocal: boolean;
  sourceType: string;
  externalId: string;
  trackCount: number;
  totalSec: number;
  createdAt: string;
  updatedAt: string;
}

export interface PlaylistDetail {
  playlist: Playlist;
  tracks: Track[];
}

export interface Source {
  id: number;
  name: string;
  type: string;
  path: string;
  configJson: string;
  trackCount: number;
  enabled: boolean;
  lastSyncedAt: string;
}

export interface PlayHistoryItem {
  id: number;
  trackId: number;
  title: string;
  artistName: string;
  albumTitle: string;
  coverUrl: string;
  sourceType: string;
  format: string;
  durationSec: number;
  playedAt: string;
  durationListenedSec: number;
  completed: boolean;
}

export interface DailyStreakRow {
  date: string;
  secondsListened: number;
  tracksCount: number;
  streakMet: boolean;
}

export interface FireStreakStatus {
  currentStreak: number;
  longestStreak: number;
  todaySeconds: number;
  todayMinutes: number;
  todayTracks: number;
  goalMinutes: number;
  goalTracks: number;
  streakMetToday: boolean;
  progressPercent: number;
  last30Days: DailyStreakRow[];
}

export interface HourBucket {
  hour: number;
  label: string;
  minutes: number;
  plays: number;
}

export interface DayBucket {
  dayIndex: number;
  label: string;
  fullName: string;
  minutes: number;
  plays: number;
}

export interface TopStatItem {
  name: string;
  subText: string;
  plays: number;
  minutes: number;
}

export interface ListeningStats {
  timeOfDay: HourBucket[];
  dayOfWeek: DayBucket[];
  totalMinutes: number;
  totalPlays: number;
  uniqueTracks: number;
  peakHourLabel: string;
  peakPersona: string;
  mostActiveDay: string;
  topTracks: TopStatItem[];
  topArtists: TopStatItem[];
}

export interface YTMusicTrack {
  videoId: string;
  title: string;
  artist: string;
  album: string;
  durationSec: number;
  coverUrl: string;
  format: string;
  playCountText?: string;
}

export interface YTMusicCard {
  id: string;
  title: string;
  subtitle: string;
  coverUrl: string;
  kind: 'playlist' | 'album' | 'song' | string;
  videoId: string;
  browseId: string;
  playlistId: string;
  artist: string;
  album: string;
  durationSec: number;
}

export interface YTMusicShelf {
  title: string;
  items: YTMusicCard[];
}

export interface YTMusicHomeFeed {
  shelves: YTMusicShelf[];
  tracks: YTMusicTrack[];
}

export interface YTMusicAlbumView {
  browseId: string;
  title: string;
  artist: string;
  artistBrowseId: string;
  year: string;
  coverUrl: string;
  subtitle: string;
  tracks: Track[];
  artistAlbums: YTMusicCard[];
}

export interface LyricLine {
  timeSec: number;
  text: string;
}

export interface TrackLyrics {
  trackId: number;
  title: string;
  artist: string;
  plainLyrics: string;
  syncedLyrics: LyricLine[];
  source: string;
}

export interface BootstrapData {
  tracks: Track[];
  albums: Album[];
  artists: Artist[];
  playlists: Playlist[];
  sources: Source[];
  history: PlayHistoryItem[];
  preferences: Record<string, string>;
  fireStreak: FireStreakStatus;
  listeningStats: ListeningStats;
  mediaBaseUrl: string;
}

export type HomeFilter = 'semua' | 'musik' | 'artis' | 'album' | 'playlist' | 'podcast' | 'buku';

export type ViewRoute =
  | { name: 'home'; filter?: HomeFilter }
  | { name: 'search'; query: string }
  | { name: 'liked-songs' }
  | { name: 'favorite-albums' }
  | { name: 'album-detail'; albumId: number }
  | { name: 'favorite-artists' }
  | { name: 'artist-detail'; artistId: number }
  | { name: 'playlists' }
  | { name: 'local-playlists' }
  | { name: 'downloaded' }
  | { name: 'playlist-detail'; playlistId: number }
  | { name: 'history' }
  | { name: 'sources' }
  | { name: 'preferences' }
  | { name: 'listening-stats' }
  | { name: 'fire-streak' }
  | { name: 'lyrics' }
  | { name: 'track-detail'; trackId: number }
  | {
      name: 'yt-album-detail';
      videoId: string;
      browseId?: string;
      title: string;
      artist: string;
      albumTitle: string;
      coverUrl: string;
    };
