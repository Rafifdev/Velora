<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { cubicIn, cubicOut } from 'svelte/easing';
  import { api, groupActiveDownloads, isJamRemote, isYTVideoThumb, resolveCoverUrl, safeEventsOn, sendJamCommand } from './lib/api';
  import { VeloraAudioEngine, type NormalizeLevel, type Spatial8DSpeed } from './lib/audioEngine';
  import { getPrefBool, getPrefValue } from './lib/preferences';
  import { getPrefLang, tr } from './lib/i18n';
  import type {
    Album,
    Artist,
    AudioOutputDevice,
    DownloadGroup,
    DownloadProgress,
    FireStreakStatus,
    GroupedCompletedDownload,
    HomeFilter,
    JamCommand,
    JamStatus,
    ListeningStats,
    PlayHistoryItem,
    Playlist,
    PlaylistDetail,
    Source,
    Track,
    TrackLyrics,
    ViewRoute,
    YTMusicAlbumView,
    YTMusicArtistView,
    YTMusicCard,
    YTMusicShelf,
    YTMusicTrack
  } from './lib/types';

  import Titlebar from './components/Titlebar.svelte';
  import Icon from './components/Icon.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import RightPanel from './components/RightPanel.svelte';
  import PlayerBar from './components/PlayerBar.svelte';
  import MobileNowPlaying from './components/MobileNowPlaying.svelte';
  import MobileQueueSheet from './components/MobileQueueSheet.svelte';
  import SplashScreen from './components/SplashScreen.svelte';

  import HomeView from './views/HomeView.svelte';

  let tracks = $state<Track[]>([]);
  let trackLoading = $state(false);
  let networkToast = $state('');
  let networkToastIcon = $state<'wifi-slash' | 'check-circle'>('wifi-slash');
  let networkToastTimer: ReturnType<typeof setTimeout> | undefined;

  function showNetworkToast(msg: string, icon: 'wifi-slash' | 'check-circle' = 'wifi-slash') {
    if (networkToastTimer) clearTimeout(networkToastTimer);
    networkToastIcon = icon;
    networkToast = msg;
    networkToastTimer = setTimeout(() => {
      networkToast = '';
    }, 3200);
  }
  let downloadStates = $state<Record<number, DownloadProgress>>({});
  let jamStatus = $state<JamStatus>({
    enabled: false,
    running: false,
    port: 8899,
    lanIp: '127.0.0.1',
    remoteUrl: 'http://127.0.0.1:8899/jam',
    apiUrl: 'http://127.0.0.1:8899/api',
    qrDataUrl: '',
    connectedClients: 0
  });
  let jamModalOpen = $state(false);
  let jamLoading = $state(false);
  let jamConnected = $state(false);
  let isMobile = $state(false);
  let mobileShowQueue = $state(false);
  let mobileSheetMode = $state<'queue' | 'devices'>('queue');
  let mobileMenuOpen = $state(false);
  let mobileNowPlayingOpen = $state(false);
  let mainScrollEl: HTMLElement | undefined = $state();
  let ytAlbumDetail = $state<YTMusicAlbumView | null>(null);
  let ytAlbumLoading = $state(false);
  let ytAlbumReqSeq = 0;
  let ytArtistDetail = $state<YTMusicArtistView | null>(null);
  let ytArtistLoading = $state(false);
  let ytArtistReqSeq = 0;
  let homeFilter = $state<HomeFilter>('semua');
  let ytLoadingMorePopularArtists = $state(false);
  const mobileNavTab = $derived.by(() => {
    if (mobileShowQueue) return 'queue';
    if (currentRoute.name === 'home') return 'home';
    if (currentRoute.name === 'search') return 'search';
    if (
      currentRoute.name === 'library' ||
      currentRoute.name === 'liked-songs' ||
      currentRoute.name === 'downloaded' ||
      currentRoute.name === 'favorite-albums' ||
      currentRoute.name === 'favorite-artists'
    ) {
      return 'library';
    }
    if (
      currentRoute.name === 'playlists' ||
      currentRoute.name === 'local-playlists' ||
      currentRoute.name === 'playlist-detail'
    ) {
      return 'playlists';
    }
    return '';
  });

  const SPLASH_SEEN_KEY = 'velora.splashShownOnce';
  let showSplash = $state(
    isJamRemote
      ? (typeof localStorage !== 'undefined' ? localStorage.getItem(SPLASH_SEEN_KEY) !== '1' : false)
      : true
  );
  let sessionDeletedDownloadTracks = $state<Track[]>([]);
  const activeDownloadsList = $derived(
    Object.values(downloadStates).filter(
      (d) => d.status === 'queued' || d.status === 'downloading' || d.status === 'paused' || d.status === 'failed'
    )
  );
  const activeGroupKeys = $derived(
    new Set(activeDownloadsList.map((d) => (d.groupKey || '').trim()).filter(Boolean))
  );
  const downloadedGroupKeys = $derived.by(() => {
    const keys = new Set<string>();
    for (const t of tracks) {
      if (t.isDownloaded && t.downloadGroupKey && !activeGroupKeys.has(t.downloadGroupKey)) {
        keys.add(t.downloadGroupKey);
      }
    }
    return keys;
  });
  const groupedActiveDownloads = $derived(groupActiveDownloads(activeDownloadsList));
  const activeDownloadsCount = $derived(
    groupedActiveDownloads.filter((d) => d.status === 'queued' || d.status === 'downloading').length
  );
  const downloadProgressPct = $derived.by(() => {
    const active = groupedActiveDownloads.filter((d) => d.status === 'queued' || d.status === 'downloading');
    if (active.length === 0) return 0;
    const sum = active.reduce((acc, d) => acc + Math.max(0, Math.min(100, d.progress || 0)), 0);
    return Math.round(sum / active.length);
  });
  let albums = $state<Album[]>([]);
  let artists = $state<Artist[]>([]);
  let playlists = $state<Playlist[]>([]);
  let playlistsLoading = $state(true);
  let playlistDetailLoading = $state(false);
  let sources = $state<Source[]>([]);
  let history = $state<PlayHistoryItem[]>([]);
  let preferences = $state<Record<string, string>>({});
  const appLang = $derived(getPrefLang(preferences));
  const lockLeftHidden = $derived(getPrefBool(preferences, 'layout.hideLeft'));
  const lockRightHidden = $derived(getPrefBool(preferences, 'layout.hideRight'));
  let mediaBaseUrl = $state('');

  let fireStreak = $state<FireStreakStatus>({
    currentStreak: 0,
    longestStreak: 0,
    todaySeconds: 0,
    todayMinutes: 0,
    todayTracks: 0,
    goalMinutes: 10,
    goalTracks: 3,
    streakMetToday: false,
    progressPercent: 0,
    last30Days: []
  });

  let listeningStats = $state<ListeningStats>({
    timeOfDay: [],
    dayOfWeek: [],
    totalMinutes: 0,
    totalPlays: 0,
    uniqueTracks: 0,
    peakHourLabel: '-',
    peakPersona: 'Belum Ada Data',
    mostActiveDay: '-',
    topTracks: [],
    topArtists: []
  });

  let navStack = $state<ViewRoute[]>([{ name: 'home' }]);
  let navIndex = $state(0);
  const currentRoute = $derived(navStack[navIndex] || { name: 'home' });
  const activeSidebarSection = $derived.by(() => {
    for (let i = navIndex; i >= 0; i--) {
      const r = navStack[i];
      if (!r) continue;
      if (i > 0 && navStack[i - 1]?.name === 'downloads') {
        if (r.name === 'playlist-detail' || (r.name === 'yt-album-detail' && r.kind === 'playlist')) {
          return 'playlists';
        }
        if (
          r.name === 'album-detail' ||
          (r.name === 'yt-album-detail' && r.kind !== 'playlist') ||
          (r.name === 'track-detail' && !r.singleTrack)
        ) {
          return 'favorite-albums';
        }
        if (r.name === 'track-detail' && r.singleTrack) {
          return 'liked-songs';
        }
      }
      if (r.name === 'home') return 'home';
      if (r.name === 'favorite-albums') return 'favorite-albums';
      if (r.name === 'liked-songs') return 'liked-songs';
      if (r.name === 'favorite-artists') return 'favorite-artists';
      if (r.name === 'playlists' || r.name === 'local-playlists') return 'playlists';
      if (r.name === 'history') return 'history';
      if (r.name === 'listening-stats' || r.name === 'fire-streak') return 'listening-stats';
      if (r.name === 'preferences') return 'preferences';
    }
    return 'home';
  });
  const routeTransitionKey = $derived.by(() => {
    const r = currentRoute;
    if (r.name === 'artist-detail') return `artist-detail:${r.artistId ?? r.browseId ?? r.artistName ?? ''}`;
    if (r.name === 'playlist-detail') return `playlist-detail:${r.playlistId}`;
    if (r.name === 'album-detail') return `album-detail:${r.albumId}`;
    if (r.name === 'track-detail') return `track-detail:${r.trackId}`;
    if (r.name === 'yt-album-detail') return `yt-album-detail:${r.browseId}:${r.kind ?? 'album'}`;
    return r.name;
  });
  const isFullHeightRoute = $derived(
    currentRoute.name !== 'home' && currentRoute.name !== 'sources'
  );

  function routeExitFade(node: HTMLElement, { duration = 180, shiftX = 0 }: { duration?: number; shiftX?: number } = {}) {
    const top = node.offsetTop;
    const clipHeight = mainScrollEl?.clientHeight || node.parentElement?.clientHeight || 0;
    node.style.position = 'absolute';
    node.style.top = `${top}px`;
    node.style.left = '0';
    node.style.right = '0';
    if (clipHeight > 0) {
      node.style.height = `${clipHeight}px`;
    }
    node.style.overflow = 'hidden';
    node.style.pointerEvents = 'none';
    node.style.zIndex = '0';
    return {
      duration,
      easing: cubicOut,
      css: (t: number, u: number) => `opacity: ${t}; transform: translate3d(${(shiftX * u).toFixed(2)}px, ${(u * -4).toFixed(2)}px, 0);`
    };
  }

  let selectedPlaylistDetail = $state<PlaylistDetail | null>(null);
  let detailTracks = $state<Track[]>([]);
  let searchQuery = $state('');
  let ytShelves = $state<YTMusicShelf[]>([]);
  let ytResults = $state<YTMusicTrack[]>([]);
  let ytPlaylistResults = $state<YTMusicCard[]>([]);
  let ytAlbumResults = $state<YTMusicCard[]>([]);
  let ytArtistResults = $state<YTMusicCard[]>([]);
  let ytLoading = $state(false);
  let ytHomeLoading = $state(true);
  let ytHomeContinuation = $state('');
  let ytLoadingMore = $state(false);
  let ytMorePage = $state(0);
  let ytMoreAllBuffer: YTMusicCard[] = [];
  let ytMoreMusicBuffer: YTMusicTrack[] = [];
  let ytMoreAlbumBuffer: YTMusicCard[] = [];
  let ytMoreArtistBuffer: YTMusicCard[] = [];
  let ytMorePlaylistBuffer: YTMusicCard[] = [];
  let ytMorePopularArtistsBuffer: YTMusicCard[] = [];
  let loadingCollectionId = $state<string | null>(null);
  let playRequestSeq = 0;
  let switchingTrack = false;
  let pendingResumeSeek = 0;
  let silentTailTicks = 0;
  let lastQueueSaveAt = 0;

  function getRadioLimit(): number {
    const val = Number(getPrefValue(preferences, 'playback.radioSize'));
    if (val === 0) return 50;
    return Number.isFinite(val) && val > 0 ? val : 20;
  }

  function saveLastQueue(force = false): void {
    if (!currentTrack || queue.length === 0) return;
    const now = Date.now();
    if (!force && now - lastQueueSaveAt < 2500) return;
    lastQueueSaveAt = now;
    try {
      const idx = Math.max(0, resolveActiveQueueIndex());
      const start = idx >= 40 && queue.length > 50 ? Math.min(idx - 10, queue.length - 50) : 0;
      const slicedQueue = queue.slice(start, start + 50);
      const slicedIdx = Math.max(0, Math.min(slicedQueue.length - 1, idx - start));
      localStorage.setItem('velora.lastQueue', JSON.stringify({
        queue: slicedQueue,
        index: slicedIdx,
        positionSec: Math.floor(positionSec || 0),
        queueSourceTitle,
        queueSourceKey
      }));
    } catch {
    }
  }
  let sourcesStatus = $state('');

  let leftWidth = $state(212);
  let rightWidth = $state(328);
  let leftCollapsed = $state(false);
  let rightCollapsed = $state(false);
  let rightPanelMode = $state<'now-playing' | 'queue' | 'devices'>('now-playing');
  let currentQueueIndex = $state(-1);
  let lastDIQueueOffset = 0;

  function resolveActiveQueueIndex(tr: Track | null = currentTrack, q: Track[] = queue): number {
    if (!tr || q.length === 0) return -1;
    if (tr.historyEntryId != null) {
      const hIdx = q.findIndex((item) => item.historyEntryId === tr.historyEntryId);
      if (hIdx >= 0) return hIdx;
    }
    if (currentQueueIndex >= 0 && currentQueueIndex < q.length && trackQueueKey(q[currentQueueIndex]) === trackQueueKey(tr)) {
      return currentQueueIndex;
    }
    if (tr.id > 0 || tr.externalId) {
      return q.findIndex((item) => trackQueueKey(item) === trackQueueKey(tr));
    }
    return -1;
  }
  let audioOutputs = $state<AudioOutputDevice[]>([]);
  let switchingOutputId = $state<string | null>(null);
  let focusRightSignal = $state<{ card: 'album' | 'lyrics' | 'queue'; seq: number } | null>(null);
  let activePrefTab = $state<string>('appearance');
  let prefSearch = $state('');
  let prefSavedToast = $state('');
  let seekDeltaSec = $state(0);
  let seekDeltaSeq = $state(0);
  let seekDeltaTimer: ReturnType<typeof setTimeout> | null = null;

  let showStreakModal = $state(false);
  let showCreatePlaylistModal = $state(false);
  let createPlaylistIsLocal = $state(true);
  let ytTop100Tracks = $state<YTMusicTrack[]>([]);
  let ytTop100Loading = $state(false);
  let lastRecPlaylistId = $state<number | null>(null);

  async function loadPlaylistRecommendations(excludeIds: string[] = []) {
    ytTop100Loading = true;
    try {
      const res = await api.getYTMusicTop100Recommended(excludeIds);
      ytTop100Tracks = res ?? [];
    } catch (err) {
      console.error('Failed to load YT Music Top 100:', err);
    } finally {
      ytTop100Loading = false;
    }
  }

  $effect(() => {
    if (currentRoute.name === 'playlist-detail') {
      const pid = currentRoute.playlistId;
      if (pid !== lastRecPlaylistId) {
        lastRecPlaylistId = pid;
        const exclude = selectedPlaylistDetail?.tracks.map((t) => t.externalId).filter(Boolean) ?? [];
        void loadPlaylistRecommendations(exclude);
      }
    } else {
      lastRecPlaylistId = null;
    }
  });

  let audioEl: HTMLAudioElement | undefined = $state();
  const audioEngine = new VeloraAudioEngine();

  $effect(() => {
    if (audioEl) {
      audioEngine.attach(audioEl);
    }
  });

  $effect(() => {
    audioEngine.setVolume(volume, muted);
  });

  $effect(() => {
    audioEngine.setPreset(getPrefValue(preferences, 'playback.eqPreset') || 'flat');
    audioEngine.setNormalize(
      getPrefBool(preferences, 'playback.audioNormalize'),
      (getPrefValue(preferences, 'playback.normalizeLevel') || 'normal') as NormalizeLevel
    );
    audioEngine.setSpatial8D(
      getPrefBool(preferences, 'playback.spatial8d'),
      (getPrefValue(preferences, 'playback.spatial8dSpeed') || 'normal') as Spatial8DSpeed
    );
  });

  $effect(() => {
    const hideTitle = getPrefBool(preferences, 'system.hideTitle');
    const nextTitle = hideTitle || !currentTrack
      ? 'Velora'
      : currentTrack.artistName
        ? `${currentTrack.title} - ${currentTrack.artistName}`
        : currentTrack.title;
    document.title = nextTitle;
    void api.setWindowTitle(nextTitle);
  });

  $effect(() => {
    const keepAwake = isPlaying && getPrefBool(preferences, 'system.keepAwake');
    void api.setKeepAwake(keepAwake);
  });

  $effect(() => {
    if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) return;
    const ms = navigator.mediaSession;
    const mediaKeysEnabled = getPrefBool(preferences, 'system.mediaKeys');
    if (!mediaKeysEnabled) {
      try {
        ms.setActionHandler('play', null);
        ms.setActionHandler('pause', null);
        ms.setActionHandler('previoustrack', null);
        ms.setActionHandler('nexttrack', null);
        ms.setActionHandler('seekto', null);
      } catch {
      }
      return;
    }

    if (currentTrack && typeof MediaMetadata !== 'undefined') {
      const artworkUrl = resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl);
      ms.metadata = new MediaMetadata({
        title: currentTrack.title,
        artist: currentTrack.artistName || 'Velora',
        album: currentTrack.albumTitle || '',
        artwork: artworkUrl ? [{ src: artworkUrl, sizes: '512x512', type: 'image/jpeg' }] : []
      });
    }
    ms.playbackState = isPlaying ? 'playing' : currentTrack ? 'paused' : 'none';

    try {
      ms.setActionHandler('play', () => {
        if (!isPlaying) void togglePlay();
      });
      ms.setActionHandler('pause', () => {
        if (isPlaying) void togglePlay();
      });
      ms.setActionHandler('previoustrack', () => void playPrev());
      ms.setActionHandler('nexttrack', () => void playNext());
      ms.setActionHandler('seekto', (details) => {
        if (typeof details.seekTime === 'number') {
          handleSeek(details.seekTime);
        }
      });
    } catch {
    }
  });
  let queue = $state<Track[]>([]);
  let queueSourceTitle = $state('');
  let queueSourceKey = $state('');
  let originalQueue: Track[] = [];
  let shufflePlayedKeys = new Set<string>();
  let shuffleHistory: Track[] = [];
  let playedAlgoKeys = new Set<string>();
  let algoBatchRound = 0;
  let currentTrack = $state<Track | null>(null);
  let currentLyrics = $state<TrackLyrics | null>(null);
  let lyricsLoading = $state(false);
  let lyricsLoadedTrackKey = '';
  let lyricsLoadedForDur = 0;
  let lyricsRequestSeq = 0;
  let isPlaying = $state(false);
  let positionSec = $state(0);
  let durationSec = $state(0);
  let volume = $state(80);
  let muted = $state(false);
  let shuffle = $state(false);
  let repeatMode = $state<'off' | 'all' | 'one'>('off');

  let listenedAccumulator = 0;
  let recordedForCurrentTrack = false;

  const accentColor = '#F6847E';

  let lyricOffsetVersion = $state(0);

  function getTrackLyricOffsetSec(tr: Track | null): number {
    if (!tr || typeof localStorage === 'undefined') return 0;
    const key = (tr.externalId || `${tr.title}::${tr.artistName}`).toLowerCase();
    if (!key) return 0;
    const saved = Number(localStorage.getItem(`velora.lyricOffset:${key}`) || '0');
    return Number.isFinite(saved) ? saved : 0;
  }

  const currentLyricOffsetSec = $derived.by(() => {
    void lyricOffsetVersion;
    return getTrackLyricOffsetSec(currentTrack);
  });

  const activeOutputDevice = $derived(
    audioOutputs.find((d) => d.isActive) ?? audioOutputs[0] ?? null
  );

  async function refreshAudioOutputs() {
    if (isJamRemote) return;
    try {
      const list = await api.getAudioOutputs();
      audioOutputs = list;
    } catch {
    }
  }

  async function handleSelectOutputDevice(deviceId: string) {
    if (!deviceId || switchingOutputId) return;
    switchingOutputId = deviceId;
    try {
      const ok = await api.setAudioOutput(deviceId);
      if (ok) {
        await refreshAudioOutputs();
      }
    } finally {
      switchingOutputId = null;
    }
  }

  $effect(() => {
    if (isJamRemote || rightCollapsed || rightPanelMode !== 'devices') return;
    void refreshAudioOutputs();
    const id = setInterval(() => {
      void refreshAudioOutputs();
    }, 2500);
    return () => clearInterval(id);
  });

  onMount(() => {
    const savedZoom = Number(localStorage.getItem('velora.zoom') || '1');
    if (Number.isFinite(savedZoom) && savedZoom >= 0.5 && savedZoom <= 1.5) {
      applyZoom(savedZoom, false);
    }
    const onWheel = (e: WheelEvent) => {
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        applyZoom(zoomLevel + (e.deltaY < 0 ? 0.05 : -0.05));
      }
    };
    window.addEventListener('wheel', onWheel, { passive: false });

    const hideTooltip = () => {
      if (tooltipTimer) {
        clearTimeout(tooltipTimer);
        tooltipTimer = null;
      }
      activeTooltipEl = null;
      tooltipVisible = false;
    };

    const showTooltipFor = (el: HTMLElement) => {
      const rawTitle = el.getAttribute('title');
      if (rawTitle) {
        el.setAttribute('data-tooltip', rawTitle);
        el.removeAttribute('title');
      }
      const text = (el.getAttribute('data-tooltip') || '').trim();
      if (!text) return;
      if (activeTooltipEl === el && tooltipVisible) return;
      activeTooltipEl = el;
      if (tooltipTimer) clearTimeout(tooltipTimer);
      tooltipTimer = setTimeout(() => {
        if (!activeTooltipEl || !document.contains(activeTooltipEl)) return;
        const rect = activeTooltipEl.getBoundingClientRect();
        if (rect.width === 0 && rect.height === 0) return;
        tooltipText = activeTooltipEl.getAttribute('data-tooltip') || '';
        const placeBelow = rect.top < 44;
        tooltipBelow = placeBelow;
        const pad = 12;
        tooltipX = Math.min(window.innerWidth - pad, Math.max(pad, rect.left + rect.width / 2));
        tooltipY = placeBelow ? rect.bottom + 8 : rect.top - 8;
        tooltipVisible = true;
      }, 1000);
    };

    const onMouseOver = (e: MouseEvent) => {
      const target = (e.target as HTMLElement | null)?.closest?.('[title], [data-tooltip]') as HTMLElement | null;
      if (!target) {
        hideTooltip();
        return;
      }
      showTooltipFor(target);
    };

    const onMouseOut = (e: MouseEvent) => {
      if (!activeTooltipEl) return;
      const related = e.relatedTarget as Node | null;
      if (related && activeTooltipEl.contains(related)) return;
      hideTooltip();
    };

    document.addEventListener('mouseover', onMouseOver, true);
    document.addEventListener('mouseout', onMouseOut, true);
    document.addEventListener('pointerdown', hideTooltip, true);
    document.addEventListener('scroll', hideTooltip, true);

    void Promise.all([refreshAll(), loadYTMusicHome()]);
    const offTaskbar = safeEventsOn('taskbar:control', (action: string) => {
      if (action === 'prev') {
        void playPrev();
      } else if (action === 'next') {
        void playNext();
      } else if (action === 'toggle') {
        void togglePlay();
      }
    });
    const offIslandSeek = safeEventsOn('island:seek', (sec: number) => {
      if (Number.isFinite(sec)) {
        handleSeek(sec);
      }
    });
    const offIslandAction = safeEventsOn('island:action', (action: string) => {
      if (action === 'toggle-favorite') {
        if (currentTrack && currentTrack.id > 0) {
          void handleToggleFavoriteTrack(currentTrack.id);
        }
      } else if (action === 'toggle-shuffle') {
        shuffle = !shuffle;
      } else if (action === 'cycle-repeat') {
        repeatMode = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
      } else if (action === 'toggle-mute') {
        handleToggleMute();
      } else if (action === 'vol-up') {
        const base = muted ? 0 : volume;
        handleVolumeChange(Math.min(100, base + 5));
      } else if (action === 'vol-down') {
        const base = muted ? 0 : volume;
        handleVolumeChange(Math.max(0, base - 5));
      } else if (action === 'cycle-eq') {
        const eqOrder = ['flat', 'spotify', 'bass', 'vocal', 'electronic'];
        const curEq = getPrefValue(preferences, 'playback.eqPreset') || 'flat';
        const nextEq = eqOrder[(eqOrder.indexOf(curEq) + 1) % eqOrder.length];
        void patchPrefs({ 'playback.eqPreset': nextEq });
      }
    });
    const offIslandPlayQueue = safeEventsOn('island:play-queue-index', (idx: number) => {
      if (Number.isInteger(idx) && idx >= 0 && idx < queue.length) {
        void playTrack(queue[idx], queue, queueSourceTitle, queueSourceKey);
      }
    });
    const offIslandFavId = safeEventsOn('island:toggle-favorite-id', (tid: number) => {
      if (Number.isInteger(tid) && tid > 0) {
        void handleToggleFavoriteTrack(tid);
      }
    });
    const onMediaDeviceChange = () => {
      api.notifyDynamicIslandAudioDevice('');
      void refreshAudioOutputs();
    };
    const onLyricOffsetChange = (ev: Event) => {
      lyricOffsetVersion += 1;
      const detail = (ev as CustomEvent<{ offsetSec?: number }>).detail;
      if (typeof detail?.offsetSec === 'number') {
        api.setDynamicIslandLyricOffset(detail.offsetSec);
      }
    };
    void refreshAudioOutputs();
    const onBeforeUnload = () => saveLastQueue(true);
    window.addEventListener('beforeunload', onBeforeUnload);
    window.addEventListener('velora:lyric-offset', onLyricOffsetChange);
    navigator.mediaDevices?.addEventListener?.('devicechange', onMediaDeviceChange);
    let dlRefreshTimer: ReturnType<typeof setTimeout> | null = null;
    const pendingCompletedIds = new Set<number>();
    const flushCompletedDownloadRefresh = () => {
      dlRefreshTimer = null;
      const doneIds = new Set(pendingCompletedIds);
      pendingCompletedIds.clear();
      void api.getTracks('all', '').then((list) => {
        tracks = list;
        if (doneIds.size === 0) return;
        const byId = new Map(list.map((t) => [t.id, t]));
        const byExt = new Map(list.filter((t) => t.externalId).map((t) => [t.externalId, t]));
        const matchTrack = (t: Track): Track =>
          doneIds.has(t.id)
            ? (byId.get(t.id) ?? (t.externalId ? byExt.get(t.externalId) : undefined) ?? t)
            : (t.externalId && byExt.has(t.externalId) && doneIds.has(byExt.get(t.externalId)!.id)
                ? byExt.get(t.externalId)!
                : t);
        detailTracks = detailTracks.map(matchTrack);
        if (selectedPlaylistDetail) {
          selectedPlaylistDetail = {
            ...selectedPlaylistDetail,
            tracks: selectedPlaylistDetail.tracks.map(matchTrack)
          };
        }
        if (ytAlbumDetail) {
          ytAlbumDetail = {
            ...ytAlbumDetail,
            tracks: ytAlbumDetail.tracks.map(matchTrack)
          };
        }
        if (ytArtistDetail) {
          ytArtistDetail = {
            ...ytArtistDetail,
            topTracks: ytArtistDetail.topTracks.map(matchTrack)
          };
        }
      });
    };

    const offDownload = safeEventsOn('download:progress', (p: DownloadProgress) => {
      if (!p || !p.trackId) return;
      if (p.status === 'completed') {
        const next = { ...downloadStates };
        delete next[p.trackId];
        downloadStates = next;
        delete retriedDownloadGroups[p.trackId];
        sessionDeletedDownloadTracks = sessionDeletedDownloadTracks.filter(
          (d) => !(p.trackId > 0 && d.id === p.trackId) && !(Boolean(p.externalId) && d.externalId === p.externalId)
        );
        pendingCompletedIds.add(p.trackId);
        if (!dlRefreshTimer) {
          dlRefreshTimer = setTimeout(flushCompletedDownloadRefresh, 150);
        }
      } else if (p.status === 'cancelled') {
        const next = { ...downloadStates };
        delete next[p.trackId];
        downloadStates = next;
      } else if (p.status === 'failed') {
        const next = { ...downloadStates };
        delete next[p.trackId];
        downloadStates = next;
        recordFailedDownload(p);
      } else {
        downloadStates = { ...downloadStates, [p.trackId]: p };
      }
    });
    const offJamStatus = safeEventsOn('jam:status', (st: JamStatus) => {
      if (st) jamStatus = st;
    });
    const offJamCmd = safeEventsOn('jam:command', (cmd: JamCommand) => {
      if (cmd?.action) void handleJamCommand(cmd);
    });
    void api.getJamStatus().then((st) => {
      if (st) jamStatus = st;
    });

    let jamES: EventSource | null = null;
    if (isJamRemote) {
      jamES = new EventSource('/api/events');
      jamES.onopen = () => {
        jamConnected = true;
      };
      jamES.onerror = () => {
        jamConnected = false;
      };
      jamES.addEventListener('state', (ev: MessageEvent) => {
        jamConnected = true;
        try {
          const snap = JSON.parse(ev.data);
          isPlaying = !!snap.isPlaying;
          positionSec = Number(snap.positionSec) || 0;
          durationSec = Number(snap.durationSec) || 0;
          volume = Number(snap.volume ?? 80);
          muted = !!snap.muted;
          shuffle = !!snap.shuffle;
          repeatMode = (snap.repeatMode as 'off' | 'all' | 'one') || 'off';
          if (Array.isArray(snap.queue)) {
            queue = snap.queue.map((q: any) => ({
              id: q.trackId || 0,
              title: q.title || 'Untitled',
              artistId: 0,
              artistName: q.artist || 'Unknown Artist',
              albumId: 0,
              albumTitle: '',
              durationSec: q.durationSec || 0,
              format: '',
              bitrate: 0,
              sourceType: q.externalId ? 'ytmusic' : 'local',
              filePath: '',
              externalId: q.externalId || '',
              coverUrl: q.coverUrl || '',
              isFavorite: false,
              playCount: 0,
              createdAt: ''
            }));
          }
          if (snap.title) {
            const matched = tracks.find(
              (t) => (snap.trackId && t.id === snap.trackId) || (snap.externalId && t.externalId === snap.externalId)
            );
            const nextTr: Track = matched ?? {
              id: snap.trackId || 0,
              title: snap.title,
              artistId: 0,
              artistName: snap.artist || 'Unknown Artist',
              albumId: 0,
              albumTitle: snap.album || '',
              durationSec: snap.durationSec || 0,
              format: '',
              bitrate: 0,
              sourceType: snap.externalId ? 'ytmusic' : 'local',
              filePath: '',
              externalId: snap.externalId || '',
              coverUrl: snap.coverUrl || '',
              isFavorite: false,
              playCount: 0,
              createdAt: ''
            };
            const changed = !currentTrack || currentTrack.id !== nextTr.id || currentTrack.externalId !== nextTr.externalId || currentTrack.title !== nextTr.title;
            currentTrack = nextTr;
            if (changed) {
              void loadLyricsForTrack(nextTr);
            }
          }
        } catch {
        }
      });
    }

    const mq = window.matchMedia('(max-width: 767px)');
    const syncMobile = () => {
      isMobile = mq.matches;
      if (!isMobile) mobileShowQueue = false;
    };
    syncMobile();
    mq.addEventListener('change', syncMobile);

    return () => {
      mq.removeEventListener('change', syncMobile);
      window.removeEventListener('beforeunload', onBeforeUnload);
      window.removeEventListener('wheel', onWheel);
      document.removeEventListener('mouseover', onMouseOver, true);
      document.removeEventListener('mouseout', onMouseOut, true);
      document.removeEventListener('pointerdown', hideTooltip, true);
      document.removeEventListener('scroll', hideTooltip, true);
      hideTooltip();
      offTaskbar?.();
      offIslandSeek?.();
      offIslandAction?.();
      offIslandPlayQueue?.();
      offIslandFavId?.();
      window.removeEventListener('velora:lyric-offset', onLyricOffsetChange);
      navigator.mediaDevices?.removeEventListener?.('devicechange', onMediaDeviceChange);
      offDownload?.();
      offJamStatus?.();
      offJamCmd?.();
      jamES?.close();
    };
  });

  $effect(() => {
    if (isJamRemote) return;
    const tr = currentTrack;
    const playing = isPlaying;
    const cover = tr ? resolveCoverUrl(tr.coverUrl, mediaBaseUrl) : '';
    api.updateTaskbarState(tr?.title ?? '', tr?.artistName ?? '', cover, playing);
  });

  $effect(() => {
    if (isJamRemote) return;
    const tr = currentTrack;
    const cover = tr ? resolveCoverUrl(tr.coverUrl, mediaBaseUrl, 120) : '';
    const eqPreset = getPrefValue(preferences, 'playback.eqPreset') || 'flat';
    const qIdx = resolveActiveQueueIndex(tr, queue);
    const qStart = qIdx > 20 && queue.length > 80 ? Math.min(qIdx - 20, queue.length - 80) : 0;
    lastDIQueueOffset = qStart;
    const qSlice = queue.slice(qStart, qStart + 80);
    const qSliceIdx = qIdx >= 0 ? qIdx - qStart : -1;
    api.syncDynamicIsland({
      title: tr?.title ?? '',
      artist: tr?.artistName ?? '',
      album: tr?.albumTitle ?? '',
      coverUrl: cover,
      isPlaying,
      isFavorite: Boolean(tr?.isFavorite),
      positionSec: untrack(() => positionSec),
      durationSec: untrack(() => durationSec) || tr?.durationSec || 0,
      volume,
      muted,
      shuffle,
      repeatMode,
      eqPreset,
      themeSeed: 'monolith',
      accentColor,
      streakDays: fireStreak?.currentStreak ?? 0,
      lyricsLoading,
      trackLoading,
      lyricOffsetSec: currentLyricOffsetSec,
      queueTitle: queueSourceTitle || 'Queue',
      queueIndex: qSliceIdx,
      queue: qSlice.map((q) => ({
        id: q.id,
        externalId: q.externalId ?? '',
        title: q.title,
        artist: q.artistName,
        coverUrl: resolveCoverUrl(q.coverUrl, mediaBaseUrl, 96),
        isFavorite: Boolean(q.isFavorite)
      }))
    });
  });

  function shuffleAndSpreadByArtist<T>(
    items: T[],
    getArtist: (item: T) => string,
    tailArtists: string[] = []
  ): T[] {
    if (items.length <= 1) return items;
    const shuffled = [...items];
    for (let i = shuffled.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
    }
    const byArtist = new Map<string, T[]>();
    const artistOrder: string[] = [];
    for (const item of shuffled) {
      const ar = (getArtist(item) || '').trim().toLowerCase() || `__anon_${artistOrder.length}`;
      let list = byArtist.get(ar);
      if (!list) {
        list = [];
        byArtist.set(ar, list);
        artistOrder.push(ar);
      }
      list.push(item);
    }

    const out: T[] = [];
    const recent = tailArtists
      .map((a) => (a || '').trim().toLowerCase())
      .filter(Boolean)
      .slice(-3);

    while (out.length < shuffled.length) {
      let bestArtist = '';
      let bestCount = -1;

      for (const ar of artistOrder) {
        const rem = byArtist.get(ar)?.length || 0;
        if (rem === 0) continue;
        if (!recent.includes(ar) && rem > bestCount) {
          bestCount = rem;
          bestArtist = ar;
        }
      }
      if (!bestArtist) {
        const last = recent[recent.length - 1] || '';
        for (const ar of artistOrder) {
          const rem = byArtist.get(ar)?.length || 0;
          if (rem > 0 && ar !== last && rem > bestCount) {
            bestCount = rem;
            bestArtist = ar;
          }
        }
      }
      if (!bestArtist) {
        for (const ar of artistOrder) {
          if ((byArtist.get(ar)?.length || 0) > 0) {
            bestArtist = ar;
            break;
          }
        }
      }
      if (!bestArtist) break;

      const list = byArtist.get(bestArtist)!;
      out.push(list.shift()!);
      recent.push(bestArtist);
      if (recent.length > 3) recent.shift();
    }
    return out;
  }

  function isHideExplicitEnabled(): boolean {
    return getPrefBool(preferences, 'content.hideExplicit');
  }

  function isExplicitItem(item: { explicit?: boolean; title?: string } | null | undefined): boolean {
    if (!item) return false;
    return Boolean(item.explicit) || /\b(explicit)\b/i.test(item.title || '');
  }

  function filterExplicit<T extends { explicit?: boolean; title?: string }>(items: T[]): T[] {
    if (!isHideExplicitEnabled()) return items;
    return items.filter((item) => !isExplicitItem(item));
  }

  function isExcludedHomeShelfTitle(title: string): boolean {
    const low = (title || '').trim().toLowerCase();
    return (
      low.includes('covers & remixes') ||
      low.includes('covers and remixes') ||
      low.includes('cover & remix')
    );
  }

  function homeShelfPriority(title: string): number {
    const low = (title || '').trim().toLowerCase();
    if (low.includes('new release') || low.includes('rilis baru')) return 0;
    if (
      low.includes("today's indonesian hits") ||
      low.includes('today indonesian hits') ||
      low.includes('today indonesia hits') ||
      low.includes('hits indonesia hari ini') ||
      low.includes('hits indonesia')
    ) {
      return 1;
    }
    if (low.includes('all-time hits') || low.includes('all time hits') || low.includes('hits sepanjang masa')) {
      return 2;
    }
    if (low.includes('trending songs for you') || low.includes('lagu trending untuk anda')) return 3;
    if (low.includes('trending community playlist') || low.includes('playlist komunitas trending')) return 4;
    if (low.includes('song mixes') || low.includes('songs mixes') || low.includes('mix lagu')) return 5;
    if (low.includes('rain + feels')) return 6;
    if (low.includes('monsoon melancholy')) return 7;
    return 1000;
  }

  function normalizeHomeShelves(shelves: YTMusicShelf[]): YTMusicShelf[] {
    return shelves
      .filter((shelf) => !isExcludedHomeShelfTitle(shelf.title))
      .map((shelf) => ({
        ...shelf,
        items: filterExplicit(shelf.items || [])
      }))
      .filter((shelf) => shelf.items.length > 0)
      .sort((a, b) => homeShelfPriority(a.title) - homeShelfPriority(b.title));
  }

  async function loadYTMusicHome() {
    ytHomeLoading = true;
    ytMoreAllBuffer = [];
    ytMoreMusicBuffer = [];
    ytMoreAlbumBuffer = [];
    ytMoreArtistBuffer = [];
    ytMorePlaylistBuffer = [];
    ytMorePopularArtistsBuffer = [];
    try {
      const feed = await api.getYTMusicHome();
      ytHomeContinuation = (feed?.continuation || '').trim();
      ytShelves = normalizeHomeShelves(feed?.shelves || []);
      ytResults = filterExplicit(feed?.tracks || []);
    } catch {
      ytShelves = [];
      ytHomeContinuation = '';
    } finally {
      ytHomeLoading = false;
    }
  }

  let retriedDownloadGroups: Record<number, DownloadGroup> = {};

  function findKnownTrackMeta(trackId: number, externalId?: string): Track | undefined {
    const matches = (t: Track) =>
      (trackId > 0 && t.id === trackId) || (Boolean(externalId) && t.externalId === externalId);
    const candidates = [
      tracks.find(matches),
      selectedPlaylistDetail?.tracks.find(matches),
      ytAlbumDetail?.tracks.find(matches),
      ytArtistDetail?.topTracks.find(matches)
    ].filter((t): t is Track => Boolean(t));
    if (candidates.length === 0) return undefined;
    return candidates.find((t) => Boolean(t.coverUrl) && t.durationSec > 0) ?? candidates[0];
  }

  function recordFailedDownload(p: DownloadProgress) {
    const existing = findKnownTrackMeta(p.trackId, p.externalId);
    const retriedGrp = retriedDownloadGroups[p.trackId];
    const alreadyRecorded = sessionDeletedDownloadTracks.some(
      (d) =>
        (p.trackId > 0 && d.id === p.trackId) ||
        (Boolean(p.externalId) && d.externalId === p.externalId)
    );
    if (alreadyRecorded) return;
    const base: Track = existing ?? {
      id: p.trackId,
      title: p.title,
      artistId: 0,
      artistName: p.artist,
      albumId: 0,
      albumTitle: '',
      durationSec: 0,
      format: '',
      bitrate: 0,
      sourceType: 'ytmusic',
      filePath: '',
      externalId: p.externalId,
      coverUrl: '',
      isFavorite: false,
      playCount: 0,
      createdAt: new Date().toISOString()
    };
    sessionDeletedDownloadTracks = [
      {
        ...base,
        isDownloaded: false,
        deletedReason: 'failed',
        downloadGroupKey: p.groupKey || retriedGrp?.key || existing?.downloadGroupKey || '',
        downloadGroupTitle: p.groupTitle || retriedGrp?.title || existing?.downloadGroupTitle || '',
        downloadGroupCover: p.groupCover || retriedGrp?.cover || existing?.downloadGroupCover || '',
        downloadGroupKind: p.groupKind || retriedGrp?.kind || existing?.downloadGroupKind || ''
      },
      ...sessionDeletedDownloadTracks
    ];
  }

  async function syncRemoteDownloadStates() {
    try {
      const list = await api.getDownloadStates();
      const next: Record<number, DownloadProgress> = {};
      for (const item of list || []) {
        if (!item || !item.trackId) continue;
        if (item.status === 'failed') {
          recordFailedDownload(item);
          continue;
        }
        if (item.status !== 'completed' && item.status !== 'cancelled') {
          next[item.trackId] = item;
        }
      }
      downloadStates = next;
    } catch {
    }
  }

  $effect(() => {
    if (!isJamRemote) return;
    const hasRealActive = Object.values(downloadStates).some(
      (d) => d.status === 'queued' || d.status === 'downloading'
    );
    const shouldPoll = currentRoute.name === 'downloads' || hasRealActive;
    void syncRemoteDownloadStates();
    if (!shouldPoll) return;
    const timer = setInterval(() => {
      void syncRemoteDownloadStates();
    }, 1500);
    return () => clearInterval(timer);
  });

  async function handleRetryFailedDownload(entry: GroupedCompletedDownload) {
    const tr = entry?.primaryTrack;
    if (!tr) return;
    const idSet = new Set(entry.tracks.map((t) => t.id).filter((id) => id > 0));
    const extSet = new Set(entry.tracks.map((t) => t.externalId).filter(Boolean));
    sessionDeletedDownloadTracks = sessionDeletedDownloadTracks.filter(
      (d) => !(d.id > 0 && idSet.has(d.id)) && !(Boolean(d.externalId) && extSet.has(d.externalId))
    );
    const grp: DownloadGroup = {
      key: tr.downloadGroupKey || '',
      title: tr.downloadGroupTitle || '',
      cover: tr.downloadGroupCover || '',
      kind: tr.downloadGroupKind === 'album' ? 'album' : 'playlist'
    };
    if (tr.id > 0) {
      if (grp.key) {
        retriedDownloadGroups[tr.id] = grp;
      }
      showNetworkToast(appLang === 'id' ? 'Mencoba mengunduh ulang...' : 'Retrying download...', 'check-circle');
      await api.retryDownloadTrack(tr.id, grp);
      if (isJamRemote) void syncRemoteDownloadStates();
      return;
    }
    await handleDownloadTrack(tr);
  }

  async function handleDownloadTrack(tr: Track) {
    if (!tr) return;
    sessionDeletedDownloadTracks = sessionDeletedDownloadTracks.filter(
      (d) => !(tr.id > 0 && d.id === tr.id) && !(Boolean(tr.externalId) && d.externalId === tr.externalId)
    );
    showNetworkToast(appLang === 'id' ? 'Mengunduh lagu...' : 'Downloading track...', 'check-circle');
    if (tr.id > 0) {
      await api.downloadTrack(tr.id);
      if (isJamRemote) void syncRemoteDownloadStates();
      return;
    }
    if (tr.externalId) {
      const saved = await api.downloadYTMusicTrack({
        videoId: tr.externalId,
        title: tr.title,
        artist: tr.artistName,
        album: tr.albumTitle,
        durationSec: tr.durationSec,
        coverUrl: tr.coverUrl,
        format: tr.format || 'OPUS'
      });
      if (saved && saved.id > 0) {
        syncTrackAcrossViews(0, tr.externalId, (t) => ({ ...t, id: saved.id }));
      }
      if (isJamRemote) void syncRemoteDownloadStates();
    }
  }

  async function handleDownloadYTMusicTrack(yt: YTMusicTrack) {
    if (!yt?.videoId) return;
    await api.downloadYTMusicTrack(yt);
    if (isJamRemote) void syncRemoteDownloadStates();
  }

  async function handleRemoveDownload(trackId: number, rememberDeleted = true) {
    const existing = tracks.find((t) => t.id === trackId);
    syncTrackAcrossViews(trackId, existing?.externalId, (t) => ({ ...t, isDownloaded: false }));
    const updated = await api.removeDownloadedTrack(trackId);
    if (rememberDeleted && existing && !sessionDeletedDownloadTracks.some((d) => d.id === trackId)) {
      sessionDeletedDownloadTracks = [{ ...existing, deletedReason: 'deleted' }, ...sessionDeletedDownloadTracks];
    }
    syncTrackAcrossViews(trackId, updated?.externalId ?? existing?.externalId, () => updated);
  }

  async function handleRemoveDownloadGroup(groupTracks: Track[], rememberDeleted = true) {
    const ids = groupTracks.map((t) => t.id).filter((id) => id > 0);
    if (ids.length === 0) return;
    if (rememberDeleted) {
      const toRemember = groupTracks
        .filter((gt) => gt.id > 0 && !sessionDeletedDownloadTracks.some((d) => d.id === gt.id))
        .map((gt) => ({ ...gt, deletedReason: 'deleted' as const }));
      if (toRemember.length > 0) {
        sessionDeletedDownloadTracks = [...toRemember, ...sessionDeletedDownloadTracks];
      }
    }
    await api.removeDownloadedTracks(ids);
    const idSet = new Set(ids);
    const refreshed = await api.getTracks('all', '');
    tracks = refreshed;
    detailTracks = detailTracks.map((t) => (idSet.has(t.id) ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t));
    if (selectedPlaylistDetail) {
      selectedPlaylistDetail = {
        ...selectedPlaylistDetail,
        tracks: selectedPlaylistDetail.tracks.map((t) =>
          idSet.has(t.id) ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t
        )
      };
    }
    if (ytAlbumDetail) {
      ytAlbumDetail = {
        ...ytAlbumDetail,
        tracks: ytAlbumDetail.tracks.map((t) =>
          idSet.has(t.id) ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t
        )
      };
    }
  }

  async function handleDownloadCollection(group: DownloadGroup, list: Track[]) {
    const isAlreadyDownloaded = (t: Track): boolean => {
      if (t.isDownloaded) return true;
      return tracks.some(
        (trItem) =>
          trItem.isDownloaded &&
          ((t.id > 0 && trItem.id === t.id) || (Boolean(t.externalId) && trItem.externalId === t.externalId))
      );
    };
    const pending = list.filter((t) => !isAlreadyDownloaded(t) && (t.id > 0 || Boolean(t.externalId)));
    if (pending.length === 0) {
      if (list.length > 0 && list.some((t) => isAlreadyDownloaded(t))) {
        showNetworkToast(
          group.kind === 'playlist'
            ? tr(appLang, 'nav.playlistAlreadyDownloaded')
            : tr(appLang, 'nav.albumAlreadyDownloaded'),
          'check-circle'
        );
      }
      return;
    }
    const toSend = list.filter((t) => t.id > 0 || Boolean(t.externalId));
    const reDlSet = new Set(toSend.map((t) => t.id).filter((id) => id > 0));
    if (reDlSet.size > 0) {
      sessionDeletedDownloadTracks = sessionDeletedDownloadTracks.filter((d) => !reDlSet.has(d.id));
    }
    showNetworkToast(
      group.kind === 'playlist'
        ? tr(appLang, 'nav.playlistDownloadStarted')
        : tr(appLang, 'nav.albumDownloadStarted'),
      'check-circle'
    );
    const hasUnsaved = toSend.some((t) => t.id <= 0);
    await api.downloadCollection(group, toSend);
    if (hasUnsaved) {
      tracks = await api.getTracks('all', '');
    }
    if (isJamRemote) void syncRemoteDownloadStates();
  }

  async function handleCancelDownload(trackId: number) {
    const existing = tracks.find((t) => t.id === trackId);
    const st = downloadStates[trackId];
    await api.cancelDownload(trackId);
    const next = { ...downloadStates };
    delete next[trackId];
    downloadStates = next;
    const refreshed = await api.getTracks('all', '');
    tracks = refreshed;
    detailTracks = detailTracks.map((t) => (t.id === trackId ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t));
    if (selectedPlaylistDetail) {
      selectedPlaylistDetail = {
        ...selectedPlaylistDetail,
        tracks: selectedPlaylistDetail.tracks.map((t) =>
          t.id === trackId ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t
        )
      };
    }
    if (ytAlbumDetail) {
      ytAlbumDetail = {
        ...ytAlbumDetail,
        tracks: ytAlbumDetail.tracks.map((t) =>
          t.id === trackId ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t
        )
      };
    }
    if (existing && !sessionDeletedDownloadTracks.some((d) => d.id === trackId)) {
      sessionDeletedDownloadTracks = [
        { ...existing, isDownloaded: false, downloadGroupKey: '', deletedReason: 'canceled' },
        ...sessionDeletedDownloadTracks
      ];
    } else if (st && !sessionDeletedDownloadTracks.some((d) => d.id === trackId)) {
      sessionDeletedDownloadTracks = [
        {
          id: trackId,
          title: st.title,
          artistId: 0,
          artistName: st.artist,
          albumId: 0,
          albumTitle: '',
          durationSec: 0,
          format: '',
          bitrate: 0,
          sourceType: 'ytmusic',
          filePath: '',
          externalId: st.externalId,
          coverUrl: '',
          isFavorite: false,
          playCount: 0,
          isDownloaded: false,
          deletedReason: 'canceled',
          createdAt: new Date().toISOString()
        },
        ...sessionDeletedDownloadTracks
      ];
    }
  }

  async function handleCancelDownloadGroup(groupKey: string, trackIds: number[]) {
    const idSet = new Set(trackIds);
    const sampleState = Object.values(downloadStates).find(
      (st) => st.groupKey === groupKey || idSet.has(st.trackId)
    );
    const snapshotTracks = tracks.filter(
      (t) => (idSet.has(t.id) || t.downloadGroupKey === groupKey) && !sessionDeletedDownloadTracks.some((d) => d.id === t.id)
    );
    await api.cancelDownloadGroup(groupKey);
    const next: Record<number, DownloadProgress> = {};
    for (const [k, st] of Object.entries(downloadStates)) {
      if (st.groupKey === groupKey || idSet.has(st.trackId)) continue;
      next[Number(k)] = st;
    }
    downloadStates = next;
    const refreshed = await api.getTracks('all', '');
    tracks = refreshed;
    detailTracks = detailTracks.map((t) =>
      idSet.has(t.id) ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t
    );
    if (selectedPlaylistDetail) {
      selectedPlaylistDetail = {
        ...selectedPlaylistDetail,
        tracks: selectedPlaylistDetail.tracks.map((t) =>
          idSet.has(t.id) ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t
        )
      };
    }
    if (ytAlbumDetail) {
      ytAlbumDetail = {
        ...ytAlbumDetail,
        tracks: ytAlbumDetail.tracks.map((t) =>
          idSet.has(t.id) ? refreshed.find((r) => r.id === t.id) ?? { ...t, isDownloaded: false } : t
        )
      };
    }
    const cancelledTracks = snapshotTracks.map((t) => ({
      ...t,
      isDownloaded: false,
      deletedReason: 'canceled' as const,
      downloadGroupKey: groupKey,
      downloadGroupTitle: sampleState?.groupTitle || t.downloadGroupTitle || '',
      downloadGroupCover: sampleState?.groupCover || t.downloadGroupCover || '',
      downloadGroupKind: sampleState?.groupKind || t.downloadGroupKind || 'playlist'
    }));
    if (cancelledTracks.length > 0) {
      sessionDeletedDownloadTracks = [...cancelledTracks, ...sessionDeletedDownloadTracks];
    }
  }

  async function handlePauseDownload(trackId: number) {
    if (downloadStates[trackId]) {
      downloadStates = {
        ...downloadStates,
        [trackId]: { ...downloadStates[trackId], status: 'paused', speedBps: 0 }
      };
    }
    await api.pauseDownload(trackId);
  }

  async function handleResumeDownload(trackId: number) {
    if (downloadStates[trackId]) {
      downloadStates = {
        ...downloadStates,
        [trackId]: { ...downloadStates[trackId], status: 'queued' }
      };
    }
    await api.resumeDownload(trackId);
  }

  async function handlePauseDownloadGroup(groupKey: string) {
    const next: Record<number, DownloadProgress> = {};
    for (const [k, st] of Object.entries(downloadStates)) {
      next[Number(k)] =
        st.groupKey === groupKey
          ? { ...st, status: 'paused', speedBps: 0, groupSpeedBps: 0 }
          : st;
    }
    downloadStates = next;
    await api.pauseDownloadGroup(groupKey);
  }

  async function handleResumeDownloadGroup(groupKey: string) {
    const next: Record<number, DownloadProgress> = {};
    for (const [k, st] of Object.entries(downloadStates)) {
      next[Number(k)] =
        st.groupKey === groupKey && st.status === 'paused'
          ? { ...st, status: 'queued' }
          : st;
    }
    downloadStates = next;
    await api.resumeDownloadGroup(groupKey);
  }

  async function refreshAll() {
    const boot = await api.getBootstrapData();
    tracks = boot.tracks;
    albums = boot.albums;
    artists = boot.artists;
    playlists = boot.playlists;
    playlistsLoading = false;
    sources = boot.sources;
    history = boot.history;
    const normalizedPrefs = { ...boot.preferences };
    if (normalizedPrefs['content.country'] && normalizedPrefs['content.country'].includes(',')) {
      normalizedPrefs['content.country'] = normalizedPrefs['content.country'].split(',')[0].trim() || 'ID';
    }
    preferences = normalizedPrefs;
    fireStreak = boot.fireStreak;
    listeningStats = boot.listeningStats;
    mediaBaseUrl = boot.mediaBaseUrl;

    if (!currentTrack) {
      try {
        const raw = localStorage.getItem('velora.lastQueue');
        if (raw) {
          const saved = JSON.parse(raw) as { queue?: Track[]; index?: number; positionSec?: number; queueSourceTitle?: string; queueSourceKey?: string };
          if (Array.isArray(saved.queue) && saved.queue.length > 0) {
            const idx = typeof saved.index === 'number' && saved.index >= 0 && saved.index < saved.queue.length ? saved.index : 0;
            queue = saved.queue;
            currentQueueIndex = idx;
            queueSourceTitle = saved.queueSourceTitle || '';
            queueSourceKey = saved.queueSourceKey || inferQueueSourceKeyByTitle(queueSourceTitle);
            currentTrack = saved.queue[idx];
            const restorePos = getPrefBool(preferences, 'playback.resumeQueue') && typeof saved.positionSec === 'number' && saved.positionSec > 0
              ? saved.positionSec
              : 0;
            positionSec = restorePos;
            durationSec = currentTrack.durationSec || 0;
            pendingResumeSeek = restorePos;
            isPlaying = false;
            loadLyricsForTrack(currentTrack);
          }
        }
      } catch {
      }
    }

    if (!currentTrack && history.length > 0) {
      const h = history[0];
      const matched = tracks.find((t) => t.id === h.trackId) ?? {
        id: h.trackId,
        title: h.title,
        artistId: 0,
        artistName: h.artistName || 'Unknown Artist',
        albumId: 0,
        albumTitle: h.albumTitle || 'Unknown Album',
        durationSec: h.durationSec || 0,
        format: h.format || 'OPUS',
        bitrate: 160,
        sourceType: h.sourceType || 'ytmusic',
        filePath: '',
        externalId: '',
        coverUrl: h.coverUrl || '',
        isFavorite: false,
        playCount: 1,
        createdAt: h.playedAt || ''
      };
      currentTrack = matched;
      durationSec = matched.durationSec || 0;
      isPlaying = false;
      loadLyricsForTrack(matched);
    }

    if (queue.length === 0 && tracks.length > 0) {
      if (!currentTrack) {
        currentTrack = tracks[0];
        durationSec = tracks[0].durationSec;
        loadLyricsForTrack(tracks[0]);
      }
      queue = buildLocalAlgorithmQueue(currentTrack ?? tracks[0]);
      currentQueueIndex = 0;
      void populateYTMusicAlgorithmQueue(currentTrack ?? tracks[0]);
    }

    const lw = Number(preferences['layout.leftSidebarWidth']);
    if (Number.isFinite(lw) && lw >= 212 && lw <= 380) leftWidth = lw;
    const rw = Number(preferences['layout.rightSidebarWidth']);
    if (Number.isFinite(rw) && rw >= 264 && rw <= 520) rightWidth = rw;
    if (preferences['layout.leftCollapsed'] !== undefined) {
      leftCollapsed = preferences['layout.leftCollapsed'] === 'true';
    }
    if (preferences['layout.rightCollapsed'] !== undefined) {
      rightCollapsed = preferences['layout.rightCollapsed'] === 'true';
    }
    if (preferences['dynamicIsland.enabled'] !== undefined) {
      api.setDynamicIslandEnabled(preferences['dynamicIsland.enabled'] !== 'false');
    }
    api.setDynamicIslandLanguage(getPrefValue(preferences, 'content.language') || 'id');
    if (preferences['dynamicIsland.style'] !== undefined) {
      api.setDynamicIslandStyle(preferences['dynamicIsland.style']);
    }
    api.setDynamicIslandShapeSize(
      getPrefValue(preferences, 'dynamicIsland.compactStyle') || 'RINGKAS',
      getPrefValue(preferences, 'dynamicIsland.size') || 'medium',
      Number(getPrefValue(preferences, 'dynamicIsland.opacity') || '100')
    );
    api.setDynamicIslandHideWhenAppActive(getPrefValue(preferences, 'dynamicIsland.hideApp') !== 'false');
    api.setDynamicIslandHideInFullscreen(getPrefValue(preferences, 'dynamicIsland.hideFull') !== 'false');
    api.setDynamicIslandIdleHideSeconds(Number(getPrefValue(preferences, 'dynamicIsland.idleHideSec') || '0') || 0);
    api.setDynamicIslandScreen(getPrefValue(preferences, 'dynamicIsland.screen') || 'primary');
    api.setDynamicIslandAnimSpeed(getPrefValue(preferences, 'dynamicIsland.animSpeed') || 'normal');
    api.setDynamicIslandOpenTrigger(getPrefValue(preferences, 'dynamicIsland.openTrigger') || 'hover');
    api.setDynamicIslandOnlyPlaying(getPrefValue(preferences, 'dynamicIsland.onlyPlaying') === 'true');
    api.setDynamicIslandPauseHideSeconds(Number(getPrefValue(preferences, 'dynamicIsland.pauseHide') || '0') || 0);
    api.setDynamicIslandContent(
      getPrefValue(preferences, 'dynamicIsland.showStatus') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showCover') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showClosedCover') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showArtist') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showLyric') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showHeart') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showDots') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showProgress') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.showClosedProgress') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.progressStyle') || 'thin'
    );
    api.setDynamicIslandCoverOptions(
      getPrefValue(preferences, 'dynamicIsland.coverStyle') || 'square',
      getPrefValue(preferences, 'dynamicIsland.coverSpin') !== 'false',
      getPrefValue(preferences, 'dynamicIsland.closedCoverSpin') === 'true'
    );
    api.setDynamicIslandLyricSweep(getPrefValue(preferences, 'dynamicIsland.lyricSweep') !== 'false');
    if (preferences['general.zoomLevel']) {
      const z = Number(preferences['general.zoomLevel']);
      if (Number.isFinite(z)) applyZoom(z, false);
    }

    if (preferences['volume']) {
      const v = Number(preferences['volume']);
      if (Number.isFinite(v)) volume = v;
    }
    const startVol = Number(getPrefValue(preferences, 'playback.startVolume'));
    if (Number.isFinite(startVol) && startVol >= 0) {
      volume = startVol;
    }
    if (getPrefBool(preferences, 'playback.shuffleOnStart')) {
      shuffle = true;
    }
    audioEngine.setVolume(volume, muted);

    const startPage = getPrefValue(preferences, 'appearance.startPage');
    if (startPage === 'explore') {
      homeFilter = 'musik';
      void navigate({ name: 'home', filter: 'musik' });
    } else if (startPage === 'library') {
      void navigate({ name: 'liked-songs' });
    }
  }

  $effect(() => {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    const perf = getPrefValue(preferences, 'appearance.performance') || 'full';
    root.dataset.perf = perf;
    root.dataset.shade = getPrefValue(preferences, 'appearance.shade') || 'soft';
    root.dataset.animations = perf === 'lite' ? 'off' : 'on';
  });

  async function patchPrefs(patch: Record<string, string>) {
    preferences = { ...preferences, ...patch };
    await api.savePreferences(patch);
    if (patch['content.language'] !== undefined) {
      api.setDynamicIslandLanguage(patch['content.language']);
    }
    if (
      patch['content.country'] !== undefined ||
      patch['content.hideExplicit'] !== undefined ||
      patch['content.language'] !== undefined
    ) {
      if (patch['content.country'] !== undefined || patch['content.language'] !== undefined) {
        // Mirror Auralys PageMemory.clear() + HomeMemory.clear() on locale change
        ytShelves = [];
        ytResults = [];
        ytPlaylistResults = [];
        ytAlbumResults = [];
        ytArtistResults = [];
        ytTop100Tracks = [];
        ytMoreAllBuffer = [];
        ytHomeContinuation = '';
        ytMorePage = 0;
      }
      void loadYTMusicHome();
    }
    if (patch['layout.leftCollapsed'] !== undefined) {
      leftCollapsed = patch['layout.leftCollapsed'] === 'true';
    }
    if (patch['layout.rightCollapsed'] !== undefined) {
      rightCollapsed = patch['layout.rightCollapsed'] === 'true';
    }
    if (patch['dynamicIsland.enabled'] !== undefined) {
      api.setDynamicIslandEnabled(patch['dynamicIsland.enabled'] !== 'false');
    }
    if (patch['playback.eqPreset'] !== undefined) {
      audioEngine.setPreset(patch['playback.eqPreset']);
    }
    if (patch['playback.audioNormalize'] !== undefined || patch['playback.normalizeLevel'] !== undefined) {
      audioEngine.setNormalize(
        getPrefBool(preferences, 'playback.audioNormalize'),
        (getPrefValue(preferences, 'playback.normalizeLevel') || 'normal') as NormalizeLevel
      );
    }
    if (patch['playback.spatial8d'] !== undefined || patch['playback.spatial8dSpeed'] !== undefined) {
      audioEngine.setSpatial8D(
        getPrefBool(preferences, 'playback.spatial8d'),
        (getPrefValue(preferences, 'playback.spatial8dSpeed') || 'normal') as Spatial8DSpeed
      );
    }
    if (patch['stream_quality'] !== undefined && audioEl && currentTrack && currentTrack.sourceType !== 'local') {
      const nextUrl = resolveDirectStreamUrl(currentTrack);
      if (nextUrl) {
        const resumeAt = audioEl.currentTime || 0;
        const wasPlaying = !audioEl.paused;
        switchingTrack = true;
        audioEl.pause();
        audioEl.src = nextUrl;
        audioEl.load();
        try {
          if (resumeAt > 0 && Number.isFinite(resumeAt)) {
            audioEl.currentTime = resumeAt;
          }
          if (wasPlaying) {
            await audioEngine.resume();
            await audioEl.play();
            isPlaying = true;
          }
        } catch {
        } finally {
          switchingTrack = false;
        }
      }
    }
    if (patch['dynamicIsland.style'] !== undefined) {
      api.setDynamicIslandStyle(patch['dynamicIsland.style']);
    }
    if (
      patch['dynamicIsland.compactStyle'] !== undefined ||
      patch['dynamicIsland.size'] !== undefined ||
      patch['dynamicIsland.opacity'] !== undefined
    ) {
      api.setDynamicIslandShapeSize(
        getPrefValue(preferences, 'dynamicIsland.compactStyle') || 'RINGKAS',
        getPrefValue(preferences, 'dynamicIsland.size') || 'medium',
        Number(getPrefValue(preferences, 'dynamicIsland.opacity') || '100')
      );
    }
    if (patch['dynamicIsland.hideApp'] !== undefined) {
      api.setDynamicIslandHideWhenAppActive(patch['dynamicIsland.hideApp'] !== 'false');
    }
    if (patch['dynamicIsland.hideFull'] !== undefined) {
      api.setDynamicIslandHideInFullscreen(patch['dynamicIsland.hideFull'] !== 'false');
    }
    if (patch['dynamicIsland.idleHideSec'] !== undefined) {
      api.setDynamicIslandIdleHideSeconds(Number(patch['dynamicIsland.idleHideSec']) || 0);
    }
    if (patch['dynamicIsland.screen'] !== undefined) {
      api.setDynamicIslandScreen(patch['dynamicIsland.screen']);
    }
    if (patch['dynamicIsland.animSpeed'] !== undefined) {
      api.setDynamicIslandAnimSpeed(patch['dynamicIsland.animSpeed']);
    }
    if (patch['dynamicIsland.openTrigger'] !== undefined) {
      api.setDynamicIslandOpenTrigger(patch['dynamicIsland.openTrigger']);
    }
    if (patch['dynamicIsland.onlyPlaying'] !== undefined) {
      api.setDynamicIslandOnlyPlaying(patch['dynamicIsland.onlyPlaying'] === 'true');
    }
    if (patch['dynamicIsland.pauseHide'] !== undefined) {
      api.setDynamicIslandPauseHideSeconds(Number(patch['dynamicIsland.pauseHide']) || 0);
    }
    if (
      patch['dynamicIsland.showStatus'] !== undefined ||
      patch['dynamicIsland.showCover'] !== undefined ||
      patch['dynamicIsland.showClosedCover'] !== undefined ||
      patch['dynamicIsland.showArtist'] !== undefined ||
      patch['dynamicIsland.showLyric'] !== undefined ||
      patch['dynamicIsland.showHeart'] !== undefined ||
      patch['dynamicIsland.showDots'] !== undefined ||
      patch['dynamicIsland.showProgress'] !== undefined ||
      patch['dynamicIsland.showClosedProgress'] !== undefined ||
      patch['dynamicIsland.progressStyle'] !== undefined
    ) {
      api.setDynamicIslandContent(
        getPrefValue(preferences, 'dynamicIsland.showStatus') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showCover') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showClosedCover') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showArtist') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showLyric') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showHeart') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showDots') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showProgress') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.showClosedProgress') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.progressStyle') || 'thin'
      );
    }
    if (patch['dynamicIsland.lyricSweep'] !== undefined) {
      api.setDynamicIslandLyricSweep(patch['dynamicIsland.lyricSweep'] !== 'false');
    }
    if (
      patch['dynamicIsland.coverStyle'] !== undefined ||
      patch['dynamicIsland.coverSpin'] !== undefined ||
      patch['dynamicIsland.closedCoverSpin'] !== undefined
    ) {
      api.setDynamicIslandCoverOptions(
        getPrefValue(preferences, 'dynamicIsland.coverStyle') || 'square',
        getPrefValue(preferences, 'dynamicIsland.coverSpin') !== 'false',
        getPrefValue(preferences, 'dynamicIsland.closedCoverSpin') === 'true'
      );
    }
    if (patch['general.zoomLevel'] !== undefined) {
      const z = Number(patch['general.zoomLevel']);
      if (Number.isFinite(z)) applyZoom(z, true);
    }
    if (patch['integrations.jam.enabled'] !== undefined) {
      if (patch['integrations.jam.enabled'] !== 'true') {
        jamModalOpen = false;
        if (jamStatus.running) {
          jamStatus = await api.stopJam();
        }
        preferences = {
          ...preferences,
          'integrations.jam.remoteUrl': ''
        };
      }
    }
  }

  async function toggleJamServer() {
    if (!getPrefBool(preferences, 'integrations.jam.enabled')) return;
    jamLoading = true;
    try {
      if (jamStatus.running) {
        jamStatus = await api.stopJam();
        preferences = {
          ...preferences,
          'integrations.jam.remoteUrl': ''
        };
      } else {
        jamStatus = await api.startJam();
        preferences = {
          ...preferences,
          'integrations.jam.remoteUrl': jamStatus.remoteUrl
        };
      }
    } finally {
      jamLoading = false;
    }
  }

  async function handleJamCommand(cmd: JamCommand) {
    switch (cmd.action) {
      case 'toggle':
        await togglePlay();
        break;
      case 'play':
        if (!isPlaying) await togglePlay();
        break;
      case 'pause':
        if (isPlaying) await togglePlay();
        break;
      case 'next':
        await playNext();
        break;
      case 'prev':
        await playPrev();
        break;
      case 'seek':
        handleSeek(Math.max(0, Number(cmd.value) || 0));
        break;
      case 'volume': {
        const v = Math.max(0, Math.min(100, Math.round(Number(cmd.value) || 0)));
        volume = v;
        if (v > 0 && muted) muted = false;
        audioEngine.setVolume(volume, muted);
        api.playerControl('volume', volume);
        void patchPrefs({ volume: String(volume) });
        break;
      }
      case 'mute':
        muted = !muted;
        audioEngine.setVolume(volume, muted);
        api.playerControl('mute', muted ? 1 : 0);
        break;
      case 'shuffle':
        shuffle = !shuffle;
        break;
      case 'repeat':
        repeatMode = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
        break;
      case 'play-index': {
        const idx = Math.round(Number(cmd.value) || 0);
        const realIdx = lastDIQueueOffset + idx;
        if (idx >= 0 && realIdx >= 0 && realIdx < queue.length) {
          await playTrack(queue[realIdx], queue, queueSourceTitle, queueSourceKey, false, realIdx);
        }
        break;
      }
      case 'remove-index': {
        const idx = Math.round(Number(cmd.value) || 0);
        const realIdx = lastDIQueueOffset + idx;
        if (idx >= 0 && realIdx >= 0 && realIdx < queue.length) {
          queue = queue.filter((_, i) => i !== realIdx);
          if (realIdx < currentQueueIndex) {
            currentQueueIndex -= 1;
          } else if (currentQueueIndex >= queue.length) {
            currentQueueIndex = queue.length - 1;
          }
        }
        break;
      }
      case 'play-track': {
        if (cmd.trackId && cmd.trackId > 0) {
          const found = tracks.find((t) => t.id === cmd.trackId);
          if (found) {
            await playTrack(found, queue.length > 0 ? queue : [found]);
            break;
          }
        }
        if (cmd.externalId) {
          await playYTMusicSingle({
            videoId: cmd.externalId,
            title: cmd.title || 'Untitled',
            artist: cmd.artist || 'Unknown Artist',
            album: cmd.album || 'Single',
            durationSec: cmd.durationSec || 180,
            coverUrl: cmd.coverUrl || '',
            format: 'OPUS'
          });
        }
        break;
      }
      case 'play-next': {
        let tr: Track | undefined = cmd.trackId && cmd.trackId > 0 ? tracks.find((t) => t.id === cmd.trackId) : undefined;
        if (!tr && cmd.externalId) {
          tr = await api.saveYTMusicTrack({
            videoId: cmd.externalId,
            title: cmd.title || 'Untitled',
            artist: cmd.artist || 'Unknown Artist',
            album: cmd.album || 'Single',
            durationSec: cmd.durationSec || 180,
            coverUrl: cmd.coverUrl || '',
            format: 'OPUS'
          });
        }
        if (tr) {
          handleQueuePlayNext(tr);
        }
        break;
      }
      case 'add-track': {
        let tr: Track | undefined = cmd.trackId && cmd.trackId > 0 ? tracks.find((t) => t.id === cmd.trackId) : undefined;
        if (!tr && cmd.externalId) {
          tr = await api.saveYTMusicTrack({
            videoId: cmd.externalId,
            title: cmd.title || 'Untitled',
            artist: cmd.artist || 'Unknown Artist',
            album: cmd.album || 'Single',
            durationSec: cmd.durationSec || 180,
            coverUrl: cmd.coverUrl || '',
            format: 'OPUS'
          });
        }
        if (tr && !queue.some((q) => q.id === tr!.id)) {
          queue = [...queue, tr];
        }
        break;
      }
    }
  }

  let lastJamSyncMs = 0;
  let lastJamSyncKey = '';
  $effect(() => {
    if (isJamRemote || !jamStatus.running) return;
    const tr = currentTrack;
    const structuralKey = `${tr?.id ?? 0}:${tr?.externalId ?? ''}:${isPlaying}:${volume}:${muted}:${shuffle}:${repeatMode}:${queue.length}`;
    const now = Date.now();
    const posRounded = Math.floor(positionSec);
    if (structuralKey === lastJamSyncKey && now - lastJamSyncMs < 450) {
      return;
    }
    lastJamSyncKey = structuralKey;
    lastJamSyncMs = now;
    api.updateJamState({
      trackId: tr?.id ?? 0,
      externalId: tr?.externalId ?? '',
      title: tr?.title ?? '',
      artist: tr?.artistName ?? '',
      album: tr?.albumTitle ?? '',
      coverUrl: tr?.coverUrl ?? '',
      isPlaying,
      positionSec: posRounded,
      durationSec: durationSec || tr?.durationSec || 0,
      volume,
      muted,
      shuffle,
      repeatMode,
      queue: queue.slice(0, 60).map((q) => ({
        trackId: q.id,
        externalId: q.externalId ?? '',
        title: q.title,
        artist: q.artistName,
        durationSec: q.durationSec,
        coverUrl: q.coverUrl ?? ''
      }))
    });
  });

  async function loadLyricsForTrack(tr: Track, overrideDurSec?: number, forceRefresh = false) {
    const targetKey = trackQueueKey(tr);
    const effDur = Math.round(overrideDurSec ?? tr.durationSec ?? 0);
    lyricsLoadedTrackKey = targetKey;
    lyricsLoadedForDur = effDur;
    const reqSeq = ++lyricsRequestSeq;
    lyricsLoading = true;
    try {
      const res = forceRefresh
        ? await api.refreshTrackLyrics(tr.id, tr.title, tr.artistName, tr.albumTitle, effDur)
        : await api.getTrackLyrics(tr.id, tr.title, tr.artistName, tr.albumTitle, effDur);
      if (reqSeq === lyricsRequestSeq && currentTrack && trackQueueKey(currentTrack) === targetKey) {
        currentLyrics = res;
      }
    } catch {
      if (reqSeq === lyricsRequestSeq && currentTrack && trackQueueKey(currentTrack) === targetKey) {
        currentLyrics = null;
      }
    } finally {
      if (reqSeq === lyricsRequestSeq && currentTrack && trackQueueKey(currentTrack) === targetKey) {
        lyricsLoading = false;
      }
    }
  }

  async function navigate(route: ViewRoute) {
    if (route.name === 'home' && !route.filter) {
      homeFilter = 'semua';
    }
    let resolvedRoute = route;
    if (isMobile) {
      if (route.name === 'liked-songs') resolvedRoute = { name: 'library', filter: 'music' };
      else if (route.name === 'downloaded') resolvedRoute = { name: 'library', filter: 'downloaded' };
      else if (route.name === 'favorite-albums') resolvedRoute = { name: 'library', filter: 'albums' };
      else if (route.name === 'favorite-artists') resolvedRoute = { name: 'library', filter: 'artists' };
    }
    const nextStack = navStack.slice(0, navIndex + 1);
    nextStack.push(resolvedRoute);
    navStack = nextStack;
    navIndex = nextStack.length - 1;
    await loadRouteData(resolvedRoute);
  }

  async function goBack() {
    if (navIndex <= 0) return;
    navIndex -= 1;
    await loadRouteData(navStack[navIndex]);
  }

  async function goForward() {
    if (navIndex >= navStack.length - 1) return;
    navIndex += 1;
    await loadRouteData(navStack[navIndex]);
  }

  async function loadRouteData(route: ViewRoute) {
    if (mainScrollEl) mainScrollEl.scrollTop = 0;
    if (route.name === 'playlist-detail') {
      playlistDetailLoading = true;
      selectedPlaylistDetail = null;
      try {
        selectedPlaylistDetail = await api.getPlaylistDetail(route.playlistId);
      } finally {
        playlistDetailLoading = false;
      }
    } else if (route.name === 'album-detail') {
      detailTracks = await api.getAlbumTracks(route.albumId);
    } else if (route.name === 'artist-detail') {
      const matched = artists.find(
        (a) =>
          (route.artistId !== undefined && a.id === route.artistId) ||
          (route.artistName && a.name.toLowerCase() === route.artistName.toLowerCase())
      );
      detailTracks = matched ? await api.getArtistTracks(matched.id) : [];
      const seq = ++ytArtistReqSeq;
      ytArtistDetail = null;
      ytArtistLoading = true;
      try {
        const res = await api.getYTMusicArtistDetail(
          route.browseId ?? matched?.externalId ?? '',
          route.artistName ?? matched?.name ?? '',
          route.avatarUrl ?? matched?.avatarUrl ?? ''
        );
        if (seq === ytArtistReqSeq) {
          ytArtistDetail = res;
        }
      } catch {
        if (seq === ytArtistReqSeq) ytArtistDetail = null;
      } finally {
        if (seq === ytArtistReqSeq) ytArtistLoading = false;
      }
    } else if (route.name === 'history') {
      history = await api.getPlayHistory();
    } else if (route.name === 'listening-stats') {
      listeningStats = await api.getListeningStats();
      fireStreak = await api.getFireStreak();
    } else if (route.name === 'fire-streak') {
      fireStreak = await api.getFireStreak();
    } else if (route.name === 'yt-album-detail') {
      const seq = ++ytAlbumReqSeq;
      if (route.localTracks && route.localTracks.length > 0) {
        ytAlbumDetail = {
          browseId: route.browseId ?? '',
          kind: route.kind,
          title: route.albumTitle || route.title,
          artist: route.artist,
          artistBrowseId: '',
          year: '',
          coverUrl: route.coverUrl,
          subtitle: '',
          tracks: route.localTracks,
          artistAlbums: []
        };
        ytAlbumLoading = false;
        return;
      }
      ytAlbumDetail = null;
      ytAlbumLoading = true;
      try {
        const res = await api.getYTMusicTrackAlbum(
          route.videoId,
          route.browseId ?? '',
          route.title,
          route.artist,
          route.albumTitle,
          route.coverUrl
        );
        if (seq === ytAlbumReqSeq) {
          ytAlbumDetail = res;
        }
      } catch (err) {
        console.error('Gagal memuat album YouTube Music:', err);
      } finally {
        if (seq === ytAlbumReqSeq) {
          ytAlbumLoading = false;
        }
      }
    }
  }

  async function openTrackArtistDetail(tr: Track) {
    const primaryArtist = (tr.artistName || '').split(',')[0].trim();
    if (!primaryArtist && !tr.artistId) return;
    await navigate({
      name: 'artist-detail',
      artistId: tr.artistId > 0 ? tr.artistId : undefined,
      artistName: primaryArtist
    });
  }

  async function openTrackAlbumDetail(tr: Track) {
    if (tr.sourceType === 'ytmusic' || tr.externalId) {
      await navigate({
        name: 'yt-album-detail',
        videoId: tr.externalId,
        title: tr.title,
        artist: tr.artistName,
        albumTitle: tr.albumTitle,
        coverUrl: tr.coverUrl
      });
    } else if (tr.albumId > 0) {
      await navigate({ name: 'album-detail', albumId: tr.albumId });
    } else {
      await navigate({ name: 'track-detail', trackId: tr.id });
    }
  }

  async function openYTAlbumCard(card: YTMusicCard) {
    let bID = card.browseId || '';
    if (card.kind !== 'song' && !bID && card.playlistId) {
      bID = card.playlistId.startsWith('VL') ? card.playlistId : `VL${card.playlistId}`;
    }
    const kind: 'album' | 'playlist' =
      card.kind === 'playlist' || bID.startsWith('VL') || bID.startsWith('PL') || bID.startsWith('RD')
        ? 'playlist'
        : 'album';
    await navigate({
      name: 'yt-album-detail',
      videoId: card.kind === 'playlist' ? '' : (card.videoId || ''),
      browseId: bID,
      kind,
      title: card.title,
      artist: card.artist || card.subtitle || '',
      albumTitle: card.kind === 'song' ? (card.album || card.title) : card.title,
      coverUrl: card.coverUrl
    });
  }

  async function openYTSongCard(yt: YTMusicTrack) {
    await navigate({
      name: 'yt-album-detail',
      videoId: yt.videoId,
      browseId: '',
      title: yt.title,
      artist: yt.artist,
      albumTitle: yt.album,
      coverUrl: yt.coverUrl
    });
  }

  function inferQueueSourceKeyByTitle(rawTitle: string): string {
    const clean = rawTitle.trim().toLowerCase();
    if (!clean) return '';
    const pl = playlists.find((p) => p.name.trim().toLowerCase() === clean);
    if (pl) return `local-playlist:${pl.id}`;
    const al = albums.find((a) => a.title.trim().toLowerCase() === clean);
    if (al) return `local-album:${al.id}`;
    const ar = artists.find((a) => a.name.trim().toLowerCase() === clean);
    if (ar) return `local-artist:${ar.id}`;
    if (clean === 'favorite music' || clean === 'lagu disukai') return 'liked-songs';
    return '';
  }

  async function openQueueSource() {
    const key = queueSourceKey.trim() || inferQueueSourceKeyByTitle(queueSourceTitle);
    if (!key) return;
    if (key.startsWith('local-playlist:')) {
      const id = Number(key.slice('local-playlist:'.length));
      if (Number.isFinite(id) && id > 0) {
        await navigate({ name: 'playlist-detail', playlistId: id });
      }
      return;
    }
    if (key.startsWith('local-album:')) {
      const id = Number(key.slice('local-album:'.length));
      if (Number.isFinite(id) && id > 0) {
        await handleOpenAlbum(id);
      }
      return;
    }
    if (key.startsWith('local-artist:')) {
      const id = Number(key.slice('local-artist:'.length));
      if (Number.isFinite(id) && id > 0) {
        await navigate({ name: 'artist-detail', artistId: id });
      }
      return;
    }
    if (key === 'liked-songs') {
      await navigate({ name: 'liked-songs' });
      return;
    }
    if (key === 'history') {
      await navigate({ name: 'history' });
      return;
    }
    if (key.startsWith('UC')) {
      await navigate({
        name: 'artist-detail',
        browseId: key,
        artistName: queueSourceTitle
      });
      return;
    }
    await navigate({
      name: 'yt-album-detail',
      videoId: currentTrack?.externalId || '',
      browseId: key,
      title: queueSourceTitle || currentTrack?.albumTitle || '',
      artist: currentTrack?.artistName || '',
      albumTitle: queueSourceTitle || currentTrack?.albumTitle || '',
      coverUrl: currentTrack?.coverUrl || ''
    });
  }

  let ytSearchSeq = 0;

  async function handleSearchSuggestions(q: string) {
    searchQuery = q;
    if (q.trim()) {
      await handleSearchYTMusic(q);
    }
  }

  async function handleGlobalSearch(q: string) {
    searchQuery = q;
    if (currentRoute.name !== 'search') {
      await navigate({ name: 'search', query: q });
    }
    if (q.trim()) {
      const startFilter = getPrefValue(preferences, 'content.searchStart') || 'song';
      if (startFilter === 'album') {
        await handleSearchYTMusicAlbums(q);
      } else if (startFilter === 'artist') {
        await handleSearchYTMusicArtists(q);
      } else if (startFilter === 'playlist') {
        await handleSearchYTMusicPlaylists(q);
      } else {
        await handleSearchYTMusic(q);
      }
    }
  }

  async function handleSearchYTMusic(q: string) {
    const seq = ++ytSearchSeq;
    ytLoading = true;
    if (!q.trim()) {
      ytResults = [];
    }
    try {
      const primary = filterExplicit((await api.searchYouTubeMusic(q)) || []);
      if (seq !== ytSearchSeq) return;
      if (!q.trim() || primary.length >= 20) {
        ytResults = !q.trim() ? shuffleAndSpreadByArtist(primary, (t) => t.artist || '') : primary;
        return;
      }
      const seen = new Set(primary.map((t) => t.videoId));
      const merged = [...primary];
      const extra = filterExplicit((await api.searchYouTubeMusic(`${q} populer`).catch(() => [])) || []);
      if (seq !== ytSearchSeq) return;
      for (const item of extra) {
        if (!item.videoId || seen.has(item.videoId)) continue;
        seen.add(item.videoId);
        merged.push(item);
      }
      ytResults = merged;
    } finally {
      if (seq === ytSearchSeq) {
        ytLoading = false;
      }
    }
  }

  async function handleSearchYTMusicPlaylists(q: string) {
    const seq = ++ytSearchSeq;
    ytLoading = true;
    try {
      const primary = await api.searchYouTubeMusicPlaylists(q);
      if (seq !== ytSearchSeq) return;
      const seen = new Set<string>();
      const merged: YTMusicCard[] = [];
      const pushUnique = (card: YTMusicCard) => {
        const key = (card.browseId || card.playlistId || card.id || '').trim();
        if (!key || seen.has(key)) return;
        seen.add(key);
        merged.push(card);
      };
      for (const c of primary || []) pushUnique(c);
      if (merged.length < 24) {
        const extra = await api.searchYouTubeMusicPlaylists(q.trim() ? `${q} mix` : 'Top Hits Playlist').catch(() => []);
        if (seq !== ytSearchSeq) return;
        for (const c of extra || []) pushUnique(c);
      }
      if (merged.length < 24 && !q.trim()) {
        for (const shelf of ytShelves) {
          for (const card of shelf.items) {
            if (card.kind === 'playlist') pushUnique(card);
          }
        }
      }
      ytPlaylistResults = merged;
    } catch {
      if (seq === ytSearchSeq) {
        ytPlaylistResults = [];
      }
    } finally {
      if (seq === ytSearchSeq) {
        ytLoading = false;
      }
    }
  }

  async function handleSearchYTMusicAlbums(q: string) {
    const seq = ++ytSearchSeq;
    ytLoading = true;
    try {
      const res = await api.searchYouTubeMusicAlbums(q);
      if (seq === ytSearchSeq) {
        ytAlbumResults = res || [];
      }
    } catch {
      if (seq === ytSearchSeq) {
        ytAlbumResults = [];
      }
    } finally {
      if (seq === ytSearchSeq) {
        ytLoading = false;
      }
    }
  }

  async function handleSearchYTMusicArtists(q: string) {
    const seq = ++ytSearchSeq;
    ytLoading = true;
    try {
      const res = await api.searchYouTubeMusicArtists(q);
      if (seq === ytSearchSeq) {
        ytArtistResults = res || [];
      }
    } catch {
      if (seq === ytSearchSeq) {
        ytArtistResults = [];
      }
    } finally {
      if (seq === ytSearchSeq) {
        ytLoading = false;
      }
    }
  }

  const MORESONGS_QUERIES = [
    'Lagu Indonesia Populer Terbaru',
    'Top Global Hits Music',
    'Lagu Indie & Chill Indonesia',
    'R&B Soul Groove Hits',
    'Acoustic & Lo-Fi Chill Music',
    'Pop Rock & Alternative Hits',
    'Throwback Hits 2000s 2010s',
    'Electronic & Synthwave Music'
  ];

  const CATEGORY_MORE_QUERIES: Record<'musik' | 'artis' | 'album' | 'playlist' | 'podcast' | 'buku', string[]> = {
    musik: [
      'Trending Pop Music Indonesia',
      'Top Hits Indonesia 2025 2026',
      'Viral Pop & R&B Hits',
      'Best Acoustic Pop Hits',
      'Lagu Indie & Chill Indonesia',
      'Top Global Pop Songs',
      'Throwback Hits 2000s 2010s',
      'Feel Good & Chill Music'
    ],
    artis: [
      'Popular Indonesian Singers',
      'Top Global Pop Artists',
      'Best Indie Artists Indonesia',
      'Top R&B & Soul Artists',
      'Popular Rock & Alternative Bands',
      'Trending Acoustic & Folk Artists',
      'Electronic & Dance Artists',
      'Asia Pop & Ballad Singers'
    ],
    album: [
      'Top Pop Indonesia Album',
      'Best Indie Album Indonesia',
      'Top Global Pop Album',
      'R&B Soul Classic Album',
      'Best Rock & Alternative Album',
      'Acoustic & Singer Songwriter Album',
      'Trending Hip Hop & R&B Album',
      'Essential Greatest Hits Album'
    ],
    playlist: [
      'Top Hits Indonesia Playlist',
      'Indo Chill & Galau Playlist',
      'Top Global Hits Playlist',
      'Workout & Mood Booster Playlist'
    ],
    podcast: [
      'Podcast Musik Indonesia',
      'Podcast Obrolan Santai Indonesia',
      'Music Talk Podcast Episode'
    ],
    buku: [
      'Audiobook Chill Ambient',
      'Audiobook Story Relaxation',
      'Calm Reading Instrumental Background'
    ]
  };

  async function loadMoreCollectionBatch(opts: {
    targetBatch: number;
    existingList: YTMusicCard[];
    buffer: YTMusicCard[];
    fetchPrimary: () => Promise<YTMusicCard[]>;
    searchFallback: (query: string) => Promise<YTMusicCard[]>;
    queries: string[];
    dedupeByCover?: boolean;
    spreadByArtist?: boolean;
  }): Promise<{ added: YTMusicCard[]; nextBuffer: YTMusicCard[] }> {
    const existingKeys = new Set(
      opts.existingList.map((c) => (c.browseId || c.title.trim().toLowerCase()).trim().toLowerCase()).filter(Boolean)
    );
    const existingTitles = new Set(
      opts.existingList.map((c) => c.title.trim().toLowerCase()).filter(Boolean)
    );
    const existingCovers = new Set(
      opts.existingList.map((c) => (c.coverUrl || '').trim()).filter(Boolean)
    );

    const pool: YTMusicCard[] = [];
    for (const c of opts.buffer) {
      const k = (c.browseId || c.title.trim().toLowerCase()).trim().toLowerCase();
      const tLow = c.title.trim().toLowerCase();
      if (!k || existingKeys.has(k) || existingTitles.has(tLow)) continue;
      existingKeys.add(k);
      existingTitles.add(tLow);
      const cov = (c.coverUrl || '').trim();
      if (cov) existingCovers.add(cov);
      pool.push(c);
    }

    const batchArtistCount = new Map<string, number>();
    const pushCards = (cards: YTMusicCard[] | null | undefined, strict = true) => {
      for (const c of cards || []) {
        if (pool.length >= opts.targetBatch) break;
        const k = (c.browseId || c.title.trim().toLowerCase()).trim().toLowerCase();
        const tLow = c.title.trim().toLowerCase();
        if (!k || existingKeys.has(k) || existingTitles.has(tLow)) continue;
        const cov = (c.coverUrl || '').trim();
        const ar = (c.artist || c.subtitle || '').trim().toLowerCase();
        if (strict) {
          if (opts.dedupeByCover && cov && existingCovers.has(cov)) continue;
          if (opts.spreadByArtist && ar && (batchArtistCount.get(ar) || 0) >= 2) continue;
        }
        existingKeys.add(k);
        existingTitles.add(tLow);
        if (cov) existingCovers.add(cov);
        if (ar) batchArtistCount.set(ar, (batchArtistCount.get(ar) || 0) + 1);
        pool.push(c);
      }
    };

    if (pool.length < opts.targetBatch) {
      const primary = await opts.fetchPrimary().catch(() => []);
      pushCards(primary, true);
    }

    let attempts = 0;
    while (pool.length < opts.targetBatch && attempts < 6) {
      attempts++;
      const page = ytMorePage++;
      const q = opts.queries[page % opts.queries.length];
      const more = await opts.searchFallback(q).catch(() => []);
      pushCards(more, true);
      if (pool.length < opts.targetBatch && attempts >= 3) {
        pushCards(more, false);
      }
    }

    if (pool.length === 0) {
      return { added: [], nextBuffer: [] };
    }

    const ordered = opts.spreadByArtist
      ? shuffleAndSpreadByArtist(
          pool,
          (c) => c.artist || c.subtitle || '',
          opts.existingList.slice(-3).map((c) => c.artist || c.subtitle || '')
        )
      : pool;
    const takeCount = Math.min(opts.targetBatch, ordered.length);
    return {
      added: ordered.slice(0, takeCount),
      nextBuffer: ordered.slice(takeCount)
    };
  }

  async function handleLoadMorePopularArtists() {
    if (ytLoadingMorePopularArtists) return;
    ytLoadingMorePopularArtists = true;
    try {
      const idx = ytShelves.findIndex((s) => s.title === 'Artis Populer');
      if (idx < 0) return;
      const existingList = ytShelves[idx].items;
      const existingKeys = existingList.map((c) => c.browseId || c.title.toLowerCase());
      const { added, nextBuffer } = await loadMoreCollectionBatch({
        targetBatch: 12,
        existingList,
        buffer: ytMorePopularArtistsBuffer,
        fetchPrimary: () => api.getMoreArtists(existingList.length, existingKeys),
        searchFallback: (q) => api.searchYouTubeMusicArtists(q),
        queries: CATEGORY_MORE_QUERIES.artis
      });
      ytMorePopularArtistsBuffer = nextBuffer;
      if (added.length > 0) {
        const updated = [...ytShelves];
        updated[idx] = {
          ...updated[idx],
          items: [...updated[idx].items, ...added]
        };
        ytShelves = updated;
      }
    } finally {
      ytLoadingMorePopularArtists = false;
    }
  }

  async function handleLoadMoreYTMusic(activeCategory: HomeFilter) {
    if (ytLoadingMore) return;
    ytLoadingMore = true;
    try {
      if (activeCategory === 'album') {
        const targetBatch = 12;
        const { added, nextBuffer } = await loadMoreCollectionBatch({
          targetBatch,
          existingList: ytAlbumResults,
          buffer: ytMoreAlbumBuffer,
          fetchPrimary: () => api.searchYouTubeMusicAlbums(CATEGORY_MORE_QUERIES.album[(ytMorePage++) % CATEGORY_MORE_QUERIES.album.length]),
          searchFallback: (q) => api.searchYouTubeMusicAlbums(q),
          queries: CATEGORY_MORE_QUERIES.album,
          dedupeByCover: true,
          spreadByArtist: true
        });
        ytMoreAlbumBuffer = nextBuffer;
        if (added.length > 0) {
          ytAlbumResults = [...ytAlbumResults, ...added];
        }
        return;
      }

      if (activeCategory === 'artis') {
        const targetBatch = 12;
        const existingKeys = ytArtistResults.map((c) => c.browseId || c.title.toLowerCase());
        const { added, nextBuffer } = await loadMoreCollectionBatch({
          targetBatch,
          existingList: ytArtistResults,
          buffer: ytMoreArtistBuffer,
          fetchPrimary: () => api.getMoreArtists(ytArtistResults.length, existingKeys),
          searchFallback: (q) => api.searchYouTubeMusicArtists(q),
          queries: CATEGORY_MORE_QUERIES.artis
        });
        ytMoreArtistBuffer = nextBuffer;
        if (added.length > 0) {
          ytArtistResults = [...ytArtistResults, ...added];
        }
        return;
      }

      if (activeCategory === 'playlist') {
        const targetBatch = 12;
        const { added, nextBuffer } = await loadMoreCollectionBatch({
          targetBatch,
          existingList: ytPlaylistResults,
          buffer: ytMorePlaylistBuffer,
          fetchPrimary: () =>
            api.searchYouTubeMusicPlaylists(
              CATEGORY_MORE_QUERIES.playlist[(ytMorePage++) % CATEGORY_MORE_QUERIES.playlist.length]
            ),
          searchFallback: (q) => api.searchYouTubeMusicPlaylists(q),
          queries: CATEGORY_MORE_QUERIES.playlist,
          dedupeByCover: true
        });
        ytMorePlaylistBuffer = nextBuffer;
        if (added.length > 0) {
          ytPlaylistResults = [...ytPlaylistResults, ...added];
        }
        return;
      }

      if (activeCategory === 'semua') {
        // First try real YouTube Music home continuation shelves (matching Auralys YouTube.homeContinuation)
        if (ytHomeContinuation) {
          try {
            const nextPage = await api.getYTMusicHomeContinuation(ytHomeContinuation);
            ytHomeContinuation = (nextPage?.continuation || '').trim();
            const existingTitles = new Set(ytShelves.map((s) => s.title.trim().toLowerCase()));
            const newShelves = normalizeHomeShelves(nextPage?.shelves || []).filter(
              (sh) => !existingTitles.has(sh.title.trim().toLowerCase())
            );
            if (newShelves.length > 0) {
              ytShelves = normalizeHomeShelves([...ytShelves, ...newShelves]);
              const existingVids = new Set(ytResults.map((t) => t.videoId));
              const extraTracks = filterExplicit(nextPage?.tracks || []).filter((t) => t.videoId && !existingVids.has(t.videoId));
              if (extraTracks.length > 0) {
                ytResults = [...ytResults, ...extraTracks];
              }
              return;
            }
          } catch {
            ytHomeContinuation = '';
          }
        }

        const targetTitle = 'Lagu Lainnya untuk Anda';
        const existingShelf = ytShelves.find((s) => s.title === targetTitle);
        const cols = 7;
        const currentLen = existingShelf?.items.length || 0;
        const remainder = currentLen % cols;
        const targetBatch = cols * 2 + (remainder === 0 ? 0 : cols - remainder); // 2 full rows x 7 items
        const tailArtists = (existingShelf?.items || []).slice(-3).map((c) => c.artist || c.subtitle || '');

        const existingVideoIds = new Set<string>(ytMoreAllBuffer.map((c) => c.videoId));
        const existingCovers = new Set<string>(ytMoreAllBuffer.map((c) => (c.coverUrl || '').trim()).filter(Boolean));
        const existingAlbums = new Set<string>(
          ytMoreAllBuffer.map((c) => (c.album || '').trim().toLowerCase()).filter((a) => a && a !== 'single')
        );
        for (const shelf of ytShelves) {
          for (const item of shelf.items) {
            if (item.videoId) existingVideoIds.add(item.videoId);
            const cov = (item.coverUrl || '').trim();
            if (cov) existingCovers.add(cov);
            const alb = (item.album || '').trim().toLowerCase();
            if (alb && alb !== 'single') existingAlbums.add(alb);
          }
        }

        const pool = [...ytMoreAllBuffer];
        const batchArtistCount = new Map<string, number>();
        for (const item of pool) {
          const ar = (item.artist || item.subtitle || '').trim().toLowerCase();
          if (ar) batchArtistCount.set(ar, (batchArtistCount.get(ar) || 0) + 1);
        }

        const pushUniqueTopSongs = (tracksList: YTMusicTrack[] | null | undefined, sourceLabel: string, strict = true) => {
          for (const t of tracksList || []) {
            if (pool.length >= targetBatch + 7) break;
            if (!t.videoId || existingVideoIds.has(t.videoId)) continue;
            const cov = (t.coverUrl || '').trim();
            const alb = (t.album || '').trim().toLowerCase();
            const ar = (t.artist || '').trim().toLowerCase();
            if (strict) {
              if (cov && existingCovers.has(cov)) continue;
              if (alb && alb !== 'single' && existingAlbums.has(alb)) continue;
              if (ar && (batchArtistCount.get(ar) || 0) >= 2) continue;
            }

            existingVideoIds.add(t.videoId);
            if (cov) existingCovers.add(cov);
            if (alb && alb !== 'single') existingAlbums.add(alb);
            if (ar) batchArtistCount.set(ar, (batchArtistCount.get(ar) || 0) + 1);

            pool.push({
              id: `song:${t.videoId}`,
              title: t.title,
              subtitle: t.artist || sourceLabel,
              coverUrl: t.coverUrl,
              kind: 'song' as const,
              videoId: t.videoId,
              browseId: '',
              playlistId: '',
              artist: t.artist,
              album: t.album,
              durationSec: t.durationSec
            });
          }
        };

        const [recTracks, topSongs] = await Promise.all([
          api.getYTMusicTop100Recommended(Array.from(existingVideoIds)).catch(() => []),
          api.getMoreTopSongs((ytMorePage++) * 20, Array.from(existingVideoIds)).catch(() => [])
        ]);
        pushUniqueTopSongs(filterExplicit(recTracks || []), 'Velora Music');
        pushUniqueTopSongs(filterExplicit(topSongs || []), 'Top Songs');

        let attempts = 0;
        while (pool.length < targetBatch && attempts < 6) {
          attempts++;
          const page = ytMorePage++;
          const q = MORESONGS_QUERIES[page % MORESONGS_QUERIES.length];
          const moreTracks = await api.searchYouTubeMusic(q).catch(() => []);
          const filteredMore = filterExplicit(moreTracks || []);
          pushUniqueTopSongs(filteredMore, 'Top Songs', true);
          if (pool.length < targetBatch && attempts >= 3) {
            pushUniqueTopSongs(filteredMore, 'Top Songs', false);
          }
        }

        if (pool.length === 0) return;

        const spreadPool = shuffleAndSpreadByArtist(pool, (c) => c.artist || c.subtitle || '', tailArtists);
        const takeCount = Math.min(targetBatch, spreadPool.length);
        const toAppend = spreadPool.slice(0, takeCount);
        ytMoreAllBuffer = spreadPool.slice(takeCount);

        if (toAppend.length > 0) {
          const existingIdx = ytShelves.findIndex((s) => s.title === targetTitle);
          if (existingIdx >= 0) {
            const updated = [...ytShelves];
            updated[existingIdx] = {
              ...updated[existingIdx],
              items: [...updated[existingIdx].items, ...toAppend]
            };
            ytShelves = updated;
          } else {
            ytShelves = [...ytShelves, { title: targetTitle, items: toAppend }];
          }
        }
      } else {
        const targetBatch = 14;
        const existingIds = new Set(ytResults.map((t) => t.videoId));
        const existingCovers = new Set(ytResults.map((t) => (t.coverUrl || '').trim()).filter(Boolean));
        const existingAlbums = new Set(
          ytResults.map((t) => (t.album || '').trim().toLowerCase()).filter((a) => a && a !== 'single')
        );
        for (const t of ytMoreMusicBuffer) {
          if (t.videoId) existingIds.add(t.videoId);
          const cov = (t.coverUrl || '').trim();
          if (cov) existingCovers.add(cov);
          const alb = (t.album || '').trim().toLowerCase();
          if (alb && alb !== 'single') existingAlbums.add(alb);
        }
        const tailArtists = ytResults.slice(-3).map((t) => t.artist || '');
        const pool: YTMusicTrack[] = [...ytMoreMusicBuffer];
        const batchArtistCount = new Map<string, number>();
        for (const item of pool) {
          const ar = (item.artist || '').trim().toLowerCase();
          if (ar) batchArtistCount.set(ar, (batchArtistCount.get(ar) || 0) + 1);
        }

        const collectTracks = (list: YTMusicTrack[] | null | undefined, strict = true) => {
          for (const t of list || []) {
            if (pool.length >= targetBatch) break;
            if (!t.videoId || existingIds.has(t.videoId)) continue;
            const cov = (t.coverUrl || '').trim();
            const alb = (t.album || '').trim().toLowerCase();
            const ar = (t.artist || '').trim().toLowerCase();
            if (strict) {
              if (cov && existingCovers.has(cov)) continue;
              if (alb && alb !== 'single' && existingAlbums.has(alb)) continue;
              if (ar && (batchArtistCount.get(ar) || 0) >= 2) continue;
            }

            existingIds.add(t.videoId);
            if (cov) existingCovers.add(cov);
            if (alb && alb !== 'single') existingAlbums.add(alb);
            if (ar) batchArtistCount.set(ar, (batchArtistCount.get(ar) || 0) + 1);
            pool.push(t);
          }
        };

        if (activeCategory === 'musik' && pool.length < targetBatch) {
          const [rec, topSongs] = await Promise.all([
            api.getYTMusicTop100Recommended(Array.from(existingIds)).catch(() => []),
            api.getMoreTopSongs(ytResults.length + (ytMorePage++) * 20, Array.from(existingIds)).catch(() => [])
          ]);
          collectTracks(filterExplicit(rec || []), true);
          collectTracks(filterExplicit(topSongs || []), true);
        }

        const queries = CATEGORY_MORE_QUERIES[activeCategory] || CATEGORY_MORE_QUERIES.musik;
        let attempts = 0;
        while (pool.length < targetBatch && attempts < 6) {
          attempts++;
          const page = ytMorePage++;
          const q = queries[page % queries.length];
          const moreTracks = await api.searchYouTubeMusic(q).catch(() => []);
          const filtered = filterExplicit(moreTracks || []);
          collectTracks(filtered, true);
          if (pool.length < targetBatch && attempts >= 3) {
            collectTracks(filtered, false);
          }
        }

        if (pool.length > 0) {
          const spreadPool = shuffleAndSpreadByArtist(pool, (t) => t.artist || '', tailArtists);
          const takeCount = Math.min(targetBatch, spreadPool.length);
          ytMoreMusicBuffer = spreadPool.slice(takeCount);
          ytResults = [...ytResults, ...spreadPool.slice(0, takeCount)];
        }
      }
    } finally {
      ytLoadingMore = false;
    }
  }

  const atvCheckedTrackIds = new Set<number>();

  function shouldUpgradeVideoTrackToATV(tr: Track): boolean {
    if (tr.sourceType !== 'ytmusic' || tr.isDownloaded || tr.id <= 0) return false;
    if (!isYTVideoThumb(tr.coverUrl)) return false;
    const low = (tr.title || '').toLowerCase();
    for (const marker of [
      'official video', 'music video', 'lyric video', ' mv', '(mv', '[mv', 'm/v',
      'live', 'session', 'cover', 'karaoke', 'remix', 'acoustic', 'dangdut', 'koplo',
      'slowed', 'sped up', 'nightcore', 'instrumental'
    ]) {
      if (low.includes(marker)) return false;
    }
    return true;
  }

  function resolveDirectStreamUrl(tr: Track): string {
    if (!mediaBaseUrl) return '';
    if (tr.sourceType === 'local' && tr.id > 0) {
      return `${mediaBaseUrl}/media/local?id=${tr.id}`;
    }
    if (tr.externalId && !tr.externalId.startsWith('dz:') && !tr.externalId.startsWith('sp:')) {
      if (shouldUpgradeVideoTrackToATV(tr) && !atvCheckedTrackIds.has(tr.id)) {
        return '';
      }
      const q = encodeURIComponent(getPrefValue(preferences, 'stream_quality') || 'high');
      return `${mediaBaseUrl}/media/yt?v=${encodeURIComponent(tr.externalId)}&q=${q}`;
    }
    return '';
  }

  function trackQueueKey(t: Track): string {
    return t.externalId ? `yt:${t.externalId}` : `local:${t.id}`;
  }

  function shuffleArray<T>(items: T[]): T[] {
    const arr = [...items];
    for (let i = arr.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [arr[i], arr[j]] = [arr[j], arr[i]];
    }
    return arr;
  }

  function ytTrackToQueueTrack(yt: YTMusicTrack, idx: number): Track {
    return {
      id: -(idx + 1 + algoBatchRound * 100),
      sourceType: 'ytmusic',
      filePath: '',
      externalId: yt.videoId,
      title: yt.title,
      artistId: 0,
      artistName: yt.artist || 'YouTube Music',
      albumId: 0,
      albumTitle: yt.album || 'YouTube Music',
      durationSec: yt.durationSec || 200,
      coverUrl: yt.coverUrl,
      format: yt.format || 'OPUS',
      bitrate: 160,
      isFavorite: false,
      playCount: 0,
      createdAt: ''
    };
  }

  function collectFallbackGenrePool(excludeKeys: Set<string>, limit = 20): Track[] {
    const pool: Track[] = [];
    const seen = new Set<string>(excludeKeys);

    const addIfNew = (candidate: Track) => {
      const k = trackQueueKey(candidate);
      if (seen.has(k)) return;
      seen.add(k);
      pool.push(candidate);
    };

    for (let i = 0; i < ytResults.length; i++) {
      if (ytResults[i].videoId) addIfNew(ytTrackToQueueTrack(ytResults[i], i));
    }
    for (const shelf of ytShelves) {
      for (let i = 0; i < shelf.items.length; i++) {
        const card = shelf.items[i];
        if (card.kind === 'song' && card.videoId) {
          addIfNew(
            ytTrackToQueueTrack(
              {
                videoId: card.videoId,
                title: card.title,
                artist: card.artist || card.subtitle || 'YouTube Music',
                album: card.album || 'YouTube Music',
                durationSec: card.durationSec || 200,
                coverUrl: card.coverUrl,
                format: 'OPUS'
              },
              pool.length + i
            )
          );
        }
      }
    }
    for (const t of tracks) {
      addIfNew(t);
    }

    const diverse: Track[] = [];
    const artistCounts = new Map<string, number>();
    const albumCounts = new Map<string, number>();
    for (const cand of shuffleArray(pool)) {
      const ar = (cand.artistName || '').trim().toLowerCase();
      const al = (cand.albumTitle || '').trim().toLowerCase();
      if (ar && (artistCounts.get(ar) ?? 0) >= 2) continue;
      if (al && al !== 'single' && al !== 'youtube music' && (albumCounts.get(al) ?? 0) >= 1) continue;
      if (ar) artistCounts.set(ar, (artistCounts.get(ar) ?? 0) + 1);
      if (al) albumCounts.set(al, (albumCounts.get(al) ?? 0) + 1);
      diverse.push(cand);
      if (diverse.length >= limit) break;
    }
    return shuffleAndSpreadByArtist(diverse, (t) => t.artistName || t.albumTitle || '').slice(0, limit);
  }

  function buildLocalAlgorithmQueue(tr: Track): Track[] {
    const k = trackQueueKey(tr);
    playedAlgoKeys = new Set<string>([k]);
    algoBatchRound = 0;
    const trArtist = (tr.artistName || '').trim().toLowerCase();
    const others = tracks.filter((t) => t.id !== tr.id);
    const artistCounts = new Map<string, number>();
    if (trArtist) artistCounts.set(trArtist, 1);
    const diverse: Track[] = [];
    for (const cand of shuffleArray(others)) {
      const ar = (cand.artistName || '').trim().toLowerCase();
      if (ar && (artistCounts.get(ar) ?? 0) >= 2) continue;
      if (ar) artistCounts.set(ar, (artistCounts.get(ar) ?? 0) + 1);
      diverse.push(cand);
      if (diverse.length >= 20) break;
    }
    let picked = shuffleAndSpreadByArtist(diverse, (t) => t.artistName || t.albumTitle || '').slice(0, 20);
    if (picked.length < 20) {
      const extra = collectFallbackGenrePool(new Set([k, ...picked.map(trackQueueKey)]), 20 - picked.length);
      picked = shuffleAndSpreadByArtist([...picked, ...extra], (t) => t.artistName || t.albumTitle || '').slice(0, 20);
    }
    for (const item of picked) {
      playedAlgoKeys.add(trackQueueKey(item));
    }
    return [tr, ...picked.slice(0, 20)];
  }

  function getGenreQueriesForBatch(_baseTrack: Track, round: number): string[] {
    const allQueries = [
      'Top Hits Indonesia',
      'Hot Hits Indonesia',
      'Lagu Barat Terpopuler di Indonesia',
      'Pop Indonesia Hits Terbaru',
      'Top 50 Indonesia Viral',
      'Top Global Pop Hits',
      'Lagu Indonesia Populer Terbaru',
      'Trending English Pop Songs',
      'Indo Pop & Indie Hits'
    ];

    const start = (round * 3) % allQueries.length;
    return [
      allQueries[start % allQueries.length],
      allQueries[(start + 1) % allQueries.length],
      allQueries[(start + 2) % allQueries.length]
    ];
  }

  async function fetchTwentyGenreTracks(
    baseTrack: Track,
    excludeKeys: Set<string>,
    round: number
  ): Promise<Track[]> {
    const excludeVideoIds = Array.from(excludeKeys)
      .filter((k) => k.startsWith('yt:'))
      .map((k) => k.slice(3));

    const seen = new Set<string>(excludeKeys);
    const candidates: Track[] = [];
    let idxCounter = 0;

    try {
      const recItems = await api.getYTMusicTop100Recommended(excludeVideoIds);
      for (const item of recItems || []) {
        if (!item.videoId) continue;
        if (isHideExplicitEnabled() && isExplicitItem(item)) continue;
        const key = `yt:${item.videoId}`;
        if (seen.has(key)) continue;
        seen.add(key);
        candidates.push(ytTrackToQueueTrack(item, idxCounter++));
      }
    } catch {
    }

    if (candidates.length < 15) {
      const queries = getGenreQueriesForBatch(baseTrack, round);
      const responses = await Promise.all(
        queries.map((q) => api.searchYouTubeMusic(q).catch(() => [] as YTMusicTrack[]))
      );
      for (const list of responses) {
        for (const item of list || []) {
          if (!item.videoId) continue;
          if (isHideExplicitEnabled() && isExplicitItem(item)) continue;
          const key = `yt:${item.videoId}`;
          if (seen.has(key)) continue;
          seen.add(key);
          candidates.push(ytTrackToQueueTrack(item, idxCounter++));
        }
      }
    }

    const artistCounts = new Map<string, number>();
    const albumCounts = new Map<string, number>();
    const baseArtist = (baseTrack.artistName || '').trim().toLowerCase();
    if (baseArtist) artistCounts.set(baseArtist, 1);

    const diverse: Track[] = [];
    for (const cand of shuffleArray(candidates)) {
      const ar = (cand.artistName || '').trim().toLowerCase();
      const al = (cand.albumTitle || '').trim().toLowerCase();
      if (ar && (artistCounts.get(ar) ?? 0) >= 2) continue;
      if (al && al !== 'single' && al !== 'youtube music' && (albumCounts.get(al) ?? 0) >= 1) continue;
      if (ar) artistCounts.set(ar, (artistCounts.get(ar) ?? 0) + 1);
      if (al) albumCounts.set(al, (albumCounts.get(al) ?? 0) + 1);
      diverse.push(cand);
      if (diverse.length >= 20) break;
    }

    const radioLimit = getRadioLimit();
    let picked = shuffleAndSpreadByArtist(diverse, (t) => t.artistName || t.albumTitle || '').slice(0, radioLimit);
    if (picked.length < radioLimit) {
      const extra = collectFallbackGenrePool(seen, radioLimit - picked.length);
      picked = shuffleAndSpreadByArtist([...picked, ...extra], (t) => t.artistName || t.albumTitle || '').slice(0, radioLimit);
    }

    for (const item of picked) {
      excludeKeys.add(trackQueueKey(item));
    }
    return picked;
  }

  async function populateYTMusicAlgorithmQueue(tr: Track) {
    try {
      const freshExclude = new Set<string>([trackQueueKey(tr)]);
      const batch20 = await fetchTwentyGenreTracks(tr, freshExclude, 0);
      if (batch20.length === 0) return;
      if (currentTrack?.id !== tr.id || queueSourceTitle !== '') return;

      playedAlgoKeys = freshExclude;
      queue = [tr, ...batch20];
    } catch {
    }
  }

  async function playYTMusicSingle(yt: YTMusicTrack) {
    queueSourceTitle = '';
    algoBatchRound = 0;
    trackLoading = true;
    let tr: Track;
    try {
      tr = await api.playYTMusicTrack(yt);
    } catch {
      trackLoading = false;
      showNetworkToast(appLang === 'id' ? 'Koneksi lambat atau terputus, coba lagi' : 'Slow or lost connection, please try again');
      return;
    }
    playedAlgoKeys = new Set<string>([trackQueueKey(tr)]);
    if (!getPrefBool(preferences, 'playback.radioOnPick')) {
      await playTrack(tr, [tr], '');
      void refreshAll();
      return;
    }
    const seed = collectFallbackGenrePool(playedAlgoKeys, getRadioLimit());
    for (const s of seed) {
      playedAlgoKeys.add(trackQueueKey(s));
    }
    await playTrack(tr, [tr, ...seed], '');
    void populateYTMusicAlgorithmQueue(tr);
    void refreshAll();
  }

  async function handlePlayYTCollection(card: YTMusicCard) {
    if (card.kind === 'song' && card.videoId) {
      const ytTrack: YTMusicTrack = {
        videoId: card.videoId,
        title: card.title,
        artist: card.artist || card.subtitle || 'YouTube Music',
        album: card.album || 'YouTube Music',
        durationSec: card.durationSec || 200,
        coverUrl: card.coverUrl,
        format: 'OPUS'
      };
      await playYTMusicSingle(ytTrack);
      return;
    }

    loadingCollectionId = card.id;
    try {
      const colTracks = await api.playYTMusicCollection(
        card.browseId,
        card.playlistId,
        card.title,
        card.subtitle,
        card.coverUrl
      );
      if (colTracks && colTracks.length > 0) {
        await playTrack(colTracks[0], colTracks, card.title, card.browseId || card.playlistId || card.id);
        void refreshAll();
      } else if (card.videoId) {
        const ytTrack: YTMusicTrack = {
          videoId: card.videoId,
          title: card.title,
          artist: card.artist || card.subtitle || 'YouTube Music',
          album: card.album || 'YouTube Music',
          durationSec: card.durationSec || 200,
          coverUrl: card.coverUrl,
          format: 'OPUS'
        };
        await playYTMusicSingle(ytTrack);
      }
    } finally {
      loadingCollectionId = null;
    }
  }

  async function handlePlayPlaylist(playlistId: number) {
    const detail = await api.getPlaylistDetail(playlistId);
    if (detail?.tracks && detail.tracks.length > 0) {
      await playTrack(
        detail.tracks[0],
        detail.tracks,
        detail.playlist?.name || 'Playlist',
        `local-playlist:${playlistId}`
      );
    }
  }

  async function handleOpenAlbum(albumId: number) {
    const al = albums.find((a) => a.id === albumId);
    if (al?.sourceType === 'ytmusic') {
      const ext = (al.externalId || '').trim();
      const isBrowse =
        ext.startsWith('MPREb_') ||
        ext.startsWith('OLAK') ||
        ext.startsWith('dz:album:') ||
        ext.startsWith('sp:album:');
      await navigate({
        name: 'yt-album-detail',
        browseId: isBrowse ? ext : '',
        videoId: isBrowse ? '' : ext,
        kind: 'album',
        title: al.title,
        artist: al.artistName,
        albumTitle: al.title,
        coverUrl: al.coverUrl
      });
      return;
    }
    await navigate({ name: 'album-detail', albumId });
  }

  async function handlePlayAlbum(albumId: number) {
    const al = albums.find((a) => a.id === albumId);
    if (al?.sourceType === 'ytmusic' && al.externalId) {
      await handlePlayYTCollection({
        id: al.externalId,
        title: al.title,
        subtitle: al.artistName,
        coverUrl: al.coverUrl,
        kind: 'album',
        videoId: '',
        browseId: al.externalId,
        playlistId: '',
        artist: al.artistName,
        album: al.title,
        durationSec: 0
      });
      return;
    }
    const albumTracks = await api.getAlbumTracks(albumId);
    if (albumTracks && albumTracks.length > 0) {
      const albumName = al?.title || 'Album';
      await playTrack(albumTracks[0], albumTracks, albumName, `local-album:${albumId}`);
    }
  }

  async function handlePlayArtist(artistId: number) {
    const artistTracks = await api.getArtistTracks(artistId);
    if (artistTracks && artistTracks.length > 0) {
      const artistName = artists.find((a) => a.id === artistId)?.name || 'Artis';
      await playTrack(artistTracks[0], artistTracks, artistName, `local-artist:${artistId}`);
    }
  }

  function fisherYatesShuffle<T>(arr: T[]): T[] {
    const copy = [...arr];
    for (let i = copy.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [copy[i], copy[j]] = [copy[j], copy[i]];
    }
    return copy;
  }

  function toggleShuffleMode() {
    const next = !shuffle;
    shuffle = next;
    if (isJamRemote) {
      void sendJamCommand('shuffle');
      return;
    }
    api.playerControl('shuffle');
    if (queue.length === 0) return;

    if (next) {
      if (originalQueue.length === 0) {
        originalQueue = [...queue];
      }
      const curKey = currentTrack ? trackQueueKey(currentTrack) : '';
      shufflePlayedKeys = new Set(curKey ? [curKey] : []);
      const rest = curKey
        ? queue.filter((t) => trackQueueKey(t) !== curKey)
        : [...queue];
      const shuffledRest = fisherYatesShuffle(rest);
      queue = currentTrack ? [currentTrack, ...shuffledRest] : shuffledRest;
      saveLastQueue(true);
    } else {
      if (originalQueue.length > 0) {
        const orderByKey = new Map<string, number>();
        originalQueue.forEach((t, i) => orderByKey.set(trackQueueKey(t), i));
        queue = [...queue].sort((a, b) => {
          const ia = orderByKey.get(trackQueueKey(a)) ?? Number.MAX_SAFE_INTEGER;
          const ib = orderByKey.get(trackQueueKey(b)) ?? Number.MAX_SAFE_INTEGER;
          return ia - ib;
        });
        originalQueue = [];
      }
      shufflePlayedKeys.clear();
      shuffleHistory = [];
      saveLastQueue(true);
    }
  }

  async function playShuffledList(list: Track[], sourceTitle?: string, sourceKey?: string) {
    if (!list || list.length === 0) return;
    const wasShuffled = shuffle;
    shuffle = true;
    if (!wasShuffled) {
      api.playerControl('shuffle');
    }
    originalQueue = [...list];
    const shuffled = fisherYatesShuffle(list);
    shufflePlayedKeys = new Set([trackQueueKey(shuffled[0])]);
    shuffleHistory = [];
    await playTrack(shuffled[0], shuffled, sourceTitle, sourceKey, true);
  }

  async function playTrack(tr: Track, list: Track[] = [], sourceTitle?: string, sourceKey?: string, keepOriginalQueue = false, preferredQueueIndex?: number) {
    if (isJamRemote) {
      await sendJamCommand('play-track', {
        trackId: tr.id,
        externalId: tr.externalId || '',
        title: tr.title,
        artist: tr.artistName,
        album: tr.albumTitle,
        coverUrl: tr.coverUrl,
        durationSec: tr.durationSec
      });
      return;
    }
    const clickedRefIdx = list.length > 0 ? list.indexOf(tr) : -1;
    if (sourceKey !== undefined) {
      queueSourceKey = sourceKey;
    } else if (sourceTitle !== undefined) {
      queueSourceKey = '';
    }
    if (sourceTitle !== undefined) {
      queueSourceTitle = sourceTitle;
    } else if (list === queue && (queueSourceTitle || queueSourceKey)) {
      // Playing a track from within the existing queue preserves its source navigation link.
    } else if (list.length > 0) {
      if (currentRoute.name === 'playlist-detail') {
        queueSourceTitle = selectedPlaylistDetail?.playlist?.name || 'Playlist';
        queueSourceKey = `local-playlist:${currentRoute.playlistId}`;
      } else if (currentRoute.name === 'album-detail') {
        queueSourceTitle = albums.find((a) => a.id === currentRoute.albumId)?.title || 'Album';
        queueSourceKey = `local-album:${currentRoute.albumId}`;
      } else if (currentRoute.name === 'artist-detail') {
        queueSourceTitle = artists.find((a) => a.id === currentRoute.artistId)?.name || 'Artis';
        queueSourceKey = currentRoute.browseId || `local-artist:${currentRoute.artistId ?? ''}`;
      } else if (currentRoute.name === 'liked-songs') {
        queueSourceTitle = 'Favorite music';
        queueSourceKey = 'liked-songs';
      } else if (currentRoute.name === 'history') {
        queueSourceTitle = appLang === 'id' ? 'Riwayat' : 'History';
        queueSourceKey = 'history';
      } else if (currentRoute.name === 'yt-album-detail') {
        queueSourceTitle = currentRoute.albumTitle || currentRoute.title || 'Album';
        queueSourceKey = currentRoute.browseId || currentRoute.videoId || '';
      }
    } else {
      queueSourceTitle = '';
      queueSourceKey = '';
    }

    if (list.length > 0) {
      if (list === queue) {
        queue = [...list];
        if (preferredQueueIndex !== undefined && preferredQueueIndex >= 0 && preferredQueueIndex < queue.length && trackQueueKey(queue[preferredQueueIndex]) === trackQueueKey(tr)) {
          currentQueueIndex = preferredQueueIndex;
        } else if (clickedRefIdx >= 0 && clickedRefIdx < queue.length) {
          currentQueueIndex = clickedRefIdx;
        } else {
          currentQueueIndex = resolveActiveQueueIndex(tr, queue);
        }
      } else {
        if (!keepOriginalQueue) {
          originalQueue = [...list];
        }
        const curKey = trackQueueKey(tr);
        shufflePlayedKeys = new Set([curKey]);
        if (shuffle && !keepOriginalQueue) {
          const rest = clickedRefIdx >= 0
            ? list.filter((_, idx) => idx !== clickedRefIdx)
            : list.filter((item) => trackQueueKey(item) !== curKey);
          queue = [tr, ...fisherYatesShuffle(rest)];
          currentQueueIndex = 0;
        } else {
          queue = [...list];
          if (preferredQueueIndex !== undefined && preferredQueueIndex >= 0 && preferredQueueIndex < queue.length && trackQueueKey(queue[preferredQueueIndex]) === trackQueueKey(tr)) {
            currentQueueIndex = preferredQueueIndex;
          } else if (clickedRefIdx >= 0 && clickedRefIdx < queue.length) {
            currentQueueIndex = clickedRefIdx;
          } else {
            currentQueueIndex = resolveActiveQueueIndex(tr, queue);
          }
        }
      }
    } else {
      originalQueue = [];
      if (getPrefBool(preferences, 'playback.radioOnPick')) {
        queue = buildLocalAlgorithmQueue(tr);
        void populateYTMusicAlgorithmQueue(tr);
      } else {
        queue = [tr];
      }
      currentQueueIndex = 0;
      shufflePlayedKeys = new Set([trackQueueKey(tr)]);
    }

    const prevTrack = currentTrack;
    if (shuffle && prevTrack && trackQueueKey(prevTrack) !== trackQueueKey(tr)) {
      shuffleHistory = [...shuffleHistory.slice(-49), prevTrack];
    }
    shufflePlayedKeys.add(trackQueueKey(tr));
    const prevAcc = listenedAccumulator;
    const prevRecorded = recordedForCurrentTrack;
    if (prevTrack && prevAcc >= 3 && !prevRecorded) {
      void flushListenProgress(prevTrack.id, Math.round(prevAcc), false);
    }

    const seq = ++playRequestSeq;
    const resumeSeek = pendingResumeSeek > 0 && currentTrack && trackQueueKey(currentTrack) === trackQueueKey(tr) ? pendingResumeSeek : 0;
    pendingResumeSeek = 0;
    silentTailTicks = 0;
    trackLoading = true;
    currentTrack = tr;
    listenedAccumulator = 0;
    recordedForCurrentTrack = false;
    positionSec = resumeSeek;
    durationSec = tr.durationSec;
    loadLyricsForTrack(tr);
    saveLastQueue(true);

    let streamUrl = resolveDirectStreamUrl(tr);
    if (streamUrl) {
      if (tr.id > 0) {
        void api.getStreamUrl(tr.id).catch(() => {});
      }
    } else {
      try {
        streamUrl = await api.getStreamUrl(tr.id);
        if (tr.id > 0) atvCheckedTrackIds.add(tr.id);
        let resolvedVid = '';
        try {
          resolvedVid = new URL(streamUrl, window.location.origin).searchParams.get('v') || '';
        } catch {
        }
        if (seq === playRequestSeq && resolvedVid && resolvedVid !== tr.externalId) {
          const refreshedList = await api.getTracks('all', '');
          tracks = refreshedList;
          const upgraded =
            refreshedList.find((item) => item.id === tr.id || item.externalId === resolvedVid) ??
            { ...tr, externalId: resolvedVid };
          tr = upgraded;
          if (seq === playRequestSeq) {
            currentTrack = upgraded;
            if (upgraded.durationSec > 0) durationSec = upgraded.durationSec;
            queue = queue.map((q) => (q.id === upgraded.id || q.externalId === tr.externalId ? upgraded : q));
            void loadLyricsForTrack(
              upgraded,
              upgraded.durationSec > 0 ? upgraded.durationSec : undefined,
              true
            );
          }
        }
      } catch {
        if (seq === playRequestSeq) {
          trackLoading = false;
          isPlaying = false;
          showNetworkToast(appLang === 'id' ? 'Gagal memuat audio, periksa koneksi internet' : 'Failed to load audio, check your connection');
        }
        return;
      }
    }

    if (seq !== playRequestSeq) return;
    if (audioEl && streamUrl) {
      audioEngine.attach(audioEl);
      audioEngine.setTrackLoudness(null);
      const loudnessSeq = seq;
      void api.getTrackLoudness(tr.id, tr.externalId || '').then((res) => {
        if (loudnessSeq === playRequestSeq) {
          audioEngine.setTrackLoudness(res?.hasLoudness ? res.loudnessDb : null);
        }
      }).catch(() => {});
      switchingTrack = true;
      audioEl.pause();
      audioEl.src = streamUrl;
      audioEl.load();
      audioEngine.setVolume(volume, muted);
      try {
        await audioEngine.resume();
        const cfSec = Number(getPrefValue(preferences, 'playback.crossfadeSec'));
        audioEngine.fadeIn(Number.isFinite(cfSec) && cfSec > 0 ? Math.min(cfSec * 0.45, 2.5) : 0.03);
        if (resumeSeek > 0 && Number.isFinite(resumeSeek)) {
          audioEl.currentTime = resumeSeek;
        }
        await audioEl.play();
        if (seq === playRequestSeq) {
          isPlaying = true;
          trackLoading = false;
        }
      } catch (err: any) {
        if (err?.name !== 'AbortError' && seq === playRequestSeq) {
          isPlaying = false;
          trackLoading = false;
        }
      } finally {
        if (seq === playRequestSeq) {
          switchingTrack = false;
        }
      }
    } else if (seq === playRequestSeq) {
      trackLoading = false;
    }
  }

  async function flushListenProgress(trackId: number, sec: number, completed: boolean) {
    if (sec < 20) return;
    if (!getPrefBool(preferences, 'history.enabled') && getPrefValue(preferences, 'history.enabled') === 'false') {
      return;
    }
    fireStreak = await api.recordListenProgress(trackId, sec, completed);
    history = await api.getPlayHistory();
    listeningStats = await api.getListeningStats();
  }

  function handleAudioTimeUpdate() {
    if (!audioEl || !currentTrack) return;
    positionSec = audioEl.currentTime || 0;
    if (Number.isFinite(audioEl.duration) && audioEl.duration > 0) {
      durationSec = audioEl.duration;
      const realDur = Math.round(audioEl.duration);
      const curKey = trackQueueKey(currentTrack);
      if (realDur > 0 && lyricsLoadedTrackKey === curKey && Math.abs(realDur - lyricsLoadedForDur) >= 3) {
        void loadLyricsForTrack(currentTrack, realDur);
      }
    }
    api.updateDynamicIslandProgress(positionSec, durationSec);
    if (isPlaying) {
      listenedAccumulator += 0.25;
      if (listenedAccumulator >= 20 && !recordedForCurrentTrack) {
        recordedForCurrentTrack = true;
        flushListenProgress(currentTrack.id, 20, false);
      }
      saveLastQueue();

      if (durationSec > 0) {
        const remaining = durationSec - positionSec;
        const cfSec = Number(getPrefValue(preferences, 'playback.crossfadeSec'));
        if (Number.isFinite(cfSec) && cfSec > 0 && durationSec > cfSec * 2) {
          if (remaining <= cfSec && remaining > 0) {
            audioEngine.setFadeLevel(Math.max(0.05, remaining / cfSec));
          } else if (positionSec > Math.min(cfSec * 0.45, 2.5)) {
            audioEngine.setFadeLevel(1.0);
          }
        }
        if (!switchingTrack && getPrefBool(preferences, 'playback.skipSilence') && durationSec > 15 && remaining <= 5 && remaining > 0.4) {
          if (audioEngine.getRmsLevel() < 0.006) {
            silentTailTicks += 1;
            if (silentTailTicks >= 3) {
              silentTailTicks = 0;
              void handleAudioEnded();
            }
          } else {
            silentTailTicks = 0;
          }
        } else {
          silentTailTicks = 0;
        }
      }
    }
  }

  async function handleAudioEnded() {
    if (currentTrack) {
      const rem = Math.round(listenedAccumulator);
      if (!recordedForCurrentTrack && rem >= 20) {
        recordedForCurrentTrack = true;
        await flushListenProgress(currentTrack.id, rem, true);
      }
    }
    if (repeatMode === 'one' && currentTrack) {
      await playTrack(currentTrack, queue);
      return;
    }
    await playNext();
  }

  async function togglePlay() {
    if (isJamRemote) {
      await sendJamCommand('toggle');
      return;
    }
    if (!audioEl) return;
    if (!currentTrack && tracks.length > 0) {
      await playTrack(tracks[0], tracks);
      return;
    }
    if (currentTrack && !audioEl.src) {
      await playTrack(currentTrack, queue);
      return;
    }
    if (audioEl.paused) {
      audioEngine.attach(audioEl);
      await audioEngine.resume();
      audioEngine.fadeIn(0.025);
      await audioEl.play();
      isPlaying = true;
      await api.playerControl('play');
    } else {
      audioEl.pause();
      isPlaying = false;
      await api.playerControl('pause');
    }
  }

  function handleQueuePlayNext(tr: Track) {
    if (isJamRemote) {
      void sendJamCommand('play-next', {
        trackId: tr.id,
        externalId: tr.externalId,
        title: tr.title,
        artist: tr.artistName,
        album: tr.albumTitle,
        durationSec: tr.durationSec,
        coverUrl: tr.coverUrl
      });
    }
    showNetworkToast(appLang === 'id' ? 'Diputar berikutnya' : 'Playing next', 'check-circle');
    if (!currentTrack) {
      queue = [tr];
      if (!isJamRemote) void playTrack(tr, [tr]);
      return;
    }
    const curKey = trackQueueKey(currentTrack);
    const targetKey = trackQueueKey(tr);
    if (curKey === targetKey) return;
    const filtered = queue.filter((q) => trackQueueKey(q) !== targetKey);
    const curIdx = filtered.findIndex((q) => trackQueueKey(q) === curKey);
    const insertAt = curIdx >= 0 ? curIdx + 1 : 0;
    queue = [...filtered.slice(0, insertAt), tr, ...filtered.slice(insertAt)];
  }

  function handleAddToQueue(tr: Track) {
    if (isJamRemote) {
      void sendJamCommand('add-track', {
        trackId: tr.id,
        externalId: tr.externalId,
        title: tr.title,
        artist: tr.artistName,
        album: tr.albumTitle,
        durationSec: tr.durationSec,
        coverUrl: tr.coverUrl
      });
    }
    const targetKey = trackQueueKey(tr);
    if (!queue.some((q) => trackQueueKey(q) === targetKey)) {
      queue = [...queue, tr];
    }
    showNetworkToast(appLang === 'id' ? 'Ditambahkan ke antrean' : 'Added to queue', 'check-circle');
  }

  function handleCopyTrackLink(tr: Track) {
    const link = tr.externalId
      ? `https://music.youtube.com/watch?v=${tr.externalId}`
      : `${tr.title}${tr.artistName ? ` — ${tr.artistName}` : ''}`;
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(link).catch(() => {});
    } else if (typeof document !== 'undefined') {
      const ta = document.createElement('textarea');
      ta.value = link;
      ta.setAttribute('readonly', '');
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      try {
        document.execCommand('copy');
      } catch {}
      ta.remove();
    }
    showNetworkToast(appLang === 'id' ? 'Tautan disalin' : 'Link copied', 'check-circle');
  }

  async function playNext() {
    if (isJamRemote) {
      await sendJamCommand('next');
      return;
    }
    if (queue.length === 0) return;
    if (currentTrack) {
      shufflePlayedKeys.add(trackQueueKey(currentTrack));
    }
    if (shuffle && queue.length > 1) {
      const curKey = currentTrack ? trackQueueKey(currentTrack) : '';
      const curIdx = resolveActiveQueueIndex();
      let nextCandidate: Track | null = null;
      for (let step = 1; step <= queue.length; step++) {
        const cand = queue[(Math.max(0, curIdx) + step) % queue.length];
        const cKey = trackQueueKey(cand);
        if (cKey !== curKey && !shufflePlayedKeys.has(cKey)) {
          nextCandidate = cand;
          break;
        }
      }
      if (nextCandidate) {
        shufflePlayedKeys.add(trackQueueKey(nextCandidate));
        await playTrack(nextCandidate, queue, queueSourceTitle, queueSourceKey, true);
        return;
      }
      // All tracks in the current shuffle cycle have been played once.
      if (repeatMode === 'all') {
        let reshuffled = fisherYatesShuffle(queue);
        if (reshuffled.length > 1 && trackQueueKey(reshuffled[0]) === curKey) {
          const swapIdx = 1 + Math.floor(Math.random() * (reshuffled.length - 1));
          [reshuffled[0], reshuffled[swapIdx]] = [reshuffled[swapIdx], reshuffled[0]];
        }
        shufflePlayedKeys = new Set(curKey ? [curKey] : []);
        const pick = reshuffled[0];
        shufflePlayedKeys.add(trackQueueKey(pick));
        queue = reshuffled;
        await playTrack(pick, queue, queueSourceTitle, queueSourceKey, true);
        return;
      }
      if (currentTrack && getPrefBool(preferences, 'playback.autoplay') && queueSourceTitle === '') {
        algoBatchRound += 1;
        const nextBatch = await fetchTwentyGenreTracks(currentTrack, playedAlgoKeys, algoBatchRound);
        if (nextBatch.length > 0) {
          await playTrack(nextBatch[0], nextBatch, '');
          return;
        }
      }
      isPlaying = false;
      return;
    }
    const idx = resolveActiveQueueIndex();
    const nextIdx = idx + 1;
    if (nextIdx < queue.length) {
      await playTrack(queue[nextIdx], queue, queueSourceTitle, queueSourceKey, false, nextIdx);
    } else if (repeatMode === 'all' && queueSourceTitle !== '') {
      await playTrack(queue[0], queue, queueSourceTitle, queueSourceKey, false, 0);
    } else if (currentTrack && getPrefBool(preferences, 'playback.autoplay')) {
      if (queueSourceTitle !== '') {
        algoBatchRound = 0;
        playedAlgoKeys = new Set<string>(queue.map(trackQueueKey));
        playedAlgoKeys.add(trackQueueKey(currentTrack));
      } else {
        algoBatchRound += 1;
      }
      const nextBatch = await fetchTwentyGenreTracks(currentTrack, playedAlgoKeys, algoBatchRound);
      if (nextBatch.length > 0) {
        await playTrack(nextBatch[0], nextBatch, '');
      } else if (repeatMode === 'all') {
        await playTrack(queue[0], queue, queueSourceTitle, queueSourceKey, false, 0);
      } else {
        isPlaying = false;
      }
    } else if (repeatMode === 'all') {
      await playTrack(queue[0], queue, queueSourceTitle, queueSourceKey, false, 0);
    } else {
      isPlaying = false;
    }
  }

  async function playPrev() {
    if (isJamRemote) {
      await sendJamCommand('prev');
      return;
    }
    const restartThreshold = Number(getPrefValue(preferences, 'playback.prevRestartSec'));
    if (audioEl && Number.isFinite(restartThreshold) && restartThreshold > 0 && audioEl.currentTime > restartThreshold) {
      audioEl.currentTime = 0;
      return;
    }
    if (queue.length === 0) return;
    if (shuffle && shuffleHistory.length > 0) {
      const prevTrack = shuffleHistory[shuffleHistory.length - 1];
      shuffleHistory = shuffleHistory.slice(0, -1);
      await playTrack(prevTrack, queue, queueSourceTitle, queueSourceKey, true);
      return;
    }
    const idx = Math.max(0, resolveActiveQueueIndex());
    const prevIdx = idx - 1 >= 0 ? idx - 1 : queue.length - 1;
    await playTrack(queue[prevIdx], queue, queueSourceTitle, queueSourceKey, false, prevIdx);
  }

  function handleSeek(sec: number) {
    if (isJamRemote) {
      positionSec = sec;
      void sendJamCommand('seek', sec);
      return;
    }
    if (!audioEl) return;
    audioEl.currentTime = sec;
    positionSec = sec;
    api.playerControl('seek', sec);
  }

  function nudgeSeek(delta: number) {
    if (!currentTrack) return;
    const maxDur = durationSec > 0 ? durationSec : currentTrack.durationSec || 0;
    const targetSec = Math.max(0, maxDur > 0 ? Math.min(maxDur, positionSec + delta) : positionSec + delta);
    const actualDelta = targetSec - positionSec;
    if (Math.abs(actualDelta) < 0.05) {
      return;
    }
    handleSeek(targetSec);

    if ((delta > 0 && seekDeltaSec < 0) || (delta < 0 && seekDeltaSec > 0)) {
      seekDeltaSec = delta;
    } else {
      seekDeltaSec += delta;
    }
    seekDeltaSeq += 1;

    if (seekDeltaTimer) clearTimeout(seekDeltaTimer);
    seekDeltaTimer = setTimeout(() => {
      seekDeltaSec = 0;
    }, 1100);
  }

  let zoomLevel = $state(1);
  let zoomBadgeVisible = $state(false);
  let zoomBadgeTimer: ReturnType<typeof setTimeout> | null = null;
  let tooltipText = $state('');
  let tooltipVisible = $state(false);
  let tooltipX = $state(0);
  let tooltipY = $state(0);
  let tooltipBelow = $state(false);
  let tooltipTimer: ReturnType<typeof setTimeout> | null = null;
  let activeTooltipEl: HTMLElement | null = null;

  function applyZoom(next: number, showBadge = true) {
    const clamped = Math.round(Math.min(1.3, Math.max(0.7, next)) * 100) / 100;
    zoomLevel = clamped;
    if (preferences['general.zoomLevel'] !== String(clamped)) {
      preferences = { ...preferences, 'general.zoomLevel': String(clamped) };
    }
    try {
      localStorage.setItem('velora.zoom', String(clamped));
    } catch {
    }
    if (showBadge) {
      zoomBadgeVisible = true;
      if (zoomBadgeTimer) clearTimeout(zoomBadgeTimer);
      zoomBadgeTimer = setTimeout(() => {
        zoomBadgeVisible = false;
      }, 2000);
    }
  }

  function handleGlobalKeydown(e: KeyboardEvent) {
    if (e.ctrlKey || e.metaKey) {
      if (e.key.toLowerCase() === 'r' || e.code === 'KeyR') {
        e.preventDefault();
        window.location.reload();
        return;
      }
      if (e.key === '+' || e.key === '=' || e.code === 'NumpadAdd') {
        e.preventDefault();
        applyZoom(zoomLevel + 0.1);
        return;
      }
      if (e.key === '-' || e.key === '_' || e.code === 'NumpadSubtract') {
        e.preventDefault();
        applyZoom(zoomLevel - 0.1);
        return;
      }
      if (e.key === '0' || e.code === 'Numpad0') {
        e.preventDefault();
        applyZoom(1);
        return;
      }
    }

    if (e.key === 'Tab') {
      e.preventDefault();
      return;
    }

    const target = e.target as HTMLElement | null;
    const isTextInput =
      !!target &&
      ((target.tagName === 'INPUT' && (target as HTMLInputElement).type !== 'range') ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable);
    if (isTextInput) return;

    if (e.code === 'Space' || e.key === ' ') {
      e.preventDefault();
      if (currentTrack) togglePlay();
      return;
    }

    if (e.key === 'ArrowUp') {
      e.preventDefault();
      const base = muted ? 0 : volume;
      handleVolumeChange(Math.min(100, base + 5));
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      const base = muted ? 0 : volume;
      handleVolumeChange(Math.max(0, base - 5));
      return;
    }

    if (e.key === 'ArrowRight') {
      e.preventDefault();
      nudgeSeek(5);
      return;
    }

    if (e.key === 'ArrowLeft') {
      e.preventDefault();
      nudgeSeek(-5);
    }
  }

  function handleVolumeChange(vol: number) {
    volume = vol;
    muted = vol === 0;
    if (isJamRemote) {
      void sendJamCommand('volume', vol);
      return;
    }
    audioEngine.setVolume(volume, muted);
    api.playerControl('volume', vol);
  }

  function handleToggleMute() {
    muted = !muted;
    if (isJamRemote) {
      void sendJamCommand('mute');
      return;
    }
    audioEngine.setVolume(volume, muted);
    api.playerControl('mute');
  }

  let pendingCreatePlaylistTrack = $state<Track | null>(null);

  async function ensurePersistedTrack(track?: Track | null): Promise<Track | null> {
    if (!track) return null;
    if (track.id > 0) return track;
    if (!track.externalId) return null;
    const existing = tracks.find((t) => t.externalId === track.externalId);
    if (existing) return existing;
    const saved = await api.saveYTMusicTrack({
      videoId: track.externalId,
      title: track.title || 'Untitled',
      artist: track.artistName || 'Unknown Artist',
      album: track.albumTitle || 'Single',
      durationSec: track.durationSec || 0,
      coverUrl: track.coverUrl || '',
      format: track.format || 'OPUS'
    });
    if (saved) {
      tracks = tracks.some((t) => t.id === saved.id)
        ? tracks.map((t) => (t.id === saved.id ? saved : t))
        : [saved, ...tracks];
    }
    return saved ?? null;
  }

  function syncTrackAcrossViews(
    matchId: number,
    matchExternalId: string | undefined | null,
    updater: (t: Track) => Track
  ) {
    const ext = (matchExternalId || '').trim();
    const matches = (t: Track) =>
      (matchId > 0 && t.id === matchId) || (Boolean(ext) && t.externalId === ext);
    tracks = tracks.map((t) => (matches(t) ? updater(t) : t));
    detailTracks = detailTracks.map((t) => (matches(t) ? updater(t) : t));
    queue = queue.map((t) => (matches(t) ? updater(t) : t));
    if (currentTrack && matches(currentTrack)) {
      currentTrack = updater(currentTrack);
    }
    if (selectedPlaylistDetail) {
      selectedPlaylistDetail = {
        ...selectedPlaylistDetail,
        tracks: selectedPlaylistDetail.tracks.map((t) => (matches(t) ? updater(t) : t))
      };
    }
    if (ytAlbumDetail) {
      ytAlbumDetail = {
        ...ytAlbumDetail,
        tracks: ytAlbumDetail.tracks.map((t) => (matches(t) ? updater(t) : t))
      };
    }
    if (ytArtistDetail) {
      ytArtistDetail = {
        ...ytArtistDetail,
        topTracks: ytArtistDetail.topTracks.map((t) => (matches(t) ? updater(t) : t))
      };
    }
  }

  async function handleToggleFavoriteTrack(id: number, track?: Track) {
    const existing =
      track ??
      (currentTrack && currentTrack.id === id
        ? currentTrack
        : tracks.find((t) => t.id === id));
    const prevFav = Boolean(existing?.isFavorite);
    const nextFav = !prevFav;
    const matchExt = existing?.externalId;

    syncTrackAcrossViews(id, matchExt, (t) => ({ ...t, isFavorite: nextFav }));

    try {
      let targetId = id;
      if (targetId <= 0) {
        const saved = await ensurePersistedTrack(existing);
        if (!saved) {
          syncTrackAcrossViews(id, matchExt, (t) => ({ ...t, isFavorite: prevFav }));
          return;
        }
        targetId = saved.id;
      }
      const fav = await api.toggleFavoriteTrack(targetId);
      syncTrackAcrossViews(targetId, matchExt, (t) => ({ ...t, id: targetId, isFavorite: fav }));
    } catch {
      syncTrackAcrossViews(id, matchExt, (t) => ({ ...t, isFavorite: prevFav }));
    }
  }

  async function handleAddTrackToPlaylist(playlistId: number, trackId: number, track?: Track) {
    showNetworkToast(appLang === 'id' ? 'Ditambahkan ke playlist' : 'Added to playlist', 'check-circle');
    let targetId = trackId;
    if (targetId <= 0 && track) {
      const saved = await ensurePersistedTrack(track);
      if (!saved) return;
      targetId = saved.id;
      syncTrackAcrossViews(0, track.externalId, (t) => ({ ...t, id: targetId }));
    }
    if (targetId <= 0) return;
    await api.addTrackToPlaylist(playlistId, targetId);
    playlists = await api.getPlaylists(false);
    if (selectedPlaylistDetail && selectedPlaylistDetail.playlist.id === playlistId) {
      selectedPlaylistDetail = await api.getPlaylistDetail(playlistId);
    }
  }

  async function handleToggleFavoriteAlbum(id: number) {
    const fav = await api.toggleFavoriteAlbum(id);
    albums = albums.map((a) => (a.id === id ? { ...a, isFavorite: fav } : a));
  }

  async function handleToggleFavoriteYTMusicAlbum(payload: {
    browseId: string;
    title: string;
    artist: string;
    coverUrl: string;
    releaseYear: number;
  }) {
    const updated = await api.toggleFavoriteYTMusicAlbum(
      payload.browseId,
      payload.title,
      payload.artist,
      payload.coverUrl,
      payload.releaseYear
    );
    const exists = albums.some((a) => a.id === updated.id);
    albums = exists
      ? albums.map((a) => (a.id === updated.id ? updated : a))
      : [updated, ...albums];
  }

  async function handleToggleSaveYTMusicPlaylist(payload: {
    browseId: string;
    title: string;
    subtitle: string;
    coverUrl: string;
    trackIds: number[];
  }) {
    await api.toggleSaveYTMusicPlaylist(
      payload.browseId,
      payload.title,
      payload.subtitle,
      payload.coverUrl,
      payload.trackIds
    );
    playlists = await api.getPlaylists(false);
  }

  async function handleToggleFavoriteArtist(id: number) {
    const fav = await api.toggleFavoriteArtist(id);
    artists = artists.map((a) => (a.id === id ? { ...a, isFavorite: fav } : a));
  }

  async function handleToggleFavoriteYTMusicArtist(payload: {
    externalId: string;
    name: string;
    avatarUrl: string;
  }) {
    const updated = await api.toggleFavoriteYTMusicArtist(
      payload.externalId,
      payload.name,
      payload.avatarUrl
    );
    const exists = artists.some((a) => a.id === updated.id);
    artists = exists
      ? artists.map((a) => (a.id === updated.id ? updated : a))
      : [updated, ...artists];
  }

  async function handleCreatePlaylist(name: string, desc: string, isLocal: boolean, color: string) {
    const created = await api.createPlaylist(name, desc, isLocal, color);
    if (pendingCreatePlaylistTrack) {
      const pending = pendingCreatePlaylistTrack;
      pendingCreatePlaylistTrack = null;
      await handleAddTrackToPlaylist(created.id, pending.id, pending);
    } else {
      playlists = await api.getPlaylists(false);
    }
    await navigate({ name: 'playlist-detail', playlistId: created.id });
  }

  async function handleUploadLocalFiles() {
    const imported = await api.uploadLocalFiles();
    if (imported && imported.length > 0) {
      await refreshAll();
      sourcesStatus = `${imported.length} file audio berhasil di-upload ke Velora!`;
    }
  }

  async function handleAddLocalFolder() {
    const scanned = await api.addLocalFolder();
    if (scanned) {
      await refreshAll();
      sourcesStatus = `Folder berhasil di-scan (${scanned.length} lagu ditemukan).`;
    }
  }
  function handleGlobalDragStart(e: DragEvent) {
    const target = e.target as HTMLElement | null;
    if (!target) return;
    if (target.tagName === 'IMG' || !target.closest('[draggable="true"]')) {
      e.preventDefault();
    }
  }
