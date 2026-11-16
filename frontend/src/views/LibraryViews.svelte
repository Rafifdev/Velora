<script lang="ts">
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
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

  let artistFilterText = $state('');
  let ytQuery = $state('');
  let showAllArtists = $state(false);
  let albumsScrolled = $state(false);

  $effect(() => {
    if (mode === 'search' && searchQuery) {
      ytQuery = searchQuery;
    }
  });

  const likedTracks = $derived(tracks.filter((t) => t.isFavorite));

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

  const displayedAlbums = $derived(albums.filter((a) => a.isFavorite));
  const favoriteAlbumsCount = $derived(displayedAlbums.length);

  function playShuffledAlbum() {
    if (displayedAlbums.length === 0 || !onPlayAlbum) return;
    const rand = displayedAlbums[Math.floor(Math.random() * displayedAlbums.length)];
    onPlayAlbum(rand.id);
  }

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
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <!-- Hero Banner (gradasi oren senada, dibatasi garis pemisah di bawah) -->
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary) 36%, var(--md-sys-color-tertiary-container)) 0%, color-mix(in srgb, var(--md-sys-color-tertiary-container) 74%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg flex items-center justify-center shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)]"
            style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-tertiary) 92%, #ff5500) 0%, color-mix(in srgb, var(--md-sys-color-tertiary) 75%, #ffffff) 52%, color-mix(in srgb, var(--md-sys-color-on-tertiary-container) 88%, #ffffff) 100%);"
          >
            <Icon name="heart" weight="fill" size={44} class="text-white drop-shadow-sm" />
          </div>
          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-tertiary)]">
              Favorites • Music
            </div>
            <h1 class="text-3xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Favorite music
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {likedTracks.length} lagu disukai • Local & YouTube Music
            </p>
          </div>
        </div>

        <!-- Tombol Play All & Shuffle di sisi kanan sejajar dengan teks -->
        <div class="mb-[4px] flex items-center gap-2.5">
          <button
            type="button"
            disabled={likedTracks.length === 0}
            onclick={() => likedTracks[0] && onPlayTrack(likedTracks[0], likedTracks)}
            title="Putar Semua"
            aria-label="Putar Semua"
            class="w-11 h-11 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] inline-flex items-center justify-center cursor-pointer transition-transform duration-150 hover:scale-[1.04] active:scale-95 disabled:opacity-40"
          >
            <Icon name="play" weight="fill" size={18} />
          </button>

          <button
            type="button"
            disabled={likedTracks.length === 0}
            onclick={() => playShuffled(likedTracks)}
            title="Acak (Shuffle)"
            aria-label="Acak (Shuffle)"
            class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] transition-colors disabled:opacity-40"
            style="color: var(--md-sys-color-on-surface);"
          >
            <Icon name="shuffle" size={18} />
          </button>
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-8 flex-1 flex flex-col"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-tertiary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-tertiary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      {#if likedTracks.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
            <Icon name="heart" weight="fill" size={38} />
          </div>
          <h2 class="text-2xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            Belum ada lagu favorit
          </h2>
          <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            Simpan lagu dengan mengetuk ikon hati.
          </p>
          <button
            type="button"
            onclick={() => onNavigate?.({ name: 'home', filter: 'musik' })}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            Cari lagu
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
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <!-- Top Gradient Fade saat scroll -->
    <div
      class="pointer-events-none sticky top-0 left-0 right-0 h-16 -mb-16 z-30 transition-opacity duration-300"
      style="opacity: {albumsScrolled ? 1 : 0}; background: linear-gradient(to bottom, var(--md-sys-color-surface) 0%, var(--md-sys-color-surface) 38%, color-mix(in srgb, var(--md-sys-color-surface) 82%, transparent) 58%, color-mix(in srgb, var(--md-sys-color-surface) 42%, transparent) 80%, transparent 100%);"
    ></div>

    <!-- Hero Banner (gradasi biru senada, dibatasi garis pemisah di bawah) -->
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary) 30%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 74%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)] flex items-center justify-center"
            style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-primary) 45%, #0077b6) 0%, color-mix(in srgb, var(--md-sys-color-primary) 82%, #38bdf8) 52%, color-mix(in srgb, var(--md-sys-color-on-primary-container) 88%, #ffffff) 100%);"
          >
            <Icon name="vinyl-record" weight="fill" size={44} class="text-white drop-shadow-sm" />
          </div>

          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
              Favorites • Albums
            </div>
            <h1 class="text-3xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Favorite albums
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {favoriteAlbumsCount} album favorit • Local & YouTube Music
            </p>
          </div>
        </div>

        <!-- Tombol Play All & Shuffle di sisi kanan sejajar dengan teks -->
        <div class="mb-[4px] flex items-center gap-2.5">
          <button
            type="button"
            disabled={displayedAlbums.length === 0 || !onPlayAlbum}
            onclick={() => displayedAlbums[0] && onPlayAlbum?.(displayedAlbums[0].id)}
            title="Putar Semua"
            aria-label="Putar Semua"
            class="w-11 h-11 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-primary)] text-[var(--md-sys-color-on-primary)] inline-flex items-center justify-center cursor-pointer transition-transform duration-150 hover:scale-[1.04] active:scale-95 disabled:opacity-40"
          >
            <Icon name="play" weight="fill" size={18} />
          </button>

          <button
            type="button"
            disabled={displayedAlbums.length === 0 || !onPlayAlbum}
            onclick={playShuffledAlbum}
            title="Acak (Shuffle)"
            aria-label="Acak (Shuffle)"
            class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] transition-colors disabled:opacity-40"
            style="color: var(--md-sys-color-on-surface);"
          >
            <Icon name="shuffle" size={18} />
          </button>
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-10 flex-1 flex flex-col"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-primary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      {#if displayedAlbums.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
            <Icon name="vinyl-record" weight="fill" size={38} />
          </div>
          <h2 class="text-2xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            Belum ada album favorit
          </h2>
          <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            Simpan album dengan mengetuk ikon hati.
          </p>
          <button
            type="button"
            onclick={() => onNavigate?.({ name: 'home', filter: 'album' })}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            Jelajahi album
          </button>
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
  <div class="p-6 min-h-full flex flex-col gap-5" use:svelteMotionEnter={{ y: 8 }}>
    <div class="flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-2xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
        Favorite artists
      </h1>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onclick={() => (showAllArtists = false)}
          class="h-8 px-3.5 rounded-full text-xs font-semibold cursor-pointer transition-colors"
          style={!showAllArtists
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
        >
          Favorit ({artists.filter((a) => a.isFavorite).length})
        </button>
        <button
          type="button"
          onclick={() => (showAllArtists = true)}
          class="h-8 px-3.5 rounded-full text-xs font-semibold cursor-pointer transition-colors"
          style={showAllArtists
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
        >
          Semua Artis ({artists.length})
        </button>
      </div>
    </div>

    {#if displayedArtists.length === 0}
      <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
        <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
          <Icon name="users" weight="fill" size={38} />
        </div>
        <h2 class="text-2xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
          Belum ada artis favorit
        </h2>
        <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
          Pilih artis favoritmu dari daftar semua artis.
        </p>
        <button
          type="button"
          onclick={() => (showAllArtists = true)}
          class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
          style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
        >
          Lihat semua artis
        </button>
      </div>
    {:else}
      <div class="velora-card-grid">
        {#each displayedArtists as ar (ar.id)}
          {@const avatar = resolveCoverUrl(ar.avatarUrl, mediaBaseUrl)}
          <div class="group w-[172px] flex flex-col justify-start items-stretch self-start p-2.5 rounded-xl relative transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]">
            <div
              class="velora-card-artwork mb-2.5 flex items-center justify-center shadow-md"
              style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
            >
              <button
                type="button"
                onclick={() => onOpenArtist(ar.id)}
                aria-label="Buka artis {ar.name}"
                class="velora-card-artwork-hit cursor-pointer"
              >
                {#if avatar}
                  <img
                    src={avatar}
                    alt={ar.name}
                    loading="lazy"
                    decoding="async"
                    class="transition-transform duration-300 group-hover:scale-[1.03]"
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
                class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-lg bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200 hover:scale-105"
              >
                <Icon name="play" weight="fill" size={18} />
              </button>
            </div>

            <button type="button" onclick={() => onOpenArtist(ar.id)} class="w-full text-left cursor-pointer">
              <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                {ar.name}
              </div>
            </button>

            <button
              type="button"
              onclick={() => onToggleFavoriteArtist(ar.id)}
              aria-label="Favoritkan artis {ar.name}"
              class="border-0 shadow-none absolute top-4 right-4 w-8 h-8 rounded-full backdrop-blur-xs flex items-center justify-center cursor-pointer transition-opacity {ar.isFavorite
                ? 'opacity-100'
                : 'opacity-0 group-hover:opacity-100'}"
              style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-highest) 78%, transparent); color: {ar.isFavorite ? 'var(--md-sys-color-tertiary)' : 'var(--md-sys-color-on-surface-variant)'};"
            >
              <Icon
                name="heart"
                weight={ar.isFavorite ? 'fill' : 'regular'}
                size={15}
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
