<script lang="ts">
  import { extractDominantColor, formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Album, Artist, Playlist, Track, ViewRoute, YTMusicTrack } from '../lib/types';
  import Icon from '../components/Icon.svelte';
  import TrackTable from '../components/TrackTable.svelte';

  interface Props {
    mode:
      | 'search'
      | 'liked-songs'
      | 'favorite-albums'
      | 'favorite-artists'
      | 'artist-detail';
    searchQuery: string;
    tracks: Track[];
    albums: Album[];
    artists: Artist[];
    selectedArtist?: Artist | null;
    detailTracks?: Track[];
    ytResults: YTMusicTrack[];
    ytLoading: boolean;
    playlists: Playlist[];
    currentTrackId: number | null;
    currentExternalId?: string | null;
    currentTrackTitle?: string | null;
    currentTrackArtist?: string | null;
    isPlaying: boolean;
    mediaBaseUrl: string;
    accentColor: string;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onTogglePlay?: () => void;
    onPlayAlbum?: (albumId: number) => void;
    onPlayArtist?: (artistId: number) => void;
    onPlayYTMusic: (yt: YTMusicTrack) => void;
    onSaveYTMusic: (yt: YTMusicTrack) => void;
    onSearchYTMusic: (q: string) => void;
    onToggleFavoriteTrack: (id: number) => void;
    onToggleFavoriteAlbum: (id: number) => void;
    onToggleFavoriteArtist: (id: number) => void;
    onAddToPlaylist: (playlistId: number, trackId: number) => void;
    onOpenAlbum: (albumId: number) => void;
    onOpenArtist: (artistId: number) => void;
    onOpenYTMusicAlbum?: (yt: YTMusicTrack) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onNavigate?: (route: ViewRoute) => void;
  }

  let {
    mode,
    searchQuery,
    tracks,
    albums,
    artists,
    selectedArtist = null,
    detailTracks = [],
    ytResults,
    ytLoading,
    playlists,
    currentTrackId,
    currentExternalId = null,
    currentTrackTitle = null,
    currentTrackArtist = null,
    isPlaying,
    mediaBaseUrl,
    accentColor,
    onPlayTrack,
    onTogglePlay,
    onPlayAlbum,
    onPlayArtist,
    onPlayYTMusic,
    onSaveYTMusic,
    onSearchYTMusic,
    onToggleFavoriteTrack,
    onToggleFavoriteAlbum,
    onToggleFavoriteArtist,
    onAddToPlaylist,
    onOpenAlbum,
    onOpenArtist,
    onOpenYTMusicAlbum,
    onOpenTrackAlbum,
    onNavigate
  }: Props = $props();

  let localFilterText = $state('');
  let albumFilterText = $state('');
  let artistFilterText = $state('');
  let ytQuery = $state('');
  let showAllArtists = $state(false);
  let albumsScrolled = $state(false);
  let heroDominantColor = $state<string | null>(null);

  $effect(() => {
    if (mode === 'search' && searchQuery) {
      ytQuery = searchQuery;
    }
  });

  const likedTracks = $derived(
    tracks.filter(
      (t) =>
        t.isFavorite &&
        (!localFilterText.trim() ||
          t.title.toLowerCase().includes(localFilterText.toLowerCase()) ||
          t.artistName.toLowerCase().includes(localFilterText.toLowerCase()))
    )
  );

  const searchMatchedLocal = $derived(
    tracks.filter((t) => {
      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase();
      return (
        t.title.toLowerCase().includes(q) ||
        t.artistName.toLowerCase().includes(q) ||
        t.albumTitle.toLowerCase().includes(q)
      );
    })
  );

  const topYTMatch = $derived.by(() => {
    if (ytResults.length === 0) return null;
    const q = (searchQuery || ytQuery).trim().toLowerCase();
    if (!q) return ytResults[0];
    let best = ytResults[0];
    let bestScore = -1;
    for (let i = 0; i < ytResults.length; i++) {
      const item = ytResults[i];
      const title = item.title.toLowerCase();
      const artist = (item.artist || '').toLowerCase();
      let score = 0;
      if (title === q) score += 100;
      else if (title.startsWith(q)) score += 60;
      else if (title.includes(q)) score += 30;
      if (artist.includes(q)) score += 15;
      // Preserve YouTube relevance order on ties
      score -= i * 0.1;
      if (score > bestScore) {
        bestScore = score;
        best = item;
      }
    }
    return best;
  });

  const otherYTResults = $derived(
    topYTMatch ? ytResults.filter((t) => t.videoId !== topYTMatch.videoId) : ytResults
  );

  const favoriteAlbumsCount = $derived(albums.filter((a) => a.isFavorite).length);

  const displayedAlbums = $derived.by(() => {
    const base = albums.filter((a) => a.isFavorite);
    const q = albumFilterText.trim().toLowerCase();
    if (!q) return base;
    return base.filter((a) => a.title.toLowerCase().includes(q) || a.artistName.toLowerCase().includes(q));
  });

  const heroAlbumCovers = $derived.by(() => {
    const source = displayedAlbums.length > 0 ? displayedAlbums : albums.filter((a) => a.isFavorite);
    const covers: string[] = [];
    for (const al of source) {
      const c = resolveCoverUrl(al.coverUrl, mediaBaseUrl);
      if (c && !covers.includes(c)) {
        covers.push(c);
        if (covers.length >= 4) break;
      }
    }
    return covers;
  });

  $effect(() => {
    const topCover = heroAlbumCovers[0];
    if (!topCover) {
      heroDominantColor = null;
      return;
    }
    let cancelled = false;
    extractDominantColor(topCover).then((color) => {
      if (!cancelled) {
        heroDominantColor = color;
      }
    });
    return () => {
      cancelled = true;
    };
  });

  const displayedArtists = $derived.by(() => {
    const base = showAllArtists ? artists : artists.filter((a) => a.isFavorite);
    const q = artistFilterText.trim().toLowerCase();
    if (!q) return base;
    return base.filter((a) => a.name.toLowerCase().includes(q));
  });

  function playShuffled(list: Track[]) {
    if (list.length === 0) return;
    const copy = [...list];
    for (let i = copy.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [copy[i], copy[j]] = [copy[j], copy[i]];
    }
    onPlayTrack(copy[0], copy);
  }

  function normText(s: string): string {
    return (s || '')
      .toLowerCase()
      .replace(/\s*[\(\[](?:official|audio|video|mv|lyrics?|hd|4k).*?[\)\]]/gi, '')
      .replace(/\s+/g, ' ')
      .trim();
  }

  function isYTActive(yt: YTMusicTrack): boolean {
    if (yt.videoId && currentExternalId && yt.videoId === currentExternalId) return true;
    if (currentTrackTitle && normText(yt.title) === normText(currentTrackTitle)) {
      if (!currentTrackArtist || !yt.artist) return true;
      const a1 = normText(yt.artist);
      const a2 = normText(currentTrackArtist);
      return a1 === a2 || a1.includes(a2) || a2.includes(a1);
    }
    return false;
  }