</script>

<svelte:window onkeydown={handleGlobalKeydown} ondragstart={handleGlobalDragStart} />

<audio
  bind:this={audioEl}
  crossorigin="anonymous"
  preload="auto"
  ontimeupdate={handleAudioTimeUpdate}
  onended={handleAudioEnded}
  onwaiting={() => {
    if (currentTrack) trackLoading = true;
  }}
  onstalled={() => {
    if (currentTrack) {
      trackLoading = true;
      showNetworkToast(appLang === 'id' ? 'Koneksi lambat, sedang buffering...' : 'Slow connection, buffering...');
    }
  }}
  onerror={() => {
    trackLoading = false;
    isPlaying = false;
    if (currentTrack) {
      showNetworkToast(appLang === 'id' ? 'Gagal memutar audio, periksa koneksi internet' : 'Playback failed, check your connection');
    }
  }}
  oncanplay={() => {
    if (!switchingTrack) trackLoading = false;
  }}
  onplay={() => (isPlaying = true)}
  onplaying={() => {
    isPlaying = true;
    trackLoading = false;
  }}
  onpause={() => {
    if (!switchingTrack && !audioEl?.ended) isPlaying = false;
  }}
  class="hidden"
></audio>

<div
  class="flex flex-col overflow-hidden transition-colors duration-300"
  style="zoom: {zoomLevel}; width: {100 / zoomLevel}vw; height: {100 / zoomLevel}dvh; background-color: var(--md-sys-color-surface-container-lowest); color: var(--md-sys-color-on-surface);"
