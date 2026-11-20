<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { cubicIn, cubicOut } from 'svelte/easing';
  import { api, isJamRemote, resolveCoverUrl, safeEventsOn, sendJamCommand } from './lib/api';
  import { VeloraAudioEngine } from './lib/audioEngine';
  import { getPrefBool, getPrefValue } from './lib/preferences';
  import { getPrefLang, tr } from './lib/i18n';
  import type {
    Album,
    Artist,
    DownloadProgress,
    FireStreakStatus,
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
  import StreakModal from './components/StreakModal.svelte';
  import CreatePlaylistModal from './components/CreatePlaylistModal.svelte';
  import SplashScreen from './components/SplashScreen.svelte';
  import LyricsView from './components/LyricsView.svelte';
  import TrackDetailView from './components/TrackDetailView.svelte';
  import MobileNowPlaying from './components/MobileNowPlaying.svelte';

  import HomeView from './views/HomeView.svelte';
  import LibraryViews from './views/LibraryViews.svelte';
  import PlaylistViews from './views/PlaylistViews.svelte';
  import SourcesView from './views/SourcesView.svelte';
  import AnalyticsAndSettingsViews from './views/AnalyticsAndSettingsViews.svelte';

  let tracks = $state<Track[]>([]);
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
  let showSplash = $state(true);
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

  // Navigation stack
  let navStack = $state<ViewRoute[]>([{ name: 'home' }]);
  let navIndex = $state(0);
  const currentRoute = $derived(navStack[navIndex] || { name: 'home' });

  // Active details & YTM search
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
  let ytLoadingMore = $state(false);
  let ytMorePage = $state(0);
  let ytMoreAllBuffer: YTMusicCard[] = [];
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
    if (!getPrefBool(preferences, 'playback.resumeQueue') || !currentTrack || queue.length === 0) return;
    const now = Date.now();
    if (!force && now - lastQueueSaveAt < 2500) return;
    lastQueueSaveAt = now;
    try {
      const idx = Math.max(0, queue.findIndex((t) => trackQueueKey(t) === trackQueueKey(currentTrack!)));
      localStorage.setItem('velora.lastQueue', JSON.stringify({
        queue: queue.slice(0, 50),
        index: idx < 50 ? idx : 0,
        positionSec: Math.floor(positionSec || 0),
        queueSourceTitle,
      }));
    } catch {
      // Ignore storage quota errors
    }
  }
  let sourcesStatus = $state('');

  // Dynamic Left & Right Panel Width + Collapse States
  let leftWidth = $state(212);
  let rightWidth = $state(328);
  let leftCollapsed = $state(false);
  let rightCollapsed = $state(false);
  let rightPanelMode = $state<'now-playing' | 'queue'>('now-playing');
  let focusRightSignal = $state<{ card: 'album' | 'lyrics' | 'queue'; seq: number } | null>(null);
  let activePrefTab = $state<string>('appearance');
  let prefSearch = $state('');
  let prefSavedToast = $state('');
  let seekDeltaSec = $state(0);
  let seekDeltaSeq = $state(0);
  let seekDeltaTimer: ReturnType<typeof setTimeout> | null = null;

  // Modals & Recommendations
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

  // Audio playback & Lyrics state
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
    audioEngine.setPreset(getPrefValue(preferences, 'playback.eqPreset'));
    audioEngine.setNormalize(getPrefBool(preferences, 'playback.audioNormalize'));
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
        // Ignore unsupported actions
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
      // Ignore unsupported actions
    }
  });
  let queue = $state<Track[]>([]);
  let queueSourceTitle = $state('');
  let playedAlgoKeys = new Set<string>();
  let algoBatchRound = 0;
  let currentTrack = $state<Track | null>(null);
  let currentLyrics = $state<TrackLyrics | null>(null);
  let lyricsLoading = $state(false);
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
        const eqOrder = ['spotify', 'bass', 'vocal', 'electronic', 'flat'];
        const curEq = getPrefValue(preferences, 'playback.eqPreset') || 'spotify';
        const nextEq = eqOrder[(eqOrder.indexOf(curEq) + 1) % eqOrder.length];
        void patchPrefs({ 'playback.eqPreset': nextEq });
      }
    });
    const offIslandPlayQueue = safeEventsOn('island:play-queue-index', (idx: number) => {
      if (Number.isInteger(idx) && idx >= 0 && idx < queue.length) {
        void playTrack(queue[idx], queue, queueSourceTitle);
      }
    });
    const offIslandFavId = safeEventsOn('island:toggle-favorite-id', (tid: number) => {
      if (Number.isInteger(tid) && tid > 0) {
        void handleToggleFavoriteTrack(tid);
      }
    });
    const onMediaDeviceChange = () => {
      api.notifyDynamicIslandAudioDevice('');
    };
    navigator.mediaDevices?.addEventListener?.('devicechange', onMediaDeviceChange);
    const offDownload = safeEventsOn('download:progress', (p: DownloadProgress) => {
      if (!p || !p.trackId) return;
      if (p.status === 'completed') {
        const next = { ...downloadStates };
        delete next[p.trackId];
        downloadStates = next;
        void api.getTracks('all', '').then((list) => {
          tracks = list;
          const updated = list.find((t) => t.id === p.trackId);
          if (updated) {
            detailTracks = detailTracks.map((t) => (t.id === p.trackId ? updated : t));
            if (selectedPlaylistDetail) {
              selectedPlaylistDetail = {
                ...selectedPlaylistDetail,
                tracks: selectedPlaylistDetail.tracks.map((t) => (t.id === p.trackId ? updated : t))
              };
            }
          }
        });
      } else if (p.status === 'cancelled') {
        const next = { ...downloadStates };
        delete next[p.trackId];
        downloadStates = next;
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
          // ignore malformed SSE
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
    const cover = tr ? resolveCoverUrl(tr.coverUrl, mediaBaseUrl) : '';
    const eqPreset = getPrefValue(preferences, 'playback.eqPreset') || 'spotify';
    const qIdx = tr ? queue.findIndex((q) => (tr.id > 0 && q.id === tr.id) || (tr.externalId && q.externalId === tr.externalId)) : -1;
    api.syncDynamicIsland({
      title: tr?.title ?? '',
      artist: tr?.artistName ?? '',
      album: tr?.albumTitle ?? '',
      coverUrl: cover,
      isPlaying,
      isFavorite: Boolean(tr?.isFavorite),
      positionSec,
      durationSec: durationSec || tr?.durationSec || 0,
      volume,
      muted,
      shuffle,
      repeatMode,
      eqPreset,
      themeSeed: 'monolith',
      accentColor,
      streakDays: fireStreak?.currentStreak ?? 0,
      lyricsLoading,
      queueTitle: queueSourceTitle || 'Queue',
      queueIndex: qIdx,
      queue: queue.slice(0, 250).map((q) => ({
        id: q.id,
        externalId: q.externalId ?? '',
        title: q.title,
        artist: q.artistName,
        coverUrl: resolveCoverUrl(q.coverUrl, mediaBaseUrl),
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

  async function loadYTMusicHome() {
    ytHomeLoading = true;
    try {
      const feed = await api.getYTMusicHome();
      ytShelves = (feed?.shelves || []).map((shelf) =>
        shelf.title === 'Lagu Lainnya untuk Anda'
          ? {
              ...shelf,
              items: shuffleAndSpreadByArtist(shelf.items || [], (c) => c.artist || c.subtitle || '')
            }
          : shelf
      );
      ytResults = shuffleAndSpreadByArtist(feed?.tracks || [], (t) => t.artist || '');
    } catch {
      ytShelves = [];
    } finally {
      ytHomeLoading = false;
    }
  }

  async function handleDownloadTrack(tr: Track) {
    if (!tr) return;
    if (tr.id > 0) {
      await api.downloadTrack(tr.id);
      return;
    }
    if (tr.externalId) {
      await api.downloadYTMusicTrack({
        videoId: tr.externalId,
        title: tr.title,
        artist: tr.artistName,
        album: tr.albumTitle,
        durationSec: tr.durationSec,
        coverUrl: tr.coverUrl,
        format: tr.format || 'OPUS'
      });
    }
  }

  async function handleDownloadYTMusicTrack(yt: YTMusicTrack) {
    if (!yt?.videoId) return;
    await api.downloadYTMusicTrack(yt);
  }

  async function handleRemoveDownload(trackId: number) {
    const updated = await api.removeDownloadedTrack(trackId);
    tracks = tracks.map((t) => (t.id === trackId ? updated : t));
    detailTracks = detailTracks.map((t) => (t.id === trackId ? updated : t));
    if (selectedPlaylistDetail) {
      selectedPlaylistDetail = {
        ...selectedPlaylistDetail,
        tracks: selectedPlaylistDetail.tracks.map((t) => (t.id === trackId ? updated : t))
      };
    }
  }

  async function handleCancelDownload(trackId: number) {
    await api.cancelDownload(trackId);
    const next = { ...downloadStates };
    delete next[trackId];
    downloadStates = next;
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
    preferences = boot.preferences;
    fireStreak = boot.fireStreak;
    listeningStats = boot.listeningStats;
    mediaBaseUrl = boot.mediaBaseUrl;

    if (queue.length === 0 && tracks.length > 0) {
      if (!currentTrack) {
        currentTrack = tracks[0];
        durationSec = tracks[0].durationSec;
        loadLyricsForTrack(tracks[0]);
      }
      queue = buildLocalAlgorithmQueue(currentTrack ?? tracks[0]);
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

    if (getPrefBool(preferences, 'playback.resumeQueue') && !currentTrack) {
      try {
        const raw = localStorage.getItem('velora.lastQueue');
        if (raw) {
          const saved = JSON.parse(raw) as { queue?: Track[]; index?: number; positionSec?: number; queueSourceTitle?: string };
          if (Array.isArray(saved.queue) && saved.queue.length > 0) {
            const idx = typeof saved.index === 'number' && saved.index >= 0 && saved.index < saved.queue.length ? saved.index : 0;
            queue = saved.queue;
            queueSourceTitle = saved.queueSourceTitle || '';
            currentTrack = saved.queue[idx];
            positionSec = typeof saved.positionSec === 'number' && saved.positionSec > 0 ? saved.positionSec : 0;
            durationSec = currentTrack.durationSec || 0;
            pendingResumeSeek = positionSec;
            isPlaying = false;
            loadLyricsForTrack(currentTrack);
          }
        }
      } catch {
        // Ignore corrupt saved queue
      }
    }

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
    if (patch['content.country'] !== undefined) {
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
    if (patch['playback.audioNormalize'] !== undefined) {
      audioEngine.setNormalize(patch['playback.audioNormalize'] !== 'false');
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
          // Ignore reload abort
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
    if (patch['general.zoomLevel'] !== undefined) {
      const z = Number(patch['general.zoomLevel']);
      if (Number.isFinite(z)) applyZoom(z, true);
    }
    if (patch['integrations.jam.enabled'] !== undefined) {
      if (patch['integrations.jam.enabled'] === 'true') {
        jamStatus = await api.startJam();
        preferences = {
          ...preferences,
          'integrations.jam.remoteUrl': jamStatus.remoteUrl
        };
      } else {
        jamStatus = await api.stopJam();
        preferences = {
          ...preferences,
          'integrations.jam.remoteUrl': ''
        };
      }
    }
  }

  async function toggleJamServer() {
    jamLoading = true;
    try {
      if (jamStatus.running) {
        jamStatus = await api.stopJam();
        preferences = { ...preferences, 'integrations.jam.enabled': 'false' };
      } else {
        jamStatus = await api.startJam();
        preferences = {
          ...preferences,
          'integrations.jam.enabled': 'true',
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
        if (idx >= 0 && idx < queue.length) {
          await playTrack(queue[idx], queue);
        }
        break;
      }
      case 'remove-index': {
        const idx = Math.round(Number(cmd.value) || 0);
        if (idx >= 0 && idx < queue.length) {
          queue = queue.filter((_, i) => i !== idx);
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

  async function loadLyricsForTrack(tr: Track) {
    lyricsLoading = true;
    try {
      const res = await api.getTrackLyrics(
        tr.id,
        tr.title,
        tr.artistName,
        tr.albumTitle,
        tr.durationSec
      );
      if (currentTrack?.id === tr.id) {
        currentLyrics = res;
      }
    } catch {
      if (currentTrack?.id === tr.id) {
        currentLyrics = null;
      }
    } finally {
      if (currentTrack?.id === tr.id) {
        lyricsLoading = false;
      }
    }
  }

  async function navigate(route: ViewRoute) {
    const nextStack = navStack.slice(0, navIndex + 1);
    nextStack.push(route);
    navStack = nextStack;
    navIndex = nextStack.length - 1;
    await loadRouteData(route);
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
    await navigate({
      name: 'yt-album-detail',
      videoId: card.videoId || '',
      browseId: card.browseId || '',
      title: card.title,
      artist: card.artist,
      albumTitle: card.title,
      coverUrl: card.coverUrl
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
      await handleSearchYTMusic(q);
    }
  }

  async function handleSearchYTMusic(q: string) {
    const seq = ++ytSearchSeq;
    ytLoading = true;
    if (!q.trim()) {
      ytResults = [];
    }
    try {
      const primary = await api.searchYouTubeMusic(q);
      if (seq !== ytSearchSeq) return;
      if (!q.trim() || (primary && primary.length >= 20)) {
        const list = primary || [];
        ytResults = !q.trim() ? shuffleAndSpreadByArtist(list, (t) => t.artist || '') : list;
        return;
      }
      const seen = new Set((primary || []).map((t) => t.videoId));
      const merged = [...(primary || [])];
      const extra = await api.searchYouTubeMusic(`${q} populer`).catch(() => []);
      if (seq !== ytSearchSeq) return;
      for (const item of extra || []) {
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
    ytLoading = true;
    try {
      const primary = await api.searchYouTubeMusicPlaylists(q);
      const seen = new Set<string>();
      const merged: YTMusicCard[] = [];
      const pushUnique = (card: YTMusicCard) => {
        const key = card.browseId || card.playlistId || card.id;
        if (!key || seen.has(key)) return;
        seen.add(key);
        merged.push(card);
      };
      for (const c of primary || []) pushUnique(c);
      if (merged.length < 20) {
        const extra = await api.searchYouTubeMusicPlaylists(`${q} mix`).catch(() => []);
        for (const c of extra || []) pushUnique(c);
      }
      if (merged.length < 20) {
        for (const shelf of ytShelves) {
          for (const card of shelf.items) {
            if (card.kind === 'playlist') pushUnique(card);
          }
        }
      }
      ytPlaylistResults = merged;
    } finally {
      ytLoading = false;
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
      'Best Acoustic Pop Hits'
    ],
    artis: [
      'Popular Indonesian Singers',
      'Top Global Pop Artists',
      'Best Indie Artists Indonesia',
      'Top R&B & Soul Artists'
    ],
    album: [
      'Top Pop Indonesia Album',
      'Best Indie Album Indonesia',
      'Top Global Pop Album',
      'R&B Soul Classic Album'
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

  async function handleLoadMorePopularArtists() {
    if (ytLoadingMorePopularArtists) return;
    ytLoadingMorePopularArtists = true;
    try {
      const idx = ytShelves.findIndex((s) => s.title === 'Artis Populer');
      if (idx < 0) return;
      const existingList = ytShelves[idx].items;
      const existingKeys = new Set(existingList.map((c) => c.browseId || c.title.toLowerCase()));
      const moreArtists = await api.getMoreArtists(existingList.length, Array.from(existingKeys));
      const added = (moreArtists || []).filter((c) => {
        const k = c.browseId || c.title.toLowerCase();
        if (!k || existingKeys.has(k)) return false;
        existingKeys.add(k);
        return true;
      });
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
        const existingKeys = new Set(ytAlbumResults.map((c) => c.browseId || c.id));
        const existingCovers = new Set(ytAlbumResults.map((c) => (c.coverUrl || '').trim()).filter(Boolean));
        const added: YTMusicCard[] = [];
        for (let attempt = 0; attempt < 3 && added.length < 14; attempt++) {
          const page = ytMorePage++;
          const q = CATEGORY_MORE_QUERIES.album[page % CATEGORY_MORE_QUERIES.album.length];
          const moreAlbums = await api.searchYouTubeMusicAlbums(q).catch(() => []);
          for (const c of moreAlbums || []) {
            const k = c.browseId || c.id;
            const cov = (c.coverUrl || '').trim();
            if (!k || existingKeys.has(k)) continue;
            if (cov && existingCovers.has(cov)) continue;
            existingKeys.add(k);
            if (cov) existingCovers.add(cov);
            added.push(c);
          }
        }
        if (added.length > 0) {
          ytAlbumResults = [...ytAlbumResults, ...added];
        }
        return;
      }

      if (activeCategory === 'artis') {
        const existingKeys = new Set(ytArtistResults.map((c) => c.browseId || c.title.toLowerCase()));
        const moreArtists = await api.getMoreArtists(ytArtistResults.length, Array.from(existingKeys));
        const added = (moreArtists || []).filter((c) => {
          const k = c.browseId || c.title.toLowerCase();
          if (!k || existingKeys.has(k)) return false;
          existingKeys.add(k);
          return true;
        });
        if (added.length > 0) {
          ytArtistResults = [...ytArtistResults, ...added];
        }
        return;
      }

      if (activeCategory === 'playlist') {
        const existingKeys = new Set(ytPlaylistResults.map((c) => c.browseId || c.playlistId || c.id));
        const added: YTMusicCard[] = [];
        for (let attempt = 0; attempt < 3 && added.length < 14; attempt++) {
          const page = ytMorePage++;
          const q = CATEGORY_MORE_QUERIES.playlist[page % CATEGORY_MORE_QUERIES.playlist.length];
          const morePlaylists = await api.searchYouTubeMusicPlaylists(q).catch(() => []);
          for (const c of morePlaylists || []) {
            const k = c.browseId || c.playlistId || c.id;
            if (k && !existingKeys.has(k)) {
              existingKeys.add(k);
              added.push(c);
            }
          }
        }
        if (added.length > 0) {
          ytPlaylistResults = [...ytPlaylistResults, ...added];
        }
        return;
      }

      if (activeCategory === 'semua') {
        const targetBatch = 35; // 5 rows x 7 items
        const targetTitle = 'Lagu Lainnya untuk Anda';
        const existingShelf = ytShelves.find((s) => s.title === targetTitle);
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

        const pushUniqueTopSongs = (tracksList: YTMusicTrack[] | null | undefined, sourceLabel: string) => {
          for (const t of tracksList || []) {
            if (!t.videoId || existingVideoIds.has(t.videoId)) continue;
            const cov = (t.coverUrl || '').trim();
            const alb = (t.album || '').trim().toLowerCase();
            const ar = (t.artist || '').trim().toLowerCase();
            if (cov && existingCovers.has(cov)) continue;
            if (alb && alb !== 'single' && existingAlbums.has(alb)) continue;
            if (ar && (batchArtistCount.get(ar) || 0) >= 2) continue;

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
        pushUniqueTopSongs(recTracks, 'Velora Music');
        pushUniqueTopSongs(topSongs, 'Top Songs');

        let attempts = 0;
        while (pool.length < targetBatch && attempts < 3) {
          attempts++;
          const page = ytMorePage++;
          const q = MORESONGS_QUERIES[page % MORESONGS_QUERIES.length];
          const moreTracks = await api.searchYouTubeMusic(q).catch(() => []);
          pushUniqueTopSongs(moreTracks, 'Top Songs');
        }

        if (pool.length === 0) return;

        const spreadPool = shuffleAndSpreadByArtist(pool, (c) => c.artist || c.subtitle || '', tailArtists);
        const takeCount =
          spreadPool.length >= targetBatch
            ? targetBatch
            : spreadPool.length >= 7
              ? Math.floor(spreadPool.length / 7) * 7
              : spreadPool.length;
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
        const existingIds = new Set(ytResults.map((t) => t.videoId));
        const existingCovers = new Set(ytResults.map((t) => (t.coverUrl || '').trim()).filter(Boolean));
        const existingAlbums = new Set(
          ytResults.map((t) => (t.album || '').trim().toLowerCase()).filter((a) => a && a !== 'single')
        );
        const tailArtists = ytResults.slice(-3).map((t) => t.artist || '');
        const batchArtistCount = new Map<string, number>();
        const added: YTMusicTrack[] = [];
        const targetMore = 21;

        const collectTracks = (list: YTMusicTrack[] | null | undefined) => {
          for (const t of list || []) {
            if (!t.videoId || existingIds.has(t.videoId)) continue;
            const cov = (t.coverUrl || '').trim();
            const alb = (t.album || '').trim().toLowerCase();
            const ar = (t.artist || '').trim().toLowerCase();
            if (cov && existingCovers.has(cov)) continue;
            if (alb && alb !== 'single' && existingAlbums.has(alb)) continue;
            if (ar && (batchArtistCount.get(ar) || 0) >= 2) continue;

            existingIds.add(t.videoId);
            if (cov) existingCovers.add(cov);
            if (alb && alb !== 'single') existingAlbums.add(alb);
            if (ar) batchArtistCount.set(ar, (batchArtistCount.get(ar) || 0) + 1);
            added.push(t);
          }
        };

        if (activeCategory === 'musik') {
          const [rec, topSongs] = await Promise.all([
            api.getYTMusicTop100Recommended(Array.from(existingIds)).catch(() => []),
            api.getMoreTopSongs(ytResults.length + (ytMorePage++) * 20, Array.from(existingIds)).catch(() => [])
          ]);
          collectTracks(rec);
          collectTracks(topSongs);
        }

        const queries = CATEGORY_MORE_QUERIES[activeCategory] || CATEGORY_MORE_QUERIES.musik;
        for (let attempt = 0; attempt < 3 && added.length < targetMore; attempt++) {
          const page = ytMorePage++;
          const q = queries[page % queries.length];
          const moreTracks = await api.searchYouTubeMusic(q).catch(() => []);
          collectTracks(moreTracks);
        }

        if (added.length > 0) {
          const spreadAdded = shuffleAndSpreadByArtist(added, (t) => t.artist || '', tailArtists);
          ytResults = [...ytResults, ...spreadAdded];
        }
      }
    } finally {
      ytLoadingMore = false;
    }
  }

  function resolveDirectStreamUrl(tr: Track): string {
    if (!mediaBaseUrl) return '';
    if (tr.sourceType === 'local' && tr.id > 0) {
      return `${mediaBaseUrl}/media/local?id=${tr.id}`;
    }
    if (tr.externalId && !tr.externalId.startsWith('dz:') && !tr.externalId.startsWith('sp:')) {
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

    // Primary: Backend Top Indonesia + Popular Global + History-based Recommendations
    try {
      const recItems = await api.getYTMusicTop100Recommended(excludeVideoIds);
      for (const item of recItems || []) {
        if (!item.videoId) continue;
        const key = `yt:${item.videoId}`;
        if (seen.has(key)) continue;
        seen.add(key);
        candidates.push(ytTrackToQueueTrack(item, idxCounter++));
      }
    } catch {
      // fallback to query search below
    }

    if (candidates.length < 15) {
      const queries = getGenreQueriesForBatch(baseTrack, round);
      const responses = await Promise.all(
        queries.map((q) => api.searchYouTubeMusic(q).catch(() => [] as YTMusicTrack[]))
      );
      for (const list of responses) {
        for (const item of list || []) {
          if (!item.videoId) continue;
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
      // keep seed algorithm queue
    }
  }

  async function playYTMusicSingle(yt: YTMusicTrack) {
    queueSourceTitle = '';
    algoBatchRound = 0;
    const tr = await api.playYTMusicTrack(yt);
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
        await playTrack(colTracks[0], colTracks, card.title);
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
      await playTrack(detail.tracks[0], detail.tracks, detail.playlist?.name || 'Playlist');
    }
  }

  async function handleOpenAlbum(albumId: number) {
    const al = albums.find((a) => a.id === albumId);
    if (al?.sourceType === 'ytmusic') {
      await navigate({
        name: 'yt-album-detail',
        browseId: al.externalId || '',
        videoId: '',
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
      await playTrack(albumTracks[0], albumTracks, albumName);
    }
  }

  async function handlePlayArtist(artistId: number) {
    const artistTracks = await api.getArtistTracks(artistId);
    if (artistTracks && artistTracks.length > 0) {
      const artistName = artists.find((a) => a.id === artistId)?.name || 'Artis';
      await playTrack(artistTracks[0], artistTracks, artistName);
    }
  }

  async function playTrack(tr: Track, list: Track[] = [], sourceTitle?: string) {
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
    if (sourceTitle !== undefined) {
      queueSourceTitle = sourceTitle;
    } else if (list.length > 0) {
      if (currentRoute.name === 'playlist-detail') {
        queueSourceTitle = selectedPlaylistDetail?.playlist?.name || 'Playlist';
      } else if (currentRoute.name === 'album-detail') {
        queueSourceTitle = albums.find((a) => a.id === currentRoute.albumId)?.title || 'Album';
      } else if (currentRoute.name === 'artist-detail') {
        queueSourceTitle = artists.find((a) => a.id === currentRoute.artistId)?.name || 'Artis';
      } else if (currentRoute.name === 'liked-songs') {
        queueSourceTitle = 'Favorite music';
      }
    } else {
      queueSourceTitle = '';
    }

    if (list.length > 0) {
      if (list === queue) {
        queue = [...list];
      } else {
        const curKey = trackQueueKey(tr);
        const rest = list.filter((item) => trackQueueKey(item) !== curKey);
        queue = [tr, ...shuffleAndSpreadByArtist(rest, (item) => item.artistName || item.albumTitle || '')];
      }
    } else {
      if (getPrefBool(preferences, 'playback.radioOnPick')) {
        queue = buildLocalAlgorithmQueue(tr);
        void populateYTMusicAlgorithmQueue(tr);
      } else {
        queue = [tr];
      }
    }

    const prevTrack = currentTrack;
    const prevAcc = listenedAccumulator;
    const prevRecorded = recordedForCurrentTrack;
    if (prevTrack && prevAcc >= 3 && !prevRecorded) {
      void flushListenProgress(prevTrack.id, Math.round(prevAcc), false);
    }

    const seq = ++playRequestSeq;
    const resumeSeek = pendingResumeSeek > 0 && currentTrack && trackQueueKey(currentTrack) === trackQueueKey(tr) ? pendingResumeSeek : 0;
    pendingResumeSeek = 0;
    silentTailTicks = 0;
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
      streamUrl = await api.getStreamUrl(tr.id);
    }

    if (seq !== playRequestSeq) return;
    if (audioEl && streamUrl) {
      audioEngine.attach(audioEl);
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
        }
      } catch (err: any) {
        if (err?.name !== 'AbortError' && seq === playRequestSeq) {
          isPlaying = false;
        }
      } finally {
        if (seq === playRequestSeq) {
          switchingTrack = false;
        }
      }
    }
  }

  async function flushListenProgress(trackId: number, sec: number, completed: boolean) {
    if (sec <= 0) return;
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
    }
    api.updateDynamicIslandProgress(positionSec, durationSec);
    if (isPlaying) {
      listenedAccumulator += 0.25;
      if (listenedAccumulator >= 10 && !recordedForCurrentTrack) {
        recordedForCurrentTrack = true;
        flushListenProgress(currentTrack.id, 10, false);
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
      const rem = Math.max(5, Math.round(listenedAccumulator));
      if (!recordedForCurrentTrack) {
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

  async function playNext() {
    if (isJamRemote) {
      await sendJamCommand('next');
      return;
    }
    if (queue.length === 0) return;
    if (shuffle && queue.length > 1 && queueSourceTitle !== '') {
      const rand = Math.floor(Math.random() * queue.length);
      await playTrack(queue[rand], queue, queueSourceTitle);
      return;
    }
    const idx = currentTrack ? queue.findIndex((t) => t.id === currentTrack!.id) : -1;
    const nextIdx = idx + 1;
    if (nextIdx < queue.length) {
      await playTrack(queue[nextIdx], queue, queueSourceTitle);
    } else if (repeatMode === 'all' && queueSourceTitle !== '') {
      await playTrack(queue[0], queue, queueSourceTitle);
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
        await playTrack(queue[0], queue, queueSourceTitle);
      } else {
        isPlaying = false;
      }
    } else if (repeatMode === 'all') {
      await playTrack(queue[0], queue, queueSourceTitle);
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
    const idx = currentTrack ? queue.findIndex((t) => t.id === currentTrack!.id) : 0;
    const prevIdx = idx - 1 >= 0 ? idx - 1 : queue.length - 1;
    await playTrack(queue[prevIdx], queue);
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
      // ignore
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

  async function handleToggleFavoriteTrack(id: number) {
    const fav = await api.toggleFavoriteTrack(id);
    tracks = tracks.map((t) => (t.id === id ? { ...t, isFavorite: fav } : t));
    queue = queue.map((t) => (t.id === id ? { ...t, isFavorite: fav } : t));
    if (currentTrack && currentTrack.id === id) {
      currentTrack = { ...currentTrack, isFavorite: fav };
    }
    if (selectedPlaylistDetail) {
      selectedPlaylistDetail = {
        ...selectedPlaylistDetail,
        tracks: selectedPlaylistDetail.tracks.map((t) => (t.id === id ? { ...t, isFavorite: fav } : t))
      };
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
    playlists = await api.getPlaylists(false);
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
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<audio
  bind:this={audioEl}
  crossorigin="anonymous"
  preload="auto"
  ontimeupdate={handleAudioTimeUpdate}
  onended={handleAudioEnded}
  onplay={() => (isPlaying = true)}
  onplaying={() => (isPlaying = true)}
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
    <SplashScreen onComplete={() => (showSplash = false)} />
  {/if}

  <!-- Top App Bar (Borderless) -->
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
    jamPrefEnabled={getPrefValue(preferences, 'integrations.jam.enabled') !== 'false'}
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
    onOpenStreakModal={() => (showStreakModal = true)}
    onResetZoom={() => applyZoom(1)}
  />

  <!-- Main 3-Section Resizable & Collapsible Borderless Workspace -->
  <div
    class="flex-1 min-h-0 flex gap-2 pl-2 pb-2 overflow-hidden {rightCollapsed &&
    !lockRightHidden &&
    currentRoute.name !== 'preferences' &&
    currentRoute.name !== 'track-detail' &&
    currentRoute.name !== 'album-detail' &&
    currentRoute.name !== 'yt-album-detail' &&
    currentRoute.name !== 'lyrics'
      ? 'pr-0'
      : 'pr-2'}"
  >
    <!-- SECTION 1 (LEFT): Menu Sidebar (hidden on mobile; mobile uses bottom nav) -->
    {#if !isMobile}
      <Sidebar
      {currentRoute}
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

    <!-- SECTION 2 (CENTER): Songs & Playlists + Spotify-style Home -->
    <main
      bind:this={mainScrollEl}
      class="flex-1 min-w-0 min-h-0 overflow-y-auto overflow-x-hidden scroll-pt-16 rounded-lg transition-colors duration-300 {isMobile && mobileShowQueue ? 'hidden' : ''}"
      style="background-color: var(--md-sys-color-surface);"
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
          {isPlaying}
          onTogglePlay={togglePlay}
          {mediaBaseUrl}
          onPlayYTMusic={playYTMusicSingle}
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
          onNavigate={navigate}
        />
      {:else if currentRoute.name === 'search' || currentRoute.name === 'liked-songs' || currentRoute.name === 'downloaded' || currentRoute.name === 'favorite-albums' || currentRoute.name === 'favorite-artists' || currentRoute.name === 'artist-detail'}
        <LibraryViews
          lang={appLang}
          mode={currentRoute.name}
          {searchQuery}
          {tracks}
          {albums}
          {artists}
          selectedArtist={currentRoute.name === 'artist-detail'
            ? artists.find(
                (a) =>
                  (currentRoute.artistId !== undefined && a.id === currentRoute.artistId) ||
                  (currentRoute.artistName && a.name.toLowerCase() === currentRoute.artistName.toLowerCase())
              ) ?? null
            : null}
          artistFallbackName={currentRoute.name === 'artist-detail' ? currentRoute.artistName ?? '' : ''}
          artistFallbackAvatar={currentRoute.name === 'artist-detail' ? currentRoute.avatarUrl ?? '' : ''}
          artistFallbackBrowseId={currentRoute.name === 'artist-detail' ? currentRoute.browseId ?? '' : ''}
          {ytArtistDetail}
          {ytArtistLoading}
          {detailTracks}
          {ytResults}
          {ytLoading}
          {playlists}
          {downloadStates}
          currentTrackId={currentTrack?.id ?? null}
          currentExternalId={currentTrack?.externalId ?? null}
          currentTrackTitle={currentTrack?.title ?? null}
          currentTrackArtist={currentTrack?.artistName ?? null}
          {isPlaying}
          onTogglePlay={togglePlay}
          {mediaBaseUrl}
          {accentColor}
          onPlayTrack={playTrack}
          onPlayAlbum={handlePlayAlbum}
          onPlayArtist={handlePlayArtist}
          onPlayYTMusic={playYTMusicSingle}
          onSaveYTMusic={async (yt) => {
            await api.saveYTMusicTrack(yt);
            await refreshAll();
          }}
          onSearchYTMusic={handleSearchYTMusic}
          onToggleFavoriteTrack={handleToggleFavoriteTrack}
          onToggleFavoriteAlbum={handleToggleFavoriteAlbum}
          onToggleFavoriteArtist={handleToggleFavoriteArtist}
          onToggleFavoriteYTMusicArtist={handleToggleFavoriteYTMusicArtist}
          onAddToPlaylist={async (pid, tid) => {
            await api.addTrackToPlaylist(pid, tid);
            playlists = await api.getPlaylists(false);
          }}
          onDownloadTrack={handleDownloadTrack}
          onDownloadYTMusic={handleDownloadYTMusicTrack}
          onRemoveDownload={handleRemoveDownload}
          onCancelDownload={handleCancelDownload}
          onOpenAlbum={handleOpenAlbum}
          onOpenArtist={(artistId) => navigate({ name: 'artist-detail', artistId })}
          onOpenTrackAlbum={openTrackAlbumDetail}
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
      {:else if currentRoute.name === 'playlists' || currentRoute.name === 'local-playlists' || currentRoute.name === 'playlist-detail'}
        <PlaylistViews
          lang={appLang}
          mode={currentRoute.name}
          {playlists}
          {playlistsLoading}
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
          {isPlaying}
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
          onPlayTrack={playTrack}
          onPlayYTMusic={playYTMusicSingle}
          onAddTrack={async (pid, tid) => {
            await api.addTrackToPlaylist(pid, tid);
            playlists = await api.getPlaylists(false);
            selectedPlaylistDetail = await api.getPlaylistDetail(pid);
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
          onAddToQueue={(tr) => {
            if (!queue.some((q) => q.id === tr.id)) {
              queue = [...queue, tr];
            }
          }}
          onDownloadTrack={handleDownloadTrack}
          onRemoveDownload={handleRemoveDownload}
          onOpenArtist={(artistId) => navigate({ name: 'artist-detail', artistId })}
          onOpenTrackAlbum={openTrackAlbumDetail}
          onExportM3U={(pid) => api.exportLocalPlaylistM3U(pid)}
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
      {:else if currentRoute.name === 'sources'}
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
            : currentRoute.name === 'track-detail'
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
        <TrackDetailView
          track={detailTrack ?? albumTrackList[0] ?? null}
          album={detailAlbum}
          albumTracks={albumTrackList}
          ytAlbum={currentRoute.name === 'yt-album-detail' ? ytAlbumDetail : null}
          ytAlbumLoading={currentRoute.name === 'yt-album-detail' ? ytAlbumLoading : false}
          currentTrackId={currentTrack?.id ?? null}
          currentExternalId={currentTrack?.externalId ?? null}
          currentTrackTitle={currentTrack?.title ?? null}
          currentTrackArtist={currentTrack?.artistName ?? null}
          {isPlaying}
          {tracks}
          {albums}
          {playlists}
          {mediaBaseUrl}
          {accentColor}
          onPlayTrack={playTrack}
          onTogglePlay={() => void togglePlay()}
          onToggleFavoriteTrack={handleToggleFavoriteTrack}
          onToggleFavoriteAlbum={handleToggleFavoriteAlbum}
          onToggleFavoriteYTMusicAlbum={handleToggleFavoriteYTMusicAlbum}
          onAddToPlaylist={async (plId, trId) => {
            await api.addTrackToPlaylist(plId, trId);
            playlists = await api.getPlaylists(false);
          }}
          onOpenArtist={(artistId) => navigate({ name: 'artist-detail', artistId })}
          onOpenAlbum={handleOpenAlbum}
          onOpenYTAlbum={(card) => void openYTAlbumCard(card)}
          onDownloadTrack={handleDownloadTrack}
          onRemoveDownload={handleRemoveDownload}
          onAddToQueue={(tr) => {
            if (!queue.some((q) => q.id === tr.id)) {
              queue = [...queue, tr];
            }
          }}
        />
      {:else if currentRoute.name === 'lyrics'}
        <LyricsView
          track={currentTrack}
          lyrics={currentLyrics}
          {lyricsLoading}
          {positionSec}
          {mediaBaseUrl}
          {preferences}
          onSeek={handleSeek}
          onResync={() => currentTrack && loadLyricsForTrack(currentTrack)}
          onClose={() => {
            if (navIndex > 0) goBack();
            else navigate({ name: 'home' });
          }}
        />
      {:else}
        <AnalyticsAndSettingsViews
          mode={currentRoute.name}
          {history}
          {listeningStats}
          {fireStreak}
          {preferences}
          {tracks}
          {mediaBaseUrl}
          {accentColor}
          bind:activePrefTab
          bind:prefSearch
          onPlayTrack={playTrack}
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
          onNavigate={navigate}
        />
      {/if}
    </main>

    <!-- SECTION 3 (RIGHT): Hidden when locked in Settings or on Preferences, Track/Album Detail & Lyrics pages -->
    {#if !lockRightHidden && (!isMobile || mobileShowQueue) && currentRoute.name !== 'preferences' && currentRoute.name !== 'track-detail' && currentRoute.name !== 'album-detail' && currentRoute.name !== 'yt-album-detail' && currentRoute.name !== 'lyrics'}
      <RightPanel
        width={isMobile ? window.innerWidth - 16 : rightWidth}
        collapsed={isMobile ? false : rightCollapsed}
        mode={rightPanelMode}
        {currentTrack}
        {queue}
        {queueSourceTitle}
        {isPlaying}
        {positionSec}
        lyrics={currentLyrics}
        {lyricsLoading}
        {preferences}
        {mediaBaseUrl}
        {accentColor}
        focusCardSignal={focusRightSignal}
        onChangeMode={(m) => (rightPanelMode = m)}
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
      />
    {/if}
  </div>

  <!-- Bottom Full-Width Flat Player Bar (No rounded corners) -->
  <div class="w-full shrink-0">
    <PlayerBar
      lang={appLang}
      {currentTrack}
      {isPlaying}
      {positionSec}
      {durationSec}
      {volume}
      {muted}
      {shuffle}
      {repeatMode}
      {rightCollapsed}
      {rightPanelMode}
      isLyricsOpen={currentRoute.name === 'lyrics'}
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
      onToggleShuffle={() => {
        shuffle = !shuffle;
        if (isJamRemote) void sendJamCommand('shuffle');
        else api.playerControl('shuffle');
      }}
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
          if (
            currentRoute.name === 'track-detail' ||
            currentRoute.name === 'album-detail' ||
            currentRoute.name === 'yt-album-detail' ||
            currentRoute.name === 'lyrics' ||
            currentRoute.name === 'preferences'
          ) {
            if (navIndex > 0) goBack();
            else navigate({ name: 'home' });
            rightCollapsed = false;
            rightPanelMode = 'now-playing';
            patchPrefs({ 'layout.rightCollapsed': 'false' });
            focusRightSignal = { card, seq: Date.now() };
            return;
          }
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

  {#if isMobile}
    {#if mobileMenuOpen}
      <div
        role="presentation"
        transition:fade={{ duration: 160 }}
        onclick={() => (mobileMenuOpen = false)}
        class="fixed inset-0 z-50 bg-black/50 flex items-end"
      >
        <div
          role="dialog"
          aria-label="Menu Navigasi"
          in:fly={{ y: 260, duration: 240, easing: cubicOut }}
          out:fly={{ y: 260, duration: 180, easing: cubicIn }}
          onclick={(e) => e.stopPropagation()}
          class="w-full rounded-t-2xl p-4 pb-6 space-y-2 border-t"
          style="background-color: var(--md-sys-color-surface-container-high); border-color: var(--md-sys-color-outline-variant);"
        >
          <div class="flex items-center justify-between px-2 pb-2">
            <span class="text-xs font-bold uppercase tracking-wider" style="color: var(--md-sys-color-on-surface-variant);">
              Menu Velora
            </span>
            <button
              type="button"
              onclick={() => (mobileMenuOpen = false)}
              class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
            >
              <Icon name="x" size={15} />
            </button>
          </div>
          <div class="grid grid-cols-2 gap-2">
            {#each [
              { label: tr(appLang, 'nav.favoriteAlbums'), icon: 'vinyl-record', route: { name: 'favorite-albums' } as ViewRoute },
              { label: tr(appLang, 'nav.favoriteMusic'), icon: 'music-note', route: { name: 'liked-songs' } as ViewRoute },
              { label: tr(appLang, 'nav.favoriteArtists'), icon: 'user', route: { name: 'favorite-artists' } as ViewRoute },
              { label: tr(appLang, 'nav.playlists'), icon: 'playlist', route: { name: 'playlists' } as ViewRoute },
              { label: tr(appLang, 'nav.downloaded'), icon: 'download', route: { name: 'downloaded' } as ViewRoute },
              { label: tr(appLang, 'nav.history'), icon: 'clock-counter-clockwise', route: { name: 'history' } as ViewRoute },
              { label: tr(appLang, 'nav.settings'), icon: 'gear', route: { name: 'preferences' } as ViewRoute }
            ] as entry (entry.icon)}
              <button
                type="button"
                onclick={() => {
                  mobileMenuOpen = false;
                  mobileShowQueue = false;
                  void navigate(entry.route);
                }}
                class="flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-xs font-semibold cursor-pointer text-left transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                style="background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface);"
              >
                <Icon name={entry.icon} size={17} />
                <span class="truncate">{entry.label}</span>
              </button>
            {/each}
          </div>
        </div>
      </div>
    {/if}

    <nav
      aria-label="Mobile Navigation"
      class="w-full shrink-0 min-h-14 pb-[env(safe-area-inset-bottom)] px-2 grid grid-cols-4 items-center border-t"
      style="background-color: var(--md-sys-color-surface-container-lowest); border-color: var(--md-sys-color-outline-variant);"
    >
      <button
        type="button"
        onclick={() => {
          mobileMenuOpen = false;
          mobileShowQueue = false;
          void navigate({ name: 'home' });
        }}
        class="flex flex-col items-center justify-center gap-0.5 py-1 rounded-lg cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {!mobileShowQueue && !mobileMenuOpen && currentRoute.name === 'home'
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon
          name="house"
          weight={!mobileShowQueue && !mobileMenuOpen && currentRoute.name === 'home' ? 'fill' : 'regular'}
          size={20}
        />
        <span class="text-[10px] font-semibold">Home</span>
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
        class="flex flex-col items-center justify-center gap-0.5 py-1 rounded-lg cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {!mobileShowQueue && !mobileMenuOpen && currentRoute.name === 'search'
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon
          name="magnifying-glass"
          weight={!mobileShowQueue && !mobileMenuOpen && currentRoute.name === 'search' ? 'bold' : 'regular'}
          size={20}
        />
        <span class="text-[10px] font-semibold">Search</span>
      </button>

      <button
        type="button"
        onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
        class="flex flex-col items-center justify-center gap-0.5 py-1 rounded-lg cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {mobileMenuOpen
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon name="books" weight={mobileMenuOpen ? 'fill' : 'regular'} size={20} />
        <span class="text-[10px] font-semibold">Your Library</span>
      </button>

      <button
        type="button"
        onclick={() => {
          mobileMenuOpen = false;
          mobileShowQueue = !mobileShowQueue;
        }}
        class="flex flex-col items-center justify-center gap-0.5 py-1 rounded-lg cursor-pointer transition-all duration-150 hover:bg-[var(--md-sys-color-state-hover)] active:scale-95"
        style="color: {mobileShowQueue
          ? 'var(--md-sys-color-on-surface)'
          : 'var(--md-sys-color-on-surface-variant)'};"
      >
        <Icon name="queue" weight={mobileShowQueue ? 'fill' : 'regular'} size={20} />
        <span class="text-[10px] font-semibold">Queue</span>
      </button>
    </nav>

    <MobileNowPlaying
      open={mobileNowPlayingOpen}
      {currentTrack}
      lyrics={currentLyrics}
      {queue}
      {isPlaying}
      {positionSec}
      {durationSec}
      {shuffle}
      {repeatMode}
      {mediaBaseUrl}
      showMiniLyrics={getPrefBool(preferences, 'lyrics.miniLyrics')}
      plainFallback={getPrefBool(preferences, 'lyrics.plainFallback')}
      onClose={() => (mobileNowPlayingOpen = false)}
      onTogglePlay={togglePlay}
      onPrev={playPrev}
      onNext={playNext}
      onSeek={handleSeek}
      onToggleShuffle={() => {
        shuffle = !shuffle;
        if (isJamRemote) void sendJamCommand('shuffle');
        else api.playerControl('shuffle');
      }}
      onCycleRepeat={() => {
        repeatMode = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
        if (isJamRemote) void sendJamCommand('repeat');
        else api.playerControl('repeat', 0, repeatMode);
      }}
      onToggleFavorite={handleToggleFavoriteTrack}
      onPlayTrack={playTrack}
      onOpenLyrics={() => {
        mobileShowQueue = false;
        void navigate({ name: 'lyrics' });
      }}
      onOpenQueue={() => {
        mobileShowQueue = true;
      }}
    />
  {/if}
</div>

<StreakModal
  open={showStreakModal}
  {fireStreak}
  onClose={() => (showStreakModal = false)}
  onOpenFullPage={() => navigate({ name: 'fire-streak' })}
/>

<CreatePlaylistModal
  open={showCreatePlaylistModal}
  initialIsLocal={createPlaylistIsLocal}
  onClose={() => (showCreatePlaylistModal = false)}
  onSubmit={handleCreatePlaylist}
/>

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
