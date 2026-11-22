<script lang="ts">
  import { fade } from 'svelte/transition';
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { Album, Playlist, Track, YTMusicAlbumView, YTMusicCard } from '../lib/types';
  import CoverImage from './CoverImage.svelte';
  import EqIndicator from './EqIndicator.svelte';
  import Icon from './Icon.svelte';
  import TrackActionMenu from './TrackActionMenu.svelte';

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
    lang?: LangCode;
    onPlayTrack: (tr: Track, list?: Track[]) => void;
    onTogglePlay: () => void;
    onToggleFavoriteTrack: (id: number) => void;
    onToggleFavoriteAlbum: (id: number) => void;
    onAddToPlaylist: (playlistId: number, trackId: number) => void;
    onPlayNext?: (tr: Track) => void;
    onAddToQueue?: (tr: Track) => void;
    onCopyLink?: (tr: Track) => void;
    onDownloadTrack?: (tr: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist: (artistId: number) => void;
    onOpenAlbum: (albumId: number) => void;
    shuffle?: boolean;
    queueSourceKey?: string | null;
    onToggleShuffle?: () => void;
    onPlayShuffled?: (list: Track[], sourceTitle?: string, sourceKey?: string) => void;
    ytAlbum?: YTMusicAlbumView | null;
    ytAlbumLoading?: boolean;
    routeKind?: 'album' | 'playlist';
    recommendedPlaylists?: YTMusicCard[];
    onOpenYTAlbum?: (card: YTMusicCard) => void;
    onToggleFavoriteYTMusicAlbum?: (payload: {
      browseId: string;
      title: string;
      artist: string;
      coverUrl: string;
      releaseYear: number;
    }) => void;
    onToggleSaveYTMusicPlaylist?: (payload: {
      browseId: string;
      title: string;
      subtitle: string;
      coverUrl: string;
      trackIds: number[];
    }) => Promise<void> | void;
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
    lang = 'en',
    onPlayTrack,
    onTogglePlay,
    onToggleFavoriteTrack,
    onToggleFavoriteAlbum,
    onAddToPlaylist,
    onPlayNext,
    onAddToQueue,
    onCopyLink,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenAlbum,
    shuffle = false,
    queueSourceKey = null,
    onToggleShuffle,
    onPlayShuffled,
    ytAlbum = null,
    ytAlbumLoading = false,
    routeKind,
    recommendedPlaylists = [],
    onOpenYTAlbum,
    onToggleFavoriteYTMusicAlbum,
    onToggleSaveYTMusicPlaylist
  }: Props = $props();

  const isPlaylistView = $derived(ytAlbum?.kind === 'playlist' || (!album && routeKind === 'playlist'));
  const displayTitle = $derived(ytAlbum?.title || album?.title || track?.albumTitle || track?.title || 'Album');
  const displayArtist = $derived(ytAlbum?.artist || album?.artistName || track?.artistName || 'Unknown Artist');
  const displayArtistId = $derived(ytAlbum ? 0 : track?.artistId || 0);
  const rawCover = $derived(ytAlbum?.coverUrl || album?.coverUrl || track?.coverUrl || '');
  const cover = $derived(resolveCoverUrl(rawCover, mediaBaseUrl));

  let savingPlaylist = $state(false);
  const isYTSavedPlaylist = $derived.by(() => {
    if (!ytAlbum || !isPlaylistView) return false;
    const bid = (ytAlbum.browseId || '').trim();
    const tLow = displayTitle.trim().toLowerCase();
    return playlists.some(
      (p) =>
        !p.isLocal &&
        ((bid !== '' && p.externalId === bid) || p.name.trim().toLowerCase() === tLow)
    );
  });

  const isYTFavAlbum = $derived.by(() => {
    if (!ytAlbum || isPlaylistView) return false;
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

  const isViewQueueActive = $derived.by(() => {
    const viewKey = ytAlbum?.browseId || (album ? `local-album:${album.id}` : '');
    if (queueSourceKey && viewKey && queueSourceKey === viewKey) return true;
    return currentTrackId !== null && listTracks.some((t) => t.id === currentTrackId);
  });

  function playShuffled(list: Track[]) {
    if (list.length === 0) return;
    if (shuffle && isViewQueueActive && onToggleShuffle) {
      onToggleShuffle();
      return;
    }
    if (onPlayShuffled) {
      onPlayShuffled(list, displayTitle, ytAlbum?.browseId || (album ? `local-album:${album.id}` : ''));
      return;
    }
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

    if (isPlaylistView) {
      const currentBrowse = (ytAlbum?.browseId || '').trim().toLowerCase();
      const currentTitleLow = displayTitle.trim().toLowerCase();
      if (currentTitleLow) seenTitles.add(currentTitleLow);
      const seenKeys = new Set<string>();
      if (currentBrowse) seenKeys.add(currentBrowse);
      for (const yt of [...(ytAlbum?.artistAlbums ?? []), ...recommendedPlaylists]) {
        const k = (yt.browseId || yt.playlistId || yt.id || '').trim().toLowerCase();
        const titleLow = yt.title.trim().toLowerCase();
        if (!titleLow || seenTitles.has(titleLow) || (k && seenKeys.has(k))) continue;
        seenTitles.add(titleLow);
        if (k) seenKeys.add(k);
        cards.push({
          kind: 'yt-album',
          key: `yt-pl-${k || titleLow}`,
          title: yt.title,
          subtitle: yt.subtitle || 'Playlist',
          cover: yt.coverUrl,
          ytCard: { ...yt, kind: yt.kind || 'playlist' }
        });
        if (cards.length >= 12) break;
      }
      return cards;
    }

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
        <CoverImage
          src={cover}
          fallbackIcon={isPlaylistView ? 'playlist' : 'vinyl-record'}
          iconSize={56}
          isVideoThumb={isYTVideoThumb(cover)}
          loading="eager"
        />
      </div>

      <div class="min-w-0 flex-1 space-y-2">
        <div class="flex flex-wrap items-center gap-2 text-xs font-semibold">
          <span
            class="px-2.5 py-0.5 rounded-full"
            style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
          >
            {isPlaylistView ? tr(lang, 'nav.playlist') : tr(lang, 'nav.album')}
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
          class="{displayTitle.length > 44
            ? 'text-lg sm:text-xl lg:text-[24px]'
            : displayTitle.length > 24
              ? 'text-xl sm:text-2xl lg:text-[28px]'
              : 'text-2xl sm:text-3xl lg:text-[38px]'} font-bold tracking-tight leading-[1.18] line-clamp-2"
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
          <span class="tabular-nums">{listTracks.length} {tr(lang, 'nav.songs').toLowerCase()}</span>
          {#if totalDurationSec > 0}
            <span>•</span>
            <span class="tabular-nums">{formatDuration(totalDurationSec)}</span>
          {/if}
        </div>

        <div class="pt-1 flex flex-wrap items-center gap-3">
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
            title={isAlbumCurrentlyPlaying ? tr(lang, 'nav.pause') : isPlaylistView ? 'Putar Playlist' : 'Putar Album'}
            aria-label={isAlbumCurrentlyPlaying ? tr(lang, 'nav.pause') : isPlaylistView ? 'Putar Playlist' : 'Putar Album'}
            class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110 disabled:opacity-40"
            style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
          >
            <Icon name={isAlbumCurrentlyPlaying ? 'pause' : 'play'} weight="fill" size={20} />
          </button>

          <button
            type="button"
            disabled={listTracks.length === 0}
            onclick={() => playShuffled(listTracks)}
            title={tr(lang, 'nav.shuffle')}
            aria-label={tr(lang, 'nav.shuffle')}
            aria-pressed={shuffle}
            class="w-11 h-11 rounded-full relative inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110 disabled:opacity-40"
            style={shuffle
              ? 'background-color: color-mix(in srgb, var(--md-sys-color-primary) 22%, var(--md-sys-color-surface-container-high)); color: var(--md-sys-color-primary);'
              : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
          >
            <Icon name="shuffle" weight={shuffle ? 'bold' : 'regular'} size={18} />
            {#if shuffle}
              <span
                class="absolute bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full"
                style="background-color: var(--md-sys-color-primary);"
              ></span>
            {/if}
          </button>

          {#if album}
            <button
              type="button"
              onclick={() => onToggleFavoriteAlbum(album.id)}
              class="velora-heart-btn w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer hover:brightness-110 {album.isFavorite ? 'is-loved' : ''}"
              style="background-color: var(--md-sys-color-surface-container-high); color: {album.isFavorite
                ? 'var(--md-sys-color-tertiary)'
                : 'var(--md-sys-color-on-surface)'};"
              title={album.isFavorite ? 'Hapus Album dari Favorit' : 'Sukai Album'}
              aria-label={album.isFavorite ? 'Hapus Album dari Favorit' : 'Sukai Album'}
            >
              <Icon name="heart" weight={album.isFavorite ? 'fill' : 'regular'} size={18} />
            </button>
          {:else if ytAlbum && isPlaylistView && onToggleSaveYTMusicPlaylist}
            <button
              type="button"
              disabled={savingPlaylist}
              onclick={async () => {
                if (savingPlaylist) return;
                savingPlaylist = true;
                try {
                  await onToggleSaveYTMusicPlaylist({
                    browseId: ytAlbum.browseId || '',
                    title: displayTitle,
                    subtitle: ytAlbum.subtitle || displayArtist || 'YouTube Music',
                    coverUrl: rawCover,
                    trackIds: listTracks.map((t) => t.id).filter((id) => id > 0)
                  });
                } finally {
                  savingPlaylist = false;
                }
              }}
              class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110 disabled:opacity-60"
              style={isYTSavedPlaylist
                ? 'background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);'
                : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
              title={isYTSavedPlaylist ? tr(lang, 'nav.savedPlaylist') : tr(lang, 'nav.savePlaylist')}
              aria-label={isYTSavedPlaylist ? tr(lang, 'nav.savedPlaylist') : tr(lang, 'nav.savePlaylist')}
            >
              <Icon
                name={isYTSavedPlaylist ? 'check' : 'bookmark-simple'}
                weight={isYTSavedPlaylist ? 'bold' : 'regular'}
                size={18}
              />
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
              class="velora-heart-btn w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer hover:brightness-110 {isYTFavAlbum ? 'is-loved' : ''}"
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
              class="velora-heart-btn w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer hover:brightness-110 {track.isFavorite ? 'is-loved' : ''}"
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
          <div>{tr(lang, 'nav.titleCol')}</div>
          <div class="hidden md:block">{tr(lang, 'nav.releaseCol')}</div>
          <div class="hidden md:block">{tr(lang, 'nav.playsCol')}</div>
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
          <span>{tr(lang, 'nav.titleCol')}</span>
          <span class="hidden md:block">{tr(lang, 'nav.releaseCol')}</span>
          <span class="hidden md:block">{tr(lang, 'nav.playsCol')}</span>
          <span class="flex justify-end pr-1">
            <Icon name="clock" size={14} />
          </span>
          <span></span>
        </div>

        <div class="space-y-0.5" role="list">
          {#each listTracks as item, idx (item.id + '-' + idx)}
            {@const active = isTrackActive(item)}
            {@const menuOpen = openActionTrackId === item.id}
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
                    onclick={() => handleRowPlayClick(item, active)}
                    aria-label="{tr(lang, 'nav.pause')} {item.title}"
                    class="hidden group-hover:flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                  >
                    <Icon name="pause" weight="fill" size={14} />
                  </button>
                {:else if active && !isPlaying}
                  <button
                    type="button"
                    onclick={() => handleRowPlayClick(item, active)}
                    aria-label="{tr(lang, 'nav.play')} {item.title}"
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
                    onclick={() => handleRowPlayClick(item, active)}
                    aria-label="{tr(lang, 'nav.play')} {item.title}"
                    class="hidden group-hover:flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                  >
                    <Icon name="play" weight="fill" size={14} />
                  </button>
                {/if}
              </div>

              <button
                type="button"
                onclick={() => handleRowPlayClick(item, active)}
                class="min-w-0 pr-4 text-left cursor-pointer"
              >
                <div class="text-xs font-bold line-clamp-2 whitespace-normal leading-snug {active ? 'text-[var(--md-sys-color-primary)]' : ''}">
                  {item.title}
                </div>
                <div class="text-[11px] truncate opacity-75 mt-0.5">
                  {item.artistName}
                </div>
              </button>

              <div class="hidden md:block text-xs tabular-nums truncate opacity-75">
                {releaseDateLabel}
              </div>

              <div class="hidden md:block text-xs tabular-nums truncate opacity-75">
                {item.globalPlays || '-'}
              </div>

              <span class="text-[11px] font-medium tabular-nums text-right opacity-75 pr-1">
                {formatDuration(item.durationSec)}
              </span>

              <TrackActionMenu
                track={item}
                open={menuOpen}
                {playlists}
                {lang}
                triggerClass="w-8 h-8 {menuOpen ? 'opacity-100' : 'opacity-65 hover:opacity-100'}"
                onToggleOpen={() => (openActionTrackId = menuOpen ? null : item.id)}
                onClose={() => (openActionTrackId = null)}
                {onPlayNext}
                {onAddToQueue}
                {onAddToPlaylist}
                onToggleFavorite={onToggleFavoriteTrack}
                {onDownloadTrack}
                {onRemoveDownload}
                {onOpenArtist}
                {onCopyLink}
              />
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  {#if artistReleases.length > 0}
    <section class="px-6 sm:px-10 pb-10">
      <h2 class="text-lg font-bold mb-4" style="color: var(--md-sys-color-on-surface);">
        {isPlaylistView
          ? tr(lang, 'nav.recommendedPlaylists')
          : `${tr(lang, 'nav.moreFromArtist')} ${displayArtist}`}
      </h2>
      <div
        use:shelfEdgeFade
        class="flex items-start gap-4 overflow-x-auto no-scrollbar velora-shelf-scroll pb-2"
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
            class="w-36 sm:w-40 shrink-0 flex flex-col items-stretch justify-start self-start text-left group cursor-pointer"
          >
            <div
              class="w-full aspect-square rounded-lg overflow-hidden mb-2.5 shadow-md"
              style="background-color: var(--md-sys-color-surface-container-highest);"
            >
              <CoverImage
                src={card.cover}
                fallbackIcon={card.kind === 'yt-album' && card.ytCard.kind === 'playlist' ? 'playlist' : 'music-note'}
                iconSize={32}
                imgClass="block w-full h-full object-cover object-center transition-transform duration-300 group-hover:scale-[1.03]"
                isVideoThumb={isYTVideoThumb(card.cover)}
              />
            </div>
            <div class="text-[13px] font-bold leading-snug line-clamp-2 break-words whitespace-normal" style="color: var(--md-sys-color-on-surface);">
              {card.title}
            </div>
            <div class="text-[11px] leading-snug truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
              {card.subtitle}
            </div>
          </button>
        {/each}
      </div>
    </section>
  {/if}
</div>