>
  {#if showSplash}
    <SplashScreen
      onComplete={() => {
        if (isJamRemote) {
          try {
            localStorage.setItem(SPLASH_SEEN_KEY, '1');
          } catch {
          }
        }
        showSplash = false;
      }}
    />
  {/if}

  <Titlebar
    {currentRoute}
    canGoBack={navIndex > 0}
    canGoForward={navIndex < navStack.length - 1}
    {searchQuery}
    {fireStreak}
    {accentColor}
    {tracks}
    {artists}
    {ytResults}
    {ytLoading}
    {mediaBaseUrl}
    {zoomLevel}
    {zoomBadgeVisible}
    showStreak={true}
    lang={appLang}
    jamRunning={jamStatus.running}
    {jamStatus}
    jamOpen={jamModalOpen}
    {jamLoading}
    {jamConnected}
    jamPrefEnabled={getPrefBool(preferences, 'integrations.jam.enabled')}
    {activeDownloadsCount}
    {downloadProgressPct}
    saveSearches={getPrefBool(preferences, 'privacy.saveSearches')}
    onOpenJamSettings={() => {
      activePrefTab = 'backup';
      void navigate({ name: 'preferences' });
    }}
    onOpenJam={() => (jamModalOpen = true)}
    onCloseJam={() => (jamModalOpen = false)}
    onToggleJam={() => void toggleJamServer()}
    onBack={goBack}
    onForward={goForward}
    onNavigate={navigate}
    onSearch={handleGlobalSearch}
    onSuggestSearch={handleSearchSuggestions}
    onPlayLocalTrack={(tr) => {
      void playTrack(tr, tracks);
      void openTrackAlbumDetail(tr);
    }}
    onPlayYTMusic={(yt) => {
      void playYTMusicSingle(yt);
      void navigate({
        name: 'yt-album-detail',
        videoId: yt.videoId,
        title: yt.title,
        artist: yt.artist,
        albumTitle: yt.album,
        coverUrl: yt.coverUrl
      });
    }}
    onToggleFavoriteArtist={handleToggleFavoriteArtist}
    onToggleFavoriteYTMusicArtist={handleToggleFavoriteYTMusicArtist}
    onOpenStreakModal={() => void navigate({ name: 'listening-stats' })}
    onResetZoom={() => applyZoom(1)}
  />

  <div
    class="flex-1 min-h-0 flex gap-2 pl-2 pb-2 overflow-hidden {isMobile
      ? 'pt-2 pr-2'
      : rightCollapsed && !lockRightHidden
        ? 'pr-0'
        : 'pr-2'}"
  >
    {#if !isMobile}
      <Sidebar
      {currentRoute}
      activeSection={activeSidebarSection}
      width={leftWidth}
      collapsed={lockLeftHidden ? true : leftCollapsed}
      lockedCollapsed={lockLeftHidden}
      {accentColor}
      favoriteAlbumsCount={albums.filter((a) => a.isFavorite).length}
      favoriteTracksCount={tracks.filter((t) => t.isFavorite).length}
      favoriteArtistsCount={artists.filter((a) => a.isFavorite).length}
      playlistsCount={playlists.length}
      downloadedCount={tracks.filter((t) => t.isDownloaded).length}
      sourcesCount={sources.length}
      isPreferencesMode={currentRoute.name === 'preferences'}
      bind:activePrefTab
      bind:prefSearch
      {prefSavedToast}
      lang={appLang}
      onBackFromPreferences={() => {
        if (navIndex > 0) goBack();
        else navigate({ name: 'home' });
      }}
      onClearCache={async () => {
        await api.clearCache();
        prefSavedToast = 'Artwork cache cleared';
        setTimeout(() => (prefSavedToast = ''), 1800);
      }}
      onNavigate={navigate}
      onToggleCollapse={() => {
        leftCollapsed = !leftCollapsed;
        patchPrefs({ 'layout.leftCollapsed': leftCollapsed ? 'true' : 'false' });
      }}
      onResizeWidth={(w) => (leftWidth = w)}
      onResizeEnd={(w) => patchPrefs({ 'layout.leftSidebarWidth': String(Math.round(w)) })}
    />
    {/if}

    <main
      bind:this={mainScrollEl}
      class="flex-1 min-w-0 min-h-0 {isFullHeightRoute ? 'overflow-hidden' : 'overflow-y-auto overflow-x-hidden'} scroll-pt-16 rounded-lg transition-colors duration-300"
      style="background-color: var(--md-sys-color-surface);"
    >
      <div class="velora-route-stage {isFullHeightRoute ? 'h-full' : isMobile ? 'pb-20' : ''}">
        {#key routeTransitionKey}
          <div
            in:fly={{ x: isMobile ? 16 : 0, y: isMobile ? 0 : 10, duration: 260, opacity: 0, easing: cubicOut }}
            out:routeExitFade={{ duration: 180, shiftX: isMobile ? -12 : 0 }}
            class="velora-route-pane {isFullHeightRoute ? 'h-full min-h-0' : ''}"
          >
      {#if currentRoute.name === 'home'}
        <HomeView
          lang={appLang}
          showGreeting={getPrefBool(preferences, 'appearance.showGreeting')}
          showRecentHome={getPrefBool(preferences, 'appearance.showRecentHome')}
          meltCards={getPrefBool(preferences, 'appearance.meltCards')}
          {albums}
          {artists}
          {tracks}
          {playlists}
          {downloadedGroupKeys}
          {ytShelves}
          {ytResults}
          {ytPlaylistResults}
          {ytAlbumResults}
          {ytArtistResults}
          initialFilter={currentRoute.filter ?? homeFilter}
          onFilterChange={(f) => (homeFilter = f)}
          {ytLoading}
          {ytHomeLoading}
          {ytLoadingMore}
          {ytLoadingMorePopularArtists}
          {loadingCollectionId}
          currentExternalId={currentTrack?.externalId ?? null}
          currentTrackTitle={currentTrack?.title ?? null}
          currentTrackArtist={currentTrack?.artistName ?? null}
          currentAlbumTitle={currentTrack?.albumTitle ?? null}
          {queueSourceTitle}
          {queueSourceKey}
          {isPlaying}
          {trackLoading}
          onTogglePlay={togglePlay}
          {mediaBaseUrl}
          onPlayYTMusic={playYTMusicSingle}
          onOpenYTSong={openYTSongCard}
          onPlayYTCollection={handlePlayYTCollection}
          onOpenYTAlbum={openYTAlbumCard}
          onPlayPlaylist={handlePlayPlaylist}
          onPlayAlbum={handlePlayAlbum}
          onOpenArtist={(artistId) => navigate({ name: 'artist-detail', artistId })}
          onToggleFavoriteArtist={handleToggleFavoriteArtist}
          onToggleFavoriteYTMusicArtist={handleToggleFavoriteYTMusicArtist}
          onSearchYTMusic={handleSearchYTMusic}
          onSearchYTMusicPlaylists={handleSearchYTMusicPlaylists}
          onSearchYTMusicAlbums={handleSearchYTMusicAlbums}
          onSearchYTMusicArtists={handleSearchYTMusicArtists}
          onRefreshYTMusicHome={loadYTMusicHome}
          onLoadMoreYTMusic={handleLoadMoreYTMusic}
          onLoadMorePopularArtists={handleLoadMorePopularArtists}
          onCreatePlaylist={() => (showCreatePlaylistModal = true)}
          onPlayNext={handleQueuePlayNext}
          onAddToQueue={handleAddToQueue}
          onAddToPlaylist={handleAddTrackToPlaylist}
          onCreatePlaylistTrack={(tr) => {
            pendingCreatePlaylistTrack = tr ?? null;
            showCreatePlaylistModal = true;
          }}
          onToggleFavoriteTrack={handleToggleFavoriteTrack}
          onDownloadTrack={handleDownloadTrack}
          onRemoveDownload={handleRemoveDownload}
          onOpenTrackAlbum={(tr) => void openTrackAlbumDetail(tr)}
          {activeDownloadsCount}
          {downloadProgressPct}
          onNavigate={navigate}
        />
      {:else if currentRoute.name === 'downloads'}
        {#await import('./views/MobileDownloadsView.svelte') then { default: MobileDownloadsView }}
        <MobileDownloadsView
          lang={appLang}
          activeDownloads={activeDownloadsList}
          downloadedTracks={tracks.filter((t) => t.isDownloaded)}
          deletedTracks={sessionDeletedDownloadTracks}
          allTracks={tracks}
          {mediaBaseUrl}
          onBack={goBack}
          onPlayTrack={playTrack}
          onOpenTrackAlbum={openTrackAlbumDetail}
          onOpenDownloadedMusic={(tr) => navigate({ name: 'track-detail', trackId: tr.id, singleTrack: true })}
          onOpenGroup={(entry) => {
            const groupKey = entry.key.startsWith('group:') ? entry.key.slice('group:'.length) : entry.key;
            if (entry.kind === 'playlist') {
              const rawId = groupKey.startsWith('playlist:') ? groupKey.slice('playlist:'.length).trim() : '';
              const normPlId = (s: string) => s.trim().replace(/^VL/i, '');
              const numId = Number(rawId);
              const match = playlists.find(
                (p) =>
                  (Number.isFinite(numId) && numId > 0 && p.id === numId) ||
                  (rawId !== '' && normPlId(p.externalId || '') === normPlId(rawId)) ||
                  p.name.trim().toLowerCase() === entry.title.trim().toLowerCase()
              );
              if (match) {
                navigate({ name: 'playlist-detail', playlistId: match.id });
              } else {
                navigate({
                  name: 'yt-album-detail',
                  videoId: entry.primaryTrack.externalId || '',
                  browseId: rawId,
                  title: entry.title,
                  artist: entry.primaryTrack.artistName || 'YouTube Music',
                  albumTitle: entry.title,
                  coverUrl: entry.coverUrl || entry.primaryTrack.coverUrl || '',
                  kind: 'playlist',
                  localTracks: entry.tracks
                });
              }
              return;
            }
            const rawAlbumId = groupKey.startsWith('album:') ? groupKey.slice('album:'.length).trim() : '';
            const numAlbumId = Number(rawAlbumId);
            if (Number.isFinite(numAlbumId) && numAlbumId > 0 && albums.some((a) => a.id === numAlbumId)) {
              navigate({ name: 'album-detail', albumId: numAlbumId });
            } else {
              navigate({
                name: 'yt-album-detail',
                videoId: entry.primaryTrack.externalId || '',
                browseId: rawAlbumId,
                title: entry.title,
                artist: entry.primaryTrack.artistName || '',
                albumTitle: entry.title,
                coverUrl: entry.coverUrl || entry.primaryTrack.coverUrl || '',
                kind: 'album',
                localTracks: entry.tracks
              });
            }
          }}
          onCancelDownload={handleCancelDownload}
          onCancelDownloadGroup={handleCancelDownloadGroup}
          onPauseDownload={handlePauseDownload}
          onResumeDownload={handleResumeDownload}
          onPauseDownloadGroup={handlePauseDownloadGroup}
          onResumeDownloadGroup={handleResumeDownloadGroup}
          onRemoveDownload={(id) => handleRemoveDownload(id, false)}
          onRemoveDownloadGroup={(list) => handleRemoveDownloadGroup(list, false)}
          onDismissDeletedEntry={(entry) => {
            const idSet = new Set(entry.tracks.map((t) => t.id));
            sessionDeletedDownloadTracks = sessionDeletedDownloadTracks.filter((d) => !idSet.has(d.id));
          }}
          onRetryFailedEntry={handleRetryFailedDownload}
        />
        {/await}
      {:else if currentRoute.name === 'library' || (isMobile && (currentRoute.name === 'liked-songs' || currentRoute.name === 'downloaded' || currentRoute.name === 'favorite-albums' || currentRoute.name === 'favorite-artists'))}
        {#await import('./views/MobileLibraryView.svelte') then { default: MobileLibraryView }}
        <MobileLibraryView
          lang={appLang}
          {tracks}
          {albums}
          {artists}
          {playlists}
          initialFilter={currentRoute.name === 'library'
            ? (currentRoute.filter ?? 'all')
            : currentRoute.name === 'liked-songs'
              ? 'music'
              : currentRoute.name === 'downloaded'
                ? 'downloaded'
                : currentRoute.name === 'favorite-albums'
                  ? 'albums'
                  : 'artists'}
          currentTrackId={currentTrack?.id ?? null}
          currentExternalId={currentTrack?.externalId ?? null}
          {isPlaying}
          {shuffle}
          {mediaBaseUrl}
          {activeDownloadsCount}
          {downloadProgressPct}
          onPlayTrack={playTrack}
          onTogglePlay={togglePlay}
          onPlayShuffled={playShuffledList}
          onPlayNext={handleQueuePlayNext}
          onAddToQueue={handleAddToQueue}
          onAddToPlaylist={handleAddTrackToPlaylist}
          onCreatePlaylistTrack={(tr) => {
            pendingCreatePlaylistTrack = tr ?? null;
            showCreatePlaylistModal = true;
          }}
          onToggleFavoriteTrack={handleToggleFavoriteTrack}
          onToggleFavoriteAlbum={handleToggleFavoriteAlbum}
          onToggleFavoriteArtist={handleToggleFavoriteArtist}
          onDownloadTrack={handleDownloadTrack}
          onRemoveDownload={handleRemoveDownload}
          onOpenAlbum={handleOpenAlbum}
          onPlayAlbum={handlePlayAlbum}
          onOpenArtist={(artistId) => navigate({ name: 'artist-detail', artistId })}
          onOpenTrackAlbum={openTrackAlbumDetail}
          onNavigate={navigate}
        />
        {/await}
      {:else if currentRoute.name === 'search' || currentRoute.name === 'liked-songs' || currentRoute.name === 'downloaded' || currentRoute.name === 'favorite-albums' || currentRoute.name === 'favorite-artists' || currentRoute.name === 'artist-detail'}
        {#await import('./views/LibraryViews.svelte') then { default: LibraryViews }}
        <LibraryViews
          lang={appLang}
          mode={currentRoute.name}
          onBack={() => (navIndex > 0 ? goBack() : void navigate({ name: 'home' }))}
          {searchQuery}
          {tracks}
          {albums}
          {artists}
          selectedArtist={currentRoute.name === 'artist-detail'
            ? artists.find(
                (a) =>
                  (currentRoute.artistId !== undefined && a.id === currentRoute.artistId) ||
                  (Boolean(currentRoute.browseId) && a.externalId === currentRoute.browseId) ||
                  (Boolean(ytArtistDetail?.browseId) && a.externalId === ytArtistDetail?.browseId) ||
                  (Boolean(currentRoute.artistName) && a.name.toLowerCase() === currentRoute.artistName!.toLowerCase()) ||
                  (Boolean(ytArtistDetail?.name) && a.name.toLowerCase() === ytArtistDetail!.name.toLowerCase())
              ) ?? null
            : null}
          artistFallbackName={currentRoute.name === 'artist-detail' ? currentRoute.artistName ?? '' : ''}
          artistFallbackAvatar={currentRoute.name === 'artist-detail' ? currentRoute.avatarUrl ?? '' : ''}
          artistFallbackBrowseId={currentRoute.name === 'artist-detail' ? currentRoute.browseId ?? '' : ''}
          {ytArtistDetail}
          {ytArtistLoading}
          {detailTracks}
          {ytResults}
          {ytPlaylistResults}
          {ytAlbumResults}
          {ytArtistResults}
          {ytLoading}
          searchStart={(getPrefValue(preferences, 'content.searchStart') as 'song' | 'album' | 'artist' | 'playlist' | 'video') || 'song'}
          {playlists}
          {downloadStates}
          {downloadedGroupKeys}
          currentTrackId={currentTrack?.id ?? null}
          currentExternalId={currentTrack?.externalId ?? null}
          currentTrackTitle={currentTrack?.title ?? null}
          currentTrackArtist={currentTrack?.artistName ?? null}
          {isPlaying}
          onTogglePlay={togglePlay}
          {mediaBaseUrl}
          {accentColor}
          onPlayTrack={playTrack}
          onPlayNext={handleQueuePlayNext}
          onAddToQueue={handleAddToQueue}
          onCopyLink={handleCopyTrackLink}
          onPlayAlbum={handlePlayAlbum}
          onPlayArtist={handlePlayArtist}
          onPlayYTMusic={playYTMusicSingle}
          onSaveYTMusic={async (yt) => {
            await api.saveYTMusicTrack(yt);
            await refreshAll();
          }}
          onSearchYTMusic={handleSearchYTMusic}
          onSearchYTMusicPlaylists={handleSearchYTMusicPlaylists}
          onSearchYTMusicAlbums={handleSearchYTMusicAlbums}
          onSearchYTMusicArtists={handleSearchYTMusicArtists}
          onOpenYTCard={openYTAlbumCard}
          onPlayYTCollection={handlePlayYTCollection}
          onToggleFavoriteTrack={handleToggleFavoriteTrack}
          onToggleFavoriteAlbum={handleToggleFavoriteAlbum}
          onToggleFavoriteArtist={handleToggleFavoriteArtist}
          onToggleFavoriteYTMusicArtist={handleToggleFavoriteYTMusicArtist}
          onAddToPlaylist={handleAddTrackToPlaylist}
          onCreatePlaylistTrack={(tr) => {
            pendingCreatePlaylistTrack = tr ?? null;
            showCreatePlaylistModal = true;
          }}
          {trackLoading}
          onDownloadTrack={handleDownloadTrack}
          onDownloadYTMusic={handleDownloadYTMusicTrack}
          onRemoveDownload={handleRemoveDownload}
          onRemoveDownloadGroup={handleRemoveDownloadGroup}
          onCancelDownload={handleCancelDownload}
          onCancelDownloadGroup={handleCancelDownloadGroup}
          onOpenAlbum={handleOpenAlbum}
          onOpenArtist={(artistId, tr) =>
            tr ? void openTrackArtistDetail(tr) : void navigate({ name: 'artist-detail', artistId })}
          onOpenTrackAlbum={openTrackAlbumDetail}
          {shuffle}
          {queueSourceKey}
          onToggleShuffle={toggleShuffleMode}
          onPlayShuffled={playShuffledList}
          onOpenYTMusicAlbum={(yt) =>
            navigate({
              name: 'yt-album-detail',
              videoId: yt.videoId,
              title: yt.title,
              artist: yt.artist,
              albumTitle: yt.album,
              coverUrl: yt.coverUrl
            })}
          onNavigate={navigate}
        />
        {/await}
      {:else if currentRoute.name === 'playlists' || currentRoute.name === 'local-playlists' || currentRoute.name === 'playlist-detail'}
        {#await import('./views/PlaylistViews.svelte') then { default: PlaylistViews }}
        <PlaylistViews
          lang={appLang}
          mode={currentRoute.name}
          {playlists}
          {playlistsLoading}
          {downloadedGroupKeys}
          detail={selectedPlaylistDetail}
          detailLoading={playlistDetailLoading}
          allTracks={tracks}
          {ytTop100Tracks}
          {ytTop100Loading}
          onRefreshRecommendations={() => {
            const exclude = selectedPlaylistDetail?.tracks.map((t) => t.externalId).filter(Boolean) ?? [];
            void loadPlaylistRecommendations(exclude);
          }}
          currentTrackId={currentTrack?.id ?? null}
          currentExternalId={currentTrack?.externalId ?? null}
          currentTrackTitle={currentTrack?.title ?? null}
          currentTrackArtist={currentTrack?.artistName ?? null}
          {isPlaying}
          {trackLoading}
          {downloadStates}
          onTogglePlay={togglePlay}
          {mediaBaseUrl}
          {accentColor}
          onOpenPlaylist={(playlistId) => navigate({ name: 'playlist-detail', playlistId })}
          onPlayPlaylist={handlePlayPlaylist}
          onCreatePlaylist={async () => {
            let n = playlists.length + 1;
            while (playlists.some((p) => p.name.trim().toLowerCase() === `playlist #${n}`)) {
              n += 1;
            }
            await handleCreatePlaylist(`Playlist #${n}`, '', true, accentColor);
          }}
          onRenamePlaylist={async (id, name, desc, color) => {
            await api.updatePlaylist(id, name, desc, color);
            playlists = await api.getPlaylists(false);
            selectedPlaylistDetail = await api.getPlaylistDetail(id);
          }}
          onDeletePlaylist={async (id) => {
            await api.deletePlaylist(id);
            playlists = await api.getPlaylists(false);
            await navigate({ name: 'playlists' });
          }}
          onToggleSaveYTMusicPlaylist={handleToggleSaveYTMusicPlaylist}
          onPlayTrack={playTrack}
          {shuffle}
          {queueSourceKey}
          onToggleShuffle={toggleShuffleMode}
          onPlayShuffled={playShuffledList}
          onPlayYTMusic={playYTMusicSingle}
          onAddTrack={handleAddTrackToPlaylist}
          onCreatePlaylistTrack={(tr) => {
            pendingCreatePlaylistTrack = tr ?? null;
            showCreatePlaylistModal = true;
          }}
          onAddYTMusicTrack={async (pid, yt) => {
            const saved = await api.saveYTMusicTrack(yt);
            await api.addTrackToPlaylist(pid, saved.id);
            playlists = await api.getPlaylists(false);
            selectedPlaylistDetail = await api.getPlaylistDetail(pid);
            tracks = await api.getTracks('all', '');
          }}
          onRemoveTrack={async (pid, tid) => {
            await api.removeTrackFromPlaylist(pid, tid);
            playlists = await api.getPlaylists(false);
            selectedPlaylistDetail = await api.getPlaylistDetail(pid);
          }}
          onReorderTracks={async (pid, ids) => {
            await api.reorderPlaylistTracks(pid, ids);
            selectedPlaylistDetail = await api.getPlaylistDetail(pid);
          }}
          onToggleFavorite={handleToggleFavoriteTrack}
          onPlayNext={handleQueuePlayNext}
          onAddToQueue={handleAddToQueue}
          onCopyLink={handleCopyTrackLink}
          onDownloadTrack={handleDownloadTrack}
          onDownloadCollection={handleDownloadCollection}
          onRemoveDownload={handleRemoveDownload}
          onRemoveDownloadGroup={handleRemoveDownloadGroup}
          onOpenArtist={(artistId, tr) =>
            tr ? void openTrackArtistDetail(tr) : void navigate({ name: 'artist-detail', artistId })}
          onOpenTrackAlbum={openTrackAlbumDetail}
          {activeDownloadsCount}
          {downloadProgressPct}
          onNavigate={navigate}
          onBack={goBack}
          onImportM3U={async () => {
            const pl = await api.importLocalPlaylistM3U();
            if (pl?.id) {
              playlists = await api.getPlaylists(false);
              await refreshAll();
              await navigate({ name: 'playlist-detail', playlistId: pl.id });
            }
          }}
          onImportSpotify={async (url) => {
            const pl = await api.importSpotifyPlaylist(url);
            if (pl?.id) {
              playlists = await api.getPlaylists(false);
              await refreshAll();
              await navigate({ name: 'playlist-detail', playlistId: pl.id });
            }
          }}
        />
        {/await}
      {:else if currentRoute.name === 'sources'}
        {#await import('./views/SourcesView.svelte') then { default: SourcesView }}
        <SourcesView
          {sources}
          {ytResults}
          {ytLoading}
          statusMessage={sourcesStatus}
          {accentColor}
          onUploadFiles={handleUploadLocalFiles}
          onAddFolder={handleAddLocalFolder}
          onRescanSource={async (id) => {
            const found = await api.rescanSource(id);
            await refreshAll();
            sourcesStatus = `Scan ulang selesai (${found.length} lagu).`;
          }}
          onRemoveSource={async (id) => {
            await api.removeSource(id);
            sources = await api.getSources();
          }}
          onSearchYTMusic={handleSearchYTMusic}
          onSaveYTMusic={async (yt) => {
            await api.saveYTMusicTrack(yt);
            await refreshAll();
            sourcesStatus = `"${yt.title}" ditambahkan ke Library.`;
          }}
          onPlayYTMusic={async (yt) => {
            const tr = await api.playYTMusicTrack(yt);
            await refreshAll();
            await playTrack(tr, [tr, ...tracks]);
          }}
        />
        {/await}
      {:else if currentRoute.name === 'track-detail' || currentRoute.name === 'album-detail' || currentRoute.name === 'yt-album-detail'}
        {@const detailTrack =
          currentRoute.name === 'track-detail'
            ? (currentTrack?.id === currentRoute.trackId ? currentTrack : null) ??
              tracks.find((t) => t.id === currentRoute.trackId) ??
              queue.find((t) => t.id === currentRoute.trackId) ??
              currentTrack
            : currentRoute.name === 'yt-album-detail'
              ? currentTrack
              : null}
        {@const targetAlbumId =
          currentRoute.name === 'album-detail'
            ? currentRoute.albumId
            : currentRoute.name === 'track-detail' && !currentRoute.singleTrack
              ? detailTrack?.albumId ?? 0
              : 0}
        {@const detailAlbum = targetAlbumId > 0 ? albums.find((a) => a.id === targetAlbumId) ?? null : null}
        {@const albumTrackList =
          currentRoute.name === 'album-detail'
            ? detailTracks
            : currentRoute.name === 'yt-album-detail'
              ? ytAlbumDetail?.tracks ?? (currentTrack ? [currentTrack] : [])
              : targetAlbumId > 0
                ? tracks.filter((t) => t.albumId === targetAlbumId)
                : detailTrack
                  ? [detailTrack]
                  : []}
        {#await import('./components/TrackDetailView.svelte') then { default: TrackDetailView }}
        <TrackDetailView
          lang={appLang}
          track={detailTrack ?? albumTrackList[0] ?? null}
          album={detailAlbum}
          albumTracks={albumTrackList}
          ytAlbum={currentRoute.name === 'yt-album-detail' ? ytAlbumDetail : null}
          ytAlbumLoading={currentRoute.name === 'yt-album-detail' ? ytAlbumLoading : false}
          routeKind={currentRoute.name === 'yt-album-detail'
            ? currentRoute.kind
            : currentRoute.name === 'track-detail' && currentRoute.singleTrack
              ? 'music'
              : undefined}
          recommendedPlaylists={currentRoute.name === 'yt-album-detail' && currentRoute.kind === 'playlist'
            ? [...ytShelves.flatMap((s) => s.items).filter((i) => i.kind === 'playlist'), ...ytPlaylistResults]
            : []}
          currentTrackId={currentTrack?.id ?? null}
          currentExternalId={currentTrack?.externalId ?? null}
          currentTrackTitle={currentTrack?.title ?? null}
          currentTrackArtist={currentTrack?.artistName ?? null}
          {isPlaying}
          {trackLoading}
          {tracks}
          {albums}
          {playlists}
          {downloadStates}
          {mediaBaseUrl}
          {accentColor}
          onPlayTrack={playTrack}
          {shuffle}
          {queueSourceKey}
          onToggleShuffle={toggleShuffleMode}
          onPlayShuffled={playShuffledList}
          onTogglePlay={() => void togglePlay()}
          onToggleFavoriteTrack={handleToggleFavoriteTrack}
          onToggleFavoriteAlbum={handleToggleFavoriteAlbum}
          onToggleFavoriteYTMusicAlbum={handleToggleFavoriteYTMusicAlbum}
          onToggleSaveYTMusicPlaylist={handleToggleSaveYTMusicPlaylist}
          onAddToPlaylist={handleAddTrackToPlaylist}
          onCreatePlaylistTrack={(tr) => {
            pendingCreatePlaylistTrack = tr ?? null;
            showCreatePlaylistModal = true;
          }}
          onOpenArtist={(artistId, tr) =>
            tr ? void openTrackArtistDetail(tr) : void navigate({ name: 'artist-detail', artistId })}
          onOpenAlbum={handleOpenAlbum}
          onOpenTrackAlbum={openTrackAlbumDetail}
          onBack={goBack}
          onOpenYTAlbum={(card) => void openYTAlbumCard(card)}
          onDownloadTrack={handleDownloadTrack}
          onDownloadCollection={handleDownloadCollection}
          onRemoveDownload={handleRemoveDownload}
          onRemoveDownloadGroup={handleRemoveDownloadGroup}
          onPlayNext={handleQueuePlayNext}
          onAddToQueue={handleAddToQueue}
          onCopyLink={handleCopyTrackLink}
        />
        {/await}
      {:else if currentRoute.name === 'lyrics'}
        {#await import('./components/LyricsView.svelte') then { default: LyricsView }}
        <LyricsView
          track={currentTrack}
          lyrics={currentLyrics}
          {lyricsLoading}
          {isPlaying}
          {positionSec}
          {mediaBaseUrl}
          {preferences}
          onSeek={handleSeek}
          onResync={() => currentTrack && loadLyricsForTrack(currentTrack, durationSec > 0 ? Math.round(durationSec) : undefined, true)}
          onClose={() => {
            if (navIndex > 0) goBack();
            else navigate({ name: 'home' });
          }}
        />
        {/await}
      {:else}
        {#await import('./views/AnalyticsAndSettingsViews.svelte') then { default: AnalyticsAndSettingsViews }}
        <AnalyticsAndSettingsViews
          mode={currentRoute.name}
          {history}
          {listeningStats}
          {fireStreak}
          {preferences}
          {tracks}
          {playlists}
          {downloadStates}
          currentTrackId={currentTrack?.id ?? null}
          currentExternalId={currentTrack?.externalId ?? null}
          currentTrackTitle={currentTrack?.title ?? null}
          currentTrackArtist={currentTrack?.artistName ?? null}
          {isPlaying}
          {trackLoading}
          {mediaBaseUrl}
          {accentColor}
          {isMobile}
          bind:activePrefTab
          bind:prefSearch
          onBackFromPreferences={() => {
            if (navIndex > 0) goBack();
            else navigate({ name: 'home' });
          }}
          onPlayTrack={playTrack}
          onTogglePlay={togglePlay}
          onPlayNext={handleQueuePlayNext}
          onAddToQueue={handleAddToQueue}
          onCopyLink={handleCopyTrackLink}
          onAddToPlaylist={handleAddTrackToPlaylist}
          onCreatePlaylistTrack={(tr) => {
            pendingCreatePlaylistTrack = tr ?? null;
            showCreatePlaylistModal = true;
          }}
          onToggleFavorite={handleToggleFavoriteTrack}
          onDownloadTrack={handleDownloadTrack}
          onRemoveDownload={handleRemoveDownload}
          onOpenArtist={(artistId, tr) =>
            tr ? void openTrackArtistDetail(tr) : void navigate({ name: 'artist-detail', artistId })}
          onOpenTrackAlbum={openTrackAlbumDetail}
          onClearHistory={async () => {
            await api.clearPlayHistory();
            history = [];
            listeningStats = await api.getListeningStats();
          }}
          onSavePreferences={async (prefs) => {
            await patchPrefs(prefs);
            fireStreak = await api.getFireStreak();
          }}
          onClearCache={async () => {
            return await api.clearCache();
          }}
          onClearDownloads={async () => {
            const freed = await api.clearAllDownloads();
            downloadStates = {};
            sessionDeletedDownloadTracks = [];
            tracks = await api.getTracks('all', '');
            return freed;
          }}
          onNavigate={navigate}
        />
        {/await}
      {/if}
          </div>
        {/key}
      </div>
    </main>

    {#if !lockRightHidden && !isMobile}
      <RightPanel
        width={rightWidth}
        collapsed={rightCollapsed}
        mode={rightPanelMode}
        {currentTrack}
        {queue}
        queueIndex={resolveActiveQueueIndex()}
        {queueSourceTitle}
        {audioOutputs}
        {switchingOutputId}
        {isPlaying}
        {positionSec}
        lyricOffsetSec={currentLyricOffsetSec}
        lyrics={currentLyrics}
        {lyricsLoading}
        {preferences}
        {mediaBaseUrl}
        {accentColor}
        focusCardSignal={focusRightSignal}
        onChangeMode={(m) => (rightPanelMode = m)}
        onSelectOutputDevice={(id) => void handleSelectOutputDevice(id)}
        onRefreshAudioOutputs={() => void refreshAudioOutputs()}
        onToggleCollapse={() => {
          rightCollapsed = !rightCollapsed;
          patchPrefs({ 'layout.rightCollapsed': rightCollapsed ? 'true' : 'false' });
        }}
        onResizeWidth={(w) => (rightWidth = w)}
        onResizeEnd={(w) => patchPrefs({ 'layout.rightSidebarWidth': String(Math.round(w)) })}
        onPlayTrack={playTrack}
        onRemoveFromQueue={(id) => {
          queue = queue.filter((t) => t.id !== id);
        }}
        onClearQueue={() => {
          queue = [];
        }}
        onToggleFavorite={handleToggleFavoriteTrack}
        onSeek={handleSeek}
        onOpenTrackDetail={(tr) => void openTrackAlbumDetail(tr)}
        onOpenArtistDetail={(tr) => void openTrackArtistDetail(tr)}
        onOpenLyrics={() => navigate({ name: 'lyrics' })}
        onResyncLyrics={() => currentTrack && void loadLyricsForTrack(currentTrack, durationSec > 0 ? Math.round(durationSec) : undefined, true)}
        onOpenQueueSource={queueSourceKey || inferQueueSourceKeyByTitle(queueSourceTitle)
          ? openQueueSource
          : undefined}
      />
    {/if}
  </div>

  <div class="w-full shrink-0">
    <PlayerBar
      lang={appLang}
      {currentTrack}
      {isPlaying}
      {trackLoading}
      {positionSec}
      {durationSec}
      {volume}
      {muted}
      {shuffle}
      {repeatMode}
      {rightCollapsed}
      {rightPanelMode}
      {activeOutputDevice}
      isLyricsOpen={currentRoute.name === 'lyrics'}
      spatial8d={getPrefBool(preferences, 'playback.spatial8d')}
      {seekDeltaSec}
      {seekDeltaSeq}
      {mediaBaseUrl}
      {accentColor}
      waveBar={getPrefBool(preferences, 'playback.waveBar')}
      onTogglePlay={togglePlay}
      onPrev={playPrev}
      onNext={playNext}
      onSeek={handleSeek}
      onVolumeChange={handleVolumeChange}
      onToggleMute={handleToggleMute}
      onToggleShuffle={toggleShuffleMode}
      onCycleRepeat={() => {
        repeatMode = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
        if (isJamRemote) void sendJamCommand('repeat');
        else api.playerControl('repeat', 0, repeatMode);
      }}
      onToggleFavorite={handleToggleFavoriteTrack}
      onOpenTrackDetail={(tr) => void openTrackAlbumDetail(tr)}
      onOpenArtistDetail={(tr) => void openTrackArtistDetail(tr)}
      onOpenMobileNowPlaying={() => (mobileNowPlayingOpen = true)}
      onToggleLyrics={() => {
        if (currentRoute.name === 'lyrics') {
          if (navIndex > 0) goBack();
          else navigate({ name: 'home' });
        } else {
          navigate({ name: 'lyrics' });
        }
      }}
      onToggleSpatial8d={() => {
        const next = !getPrefBool(preferences, 'playback.spatial8d');
        void patchPrefs({ 'playback.spatial8d': next ? 'true' : 'false' });
      }}
      onToggleDevices={() => {
        if (lockRightHidden) return;
        if (!rightCollapsed && rightPanelMode === 'devices') {
          rightCollapsed = true;
          patchPrefs({ 'layout.rightCollapsed': 'true' });
        } else {
          rightCollapsed = false;
          rightPanelMode = 'devices';
          patchPrefs({ 'layout.rightCollapsed': 'false' });
          void refreshAudioOutputs();
        }
      }}
      onFocusRightCard={(card) => {
        if (lockRightHidden) return;
        if (card === 'queue') {
          if (!rightCollapsed && rightPanelMode === 'queue') {
            rightCollapsed = true;
            patchPrefs({ 'layout.rightCollapsed': 'true' });
          } else {
            rightCollapsed = false;
            rightPanelMode = 'queue';
            patchPrefs({ 'layout.rightCollapsed': 'false' });
            focusRightSignal = { card, seq: Date.now() };
          }
          return;
        }
        if (card === 'album') {
          if (!rightCollapsed && rightPanelMode === 'now-playing') {
            rightCollapsed = true;
            patchPrefs({ 'layout.rightCollapsed': 'true' });
          } else {
            rightCollapsed = false;
            rightPanelMode = 'now-playing';
            patchPrefs({ 'layout.rightCollapsed': 'false' });
            focusRightSignal = { card, seq: Date.now() };
          }
          return;
        }
        rightCollapsed = false;
        rightPanelMode = 'now-playing';
        patchPrefs({ 'layout.rightCollapsed': 'false' });
        focusRightSignal = { card, seq: Date.now() };
      }}
    />
  </div>

  {#if networkToast}
    <div
      in:fly={{ y: 12, duration: 180 }}
      out:fly={{ y: 12, duration: 150 }}
      class="fixed bottom-28 sm:bottom-32 left-1/2 -translate-x-1/2 z-50 px-4 py-2 rounded-full text-xs font-semibold shadow-lg pointer-events-none flex items-center gap-2"
      style="background-color: var(--md-sys-color-inverse-surface); color: var(--md-sys-color-inverse-on-surface);"
    >
      <Icon name={networkToastIcon} weight={networkToastIcon === 'check-circle' ? 'fill' : 'regular'} size={14} />
      <span>{networkToast}</span>
    </div>
  {/if}

  {#if isMobile}
    <nav
      aria-label="Mobile Navigation"
      class="w-full shrink-0 min-h-[68px] pt-1.5 pb-[max(0.5rem,env(safe-area-inset-bottom,0px))] px-2 grid grid-cols-4 gap-1 items-center"
      style="background-color: var(--md-sys-color-surface-container-lowest);"
    >
      <button
        type="button"
        onclick={() => {
          mobileMenuOpen = false;
          mobileShowQueue = false;
          void navigate({ name: 'home' });
        }}
        aria-current={mobileNavTab === 'home' ? 'page' : undefined}
        class="min-h-[52px] min-w-[56px] w-full flex flex-col items-center justify-center gap-1 py-1.5 rounded-xl cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {mobileNavTab === 'home'
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon
          name="house"
          weight={mobileNavTab === 'home' ? 'fill' : 'regular'}
          size={23}
        />
        <span class="text-[11px] font-semibold truncate max-w-full">{tr(appLang, 'nav.home')}</span>
      </button>

      <button
        type="button"
        onclick={() => {
          mobileMenuOpen = false;
          mobileShowQueue = false;
          void navigate({ name: 'search', query: searchQuery }).then(() =>
            requestAnimationFrame(() => (document.getElementById('mobile-search-input') as HTMLInputElement | null)?.focus())
          );
        }}
        aria-current={mobileNavTab === 'search' ? 'page' : undefined}
        class="min-h-[52px] min-w-[56px] w-full flex flex-col items-center justify-center gap-1 py-1.5 rounded-xl cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {mobileNavTab === 'search'
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon
          name="magnifying-glass"
          weight={mobileNavTab === 'search' ? 'bold' : 'regular'}
          size={23}
        />
        <span class="text-[11px] font-semibold truncate max-w-full">{tr(appLang, 'nav.search')}</span>
      </button>

      <button
        type="button"
        onclick={() => {
          mobileMenuOpen = false;
          mobileShowQueue = false;
          void navigate({ name: 'library' });
        }}
        aria-current={mobileNavTab === 'library' ? 'page' : undefined}
        class="min-h-[52px] min-w-[56px] w-full flex flex-col items-center justify-center gap-1 py-1.5 rounded-xl cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {mobileNavTab === 'library'
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon name="books" weight={mobileNavTab === 'library' ? 'fill' : 'regular'} size={23} />
        <span class="text-[11px] font-semibold truncate max-w-full">{tr(appLang, 'nav.library')}</span>
      </button>

      <button
        type="button"
        onclick={() => {
          mobileMenuOpen = false;
          mobileShowQueue = false;
          void navigate({ name: 'playlists' });
        }}
        aria-current={mobileNavTab === 'playlists' ? 'page' : undefined}
        class="min-h-[52px] min-w-[56px] w-full flex flex-col items-center justify-center gap-1 py-1.5 rounded-xl cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {mobileNavTab === 'playlists'
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon
          name="playlist"
          weight={mobileNavTab === 'playlists' ? 'fill' : 'regular'}
          size={23}
        />
        <span class="text-[11px] font-semibold truncate max-w-full">{tr(appLang, 'nav.playlists')}</span>
      </button>
    </nav>

    <MobileQueueSheet open={mobileShowQueue} onClose={() => (mobileShowQueue = false)}>
      <RightPanel
        width={360}
        collapsed={false}
        embeddedMobile={true}
        mode={mobileSheetMode}
        {currentTrack}
        {queue}
        queueIndex={resolveActiveQueueIndex()}
        {queueSourceTitle}
        {audioOutputs}
        {switchingOutputId}
        {isPlaying}
        {positionSec}
        lyricOffsetSec={currentLyricOffsetSec}
        lyrics={currentLyrics}
        {lyricsLoading}
        {preferences}
        {mediaBaseUrl}
        {accentColor}
        onChangeMode={(m) => (rightPanelMode = m)}
        onSelectOutputDevice={(id) => void handleSelectOutputDevice(id)}
        onRefreshAudioOutputs={() => void refreshAudioOutputs()}
        onToggleCollapse={() => (mobileShowQueue = false)}
        onResizeWidth={() => {}}
        onResizeEnd={() => {}}
        onPlayTrack={playTrack}
        onRemoveFromQueue={(id) => {
          queue = queue.filter((t) => t.id !== id);
        }}
        onClearQueue={() => {
          queue = [];
        }}
        onToggleFavorite={handleToggleFavoriteTrack}
        onSeek={handleSeek}
        onOpenTrackDetail={(tr) => {
          mobileShowQueue = false;
          void openTrackAlbumDetail(tr);
        }}
        onOpenArtistDetail={(tr) => {
          mobileShowQueue = false;
          void openTrackArtistDetail(tr);
        }}
        onOpenLyrics={() => {
          mobileShowQueue = false;
          void navigate({ name: 'lyrics' });
        }}
        onResyncLyrics={() => currentTrack && void loadLyricsForTrack(currentTrack, durationSec > 0 ? Math.round(durationSec) : undefined, true)}
        onOpenQueueSource={queueSourceKey || inferQueueSourceKeyByTitle(queueSourceTitle)
          ? () => {
              mobileShowQueue = false;
              void openQueueSource();
            }
          : undefined}
      />
    </MobileQueueSheet>

    <MobileNowPlaying
      open={mobileNowPlayingOpen}
      lang={appLang}
      {currentTrack}
      lyrics={currentLyrics}
      {lyricsLoading}
      lyricOffsetSec={currentLyricOffsetSec}
      {queue}
      queueIndex={resolveActiveQueueIndex()}
      allTracks={tracks}
      ytRecommendedTracks={ytTop100Tracks}
      {playlists}
      {preferences}
      {isPlaying}
      {positionSec}
      {durationSec}
      {shuffle}
      {repeatMode}
      {mediaBaseUrl}
      onClose={() => (mobileNowPlayingOpen = false)}
      onTogglePlay={togglePlay}
      onPrev={playPrev}
      onNext={playNext}
      onSeek={handleSeek}
      onToggleShuffle={toggleShuffleMode}
      onCycleRepeat={() => {
        repeatMode = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
        if (isJamRemote) void sendJamCommand('repeat');
        else api.playerControl('repeat', 0, repeatMode);
      }}
      onToggleFavorite={handleToggleFavoriteTrack}
      onPlayTrack={playTrack}
      onPlayYTMusic={playYTMusicSingle}
      onPlayNext={handleQueuePlayNext}
      onAddToQueue={handleAddToQueue}
      onAddToPlaylist={handleAddTrackToPlaylist}
      onCreatePlaylist={(tr) => {
        pendingCreatePlaylistTrack = tr ?? null;
        showCreatePlaylistModal = true;
      }}
      onDownloadTrack={handleDownloadTrack}
      onRemoveDownload={handleRemoveDownload}
      onOpenArtist={(artistId) => void navigate({ name: 'artist-detail', artistId })}
      onOpenArtistDetail={(tr) => void openTrackArtistDetail(tr)}
      onOpenTrackDetail={(tr) => void openTrackAlbumDetail(tr)}
      onOpenTrackAlbum={(tr) => void openTrackAlbumDetail(tr)}
      onOpenLyrics={() => {
        mobileShowQueue = false;
        void navigate({ name: 'lyrics' });
      }}
      onResyncLyrics={() => currentTrack && void loadLyricsForTrack(currentTrack, durationSec > 0 ? Math.round(durationSec) : undefined, true)}
      onOpenQueue={() => {
        mobileSheetMode = 'queue';
        mobileShowQueue = true;
      }}
      onOpenDevices={() => {
        mobileSheetMode = 'devices';
        mobileShowQueue = true;
        void refreshAudioOutputs();
      }}
    />
  {/if}
</div>

{#if showStreakModal}
  {#await import('./components/StreakModal.svelte') then { default: StreakModal }}
    <StreakModal
      open={showStreakModal}
      {fireStreak}
      lang={appLang}
      onClose={() => (showStreakModal = false)}
      onOpenFullPage={() => navigate({ name: 'listening-stats' })}
    />
  {/await}
{/if}

{#if showCreatePlaylistModal}
  {#await import('./components/CreatePlaylistModal.svelte') then { default: CreatePlaylistModal }}
    <CreatePlaylistModal
      open={showCreatePlaylistModal}
      initialIsLocal={createPlaylistIsLocal}
      lang={appLang}
      onClose={() => (showCreatePlaylistModal = false)}
      onSubmit={handleCreatePlaylist}
    />
  {/await}
{/if}

{#if tooltipVisible && tooltipText}
  <div
    role="tooltip"
    class="pointer-events-none fixed z-[9999] px-2.5 py-1.5 rounded-lg text-[12px] font-semibold leading-none whitespace-nowrap shadow-xl border transition-opacity duration-150"
    style="
      left: {tooltipX}px;
      top: {tooltipY}px;
      transform: translate({tooltipX > window.innerWidth - 90 ? '-85%' : tooltipX < 90 ? '-15%' : '-50%'}, {tooltipBelow ? '0%' : '-100%'});
      background-color: var(--md-sys-color-surface-container-highest);
      color: var(--md-sys-color-on-surface);
      border-color: rgba(128, 128, 128, 0.22);
    "
  >
    {tooltipText}
  </div>
{/if}

<style>
  .velora-route-stage {
    position: relative;
    width: 100%;
  }

  .velora-route-pane {
    position: relative;
    z-index: 1;
    width: 100%;
    min-width: 0;
    will-change: transform, opacity;
  }
</style>
