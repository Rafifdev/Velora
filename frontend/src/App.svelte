<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from './lib/api';
  import { VeloraAudioEngine } from './lib/audioEngine';
  import { getPrefBool, getPrefValue } from './lib/nuclearPrefs';
  import type {
    Album,
    Artist,
    FireStreakStatus,
    ListeningStats,
    PlayHistoryItem,
    Playlist,
    PlaylistDetail,
    Source,
    Track,
    TrackLyrics,
    ViewRoute,
    YTMusicCard,
    YTMusicShelf,
    YTMusicTrack
  } from './lib/types';

  import Titlebar from './components/Titlebar.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import RightPanel from './components/RightPanel.svelte';
  import PlayerBar from './components/PlayerBar.svelte';
  import StreakModal from './components/StreakModal.svelte';
  import CreatePlaylistModal from './components/CreatePlaylistModal.svelte';
  import SplashScreen from './components/SplashScreen.svelte';
  import LyricsView from './components/LyricsView.svelte';

  import HomeView from './views/HomeView.svelte';
  import LibraryViews from './views/LibraryViews.svelte';
  import PlaylistViews from './views/PlaylistViews.svelte';
  import SourcesView from './views/SourcesView.svelte';
  import AnalyticsAndSettingsViews from './views/AnalyticsAndSettingsViews.svelte';

  let tracks = $state<Track[]>([]);
  let showSplash = $state(true);
  let albums = $state<Album[]>([]);
  let artists = $state<Artist[]>([]);
  let playlists = $state<Playlist[]>([]);
  let sources = $state<Source[]>([]);
  let history = $state<PlayHistoryItem[]>([]);
  let preferences = $state<Record<string, string>>({});
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
  let ytLoading = $state(false);
  let ytHomeLoading = $state(false);
  let ytLoadingMore = $state(false);
  let ytMorePage = $state(0);
  let ytMoreAllBuffer: YTMusicCard[] = [];
  let loadingCollectionId = $state<string | null>(null);
  let playRequestSeq = 0;
  let switchingTrack = false;
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

  // Modals
  let showStreakModal = $state(false);
  let showCreatePlaylistModal = $state(false);
  let createPlaylistIsLocal = $state(true);

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

  const isDark = $derived(getPrefBool(preferences, 'theme.dark'));
  const themeSeed = $derived(getPrefValue(preferences, 'theme.seed') || 'monolith');
  const accentColor = $derived(
    preferences['accent_color'] || (isDark ? '#FF9E64' : '#EA6A1F')
  );

  onMount(async () => {
    await Promise.all([refreshAll(), loadYTMusicHome()]);
  });

  async function loadYTMusicHome() {
    ytHomeLoading = true;
    try {
      const feed = await api.getYTMusicHome();
      ytShelves = feed?.shelves || [];
      ytResults = feed?.tracks || [];
    } catch {
      ytShelves = [];
    } finally {
      ytHomeLoading = false;
    }
  }

  async function refreshAll() {
    const boot = await api.getBootstrapData();
    tracks = boot.tracks;
    albums = boot.albums;
    artists = boot.artists;
    playlists = boot.playlists;
    sources = boot.sources;
    history = boot.history;
    preferences = boot.preferences;
    fireStreak = boot.fireStreak;
    listeningStats = boot.listeningStats;
    mediaBaseUrl = boot.mediaBaseUrl;

    if (queue.length === 0 && tracks.length > 0) {
      queue = [...tracks];
      if (!currentTrack) {
        currentTrack = tracks[0];
        durationSec = tracks[0].durationSec;
        loadLyricsForTrack(tracks[0]);
      }
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

    if (preferences['volume']) {
      const v = Number(preferences['volume']);
      if (Number.isFinite(v)) volume = v;
    }
    audioEngine.setVolume(volume, muted);
  }

  async function patchPrefs(patch: Record<string, string>) {
    preferences = { ...preferences, ...patch };
    await api.savePreferences(patch);
  }

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
    if (route.name === 'playlist-detail') {
      selectedPlaylistDetail = await api.getPlaylistDetail(route.playlistId);
    } else if (route.name === 'album-detail') {
      detailTracks = await api.getAlbumTracks(route.albumId);
    } else if (route.name === 'artist-detail') {
      detailTracks = await api.getArtistTracks(route.artistId);
    } else if (route.name === 'history') {
      history = await api.getPlayHistory();
    } else if (route.name === 'listening-stats') {
      listeningStats = await api.getListeningStats();
      fireStreak = await api.getFireStreak();
    } else if (route.name === 'fire-streak') {
      fireStreak = await api.getFireStreak();
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
    ytLoading = true;
    try {
      const primary = await api.searchYouTubeMusic(q);
      if (primary && primary.length >= 20) {
        ytResults = primary;
        return;
      }
      const seen = new Set((primary || []).map((t) => t.videoId));
      const merged = [...(primary || [])];
      const extra = await api.searchYouTubeMusic(`${q} populer`).catch(() => []);
      for (const item of extra || []) {
        if (!item.videoId || seen.has(item.videoId)) continue;
        seen.add(item.videoId);
        merged.push(item);
      }
      ytResults = merged;
    } finally {
      ytLoading = false;
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

  const CATEGORY_MORE_QUERIES: Record<'musik' | 'playlist' | 'podcast' | 'buku', string[]> = {
    musik: [
      'Trending Pop Music Indonesia',
      'Top Hits Indonesia 2025 2026',
      'Viral Pop & R&B Hits',
      'Best Acoustic Pop Hits'
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

  async function handleLoadMoreYTMusic(activeCategory: 'semua' | 'musik' | 'playlist' | 'podcast' | 'buku') {
    if (ytLoadingMore) return;
    ytLoadingMore = true;
    try {
      const page = ytMorePage++;
      const q =
        activeCategory === 'semua'
          ? MORESONGS_QUERIES[page % MORESONGS_QUERIES.length]
          : CATEGORY_MORE_QUERIES[activeCategory][page % CATEGORY_MORE_QUERIES[activeCategory].length];

      if (activeCategory === 'playlist') {
        const morePlaylists = await api.searchYouTubeMusicPlaylists(q);
        if (!morePlaylists || morePlaylists.length === 0) return;
        const existingKeys = new Set(ytPlaylistResults.map((c) => c.browseId || c.playlistId || c.id));
        const unique = morePlaylists.filter((c) => {
          const k = c.browseId || c.playlistId || c.id;
          return k && !existingKeys.has(k);
        });
        if (unique.length > 0) {
          ytPlaylistResults = [...ytPlaylistResults, ...unique];
        }
        return;
      }

      const moreTracks = await api.searchYouTubeMusic(q);
      if (!moreTracks || moreTracks.length === 0) return;

      if (activeCategory === 'semua') {
        const existingVideoIds = new Set<string>(ytMoreAllBuffer.map((c) => c.videoId));
        for (const shelf of ytShelves) {
          for (const item of shelf.items) {
            if (item.videoId) existingVideoIds.add(item.videoId);
          }
        }
        const newCards = moreTracks
          .filter((t) => t.videoId && !existingVideoIds.has(t.videoId))
          .map((t) => ({
            id: `song:${t.videoId}`,
            title: t.title,
            subtitle: t.artist || 'YouTube Music',
            coverUrl: t.coverUrl,
            kind: 'song' as const,
            videoId: t.videoId,
            browseId: '',
            playlistId: '',
            artist: t.artist,
            album: t.album,
            durationSec: t.durationSec
          }));

        const pool = [...ytMoreAllBuffer, ...newCards];
        const fullCount = pool.length >= 5 ? Math.floor(pool.length / 5) * 5 : pool.length;
        const toAppend = pool.slice(0, fullCount);
        ytMoreAllBuffer = pool.slice(fullCount);

        if (toAppend.length > 0) {
          const targetTitle = 'Lagu Lainnya untuk Anda';
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
        const unique = moreTracks.filter((t) => t.videoId && !existingIds.has(t.videoId));
        if (unique.length > 0) {
          ytResults = [...ytResults, ...unique];
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
    if (tr.externalId) {
      return `${mediaBaseUrl}/media/yt?v=${encodeURIComponent(tr.externalId)}`;
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

    return shuffleArray(pool).slice(0, limit);
  }

  function buildLocalAlgorithmQueue(tr: Track): Track[] {
    const k = trackQueueKey(tr);
    playedAlgoKeys = new Set<string>([k]);
    algoBatchRound = 0;
    const sameArtist = tracks.filter(
      (t) => t.id !== tr.id && t.artistName.toLowerCase() === tr.artistName.toLowerCase()
    );
    const sameAlbum = tracks.filter(
      (t) =>
        t.id !== tr.id &&
        t.artistName.toLowerCase() !== tr.artistName.toLowerCase() &&
        tr.albumTitle &&
        t.albumTitle.toLowerCase() === tr.albumTitle.toLowerCase()
    );
    const others = tracks.filter(
      (t) =>
        t.id !== tr.id &&
        t.artistName.toLowerCase() !== tr.artistName.toLowerCase() &&
        (!tr.albumTitle || t.albumTitle.toLowerCase() !== tr.albumTitle.toLowerCase())
    );
    const combined = shuffleArray([...sameArtist, ...sameAlbum, ...others]);
    const picked = combined.slice(0, 20);
    if (picked.length < 20) {
      const extra = collectFallbackGenrePool(new Set([k, ...picked.map(trackQueueKey)]), 20 - picked.length);
      picked.push(...extra);
    }
    for (const item of picked) {
      playedAlgoKeys.add(trackQueueKey(item));
    }
    return [tr, ...picked.slice(0, 20)];
  }

  function getGenreQueriesForBatch(baseTrack: Track, round: number): string[] {
    const artist = baseTrack.artistName && baseTrack.artistName !== 'YouTube Music' ? baseTrack.artistName : '';
    const genre = baseTrack.albumTitle && baseTrack.albumTitle !== 'YouTube Music' ? baseTrack.albumTitle : '';
    const allQueries = [
      artist ? `${artist} lagu` : 'Lagu Indonesia Populer Terbaru',
      artist ? `${artist} mix` : 'Top Global Hits Music',
      'Pop Indonesia Hits',
      'Lagu Indie & Chill Indonesia',
      'Musik Indonesia Terbaru',
      'Top Pop Hits Mix',
      'Acoustic & Chill Pop',
      'R&B Soul Groove Hits',
      genre ? `${genre} music` : 'Trending Music Mix'
    ].filter(Boolean);

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
    const queries = getGenreQueriesForBatch(baseTrack, round);
    const responses = await Promise.all(
      queries.map((q) => api.searchYouTubeMusic(q).catch(() => [] as YTMusicTrack[]))
    );

    const seen = new Set<string>(excludeKeys);
    const candidates: Track[] = [];
    let idxCounter = 0;

    for (const list of responses) {
      for (const item of list || []) {
        if (!item.videoId) continue;
        const key = `yt:${item.videoId}`;
        if (seen.has(key)) continue;
        seen.add(key);
        candidates.push(ytTrackToQueueTrack(item, idxCounter++));
      }
    }

    let picked = shuffleArray(candidates).slice(0, 20);
    if (picked.length < 20) {
      const extra = collectFallbackGenrePool(seen, 20 - picked.length);
      picked = [...picked, ...extra].slice(0, 20);
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
    const seed = collectFallbackGenrePool(playedAlgoKeys, 20);
    for (const s of seed) {
      playedAlgoKeys.add(trackQueueKey(s));
    }
    await playTrack(tr, [tr, ...seed], '');
    void refreshAll();
    void populateYTMusicAlgorithmQueue(tr);
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

  async function handlePlayAlbum(albumId: number) {
    const albumTracks = await api.getAlbumTracks(albumId);
    if (albumTracks && albumTracks.length > 0) {
      const albumName = albums.find((a) => a.id === albumId)?.title || 'Album';
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
        queueSourceTitle = 'Favorite tracks';
      }
    } else {
      queueSourceTitle = '';
    }

    if (list.length > 0) {
      queue = [...list];
    } else {
      queue = buildLocalAlgorithmQueue(tr);
    }

    const prevTrack = currentTrack;
    const prevAcc = listenedAccumulator;
    const prevRecorded = recordedForCurrentTrack;
    if (prevTrack && prevAcc >= 3 && !prevRecorded) {
      void flushListenProgress(prevTrack.id, Math.round(prevAcc), false);
    }

    const seq = ++playRequestSeq;
    currentTrack = tr;
    listenedAccumulator = 0;
    recordedForCurrentTrack = false;
    positionSec = 0;
    durationSec = tr.durationSec;
    loadLyricsForTrack(tr);

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
      switchingTrack = false;
      try {
        await audioEngine.resume();
        audioEngine.fadeIn(0.03);
        await audioEl.play();
        if (seq === playRequestSeq) {
          isPlaying = true;
        }
      } catch (err: any) {
        if (err?.name !== 'AbortError' && seq === playRequestSeq) {
          isPlaying = false;
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
    if (isPlaying) {
      listenedAccumulator += 0.25;
      if (listenedAccumulator >= 10 && !recordedForCurrentTrack) {
        recordedForCurrentTrack = true;
        flushListenProgress(currentTrack.id, 10, false);
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
    } else if (queueSourceTitle === '' && currentTrack) {
      algoBatchRound += 1;
      const nextBatch = await fetchTwentyGenreTracks(currentTrack, playedAlgoKeys, algoBatchRound);
      if (nextBatch.length > 0) {
        await playTrack(nextBatch[0], nextBatch, '');
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
    if (audioEl && audioEl.currentTime > 3) {
      audioEl.currentTime = 0;
      return;
    }
    if (queue.length === 0) return;
    const idx = currentTrack ? queue.findIndex((t) => t.id === currentTrack!.id) : 0;
    const prevIdx = idx - 1 >= 0 ? idx - 1 : queue.length - 1;
    await playTrack(queue[prevIdx], queue);
  }

  function handleSeek(sec: number) {
    if (!audioEl) return;
    audioEl.currentTime = sec;
    positionSec = sec;
    api.playerControl('seek', sec);
  }

  function nudgeSeek(delta: number) {
    if (!currentTrack) return;
    const maxDur = durationSec > 0 ? durationSec : currentTrack.durationSec || 0;
    const targetSec = Math.max(0, maxDur > 0 ? Math.min(maxDur, positionSec + delta) : positionSec + delta);
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

  function handleGlobalKeydown(e: KeyboardEvent) {
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
    audioEngine.setVolume(volume, muted);
    api.playerControl('volume', vol);
  }

  function handleToggleMute() {
    muted = !muted;
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

  async function handleToggleFavoriteArtist(id: number) {
    const fav = await api.toggleFavoriteArtist(id);
    artists = artists.map((a) => (a.id === id ? { ...a, isFavorite: fav } : a));
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
  data-theme={isDark ? 'dark' : 'light'}
  data-seed={themeSeed}
  class="h-screen w-screen flex flex-col overflow-hidden transition-colors duration-300"
  style="background-color: var(--md-sys-color-surface-container-lowest); color: var(--md-sys-color-on-surface);"
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
    {isDark}
    {accentColor}
    onBack={goBack}
    onForward={goForward}
    onNavigate={navigate}
    onSearch={handleGlobalSearch}
    onToggleTheme={() => patchPrefs({ 'theme.dark': isDark ? 'false' : 'true' })}
    onOpenStreakModal={() => (showStreakModal = true)}
  />

  <!-- Main 3-Section Resizable & Collapsible Borderless Workspace -->
  <div class="flex-1 min-h-0 flex gap-2 px-2 pb-2 overflow-hidden">
    <!-- SECTION 1 (LEFT): Menu Sidebar (Same wrapper, swaps inner content to PreferencesSidebar when in Preferences) -->
    <Sidebar
      {currentRoute}
      width={leftWidth}
      collapsed={leftCollapsed}
      {accentColor}
      favoriteAlbumsCount={albums.filter((a) => a.isFavorite).length}
      favoriteTracksCount={tracks.filter((t) => t.isFavorite).length}
      favoriteArtistsCount={artists.filter((a) => a.isFavorite).length}
      playlistsCount={playlists.length}
      sourcesCount={sources.length}
      isPreferencesMode={currentRoute.name === 'preferences'}
      bind:activePrefTab
      bind:prefSearch
      {prefSavedToast}
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

    <!-- SECTION 2 (CENTER): Songs & Playlists + Spotify-style Home -->
    <main
      class="flex-1 min-w-0 min-h-0 overflow-y-auto overflow-x-hidden scroll-pt-16 rounded-lg transition-colors duration-300"
      style="background-color: var(--md-sys-color-surface);"
    >
      {#if currentRoute.name === 'home'}
        <HomeView
          {albums}
          {playlists}
          {ytShelves}
          {ytResults}
          {ytPlaylistResults}
          {ytLoading}
          {ytHomeLoading}
          {ytLoadingMore}
          {loadingCollectionId}
          {mediaBaseUrl}
          onPlayYTMusic={playYTMusicSingle}
          onPlayYTCollection={handlePlayYTCollection}
          onPlayPlaylist={handlePlayPlaylist}
          onPlayAlbum={handlePlayAlbum}
          onSearchYTMusic={handleSearchYTMusic}
          onSearchYTMusicPlaylists={handleSearchYTMusicPlaylists}
          onRefreshYTMusicHome={loadYTMusicHome}
          onLoadMoreYTMusic={handleLoadMoreYTMusic}
          onNavigate={navigate}
        />
      {:else if currentRoute.name === 'search' || currentRoute.name === 'liked-songs' || currentRoute.name === 'favorite-albums' || currentRoute.name === 'album-detail' || currentRoute.name === 'favorite-artists' || currentRoute.name === 'artist-detail'}
        <LibraryViews
          mode={currentRoute.name}
          {searchQuery}
          {tracks}
          {albums}
          {artists}
          selectedAlbum={currentRoute.name === 'album-detail'
            ? albums.find((a) => a.id === currentRoute.albumId) ?? null
            : null}
          selectedArtist={currentRoute.name === 'artist-detail'
            ? artists.find((a) => a.id === currentRoute.artistId) ?? null
            : null}
          {detailTracks}
          {ytResults}
          {ytLoading}
          {playlists}
          currentTrackId={currentTrack?.id ?? null}
          {isPlaying}
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
          onAddToPlaylist={async (pid, tid) => {
            await api.addTrackToPlaylist(pid, tid);
            playlists = await api.getPlaylists(false);
          }}
          onOpenAlbum={(albumId) => navigate({ name: 'album-detail', albumId })}
          onOpenArtist={(artistId) => navigate({ name: 'artist-detail', artistId })}
        />
      {:else if currentRoute.name === 'playlists' || currentRoute.name === 'local-playlists' || currentRoute.name === 'playlist-detail'}
        <PlaylistViews
          mode={currentRoute.name}
          {playlists}
          detail={selectedPlaylistDetail}
          allTracks={tracks}
          currentTrackId={currentTrack?.id ?? null}
          {mediaBaseUrl}
          {accentColor}
          onOpenPlaylist={(playlistId) => navigate({ name: 'playlist-detail', playlistId })}
          onPlayPlaylist={handlePlayPlaylist}
          onCreatePlaylist={(isLocal) => {
            createPlaylistIsLocal = isLocal;
            showCreatePlaylistModal = true;
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
          onAddTrack={async (pid, tid) => {
            await api.addTrackToPlaylist(pid, tid);
            playlists = await api.getPlaylists(false);
            selectedPlaylistDetail = await api.getPlaylistDetail(pid);
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
          onExportM3U={(pid) => api.exportLocalPlaylistM3U(pid)}
          onImportM3U={async () => {
            const pl = await api.importLocalPlaylistM3U();
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
      {:else if currentRoute.name === 'lyrics'}
        <LyricsView
          track={currentTrack}
          lyrics={currentLyrics}
          {lyricsLoading}
          {positionSec}
          {mediaBaseUrl}
          onSeek={handleSeek}
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
            await api.clearCache();
          }}
        />
      {/if}
    </main>

    <!-- SECTION 3 (RIGHT): Hidden on Preferences page so it feels like a dedicated full settings page -->
    {#if currentRoute.name !== 'preferences'}
      <RightPanel
        width={rightWidth}
        collapsed={rightCollapsed}
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
      />
    {/if}
  </div>

  <!-- Bottom Full-Width Flat Player Bar (No rounded corners) -->
  <div class="w-full shrink-0">
    <PlayerBar
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
      onTogglePlay={togglePlay}
      onPrev={playPrev}
      onNext={playNext}
      onSeek={handleSeek}
      onVolumeChange={handleVolumeChange}
      onToggleMute={handleToggleMute}
      onToggleShuffle={() => {
        shuffle = !shuffle;
        api.playerControl('shuffle');
      }}
      onCycleRepeat={() => {
        repeatMode = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
        api.playerControl('repeat', 0, repeatMode);
      }}
      onToggleFavorite={handleToggleFavoriteTrack}
      onToggleLyrics={() => {
        if (currentRoute.name === 'lyrics') {
          if (navIndex > 0) goBack();
          else navigate({ name: 'home' });
        } else {
          navigate({ name: 'lyrics' });
        }
      }}
      onFocusRightCard={(card) => {
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
        rightCollapsed = false;
        rightPanelMode = 'now-playing';
        patchPrefs({ 'layout.rightCollapsed': 'false' });
        focusRightSignal = { card, seq: Date.now() };
      }}
    />
  </div>
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
