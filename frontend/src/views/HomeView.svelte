<script lang="ts">
  import { untrack } from 'svelte';
  import { isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Album, Artist, HomeFilter, Playlist, Track, ViewRoute, YTMusicCard, YTMusicShelf, YTMusicTrack } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
  import CoverImage from '../components/CoverImage.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import EqIndicator from '../components/EqIndicator.svelte';
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
    queueSourceKey?: string | null;
    isPlaying?: boolean;
    trackLoading?: boolean;
    onTogglePlay?: () => void;
    mediaBaseUrl: string;
    onPlayYTMusic: (yt: YTMusicTrack) => void;
    onOpenYTSong?: (yt: YTMusicTrack) => void;
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
    queueSourceKey = null,
    isPlaying = false,
    trackLoading = false,
    onTogglePlay,
    mediaBaseUrl,
    onPlayYTMusic,
    onOpenYTSong,
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

  const isActuallyPlaying = $derived(isPlaying && !trackLoading);

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
    { id: 'album', label: tr(lang, 'nav.albums') },
    { id: 'playlist', label: tr(lang, 'nav.playlists') }
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
              ? tr(lang, 'nav.podcasts')
              : tr(lang, 'nav.audiobooks')
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

  const shelfArtists = $derived.by(() => {
    const out: YTMusicCard[] = [];
    const seen = new Set<string>();
    for (const sh of ytShelves) {
      for (const it of sh.items) {
        if (it.kind !== 'artist') continue;
        const k = (it.browseId || it.title).trim().toLowerCase();
        if (!k || seen.has(k)) continue;
        seen.add(k);
        out.push(it);
      }
    }
    return out;
  });

  const shelfAlbums = $derived.by(() => {
    const out: YTMusicCard[] = [];
    const seen = new Set<string>();
    for (const sh of ytShelves) {
      for (const it of sh.items) {
        if (it.kind !== 'album') continue;
        const k = (it.browseId || it.id).trim().toLowerCase();
        if (!k || seen.has(k)) continue;
        seen.add(k);
        out.push(it);
      }
    }
    return out;
  });

  const shelfPlaylists = $derived.by(() => {
    const out: YTMusicCard[] = [];
    const seen = new Set<string>();
    for (const sh of ytShelves) {
      for (const it of sh.items) {
        if (it.kind !== 'playlist') continue;
        const k = (it.browseId || it.playlistId || it.id).trim().toLowerCase();
        if (!k || seen.has(k)) continue;
        seen.add(k);
        out.push(it);
      }
    }
    return out;
  });

  const displayedHomeArtists = $derived.by<HomeArtistCard[]>(() => {
    const dbByName = new Map<string, Artist>();
    for (const ar of artists) {
      const k = ar.name.trim().toLowerCase();
      if (!k || k === 'unknown artist' || k === 'youtube music artist') continue;
      if (!dbByName.has(k)) dbByName.set(k, ar);
    }

    const byName = new Map<string, HomeArtistCard>();
    const combinedYT = [...shelfArtists, ...ytArtistResults];
    if (combinedYT.length > 0) {
      for (const ytAr of combinedYT) {
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
          subtitle: tr(lang, 'nav.artist'),
          rawAvatar: ytAr.coverUrl || dbMatch?.avatarUrl || '',
          isFavorite: dbMatch ? dbMatch.isFavorite : false,
          trackCount: dbMatch ? dbMatch.trackCount : 0
        });
      }
      const list = Array.from(byName.values());
      if (!filteredMoreLoaded && list.length > 24) {
        return list.slice(0, 24);
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
          subtitle: tr(lang, 'nav.artist'),
          rawAvatar: ar.avatarUrl,
          isFavorite: ar.isFavorite,
          trackCount: ar.trackCount
        });
      }
    }
    const list = Array.from(byName.values());
    if (!filteredMoreLoaded && list.length > 24) {
      return list.slice(0, 24);
    }
    return list;
  });

  const displayedFilteredAlbums = $derived.by(() => {
    const merged: YTMusicCard[] = [];
    const seen = new Set<string>();
    for (const a of [...shelfAlbums, ...ytAlbumResults]) {
      const k = (a.browseId || a.id).trim().toLowerCase();
      if (!k || seen.has(k)) continue;
      seen.add(k);
      merged.push(a);
    }
    if (!filteredMoreLoaded && merged.length > 24) return merged.slice(0, 24);
    return merged;
  });

  const displayedFilteredPlaylists = $derived.by(() => {
    const merged: YTMusicCard[] = [];
    const seen = new Set<string>();
    for (const p of [...shelfPlaylists, ...ytPlaylistResults]) {
      const k = (p.browseId || p.playlistId || p.id).trim().toLowerCase();
      if (!k || seen.has(k)) continue;
      seen.add(k);
      merged.push(p);
    }
    if (!filteredMoreLoaded && merged.length > 24) return merged.slice(0, 24);
    return merged;
  });

  const displayedFilteredResults = $derived.by(() => {
    const seenVideo = new Set<string>();
    const deduped: YTMusicTrack[] = [];
    for (const r of ytResults) {
      if (!r.videoId || seenVideo.has(r.videoId)) continue;
      seenVideo.add(r.videoId);
      deduped.push(r);
    }
    if (!filteredMoreLoaded && deduped.length > 24) {
      return deduped.slice(0, 24);
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
    const itemKey = (item.browseId || item.playlistId || item.id || '').trim().toLowerCase();
    const activeKey = (queueSourceKey || '').trim().toLowerCase();
    if (activeKey) {
      return Boolean(itemKey && activeKey === itemKey);
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

  function localizeShelfTitle(title: string): string {
    if (title === 'Album Populer' || title === 'Popular Albums') return tr(lang, 'nav.shelfPopularAlbums');
    if (title === 'Playlist Pilihan' || title === 'Featured Playlists') return tr(lang, 'nav.shelfFeaturedPlaylists');
    if (title === 'Artis Populer' || title === 'Popular Artists') return tr(lang, 'nav.shelfPopularArtists');
    if (title === 'Lagu Lainnya untuk Anda' || title === 'More Songs for You') return tr(lang, 'nav.shelfMoreSongs');
    return title;
  }

  function localizeItemSubtitle(item: YTMusicCard): string {
    if (item.kind === 'artist' && (!item.subtitle || item.subtitle === 'Artis' || item.subtitle === 'Artist')) {
      return tr(lang, 'nav.artist');
    }
    return item.subtitle || 'YouTube Music';
  }

  function shelfEdgeFade(node: HTMLElement) {
    const update = () => {
      const hasOverflow = node.scrollWidth > node.clientWidth + 4;
      const atStart = node.scrollLeft <= 4;
      const atEnd = node.scrollLeft + node.clientWidth >= node.scrollWidth - 4;
      node.dataset.fadeStart = hasOverflow && !atStart ? 'true' : 'false';
      node.dataset.fadeEnd = hasOverflow && !atEnd ? 'true' : 'false';
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

<div class="relative min-h-full" style="overflow-anchor: none;">
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
            } else if (pill.id === 'playlist') {
              onSearchYTMusicPlaylists?.('');
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
      {#if ytHomeLoading}
        {#each [0, 1, 2] as shelfIdx (shelfIdx)}
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
      {:else}
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
                      <CoverImage
                        src={cover}
                        fallbackIcon="playlist"
                        iconSize={44}
                        isVideoThumb={isYTVideoThumb(cover)}
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
                    {pl.description || `${pl.trackCount} ${tr(lang, 'nav.songs').toLowerCase()}`}
                  </div>
                </button>
              </div>
            {/each}
          </div>
        </section>
      {/if}

      {#if ytShelves.length === 0}
        <div class="min-h-[55vh] flex flex-col items-center justify-center text-center gap-3 px-4">
          <Icon name="wifi-slash" size={40} class="opacity-60" />
          <div class="space-y-1 max-w-md">
            <h2 class="text-base font-bold" style="color: var(--md-sys-color-on-surface);">
              {tr(lang, 'nav.homeLoadFailedTitle')}
            </h2>
            <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
              {tr(lang, 'nav.homeLoadFailedDesc')}
            </p>
          </div>
          <button
            type="button"
            onclick={onRefreshYTMusicHome}
            class="h-9 px-5 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold cursor-pointer hover:brightness-105 transition"
          >
            {tr(lang, 'nav.refresh')}
          </button>
        </div>
      {:else}
        {#each ytShelves as shelf, sIdx (shelf.title)}
          {@const shelfKey = `yt-${shelf.title}`}
          {@const isAlwaysFull = shelf.title === 'Lagu Lainnya untuk Anda'}
          {@const isExpanded = isAlwaysFull || !!expandedShelves[shelfKey]}
          <section class="min-w-0 space-y-3" use:svelteMotionEnter={{ delay: Math.min(0.18, sIdx * 0.04) }}>
            <div class="flex items-center justify-between gap-2">
              <h2 class="text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                {localizeShelfTitle(shelf.title)}
              </h2>
              {#if !isAlwaysFull && shelf.items.length > 4}
                <button
                  type="button"
                  onclick={() => toggleShelfExpand(shelfKey)}
                  class="text-xs font-bold hover:underline cursor-pointer"
                  style="color: var(--md-sys-color-on-surface-variant);"
                >
                  {isExpanded ? tr(lang, 'nav.showLess') : tr(lang, 'nav.showAll')}
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
                <div
                  role="group"
                  class="group {isExpanded ? 'w-full min-w-0' : 'w-[132px] sm:w-[172px] shrink-0'} flex flex-col justify-start items-stretch self-start p-2 sm:p-2.5 rounded-lg border-0 outline-none focus:outline-none text-left transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div class="relative mb-2.5">
                    <div
                      class="velora-card-artwork flex items-center justify-center {item.kind === 'artist' ? 'velora-artist-avatar !rounded-full' : ''}"
                      style={cover
                        ? 'background-color: var(--md-sys-color-surface-container-high);'
                        : 'background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));'}
                    >
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
                          } else if (onOpenYTAlbum) {
                            onOpenYTAlbum(item);
                          } else {
                            onPlayYTCollection(item);
                          }
                        }}
                        aria-label="Buka {item.title}"
                        class="velora-card-artwork-hit cursor-pointer"
                      >
                        <CoverImage
                          src={cover}
                          fallbackIcon={item.kind === 'artist' ? 'user' : item.kind === 'playlist' ? 'playlist' : 'vinyl-record'}
                          iconSize={42}
                          isVideoThumb={isYTVideoThumb(cover)}
                        />
                      </button>
                    </div>

                    {#if item.kind !== 'artist'}
                      <button
                        type="button"
                        onclick={(e) => {
                          e.stopPropagation();
                          if (isCardActive && onTogglePlay) {
                            onTogglePlay();
                          } else {
                            onPlayYTCollection(item);
                          }
                        }}
                        aria-label="Putar {item.title}"
                        class="absolute z-20 bottom-1.5 right-1.5 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer transition-all duration-200 {isCardActive
                          ? 'opacity-100 translate-y-0'
                          : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                      >
                        {#if isCardActive && isActuallyPlaying}
                          <EqIndicator color="currentColor" />
                          <span class="hidden group-hover:inline-flex items-center justify-center">
                            <Icon name="pause" weight="fill" size={18} />
                          </span>
                        {:else}
                          <Icon name="play" weight="fill" size={18} />
                        {/if}
                      </button>
                    {/if}
                  </div>

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
                      } else if (onOpenYTAlbum) {
                        onOpenYTAlbum(item);
                      } else {
                        onPlayYTCollection(item);
                      }
                    }}
                    class="w-full text-left cursor-pointer"
                  >
                    <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                      {item.title}
                    </div>
                    <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      {localizeItemSubtitle(item)}
                    </div>
                  </button>
                </div>
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
                  {ytLoadingMorePopularArtists ? tr(lang, 'nav.loading') : tr(lang, 'nav.loadMoreArtists')}
                </button>
              </div>
            {/if}
          </section>
        {/each}
      {/if}
      {/if}
    {:else}
      <section class="min-w-0 space-y-4" use:svelteMotionEnter={{ delay: 0.04 }}>
        <h2 class="text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
          {pillHeading}
        </h2>

        {#if ytLoading}
          <div class="velora-card-grid-7">
            {#each Array.from({ length: 24 }, (_, idx) => idx) as idx (idx)}
              <div class="w-full min-w-0 p-2.5 rounded-lg space-y-2.5">
                <div class="w-full aspect-square {activeFilterPill === 'artis' ? 'rounded-full' : 'rounded-md'} bg-[var(--md-sys-color-surface-container-high)] animate-pulse"></div>
                <div class="h-3.5 w-3/4 rounded bg-[var(--md-sys-color-surface-container-highest)] animate-pulse"></div>
                <div class="h-3 w-1/2 rounded bg-[var(--md-sys-color-surface-container)] animate-pulse"></div>
              </div>
            {/each}
          </div>
        {:else if activeFilterPill === 'artis'}
          {#if displayedHomeArtists.length === 0}
            <EmptyState
              icon="users"
              title={tr(lang, 'nav.noArtistsTitle')}
              description={tr(lang, 'nav.noArtistsDesc')}
              class="flex flex-col items-center justify-center text-center py-14 px-4"
            >
              <button
                type="button"
                onclick={() => onSearchYTMusicArtists?.('')}
                class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
                style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
              >
                {tr(lang, 'nav.refresh')}
              </button>
            </EmptyState>
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
                        <CoverImage
                          src={avatar}
                          fallbackIcon="user"
                          iconSize={42}
                        />
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
                      title={ar.isFavorite ? tr(lang, 'nav.removeFromFavorites') : tr(lang, 'nav.addToFavorites')}
                      aria-label={ar.isFavorite ? tr(lang, 'nav.removeFromFavorites') : tr(lang, 'nav.addToFavorites')}
                      class="velora-heart-btn border-0 shadow-md absolute z-20 bottom-1 right-1 w-8 h-8 rounded-full backdrop-blur-xs flex items-center justify-center cursor-pointer {ar.isFavorite
                        ? 'is-loved opacity-100'
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
          {#if displayedFilteredAlbums.length === 0}
            <EmptyState
              icon="vinyl-record"
              title={tr(lang, 'nav.noAlbumsTitle')}
              description={tr(lang, 'nav.noAlbumsDesc')}
              class="flex flex-col items-center justify-center text-center py-14 px-4"
            >
              <button
                type="button"
                onclick={() => onSearchYTMusicAlbums?.('')}
                class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
                style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
              >
                {tr(lang, 'nav.refresh')}
              </button>
            </EmptyState>
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
                      <CoverImage
                        src={cover}
                        fallbackIcon="vinyl-record"
                        iconSize={42}
                        isVideoThumb={isYTVideoThumb(cover)}
                      />
                    </button>

                    <button
                      type="button"
                      onclick={() => (isAlbumActive && onTogglePlay ? onTogglePlay() : onPlayYTCollection(item))}
                      aria-label="Putar album {item.title}"
                      class="absolute z-20 bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer transition-all duration-200 {isAlbumActive
                        ? 'opacity-100 translate-y-0'
                        : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                    >
                      {#if isAlbumActive && isActuallyPlaying}
                        <EqIndicator color="currentColor" />
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
                      {item.subtitle || 'Album'}
                    </div>
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        {:else if activeFilterPill === 'playlist'}
          {#if displayedFilteredPlaylists.length === 0}
            <EmptyState
              icon="playlist"
              title={tr(lang, 'nav.noPlaylists')}
              description={tr(lang, 'nav.noPlaylistsCategoryDesc')}
              class="flex flex-col items-center justify-center text-center py-14 px-4"
            >
              <button
                type="button"
                onclick={() => onSearchYTMusicPlaylists?.('')}
                class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
                style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
              >
                {tr(lang, 'nav.refresh')}
              </button>
            </EmptyState>
          {:else}
            <div class="velora-card-grid-7">
              {#each displayedFilteredPlaylists as item (item.id)}
                {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
                {@const isPlaylistActive = isCollectionCardActive(item)}
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
                      aria-label="Buka playlist {item.title}"
                      class="velora-card-artwork-hit cursor-pointer"
                    >
                      <CoverImage
                        src={cover}
                        fallbackIcon="playlist"
                        iconSize={42}
                        isVideoThumb={isYTVideoThumb(cover)}
                      />
                    </button>

                    <button
                      type="button"
                      onclick={(e) => {
                        e.stopPropagation();
                        if (isPlaylistActive && onTogglePlay) {
                          onTogglePlay();
                        } else {
                          onPlayYTCollection(item);
                        }
                      }}
                      aria-label="Putar playlist {item.title}"
                      class="absolute z-20 bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer transition-all duration-200 {isPlaylistActive
                        ? 'opacity-100 translate-y-0'
                        : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                    >
                      {#if isPlaylistActive && isActuallyPlaying}
                        <EqIndicator color="currentColor" />
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
                      {item.subtitle || 'Playlist'}
                    </div>
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        {:else if ytResults.length === 0}
          <EmptyState
            icon="music-note"
            title={tr(lang, 'nav.noSongsTitle')}
            description={tr(lang, 'nav.noSongsDesc')}
            class="flex flex-col items-center justify-center text-center py-14 px-4"
          >
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
              {tr(lang, 'nav.refresh')}
            </button>
          </EmptyState>
        {:else}
          <div class="velora-card-grid-7">
            {#each displayedFilteredResults as yt (yt.videoId)}
              {@const cover = resolveCoverUrl(yt.coverUrl, mediaBaseUrl)}
              {@const isSongActive = isTrackCardActive(yt.videoId, yt.title, yt.artist)}
              <div
                role="group"
                class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg border-0 outline-none focus:outline-none text-left transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
              >
                <div
                  class="velora-card-artwork mb-2.5 flex items-center justify-center"
                  style={cover
                    ? 'background-color: var(--md-sys-color-surface-container-high);'
                    : 'background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));'}
                >
                  <button
                    type="button"
                    onclick={() => (onOpenYTSong ? onOpenYTSong(yt) : onPlayYTMusic(yt))}
                    aria-label="Buka album {yt.title}"
                    class="velora-card-artwork-hit cursor-pointer"
                  >
                    <CoverImage
                      src={cover}
                      fallbackIcon="music-note"
                      iconSize={42}
                      isVideoThumb={isYTVideoThumb(cover)}
                    />
                  </button>

                  <button
                    type="button"
                    onclick={(e) => {
                      e.stopPropagation();
                      if (isSongActive && onTogglePlay) {
                        onTogglePlay();
                      } else {
                        onPlayYTMusic(yt);
                      }
                    }}
                    aria-label="Putar {yt.title}"
                    class="absolute z-20 bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer transition-all duration-200 {isSongActive
                      ? 'opacity-100 translate-y-0'
                      : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                  >
                    {#if isSongActive && isActuallyPlaying}
                      <EqIndicator color="currentColor" />
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
                  onclick={() => (onOpenYTSong ? onOpenYTSong(yt) : onPlayYTMusic(yt))}
                  class="w-full text-left cursor-pointer"
                >
                  <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                    {yt.title}
                  </div>
                  <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    {yt.artist}
                  </div>
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    {/if}

    {#if !(activeFilterPill === 'semua' && (ytHomeLoading || ytShelves.length === 0)) && !(activeFilterPill !== 'semua' && (activeFilterPill === 'playlist' ? ytPlaylistResults.length === 0 : activeFilterPill === 'album' ? ytAlbumResults.length === 0 : activeFilterPill === 'artis' ? displayedHomeArtists.length === 0 : ytResults.length === 0))}
      <div class="pt-2 pb-6 flex justify-center">
        <button
          type="button"
          disabled={ytLoadingMore}
          aria-label={ytLoadingMore
            ? tr(lang, 'nav.loading')
            : activeFilterPill === 'playlist'
              ? tr(lang, 'nav.loadMorePlaylists')
              : activeFilterPill === 'album'
                ? tr(lang, 'nav.loadMoreAlbums')
                : tr(lang, 'nav.loadMoreSongs')}
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
            <EqIndicator variant="loader" />
          {:else if activeFilterPill === 'playlist'}
            {tr(lang, 'nav.loadMorePlaylists')}
          {:else if activeFilterPill === 'artis'}
            {tr(lang, 'nav.loadMoreArtists')}
          {:else if activeFilterPill === 'album'}
            {tr(lang, 'nav.loadMoreAlbums')}
          {:else}
            {tr(lang, 'nav.loadMoreSongs')}
          {/if}
        </button>
      </div>
    {/if}
  </div>
</div>
