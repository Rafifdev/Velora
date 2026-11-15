<script lang="ts">
  import { fade } from 'svelte/transition';
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import type { Album, Playlist, Track, YTMusicAlbumView, YTMusicCard } from '../lib/types';
  import Icon from './Icon.svelte';
  import TrackTable from './TrackTable.svelte';

  interface Props {
    track: Track | null;
    album: Album | null;
    albumTracks: Track[];
    currentTrackId: number | null;
    currentExternalId?: string | null;
    currentTrackTitle?: string | null;
    currentTrackArtist?: string | null;
    isPlaying: boolean;
    tracks: Track[];
    albums: Album[];
    playlists: Playlist[];
    mediaBaseUrl: string;
    accentColor: string;
    onPlayTrack: (tr: Track, list?: Track[]) => void;
    onTogglePlay: () => void;
    onToggleFavoriteTrack: (id: number) => void;
    onToggleFavoriteAlbum: (id: number) => void;
    onAddToPlaylist: (playlistId: number, trackId: number) => void;
    onOpenArtist: (artistId: number) => void;
    onOpenAlbum: (albumId: number) => void;
    ytAlbum?: YTMusicAlbumView | null;
    ytAlbumLoading?: boolean;
    onOpenYTAlbum?: (card: YTMusicCard) => void;
  }

  let {
    track,
    album,
    albumTracks,
    currentTrackId,
    currentExternalId = null,
    currentTrackTitle = null,
    currentTrackArtist = null,
    isPlaying,
    tracks,
    albums,
    playlists,
    mediaBaseUrl,
    accentColor,
    onPlayTrack,
    onTogglePlay,
    onToggleFavoriteTrack,
    onToggleFavoriteAlbum,
    onAddToPlaylist,
    onOpenArtist,
    onOpenAlbum,
    ytAlbum = null,
    ytAlbumLoading = false,
    onOpenYTAlbum
  }: Props = $props();

  const displayTitle = $derived(ytAlbum?.title || album?.title || track?.albumTitle || track?.title || 'Album');
  const displayArtist = $derived(ytAlbum?.artist || album?.artistName || track?.artistName || 'Unknown Artist');
  const displayArtistId = $derived(ytAlbum ? 0 : track?.artistId || 0);
  const rawCover = $derived(ytAlbum?.coverUrl || album?.coverUrl || track?.coverUrl || '');
  const cover = $derived(resolveCoverUrl(rawCover, mediaBaseUrl));

  const listTracks = $derived.by(() => {
    if (ytAlbum && ytAlbum.tracks.length > 0) return ytAlbum.tracks;
    if (albumTracks.length > 0) return albumTracks;
    return track ? [track] : [];
  });

  const isAlbumCurrentlyPlaying = $derived(
    isPlaying && currentTrackId !== null && listTracks.some((t) => t.id === currentTrackId)
  );

  const totalDurationSec = $derived(
    listTracks.reduce((sum, t) => sum + (t.durationSec || 0), 0)
  );

  let albumScrolled = $state(false);

  function playShuffled(list: Track[]) {
    if (list.length === 0) return;
    const copy = [...list];
    for (let i = copy.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [copy[i], copy[j]] = [copy[j], copy[i]];
    }
    onPlayTrack(copy[0], copy);
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

  type ArtistReleaseCard =
    | { kind: 'album'; key: string; title: string; subtitle: string; cover: string; albumId: number }
    | { kind: 'yt-album'; key: string; title: string; subtitle: string; cover: string; ytCard: YTMusicCard }
    | { kind: 'track'; key: string; title: string; subtitle: string; cover: string; track: Track };

  const artistReleases = $derived.by<ArtistReleaseCard[]>(() => {
    if (ytAlbumLoading) return [];

    const cards: ArtistReleaseCard[] = [];
    const seenTitles = new Set<string>();

    if (ytAlbum) {
      for (const yt of ytAlbum.artistAlbums ?? []) {
        const key = yt.title.trim().toLowerCase();
        if (!key || seenTitles.has(key)) continue;
        seenTitles.add(key);
        cards.push({
          kind: 'yt-album',
          key: `yt-${yt.browseId || yt.id}`,
          title: yt.title,
          subtitle: yt.subtitle || 'Album',
          cover: yt.coverUrl,
          ytCard: yt
        });
      }
      return cards;
    }

    const target = displayArtist.trim().toLowerCase();
    if (!target || target === 'unknown artist') return [];
    const currentAlbumTitle = displayTitle.trim().toLowerCase();

    for (const alb of albums) {
      if (album && alb.id === album.id) continue;
      if (alb.artistName.trim().toLowerCase() !== target) continue;
      const key = alb.title.trim().toLowerCase();
      if (seenTitles.has(key)) continue;
      seenTitles.add(key);
      cards.push({
        kind: 'album',
        key: `alb-${alb.id}`,
        title: alb.title,
        subtitle: alb.releaseYear ? String(alb.releaseYear) : 'Album',
        cover: resolveCoverUrl(alb.coverUrl, mediaBaseUrl),
        albumId: alb.id
      });
    }

    for (const tr of tracks) {
      if (album && tr.albumId === album.id) continue;
      if (currentAlbumTitle && tr.albumTitle.trim().toLowerCase() === currentAlbumTitle) continue;
      if (listTracks.some((lt) => lt.id === tr.id)) continue;
      if (tr.artistName.trim().toLowerCase() !== target) continue;
      const key = tr.title.trim().toLowerCase();
      if (seenTitles.has(key)) continue;
      seenTitles.add(key);
      cards.push({
        kind: 'track',
        key: `tr-${tr.id}`,
        title: tr.title,
        subtitle: tr.albumTitle || formatDuration(tr.durationSec),
        cover: resolveCoverUrl(tr.coverUrl, mediaBaseUrl),
        track: tr
      });
    }

    return cards.slice(0, 12);
  });
</script>

<div
  in:fade={{ duration: 180 }}
  onscroll={(e) => (albumScrolled = (e.currentTarget as HTMLElement).scrollTop > 8)}
  class="relative h-full w-full overflow-y-auto rounded-xl select-none"
  style="background-color: var(--md-sys-color-surface-container-low);"
>
  <!-- Top Gradient Fade (sama persis seperti halaman Home saat scroll) -->
  <div
    class="pointer-events-none sticky top-0 left-0 right-0 h-16 -mb-16 z-30 transition-opacity duration-300"
    style="opacity: {albumScrolled ? 1 : 0}; background: linear-gradient(to bottom, var(--md-sys-color-surface) 0%, var(--md-sys-color-surface) 38%, color-mix(in srgb, var(--md-sys-color-surface) 82%, transparent) 58%, color-mix(in srgb, var(--md-sys-color-surface) 42%, transparent) 80%, transparent 100%);"
  ></div>
  <!-- HERO BANNER ALBUM -->
  <section class="relative overflow-hidden px-6 sm:px-10 pt-8 pb-8">
    {#if cover}
      <img
        src={cover}
        alt=""
        aria-hidden="true"
        class="pointer-events-none absolute inset-0 w-full h-full object-cover scale-125 blur-3xl opacity-35"
      />
    {/if}
    <div
      class="pointer-events-none absolute inset-0"
      style="background: linear-gradient(180deg, rgba(11, 12, 14, 0.35) 0%, var(--md-sys-color-surface-container-low) 100%);"
    ></div>

    <div class="relative z-10 flex flex-col sm:flex-row sm:items-end gap-6">
      <div
        class="w-44 h-44 sm:w-52 sm:h-52 rounded-2xl overflow-hidden shadow-2xl shrink-0"
        style="background-color: var(--md-sys-color-surface-container-highest);"
      >
        {#if cover}
          <img
            src={cover}
            alt={displayTitle}
            class="block w-full h-full object-cover object-center"
            class:velora-yt-video-thumb={isYTVideoThumb(cover)}
          />
        {:else}
          <div class="w-full h-full flex items-center justify-center opacity-40">
            <Icon name="vinyl-record" weight="fill" size={56} />
          </div>
        {/if}
      </div>

      <div class="min-w-0 flex-1 space-y-3">
        <div class="flex flex-wrap items-center gap-2 text-xs font-semibold">
          <span
            class="px-2.5 py-0.5 rounded-full"
            style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
          >
            Album
          </span>
          {#if album?.releaseYear}
            <span
              class="px-2.5 py-0.5 rounded-full tabular-nums"
              style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);"
            >
              {album.releaseYear}
            </span>
          {/if}
        </div>

        <h1
          class="text-2xl sm:text-4xl lg:text-[42px] font-extrabold tracking-tight leading-tight line-clamp-2"
          style="color: var(--md-sys-color-on-surface);"
        >
          {displayTitle}
        </h1>

        <div
          class="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-sm"
          style="color: var(--md-sys-color-on-surface-variant);"
        >
          {#if displayArtistId > 0}
            <button
              type="button"
              onclick={() => onOpenArtist(displayArtistId)}
              class="font-bold hover:underline cursor-pointer"
              style="color: var(--md-sys-color-on-surface);"
            >
              {displayArtist}
            </button>
          {:else}
            <span class="font-bold" style="color: var(--md-sys-color-on-surface);">
              {displayArtist}
            </span>
          {/if}

          <span>•</span>
          <span class="tabular-nums">{listTracks.length} lagu</span>
          {#if totalDurationSec > 0}
            <span>•</span>
            <span class="tabular-nums">{formatDuration(totalDurationSec)}</span>
          {/if}
        </div>

        <!-- Action Buttons (Icon Only) -->
        <div class="pt-2 flex flex-wrap items-center gap-3">
          <button
            type="button"
            disabled={listTracks.length === 0}
            onclick={() => {
              if (isAlbumCurrentlyPlaying) {
                onTogglePlay();
              } else if (listTracks[0]) {
                onPlayTrack(listTracks[0], listTracks);
              }
            }}
            title={isAlbumCurrentlyPlaying ? 'Jeda' : 'Putar Album'}
            aria-label={isAlbumCurrentlyPlaying ? 'Jeda' : 'Putar Album'}
            class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110 disabled:opacity-40"
            style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
          >
            <Icon name={isAlbumCurrentlyPlaying ? 'pause' : 'play'} weight="fill" size={20} />
          </button>

          <button
            type="button"
            disabled={listTracks.length === 0}
            onclick={() => playShuffled(listTracks)}
            title="Acak (Shuffle)"
            aria-label="Acak (Shuffle)"
            class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110 disabled:opacity-40"
            style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
          >
            <Icon name="shuffle" size={18} />
          </button>

          {#if album}
            <button
              type="button"
              onclick={() => onToggleFavoriteAlbum(album.id)}
              class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110"
              style="background-color: var(--md-sys-color-surface-container-high); color: {album.isFavorite
                ? 'var(--md-sys-color-tertiary)'
                : 'var(--md-sys-color-on-surface)'};"
              title={album.isFavorite ? 'Hapus Album dari Favorit' : 'Sukai Album'}
              aria-label={album.isFavorite ? 'Hapus Album dari Favorit' : 'Sukai Album'}
            >
              <Icon name="heart" weight={album.isFavorite ? 'fill' : 'regular'} size={18} />
            </button>
          {:else if track}
            <button
              type="button"
              onclick={() => onToggleFavoriteTrack(track.id)}
              class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110"
              style="background-color: var(--md-sys-color-surface-container-high); color: {track.isFavorite
                ? 'var(--md-sys-color-tertiary)'
                : 'var(--md-sys-color-on-surface)'};"
              title={track.isFavorite ? 'Hapus dari Favorit' : 'Tambahkan ke Favorit'}
              aria-label={track.isFavorite ? 'Hapus dari Favorit' : 'Tambahkan ke Favorit'}
            >
              <Icon name="heart" weight={track.isFavorite ? 'fill' : 'regular'} size={18} />
            </button>
          {/if}
        </div>
      </div>
    </div>
  </section>

  <!-- TRACK LIST (seperti di page album) -->
  <div class="px-6 sm:px-10 pb-8">
    {#if ytAlbumLoading}
      <div class="py-10 flex flex-col items-center justify-center gap-3 text-sm" style="color: var(--md-sys-color-on-surface-variant);">
        <span class="velora-eq-loader" aria-hidden="true">
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
        </span>
        <span>Memuat daftar lagu album...</span>
      </div>
    {:else}
      <TrackTable
        tracks={listTracks}
        {currentTrackId}
        {currentExternalId}
        {currentTrackTitle}
        {currentTrackArtist}
        {isPlaying}
        {mediaBaseUrl}
        {accentColor}
        {playlists}
        showCover={false}
        {onPlayTrack}
        {onTogglePlay}
        onToggleFavorite={onToggleFavoriteTrack}
        {onAddToPlaylist}
      />
    {/if}
  </div>

  <!-- LAINNYA DARI ARTIS (Spotify-style Horizontal Shelf) -->
  {#if artistReleases.length > 0}
    <section class="px-6 sm:px-10 pb-10">
      <h2 class="text-lg font-bold mb-4" style="color: var(--md-sys-color-on-surface);">
        Lainnya dari {displayArtist}
      </h2>
      <div
        use:shelfEdgeFade
        class="flex gap-4 overflow-x-auto no-scrollbar velora-shelf-scroll pb-2"
      >
        {#each artistReleases as card (card.key)}
          <button
            type="button"
            onclick={() => {
              if (card.kind === 'album') {
                onOpenAlbum(card.albumId);
              } else if (card.kind === 'yt-album') {
                onOpenYTAlbum?.(card.ytCard);
              } else {
                onPlayTrack(card.track, tracks);
              }
            }}
            class="w-36 sm:w-40 shrink-0 text-left group cursor-pointer"
          >
            <div
              class="w-full aspect-square rounded-lg overflow-hidden mb-2.5 shadow-md"
              style="background-color: var(--md-sys-color-surface-container-highest);"
            >
              {#if card.cover}
                <img
                  src={card.cover}
                  alt={card.title}
                  loading="lazy"
                  decoding="async"
                  class="block w-full h-full object-cover object-center transition-transform duration-300 group-hover:scale-[1.03]"
                  class:velora-yt-video-thumb={isYTVideoThumb(card.cover)}
                />
              {:else}
                <div class="w-full h-full flex items-center justify-center opacity-40">
                  <Icon name="music-note" size={32} />
                </div>
              {/if}
            </div>
            <div class="text-[13px] font-bold leading-snug break-words whitespace-normal" style="color: var(--md-sys-color-on-surface);">
              {card.title}
            </div>
            <div class="text-[11px] leading-snug break-words whitespace-normal mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
              {card.subtitle}
            </div>
          </button>
        {/each}
      </div>
    </section>
  {/if}
</div>