</script>

{#if mode === 'liked-songs'}
  <div class="relative min-h-full flex flex-col" use:svelteMotionEnter={{ y: 8 }}>
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-60"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary-container) 60%, transparent) 0%, transparent 100%);"
    ></div>

    <div class="relative z-10 p-6 flex-1 flex flex-col gap-6">
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg flex items-center justify-center shrink-0 shadow-lg"
            style="background: linear-gradient(135deg, var(--md-sys-color-tertiary-container), var(--md-sys-color-surface-container-highest));"
          >
            <Icon name="heart" weight="fill" size={44} style="color: var(--md-sys-color-on-tertiary-container);" />
          </div>
          <div class="space-y-1.5 min-w-[200px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-tertiary)]">
              Favorites • Music
            </div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Favorite music
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {likedTracks.length} lagu disukai • Local & YouTube Music
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2.5">
          <div class="relative w-64">
            <span
              class="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
              style="color: var(--md-sys-color-on-surface-variant);"
            >
              <Icon name="magnifying-glass" size={14} />
            </span>
            <input
              type="search"
              bind:value={localFilterText}
              placeholder="Filter favorite music..."
              class="w-full h-9 pl-9 pr-4 rounded-full text-xs outline-none bg-[var(--md-sys-color-surface-container-high)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
            />
          </div>

          <button
            type="button"
            disabled={likedTracks.length === 0}
            onclick={() => likedTracks[0] && onPlayTrack(likedTracks[0], likedTracks)}
            class="h-9 px-5 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold flex items-center gap-1.5 cursor-pointer hover:brightness-105 disabled:opacity-40"
          >
            <Icon name="play" weight="fill" size={15} /> Play All
          </button>

          <button
            type="button"
            disabled={likedTracks.length === 0}
            onclick={() => playShuffled(likedTracks)}
            class="h-9 px-4 rounded-full text-xs font-semibold flex items-center gap-1.5 cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] disabled:opacity-40"
            style="color: var(--md-sys-color-on-surface);"
          >
            <Icon name="shuffle" size={15} /> Shuffle
          </button>
        </div>
      </div>

      {#if likedTracks.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div
            class="w-14 h-14 mb-4 rounded-2xl flex items-center justify-center"
            style={localFilterText.trim()
              ? 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);'
              : 'background-color: var(--md-sys-color-tertiary-container); color: var(--md-sys-color-on-tertiary-container);'}
          >
            <Icon name={localFilterText.trim() ? 'magnifying-glass' : 'heart'} weight="fill" size={26} />
          </div>
          <div class="space-y-1.5">
            <p class="text-base font-bold" style="color: var(--md-sys-color-on-surface);">
              {localFilterText.trim() ? 'Lagu tidak ditemukan' : 'Belum ada lagu favorit'}
            </p>
            <p class="text-xs max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
              {localFilterText.trim()
                ? `Tidak ada lagu yang cocok dengan "${localFilterText.trim()}".`
                : 'Klik ikon hati pada lagu untuk menyimpannya di sini.'}
            </p>
          </div>
          <button
            type="button"
            onclick={() => (localFilterText.trim() ? (localFilterText = '') : onNavigate?.({ name: 'home', filter: 'musik' }))}
            class="mt-5 h-9 px-5 rounded-full text-xs font-bold flex items-center gap-2 cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
          >
            <Icon name={localFilterText.trim() ? 'x' : 'house'} weight="fill" size={14} />
            <span>{localFilterText.trim() ? 'Reset Filter' : 'Jelajahi Lagu'}</span>
          </button>
        </div>
      {:else}
        <TrackTable
          tracks={likedTracks}
          {currentTrackId}
          {currentExternalId}
          {currentTrackTitle}
          {currentTrackArtist}
          {isPlaying}
          {onTogglePlay}
          {mediaBaseUrl}
          {accentColor}
          {playlists}
          {onPlayTrack}
          onToggleFavorite={onToggleFavoriteTrack}
          {onAddToPlaylist}
        />
      {/if}
    </div>
  </div>
{:else if mode === 'favorite-albums'}
  <div
    onscroll={(e) => (albumsScrolled = (e.currentTarget as HTMLElement).scrollTop > 8)}
    class="relative h-full w-full overflow-y-auto rounded-xl select-none"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <!-- Top Gradient Fade saat scroll -->
    <div
      class="pointer-events-none sticky top-0 left-0 right-0 h-16 -mb-16 z-30 transition-opacity duration-300"
      style="opacity: {albumsScrolled ? 1 : 0}; background: linear-gradient(to bottom, var(--md-sys-color-surface) 0%, var(--md-sys-color-surface) 38%, color-mix(in srgb, var(--md-sys-color-surface) 82%, transparent) 58%, color-mix(in srgb, var(--md-sys-color-surface) 42%, transparent) 80%, transparent 100%);"
    ></div>

    <!-- Ambient Album Backdrop (dinamis dari warna gambar album teratas) -->
    <div class="pointer-events-none absolute inset-x-0 top-0 h-64 overflow-hidden">
      {#if heroAlbumCovers.length > 0}
        <img
          src={heroAlbumCovers[0]}
          alt=""
          aria-hidden="true"
          class="w-full h-full object-cover scale-125 blur-3xl opacity-35"
        />
      {/if}
      <div
        class="absolute inset-0 transition-colors duration-500"
        style="background: linear-gradient(180deg, {heroDominantColor
          ? `color-mix(in srgb, ${heroDominantColor} 45%, rgba(11, 12, 14, 0.3))`
          : heroAlbumCovers.length > 0
            ? 'rgba(11, 12, 14, 0.35)'
            : 'color-mix(in srgb, var(--md-sys-color-primary-container) 55%, rgba(11, 12, 14, 0.25))'} 0%, var(--md-sys-color-surface-container-low) 100%);"
      ></div>
    </div>

    <div class="relative z-10 p-6 min-h-full flex flex-col gap-6">
      <!-- Header Seimbang (ukuran artwork w-28 h-28 sama seperti Favorite tracks) -->
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg overflow-hidden shadow-lg shrink-0 flex items-center justify-center"
            style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
          >
            {#if heroAlbumCovers.length >= 4}
              <div class="grid grid-cols-2 grid-rows-2 w-full h-full">
                {#each heroAlbumCovers.slice(0, 4) as cov, idx (idx)}
                  <img src={cov} alt="" class="w-full h-full object-cover" />
                {/each}
              </div>
            {:else if heroAlbumCovers.length >= 1}
              <img
                src={heroAlbumCovers[0]}
                alt="Favorite albums"
                class="block w-full h-full object-cover object-center"
                class:velora-yt-video-thumb={isYTVideoThumb(heroAlbumCovers[0])}
              />
            {:else}
              <Icon name="vinyl-record" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container);" />
            {/if}
          </div>

          <div class="space-y-1.5 min-w-[200px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
              Favorites • Albums
            </div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Favorite albums
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {favoriteAlbumsCount} album favorit • Local & YouTube Music
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2.5">
          <div class="relative w-60">
            <span
              class="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
              style="color: var(--md-sys-color-on-surface-variant);"
            >
              <Icon name="magnifying-glass" size={14} />
            </span>
            <input
              type="search"
              bind:value={albumFilterText}
              placeholder="Cari album atau artis..."
              class="m3-input w-full h-9 pl-9 pr-8 rounded-full text-xs outline-none placeholder:text-[var(--md-sys-color-on-surface-variant)]"
            />
            {#if albumFilterText}
              <button
                type="button"
                onclick={() => (albumFilterText = '')}
                aria-label="Hapus pencarian"
                class="absolute right-2.5 top-1/2 -translate-y-1/2 w-5 h-5 rounded-full flex items-center justify-center cursor-pointer hover:bg-white/10"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                <Icon name="x" size={12} />
              </button>
            {/if}
          </div>

          {#if displayedAlbums.length > 0 && onPlayAlbum}
            <button
              type="button"
              onclick={() => onPlayAlbum?.(displayedAlbums[0].id)}
              class="h-9 px-5 rounded-full text-xs font-bold flex items-center gap-2 cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
              style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
            >
              <Icon name="play" weight="fill" size={14} />
              <span>Putar</span>
            </button>
          {/if}
        </div>
      </div>

      {#if displayedAlbums.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div
            class="w-14 h-14 mb-4 rounded-2xl flex items-center justify-center"
            style={albumFilterText.trim() && favoriteAlbumsCount > 0
              ? 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);'
              : 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'}
          >
            <Icon name={albumFilterText.trim() && favoriteAlbumsCount > 0 ? 'magnifying-glass' : 'vinyl-record'} weight="fill" size={28} />
          </div>
          {#if albumFilterText.trim() && favoriteAlbumsCount > 0}
            <div class="space-y-1.5">
              <p class="text-base font-bold" style="color: var(--md-sys-color-on-surface);">
                Album tidak ditemukan
              </p>
              <p class="text-xs max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
                Tidak ada album yang cocok dengan "{albumFilterText.trim()}".
              </p>
            </div>
            <button
              type="button"
              onclick={() => (albumFilterText = '')}
              class="mt-5 h-9 px-5 rounded-full text-xs font-bold flex items-center gap-1.5 cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
              style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
            >
              <Icon name="x" size={14} />
              <span>Reset Filter</span>
            </button>
          {:else}
            <div class="space-y-1.5">
              <p class="text-base font-bold" style="color: var(--md-sys-color-on-surface);">
                Belum ada album favorit
              </p>
              <p class="text-xs max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
                Klik ikon hati pada album untuk menyimpannya di sini.
              </p>
            </div>
            <button
              type="button"
              onclick={() => onNavigate?.({ name: 'home', filter: 'album' })}
              class="mt-5 h-9 px-5 rounded-full text-xs font-bold flex items-center gap-2 cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
              style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
            >
              <Icon name="house" weight="fill" size={14} />
              <span>Jelajahi Album</span>
            </button>
          {/if}
        </div>
      {:else}
        <div class="velora-card-grid">
          {#each displayedAlbums as al (al.id)}
            {@const cover = resolveCoverUrl(al.coverUrl, mediaBaseUrl)}
            <div class="group w-[172px] flex flex-col justify-start items-stretch self-start p-2.5 rounded-xl relative transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]">
              <div
                class="velora-card-artwork mb-2.5 flex items-center justify-center shadow-md"
                style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
              >
                <button
                  type="button"
                  onclick={() => onOpenAlbum(al.id)}
                  aria-label="Buka album {al.title}"
                  class="velora-card-artwork-hit cursor-pointer"
                >
                  {#if cover}
                    <img
                      src={cover}
                      alt={al.title}
                      loading="lazy"
                      decoding="async"
                      class="transition-transform duration-300 group-hover:scale-[1.03]"
                      class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                    />
                  {:else}
                    <Icon name="vinyl-record" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.55;" />
                  {/if}
                </button>
                <button
                  type="button"
                  onclick={() => (onPlayAlbum ? onPlayAlbum(al.id) : onOpenAlbum(al.id))}
                  aria-label="Putar album {al.title}"
                  class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200 hover:scale-105"
                >
                  <Icon name="play" weight="fill" size={18} />
                </button>
              </div>

              <button type="button" onclick={() => onOpenAlbum(al.id)} class="w-full flex flex-col justify-start items-stretch text-left cursor-pointer">
                <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                  {al.title}
                </div>
                <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                  {al.trackCount > 0 ? `${al.artistName} • ${al.trackCount} lagu` : `${al.artistName} • YouTube Music`}
                </div>
              </button>

              <button
                type="button"
                onclick={() => onToggleFavoriteAlbum(al.id)}
                aria-label="Hapus {al.title} dari favorit"
                class="border-0 shadow-none absolute top-4 right-4 w-8 h-8 rounded-full backdrop-blur-xs flex items-center justify-center cursor-pointer transition-all duration-200 opacity-100"
                style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-highest) 78%, transparent); color: var(--md-sys-color-tertiary);"
              >
                <Icon name="heart" weight="fill" size={15} />
              </button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{:else if mode === 'favorite-artists'}
  <div class="p-6 min-h-full flex flex-col gap-6" use:svelteMotionEnter={{ y: 8 }}>
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">Favorites • Artists</div>
        <h1 class="text-2xl font-bold tracking-tight mt-0.5" style="color: var(--md-sys-color-on-surface);">
          Favorite artists
        </h1>
        <p class="text-xs mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
          Artis favorit dari koleksi lokal & YouTube Music
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <div class="relative w-56">
          <span
            class="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="magnifying-glass" size={14} />
          </span>
          <input
            type="search"
            bind:value={artistFilterText}
            placeholder="Filter artists..."
            class="w-full h-9 pl-9 pr-3.5 rounded-full text-xs outline-none bg-[var(--md-sys-color-surface-container-high)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
          />
        </div>
        <button
          type="button"
          onclick={() => (showAllArtists = false)}
          class="px-4 py-1.5 rounded-full text-xs font-semibold cursor-pointer transition-colors"
          style={!showAllArtists
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: rgba(128, 128, 128, 0.16); color: var(--md-sys-color-on-surface);'}
        >
          Favorit ({artists.filter((a) => a.isFavorite).length})
        </button>
        <button
          type="button"
          onclick={() => (showAllArtists = true)}
          class="px-4 py-1.5 rounded-full text-xs font-semibold cursor-pointer transition-colors"
          style={showAllArtists
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: rgba(128, 128, 128, 0.16); color: var(--md-sys-color-on-surface);'}
        >
          Semua Artis ({artists.length})
        </button>
      </div>
    </div>

    {#if displayedArtists.length === 0}
      <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
        <div
          class="w-14 h-14 mb-4 rounded-2xl flex items-center justify-center"
          style={artistFilterText.trim()
            ? 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);'
            : 'background-color: var(--md-sys-color-secondary-container); color: var(--md-sys-color-on-secondary-container);'}
        >
          <Icon name={artistFilterText.trim() ? 'magnifying-glass' : 'users'} weight="fill" size={26} />
        </div>
        <div class="space-y-1.5">
          <p class="text-base font-bold" style="color: var(--md-sys-color-on-surface);">
            {artistFilterText.trim() ? 'Artis tidak ditemukan' : 'Belum ada artis favorit'}
          </p>
          <p class="text-xs max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            {artistFilterText.trim()
              ? `Tidak ada artis yang cocok dengan "${artistFilterText.trim()}".`
              : 'Pilih artis favoritmu dari daftar semua artis.'}
          </p>
        </div>
        <button
          type="button"
          onclick={() => (artistFilterText.trim() ? (artistFilterText = '') : (showAllArtists = true))}
          class="mt-5 h-9 px-5 rounded-full text-xs font-bold flex items-center gap-2 cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
          style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
        >
          <Icon name={artistFilterText.trim() ? 'x' : 'users'} weight="fill" size={14} />
          <span>{artistFilterText.trim() ? 'Reset Filter' : 'Lihat Semua Artis'}</span>
        </button>
      </div>
    {:else}
      <div class="velora-card-grid">
        {#each displayedArtists as ar (ar.id)}
          {@const avatar = resolveCoverUrl(ar.avatarUrl, mediaBaseUrl)}
          <div class="group w-[172px] flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg relative transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)] text-center">
            <div class="relative w-[152px] h-[152px] shrink-0 mb-2.5 mx-auto">
              <button
                type="button"
                onclick={() => onOpenArtist(ar.id)}
                aria-label="Buka artis {ar.name}"
                class="relative w-full h-full rounded-full overflow-clip flex items-center justify-center cursor-pointer"
                style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
              >
                {#if avatar}
                  <img
                    src={avatar}
                    alt={ar.name}
                    loading="lazy"
                    decoding="async"
                    class="absolute inset-0 block w-full h-full object-cover object-center"
                    class:velora-yt-video-thumb={isYTVideoThumb(avatar)}
                  />
                {:else}
                  <Icon name="user" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.55;" />
                {/if}
              </button>
              <button
                type="button"
                onclick={() => (onPlayArtist ? onPlayArtist(ar.id) : onOpenArtist(ar.id))}
                aria-label="Putar lagu {ar.name}"
                class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200 hover:scale-105"
              >
                <Icon name="play" weight="fill" size={18} />
              </button>
            </div>

            <button type="button" onclick={() => onOpenArtist(ar.id)} class="w-full flex flex-col justify-start items-stretch text-center cursor-pointer">
              <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                {ar.name}
              </div>
              <div class="text-[11px] leading-snug line-clamp-2 mt-0.5 tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
                Artis • {ar.trackCount} lagu
              </div>
            </button>

            <button
              type="button"
              onclick={() => onToggleFavoriteArtist(ar.id)}
              aria-label="Favoritkan artis {ar.name}"
              class="border-0 shadow-none absolute top-4 right-4 w-8 h-8 rounded-full bg-black/55 backdrop-blur-xs flex items-center justify-center cursor-pointer transition-opacity {ar.isFavorite
                ? 'opacity-100'
                : 'opacity-0 group-hover:opacity-100'}"
            >
              <Icon
                name="heart"
                weight={ar.isFavorite ? 'fill' : 'regular'}
                size={15}
                class={ar.isFavorite ? 'text-rose-400' : 'text-[var(--md-sys-color-on-surface-variant)]'}
              />
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{:else if mode === 'artist-detail' && selectedArtist}
  {@const avatar = resolveCoverUrl(selectedArtist.avatarUrl, mediaBaseUrl)}
  <div class="relative min-h-full flex flex-col" use:svelteMotionEnter={{ y: 8 }}>
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-60"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-secondary-container) 55%, transparent) 0%, transparent 100%);"
    ></div>

    <div class="relative z-10 p-6 flex-1 flex flex-col gap-6">
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-36 h-36 rounded-full overflow-hidden shrink-0 flex items-center justify-center"
            style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
          >
            {#if avatar}
              <img src={avatar} alt={selectedArtist.name} class="block w-full h-full object-cover object-center" />
            {:else}
              <Icon name="user" weight="fill" size={48} style="color: var(--md-sys-color-on-primary-container); opacity: 0.6;" />
            {/if}
          </div>
          <div class="space-y-1.5 min-w-[200px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">Artist</div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              {selectedArtist.name}
            </h1>
            <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
              {detailTracks.length} lagu tersedia di library
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2.5">
          <button
            type="button"
            disabled={detailTracks.length === 0}
            onclick={() => detailTracks[0] && onPlayTrack(detailTracks[0], detailTracks)}
            class="h-10 px-5 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold flex items-center gap-2 cursor-pointer hover:brightness-105 disabled:opacity-40"
          >
            <Icon name="play" weight="fill" size={16} /> Putar Artis
          </button>
          <button
            type="button"
            disabled={detailTracks.length === 0}
            onclick={() => playShuffled(detailTracks)}
            class="h-10 px-4 rounded-full text-xs font-semibold flex items-center gap-1.5 cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] disabled:opacity-40"
            style="color: var(--md-sys-color-on-surface);"
          >
            <Icon name="shuffle" size={15} /> Shuffle
          </button>
        </div>
      </div>

      <TrackTable
        tracks={detailTracks}
        {currentTrackId}
        {currentExternalId}
        {currentTrackTitle}
        {currentTrackArtist}
        {isPlaying}
        {onTogglePlay}
        {mediaBaseUrl}
        {accentColor}
        {playlists}
        {onPlayTrack}
        onToggleFavorite={onToggleFavoriteTrack}
        {onAddToPlaylist}
      />
    </div>
  </div>
{:else}
  <!-- Search & YouTube Music View (Nuclear Borderless Style) -->
  <div class="p-6 space-y-6" use:svelteMotionEnter={{ y: 8 }}>
    <section class="space-y-3">
      {#if ytLoading && !topYTMatch}
        <div
          class="rounded-xl p-4 flex items-center gap-4 animate-pulse"
          style="background-color: var(--md-sys-color-surface-container-high);"
        >
          <div class="w-20 h-20 rounded-md shrink-0 bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="flex-1 space-y-2.5">
            <div class="h-6 w-52 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
            <div class="h-4 w-32 rounded bg-[var(--md-sys-color-surface-container)]"></div>
          </div>
        </div>
      {:else if !topYTMatch}
        <div
          class="p-6 rounded-xl text-xs flex items-center gap-2"
          style="background-color: var(--md-sys-color-surface-container-low); color: var(--md-sys-color-on-surface-variant);"
        >
          <Icon name="magnifying-glass" size={16} /> Ketik kata kunci di kolom pencarian atas untuk mencari lagu langsung dari YouTube Music.
        </div>
      {:else}
        {@const topCover = resolveCoverUrl(topYTMatch.coverUrl, mediaBaseUrl)}
        {@const topActive = isYTActive(topYTMatch)}
        <div
          class="group rounded-xl p-4 flex items-center gap-4 transition-colors duration-150 bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)]"
        >
          <div class="flex items-center gap-4 min-w-0 flex-1 text-left">
            <button
              type="button"
              onclick={() => (topActive && onTogglePlay ? onTogglePlay() : onPlayYTMusic(topYTMatch))}
              aria-label={topActive && isPlaying ? `Jeda ${topYTMatch.title}` : `Putar ${topYTMatch.title}`}
              class="relative w-20 h-20 rounded-md overflow-clip shrink-0 bg-[var(--md-sys-color-surface-container-highest)] cursor-pointer"
            >
              <img
                src={topCover}
                alt={topYTMatch.title}
                class="absolute inset-0 block w-full h-full object-cover object-center"
                class:velora-yt-video-thumb={isYTVideoThumb(topCover)}
              />
            </button>
            <div class="min-w-0 flex-1">
              <button
                type="button"
                onclick={() => (onOpenYTMusicAlbum ? onOpenYTMusicAlbum(topYTMatch) : onPlayYTMusic(topYTMatch))}
                class="block max-w-full text-left text-2xl font-bold tracking-tight truncate hover:underline cursor-pointer"
                style="color: {topActive ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface)'};"
              >
                {topYTMatch.title}
              </button>
              <div class="text-sm truncate mt-1" style="color: var(--md-sys-color-on-surface-variant);">
                Lagu • {topYTMatch.artist || 'YouTube Music'}
              </div>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onclick={() => onSaveYTMusic(topYTMatch)}
              title="Simpan ke Library"
              aria-label="Simpan ke Library"
              class="m3-icon-btn w-10 h-10 flex items-center justify-center cursor-pointer"
            >
              <Icon name="plus" weight="bold" size={18} />
            </button>
            <button
              type="button"
              onclick={() => (topActive && onTogglePlay ? onTogglePlay() : onPlayYTMusic(topYTMatch))}
              title={topActive && isPlaying ? 'Jeda' : 'Putar Sekarang'}
              aria-label={topActive && isPlaying ? `Jeda ${topYTMatch.title}` : `Putar ${topYTMatch.title}`}
              class="w-11 h-11 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer hover:brightness-105 transition-transform active:scale-95"
            >
              {#if topActive && isPlaying}
                <span class="velora-playing-eq" style="color: var(--md-sys-color-on-tertiary);" aria-hidden="true">
                  <span></span>
                  <span></span>
                  <span></span>
                  <span></span>
                </span>
                <span class="hidden group-hover:flex items-center justify-center">
                  <Icon name="pause" weight="fill" size={18} />
                </span>
              {:else}
                <Icon name="play" weight="fill" size={18} />
              {/if}
            </button>
          </div>
        </div>

        {#if otherYTResults.length > 0}
          <div class="grid grid-cols-1 md:grid-cols-2 gap-2 pt-1">
            {#each otherYTResults as yt (yt.videoId)}
              {@const ytCover = resolveCoverUrl(yt.coverUrl, mediaBaseUrl)}
              {@const ytActive = isYTActive(yt)}
              <div class="group flex items-center gap-3 p-2.5 rounded-lg transition-colors duration-150 {ytActive
                ? 'bg-[var(--md-sys-color-state-hover)]'
                : 'hover:bg-[var(--md-sys-color-state-hover)]'}">
                <div class="flex items-center gap-3 min-w-0 flex-1 text-left">
                  <button
                    type="button"
                    onclick={() => (ytActive && onTogglePlay ? onTogglePlay() : onPlayYTMusic(yt))}
                    aria-label={ytActive && isPlaying ? `Jeda ${yt.title}` : `Putar ${yt.title}`}
                    class="relative w-12 h-12 rounded-md overflow-clip shrink-0 bg-[var(--md-sys-color-surface-container-high)] cursor-pointer"
                  >
                    <img
                      src={ytCover}
                      alt={yt.title}
                      class="absolute inset-0 block w-full h-full object-cover object-center"
                      class:velora-yt-video-thumb={isYTVideoThumb(ytCover)}
                    />
                  </button>
                  <div class="min-w-0 flex-1">
                    <button
                      type="button"
                      onclick={() => (onOpenYTMusicAlbum ? onOpenYTMusicAlbum(yt) : onPlayYTMusic(yt))}
                      class="block max-w-full text-left text-xs font-bold truncate hover:underline cursor-pointer"
                      style="color: {ytActive ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface)'};"
                    >
                      {yt.title}
                    </button>
                    <div class="text-[11px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      {yt.artist} • {formatDuration(yt.durationSec)}
                    </div>
                  </div>
                </div>
                <button
                  type="button"
                  onclick={() => onSaveYTMusic(yt)}
                  title="Simpan ke Library"
                  class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
                >
                  <Icon name="plus" weight="bold" size={14} />
                </button>
                <button
                  type="button"
                  onclick={() => (ytActive && onTogglePlay ? onTogglePlay() : onPlayYTMusic(yt))}
                  title={ytActive && isPlaying ? 'Jeda' : 'Putar Sekarang'}
                  aria-label={ytActive && isPlaying ? `Jeda ${yt.title}` : `Putar ${yt.title}`}
                  class="w-8 h-8 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer hover:brightness-105"
                >
                  {#if ytActive && isPlaying}
                    <span class="velora-playing-eq" style="color: var(--md-sys-color-on-tertiary); transform: scale(0.85);" aria-hidden="true">
                      <span></span>
                      <span></span>
                      <span></span>
                      <span></span>
                    </span>
                    <span class="hidden group-hover:flex items-center justify-center">
                      <Icon name="pause" weight="fill" size={14} />
                    </span>
                  {:else}
                    <Icon name="play" weight="fill" size={14} />
                  {/if}
                </button>
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    </section>

    <section class="space-y-3">
      <h2 class="text-base font-bold" style="color: var(--md-sys-color-on-surface);">
        Hasil di Library Velora ({searchMatchedLocal.length})
      </h2>
      {#if searchMatchedLocal.length === 0}
        <div class="p-6 rounded-lg text-xs flex items-center gap-2" style="background-color: var(--md-sys-color-surface-container-low); color: var(--md-sys-color-on-surface-variant);">
          <Icon name="music-note" size={16} /> Tidak ditemukan lagu yang cocok di library tersimpan.
        </div>
      {:else}
        <TrackTable
          tracks={searchMatchedLocal}
          {currentTrackId}
          {currentExternalId}
          {currentTrackTitle}
          {currentTrackArtist}
          {isPlaying}
          {onTogglePlay}
          {mediaBaseUrl}
          {accentColor}
          {playlists}
          {onPlayTrack}
          onToggleFavorite={onToggleFavoriteTrack}
          {onAddToPlaylist}
          {onOpenTrackAlbum}
        />
      {/if}
    </section>
  </div>
{/if}
