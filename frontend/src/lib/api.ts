import type {
  Album,
  Artist,
  BootstrapData,
  FireStreakStatus,
  ListeningStats,
  PlayHistoryItem,
  Playlist,
  PlaylistDetail,
  Source,
  Track,
  TrackLyrics,
  DownloadProgress,
  JamPlaybackSnapshot,
  JamStatus,
  YTMusicAlbumView,
  YTMusicArtistView,
  YTMusicCard,
  YTMusicHomeFeed,
  YTMusicTrack
} from './types';

export const isJamRemote =
  typeof window !== 'undefined' && !(window as any)?.go?.main?.App;

const jamRpcProxy = new Proxy(
  {},
  {
    get(_target, prop) {
      const method = String(prop);
      return async (...args: any[]) => {
        const res = await fetch(`/api/rpc/${encodeURIComponent(method)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ args })
        });
        const data = await res.json();
        if (!res.ok) {
          throw new Error(data?.error || `RPC ${method} failed (${res.status})`);
        }
        return data;
      };
    }
  }
);

function appBridge(): any {
  return (window as any)?.go?.main?.App ?? jamRpcProxy;
}

export function safeEventsOn(eventName: string, callback: (...args: any[]) => void): () => void {
  const fn = (window as any)?.runtime?.EventsOn;
  if (typeof fn === 'function') {
    return fn(eventName, callback);
  }
  return () => {};
}

export async function sendJamCommand(action: string, payload: Record<string, any> | number = {}): Promise<void> {
  try {
    await fetch(`/api/playback/${encodeURIComponent(action)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(typeof payload === 'number' ? { value: payload } : payload)
    });
  } catch {
  }
}

export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00';
  const total = Math.floor(seconds);
  const mins = Math.floor(total / 60);
  const secs = total % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

export function formatTotalMinutes(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  if (mins < 60) return `${mins} mnt`;
  const hrs = Math.floor(mins / 60);
  const rem = mins % 60;
  return `${hrs} jam ${rem} mnt`;
}

export function resolveCoverUrl(coverUrl: string, mediaBaseUrl = ''): string {
  if (!coverUrl) return '';
  let resolved = coverUrl;
  const ytImgMatch = resolved.match(/https?:\/\/i\.ytimg\.com\/(?:vi|vi_webp)\/([^/?#]+)\/[^?#]+(?:\?.*)?$/i);
  if (ytImgMatch) {
    resolved = `https://i.ytimg.com/vi/${ytImgMatch[1]}/mqdefault.jpg`;
  } else if (resolved.includes('googleusercontent.com') || resolved.includes('yt3.ggpht.com')) {
    resolved = resolved.replace(/=w\d+-h\d+[^&?]*/i, '=w544-h544-c-l90-rj');
  }
  if (resolved.startsWith('http://') || resolved.startsWith('https://')) {
    if (
      mediaBaseUrl &&
      (resolved.includes('googleusercontent.com') ||
        resolved.includes('ggpht.com') ||
        resolved.includes('ytimg.com') ||
        resolved.includes('youtube.com'))
    ) {
      return `${mediaBaseUrl}/media/cover?url=${encodeURIComponent(resolved)}`;
    }
    return resolved;
  }
  if (resolved.startsWith('data:')) {
    return resolved;
  }
  if (resolved.startsWith('/media/') && mediaBaseUrl) {
    return `${mediaBaseUrl}${resolved}`;
  }
  return resolved;
}

export function isYTVideoThumb(url: string): boolean {
  return Boolean(url && (url.includes('i.ytimg.com/') || url.includes('i.ytimg.com%2F')));
}

export const api = {
  async getBootstrapData(): Promise<BootstrapData> {
    return appBridge().GetBootstrapData();
  },
  async getTracks(filter: string, query: string): Promise<Track[]> {
    return appBridge().GetTracks(filter, query);
  },
  async getAlbums(favoritesOnly: boolean): Promise<Album[]> {
    return appBridge().GetAlbums(favoritesOnly);
  },
  async getAlbumTracks(albumId: number): Promise<Track[]> {
    return appBridge().GetAlbumTracks(albumId);
  },
  async getArtists(favoritesOnly: boolean): Promise<Artist[]> {
    return appBridge().GetArtists(favoritesOnly);
  },
  async getArtistTracks(artistId: number): Promise<Track[]> {
    return appBridge().GetArtistTracks(artistId);
  },
  async toggleFavoriteTrack(id: number): Promise<boolean> {
    return appBridge().ToggleFavoriteTrack(id);
  },
  async toggleFavoriteAlbum(id: number): Promise<boolean> {
    return appBridge().ToggleFavoriteAlbum(id);
  },
  async toggleFavoriteYTMusicAlbum(
    browseId: string,
    title: string,
    artistName: string,
    coverUrl: string,
    releaseYear: number
  ): Promise<Album> {
    return appBridge().ToggleFavoriteYTMusicAlbum(browseId, title, artistName, coverUrl, releaseYear);
  },
  async toggleFavoriteArtist(id: number): Promise<boolean> {
    return appBridge().ToggleFavoriteArtist(id);
  },
  async toggleFavoriteYTMusicArtist(
    externalId: string,
    name: string,
    avatarUrl: string
  ): Promise<Artist> {
    return appBridge().ToggleFavoriteYTMusicArtist(externalId, name, avatarUrl);
  },
  async getPlaylists(localOnly: boolean): Promise<Playlist[]> {
    return appBridge().GetPlaylists(localOnly);
  },
  async getPlaylistDetail(id: number): Promise<PlaylistDetail> {
    return appBridge().GetPlaylistDetail(id);
  },
  async createPlaylist(name: string, description: string, isLocal: boolean, coverColor: string): Promise<Playlist> {
    return appBridge().CreatePlaylist(name, description, isLocal, coverColor);
  },
  async updatePlaylist(id: number, name: string, description: string, coverColor: string): Promise<Playlist> {
    return appBridge().UpdatePlaylist(id, name, description, coverColor);
  },
  async deletePlaylist(id: number): Promise<void> {
    return appBridge().DeletePlaylist(id);
  },
  async addTrackToPlaylist(playlistId: number, trackId: number): Promise<void> {
    return appBridge().AddTrackToPlaylist(playlistId, trackId);
  },
  async removeTrackFromPlaylist(playlistId: number, trackId: number): Promise<void> {
    return appBridge().RemoveTrackFromPlaylist(playlistId, trackId);
  },
  async reorderPlaylistTracks(playlistId: number, trackIds: number[]): Promise<void> {
    return appBridge().ReorderPlaylistTracks(playlistId, trackIds);
  },
  async exportLocalPlaylistM3U(playlistId: number): Promise<string> {
    return appBridge().ExportLocalPlaylistM3U(playlistId);
  },
  async importLocalPlaylistM3U(): Promise<Playlist> {
    return appBridge().ImportLocalPlaylistM3U();
  },
  async importSpotifyPlaylist(url: string): Promise<Playlist> {
    return appBridge().ImportSpotifyPlaylist(url);
  },
  async uploadLocalFiles(): Promise<Track[]> {
    return appBridge().UploadLocalFiles();
  },
  async importLocalPaths(paths: string[]): Promise<Track[]> {
    return appBridge().ImportLocalPaths(paths);
  },
  async addLocalFolder(): Promise<Track[]> {
    return appBridge().AddLocalFolder();
  },
  async rescanSource(sourceId: number): Promise<Track[]> {
    return appBridge().RescanSource(sourceId);
  },
  async removeSource(sourceId: number): Promise<void> {
    return appBridge().RemoveSource(sourceId);
  },
  async getSources(): Promise<Source[]> {
    return appBridge().GetSources();
  },
  async searchYouTubeMusic(query: string): Promise<YTMusicTrack[]> {
    return appBridge().SearchYouTubeMusic(query);
  },
  async searchYouTubeMusicPlaylists(query: string): Promise<YTMusicCard[]> {
    return appBridge().SearchYouTubeMusicPlaylists(query);
  },
  async searchYouTubeMusicAlbums(query: string): Promise<YTMusicCard[]> {
    return appBridge().SearchYouTubeMusicAlbums(query);
  },
  async searchYouTubeMusicArtists(query: string): Promise<YTMusicCard[]> {
    return appBridge().SearchYouTubeMusicArtists(query);
  },
  async getMoreArtists(offset: number, excludeBrowseIds: string[]): Promise<YTMusicCard[]> {
    return appBridge().GetMoreArtists(offset, excludeBrowseIds);
  },
  async getYTMusicHome(): Promise<YTMusicHomeFeed> {
    return appBridge().GetYTMusicHome();
  },
  async getYTMusicHomeContinuation(continuation: string): Promise<YTMusicHomeFeed> {
    return appBridge().GetYTMusicHomeContinuation(continuation);
  },
  async getYTMusicTop100(): Promise<YTMusicTrack[]> {
    return appBridge().GetYTMusicTop100();
  },
  async getYTMusicTop100Recommended(excludeVideoIDs: string[]): Promise<YTMusicTrack[]> {
    return appBridge().GetYTMusicTop100Recommended(excludeVideoIDs);
  },
  async getMoreTopSongs(offset: number, excludeVideoIDs: string[]): Promise<YTMusicTrack[]> {
    const fn = appBridge().GetMoreTopSongs;
    if (typeof fn === 'function') {
      return fn(offset, excludeVideoIDs);
    }
    return appBridge().GetYTMusicTop100Recommended(excludeVideoIDs);
  },
  async playYTMusicCollection(
    browseId: string,
    playlistId: string,
    title: string,
    subtitle: string,
    coverUrl: string
  ): Promise<Track[]> {
    return appBridge().PlayYTMusicCollection(browseId, playlistId, title, subtitle, coverUrl);
  },
  async getYTMusicTrackAlbum(
    videoId: string,
    browseId: string,
    title: string,
    artist: string,
    albumTitle: string,
    coverUrl: string
  ): Promise<YTMusicAlbumView> {
    return appBridge().GetYTMusicTrackAlbum(videoId, browseId, title, artist, albumTitle, coverUrl);
  },
  async getYTMusicArtistDetail(
    browseId: string,
    name: string,
    avatarUrl: string
  ): Promise<YTMusicArtistView> {
    return appBridge().GetYTMusicArtistDetail(browseId, name, avatarUrl);
  },
  async getTrackLyrics(
    trackId: number,
    title: string,
    artist: string,
    album: string,
    durationSec: number
  ): Promise<TrackLyrics> {
    return appBridge().GetTrackLyrics(trackId, title, artist, album, durationSec);
  },
  async saveYTMusicTrack(track: YTMusicTrack): Promise<Track> {
    return appBridge().SaveYTMusicTrack(track);
  },
  async playYTMusicTrack(track: YTMusicTrack): Promise<Track> {
    return appBridge().PlayYTMusicTrack(track);
  },
  async getStreamUrl(trackId: number): Promise<string> {
    return appBridge().GetStreamURL(trackId);
  },
  async playerControl(action: string, value = 0, mode = ''): Promise<any> {
    return appBridge().PlayerControl(action, value, mode);
  },
  async recordListenProgress(trackId: number, seconds: number, completed: boolean): Promise<FireStreakStatus> {
    return appBridge().RecordListenProgress(trackId, seconds, completed);
  },
  async getPlayHistory(): Promise<PlayHistoryItem[]> {
    return appBridge().GetPlayHistory();
  },
  async clearPlayHistory(): Promise<void> {
    return appBridge().ClearPlayHistory();
  },
  async getListeningStats(): Promise<ListeningStats> {
    return appBridge().GetListeningStats();
  },
  async getListeningStatsByDate(dateStr: string): Promise<ListeningStats> {
    const fn = appBridge()?.GetListeningStatsByDate;
    if (!fn) return appBridge().GetListeningStats();
    return fn(dateStr);
  },
  async getFireStreak(): Promise<FireStreakStatus> {
    return appBridge().GetFireStreak();
  },
  async getMonthlyStreak(year: number, month: number): Promise<import('./types').DailyStreakRow[]> {
    const fn = appBridge()?.GetMonthlyStreak;
    if (!fn) return [];
    return (await fn(year, month)) ?? [];
  },
  async savePreferences(prefs: Record<string, string>): Promise<void> {
    return appBridge().SavePreferences(prefs);
  },
  async clearCache(): Promise<number> {
    return appBridge().ClearCache();
  },
  async getDataDir(): Promise<string> {
    return (await appBridge()?.GetDataDir?.()) || '';
  },
  openExternalUrl(url: string): void {
    const fn = appBridge()?.OpenExternalURL;
    if (fn) {
      void fn(url);
    } else if (typeof window !== 'undefined') {
      window.open(url, '_blank', 'noopener,noreferrer');
    }
  },
  async generateQRDataURL(content: string): Promise<string> {
    return (await appBridge()?.GenerateQRDataURL?.(content)) ?? '';
  },
  async checkForUpdates(): Promise<import('./types').UpdateCheckResult> {
    const fn = appBridge()?.CheckForUpdates;
    if (!fn) {
      return {
        currentVersion: '1.0.0',
        latestVersion: '1.0.0',
        hasUpdate: false,
        releaseUrl: 'https://github.com/Rafifdev/Velora/releases',
        downloadUrl: '',
        releaseNotes: '',
        publishedAt: ''
      };
    }
    return await fn();
  },
  async openDataFolder(): Promise<void> {
    return appBridge()?.OpenDataFolder?.();
  },
  async submitFeedback(
    title: string,
    detail: string,
    screenshotName = '',
    screenshotDataURL = ''
  ): Promise<void> {
    const fn = appBridge()?.SubmitFeedback;
    if (!fn) {
      throw new Error('SubmitFeedback hanya tersedia di aplikasi desktop');
    }
    return await fn(title, detail, screenshotName, screenshotDataURL);
  },
  async setWindowTitle(title: string): Promise<void> {
    return appBridge().SetWindowTitle?.(title);
  },
  async setKeepAwake(enabled: boolean): Promise<void> {
    return appBridge().SetKeepAwake?.(enabled);
  },
  async downloadTrack(trackId: number): Promise<Track> {
    return appBridge().DownloadTrack(trackId);
  },
  async downloadYTMusicTrack(track: YTMusicTrack): Promise<Track> {
    return appBridge().DownloadYTMusicTrack(track);
  },
  async removeDownloadedTrack(trackId: number): Promise<Track> {
    return appBridge().RemoveDownloadedTrack(trackId);
  },
  async cancelDownload(trackId: number): Promise<void> {
    return appBridge().CancelDownload(trackId);
  },
  async getDownloadedTracks(): Promise<Track[]> {
    return appBridge().GetDownloadedTracks();
  },
  async getDownloadStates(): Promise<DownloadProgress[]> {
    return appBridge().GetDownloadStates();
  },
  async startJam(): Promise<JamStatus> {
    return appBridge().StartJam();
  },
  async stopJam(): Promise<JamStatus> {
    return appBridge().StopJam();
  },
  async getJamStatus(): Promise<JamStatus> {
    return appBridge().GetJamStatus();
  },
  updateJamState(snap: JamPlaybackSnapshot): void {
    if (isJamRemote) return;
    appBridge()?.UpdateJamState?.(snap);
  },
  windowMinimize(): void {
    if (isJamRemote) return;
    appBridge()?.WindowMinimize();
  },
  windowMaximize(): void {
    if (isJamRemote) return;
    appBridge()?.WindowMaximize();
  },
  async windowIsMaximized(): Promise<boolean> {
    if (isJamRemote) return false;
    try {
      if (appBridge()?.WindowIsMaximized) {
        return await appBridge().WindowIsMaximized();
      }
      if ((window as any)?.runtime?.WindowIsMaximised) {
        return await (window as any).runtime.WindowIsMaximised();
      }
    } catch {
    }
    return window.outerWidth >= window.screen.availWidth - 8 && window.outerHeight >= window.screen.availHeight - 8;
  },
  windowClose(): void {
    if (isJamRemote) return;
    appBridge()?.WindowClose();
  },
  updateTaskbarState(title: string, artist: string, coverUrl: string, isPlaying: boolean): void {
    if (isJamRemote) return;
    appBridge()?.UpdateTaskbarState?.(title, artist, coverUrl, isPlaying);
  },
  updateDynamicIslandProgress(positionSec: number, durationSec: number): void {
    if (isJamRemote) return;
    appBridge()?.UpdateDynamicIslandProgress?.(positionSec, durationSec);
  },
  setDynamicIslandEnabled(enabled: boolean): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandEnabled?.(enabled);
  },
  setDynamicIslandLanguage(lang: string): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandLanguage?.(lang);
  },

  setDynamicIslandStyle(style: string): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandStyle?.(style);
  },
  setDynamicIslandShapeSize(compactStyle: string, size: string, opacity: number): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandShapeSize?.(compactStyle, size, opacity);
  },
  setDynamicIslandHideWhenAppActive(hide: boolean): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandHideWhenAppActive?.(hide);
  },
  setDynamicIslandHideInFullscreen(hide: boolean): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandHideInFullscreen?.(hide);
  },
  setDynamicIslandIdleHideSeconds(sec: number): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandIdleHideSeconds?.(sec);
  },
  setDynamicIslandScreen(mode: string): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandScreen?.(mode);
  },
  setDynamicIslandAnimSpeed(speed: string): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandAnimSpeed?.(speed);
  },
  setDynamicIslandOpenTrigger(trigger: string): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandOpenTrigger?.(trigger);
  },
  setDynamicIslandOnlyPlaying(only: boolean): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandOnlyPlaying?.(only);
  },
  setDynamicIslandPauseHideSeconds(sec: number): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandPauseHideSeconds?.(sec);
  },
  setDynamicIslandContent(
    showStatus: boolean,
    showCover: boolean,
    showClosedCover: boolean,
    showArtist: boolean,
    showLyric: boolean,
    showHeart: boolean,
    showDots: boolean,
    showProgress: boolean,
    progressStyle: string
  ): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandContent?.(
      showStatus,
      showCover,
      showClosedCover,
      showArtist,
      showLyric,
      showHeart,
      showDots,
      showProgress,
      progressStyle
    );
  },
  setDynamicIslandCoverOptions(
    coverStyle: string,
    coverSpin: boolean,
    closedCoverSpin: boolean
  ): void {
    if (isJamRemote) return;
    appBridge()?.SetDynamicIslandCoverOptions?.(coverStyle, coverSpin, closedCoverSpin);
  },
  syncDynamicIsland(payload: {
    title: string;
    artist: string;
    album: string;
    coverUrl: string;
    isPlaying: boolean;
    isFavorite: boolean;
    positionSec: number;
    durationSec: number;
    volume: number;
    muted: boolean;
    shuffle: boolean;
    repeatMode: string;
    eqPreset: string;
    themeSeed: string;
    accentColor: string;
    streakDays: number;
    lyricsLoading: boolean;
    trackLoading?: boolean;
    queueTitle: string;
    queueIndex: number;
    queue: Array<{
      id: number;
      externalId: string;
      title: string;
      artist: string;
      coverUrl: string;
      isFavorite: boolean;
    }>;
  }): void {
    if (isJamRemote) return;
    appBridge()?.SyncDynamicIsland?.(payload);
  },
  notifyDynamicIslandAudioDevice(deviceName = ''): void {
    if (isJamRemote) return;
    appBridge()?.NotifyDynamicIslandAudioDevice?.(deviceName);
  }
};

