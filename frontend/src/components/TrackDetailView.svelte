<script lang="ts">
  import { fade } from 'svelte/transition';
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import type { Album, Playlist, Track, YTMusicAlbumView, YTMusicCard } from '../lib/types';
  import EqIndicator from './EqIndicator.svelte';
  import Icon from './Icon.svelte';

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
    onAddToQueue?: (tr: Track) => void;
    onDownloadTrack?: (tr: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist: (artistId: number) => void;
    onOpenAlbum: (albumId: number) => void;
    ytAlbum?: YTMusicAlbumView | null;
    ytAlbumLoading?: boolean;
    onOpenYTAlbum?: (card: YTMusicCard) => void;
    onToggleFavoriteYTMusicAlbum?: (payload: {
      browseId: string;
      title: string;
      artist: string;
      coverUrl: string;
      releaseYear: number;
    }) => void;
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
    onAddToQueue,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenAlbum,
    ytAlbum = null,
    ytAlbumLoading = false,
    onOpenYTAlbum,
    onToggleFavoriteYTMusicAlbum
  }: Props = $props();

  const displayTitle = $derived(ytAlbum?.title || album?.title || track?.albumTitle || track?.title || 'Album');
  const displayArtist = $derived(ytAlbum?.artist || album?.artistName || track?.artistName || 'Unknown Artist');
  const displayArtistId = $derived(ytAlbum ? 0 : track?.artistId || 0);
  const rawCover = $derived(ytAlbum?.coverUrl || album?.coverUrl || track?.coverUrl || '');
  const cover = $derived(resolveCoverUrl(rawCover, mediaBaseUrl));

  const isYTFavAlbum = $derived.by(() => {
    if (!ytAlbum) return false;
    const bid = (ytAlbum.browseId || '').trim();
    const tLow = displayTitle.trim().toLowerCase();
    const aLow = displayArtist.trim().toLowerCase();
    return albums.some(
      (a) =>
        a.isFavorite &&
        a.sourceType === 'ytmusic' &&
        ((bid !== '' && a.externalId === bid) ||
          (a.title.trim().toLowerCase() === tLow && a.artistName.trim().toLowerCase() === aLow))
    );
  });

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
  let openActionTrackId = $state<number | null>(null);
  let showPlaylistSubmenu = $state(false);

  const releaseDateLabel = $derived.by(() => {
    const y = (ytAlbum?.year || '').trim();
    if (y) return y;
    if (album?.releaseYear && album.releaseYear > 0) return String(album.releaseYear);
    return '—';
  });

  function normTitle(s: string): string {
    return (s || '')
      .toLowerCase()
      .replace(/\s*[\(\[](?:official|audio|video|mv|lyrics?|hd|4k).*?[\)\]]/gi, '')
      .replace(/\s+/g, ' ')
      .trim();
  }

  function isTrackActive(tr: Track): boolean {
    if (tr.id === currentTrackId) return true;
    if (currentExternalId && tr.externalId === currentExternalId) return true;
    if (currentTrackTitle) {
      const nt = normTitle(tr.title);
      const nc = normTitle(currentTrackTitle);
      if (nt && nc && nt === nc) {
        if (!currentTrackArtist) return true;
        const a1 = (tr.artistName || '').toLowerCase();
        const a2 = currentTrackArtist.toLowerCase();
        if (a1.includes(a2) || a2.includes(a1)) return true;
      }
    }
    return false;
  }

  function handleRowPlayClick(tr: Track, active: boolean) {
    if (active) {
      onTogglePlay();
    } else {
      onPlayTrack(tr, listTracks);
    }
  }

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
  onclick={() => {
    openActionTrackId = null;
    showPlaylistSubmenu = false;
  }}
  role="presentation"
  class="relative h-full w-full overflow-y-auto rounded-xl select-none"
  style="background-color: var(--md-sys-color-surface-container-low);"
