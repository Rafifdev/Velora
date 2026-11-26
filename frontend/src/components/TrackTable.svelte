<script lang="ts">
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { DownloadProgress, Playlist, Track } from '../lib/types';
  import CoverImage from './CoverImage.svelte';
  import Icon from './Icon.svelte';
  import TrackActionMenu from './TrackActionMenu.svelte';

  interface Props {
    tracks: Track[];
    currentTrackId: number | null;
    currentExternalId?: string | null;
    currentTrackTitle?: string | null;
    currentTrackArtist?: string | null;
    isPlaying: boolean;
    mediaBaseUrl: string;
    accentColor: string;
    playlists: Playlist[];
    downloadStates?: Record<number, DownloadProgress>;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onTogglePlay?: () => void;
    onPlayNext?: (track: Track) => void;
    onAddToQueue?: (track: Track) => void;
    onToggleFavorite: (trackId: number) => void;
    onAddToPlaylist: (playlistId: number, trackId: number) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist?: (artistId: number) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onCopyLink?: (track: Track) => void;
    showCover?: boolean;
    lang?: LangCode;
  }

  let {
    tracks,
    currentTrackId,
    currentExternalId = null,
    currentTrackTitle = null,
    currentTrackArtist = null,
    mediaBaseUrl,
    playlists,
    onPlayTrack,
    onTogglePlay,
    onPlayNext,
    onAddToQueue,
    onToggleFavorite,
    onAddToPlaylist,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenTrackAlbum,
    onCopyLink,
    showCover = true,
    lang = 'en'
  }: Props = $props();

  let openMenuTrackId = $state<number | null>(null);

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
    if (active && onTogglePlay) {
      onTogglePlay();
    } else {
      onPlayTrack(tr, tracks);
    }
  }

  const activeRowTrackId = $derived.by<number | null>(() => {
    const matches = tracks.filter((t) => isTrackActive(t));
    if (matches.length === 0) return null;
    const exactId = matches.find((t) => currentTrackId !== null && t.id === currentTrackId);
    if (exactId) return exactId.id;
    const exactExt = currentExternalId ? matches.find((t) => t.externalId === currentExternalId) : undefined;
    if (exactExt) return exactExt.id;
    return matches[0].id;
  });
</script>

<div class="w-full space-y-1">
  <div
    class="-mx-2 hidden sm:grid grid-cols-[minmax(0,1fr)_52px_32px] md:grid-cols-[minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2 text-[11px] font-semibold uppercase tracking-wider"
    style="color: var(--md-sys-color-on-surface-variant);"
  >
    <span>{tr(lang, 'nav.titleCol')}</span>
    <span class="hidden md:block">{tr(lang, 'nav.playsCol')}</span>
    <span class="hidden md:block">{tr(lang, 'nav.albumCol')}</span>
    <span class="flex justify-end pr-1">
      <Icon name="clock" size={14} />
    </span>
    <span></span>
  </div>

  <div class="space-y-0.5" role="list">
    {#each tracks as item (item.id)}
      {@const active = activeRowTrackId === item.id}
      {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
      {@const menuOpen = openMenuTrackId === item.id}
      <div
        role="listitem"
        class="group relative -mx-2 grid grid-cols-[minmax(0,1fr)_52px_32px] md:grid-cols-[minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-3 sm:gap-4 px-3 py-2.5 sm:py-2 min-h-14 sm:min-h-0 rounded-lg transition-colors duration-150 {active
          ? 'bg-[var(--md-sys-color-state-active)]'
          : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
        style="color: var(--md-sys-color-on-surface);"
      >
        <button
          type="button"
          onclick={() => handleRowPlayClick(item, active)}
          class="flex items-center gap-3 min-w-0 text-left cursor-pointer"
        >
          {#if showCover}
            <div class="w-10 h-10 rounded-md overflow-hidden shrink-0">
              <CoverImage src={cover} fallbackIcon="vinyl-record" iconSize={16} isVideoThumb={isYTVideoThumb(cover)} />
            </div>
          {/if}
          <div class="min-w-0 pr-4">
            <div
              class="text-xs font-bold line-clamp-2 whitespace-normal leading-snug"
              style={active ? 'color: var(--md-sys-color-primary);' : ''}
            >
              {item.title}
            </div>
            <div class="flex items-center gap-1.5 text-[11px] truncate opacity-75 mt-0.5">
              {#if item.isDownloaded}
                <Icon name="check-circle" weight="fill" size={12} class="text-[var(--md-sys-color-primary)] shrink-0" />
              {/if}
              <span class="truncate">{item.artistName}</span>
            </div>
          </div>
        </button>

        <div class="hidden md:block text-xs tabular-nums truncate opacity-75">
          {item.globalPlays || '-'}
        </div>

        <div class="hidden md:block text-xs truncate opacity-75">
          {item.albumTitle || '-'}
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
          onToggleOpen={() => (openMenuTrackId = menuOpen ? null : item.id)}
          onClose={() => (openMenuTrackId = null)}
          {onPlayNext}
          {onAddToQueue}
          {onAddToPlaylist}
          {onToggleFavorite}
          {onDownloadTrack}
          {onRemoveDownload}
          {onOpenArtist}
          {onOpenTrackAlbum}
          {onCopyLink}
        />
      </div>
    {/each}
  </div>
</div>
