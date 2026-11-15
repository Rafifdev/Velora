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
  YTMusicAlbumView,
  YTMusicCard,
  YTMusicHomeFeed,
  YTMusicTrack
} from './types';

function appBridge(): any {
  return (window as any)?.go?.main?.App;
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
  async toggleFavoriteArtist(id: number): Promise<boolean> {
    return appBridge().ToggleFavoriteArtist(id);
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
  async getYTMusicHome(): Promise<YTMusicHomeFeed> {
    return appBridge().GetYTMusicHome();
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
  async getFireStreak(): Promise<FireStreakStatus> {
    return appBridge().GetFireStreak();
  },
  async savePreferences(prefs: Record<string, string>): Promise<void> {
    return appBridge().SavePreferences(prefs);
  },
  async clearCache(): Promise<number> {
    return appBridge().ClearCache();
  },
  windowMinimize(): void {
    appBridge()?.WindowMinimize();
  },
  windowMaximize(): void {
    appBridge()?.WindowMaximize();
  },
  async windowIsMaximized(): Promise<boolean> {
    try {
      if (appBridge()?.WindowIsMaximized) {
        return await appBridge().WindowIsMaximized();
      }
      if ((window as any)?.runtime?.WindowIsMaximised) {
        return await (window as any).runtime.WindowIsMaximised();
      }
    } catch {
      // ignore
    }
    return window.outerWidth >= window.screen.availWidth - 8 && window.outerHeight >= window.screen.availHeight - 8;
  },
  windowClose(): void {
    appBridge()?.WindowClose();
  },
  updateTaskbarState(title: string, artist: string, coverUrl: string, isPlaying: boolean): void {
    appBridge()?.UpdateTaskbarState?.(title, artist, coverUrl, isPlaying);
  }
};
