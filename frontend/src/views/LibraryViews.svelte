<script lang="ts">
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Album, Artist, DownloadProgress, Playlist, Track, ViewRoute, YTMusicArtistView, YTMusicTrack } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
  import Icon from '../components/Icon.svelte';
  import TrackTable from '../components/TrackTable.svelte';

  interface Props {
    mode:
      | 'search'
      | 'liked-songs'
      | 'downloaded'
      | 'favorite-albums'
      | 'favorite-artists'
      | 'artist-detail';
    searchQuery: string;
    tracks: Track[];
    albums: Album[];
    artists: Artist[];
    selectedArtist?: Artist | null;
    artistFallbackName?: string;
    artistFallbackAvatar?: string;
    artistFallbackBrowseId?: string;
    ytArtistDetail?: YTMusicArtistView | null;
    ytArtistLoading?: boolean;
    detailTracks?: Track[];
    ytResults: YTMusicTrack[];
    ytLoading: boolean;
    playlists: Playlist[];
    downloadStates?: Record<number, DownloadProgress>;
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
    onToggleFavoriteYTMusicArtist?: (artist: { externalId: string; name: string; avatarUrl: string }) => void;
    onAddToPlaylist: (playlistId: number, trackId: number) => void;
    onDownloadTrack?: (track: Track) => void;
    onDownloadYTMusic?: (yt: YTMusicTrack) => void;
    onRemoveDownload?: (trackId: number) => void;
    onCancelDownload?: (trackId: number) => void;
    onOpenAlbum: (albumId: number) => void;
    onOpenArtist: (artistId: number) => void;
    onOpenYTMusicAlbum?: (yt: YTMusicTrack) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onNavigate?: (route: ViewRoute) => void;
    lang?: LangCode;
  }

  let {
    mode,
    searchQuery,
    tracks,
    albums,
    artists,
    selectedArtist = null,
    artistFallbackName = '',
    artistFallbackAvatar = '',
    artistFallbackBrowseId = '',
    ytArtistDetail = null,
    ytArtistLoading = false,
    detailTracks = [],
    ytResults,
    ytLoading,
    playlists,
    downloadStates = {},
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
    onToggleFavoriteYTMusicArtist,
    onAddToPlaylist,
    onDownloadTrack,
    onDownloadYTMusic,
    onRemoveDownload,
    onCancelDownload,
    onOpenAlbum,
    onOpenArtist,
    onOpenYTMusicAlbum,
    onOpenTrackAlbum,
    onNavigate,
    lang = 'en'
  }: Props = $props();

  let mobileSearchInputEl: HTMLInputElement | undefined = $state();
  let showAllArtistTracks = $state(false);

  function formatFansCount(n: number): string {
    if (!n || n <= 0) return '';
    return `${n.toLocaleString('id-ID')} penggemar`;
  }

  function isTrackActive(tr: Track): boolean {
    if (currentTrackId !== null && tr.id === currentTrackId) return true;
    if (currentExternalId && tr.externalId && tr.externalId === currentExternalId) return true;
    if (
      currentTrackTitle &&
      currentTrackArtist &&
      tr.title.trim().toLowerCase() === currentTrackTitle.trim().toLowerCase() &&
      tr.artistName.trim().toLowerCase() === currentTrackArtist.trim().toLowerCase()
    ) {
      return true;
    }
    return false;
  }

  $effect(() => {
    if (mode === 'artist-detail') {
      showAllArtistTracks = false;
    }
  });

  $effect(() => {
    if (mode === 'search' && mobileSearchInputEl && window.innerWidth < 640) {
      requestAnimationFrame(() => mobileSearchInputEl?.focus());
    }
  });

  let artistFilterText = $state('');
  let ytQuery = $state('');
  let albumsScrolled = $state(false);
  let artistsScrolled = $state(false);
  const downloadedTracks = $derived(tracks.filter((t) => t.isDownloaded));
  const activeDownloads = $derived(
    Object.values(downloadStates).filter(
      (d) => d.status === 'queued' || d.status === 'downloading' || d.status === 'failed'
    )
  );

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

  const displayedArtists = $derived(artists.filter((a) => a.isFavorite));
  const favoriteArtistsCount = $derived(displayedArtists.length);

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
            <Icon name="music-note" weight="fill" size={44} class="text-white drop-shadow-sm" />
          </div>
          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-tertiary)]">
              Favorites • Music
            </div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              {tr(lang, 'nav.favoriteMusic')}
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
            <Icon name="music-note" weight="fill" size={38} />
          </div>
          <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            {tr(lang, 'nav.noFavSongs')}
          </h2>
          <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.noFavSongsDesc')}
          </p>
          <button
            type="button"
            onclick={() => onNavigate?.({ name: 'home', filter: 'musik' })}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            {tr(lang, 'nav.exploreSongs')}
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
          {downloadStates}
          {onPlayTrack}
          onToggleFavorite={onToggleFavoriteTrack}
          {onAddToPlaylist}
          {onDownloadTrack}
          {onRemoveDownload}
          {onOpenTrackAlbum}
        />
      {/if}
    </div>
  </div>
{:else if mode === 'downloaded'}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary) 30%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 74%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg flex items-center justify-center shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)]"
            style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-primary) 45%, #0077b6) 0%, color-mix(in srgb, var(--md-sys-color-primary) 82%, #38bdf8) 52%, color-mix(in srgb, var(--md-sys-color-on-primary-container) 88%, #ffffff) 100%);"
          >
            <Icon name="download" weight="fill" size={44} class="text-white drop-shadow-sm" />
          </div>
          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
              Offline • Library
            </div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Downloaded
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {downloadedTracks.length} lagu tersimpan • Siap diputar offline
            </p>
          </div>
        </div>

        <div class="mb-[4px] flex items-center gap-2.5">
          <button
            type="button"
            disabled={downloadedTracks.length === 0}
            onclick={() => downloadedTracks[0] && onPlayTrack(downloadedTracks[0], downloadedTracks)}
            title="Putar Semua"
            aria-label="Putar Semua"
            class="w-11 h-11 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] inline-flex items-center justify-center cursor-pointer transition-transform duration-150 hover:scale-[1.04] active:scale-95 disabled:opacity-40"
          >
            <Icon name="play" weight="fill" size={18} />
          </button>

          <button
            type="button"
            disabled={downloadedTracks.length === 0}
            onclick={() => playShuffled(downloadedTracks)}
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
      class="relative z-10 px-6 sm:px-8 pt-5 pb-8 flex-1 flex flex-col gap-5"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-primary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      {#if activeDownloads.length > 0}
        <div class="rounded-xl p-4 space-y-2.5 bg-[var(--md-sys-color-surface-container)]">
          <div class="text-xs font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
            Sedang Mengunduh ({activeDownloads.length})
          </div>
          {#each activeDownloads as dl (dl.trackId)}
            <div class="flex items-center gap-3 text-xs">
              <div class="min-w-0 flex-1">
                <div class="font-semibold truncate text-[var(--md-sys-color-on-surface)]">
                  {dl.title} <span class="font-normal opacity-70">• {dl.artist}</span>
                </div>
                {#if dl.status === 'failed'}
                  <div class="text-[11px] text-[var(--md-sys-color-error)] mt-0.5 truncate">
                    Gagal: {dl.error || 'Terjadi kesalahan'}
                  </div>
                {:else}
                  <div class="mt-1.5 h-1.5 w-full rounded-full overflow-hidden bg-[var(--md-sys-color-surface-container-highest)]">
                    <div
                      class="h-full rounded-full transition-all duration-200 bg-[var(--md-sys-color-primary)]"
                      style="width: {Math.max(4, Math.min(100, Math.round(dl.progress || 0)))}%;"
                    ></div>
                  </div>
                {/if}
              </div>
              <span class="tabular-nums text-[11px] font-semibold w-10 text-right text-[var(--md-sys-color-on-surface-variant)]">
                {dl.status === 'failed' ? 'Error' : `${Math.round(dl.progress || 0)}%`}
              </span>
              {#if onCancelDownload && dl.status !== 'failed'}
                <button
                  type="button"
                  onclick={() => onCancelDownload(dl.trackId)}
                  title="Batalkan unduhan"
                  class="m3-icon-btn w-7 h-7 flex items-center justify-center cursor-pointer"
                >
                  <Icon name="x" size={14} />
                </button>
              {/if}
            </div>
          {/each}
        </div>
      {/if}

      {#if downloadedTracks.length === 0 && activeDownloads.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
            <Icon name="download" weight="fill" size={38} />
          </div>
          <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            Belum ada lagu yang diunduh
          </h2>
          <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            Unduh lagu favoritmu agar bisa diputar kapan saja tanpa koneksi internet.
          </p>
          <button
            type="button"
            onclick={() => onNavigate?.({ name: 'home', filter: 'musik' })}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            Jelajahi lagu
          </button>
        </div>
      {:else if downloadedTracks.length > 0}
        <TrackTable
          tracks={downloadedTracks}
          {currentTrackId}
          {currentExternalId}
          {currentTrackTitle}
          {currentTrackArtist}
          {isPlaying}
          {onTogglePlay}
          {mediaBaseUrl}
          {accentColor}
          {playlists}
          {downloadStates}
          {onPlayTrack}
          onToggleFavorite={onToggleFavoriteTrack}
          {onAddToPlaylist}
          {onDownloadTrack}
          {onRemoveDownload}
          {onOpenTrackAlbum}
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

    <!-- Hero Banner (gradasi oranye senada seperti Favorite music) -->
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary) 36%, var(--md-sys-color-tertiary-container)) 0%, color-mix(in srgb, var(--md-sys-color-tertiary-container) 74%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)] flex items-center justify-center"
            style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-tertiary) 92%, #ff5500) 0%, color-mix(in srgb, var(--md-sys-color-tertiary) 75%, #ffffff) 52%, color-mix(in srgb, var(--md-sys-color-on-tertiary-container) 88%, #ffffff) 100%);"
          >
            <Icon name="vinyl-record" weight="fill" size={44} class="text-white drop-shadow-sm" />
          </div>

          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-tertiary)]">
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
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-10 flex-1 flex flex-col"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-tertiary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-tertiary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      {#if displayedAlbums.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
            <Icon name="vinyl-record" weight="fill" size={38} />
          </div>
          <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
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
                class="border-0 shadow-none absolute top-4 right-4 w-8 h-8 rounded-full backdrop-blur-xs flex items-center justify-center cursor-pointer transition-all duration-200 opacity-100 hover:scale-110"
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
  <div
    onscroll={(e) => (artistsScrolled = (e.currentTarget as HTMLElement).scrollTop > 8)}
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <!-- Top Gradient Fade saat scroll -->
    <div
      class="pointer-events-none sticky top-0 left-0 right-0 h-16 -mb-16 z-30 transition-opacity duration-300"
      style="opacity: {artistsScrolled ? 1 : 0}; background: linear-gradient(to bottom, var(--md-sys-color-surface) 0%, var(--md-sys-color-surface) 38%, color-mix(in srgb, var(--md-sys-color-surface) 82%, transparent) 58%, color-mix(in srgb, var(--md-sys-color-surface) 42%, transparent) 80%, transparent 100%);"
    ></div>

    <!-- Hero Banner (gradasi oranye senada seperti Favorite music & Favorite albums) -->
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary) 36%, var(--md-sys-color-tertiary-container)) 0%, color-mix(in srgb, var(--md-sys-color-tertiary-container) 74%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)] flex items-center justify-center"
            style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-tertiary) 92%, #ff5500) 0%, color-mix(in srgb, var(--md-sys-color-tertiary) 75%, #ffffff) 52%, color-mix(in srgb, var(--md-sys-color-on-tertiary-container) 88%, #ffffff) 100%);"
          >
            <Icon name="user" weight="fill" size={44} class="text-white drop-shadow-sm" />
          </div>

          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-tertiary)]">
              Favorites • Artists
            </div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Favorite artists
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {favoriteArtistsCount} artis favorit • Local & YouTube Music
            </p>
          </div>
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-10 flex-1 flex flex-col"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-tertiary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-tertiary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      {#if displayedArtists.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
            <Icon name="user" weight="fill" size={38} />
          </div>
          <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            Belum ada artis favorit
          </h2>
          <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            Simpan artis dengan mengetuk ikon hati.
          </p>
          <button
            type="button"
            onclick={() => onNavigate?.({ name: 'home', filter: 'artis' })}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            Jelajahi artis
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

              <button type="button" onclick={() => onOpenArtist(ar.id)} class="w-full flex flex-col justify-start items-stretch text-left cursor-pointer">
                <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                  {ar.name}
                </div>
                <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                  {ar.trackCount > 0 ? `${ar.trackCount} lagu` : 'YouTube Music'}
                </div>
              </button>

              <button
                type="button"
                onclick={() => onToggleFavoriteArtist(ar.id)}
                aria-label="Hapus {ar.name} dari favorit"
                class="border-0 shadow-none absolute top-4 right-4 w-8 h-8 rounded-full backdrop-blur-xs flex items-center justify-center cursor-pointer transition-all duration-200 opacity-100 hover:scale-110"
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
{:else if mode === 'artist-detail'}
  {@const artistName = ytArtistDetail?.name || selectedArtist?.name || artistFallbackName || 'Artist'}
  {@const rawAvatar = artistFallbackAvatar || ytArtistDetail?.avatarUrl || selectedArtist?.avatarUrl || ''}
  {@const avatar = resolveCoverUrl(rawAvatar, mediaBaseUrl)}
  {@const artistTracks = ytArtistDetail?.topTracks?.length ? ytArtistDetail.topTracks : detailTracks}
  {@const visibleTracks = showAllArtistTracks ? artistTracks : artistTracks.slice(0, 5)}
  {@const artistAlbums = ytArtistDetail?.albums ?? []}
  {@const relatedArtists = ytArtistDetail?.relatedArtists ?? []}
  {@const fansText = ytArtistDetail?.fansCount
    ? formatFansCount(ytArtistDetail.fansCount)
    : artistTracks.length > 0
      ? `${artistTracks.length} lagu tersedia`
      : ''}
  {@const isArtistFav = !!selectedArtist?.isFavorite}
  {@const isArtistPlaying = isPlaying && artistTracks.some((t) => isTrackActive(t))}

  <div class="relative min-h-full flex flex-col" use:svelteMotionEnter={{ y: 8 }}>
    {#if ytArtistLoading}
      <!-- Artist Detail Skeleton -->
      <div class="relative overflow-hidden px-4 sm:px-6 pt-8 pb-6 sm:pt-12 sm:pb-8 animate-pulse">
        <div class="flex flex-col sm:flex-row sm:items-end gap-4 sm:gap-6">
          <div class="w-28 h-28 sm:w-44 sm:h-44 rounded-full shrink-0 bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="space-y-3 flex-1 min-w-0">
            <div class="h-3.5 w-32 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
            <div class="h-10 sm:h-12 w-56 sm:w-80 rounded-lg bg-[var(--md-sys-color-surface-container-highest)]"></div>
            <div class="h-3.5 w-40 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
          </div>
        </div>
      </div>

      <div class="px-4 sm:px-6 pb-8 space-y-8 animate-pulse">
        <div class="flex items-center gap-3 sm:gap-4">
          <div class="w-13 h-13 sm:w-14 sm:h-14 rounded-full bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="w-10 h-10 rounded-full bg-[var(--md-sys-color-surface-container-high)]"></div>
          <div class="h-9 w-24 rounded-full bg-[var(--md-sys-color-surface-container-high)]"></div>
        </div>

        <div class="space-y-3">
          <div class="h-6 w-28 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="space-y-2">
            {#each [0, 1, 2, 3, 4] as idx (idx)}
              <div class="flex items-center gap-3 px-2 py-2 rounded-md">
                <div class="w-6 h-4 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                <div class="w-10 h-10 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
                <div class="flex-1 space-y-1.5">
                  <div class="h-3.5 w-48 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                  <div class="h-3 w-28 rounded bg-[var(--md-sys-color-surface-container)]"></div>
                </div>
                <div class="h-3 w-10 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
            {/each}
          </div>
        </div>

        <div class="space-y-3">
          <div class="h-6 w-32 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="flex items-start gap-3 overflow-hidden">
            {#each [0, 1, 2, 3, 4] as idx (idx)}
              <div class="w-[132px] sm:w-[168px] shrink-0 p-2 space-y-2">
                <div class="w-full aspect-square rounded-md bg-[var(--md-sys-color-surface-container-high)]"></div>
                <div class="h-3.5 w-3/4 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                <div class="h-3 w-1/2 rounded bg-[var(--md-sys-color-surface-container)]"></div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {:else}
      <!-- Spotify-Style Artist Hero Banner -->
      <div class="relative overflow-hidden px-4 sm:px-6 pt-8 pb-6 sm:pt-12 sm:pb-8">
        {#if avatar}
          <img
            src={avatar}
            alt=""
            aria-hidden="true"
            class="pointer-events-none absolute inset-0 w-full h-full object-cover object-center scale-110 blur-2xl opacity-35"
          />
        {/if}
        <div
          class="pointer-events-none absolute inset-0"
          style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary-container) 42%, transparent) 0%, color-mix(in srgb, var(--md-sys-color-surface) 82%, transparent) 72%, var(--md-sys-color-surface) 100%);"
        ></div>

        <div class="relative z-10 flex flex-col sm:flex-row sm:items-end gap-4 sm:gap-6">
          <div
            class="velora-artist-avatar w-28 h-28 sm:w-44 sm:h-44 rounded-full overflow-hidden shrink-0 flex items-center justify-center shadow-2xl"
            style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
          >
            {#if avatar}
              <img src={avatar} alt={artistName} class="block w-full h-full object-cover object-center" />
            {:else}
              <Icon name="user" weight="fill" size={56} style="color: var(--md-sys-color-on-primary-container); opacity: 0.7;" />
            {/if}
          </div>

          <div class="min-w-0 space-y-1.5 sm:space-y-2">
            <div class="inline-flex items-center gap-1.5 text-xs font-semibold" style="color: var(--md-sys-color-on-surface);">
              <Icon name="seal-check" weight="fill" size={18} style="color: #3d91f4;" />
              <span>Artis Terverifikasi</span>
            </div>
            <h1 class="text-3xl sm:text-5xl lg:text-6xl font-bold tracking-tight leading-none truncate" style="color: var(--md-sys-color-on-surface);">
              {artistName}
            </h1>
            {#if fansText}
              <p class="text-xs sm:text-sm font-medium pt-1" style="color: var(--md-sys-color-on-surface-variant);">
                {fansText}
              </p>
            {/if}
          </div>
        </div>
      </div>

      <!-- Action Bar + Content -->
      <div class="relative z-10 px-4 sm:px-6 pb-8 space-y-8">
        <div class="flex items-center gap-3 sm:gap-4">
          <button
            type="button"
            disabled={artistTracks.length === 0}
            onclick={() => {
              if (isArtistPlaying && onTogglePlay) onTogglePlay();
              else if (artistTracks[0]) onPlayTrack(artistTracks[0], artistTracks);
            }}
            aria-label={isArtistPlaying ? 'Jeda' : 'Putar Artis'}
            class="w-13 h-13 sm:w-14 sm:h-14 rounded-full border-0 outline-none shadow-lg flex items-center justify-center cursor-pointer transition-transform hover:scale-105 active:scale-95 disabled:opacity-40"
            style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
          >
            <Icon name={isArtistPlaying ? 'pause' : 'play'} weight="fill" size={24} />
          </button>

          <button
            type="button"
            disabled={artistTracks.length === 0}
            onclick={() => playShuffled(artistTracks)}
            aria-label="Acak lagu artis"
            title="Shuffle"
            class="w-10 h-10 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)] disabled:opacity-40"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="shuffle" size={22} />
          </button>

          <button
            type="button"
            onclick={() => {
              if (selectedArtist) {
                onToggleFavoriteArtist(selectedArtist.id);
              } else if (onToggleFavoriteYTMusicArtist) {
                onToggleFavoriteYTMusicArtist({
                  externalId: ytArtistDetail?.browseId || artistFallbackBrowseId,
                  name: artistName,
                  avatarUrl: rawAvatar
                });
              }
            }}
            class="h-9 px-5 rounded-full text-xs font-bold cursor-pointer transition-all hover:scale-[1.03] active:scale-95 {isArtistFav
              ? 'hover:brightness-110'
              : 'bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-[var(--md-sys-color-on-surface)]'}"
            style={isArtistFav
              ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
              : undefined}
          >
            {isArtistFav ? 'Mengikuti' : 'Ikuti'}
          </button>
        </div>

        <!-- Populer Section (Reusing TrackTable Component) -->
        <section class="space-y-3">
          <h2 class="text-lg sm:text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            Populer
          </h2>

          {#if artistTracks.length === 0}
            <p class="text-xs py-4" style="color: var(--md-sys-color-on-surface-variant);">
              Belum ada lagu populer untuk artis ini.
            </p>
          {:else}
            <TrackTable
              tracks={visibleTracks}
              {currentTrackId}
              {currentExternalId}
              {currentTrackTitle}
              {currentTrackArtist}
              {isPlaying}
              {onTogglePlay}
              {mediaBaseUrl}
              {accentColor}
              {playlists}
              {downloadStates}
              {onPlayTrack}
              onToggleFavorite={onToggleFavoriteTrack}
              {onAddToPlaylist}
              {onDownloadTrack}
              {onRemoveDownload}
              {onOpenTrackAlbum}
            />

            {#if artistTracks.length > 5}
              <button
                type="button"
                onclick={() => (showAllArtistTracks = !showAllArtistTracks)}
                class="px-2 pt-1 text-xs font-bold cursor-pointer hover:underline"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                {showAllArtistTracks ? 'Tampilkan lebih sedikit' : 'Lihat lebih banyak'}
              </button>
            {/if}
          {/if}
        </section>

        <!-- Album Section -->
        {#if artistAlbums.length > 0}
          <section class="space-y-3">
            <h2 class="text-lg sm:text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Album
            </h2>
            <div class="flex items-start gap-2.5 sm:gap-3 overflow-x-auto no-scrollbar velora-shelf-scroll pb-1">
              {#each artistAlbums as alb (alb.id)}
                {@const aCover = resolveCoverUrl(alb.coverUrl, mediaBaseUrl)}
                <button
                  type="button"
                  onclick={() =>
                    onNavigate?.({
                      name: 'yt-album-detail',
                      videoId: '',
                      browseId: alb.browseId,
                      title: alb.title,
                      artist: alb.artist || artistName,
                      albumTitle: alb.title,
                      coverUrl: alb.coverUrl
                    })}
                  class="group w-[132px] sm:w-[168px] shrink-0 flex flex-col p-2 sm:p-2.5 rounded-lg text-left cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div class="velora-card-artwork mb-2.5 flex items-center justify-center bg-[var(--md-sys-color-surface-container-high)]">
                    {#if aCover}
                      <img src={aCover} alt={alb.title} loading="lazy" decoding="async" />
                    {:else}
                      <Icon name="vinyl-record" weight="fill" size={38} class="opacity-40" />
                    {/if}
                  </div>
                  <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                    {alb.title}
                  </div>
                  <div class="text-[11px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    {alb.subtitle || 'Album'}
                  </div>
                </button>
              {/each}
            </div>
          </section>
        {/if}

        <!-- Penggemar Juga Menyukai Section -->
        {#if relatedArtists.length > 0}
          <section class="space-y-3">
            <h2 class="text-lg sm:text-xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Penggemar juga menyukai
            </h2>
            <div class="flex items-start gap-2.5 sm:gap-3 overflow-x-auto no-scrollbar velora-shelf-scroll pb-1">
              {#each relatedArtists as rel (rel.id)}
                {@const rCover = resolveCoverUrl(rel.coverUrl, mediaBaseUrl)}
                <button
                  type="button"
                  onclick={() =>
                    onNavigate?.({
                      name: 'artist-detail',
                      browseId: rel.browseId,
                      artistName: rel.title,
                      avatarUrl: rel.coverUrl
                    })}
                  class="group w-[124px] sm:w-[156px] shrink-0 flex flex-col items-center p-2 sm:p-2.5 rounded-lg text-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                >
                  <div class="velora-card-artwork velora-artist-avatar !rounded-full mb-2.5 flex items-center justify-center bg-[var(--md-sys-color-surface-container-high)] shadow-sm">
                    {#if rCover}
                      <img src={rCover} alt={rel.title} loading="lazy" decoding="async" />
                    {:else}
                      <Icon name="user" weight="fill" size={36} class="opacity-40" />
                    {/if}
                  </div>
                  <div class="w-full text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                    {rel.title}
                  </div>
                  <div class="text-[11px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    Artis
                  </div>
                </button>
              {/each}
            </div>
          </section>
        {/if}
      </div>
    {/if}
  </div>
{:else}
  <!-- Search & YouTube Music View -->
  <div class="p-3 sm:p-6 space-y-1 sm:space-y-6" use:svelteMotionEnter={{ y: 8 }}>
    <!-- Mobile Full-Width Search Bar (Sticky at exact top with smooth dark top fade) -->
    <form
      onsubmit={(e) => {
        e.preventDefault();
        onSearchYTMusic(ytQuery);
      }}
      class="sm:hidden sticky top-0 z-20 -mx-3 -mt-3 px-3 pt-3 pb-2 border-0 outline-none"
      style="background: linear-gradient(to bottom, var(--md-sys-color-surface) 0%, var(--md-sys-color-surface) 68%, color-mix(in srgb, var(--md-sys-color-surface) 78%, transparent) 86%, transparent 100%);"
    >
      <div
        class="w-full h-10 px-3 rounded-md border-0 outline-none flex items-center gap-2.5"
        style="background-color: var(--md-sys-color-surface-container-high);"
      >
        <Icon name="magnifying-glass" size={18} style="color: var(--md-sys-color-on-surface-variant);" class="shrink-0" />
        <input
          id="mobile-search-input"
          bind:this={mobileSearchInputEl}
          type="search"
          bind:value={ytQuery}
          oninput={() => onSearchYTMusic(ytQuery)}
          placeholder="Apa yang ingin kamu putar?"
          class="flex-1 min-w-0 bg-transparent border-0 outline-none ring-0 text-sm"
          style="color: var(--md-sys-color-on-surface);"
        />
        {#if ytQuery}
          <button
            type="button"
            onclick={() => {
              ytQuery = '';
              onSearchYTMusic('');
            }}
            aria-label="Hapus pencarian"
            class="w-6 h-6 rounded-full flex items-center justify-center shrink-0 cursor-pointer border-0 transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="x-circle" weight="fill" size={17} />
          </button>
        {/if}
      </div>
    </form>

    <!-- Mobile Spotify-Style Search Suggestions & Results -->
    <div class="sm:hidden space-y-1">
      {#if ytLoading && ytResults.length === 0}
        <div class="space-y-2 pt-1 animate-pulse">
          {#each [0, 1, 2, 3, 4, 5, 6, 7] as idx (idx)}
            <div class="flex items-center gap-3 px-2 py-2 rounded-md">
              <div class="w-12 h-12 rounded-md shrink-0 bg-[var(--md-sys-color-surface-container-high)]"></div>
              <div class="flex-1 min-w-0 space-y-1.5">
                <div class="h-3.5 w-44 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                <div class="h-3 w-28 rounded bg-[var(--md-sys-color-surface-container)]"></div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
      {#if ytResults.length > 0}
        {#each ytResults.slice(0, 5) as sug (`sug-${sug.videoId}`)}
          <button
            type="button"
            onclick={() => {
              ytQuery = sug.title;
              onSearchYTMusic(sug.title);
            }}
            class="w-full flex items-center gap-3 px-2 py-2.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] text-left cursor-pointer"
          >
            <Icon name="magnifying-glass" size={18} style="color: var(--md-sys-color-on-surface-variant);" class="shrink-0" />
            <span class="flex-1 min-w-0 text-sm truncate" style="color: var(--md-sys-color-on-surface);">
              {sug.title}
            </span>
            <Icon name="arrow-up-left" size={16} style="color: var(--md-sys-color-on-surface-variant);" class="shrink-0" />
          </button>
        {/each}
      {/if}

      {#each ytResults as yt (yt.videoId)}
        {@const mCover = resolveCoverUrl(yt.coverUrl, mediaBaseUrl)}
        {@const mActive = isYTActive(yt)}
        <div class="flex items-center gap-3 px-2 py-2 rounded-md hover:bg-[var(--md-sys-color-state-hover)]">
          <button
            type="button"
            onclick={() => (mActive && onTogglePlay ? onTogglePlay() : onPlayYTMusic(yt))}
            class="flex-1 min-w-0 flex items-center gap-3 text-left cursor-pointer"
          >
            <div class="w-12 h-12 rounded-md overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-high)] flex items-center justify-center">
              {#if mCover}
                <img
                  src={mCover}
                  alt={yt.title}
                  class="block w-full h-full object-cover object-center"
                  class:velora-yt-video-thumb={isYTVideoThumb(mCover)}
                />
              {:else}
                <Icon name="music-note" size={18} />
              {/if}
            </div>
            <div class="min-w-0 flex-1">
              <div
                class="text-sm font-semibold truncate"
                style="color: {mActive ? 'var(--md-sys-color-tertiary)' : 'var(--md-sys-color-on-surface)'};"
              >
                {yt.title}
              </div>
              <div class="text-xs truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                Song • {yt.artist || 'YouTube Music'}
              </div>
            </div>
          </button>
          <button
            type="button"
            onclick={() => onSaveYTMusic(yt)}
            aria-label="Simpan ke Library"
            class="w-9 h-9 rounded-full flex items-center justify-center shrink-0 cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="plus-circle" size={20} />
          </button>
        </div>
      {/each}
    </div>

    <section class="hidden sm:block space-y-3">
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
                {#if onDownloadYTMusic}
                  <button
                    type="button"
                    onclick={() => onDownloadYTMusic(yt)}
                    title="Unduh untuk offline"
                    class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
                  >
                    <Icon name="download" size={14} />
                  </button>
                {/if}
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
          {downloadStates}
          {onPlayTrack}
          onToggleFavorite={onToggleFavoriteTrack}
          {onAddToPlaylist}
          {onDownloadTrack}
          {onRemoveDownload}
          {onOpenTrackAlbum}
        />
      {/if}
    </section>
  </div>
{/if}