>
  <div
    class="pointer-events-none sticky top-0 left-0 right-0 h-16 -mb-16 z-30 transition-opacity duration-300"
    style="opacity: {albumScrolled ? 1 : 0}; background: linear-gradient(to bottom, var(--md-sys-color-surface) 0%, var(--md-sys-color-surface) 38%, color-mix(in srgb, var(--md-sys-color-surface) 82%, transparent) 58%, color-mix(in srgb, var(--md-sys-color-surface) 42%, transparent) 80%, transparent 100%);"
  ></div>
  <section class="relative overflow-hidden px-6 sm:px-10 pt-8 pb-8">
    {#if ytAlbumLoading}
      <div class="relative z-10 flex flex-col sm:flex-row sm:items-end gap-6 animate-pulse">
        <div
          class="w-44 h-44 sm:w-52 sm:h-52 rounded-2xl shrink-0"
          style="background-color: var(--md-sys-color-surface-container-highest);"
        ></div>
        <div class="min-w-0 flex-1 space-y-3">
          <div class="h-5 w-16 rounded-full bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="h-10 w-64 max-w-full rounded-lg bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="h-4 w-44 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="pt-2 flex items-center gap-3">
            <div class="w-11 h-11 rounded-full bg-[var(--md-sys-color-surface-container-highest)]"></div>
            <div class="w-11 h-11 rounded-full bg-[var(--md-sys-color-surface-container-highest)]"></div>
          </div>
        </div>
      </div>
    {:else}
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
          class="text-2xl sm:text-4xl lg:text-[42px] font-bold tracking-tight leading-tight line-clamp-2"
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
          {:else if ytAlbum && onToggleFavoriteYTMusicAlbum}
            <button
              type="button"
              onclick={() =>
                onToggleFavoriteYTMusicAlbum({
                  browseId: ytAlbum.browseId || '',
                  title: displayTitle,
                  artist: displayArtist,
                  coverUrl: rawCover,
                  releaseYear: 0
                })}
              class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110"
              style="background-color: var(--md-sys-color-surface-container-high); color: {isYTFavAlbum
                ? 'var(--md-sys-color-tertiary)'
                : 'var(--md-sys-color-on-surface)'};"
              title={isYTFavAlbum ? 'Hapus Album dari Favorit' : 'Sukai Album'}
              aria-label={isYTFavAlbum ? 'Hapus Album dari Favorit' : 'Sukai Album'}
            >
              <Icon name="heart" weight={isYTFavAlbum ? 'fill' : 'regular'} size={18} />
            </button>
          {:else if track && !ytAlbumLoading}
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
    {/if}
  </section>

  <div class="px-6 sm:px-10 pb-8">
    {#if ytAlbumLoading}
      <div class="w-full animate-pulse" aria-busy="true" aria-label="Memuat daftar lagu album">
        <div
          class="-mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] gap-4 px-3 py-2 text-[11px] font-bold uppercase tracking-wider"
          style="color: var(--md-sys-color-on-surface-variant);"
        >
          <div class="text-center">#</div>
          <div>Title</div>
          <div class="hidden md:block">Rilis</div>
          <div class="hidden md:block">Pemutaran</div>
          <div class="flex justify-end pr-1">
            <Icon name="clock" size={14} />
          </div>
          <div></div>
        </div>

        <div class="space-y-1.5 mt-1">
          {#each [62, 48, 70, 54, 66, 44] as titleW, idx (idx)}
            <div class="-mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] gap-4 px-3 py-2.5 items-center rounded-lg">
              <div class="flex justify-center">
                <div class="w-4 h-3 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
              <div class="space-y-1.5 min-w-0">
                <div class="h-3.5 rounded bg-[var(--md-sys-color-surface-container-highest)]" style="width: {titleW}%;"></div>
                <div class="h-2.5 w-28 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
              <div class="hidden md:block">
                <div class="h-3 w-12 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
              <div class="hidden md:block">
                <div class="h-3 w-14 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
              <div class="flex justify-end pr-1">
                <div class="h-3 w-9 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
              <div></div>
            </div>
          {/each}
        </div>

        <div class="mt-8 space-y-4">
          <div class="h-5 w-44 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="flex gap-3 overflow-hidden">
            {#each Array.from({ length: 6 }, (_, i) => i) as idx (idx)}
              <div class="w-[172px] shrink-0 p-2.5 space-y-2.5">
                <div class="w-[152px] h-[152px] rounded-md bg-[var(--md-sys-color-surface-container-high)]"></div>
                <div class="h-3.5 w-3/4 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                <div class="h-2.5 w-1/2 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {:else}
      <div class="w-full space-y-1">
        <div
          class="-mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2 text-[11px] font-semibold uppercase tracking-wider"
          style="color: var(--md-sys-color-on-surface-variant);"
        >
          <span class="text-center">#</span>
          <span>Title</span>
          <span class="hidden md:block">Rilis</span>
          <span class="hidden md:block">Pemutaran</span>
          <span class="flex justify-end pr-1">
            <Icon name="clock" size={14} />
          </span>
          <span></span>
        </div>

        <div class="space-y-0.5" role="list">
          {#each listTracks as tr, idx (tr.id + '-' + idx)}
            {@const active = isTrackActive(tr)}
            {@const menuOpen = openActionTrackId === tr.id}
            <div
              role="listitem"
              class="group relative -mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2 rounded-lg transition-colors duration-150 {active
                ? 'bg-[var(--md-sys-color-state-active)]'
                : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
              style="color: var(--md-sys-color-on-surface);"
            >
              <div class="flex items-center justify-center">
                {#if active && isPlaying}
                  <EqIndicator class="group-hover:hidden" />
                  <button
                    type="button"
                    onclick={() => handleRowPlayClick(tr, active)}
                    aria-label="Jeda {tr.title}"
                    class="hidden group-hover:flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                  >
                    <Icon name="pause" weight="fill" size={14} />
                  </button>
                {:else if active && !isPlaying}
                  <button
                    type="button"
                    onclick={() => handleRowPlayClick(tr, active)}
                    aria-label="Putar {tr.title}"
                    class="flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                  >
                    <Icon name="play" weight="fill" size={14} />
                  </button>
                {:else}
                  <span class="text-xs font-medium tabular-nums group-hover:hidden opacity-65">
                    {idx + 1}
                  </span>
                  <button
                    type="button"
                    onclick={() => handleRowPlayClick(tr, active)}
                    aria-label="Putar {tr.title}"
                    class="hidden group-hover:flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                  >
                    <Icon name="play" weight="fill" size={14} />
                  </button>
                {/if}
              </div>

              <button
                type="button"
                onclick={() => handleRowPlayClick(tr, active)}
                class="min-w-0 pr-4 text-left cursor-pointer"
              >
                <div class="text-xs font-bold line-clamp-2 whitespace-normal leading-snug {active ? 'text-[var(--md-sys-color-primary)]' : ''}">
                  {tr.title}
                </div>
                <div class="text-[11px] truncate opacity-75 mt-0.5">
                  {tr.artistName}
                </div>
              </button>

              <div class="hidden md:block text-xs tabular-nums truncate opacity-75">
                {releaseDateLabel}
              </div>

              <div class="hidden md:block text-xs tabular-nums truncate opacity-75">
                {tr.globalPlays || '-'}
              </div>

              <span class="text-[11px] font-medium tabular-nums text-right opacity-75 pr-1">
                {formatDuration(tr.durationSec)}
              </span>

              <div class="relative flex items-center justify-end">
                <button
                  type="button"
                  onclick={(e) => {
                    e.stopPropagation();
                    if (menuOpen) {
                      openActionTrackId = null;
                      showPlaylistSubmenu = false;
                    } else {
                      openActionTrackId = tr.id;
                      showPlaylistSubmenu = false;
                    }
                  }}
                  aria-label="Aksi lagu"
                  class="w-8 h-8 flex items-center justify-center bg-transparent hover:bg-transparent cursor-pointer transition-colors {menuOpen
                    ? 'opacity-100 text-[var(--md-sys-color-on-surface)]'
                    : 'opacity-65 hover:opacity-100 text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)]'}"
                >
                  <Icon name="dots-three-vertical" weight="bold" size={18} />
                </button>

                {#if menuOpen}
                  <div
                    onclick={(e) => e.stopPropagation()}
                    role="menu"
                    tabindex="-1"
                    class="absolute right-0 top-9 z-40 w-60 p-1.5 rounded-xl shadow-2xl text-xs"
                    style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
                  >
                    {#if playlists.length > 0}
                      <div class="relative">
                        <button
                          type="button"
                          onclick={() => (showPlaylistSubmenu = !showPlaylistSubmenu)}
                          class="w-full flex items-center justify-between gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                        >
                          <span class="flex items-center gap-2.5">
                            <Icon name="plus" size={15} />
                            <span>Tambah ke playlist</span>
                          </span>
                          <Icon name="caret-right" size={12} />
                        </button>
                        {#if showPlaylistSubmenu}
                          <div class="mt-0.5 mb-1 pl-3 space-y-0.5 max-h-36 overflow-y-auto">
                            {#each playlists as pl (pl.id)}
                              <button
                                type="button"
                                onclick={() => {
                                  onAddToPlaylist(pl.id, tr.id);
                                  openActionTrackId = null;
                                  showPlaylistSubmenu = false;
                                }}
                                class="w-full text-left px-3 py-1.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] truncate cursor-pointer opacity-90"
                              >
                                {pl.name}
                              </button>
                            {/each}
                          </div>
                        {/if}
                      </div>
                    {/if}

                    <button
                      type="button"
                      onclick={() => {
                        onToggleFavoriteTrack(tr.id);
                        openActionTrackId = null;
                      }}
                      class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                    >
                      <Icon
                        name="heart"
                        weight={tr.isFavorite ? 'fill' : 'regular'}
                        size={15}
                        class={tr.isFavorite ? 'text-[var(--md-sys-color-tertiary)]' : ''}
                      />
                      <span>{tr.isFavorite ? 'Hapus dari Lagu yang Disukai' : 'Tambahkan ke Lagu yang Disukai'}</span>
                    </button>

                    {#if tr.isDownloaded && onRemoveDownload}
                      <button
                        type="button"
                        onclick={() => {
                          onRemoveDownload(tr.id);
                          openActionTrackId = null;
                        }}
                        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                      >
                        <Icon name="check-circle" weight="fill" size={15} class="text-[var(--md-sys-color-primary)]" />
                        <span>Hapus unduhan</span>
                      </button>
                    {:else if onDownloadTrack}
                      <button
                        type="button"
                        onclick={() => {
                          onDownloadTrack(tr);
                          openActionTrackId = null;
                        }}
                        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                      >
                        <Icon name="download" size={15} />
                        <span>Unduh untuk offline</span>
                      </button>
                    {/if}

                    {#if onAddToQueue}
                      <button
                        type="button"
                        onclick={() => {
                          onAddToQueue(tr);
                          openActionTrackId = null;
                        }}
                        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                      >
                        <Icon name="queue" size={15} />
                        <span>Tambahkan ke antrean</span>
                      </button>
                    {/if}

                    {#if tr.artistId > 0}
                      <div class="my-1 h-px opacity-15" style="background-color: var(--md-sys-color-on-surface);"></div>
                      <button
                        type="button"
                        onclick={() => {
                          onOpenArtist(tr.artistId);
                          openActionTrackId = null;
                        }}
                        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                      >
                        <Icon name="user" size={15} />
                        <span>Buka artis</span>
                      </button>
                    {/if}
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

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
