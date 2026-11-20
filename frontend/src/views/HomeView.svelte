<script lang="ts">
  import { untrack } from 'svelte';
  import { isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Album, Artist, HomeFilter, Playlist, Track, ViewRoute, YTMusicCard, YTMusicShelf, YTMusicTrack } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
  import Icon from '../components/Icon.svelte';

  interface Props {
    albums?: Album[];
    artists?: Artist[];
    tracks?: Track[];
    playlists: Playlist[];
    ytShelves: YTMusicShelf[];
    ytResults: YTMusicTrack[];
    ytPlaylistResults?: YTMusicCard[];
    ytAlbumResults?: YTMusicCard[];
    ytArtistResults?: YTMusicCard[];
    initialFilter?: HomeFilter;
    onFilterChange?: (filter: HomeFilter) => void;
    ytLoading: boolean;
    ytHomeLoading: boolean;
    ytLoadingMore?: boolean;
    ytLoadingMorePopularArtists?: boolean;
    loadingCollectionId: string | null;
    currentExternalId?: string | null;
    currentTrackTitle?: string | null;
    currentTrackArtist?: string | null;
    currentAlbumTitle?: string | null;
    queueSourceTitle?: string | null;
    isPlaying?: boolean;
    onTogglePlay?: () => void;
    mediaBaseUrl: string;
    onPlayYTMusic: (yt: YTMusicTrack) => void;
    onPlayYTCollection: (card: YTMusicCard) => void;
    onOpenYTAlbum?: (card: YTMusicCard) => void;
    onPlayPlaylist?: (playlistId: number) => void;
    onPlayAlbum?: (albumId: number) => void;
    onOpenArtist?: (artistId: number) => void;
    onToggleFavoriteArtist?: (artistId: number) => void;
    onToggleFavoriteYTMusicArtist?: (payload: {
      externalId: string;
      name: string;
      avatarUrl: string;
    }) => void;
    onSearchYTMusic: (query: string) => void;
    onSearchYTMusicPlaylists?: (query: string) => void;
    onSearchYTMusicAlbums?: (query: string) => void;
    onSearchYTMusicArtists?: (query: string) => void;
    onRefreshYTMusicHome: () => void;
    onLoadMoreYTMusic?: (category: HomeFilter) => void;
    onLoadMorePopularArtists?: () => void;
    onNavigate: (route: ViewRoute) => void;
    lang?: LangCode;
    showGreeting?: boolean;
    showRecentHome?: boolean;
    meltCards?: boolean;
  }

  let {
    albums = [],
    artists = [],
    tracks = [],
    playlists,
    ytShelves,
    ytResults,
    ytPlaylistResults = [],
    ytAlbumResults = [],
    ytArtistResults = [],
    initialFilter,
    onFilterChange,
    ytLoading,
    ytHomeLoading,
    ytLoadingMore = false,
    ytLoadingMorePopularArtists = false,
    loadingCollectionId,
    currentExternalId = null,
    currentTrackTitle = null,
    currentTrackArtist = null,
    currentAlbumTitle = null,
    queueSourceTitle = null,
    isPlaying = false,
    onTogglePlay,
    mediaBaseUrl,
    onPlayYTMusic,
    onPlayYTCollection,
    onOpenYTAlbum,
    onPlayPlaylist,
    onOpenArtist,
    onToggleFavoriteArtist,
    onToggleFavoriteYTMusicArtist,
    onSearchYTMusic,
    onSearchYTMusicPlaylists,
    onSearchYTMusicAlbums,
    onSearchYTMusicArtists,
    onRefreshYTMusicHome,
    onLoadMoreYTMusic,
    onLoadMorePopularArtists,
    onNavigate,
    lang = 'en',
    showGreeting: _showGreeting = true,
    showRecentHome: _showRecentHome = true,
    meltCards: _meltCards = true
  }: Props = $props();

  let activeFilterPill = $state<HomeFilter>('semua');
  let filteredMoreLoaded = $state(false);
  let expandedShelves = $state<Record<string, boolean>>({ 'yt-Lagu Lainnya untuk Anda': true });
  let stickyContainerEl: HTMLElement | undefined = $state();
  let pillsStuck = $state(false);
  let scrollParentEl: HTMLElement | null = null;
  const categoryScrollMap: Partial<Record<HomeFilter, number>> = {};

  let lastInitFilter: HomeFilter | undefined = undefined;
  $effect(() => {
    if (!initialFilter || initialFilter === lastInitFilter) return;
    lastInitFilter = initialFilter;
    activeFilterPill = initialFilter;
    filteredMoreLoaded = false;
    untrack(() => {
      if (initialFilter === 'album' && ytAlbumResults.length === 0) {
        onSearchYTMusicAlbums?.('');
      } else if (initialFilter === 'artis' && ytArtistResults.length === 0) {
        onSearchYTMusicArtists?.('');
      } else if (initialFilter === 'musik' && ytResults.length === 0) {
        onSearchYTMusic('');
      } else if (initialFilter === 'playlist' && ytPlaylistResults.length === 0) {
        onSearchYTMusicPlaylists?.('Top Hits Indonesia Playlist');
      }
    });
  });

  $effect(() => {
    const el = stickyContainerEl;
    if (!el) return;

    function findScrollParent(node: HTMLElement | null): HTMLElement | null {
      let cur = node?.parentElement ?? null;
      while (cur && cur !== document.body && cur !== document.documentElement) {
        const oy = getComputedStyle(cur).overflowY;
        if (oy === 'auto' || oy === 'scroll' || oy === 'overlay') {
          return cur;
        }
        cur = cur.parentElement;
      }
      return null;
    }

    const scrollParent = findScrollParent(el);
    scrollParentEl = scrollParent;
    const updateStuck = () => {
      const top = scrollParent ? scrollParent.scrollTop : window.scrollY;
      pillsStuck = top > 8;
    };

    updateStuck();
    scrollParent?.addEventListener('scroll', updateStuck, { passive: true });
    window.addEventListener('scroll', updateStuck, { passive: true });

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!scrollParent) {
          pillsStuck = entry.intersectionRatio < 1;
        } else {
          pillsStuck = scrollParent.scrollTop > 8;
        }
      },
      {
        root: scrollParent,
        threshold: [1],
        rootMargin: '-1px 0px 0px 0px'
      }
    );
    observer.observe(el);

    return () => {
      scrollParent?.removeEventListener('scroll', updateStuck);
      window.removeEventListener('scroll', updateStuck);
      observer.disconnect();
    };
  });

  const filterPills = $derived([
    { id: 'semua', label: tr(lang, 'nav.all') },
    { id: 'musik', label: tr(lang, 'nav.music') },
    { id: 'artis', label: tr(lang, 'nav.artists') },
    { id: 'album', label: tr(lang, 'nav.albums') }
  ] as const);

  const pillHeading = $derived(
    activeFilterPill === 'musik'
      ? tr(lang, 'nav.music')
      : activeFilterPill === 'artis'
        ? tr(lang, 'nav.artists')
        : activeFilterPill === 'album'
          ? tr(lang, 'nav.albums')
          : activeFilterPill === 'playlist'
            ? tr(lang, 'nav.playlists')
            : activeFilterPill === 'podcast'
              ? 'Podcast'
              : 'Buku Audio'
  );

  interface HomeArtistCard {
    key: string;
    id: number | null;
    browseId: string;
    name: string;
    subtitle: string;
    rawAvatar: string;
    isFavorite: boolean;
    trackCount: number;
  }

  const ID_EN_HINT_RE = /\b(dan|yang|di|ke|dari|untuk|dengan|aku|kamu|kita|cinta|hati|lagu|indonesia|pop|hits|love|you|me|my|the|and|for|with|best|top|night|song|music|feat|ft)\b/i;
  const NON_LATIN_RE = /[^\u0000-\u024F\u1E00-\u1EFF]/;

  function langPriorityScore(text: string): number {
    const t = (text || '').trim();
    if (!t) return 2;
    if (NON_LATIN_RE.test(t)) return 2;
    if (ID_EN_HINT_RE.test(t)) return 0;
    return 1;
  }

  function prioritizeIdEn<T>(items: T[], getText: (item: T) => string): T[] {
    return [...items].sort((a, b) => langPriorityScore(getText(a)) - langPriorityScore(getText(b)));
  }

  const displayedHomeArtists = $derived.by<HomeArtistCard[]>(() => {
    const dbByName = new Map<string, Artist>();
    for (const ar of artists) {
      const k = ar.name.trim().toLowerCase();
      if (!k || k === 'unknown artist' || k === 'youtube music artist') continue;
      if (!dbByName.has(k)) dbByName.set(k, ar);
    }

    const byName = new Map<string, HomeArtistCard>();
    if (ytArtistResults.length > 0) {
      for (const ytAr of ytArtistResults) {
        const rawName = ytAr.title.trim();
        const k = rawName.toLowerCase();
        if (!k || k === 'unknown artist' || k === 'youtube music artist') continue;
        if (byName.has(k)) continue;
        const dbMatch = dbByName.get(k);
        byName.set(k, {
          key: `yt:${ytAr.browseId || k}`,
          id: dbMatch ? dbMatch.id : null,
          browseId: ytAr.browseId || dbMatch?.externalId || '',
          name: rawName,
          subtitle: 'Artis',
          rawAvatar: ytAr.coverUrl || dbMatch?.avatarUrl || '',
          isFavorite: dbMatch ? dbMatch.isFavorite : false,
          trackCount: dbMatch ? dbMatch.trackCount : 0
        });
      }
      const list = Array.from(byName.values());
      if (!filteredMoreLoaded && list.length > 28) {
        return list.slice(0, 28);
      }
      return list;
    }

    for (const ar of artists) {
      const rawName = ar.name.trim();
      const k = rawName.toLowerCase();
      if (!k || k === 'unknown artist' || k === 'youtube music artist' || rawName.includes(',')) continue;
      if (!byName.has(k)) {
        byName.set(k, {
          key: `db:${ar.id}`,
          id: ar.id,
          browseId: ar.externalId || '',
          name: rawName,
          subtitle: 'Artis',
          rawAvatar: ar.avatarUrl,
          isFavorite: ar.isFavorite,
          trackCount: ar.trackCount
        });
      }
    }
    const list = prioritizeIdEn(Array.from(byName.values()), (ar) => ar.name);
    if (!filteredMoreLoaded && list.length > 28) {
      return list.slice(0, 28);
    }
    return list;
  });

  const displayedFilteredAlbums = $derived.by(() => {
    const head = prioritizeIdEn(ytAlbumResults.slice(0, 28), (a) => `${a.title} ${a.subtitle}`);
    if (!filteredMoreLoaded) return head;
    return [...head, ...ytAlbumResults.slice(28)];
  });

  const displayedFilteredPlaylists = $derived.by(() => {
    const head = prioritizeIdEn(ytPlaylistResults.slice(0, 28), (p) => `${p.title} ${p.subtitle}`);
    if (!filteredMoreLoaded) return head;
    return [...head, ...ytPlaylistResults.slice(28)];
  });

  const displayedFilteredResults = $derived.by(() => {
    const seenCover = new Set<string>();
    const seenAlbum = new Set<string>();
    const deduped: YTMusicTrack[] = [];
    const ordered = [
      ...prioritizeIdEn(ytResults.slice(0, 28), (r) => `${r.title} ${r.artist}`),
      ...ytResults.slice(28)
    ];
    for (const r of ordered) {
      const cov = (r.coverUrl || '').trim();
      const alb = (r.album || '').trim().toLowerCase();
      if (cov && seenCover.has(cov)) continue;
      if (alb && alb !== 'single' && seenAlbum.has(alb)) continue;
      if (cov) seenCover.add(cov);
      if (alb && alb !== 'single') seenAlbum.add(alb);
      deduped.push(r);
    }
    if (!filteredMoreLoaded && deduped.length > 28) {
      return deduped.slice(0, 28);
    }
    return deduped;
  });

  function isTrackCardActive(videoId?: string | null, title?: string | null, artist?: string | null): boolean {
    if (currentExternalId && videoId && currentExternalId === videoId) return true;
    if (
      currentTrackTitle &&
      currentTrackArtist &&
      title &&
      artist &&
      title.trim().toLowerCase() === currentTrackTitle.trim().toLowerCase() &&
      artist.trim().toLowerCase() === currentTrackArtist.trim().toLowerCase()
    ) {
      return true;
    }
    return false;
  }

  function isCollectionCardActive(item: YTMusicCard): boolean {
    if (item.kind === 'song') {
      return isTrackCardActive(item.videoId, item.title, item.artist);
    }
    const itemTitle = (item.title || '').trim().toLowerCase();
    if (!itemTitle) return false;
    if (item.kind === 'album') {
      if (currentAlbumTitle && currentAlbumTitle.trim().toLowerCase() === itemTitle) return true;
      if (queueSourceTitle && queueSourceTitle.trim().toLowerCase() === itemTitle) return true;
    } else if (item.kind === 'playlist') {
      if (queueSourceTitle && queueSourceTitle.trim().toLowerCase() === itemTitle) return true;
    } else if (item.kind === 'artist') {
      if (currentTrackArtist && currentTrackArtist.toLowerCase().includes(itemTitle)) return true;
    }
    return false;
  }

  function toggleShelfExpand(key: string) {
    expandedShelves = {
      ...expandedShelves,
      [key]: !expandedShelves[key]
    };
  }

  function shelfEdgeFade(node: HTMLElement) {
    const update = () => {
      const hasOverflow = node.scrollWidth > node.clientWidth + 4;
      const atStart = node.scrollLeft <= 4;
      const atEnd = node.scrollLeft + node.clientWidth >= node.scrollWidth - 4;
      node.style.setProperty('--fade-start', hasOverflow && !atStart ? '2.75rem' : '0px');
      node.style.setProperty('--fade-end', hasOverflow && !atEnd ? '2.75rem' : '0px');
    };
    update();
    node.addEventListener('scroll', update, { passive: true });
    const ro = new ResizeObserver(update);
    ro.observe(node);
    return {
      destroy() {
        node.removeEventListener('scroll', update);
        ro.disconnect();
      }
    };
  }
