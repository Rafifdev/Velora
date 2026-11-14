<script lang="ts">
  import { formatDuration, resolveCoverUrl } from '../lib/api';
  import type { Playlist, Track } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    tracks: Track[];
    currentTrackId: number | null;
    isPlaying: boolean;
    mediaBaseUrl: string;
    accentColor: string;
    playlists: Playlist[];
    onPlayTrack: (track: Track, list: Track[]) => void;
    onToggleFavorite: (trackId: number) => void;
    onAddToPlaylist: (playlistId: number, trackId: number) => void;
  }

  let {
    tracks,
    currentTrackId,
    isPlaying,
    mediaBaseUrl,
    playlists,
    onPlayTrack,
    onToggleFavorite,
    onAddToPlaylist
  }: Props = $props();

  let openMenuTrackId = $state<number | null>(null);
</script>

<div class="w-full">
  <div
    class="grid grid-cols-[36px_1.4fr_1fr_92px_96px] gap-3 px-3.5 py-2 text-[11px] font-bold uppercase tracking-wider"
    style="color: var(--md-sys-color-on-surface-variant);"
  >
    <div class="text-center">#</div>
    <div>Title</div>
    <div class="hidden sm:block">Album</div>
    <div>Source</div>
    <div class="text-right">Time</div>
  </div>

  <div class="space-y-0.5 mt-1">
    {#each tracks as tr, idx (tr.id)}
      {@const active = tr.id === currentTrackId}
      {@const cover = resolveCoverUrl(tr.coverUrl, mediaBaseUrl)}
      <div
        class="group grid grid-cols-[36px_1.4fr_1fr_92px_96px] gap-3 px-3.5 py-2 items-center rounded-lg transition-colors duration-150 {active
          ? 'bg-[var(--md-sys-color-state-active)]'
          : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
        style="color: var(--md-sys-color-on-surface);"
      >
        <div class="flex items-center justify-center">
          {#if active && isPlaying}
            <button
              type="button"
              onclick={() => onPlayTrack(tr, tracks)}
              aria-label="Putar {tr.title}"
              class="flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
            >
              <Icon name="speaker-high" weight="fill" size={15} />
            </button>
          {:else}
            <span class="text-xs font-medium tabular-nums group-hover:hidden opacity-65 {active ? 'text-[var(--md-sys-color-primary)]' : ''}">
              {idx + 1}
            </span>
            <button
              type="button"
              onclick={() => onPlayTrack(tr, tracks)}
              aria-label="Putar {tr.title}"
              class="hidden group-hover:flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
            >
              <Icon name="play" weight="fill" size={14} />
            </button>
          {/if}
        </div>

        <button
          type="button"
          onclick={() => onPlayTrack(tr, tracks)}
          class="flex items-center gap-3 min-w-0 text-left cursor-pointer"
        >
          {#if cover}
            <img src={cover} alt={tr.title} class="block w-10 h-10 rounded-md object-cover object-center shrink-0" />
          {:else}
            <div
              class="w-10 h-10 rounded-md flex items-center justify-center shrink-0"
              style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
            >
              <Icon name="vinyl-record" weight="fill" size={18} style="color: var(--md-sys-color-on-primary-container); opacity: 0.6;" />
            </div>
          {/if}
          <div class="min-w-0">
            <div class="text-xs font-bold truncate {active ? 'text-[var(--md-sys-color-primary)]' : ''}">{tr.title}</div>
            <div class="text-[11px] truncate opacity-75">{tr.artistName}</div>
          </div>
        </button>

        <div class="hidden sm:block text-xs truncate opacity-70">
          {tr.albumTitle}
        </div>

        <div>
          <span
            class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold"
            style="background-color: rgba(255, 255, 255, 0.08); color: var(--md-sys-color-on-surface-variant);"
          >
            {tr.sourceType === 'ytmusic' ? 'YT Music' : tr.format}
          </span>
        </div>

        <div class="flex items-center justify-end gap-2 relative">
          <button
            type="button"
            onclick={() => onToggleFavorite(tr.id)}
            aria-label="Sukai {tr.title}"
            class="m3-icon-btn w-7 h-7 flex items-center justify-center transition cursor-pointer {tr.isFavorite
              ? 'opacity-100 text-[var(--md-sys-color-tertiary)]'
              : 'opacity-0 group-hover:opacity-100'}"
          >
            <Icon
              name="heart"
              weight={tr.isFavorite ? 'fill' : 'regular'}
              size={15}
            />
          </button>

          <button
            type="button"
            onclick={() => (openMenuTrackId = openMenuTrackId === tr.id ? null : tr.id)}
            aria-label="Tambah ke Playlist"
            class="m3-icon-btn w-7 h-7 flex items-center justify-center opacity-0 group-hover:opacity-100 transition cursor-pointer"
          >
            <Icon name="list-plus" size={15} />
          </button>

          <span class="text-[11px] tabular-nums w-9 text-right opacity-75">
            {formatDuration(tr.durationSec)}
          </span>

          {#if openMenuTrackId === tr.id}
            <div
              class="absolute right-0 top-9 z-30 w-52 p-1.5 rounded-lg text-xs shadow-xl"
              style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
            >
              <div class="px-2.5 py-1 text-[11px] font-semibold opacity-70">Tambahkan ke Playlist</div>
              {#each playlists as pl (pl.id)}
                <button
                  type="button"
                  onclick={() => {
                    onAddToPlaylist(pl.id, tr.id);
                    openMenuTrackId = null;
                  }}
                  class="w-full text-left px-2.5 py-1.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] truncate cursor-pointer"
                >
                  {pl.name} {pl.isLocal ? '(Local)' : ''}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/each}
  </div>
</div>
