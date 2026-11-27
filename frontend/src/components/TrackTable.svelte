<script lang="ts">
  import { formatDuration, isJamRemote, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { DownloadProgress, Playlist, Track } from '../lib/types';
  import CoverImage from './CoverImage.svelte';
  import EqIndicator from './EqIndicator.svelte';
  import Icon from './Icon.svelte';
  import TrackActionMenu from './TrackActionMenu.svelte';

  interface Props {
    tracks: Track[];
    currentTrackId: number | null;
    currentExternalId?: string | null;
    currentTrackTitle?: string | null;
    currentTrackArtist?: string | null;
    currentHistoryEntryId?: number | null;
    isPlaying: boolean;
    trackLoading?: boolean;
    mediaBaseUrl: string;
    accentColor?: string;
    playlists: Playlist[];
    downloadStates?: Record<number, DownloadProgress>;
    excludePlaylistId?: number | null;
    fallbackCoverUrl?: string;
    activeTitleColor?: string;
    col2Header?: string;
    col3Header?: string;
    getCol2?: (track: Track, index: number) => string;
    getCol3?: (track: Track, index: number) => string;
    getCol3Title?: (track: Track, index: number) => string | undefined;
    draggableRows?: boolean;
    onReorder?: (fromIndex: number, toIndex: number) => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onTogglePlay?: () => void;
    onPlayNext?: (track: Track) => void;
    onAddToQueue?: (track: Track) => void;
    onToggleFavorite?: (trackId: number, track?: Track) => void;
    onAddToPlaylist?: (playlistId: number, trackId: number, track?: Track) => void;
    onCreatePlaylist?: (track?: Track) => void;
    onRemoveFromPlaylist?: (trackId: number) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist?: (artistId: number, track?: Track) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onCopyLink?: (track: Track) => void;
    showCover?: boolean;
    mobileShowCover?: boolean;
    isMobile?: boolean;
    lang?: LangCode;
  }

  let {
    tracks,
    currentTrackId,
    currentExternalId = null,
    currentTrackTitle = null,
    currentTrackArtist = null,
    currentHistoryEntryId = null,
    isPlaying,
    trackLoading = false,
    mediaBaseUrl,
    playlists,
    downloadStates = {},
    excludePlaylistId = null,
    fallbackCoverUrl = '',
    activeTitleColor = '',
    col2Header,
    col3Header,
    getCol2,
    getCol3,
    getCol3Title,
    draggableRows = false,
    onReorder,
    onPlayTrack,
    onTogglePlay,
    onPlayNext,
    onAddToQueue,
    onToggleFavorite,
    onAddToPlaylist,
    onCreatePlaylist,
    onRemoveFromPlaylist,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenTrackAlbum,
    onCopyLink,
    showCover = true,
    mobileShowCover,
    isMobile,
    lang = 'en'
  }: Props = $props();

  let openMenuRowKey = $state<string | null>(null);
  let dragFromIdx = $state<number | null>(null);
  let mediaIsMobile = $state(false);

  $effect(() => {
    if (!isJamRemote || isMobile !== undefined || typeof window === 'undefined') return;
    const mq = window.matchMedia('(max-width: 767px)');
    const sync = () => {
      mediaIsMobile = mq.matches;
    };
    sync();
    mq.addEventListener('change', sync);
    return () => mq.removeEventListener('change', sync);
  });

  const isJamMobile = $derived(isJamRemote && (isMobile ?? mediaIsMobile));

  function getRowKey(item: Track, idx: number): string {
    return item.historyEntryId ? `hist-${item.historyEntryId}` : `${item.id}-${idx}`;
  }

  function normTitle(s: string): string {
    return (s || '')
      .toLowerCase()
      .replace(/\s*[\(\[](?:official|audio|video|mv|lyrics?|hd|4k).*?[\)\]]/gi, '')
      .replace(/\s+/g, ' ')
      .trim();
  }

  function isTrackActive(tr: Track): boolean {
    if (tr.id > 0 && tr.id === currentTrackId) return true;
    if (currentExternalId && tr.externalId && tr.externalId === currentExternalId) return true;
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

  const activeRowKey = $derived.by<string | null>(() => {
    const matches: { track: Track; key: string }[] = [];
    for (let i = 0; i < tracks.length; i++) {
      const t = tracks[i];
      if (isTrackActive(t)) {
        matches.push({ track: t, key: getRowKey(t, i) });
      }
    }
    if (matches.length === 0) return null;
    if (currentHistoryEntryId != null) {
      const histMatch = matches.find((m) => m.track.historyEntryId === currentHistoryEntryId);
      if (histMatch) return histMatch.key;
    }
    const exactId = matches.find((m) => currentTrackId !== null && m.track.id === currentTrackId);
    if (exactId) return exactId.key;
    const exactExt = currentExternalId
      ? matches.find((m) => m.track.externalId === currentExternalId)
      : undefined;
    if (exactExt) return exactExt.key;
    return matches[0].key;
  });
</script>

<div class="w-full space-y-1">
  {#if isJamMobile}
    <div
      class="-mx-2 hidden sm:grid grid-cols-[minmax(0,1fr)_52px_32px] md:grid-cols-[minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2 text-[11px] font-semibold uppercase tracking-wider"
      style="color: var(--md-sys-color-on-surface-variant);"
    >
      <span>{tr(lang, 'nav.titleCol')}</span>
      <span class="hidden md:block">{col2Header ?? tr(lang, 'nav.playsCol')}</span>
      <span class="hidden md:block">{col3Header ?? tr(lang, 'nav.albumCol')}</span>
      <span class="flex justify-end pr-1">
        <Icon name="clock" size={14} />
      </span>
      <span></span>
    </div>

    <div class="space-y-0.5" role="list">
      {#each tracks as item, idx (getRowKey(item, idx))}
        {@const rowKey = getRowKey(item, idx)}
        {@const active = activeRowKey === rowKey}
        {@const cover = resolveCoverUrl(item.coverUrl || fallbackCoverUrl || '', mediaBaseUrl)}
        {@const menuOpen = openMenuRowKey === rowKey}
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
            {#if mobileShowCover ?? showCover}
              <div class="w-10 h-10 rounded-md overflow-hidden shrink-0">
                <CoverImage src={cover} fallbackIcon="vinyl-record" iconSize={16} isVideoThumb={isYTVideoThumb(cover)} />
              </div>
            {/if}
            <div class="min-w-0 pr-4">
              <div
                class="text-xs font-bold line-clamp-2 whitespace-normal leading-snug"
                style={active ? `color: ${activeTitleColor || 'var(--md-sys-color-primary)'};` : ''}
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
            {getCol2 ? getCol2(item, idx) : (item.globalPlays || '-')}
          </div>

          <div class="hidden md:block text-xs truncate opacity-75" title={getCol3Title?.(item, idx)}>
            {getCol3 ? getCol3(item, idx) : (item.albumTitle || '-')}
          </div>

          <span class="text-[11px] font-medium tabular-nums text-right opacity-75 pr-1">
            {formatDuration(item.durationSec)}
          </span>

          <TrackActionMenu
            track={item}
            open={menuOpen}
            {playlists}
            {excludePlaylistId}
            {lang}
            triggerClass="w-8 h-8 {menuOpen ? 'opacity-100' : 'opacity-65 hover:opacity-100'}"
            onToggleOpen={() => (openMenuRowKey = menuOpen ? null : rowKey)}
            onClose={() => (openMenuRowKey = null)}
            {onPlayNext}
            {onAddToQueue}
            {onAddToPlaylist}
            {onCreatePlaylist}
            {onRemoveFromPlaylist}
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
  {:else}
    <div
      class="-mx-2 hidden sm:grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2 text-[11px] font-semibold uppercase tracking-wider"
      style="color: var(--md-sys-color-on-surface-variant);"
    >
      <span class="text-center">#</span>
      <span>{tr(lang, 'nav.titleCol')}</span>
      <span class="hidden md:block">{col2Header ?? tr(lang, 'nav.playsCol')}</span>
      <span class="hidden md:block">{col3Header ?? tr(lang, 'nav.albumCol')}</span>
      <span class="flex justify-end pr-1">
        <Icon name="clock" size={14} />
      </span>
      <span></span>
    </div>

    <div class="space-y-0.5" role="list">
      {#each tracks as item, idx (getRowKey(item, idx))}
        {@const rowKey = getRowKey(item, idx)}
        {@const active = activeRowKey === rowKey}
        {@const cover = resolveCoverUrl(item.coverUrl || fallbackCoverUrl || '', mediaBaseUrl)}
        {@const menuOpen = openMenuRowKey === rowKey}
        {@const dlState = item.id > 0 ? downloadStates[item.id] : undefined}
        {@const isDlActive = dlState && (dlState.status === 'queued' || dlState.status === 'downloading')}
        <div
          role="listitem"
          draggable={draggableRows}
          ondragstart={() => {
            if (draggableRows) dragFromIdx = idx;
          }}
          ondragover={(e) => {
            if (draggableRows) e.preventDefault();
          }}
          ondrop={() => {
            if (draggableRows && dragFromIdx !== null && dragFromIdx !== idx) {
              onReorder?.(dragFromIdx, idx);
            }
            dragFromIdx = null;
          }}
          class="group relative -mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-3 sm:gap-4 px-3 py-2 rounded-lg transition-colors duration-150 {active
            ? 'bg-[var(--md-sys-color-state-active)]'
            : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
          style="color: var(--md-sys-color-on-surface);"
        >
          <div class="flex items-center justify-center">
            {#if active && trackLoading}
              <Icon name="circle-notch" weight="bold" size={14} class="animate-spin text-[var(--md-sys-color-primary)]" />
            {:else if active && isPlaying}
              <EqIndicator class="group-hover:hidden" />
              <button
                type="button"
                onclick={() => handleRowPlayClick(item, active)}
                aria-label="{tr(lang, 'nav.pause')} {item.title}"
                class="hidden group-hover:flex w-7 h-7 items-center justify-center cursor-pointer text-[var(--md-sys-color-primary)]"
              >
                <Icon name="pause" weight="fill" size={14} />
              </button>
            {:else if active}
              <button
                type="button"
                onclick={() => handleRowPlayClick(item, active)}
                aria-label="{tr(lang, 'nav.play')} {item.title}"
                class="flex w-7 h-7 items-center justify-center cursor-pointer text-[var(--md-sys-color-primary)]"
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
                class="hidden group-hover:flex w-7 h-7 items-center justify-center cursor-pointer text-[var(--md-sys-color-on-surface)]"
              >
                <Icon name="play" weight="fill" size={14} />
              </button>
            {/if}
          </div>

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
                style={active ? `color: ${activeTitleColor || 'var(--md-sys-color-primary)'};` : ''}
              >
                {item.title}
              </div>
              <div class="flex items-center gap-1.5 text-[11px] truncate opacity-75 mt-0.5">
                {#if isDlActive}
                  <span class="text-[10px] font-bold tabular-nums text-[var(--md-sys-color-primary)] shrink-0">
                    {Math.round(dlState.progress || 0)}%
                  </span>
                {:else if item.isDownloaded}
                  <Icon name="check-circle" weight="fill" size={12} class="text-[var(--md-sys-color-primary)] shrink-0" />
                {/if}
                <span class="truncate">{item.artistName}</span>
              </div>
            </div>
          </button>

          <div class="hidden md:block min-w-0 text-xs tabular-nums truncate opacity-75">
            {getCol2 ? getCol2(item, idx) : (item.globalPlays || '-')}
          </div>

          {#if !getCol3 && onOpenTrackAlbum && item.albumTitle}
            <div class="hidden md:block min-w-0 text-xs opacity-75">
              <button
                type="button"
                onclick={() => onOpenTrackAlbum(item)}
                class="max-w-full overflow-hidden text-ellipsis whitespace-nowrap hover:underline decoration-1 underline-offset-2 cursor-pointer text-left"
              >
                {item.albumTitle}
              </button>
            </div>
          {:else}
            <div class="hidden md:block min-w-0 text-xs tabular-nums truncate opacity-75" title={getCol3Title?.(item, idx)}>
              {getCol3 ? getCol3(item, idx) : (item.albumTitle || '-')}
            </div>
          {/if}

          <span class="text-[11px] font-medium tabular-nums text-right opacity-75 pr-1">
            {formatDuration(item.durationSec)}
          </span>

          <TrackActionMenu
            track={item}
            open={menuOpen}
            {playlists}
            {excludePlaylistId}
            {lang}
            triggerClass="w-8 h-8 {menuOpen ? 'opacity-100' : 'opacity-65 hover:opacity-100'}"
            onToggleOpen={() => (openMenuRowKey = menuOpen ? null : rowKey)}
            onClose={() => (openMenuRowKey = null)}
            {onPlayNext}
            {onAddToQueue}
            {onAddToPlaylist}
            {onCreatePlaylist}
            {onRemoveFromPlaylist}
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
  {/if}
</div>
