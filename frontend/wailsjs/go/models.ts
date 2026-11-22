export namespace audio {
	
	export class PlaybackState {
	    currentTrack?: db.Track;
	    isPlaying: boolean;
	    positionSec: number;
	    durationSec: number;
	    volume: number;
	    muted: boolean;
	    shuffle: boolean;
	    repeatMode: string;
	    backend: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaybackState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentTrack = this.convertValues(source["currentTrack"], db.Track);
	        this.isPlaying = source["isPlaying"];
	        this.positionSec = source["positionSec"];
	        this.durationSec = source["durationSec"];
	        this.volume = source["volume"];
	        this.muted = source["muted"];
	        this.shuffle = source["shuffle"];
	        this.repeatMode = source["repeatMode"];
	        this.backend = source["backend"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace db {
	
	export class Album {
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
	
	    static createFrom(source: any = {}) {
	        return new Album(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artistId = source["artistId"];
	        this.artistName = source["artistName"];
	        this.coverUrl = source["coverUrl"];
	        this.releaseYear = source["releaseYear"];
	        this.sourceType = source["sourceType"];
	        this.externalId = source["externalId"];
	        this.isFavorite = source["isFavorite"];
	        this.trackCount = source["trackCount"];
	    }
	}
	export class Artist {
	    id: number;
	    name: string;
	    avatarUrl: string;
	    bio: string;
	    sourceType: string;
	    externalId: string;
	    isFavorite: boolean;
	    trackCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Artist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.avatarUrl = source["avatarUrl"];
	        this.bio = source["bio"];
	        this.sourceType = source["sourceType"];
	        this.externalId = source["externalId"];
	        this.isFavorite = source["isFavorite"];
	        this.trackCount = source["trackCount"];
	    }
	}
	export class DailyStreakRow {
	    date: string;
	    secondsListened: number;
	    tracksCount: number;
	    streakMet: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DailyStreakRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.secondsListened = source["secondsListened"];
	        this.tracksCount = source["tracksCount"];
	        this.streakMet = source["streakMet"];
	    }
	}
	export class PlayHistoryItem {
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
	
	    static createFrom(source: any = {}) {
	        return new PlayHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.trackId = source["trackId"];
	        this.title = source["title"];
	        this.artistName = source["artistName"];
	        this.albumTitle = source["albumTitle"];
	        this.coverUrl = source["coverUrl"];
	        this.sourceType = source["sourceType"];
	        this.format = source["format"];
	        this.durationSec = source["durationSec"];
	        this.playedAt = source["playedAt"];
	        this.durationListenedSec = source["durationListenedSec"];
	        this.completed = source["completed"];
	    }
	}
	export class Playlist {
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
	
	    static createFrom(source: any = {}) {
	        return new Playlist(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.coverUrl = source["coverUrl"];
	        this.coverColor = source["coverColor"];
	        this.isLocal = source["isLocal"];
	        this.sourceType = source["sourceType"];
	        this.externalId = source["externalId"];
	        this.trackCount = source["trackCount"];
	        this.totalSec = source["totalSec"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Track {
	    id: number;
	    title: string;
	    artistId: number;
	    artistName: string;
	    albumId: number;
	    albumTitle: string;
	    durationSec: number;
	    format: string;
	    bitrate: number;
	    sourceType: string;
	    filePath: string;
	    externalId: string;
	    coverUrl: string;
	    isFavorite: boolean;
	    playCount: number;
	    isDownloaded: boolean;
	    downloadedAt: string;
	    globalPlays?: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artistId = source["artistId"];
	        this.artistName = source["artistName"];
	        this.albumId = source["albumId"];
	        this.albumTitle = source["albumTitle"];
	        this.durationSec = source["durationSec"];
	        this.format = source["format"];
	        this.bitrate = source["bitrate"];
	        this.sourceType = source["sourceType"];
	        this.filePath = source["filePath"];
	        this.externalId = source["externalId"];
	        this.coverUrl = source["coverUrl"];
	        this.isFavorite = source["isFavorite"];
	        this.playCount = source["playCount"];
	        this.isDownloaded = source["isDownloaded"];
	        this.downloadedAt = source["downloadedAt"];
	        this.globalPlays = source["globalPlays"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class PlaylistDetail {
	    playlist: Playlist;
	    tracks: Track[];
	
	    static createFrom(source: any = {}) {
	        return new PlaylistDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playlist = this.convertValues(source["playlist"], Playlist);
	        this.tracks = this.convertValues(source["tracks"], Track);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Source {
	    id: number;
	    name: string;
	    type: string;
	    path: string;
	    configJson: string;
	    trackCount: number;
	    enabled: boolean;
	    lastSyncedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.path = source["path"];
	        this.configJson = source["configJson"];
	        this.trackCount = source["trackCount"];
	        this.enabled = source["enabled"];
	        this.lastSyncedAt = source["lastSyncedAt"];
	    }
	}

}

export namespace jam {
	
	export class QueueItem {
	    trackId: number;
	    externalId: string;
	    title: string;
	    artist: string;
	    durationSec: number;
	    coverUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new QueueItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trackId = source["trackId"];
	        this.externalId = source["externalId"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.durationSec = source["durationSec"];
	        this.coverUrl = source["coverUrl"];
	    }
	}
	export class PlaybackSnapshot {
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
	    queue: QueueItem[];
	
	    static createFrom(source: any = {}) {
	        return new PlaybackSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trackId = source["trackId"];
	        this.externalId = source["externalId"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.coverUrl = source["coverUrl"];
	        this.isPlaying = source["isPlaying"];
	        this.positionSec = source["positionSec"];
	        this.durationSec = source["durationSec"];
	        this.volume = source["volume"];
	        this.muted = source["muted"];
	        this.shuffle = source["shuffle"];
	        this.repeatMode = source["repeatMode"];
	        this.queue = this.convertValues(source["queue"], QueueItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Status {
	    enabled: boolean;
	    running: boolean;
	    port: number;
	    lanIp: string;
	    remoteUrl: string;
	    apiUrl: string;
	    qrDataUrl: string;
	    connectedClients: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.running = source["running"];
	        this.port = source["port"];
	        this.lanIp = source["lanIp"];
	        this.remoteUrl = source["remoteUrl"];
	        this.apiUrl = source["apiUrl"];
	        this.qrDataUrl = source["qrDataUrl"];
	        this.connectedClients = source["connectedClients"];
	        this.error = source["error"];
	    }
	}

}

export namespace main {
	
	export class BootstrapData {
	    tracks: db.Track[];
	    albums: db.Album[];
	    artists: db.Artist[];
	    playlists: db.Playlist[];
	    sources: db.Source[];
	    history: db.PlayHistoryItem[];
	    preferences: Record<string, string>;
	    fireStreak: stats.FireStreakStatus;
	    listeningStats: stats.ListeningStats;
	    mediaBaseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tracks = this.convertValues(source["tracks"], db.Track);
	        this.albums = this.convertValues(source["albums"], db.Album);
	        this.artists = this.convertValues(source["artists"], db.Artist);
	        this.playlists = this.convertValues(source["playlists"], db.Playlist);
	        this.sources = this.convertValues(source["sources"], db.Source);
	        this.history = this.convertValues(source["history"], db.PlayHistoryItem);
	        this.preferences = source["preferences"];
	        this.fireStreak = this.convertValues(source["fireStreak"], stats.FireStreakStatus);
	        this.listeningStats = this.convertValues(source["listeningStats"], stats.ListeningStats);
	        this.mediaBaseUrl = source["mediaBaseUrl"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DynamicIslandQueueItem {
	    id: number;
	    externalId: string;
	    title: string;
	    artist: string;
	    coverUrl: string;
	    isFavorite: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DynamicIslandQueueItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.externalId = source["externalId"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.coverUrl = source["coverUrl"];
	        this.isFavorite = source["isFavorite"];
	    }
	}
	export class DynamicIslandSyncPayload {
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
	    trackLoading: boolean;
	    queueTitle: string;
	    queueIndex: number;
	    queue: DynamicIslandQueueItem[];
	
	    static createFrom(source: any = {}) {
	        return new DynamicIslandSyncPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.coverUrl = source["coverUrl"];
	        this.isPlaying = source["isPlaying"];
	        this.isFavorite = source["isFavorite"];
	        this.positionSec = source["positionSec"];
	        this.durationSec = source["durationSec"];
	        this.volume = source["volume"];
	        this.muted = source["muted"];
	        this.shuffle = source["shuffle"];
	        this.repeatMode = source["repeatMode"];
	        this.eqPreset = source["eqPreset"];
	        this.themeSeed = source["themeSeed"];
	        this.accentColor = source["accentColor"];
	        this.streakDays = source["streakDays"];
	        this.lyricsLoading = source["lyricsLoading"];
	        this.trackLoading = source["trackLoading"];
	        this.queueTitle = source["queueTitle"];
	        this.queueIndex = source["queueIndex"];
	        this.queue = this.convertValues(source["queue"], DynamicIslandQueueItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateCheckResult {
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    releaseUrl: string;
	    downloadUrl: string;
	    releaseNotes: string;
	    publishedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.releaseUrl = source["releaseUrl"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseNotes = source["releaseNotes"];
	        this.publishedAt = source["publishedAt"];
	    }
	}
	export class YTMusicAlbumView {
	    browseId: string;
	    kind: string;
	    title: string;
	    artist: string;
	    artistBrowseId: string;
	    year: string;
	    coverUrl: string;
	    subtitle: string;
	    tracks: db.Track[];
	    artistAlbums: sources.YTMusicCard[];
	
	    static createFrom(source: any = {}) {
	        return new YTMusicAlbumView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browseId = source["browseId"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.artistBrowseId = source["artistBrowseId"];
	        this.year = source["year"];
	        this.coverUrl = source["coverUrl"];
	        this.subtitle = source["subtitle"];
	        this.tracks = this.convertValues(source["tracks"], db.Track);
	        this.artistAlbums = this.convertValues(source["artistAlbums"], sources.YTMusicCard);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class YTMusicArtistView {
	    browseId: string;
	    name: string;
	    avatarUrl: string;
	    fansCount: number;
	    topTracks: db.Track[];
	    albums: sources.YTMusicCard[];
	    relatedArtists: sources.YTMusicCard[];
	
	    static createFrom(source: any = {}) {
	        return new YTMusicArtistView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browseId = source["browseId"];
	        this.name = source["name"];
	        this.avatarUrl = source["avatarUrl"];
	        this.fansCount = source["fansCount"];
	        this.topTracks = this.convertValues(source["topTracks"], db.Track);
	        this.albums = this.convertValues(source["albums"], sources.YTMusicCard);
	        this.relatedArtists = this.convertValues(source["relatedArtists"], sources.YTMusicCard);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace sources {
	
	export class DownloadProgress {
	    trackId: number;
	    externalId: string;
	    title: string;
	    artist: string;
	    status: string;
	    progress: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trackId = source["trackId"];
	        this.externalId = source["externalId"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.error = source["error"];
	    }
	}
	export class LyricLine {
	    timeSec: number;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new LyricLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeSec = source["timeSec"];
	        this.text = source["text"];
	    }
	}
	export class TrackLyrics {
	    trackId: number;
	    title: string;
	    artist: string;
	    plainLyrics: string;
	    syncedLyrics: LyricLine[];
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new TrackLyrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trackId = source["trackId"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.plainLyrics = source["plainLyrics"];
	        this.syncedLyrics = this.convertValues(source["syncedLyrics"], LyricLine);
	        this.source = source["source"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class YTMusicCard {
	    id: string;
	    title: string;
	    subtitle: string;
	    coverUrl: string;
	    kind: string;
	    videoId: string;
	    browseId: string;
	    playlistId: string;
	    artist: string;
	    album: string;
	    durationSec: number;
	    explicit?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new YTMusicCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.subtitle = source["subtitle"];
	        this.coverUrl = source["coverUrl"];
	        this.kind = source["kind"];
	        this.videoId = source["videoId"];
	        this.browseId = source["browseId"];
	        this.playlistId = source["playlistId"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.durationSec = source["durationSec"];
	        this.explicit = source["explicit"];
	    }
	}
	export class YTMusicTrack {
	    videoId: string;
	    title: string;
	    artist: string;
	    album: string;
	    durationSec: number;
	    coverUrl: string;
	    format: string;
	    playCountText?: string;
	    explicit?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new YTMusicTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.videoId = source["videoId"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.durationSec = source["durationSec"];
	        this.coverUrl = source["coverUrl"];
	        this.format = source["format"];
	        this.playCountText = source["playCountText"];
	        this.explicit = source["explicit"];
	    }
	}
	export class YTMusicShelf {
	    title: string;
	    items: YTMusicCard[];
	
	    static createFrom(source: any = {}) {
	        return new YTMusicShelf(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.items = this.convertValues(source["items"], YTMusicCard);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class YTMusicHomeFeed {
	    shelves: YTMusicShelf[];
	    tracks: YTMusicTrack[];
	    continuation?: string;
	
	    static createFrom(source: any = {}) {
	        return new YTMusicHomeFeed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.shelves = this.convertValues(source["shelves"], YTMusicShelf);
	        this.tracks = this.convertValues(source["tracks"], YTMusicTrack);
	        this.continuation = source["continuation"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace stats {
	
	export class DayBucket {
	    dayIndex: number;
	    label: string;
	    fullName: string;
	    minutes: number;
	    plays: number;
	
	    static createFrom(source: any = {}) {
	        return new DayBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dayIndex = source["dayIndex"];
	        this.label = source["label"];
	        this.fullName = source["fullName"];
	        this.minutes = source["minutes"];
	        this.plays = source["plays"];
	    }
	}
	export class FireStreakStatus {
	    currentStreak: number;
	    longestStreak: number;
	    todaySeconds: number;
	    todayMinutes: number;
	    todayTracks: number;
	    goalMinutes: number;
	    goalTracks: number;
	    streakMetToday: boolean;
	    progressPercent: number;
	    last30Days: db.DailyStreakRow[];
	
	    static createFrom(source: any = {}) {
	        return new FireStreakStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentStreak = source["currentStreak"];
	        this.longestStreak = source["longestStreak"];
	        this.todaySeconds = source["todaySeconds"];
	        this.todayMinutes = source["todayMinutes"];
	        this.todayTracks = source["todayTracks"];
	        this.goalMinutes = source["goalMinutes"];
	        this.goalTracks = source["goalTracks"];
	        this.streakMetToday = source["streakMetToday"];
	        this.progressPercent = source["progressPercent"];
	        this.last30Days = this.convertValues(source["last30Days"], db.DailyStreakRow);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HourBucket {
	    hour: number;
	    label: string;
	    minutes: number;
	    plays: number;
	
	    static createFrom(source: any = {}) {
	        return new HourBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hour = source["hour"];
	        this.label = source["label"];
	        this.minutes = source["minutes"];
	        this.plays = source["plays"];
	    }
	}
	export class TopStatItem {
	    name: string;
	    subText: string;
	    imageUrl: string;
	    plays: number;
	    minutes: number;
	
	    static createFrom(source: any = {}) {
	        return new TopStatItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.subText = source["subText"];
	        this.imageUrl = source["imageUrl"];
	        this.plays = source["plays"];
	        this.minutes = source["minutes"];
	    }
	}
	export class RangeBucket {
	    label: string;
	    minutes: number;
	    plays: number;
	
	    static createFrom(source: any = {}) {
	        return new RangeBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.minutes = source["minutes"];
	        this.plays = source["plays"];
	    }
	}
	export class ListeningStats {
	    timeOfDay: HourBucket[];
	    todayHours: HourBucket[];
	    weekSeries: RangeBucket[];
	    monthSeries: RangeBucket[];
	    yearSeries: RangeBucket[];
	    dayOfWeek: DayBucket[];
	    totalMinutes: number;
	    totalPlays: number;
	    uniqueTracks: number;
	    activeDays: number;
	    peakHourLabel: string;
	    peakPersona: string;
	    mostActiveDay: string;
	    topTracks: TopStatItem[];
	    topArtists: TopStatItem[];
	
	    static createFrom(source: any = {}) {
	        return new ListeningStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timeOfDay = this.convertValues(source["timeOfDay"], HourBucket);
	        this.todayHours = this.convertValues(source["todayHours"], HourBucket);
	        this.weekSeries = this.convertValues(source["weekSeries"], RangeBucket);
	        this.monthSeries = this.convertValues(source["monthSeries"], RangeBucket);
	        this.yearSeries = this.convertValues(source["yearSeries"], RangeBucket);
	        this.dayOfWeek = this.convertValues(source["dayOfWeek"], DayBucket);
	        this.totalMinutes = source["totalMinutes"];
	        this.totalPlays = source["totalPlays"];
	        this.uniqueTracks = source["uniqueTracks"];
	        this.activeDays = source["activeDays"];
	        this.peakHourLabel = source["peakHourLabel"];
	        this.peakPersona = source["peakPersona"];
	        this.mostActiveDay = source["mostActiveDay"];
	        this.topTracks = this.convertValues(source["topTracks"], TopStatItem);
	        this.topArtists = this.convertValues(source["topArtists"], TopStatItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

