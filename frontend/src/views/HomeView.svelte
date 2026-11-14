<script lang="ts">
  import { isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Album, Playlist, ViewRoute, YTMusicCard, YTMusicShelf, YTMusicTrack } from '../lib/types';
  import Icon from '../components/Icon.svelte';

  interface Props {
    albums?: Album[];
    playlists: Playlist[];
    ytShelves: YTMusicShelf[];
    ytResults: YTMusicTrack[];
    ytPlaylistResults?: YTMusicCard[];
    ytLoading: boolean;
    ytHomeLoading: boolean;
    ytLoadingMore?: boolean;
    loadingCollectionId: string | null;
    mediaBaseUrl: string;
    onPlayYTMusic: (yt: YTMusicTrack) => void;
    onPlayYTCollection: (card: YTMusicCard) => void;
    onPlayPlaylist?: (playlistId: number) => void;
    onPlayAlbum?: (albumId: number) => void;
    onSearchYTMusic: (query: string) => void;
    onSearchYTMusicPlaylists?: (query: string) => void;
    onRefreshYTMusicHome: () => void;
    onLoadMoreYTMusic?: (category: 'semua' | 'musik' | 'playlist' | 'podcast' | 'buku') => void;
    onNavigate: (route: ViewRoute) => void;
  }

  let {
    playlists,
    ytShelves,
    ytResults,
    ytPlaylistResults = [],
    ytLoading,
    ytHomeLoading,
    ytLoadingMore = false,
    loadingCollectionId,
    mediaBaseUrl,
    onPlayYTMusic,
    onPlayYTCollection,
    onPlayPlaylist,
    onSearchYTMusic,
    onSearchYTMusicPlaylists,
    onRefreshYTMusicHome,
    onLoadMoreYTMusic,
    onNavigate
  }: Props = $props();

  let activeFilterPill = $state<'semua' | 'musik' | 'playlist' | 'podcast' | 'buku'>('semua');
  let filteredMoreLoaded = $state(false);
  let expandedShelves = $state<Record<string, boolean>>({ 'yt-Lagu Lainnya untuk Anda': true });
  let stickyContainerEl: HTMLElement | undefined = $state();
  let pillsStuck = $state(false);

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

  const filterPills = [
    { id: 'semua', label: 'Semua' },
    { id: 'musik', label: 'Musik' },
    { id: 'playlist', label: 'Playlist' },
    { id: 'podcast', label: 'Podcast' },
    { id: 'buku', label: 'Buku Audio' }
  ] as const;

  const pillHeading = $derived(
    activeFilterPill === 'musik'
      ? 'Musik YouTube Music'
      : activeFilterPill === 'playlist'
        ? 'Playlist YouTube Music'
        : activeFilterPill === 'podcast'
          ? 'Podcast'
          : 'Buku Audio'
  );

  const displayedFilteredPlaylists = $derived.by(() => {
    if (ytPlaylistResults.length <= 5) return ytPlaylistResults;
    const fullRows = Math.floor(ytPlaylistResults.length / 5) * 5;
    if (!filteredMoreLoaded) {
      return ytPlaylistResults.slice(0, Math.min(20, fullRows));
    }
    return ytPlaylistResults.slice(0, fullRows);
  });

  const displayedFilteredResults = $derived.by(() => {
    if (ytResults.length <= 5) return ytResults;
    const fullRows = Math.floor(ytResults.length / 5) * 5;
    if (!filteredMoreLoaded) {
      return ytResults.slice(0, Math.min(20, fullRows));
    }
    return ytResults.slice(0, fullRows);
  });

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
    <div class="velora-sticky-pills-bar px-6 py-3 flex flex-wrap items-center gap-2">
      {#each filterPills as pill (pill.id)}
        {@const active = activeFilterPill === pill.id}
        <button
          type="button"
          onclick={() => {
            activeFilterPill = pill.id;
            filteredMoreLoaded = false;
            if (pill.id === 'semua') {
              if (ytShelves.length === 0) onRefreshYTMusicHome();
            } else if (pill.id === 'musik') {
              onSearchYTMusic('Trending Pop Music Indonesia');
            } else if (pill.id === 'playlist') {
              onSearchYTMusicPlaylists?.('Top Hits Indonesia Playlist');
            } else if (pill.id === 'podcast') {
              onSearchYTMusic('Podcast Musik Indonesia');
            } else if (pill.id === 'buku') {
              onSearchYTMusic('Audiobook Chill Ambient');
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

  <div class="relative z-10 px-6 pt-2 pb-6 space-y-7">

    {#if activeFilterPill === 'semua'}
      <!-- Saved Library Playlists Shelf (Only shown when real playlists exist) -->
      {#if playlists.length > 0}
        {@const plExpanded = !!expandedShelves['saved-playlists']}
        <section class="min-w-0 space-y-3" use:svelteMotionEnter={{ delay: 0.03 }}>
          <div class="flex items-center justify-between gap-2">
            <h2 class="text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Playlist Tersimpan
            </h2>
            {#if playlists.length > 4}
              <button
                type="button"
                onclick={() => toggleShelfExpand('saved-playlists')}
                class="text-xs font-bold hover:underline cursor-pointer"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                {plExpanded ? 'Tampilkan lebih sedikit' : 'Tampilkan semua'}
              </button>
            {/if}
          </div>

          <div
            use:shelfEdgeFade
            class={plExpanded
              ? 'velora-card-grid'
              : 'flex items-start justify-start gap-3 overflow-x-auto no-scrollbar velora-shelf-scroll pb-1'}
          >
            {#each playlists as pl (pl.id)}
              {@const cover = resolveCoverUrl(pl.coverUrl, mediaBaseUrl)}
              <div
                role="group"
                class="group w-[172px] shrink-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
              >
                <div
                  class="velora-card-artwork mb-2.5 flex items-center justify-center"
                  style="background: linear-gradient(135deg, {pl.coverColor || 'var(--md-sys-color-primary-container)'}, var(--md-sys-color-surface-container-highest));"
                >
                  <button
                    type="button"
                    onclick={() => onNavigate({ name: 'playlist-detail', playlistId: pl.id })}
                    aria-label="Buka {pl.name}"
                    class="velora-card-artwork-hit cursor-pointer"
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
                      <Icon name="vinyl-record" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.55;" />
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
            <div class="flex items-start justify-start gap-3 overflow-hidden pb-1">
              {#each [0, 1, 2, 3, 4, 5] as cardIdx (cardIdx)}
                <div class="w-[172px] shrink-0 p-2.5 rounded-lg space-y-2.5">
                  <div class="w-[152px] h-[152px] rounded-md bg-[var(--md-sys-color-surface-container-high)] animate-pulse"></div>
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
          {@const isExpanded = !!expandedShelves[shelfKey]}
          <section class="min-w-0 space-y-3" use:svelteMotionEnter={{ delay: Math.min(0.18, sIdx * 0.04) }}>
            <div class="flex items-center justify-between gap-2">
              <h2 class="text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                {shelf.title}
              </h2>
              {#if shelf.items.length > 4}
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
                ? 'velora-card-grid-5'
                : 'flex items-start justify-start gap-3 overflow-x-auto no-scrollbar velora-shelf-scroll pb-1'}
            >
              {#each shelf.items as item (item.id)}
                {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
                {@const isItemLoading = loadingCollectionId === item.id}
                <button
                  type="button"
                  onclick={() => onPlayYTCollection(item)}
                  class="group {isExpanded ? 'w-full min-w-0' : 'w-[172px] shrink-0'} flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg text-left cursor-pointer transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div class="velora-card-artwork mb-2.5 flex items-center justify-center" style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));">
                    {#if cover}
                      <img
                        src={cover}
                        alt={item.title}
                        loading="lazy"
                        decoding="async"
                        class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                      />
                    {:else}
                      <Icon name="vinyl-record" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.55;" />
                    {/if}

                    <span
                      class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center transition-all duration-200 {isItemLoading
                        ? 'opacity-100 translate-y-0'
                        : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                    >
                      <Icon
                        name={isItemLoading ? 'spinner' : 'play'}
                        weight={isItemLoading ? 'bold' : 'fill'}
                        size={18}
                        class={isItemLoading ? 'animate-spin' : ''}
                      />
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
          <div class="velora-card-grid-5">
            {#each Array.from({ length: 20 }, (_, idx) => idx) as idx (idx)}
              <div class="w-full min-w-0 p-2.5 rounded-lg space-y-2.5">
                <div class="w-full aspect-square rounded-md bg-[var(--md-sys-color-surface-container-high)] animate-pulse"></div>
                <div class="h-3.5 w-3/4 rounded bg-[var(--md-sys-color-surface-container-highest)] animate-pulse"></div>
                <div class="h-3 w-1/2 rounded bg-[var(--md-sys-color-surface-container)] animate-pulse"></div>
              </div>
            {/each}
          </div>
        {:else if activeFilterPill === 'playlist'}
          {#if ytPlaylistResults.length === 0}
            <div class="p-8 rounded-lg text-center space-y-2" style="background-color: var(--md-sys-color-surface-container-low);">
              <Icon name="playlist" weight="fill" size={36} class="mx-auto opacity-50" />
              <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
                Belum ada playlist dari YouTube Music untuk kategori ini.
              </p>
            </div>
          {:else}
            <div class="velora-card-grid-5">
              {#each displayedFilteredPlaylists as item (item.id)}
                {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
                {@const isItemLoading = loadingCollectionId === item.id}
                <button
                  type="button"
                  onclick={() => onPlayYTCollection(item)}
                  class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg text-left cursor-pointer transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div class="velora-card-artwork mb-2.5 flex items-center justify-center" style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));">
                    {#if cover}
                      <img
                        src={cover}
                        alt={item.title}
                        loading="lazy"
                        decoding="async"
                        class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                      />
                    {:else}
                      <Icon name="playlist" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.55;" />
                    {/if}

                    <span
                      class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center transition-all duration-200 {isItemLoading
                        ? 'opacity-100 translate-y-0'
                        : 'opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0'}"
                    >
                      <Icon
                        name={isItemLoading ? 'spinner' : 'play'}
                        weight={isItemLoading ? 'bold' : 'fill'}
                        size={18}
                        class={isItemLoading ? 'animate-spin' : ''}
                      />
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
          <div class="p-8 rounded-lg text-center space-y-2" style="background-color: var(--md-sys-color-surface-container-low);">
            <Icon name="music-notes" weight="fill" size={36} class="mx-auto opacity-50" />
            <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
              Belum ada hasil dari YouTube Music untuk kategori ini.
            </p>
          </div>
        {:else}
          <div class="velora-card-grid-5">
            {#each displayedFilteredResults as yt (yt.videoId)}
              {@const cover = resolveCoverUrl(yt.coverUrl, mediaBaseUrl)}
              <button
                type="button"
                onclick={() => onPlayYTMusic(yt)}
                class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg text-left cursor-pointer transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]"
              >
                <div class="velora-card-artwork mb-2.5 flex items-center justify-center" style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));">
                  {#if cover}
                    <img
                      src={cover}
                      alt={yt.title}
                      loading="lazy"
                      decoding="async"
                      class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                    />
                  {:else}
                    <Icon name="vinyl-record" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.55;" />
                  {/if}

                  <span
                    class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200"
                  >
                    <Icon name="play" weight="fill" size={18} />
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

    {#if !(activeFilterPill === 'semua' && ytHomeLoading && ytShelves.length === 0) && !(activeFilterPill !== 'semua' && ytLoading && (activeFilterPill === 'playlist' ? ytPlaylistResults.length === 0 : ytResults.length === 0))}
      <div class="pt-2 pb-6 flex justify-center">
        <button
          type="button"
          disabled={ytLoadingMore}
          aria-label={ytLoadingMore ? 'Memuat' : activeFilterPill === 'playlist' ? 'Muat lebih banyak playlist' : 'Muat lebih banyak lagu'}
          onclick={() => {
            if (activeFilterPill !== 'semua') {
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
          {:else}
            Muat lebih banyak lagu
          {/if}
        </button>
      </div>
    {/if}
  </div>
</div>