const dominantColorCache = new Map<string, string | null>();

export async function extractDominantColor(imageUrl: string): Promise<string | null> {
  if (!imageUrl || typeof document === 'undefined') return null;
  if (dominantColorCache.has(imageUrl)) {
    return dominantColorCache.get(imageUrl) ?? null;
  }

  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      try {
        const size = 16;
        const canvas = document.createElement('canvas');
        canvas.width = size;
        canvas.height = size;
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          dominantColorCache.set(imageUrl, null);
          resolve(null);
          return;
        }
        ctx.drawImage(img, 0, 0, size, size);
        const data = ctx.getImageData(0, 0, size, size).data;
        let rSum = 0;
        let gSum = 0;
        let bSum = 0;
        let wSum = 0;

        for (let i = 0; i < data.length; i += 4) {
          const r = data[i];
          const g = data[i + 1];
          const b = data[i + 2];
          const a = data[i + 3];
          if (a < 128) continue;
          const max = Math.max(r, g, b);
          const min = Math.min(r, g, b);
          if (max < 24 || min > 235) continue;
          const sat = max === 0 ? 0 : (max - min) / max;
          const weight = 0.25 + sat * 1.5;
          rSum += r * weight;
          gSum += g * weight;
          bSum += b * weight;
          wSum += weight;
        }

        if (wSum === 0) {
          dominantColorCache.set(imageUrl, null);
          resolve(null);
          return;
        }
        const color = `rgb(${Math.round(rSum / wSum)}, ${Math.round(gSum / wSum)}, ${Math.round(bSum / wSum)})`;
        dominantColorCache.set(imageUrl, color);
        resolve(color);
      } catch {
        dominantColorCache.set(imageUrl, null);
        resolve(null);
      }
    };
    img.onerror = () => {
      dominantColorCache.set(imageUrl, null);
      resolve(null);
    };
    img.src = imageUrl;
  });
}
