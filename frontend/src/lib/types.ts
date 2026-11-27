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
  downloadGroupKey?: string;
  downloadGroupTitle?: string;
  downloadGroupCover?: string;
  downloadGroupKind?: string;
  globalPlays?: string;
  historyEntryId?: number;
  deletedReason?: 'deleted' | 'canceled' | 'failed';
  createdAt: string;
}

export interface DownloadGroup {
  key: string;
  title: string;
  cover: string;
  kind: 'playlist' | 'album' | string;
}

export interface DownloadProgress {
  trackId: number;
  externalId: string;
  title: string;
  artist: string;
  status: 'queued' | 'downloading' | 'paused' | 'completed' | 'failed' | 'cancelled' | string;
  progress: number;
  error?: string;
  groupKey?: string;
  groupTitle?: string;
  groupCover?: string;
  groupKind?: string;
  groupTotal?: number;
  groupCompleted?: number;
  groupProgress?: number;
  downloadedBytes?: number;
  totalBytes?: number;
  speedBps?: number;
  groupDownloadedBytes?: number;
  groupTotalBytes?: number;
  groupSpeedBps?: number;
}

export interface GroupedDownloadItem {
  key: string;
  isGroup: boolean;
  kind: 'track' | 'playlist' | 'album';
  trackId: number;
  trackIds: number[];
  externalId: string;
  title: string;
  subtitle: string;
  coverUrl: string;
  status: 'queued' | 'downloading' | 'paused' | 'failed';
  progress: number;
  totalCount: number;
  completedCount: number;
  downloadedBytes: number;
  totalBytes: number;
  speedBps: number;
  error?: string;
}

export interface GroupedCompletedDownload {
  key: string;
  isGroup: boolean;
  kind: 'music' | 'album' | 'playlist';
  title: string;
  subtitle: string;
  coverUrl: string;
  tracks: Track[];
  primaryTrack: Track;
  totalDurationSec: number;
  deleted: boolean;
  deletedReason?: 'deleted' | 'canceled' | 'failed';
  sortMs: number;
  downloadedAt: string;
}

export interface JamQueueItem {
  trackId: number;
  externalId: string;
  title: string;
  artist: string;
  durationSec: number;
  coverUrl: string;
}

export interface JamPlaybackSnapshot {
  trackId: number;
  externalId: string;
  title: string;
  artist: string;
  album: string;
  coverUrl: string;
  isPlaying: boolean;
  positionSec: number;
  durationSec: number;
  volume: number;
  muted: boolean;
  shuffle: boolean;
  repeatMode: string;
  queue: JamQueueItem[];
}

export interface JamCommand {
  action: string;
  value: number;
  trackId?: number;
  externalId?: string;
  title?: string;
  artist?: string;
  album?: string;
  coverUrl?: string;
  durationSec?: number;
}

export interface JamStatus {
  enabled: boolean;
  running: boolean;
  port: number;
  lanIp: string;
  remoteUrl: string;
  apiUrl: string;
  qrDataUrl: string;
  connectedClients: number;
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
  imageUrl?: string;
  plays: number;
  minutes: number;
}

export interface RangeBucket {
  label: string;
  minutes: number;
  plays: number;
}

export interface ListeningStats {
  timeOfDay: HourBucket[];
  todayHours?: HourBucket[];
  weekSeries?: RangeBucket[];
  monthSeries?: RangeBucket[];
  yearSeries?: RangeBucket[];
  dayOfWeek: DayBucket[];
  totalMinutes: number;
  totalPlays: number;
  uniqueTracks: number;
  activeDays?: number;
  peakHourLabel: string;
  peakPersona: string;
  mostActiveDay: string;
  topTracks: TopStatItem[];
  topArtists: TopStatItem[];
}

export interface UpdateCheckResult {
  currentVersion: string;
  latestVersion: string;
  hasUpdate: boolean;
  releaseUrl: string;
  downloadUrl: string;
  storeUrl: string;
  isStoreApp: boolean;
  releaseNotes: string;
  publishedAt: string;
}

export interface UpdateProgress {
  status: 'downloading' | 'installing' | 'restarting' | 'error';
  progress: number;
  error?: string;
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
  explicit?: boolean;
  videoType?: string;
}

export interface YTMusicCard {
  id: string;
  title: string;
  subtitle: string;
  coverUrl: string;
  kind: 'playlist' | 'album' | 'song' | 'artist' | string;
  videoId: string;
  browseId: string;
  playlistId: string;
  artist: string;
  album: string;
  durationSec: number;
  explicit?: boolean;
}

export interface YTMusicShelf {
  title: string;
  items: YTMusicCard[];
}

export interface YTMusicHomeFeed {
  shelves: YTMusicShelf[];
  tracks: YTMusicTrack[];
  continuation?: string;
}

export interface YTMusicAlbumView {
  browseId: string;
  kind?: 'album' | 'playlist';
  title: string;
  artist: string;
  artistBrowseId: string;
  year: string;
  coverUrl: string;
  subtitle: string;
  tracks: Track[];
  artistAlbums: YTMusicCard[];
}

export interface YTMusicArtistView {
  browseId: string;
  name: string;
  avatarUrl: string;
  fansCount: number;
  topTracks: Track[];
  albums: YTMusicCard[];
  relatedArtists: YTMusicCard[];
}

export interface AudioOutputDevice {
  id: string;
  name: string;
  connType: 'bluetooth' | 'wired' | 'speaker' | string;
  isActive: boolean;
  batPct: number;
  volPct: number;
}

export interface LyricWord {
  timeSec: number;
  durationSec: number;
  text: string;
}

export interface LyricLine {
  timeSec: number;
  text: string;
  words?: LyricWord[];
}

export interface TrackLyrics {
  trackId: number;
  title: string;
  artist: string;
  plainLyrics: string;
  syncedLyrics: LyricLine[];
  source: string;
}

export interface TrackLoudnessResult {
  loudnessDb: number;
  hasLoudness: boolean;
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
  | { name: 'library'; filter?: 'all' | 'music' | 'albums' | 'artists' | 'downloaded' }
  | { name: 'downloads' }
  | { name: 'liked-songs' }
  | { name: 'favorite-albums' }
  | { name: 'album-detail'; albumId: number }
  | { name: 'favorite-artists' }
  | { name: 'artist-detail'; artistId?: number; browseId?: string; artistName?: string; avatarUrl?: string }
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
  | { name: 'track-detail'; trackId: number; singleTrack?: boolean }
  | {
      name: 'yt-album-detail';
      videoId: string;
      browseId?: string;
      kind?: 'album' | 'playlist';
      title: string;
      artist: string;
      albumTitle: string;
      coverUrl: string;
      localTracks?: Track[];
    };