</script>

<!-- 1:1 Spotify Borderless Home View: Square Cards Only (Live YouTube Music Data, Zero Dummy Data) -->
<div class="relative min-h-full" style="overflow-anchor: none;">
  <!-- State-Aware Sticky Category Filter Pills Header -->
  <div
    bind:this={stickyContainerEl}
    class="velora-sticky-pills-container"
    class:is-stuck={pillsStuck}
  >
    <div class="velora-sticky-pills-bar px-4 sm:px-6 py-3 flex flex-wrap items-center gap-2">
      {#each filterPills as pill (pill.id)}
        {@const active = activeFilterPill === pill.id}
        <button
          type="button"
          onclick={() => {
            const prevPill = activeFilterPill;
            if (scrollParentEl && prevPill !== pill.id) {
              categoryScrollMap[prevPill] = scrollParentEl.scrollTop;
            }
            lastInitFilter = pill.id;
            activeFilterPill = pill.id;
            onFilterChange?.(pill.id);
            filteredMoreLoaded = false;
            if (prevPill !== pill.id) {
              const targetTop = categoryScrollMap[pill.id] ?? 0;
              requestAnimationFrame(() => {
                if (scrollParentEl) scrollParentEl.scrollTop = targetTop;
                pillsStuck = targetTop > 8;
              });
            }
            if (pill.id === 'semua') {
              if (ytShelves.length === 0) onRefreshYTMusicHome();
            } else if (pill.id === 'musik') {
              onSearchYTMusic('');
            } else if (pill.id === 'artis') {
              onSearchYTMusicArtists?.('');
            } else if (pill.id === 'album') {
              onSearchYTMusicAlbums?.('');
            }
          }}
          class="px-3.5 py-1.5 rounded-full text-xs font-semibold transition-colors duration-200 cursor-pointer hover:brightness-110"
          style={active
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
        >
          {pill.label}
        </button>
      {/each}
    </div>
  </div>

  <div class="relative z-10 px-4 sm:px-6 pt-2 pb-6 space-y-6 sm:space-y-7">

    {#if activeFilterPill === 'semua'}
      <!-- Saved Library Playlists Shelf (Only shown when real playlists exist) -->
      {#if playlists.length > 0}
        {@const plExpanded = !!expandedShelves['saved-playlists']}
        <section class="min-w-0 space-y-3" use:svelteMotionEnter={{ delay: 0.03 }}>
          <div class="flex items-center justify-between gap-2">
            <h2 class="text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              {tr(lang, 'nav.playlists')}
            </h2>
            {#if playlists.length > 4}
              <button
                type="button"
                onclick={() => toggleShelfExpand('saved-playlists')}
                class="text-xs font-bold hover:underline cursor-pointer"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                {plExpanded ? tr(lang, 'nav.seeLess') : tr(lang, 'nav.seeMore')}
              </button>
            {/if}
          </div>

          <div
            use:shelfEdgeFade
            class={plExpanded
              ? 'velora-card-grid-7'
              : 'flex items-start justify-start gap-2.5 sm:gap-3 overflow-x-auto no-scrollbar velora-shelf-scroll pb-1'}
          >
            {#each playlists as pl (pl.id)}
              {@const cover = resolveCoverUrl(pl.coverUrl, mediaBaseUrl)}
              <div
                role="group"
                class="group {plExpanded ? 'w-full min-w-0' : 'w-[132px] sm:w-[172px] shrink-0'} flex flex-col justify-start items-stretch self-start p-2 sm:p-2.5 rounded-lg transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
              >
                <div
                  class="velora-card-artwork mb-2.5 flex items-center justify-center"
                  style={cover
                    ? 'background-color: var(--md-sys-color-surface-container-high);'
                    : `background: linear-gradient(135deg, color-mix(in srgb, ${pl.coverColor || 'var(--md-sys-color-primary)'} 38%, var(--md-sys-color-surface-container-high)), var(--md-sys-color-surface-container-highest));`}
                >
                  <button
                    type="button"
                    onclick={() => onNavigate({ name: 'playlist-detail', playlistId: pl.id })}
                    aria-label="Buka {pl.name}"
                    class="velora-card-artwork-hit flex items-center justify-center cursor-pointer"
                  >
                    {#if cover}
                      <img
                        src={cover}
                        alt={pl.name}
                        loading="lazy"
                        decoding="async"
                        class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                      />
                    {:else}
                      <Icon name="playlist" weight="fill" size={44} class="text-white/95 drop-shadow-[0_2px_8px_rgba(0,0,0,0.45)]" />
                    {/if}
                  </button>

                  <button
                    type="button"
                    onclick={() =>
                      onPlayPlaylist
                        ? onPlayPlaylist(pl.id)
                        : onNavigate({ name: 'playlist-detail', playlistId: pl.id })}
                    aria-label="Putar {pl.name}"
                    class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200 hover:scale-105"
                  >
                    <Icon name="play" weight="fill" size={18} />
                  </button>
                </div>

                <button
                  type="button"
                  onclick={() => onNavigate({ name: 'playlist-detail', playlistId: pl.id })}
                  class="w-full flex flex-col justify-start items-stretch text-left cursor-pointer"
                >
                  <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                    {pl.name}
                  </div>
                  <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    {pl.description || `${pl.trackCount} lagu`}
                  </div>
                </button>
              </div>
            {/each}
          </div>
        </section>
      {/if}

      <!-- Live YouTube Music Shelves (Square Cards Only) -->
      {#if ytHomeLoading && ytShelves.length === 0}
        {#each [0, 1] as shelfIdx (shelfIdx)}
          <section class="min-w-0 space-y-3">
            <div class="h-6 w-48 rounded-md bg-[var(--md-sys-color-surface-container-highest)] animate-pulse"></div>
            <div class="flex items-start justify-start gap-2.5 sm:gap-3 overflow-hidden pb-1">
              {#each [0, 1, 2, 3, 4, 5] as cardIdx (cardIdx)}
                <div class="w-[132px] sm:w-[172px] shrink-0 p-2 sm:p-2.5 rounded-lg space-y-2.5">
                  <div class="w-full aspect-square rounded-md bg-[var(--md-sys-color-surface-container-high)] animate-pulse"></div>
                  <div class="h-3.5 w-3/4 rounded bg-[var(--md-sys-color-surface-container-highest)] animate-pulse"></div>
                  <div class="h-3 w-1/2 rounded bg-[var(--md-sys-color-surface-container)] animate-pulse"></div>
                </div>
              {/each}
            </div>
          </section>
        {/each}
      {:else if ytShelves.length === 0}
        <section class="p-8 rounded-lg text-center space-y-3" style="background-color: var(--md-sys-color-surface-container-low);">
          <Icon name="youtube-logo" weight="fill" size={40} class="mx-auto opacity-60" />
          <div class="space-y-1">
            <h2 class="text-base font-bold" style="color: var(--md-sys-color-on-surface);">
              Gagal memuat feed YouTube Music
            </h2>
            <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
              Periksa koneksi internet kamu lalu muat ulang untuk mengambil playlist dan musik terbaru langsung dari YouTube Music.
            </p>
          </div>
          <button
            type="button"
            onclick={onRefreshYTMusicHome}
            class="h-9 px-5 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold cursor-pointer hover:brightness-105 transition"
          >
            Muat Ulang YouTube Music
          </button>
        </section>
      {:else}
        {#each ytShelves as shelf, sIdx (shelf.title)}
          {@const shelfKey = `yt-${shelf.title}`}
          {@const isAlwaysFull = shelf.title === 'Lagu Lainnya untuk Anda'}
          {@const isExpanded = isAlwaysFull || !!expandedShelves[shelfKey]}
          <section class="min-w-0 space-y-3" use:svelteMotionEnter={{ delay: Math.min(0.18, sIdx * 0.04) }}>
            <div class="flex items-center justify-between gap-2">
              <h2 class="text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                {shelf.title}
              </h2>
              {#if !isAlwaysFull && shelf.items.length > 4}
                <button
                  type="button"
                  onclick={() => toggleShelfExpand(shelfKey)}
                  class="text-xs font-bold hover:underline cursor-pointer"
                  style="color: var(--md-sys-color-on-surface-variant);"
                >
                  {isExpanded ? 'Tampilkan lebih sedikit' : 'Tampilkan semua'}
                </button>
              {/if}
            </div>

            <div
              use:shelfEdgeFade
              class={isExpanded
                ? 'velora-card-grid-7'
                : 'flex items-start justify-start gap-2.5 sm:gap-3 overflow-x-auto no-scrollbar velora-shelf-scroll pb-1'}
            >
              {#each shelf.items as item (item.id)}
                {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
                {@const isItemLoading = loadingCollectionId === item.id}
                {@const isCardActive = isCollectionCardActive(item)}
                <button
                  type="button"
                  onclick={() => {
                    if (item.kind === 'artist') {
                      onNavigate({
                        name: 'artist-detail',
                        browseId: item.browseId,
                        artistName: item.title,
                        avatarUrl: item.coverUrl
                      });
                    } else if (isCardActive && onTogglePlay) {
                      onTogglePlay();
                    } else {
                      onPlayYTCollection(item);
                    }
                  }}
                  class="group {isExpanded ? 'w-full min-w-0' : 'w-[132px] sm:w-[172px] shrink-0'} flex flex-col justify-start items-stretch self-start p-2 sm:p-2.5 rounded-lg border-0 outline-none focus:outline-none text-left cursor-pointer transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div class="relative mb-2.5">
                    <div
                      class="velora-card-artwork flex items-center justify-center {item.kind === 'artist' ? 'velora-artist-avatar !rounded-full' : ''}"
                      style={cover
                        ? 'background-color: var(--md-sys-color-surface-container-high);'
                        : 'background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));'}
                    >
                      {#if cover}
                        <img
                          src={cover}
                          alt={item.title}
                          loading="lazy"
                          decoding="async"
                          class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                        />
                      {:else}
                        <Icon name="vinyl-record" weight="fill" size={44} class="text-white drop-shadow-sm" />
                      {/if}
                    </div>

                    <span
                      class="absolute z-20 bottom-1.5 right-1.5 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center transition-all duration-200 {isCardActive
                        ? 'opacity-100 translate-y-0'
                        : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                    >
                      {#if isCardActive && isPlaying}
                        <span class="velora-playing-eq" style="color: currentColor;" aria-hidden="true">
                          <span></span>
                          <span></span>
                          <span></span>
                          <span></span>
                        </span>
                        <span class="hidden group-hover:inline-flex items-center justify-center">
                          <Icon name="pause" weight="fill" size={18} />
                        </span>
                      {:else}
                        <Icon name="play" weight="fill" size={18} />
                      {/if}
                    </span>
                  </div>

                  <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                    {item.title}
                  </div>
                  <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    {item.subtitle || 'YouTube Music'}
                  </div>
                </button>
              {/each}
            </div>
            {#if shelf.title === 'Artis Populer' && isExpanded && onLoadMorePopularArtists}
              <div class="pt-1 flex justify-center">
                <button
                  type="button"
                  disabled={ytLoadingMorePopularArtists}
                  onclick={onLoadMorePopularArtists}
                  class="h-9 min-w-[164px] px-5 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-bold flex items-center justify-center cursor-pointer transition-all duration-200 active:scale-95 disabled:opacity-80 disabled:cursor-wait"
                  style="color: var(--md-sys-color-on-surface);"
                >
                  {ytLoadingMorePopularArtists ? 'Memuat...' : 'Muat lebih banyak artis'}
                </button>
              </div>
            {/if}
          </section>
        {/each}
      {/if}
    {:else}
      <!-- Filtered Pill View: Square Cards Grid Directly from YouTube Music -->
      <section class="min-w-0 space-y-4" use:svelteMotionEnter={{ delay: 0.04 }}>
        <h2 class="text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
          {pillHeading}
        </h2>

        {#if ytLoading}
          <div class="velora-card-grid-7">
            {#each Array.from({ length: 28 }, (_, idx) => idx) as idx (idx)}
              <div class="w-full min-w-0 p-2.5 rounded-lg space-y-2.5">
                <div class="w-full aspect-square {activeFilterPill === 'artis' ? 'rounded-full' : 'rounded-md'} bg-[var(--md-sys-color-surface-container-high)] animate-pulse"></div>
                <div class="h-3.5 w-3/4 rounded bg-[var(--md-sys-color-surface-container-highest)] animate-pulse"></div>
                <div class="h-3 w-1/2 rounded bg-[var(--md-sys-color-surface-container)] animate-pulse"></div>
              </div>
            {/each}
          </div>
        {:else if activeFilterPill === 'artis'}
          {#if displayedHomeArtists.length === 0}
            <div class="flex flex-col items-center justify-center text-center py-14 px-4">
              <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
                <Icon name="users" weight="bold" size={38} />
              </div>
              <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                Belum ada artis
              </h2>
              <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
                Artis untuk kategori ini belum tersedia.
              </p>
              <button
                type="button"
                onclick={() => onSearchYTMusicArtists?.('')}
                class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
                style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
              >
                Muat ulang
              </button>
            </div>
          {:else}
            <div class="velora-card-grid-7">
              {#each displayedHomeArtists as ar (ar.key)}
                {@const avatar = resolveCoverUrl(ar.rawAvatar, mediaBaseUrl)}
                <div class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg relative transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]">
                  <div class="relative mb-2.5">
                    <div
                      class="velora-card-artwork velora-artist-avatar !rounded-full flex items-center justify-center shadow-md"
                      style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
                    >
                      <button
                        type="button"
                        onclick={() =>
                          onNavigate({
                            name: 'artist-detail',
                            artistId: ar.id ?? undefined,
                            browseId: ar.browseId,
                            artistName: ar.name,
                            avatarUrl: ar.rawAvatar
                          })}
                        aria-label="Buka artis {ar.name}"
                        class="velora-card-artwork-hit cursor-pointer"
                      >
                        {#if avatar}
                          <img
                            src={avatar}
                            alt={ar.name}
                            loading="lazy"
                            decoding="async"
                            class="object-cover object-center"
                          />
                        {:else}
                          <Icon name="user" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.7;" />
                        {/if}
                      </button>
                    </div>

                    <button
                      type="button"
                      onclick={(e) => {
                        e.stopPropagation();
                        if (ar.id !== null) {
                          onToggleFavoriteArtist?.(ar.id);
                        } else {
                          onToggleFavoriteYTMusicArtist?.({
                            externalId: ar.browseId,
                            name: ar.name,
                            avatarUrl: ar.rawAvatar
                          });
                        }
                      }}
                      title={ar.isFavorite ? 'Hapus dari Favorit' : 'Tambahkan ke Favorit'}
                      aria-label={ar.isFavorite ? 'Hapus dari Favorit' : 'Tambahkan ke Favorit'}
                      class="border-0 shadow-md absolute z-20 bottom-1 right-1 w-8 h-8 rounded-full backdrop-blur-xs flex items-center justify-center cursor-pointer transition-opacity {ar.isFavorite
                        ? 'opacity-100'
                        : 'opacity-0 group-hover:opacity-100'}"
                      style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-highest) 90%, transparent); color: {ar.isFavorite
                        ? 'var(--md-sys-color-primary)'
                        : 'var(--md-sys-color-on-surface)'};"
                    >
                      <Icon name="heart" weight={ar.isFavorite ? 'fill' : 'regular'} size={15} />
                    </button>
                  </div>

                  <button
                    type="button"
                    onclick={() =>
                      onNavigate({
                        name: 'artist-detail',
                        artistId: ar.id ?? undefined,
                        browseId: ar.browseId,
                        artistName: ar.name,
                        avatarUrl: ar.rawAvatar
                      })}
                    class="w-full text-center cursor-pointer"
                  >
                    <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                      {ar.name}
                    </div>
                    <div class="text-[11px] leading-snug truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      {ar.subtitle}
                    </div>
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        {:else if activeFilterPill === 'album'}
          {#if ytAlbumResults.length === 0}
            <div class="flex flex-col items-center justify-center text-center py-14 px-4">
              <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
                <Icon name="vinyl-record" weight="bold" size={38} />
              </div>
              <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                Belum ada album
              </h2>
              <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
                Album untuk kategori ini belum tersedia.
              </p>
              <button
                type="button"
                onclick={() => onSearchYTMusicAlbums?.('')}
                class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
                style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
              >
                Muat ulang
              </button>
            </div>
          {:else}
            <div class="velora-card-grid-7">
              {#each displayedFilteredAlbums as item (item.id)}
                {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
                {@const isItemLoading = loadingCollectionId === item.id}
                {@const isAlbumActive = isCollectionCardActive(item)}
                <div
                  role="group"
                  class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg text-left transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div
                    class="velora-card-artwork mb-2.5 flex items-center justify-center"
                    style={cover
                      ? 'background-color: var(--md-sys-color-surface-container-high);'
                      : 'background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));'}
                  >
                    <button
                      type="button"
                      onclick={() => (onOpenYTAlbum ? onOpenYTAlbum(item) : onPlayYTCollection(item))}
                      aria-label="Buka album {item.title}"
                      class="velora-card-artwork-hit cursor-pointer"
                    >
                      {#if cover}
                        <img
                          src={cover}
                          alt={item.title}
                          loading="lazy"
                          decoding="async"
                          class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                        />
                      {:else}
                        <Icon name="vinyl-record" weight="fill" size={44} class="text-white drop-shadow-sm" />
                      {/if}
                    </button>

                    <button
                      type="button"
                      onclick={() => (isAlbumActive && onTogglePlay ? onTogglePlay() : onPlayYTCollection(item))}
                      aria-label="Putar album {item.title}"
                      class="absolute z-20 bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer transition-all duration-200 {isAlbumActive
                        ? 'opacity-100 translate-y-0'
                        : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                    >
                      {#if isAlbumActive && isPlaying}
                        <span class="velora-playing-eq" style="color: currentColor;" aria-hidden="true">
                          <span></span>
                          <span></span>
                          <span></span>
                          <span></span>
                        </span>
                        <span class="hidden group-hover:inline-flex items-center justify-center">
                          <Icon name="pause" weight="fill" size={18} />
                        </span>
                      {:else}
                        <Icon name="play" weight="fill" size={18} />
                      {/if}
                    </button>
                  </div>

                  <button
                    type="button"
                    onclick={() => (onOpenYTAlbum ? onOpenYTAlbum(item) : onPlayYTCollection(item))}
                    class="w-full text-left cursor-pointer"
                  >
                    <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                      {item.title}
                    </div>
                    <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      {item.subtitle || 'Album • YouTube Music'}
                    </div>
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        {:else if activeFilterPill === 'playlist'}
          {#if ytPlaylistResults.length === 0}
            <div class="flex flex-col items-center justify-center text-center py-14 px-4">
              <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
                <Icon name="playlist" weight="bold" size={38} />
              </div>
              <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                Belum ada playlist
              </h2>
              <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
                Playlist untuk kategori ini belum tersedia.
              </p>
              <button
                type="button"
                onclick={() => onSearchYTMusicPlaylists?.('Top Hits Indonesia Playlist')}
                class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
                style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
              >
                Muat ulang
              </button>
            </div>
          {:else}
            <div class="velora-card-grid-7">
              {#each displayedFilteredPlaylists as item (item.id)}
                {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
                {@const isPlaylistActive = isCollectionCardActive(item)}
                <button
                  type="button"
                  onclick={() => (isPlaylistActive && onTogglePlay ? onTogglePlay() : onPlayYTCollection(item))}
                  class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg text-left cursor-pointer transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div
                    class="velora-card-artwork mb-2.5 flex items-center justify-center"
                    style={cover
                      ? 'background-color: var(--md-sys-color-surface-container-high);'
                      : 'background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));'}
                  >
                    {#if cover}
                      <img
                        src={cover}
                        alt={item.title}
                        loading="lazy"
                        decoding="async"
                        class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                      />
                    {:else}
                      <Icon name="playlist" weight="fill" size={44} class="text-white drop-shadow-sm" />
                    {/if}

                    <span
                      class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center transition-all duration-200 {isPlaylistActive
                        ? 'opacity-100 translate-y-0'
                        : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                    >
                      {#if isPlaylistActive && isPlaying}
                        <span class="velora-playing-eq" style="color: currentColor;" aria-hidden="true">
                          <span></span>
                          <span></span>
                          <span></span>
                          <span></span>
                        </span>
                        <span class="hidden group-hover:inline-flex items-center justify-center">
                          <Icon name="pause" weight="fill" size={18} />
                        </span>
                      {:else}
                        <Icon name="play" weight="fill" size={18} />
                      {/if}
                    </span>
                  </div>

                  <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                    {item.title}
                  </div>
                  <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    {item.subtitle || 'Playlist • YouTube Music'}
                  </div>
                </button>
              {/each}
            </div>
          {/if}
        {:else if ytResults.length === 0}
          <div class="flex flex-col items-center justify-center text-center py-14 px-4">
            <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
              <Icon name="music-note" weight="bold" size={38} />
            </div>
            <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Belum ada lagu
            </h2>
            <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
              Hasil untuk kategori ini belum tersedia.
            </p>
            <button
              type="button"
              onclick={() =>
                onSearchYTMusic(
                  activeFilterPill === 'podcast'
                    ? 'Podcast Musik Indonesia'
                    : activeFilterPill === 'buku'
                      ? 'Audiobook Chill Ambient'
                      : 'Trending Pop Music Indonesia'
                )}
              class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
              style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
            >
              Muat ulang
            </button>
          </div>
        {:else}
          <div class="velora-card-grid-7">
            {#each displayedFilteredResults as yt (yt.videoId)}
              {@const cover = resolveCoverUrl(yt.coverUrl, mediaBaseUrl)}
              {@const isSongActive = isTrackCardActive(yt.videoId, yt.title, yt.artist)}
              <button
                type="button"
                onclick={() => (isSongActive && onTogglePlay ? onTogglePlay() : onPlayYTMusic(yt))}
                class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg border-0 outline-none focus:outline-none text-left cursor-pointer transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
              >
                <div
                  class="velora-card-artwork mb-2.5 flex items-center justify-center"
                  style={cover
                    ? 'background-color: var(--md-sys-color-surface-container-high);'
                    : 'background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));'}
                >
                  {#if cover}
                    <img
                      src={cover}
                      alt={yt.title}
                      loading="lazy"
                      decoding="async"
                      class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                    />
                  {:else}
                    <Icon name="vinyl-record" weight="fill" size={44} class="text-white drop-shadow-sm" />
                  {/if}

                  <span
                    class="absolute z-20 bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center transition-all duration-200 {isSongActive
                      ? 'opacity-100 translate-y-0'
                      : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                  >
                    {#if isSongActive && isPlaying}
                      <span class="velora-playing-eq" style="color: currentColor;" aria-hidden="true">
                        <span></span>
                        <span></span>
                        <span></span>
                        <span></span>
                      </span>
                      <span class="hidden group-hover:inline-flex items-center justify-center">
                        <Icon name="pause" weight="fill" size={18} />
                      </span>
                    {:else}
                      <Icon name="play" weight="fill" size={18} />
                    {/if}
                  </span>
                </div>

                <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                  {yt.title}
                </div>
                <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                  {yt.artist}
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </section>
    {/if}

    {#if !(activeFilterPill === 'semua' && ytHomeLoading && ytShelves.length === 0) && !(activeFilterPill !== 'semua' && ytLoading && (activeFilterPill === 'playlist' ? ytPlaylistResults.length === 0 : activeFilterPill === 'album' ? ytAlbumResults.length === 0 : activeFilterPill === 'artis' ? displayedHomeArtists.length === 0 : ytResults.length === 0))}
      <div class="pt-2 pb-6 flex justify-center">
        <button
          type="button"
          disabled={ytLoadingMore}
          aria-label={ytLoadingMore
            ? 'Memuat'
            : activeFilterPill === 'playlist'
              ? 'Muat lebih banyak playlist'
              : activeFilterPill === 'album'
                ? 'Muat lebih banyak album'
                : 'Muat lebih banyak lagu'}
          onclick={() => {
            if (activeFilterPill === 'semua') {
              expandedShelves = {
                ...expandedShelves,
                'yt-Lagu Lainnya untuk Anda': true
              };
            } else {
              filteredMoreLoaded = true;
            }
            onLoadMoreYTMusic?.(activeFilterPill);
          }}
          class="h-10 min-w-[172px] px-6 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-bold flex items-center justify-center cursor-pointer transition-all duration-200 active:scale-95 disabled:opacity-90 disabled:cursor-wait"
          style="color: var(--md-sys-color-on-surface);"
        >
          {#if ytLoadingMore}
            <span class="velora-eq-loader" aria-hidden="true">
              <span class="velora-eq-bar"></span>
              <span class="velora-eq-bar"></span>
              <span class="velora-eq-bar"></span>
              <span class="velora-eq-bar"></span>
              <span class="velora-eq-bar"></span>
            </span>
          {:else if activeFilterPill === 'playlist'}
            Muat lebih banyak playlist
          {:else if activeFilterPill === 'artis'}
            Muat lebih banyak artis
          {:else if activeFilterPill === 'album'}
            Muat lebih banyak album
          {:else}
            Muat lebih banyak lagu
          {/if}
        </button>
      </div>
    {/if}
  </div>
</div>
