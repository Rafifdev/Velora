<script lang="ts">
  import { formatDuration, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { DownloadProgress, Playlist, Track } from '../lib/types';
  import EqIndicator from './EqIndicator.svelte';
  import Icon from './Icon.svelte';

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
    onToggleFavorite: (trackId: number) => void;
    onAddToPlaylist: (playlistId: number, trackId: number) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    showCover?: boolean;
    lang?: LangCode;
  }

  let {
    tracks,
    currentTrackId,
    currentExternalId = null,
    currentTrackTitle = null,
    currentTrackArtist = null,
    isPlaying,
    mediaBaseUrl,
    playlists,
    downloadStates = {},
    onPlayTrack,
    onTogglePlay,
    onToggleFavorite,
    onAddToPlaylist,
    onDownloadTrack,
    onRemoveDownload,
    onOpenTrackAlbum,
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
</script>

<div class="w-full">
  <div
    class="grid grid-cols-[28px_minmax(0,1fr)_auto] sm:grid-cols-[36px_1.4fr_1fr_92px_96px] gap-2 sm:gap-3 px-2 sm:px-3.5 py-2 text-[11px] font-bold uppercase tracking-wider"
    style="color: var(--md-sys-color-on-surface-variant);"
  >
    <div class="text-center">#</div>
    <div>{tr(lang, 'nav.titleCol')}</div>
    <div class="hidden sm:block">{tr(lang, 'nav.albumCol')}</div>
    <div class="hidden sm:block">{tr(lang, 'nav.sourceCol')}</div>
    <div class="text-right">{tr(lang, 'nav.timeCol')}</div>
  </div>

  <div class="space-y-0.5 mt-1">
    {#each tracks as item, idx (item.id)}
      {@const active = isTrackActive(item)}
      {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
      <div
        class="velora-track-row group grid grid-cols-[28px_minmax(0,1fr)_auto] sm:grid-cols-[36px_1.4fr_1fr_92px_96px] gap-2 sm:gap-3 px-2 sm:px-3.5 py-2 items-center rounded-lg transition-colors duration-150 {active
          ? 'bg-[var(--md-sys-color-state-hover)]'
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

        <div class="flex items-center gap-3 min-w-0 text-left">
          {#if showCover}
            <button
              type="button"
              onclick={() => handleRowPlayClick(item, active)}
              aria-label="{tr(lang, 'nav.play')} {item.title}"
              class="shrink-0 cursor-pointer"
            >
              {#if cover}
                <img src={cover} alt={item.title} loading="lazy" decoding="async" class="block w-10 h-10 rounded-md object-cover object-center shrink-0" />
              {:else}
                <div
                  class="w-10 h-10 rounded-md flex items-center justify-center shrink-0"
                  style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
                >
                  <Icon name="vinyl-record" weight="fill" size={18} style="color: var(--md-sys-color-on-primary-container); opacity: 0.6;" />
                </div>
              {/if}
            </button>
          {/if}
          <div class="min-w-0 flex-1">
            {#if onOpenTrackAlbum}
              <button
                type="button"
                onclick={() => onOpenTrackAlbum(item)}
                class="block max-w-full text-left text-xs font-bold truncate hover:underline cursor-pointer {active ? 'text-[var(--md-sys-color-primary)]' : ''}"
              >
                {item.title}
              </button>
            {:else}
              <button
                type="button"
                onclick={() => handleRowPlayClick(item, active)}
                class="block max-w-full text-left text-xs font-bold truncate cursor-pointer {active ? 'text-[var(--md-sys-color-primary)]' : ''}"
              >
                {item.title}
              </button>
            {/if}
            <div class="text-[11px] truncate opacity-75">{item.artistName}</div>
          </div>
        </div>

        <div class="hidden sm:block text-xs truncate opacity-70">
          {item.albumTitle}
        </div>

        <div class="hidden sm:block">
          <span
            class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold"
            style="background-color: rgba(255, 255, 255, 0.08); color: var(--md-sys-color-on-surface-variant);"
          >
            {item.isDownloaded ? tr(lang, 'nav.offlineSource') : item.sourceType === 'ytmusic' ? 'YT Music' : tr(lang, 'nav.localSource')}
          </span>
        </div>

        <div class="flex items-center justify-end gap-1.5 relative">
          {#if onDownloadTrack || onRemoveDownload}
            {@const dlState = downloadStates[item.id]}
            {@const isDlActive = dlState && (dlState.status === 'queued' || dlState.status === 'downloading')}
            {#if isDlActive}
              <span
                class="inline-flex items-center justify-center w-7 h-7 text-[10px] font-bold tabular-nums text-[var(--md-sys-color-primary)]"
                title="{tr(lang, 'nav.downloading')} ({Math.round(dlState.progress || 0)}%)"
              >
                {Math.round(dlState.progress || 0)}%
              </span>
            {:else if item.isDownloaded}
              <button
                type="button"
                onclick={() => onRemoveDownload?.(item.id)}
                title={tr(lang, 'nav.savedOfflineRemoveHint')}
                aria-label="{tr(lang, 'nav.removeDownload')} {item.title}"
                class="m3-icon-btn w-7 h-7 flex items-center justify-center transition cursor-pointer text-[var(--md-sys-color-primary)] opacity-90 hover:opacity-100"
              >
                <Icon name="check-circle" weight="fill" size={15} />
              </button>
            {:else if onDownloadTrack}
              <button
                type="button"
                onclick={() => onDownloadTrack(item)}
                title={tr(lang, 'nav.downloadOffline')}
                aria-label="{tr(lang, 'nav.downloadOffline')} {item.title}"
                class="m3-icon-btn w-7 h-7 flex items-center justify-center opacity-0 group-hover:opacity-100 transition cursor-pointer"
              >
                <Icon name="download" size={15} />
              </button>
            {/if}
          {/if}

          <button
            type="button"
            onclick={() => onToggleFavorite(item.id)}
            aria-label="{tr(lang, 'nav.likeSong')} {item.title}"
            class="m3-icon-btn w-7 h-7 flex items-center justify-center transition cursor-pointer {item.isFavorite
              ? 'opacity-100 text-[var(--md-sys-color-tertiary)]'
              : 'opacity-0 group-hover:opacity-100'}"
          >
            <Icon
              name="heart"
              weight={item.isFavorite ? 'fill' : 'regular'}
              size={15}
            />
          </button>

          <button
            type="button"
            onclick={() => (openMenuTrackId = openMenuTrackId === item.id ? null : item.id)}
            aria-label={tr(lang, 'nav.addToPlaylist')}
            class="m3-icon-btn w-7 h-7 flex items-center justify-center opacity-0 group-hover:opacity-100 transition cursor-pointer"
          >
            <Icon name="list-plus" size={15} />
          </button>

          <span class="text-[11px] tabular-nums w-9 text-right opacity-75">
            {formatDuration(item.durationSec)}
          </span>

          {#if openMenuTrackId === item.id}
            <div
              class="absolute right-0 top-9 z-30 w-52 p-1.5 rounded-lg text-xs shadow-xl"
              style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
            >
              <div class="px-2.5 py-1 text-[11px] font-semibold opacity-70">{tr(lang, 'nav.addToPlaylist')}</div>
              {#each playlists as pl (pl.id)}
                <button
                  type="button"
                  onclick={() => {
                    onAddToPlaylist(pl.id, item.id);
                    openMenuTrackId = null;
                  }}
                  class="w-full text-left px-2.5 py-1.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] truncate cursor-pointer"
                >
                  {pl.name} {pl.isLocal ? `(${tr(lang, 'nav.localSource')})` : ''}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/each}
  </div>
</div>
