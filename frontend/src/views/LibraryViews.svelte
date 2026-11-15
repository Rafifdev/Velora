<script lang="ts">
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Album, Artist, Playlist, Track, YTMusicTrack } from '../lib/types';
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
    isPlaying: boolean;
    mediaBaseUrl: string;
    accentColor: string;
    onPlayTrack: (track: Track, list: Track[]) => void;
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
    isPlaying,
    mediaBaseUrl,
    accentColor,
    onPlayTrack,
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
    onOpenArtist
  }: Props = $props();

  let localFilterText = $state('');
  let albumFilterText = $state('');
  let artistFilterText = $state('');
  let ytQuery = $state('');
  let showAllAlbums = $state(false);
  let showAllArtists = $state(false);

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

  const displayedAlbums = $derived.by(() => {
    const base = showAllAlbums ? albums : albums.filter((a) => a.isFavorite);
    const q = albumFilterText.trim().toLowerCase();
    if (!q) return base;
    return base.filter((a) => a.title.toLowerCase().includes(q) || a.artistName.toLowerCase().includes(q));
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
</script>

{#if mode === 'liked-songs'}
  <div class="relative min-h-full" use:svelteMotionEnter={{ y: 8 }}>
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-60"
      style="background: linear-gradient(180deg, rgba(29, 185, 84, 0.16) 0%, transparent 100%);"
    ></div>

    <div class="relative z-10 p-6 space-y-6">
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-center gap-5">
          <div
            class="w-28 h-28 rounded-lg flex items-center justify-center shrink-0"
            style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
          >
            <Icon name="heart" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container);" />
          </div>
          <div class="space-y-1.5 min-w-[200px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
              Favorites • Tracks
            </div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Favorite tracks
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
              placeholder="Filter favorite tracks..."
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
        <div class="p-10 rounded-lg text-center space-y-2" style="background-color: var(--md-sys-color-surface-container-low);">
          <Icon name="heart" weight="fill" size={32} class="mx-auto opacity-45" />
          <p class="text-sm font-semibold">Belum ada lagu favorit</p>
          <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
            Klik ikon hati pada lagu mana saja saat diputar atau di daftar lagu untuk menyimpannya di sini.
          </p>
        </div>
      {:else}
        <TrackTable
          tracks={likedTracks}
          {currentTrackId}
          {isPlaying}
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
  <div class="p-6 space-y-6" use:svelteMotionEnter={{ y: 8 }}>
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">Favorites • Albums</div>
        <h1 class="text-2xl font-bold tracking-tight mt-0.5" style="color: var(--md-sys-color-on-surface);">
          Favorite albums
        </h1>
        <p class="text-xs mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
          Klik sampul untuk melihat tracklist atau tombol Play untuk langsung memutar album
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
            bind:value={albumFilterText}
            placeholder="Filter albums..."
            class="w-full h-9 pl-9 pr-3.5 rounded-full text-xs outline-none bg-[var(--md-sys-color-surface-container-high)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
          />
        </div>
        <button
          type="button"
          onclick={() => (showAllAlbums = false)}
          class="px-4 py-1.5 rounded-full text-xs font-semibold cursor-pointer transition-colors"
          style={!showAllAlbums
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: rgba(128, 128, 128, 0.16); color: var(--md-sys-color-on-surface);'}
        >
          Favorit ({albums.filter((a) => a.isFavorite).length})
        </button>
        <button
          type="button"
          onclick={() => (showAllAlbums = true)}
          class="px-4 py-1.5 rounded-full text-xs font-semibold cursor-pointer transition-colors"
          style={showAllAlbums
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: rgba(128, 128, 128, 0.16); color: var(--md-sys-color-on-surface);'}
        >
          Semua Album ({albums.length})
        </button>
      </div>
    </div>

    {#if displayedAlbums.length === 0}
      <div class="p-10 rounded-lg text-center space-y-2" style="background-color: var(--md-sys-color-surface-container-low);">
        <Icon name="vinyl-record" weight="fill" size={32} class="mx-auto opacity-50" />
        <p class="text-sm font-semibold">Belum ada album di daftar ini</p>
        <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
          Putar lagu dari YouTube Music atau pindai folder lokal, lalu tandai favorit dengan ikon hati.
        </p>
      </div>
    {:else}
      <div class="velora-card-grid">
        {#each displayedAlbums as al (al.id)}
          {@const cover = resolveCoverUrl(al.coverUrl, mediaBaseUrl)}
          <div class="group w-[172px] flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg relative transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]">
            <div
              class="velora-card-artwork mb-2.5 flex items-center justify-center"
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
                class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200 hover:scale-105"
              >
                <Icon name="play" weight="fill" size={18} />
              </button>
            </div>

            <button type="button" onclick={() => onOpenAlbum(al.id)} class="w-full flex flex-col justify-start items-stretch text-left cursor-pointer">
              <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                {al.title}
              </div>
              <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                {al.artistName} • {al.trackCount} lagu
              </div>
            </button>

            <button
              type="button"
              onclick={() => onToggleFavoriteAlbum(al.id)}
              aria-label="Favoritkan album {al.title}"
              class="border-0 shadow-none absolute top-4 right-4 w-8 h-8 rounded-full bg-black/55 backdrop-blur-xs flex items-center justify-center cursor-pointer transition-opacity {al.isFavorite
                ? 'opacity-100'
                : 'opacity-0 group-hover:opacity-100'}"
            >
              <Icon
                name="heart"
                weight={al.isFavorite ? 'fill' : 'regular'}
                size={15}
                class={al.isFavorite ? 'text-rose-400' : 'text-[var(--md-sys-color-on-surface-variant)]'}
              />
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{:else if mode === 'favorite-artists'}
  <div class="p-6 space-y-6" use:svelteMotionEnter={{ y: 8 }}>
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
      <div class="p-10 rounded-lg text-center space-y-2" style="background-color: var(--md-sys-color-surface-container-low);">
        <Icon name="users" weight="fill" size={32} class="mx-auto opacity-50" />
        <p class="text-sm font-semibold">Belum ada artis di daftar ini</p>
        <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
          Klik "Semua Artis" untuk memilih artis favoritmu.
        </p>
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
  <div class="relative min-h-full" use:svelteMotionEnter={{ y: 8 }}>
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-60"
      style="background: linear-gradient(180deg, rgba(29, 185, 84, 0.14) 0%, transparent 100%);"
    ></div>

    <div class="relative z-10 p-6 space-y-6">
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-center gap-5">
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
        {isPlaying}
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
        <div
          class="group rounded-xl p-4 flex items-center gap-4 transition-colors duration-150 bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)]"
        >
          <button
            type="button"
            onclick={() => onPlayYTMusic(topYTMatch)}
            class="flex items-center gap-4 min-w-0 flex-1 text-left cursor-pointer"
          >
            <div class="relative w-20 h-20 rounded-md overflow-clip shrink-0 bg-[var(--md-sys-color-surface-container-highest)]">
              <img
                src={topCover}
                alt={topYTMatch.title}
                class="absolute inset-0 block w-full h-full object-cover object-center"
                class:velora-yt-video-thumb={isYTVideoThumb(topCover)}
              />
            </div>
            <div class="min-w-0 flex-1">
              <div class="text-2xl font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {topYTMatch.title}
              </div>
              <div class="text-sm truncate mt-1" style="color: var(--md-sys-color-on-surface-variant);">
                Lagu • {topYTMatch.artist || 'YouTube Music'}
              </div>
            </div>
          </button>

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
              onclick={() => onPlayYTMusic(topYTMatch)}
              title="Putar Sekarang"
              aria-label="Putar Sekarang"
              class="w-11 h-11 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer hover:brightness-105 transition-transform active:scale-95"
            >
              <Icon name="play" weight="fill" size={18} />
            </button>
          </div>
        </div>

        {#if otherYTResults.length > 0}
          <div class="grid grid-cols-1 md:grid-cols-2 gap-2 pt-1">
            {#each otherYTResults as yt (yt.videoId)}
              {@const ytCover = resolveCoverUrl(yt.coverUrl, mediaBaseUrl)}
              <div class="group flex items-center gap-3 p-2.5 rounded-lg transition-colors duration-150 hover:bg-[var(--md-sys-color-state-hover)]">
                <button
                  type="button"
                  onclick={() => onPlayYTMusic(yt)}
                  class="flex items-center gap-3 min-w-0 flex-1 text-left cursor-pointer"
                >
                  <div class="relative w-12 h-12 rounded-md overflow-clip shrink-0 bg-[var(--md-sys-color-surface-container-high)]">
                    <img
                      src={ytCover}
                      alt={yt.title}
                      class="absolute inset-0 block w-full h-full object-cover object-center"
                      class:velora-yt-video-thumb={isYTVideoThumb(ytCover)}
                    />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                      {yt.title}
                    </div>
                    <div class="text-[11px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      {yt.artist} • {formatDuration(yt.durationSec)}
                    </div>
                  </div>
                </button>
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
                  onclick={() => onPlayYTMusic(yt)}
                  title="Putar Sekarang"
                  class="w-8 h-8 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer hover:brightness-105"
                >
                  <Icon name="play" weight="fill" size={14} />
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
          {isPlaying}
          {mediaBaseUrl}
          {accentColor}
          {playlists}
          {onPlayTrack}
          onToggleFavorite={onToggleFavoriteTrack}
          {onAddToPlaylist}
        />
      {/if}
    </section>
  </div>
{/if}
