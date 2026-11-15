<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import type { Track } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    currentTrack: Track | null;
    isPlaying: boolean;
    positionSec: number;
    durationSec: number;
    volume: number;
    muted: boolean;
    shuffle: boolean;
    repeatMode: 'off' | 'all' | 'one';
    rightCollapsed: boolean;
    rightPanelMode?: 'now-playing' | 'queue';
    isLyricsOpen?: boolean;
    seekDeltaSec?: number;
    seekDeltaSeq?: number;
    mediaBaseUrl: string;
    accentColor: string;
    onTogglePlay: () => void;
    onPrev: () => void;
    onNext: () => void;
    onSeek: (sec: number) => void;
    onVolumeChange: (vol: number) => void;
    onToggleMute: () => void;
    onToggleShuffle: () => void;
    onCycleRepeat: () => void;
    onToggleFavorite: (trackId: number) => void;
    onFocusRightCard: (card: 'album' | 'lyrics' | 'queue') => void;
    onOpenTrackDetail?: (track: Track) => void;
    onToggleLyrics?: () => void;
  }

  let {
    currentTrack,
    isPlaying,
    positionSec,
    durationSec,
    volume,
    muted,
    shuffle,
    repeatMode,
    rightCollapsed,
    rightPanelMode = 'now-playing',
    isLyricsOpen = false,
    seekDeltaSec = 0,
    seekDeltaSeq = 0,
    mediaBaseUrl,
    onTogglePlay,
    onPrev,
    onNext,
    onSeek,
    onVolumeChange,
    onToggleMute,
    onToggleShuffle,
    onCycleRepeat,
    onToggleFavorite,
    onFocusRightCard,
    onOpenTrackDetail,
    onToggleLyrics
  }: Props = $props();

  const effectiveDuration = $derived(
    durationSec > 0 ? durationSec : currentTrack?.durationSec || 0
  );
  const progressPct = $derived(
    effectiveDuration > 0 ? Math.min(100, (positionSec / effectiveDuration) * 100) : 0
  );
  const remainingSec = $derived(Math.max(0, effectiveDuration - positionSec));
  const displayVolume = $derived(muted ? 0 : volume);
</script>

<!-- Full-Width Flat Borderless Media Player Bar -->
<footer
  class="w-full shrink-0 select-none rounded-none overflow-visible relative z-20 transition-colors duration-300"
  style="background-color: var(--md-sys-color-surface-container-lowest);"
>
  <!-- Top Edge Full-Width Seekbar + Timestamps -->
  <div class="relative w-full">
    <input
      type="range"
      min="0"
      max={Math.max(1, effectiveDuration)}
      step="0.5"
      value={Math.min(positionSec, Math.max(1, effectiveDuration))}
      oninput={(e) => onSeek(Number((e.target as HTMLInputElement).value))}
      aria-label="Seekbar durasi lagu"
      class="velora-seekbar w-full block"
      style="background: linear-gradient(to right, var(--md-sys-color-primary) {progressPct}%, var(--md-sys-color-surface-container-highest) {progressPct}%);"
    />
    <div
      class="flex items-center justify-between px-6 pt-1 text-[11px] font-medium tabular-nums"
      style="color: var(--md-sys-color-on-surface-variant);"
    >
      <div class="flex items-center gap-1.5">
        <span>{formatDuration(positionSec)}</span>
        {#if seekDeltaSec < 0}
          <span transition:fade={{ duration: 150 }} class="tabular-nums">
            {seekDeltaSec}s
          </span>
        {/if}
      </div>
      <div class="flex items-center gap-1.5">
        {#if seekDeltaSec > 0}
          <span transition:fade={{ duration: 150 }} class="tabular-nums">
            +{seekDeltaSec}s
          </span>
        {/if}
        <span>-{formatDuration(remainingSec)}</span>
      </div>
    </div>
  </div>

  <!-- Main Player Controls Row -->
  <div class="h-[68px] px-6 pb-2 flex items-center justify-between gap-4">
    <!-- Left: Track Cover & Info (1:1 Reference Style) -->
    <div class="w-[30%] min-w-[180px] flex items-center gap-3.5">
      {#if currentTrack}
        {@const cover = resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl)}
        <button
          type="button"
          onclick={() => onFocusRightCard('album')}
          class="relative w-14 h-14 p-0 rounded-md overflow-clip shrink-0 cursor-pointer"
          title="Buka Panel Kanan"
        >
          {#if cover}
            <img
              src={cover}
              alt={currentTrack.title}
              class="block w-full h-full object-cover object-center"
              class:velora-yt-video-thumb={isYTVideoThumb(cover)}
            />
          {:else}
            <div
              class="w-full h-full flex items-center justify-center"
              style="background-color: var(--md-sys-color-surface-container-highest);"
            >
              <Icon name="music-note" size={20} />
            </div>
          {/if}
        </button>

        <div class="min-w-0 flex-1 flex flex-col justify-center">
          <button
            type="button"
            onclick={() => onOpenTrackDetail?.(currentTrack)}
            class="text-left text-[14px] font-bold leading-snug truncate hover:underline cursor-pointer"
            style="color: var(--md-sys-color-on-surface);"
            title="Lihat Detail Musik"
          >
            {currentTrack.title}
          </button>
          <div
            class="text-[12px] font-normal leading-snug truncate mt-0.5"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            {currentTrack.artistName}
          </div>
        </div>
      {:else}
        <div
          class="w-14 h-14 p-0 rounded-md flex items-center justify-center shrink-0"
          style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface-variant);"
        >
          <Icon name="disc" size={22} />
        </div>
        <div class="min-w-0 flex flex-col justify-center">
          <div class="text-[14px] font-bold leading-snug truncate" style="color: var(--md-sys-color-on-surface-variant);">
            Belum ada lagu diputar
          </div>
          <div class="text-[12px] leading-snug opacity-70 truncate mt-0.5">Pilih lagu lokal atau YouTube Music</div>
        </div>
      {/if}
    </div>

    <!-- Center: Transport Controls + M3 Primary Container Play FAB -->
    <div class="flex items-center justify-center gap-3">
      <button
        type="button"
        disabled={!currentTrack}
        onclick={() => currentTrack && onToggleFavorite(currentTrack.id)}
        aria-label="Sukai lagu"
        class="m3-icon-btn w-9 h-9 flex items-center justify-center cursor-pointer disabled:opacity-35"
      >
        <Icon
          name="heart"
          weight={currentTrack?.isFavorite ? 'fill' : 'regular'}
          size={19}
          style={currentTrack?.isFavorite ? 'color: var(--md-sys-color-primary);' : ''}
        />
      </button>

      <button
        type="button"
        onclick={onToggleShuffle}
        aria-label="Shuffle"
        class="m3-icon-btn w-9 h-9 flex items-center justify-center relative cursor-pointer"
        style={shuffle ? 'color: var(--md-sys-color-primary);' : ''}
      >
        <Icon name="shuffle" weight={shuffle ? 'bold' : 'regular'} size={18} />
        {#if shuffle}
          <span
            class="absolute bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full"
            style="background-color: var(--md-sys-color-primary);"
          ></span>
        {/if}
      </button>

      <button
        type="button"
        onclick={onPrev}
        aria-label="Previous"
        class="m3-icon-btn w-9 h-9 flex items-center justify-center cursor-pointer"
        style="color: var(--md-sys-color-on-surface);"
      >
        <Icon name="skip-back" weight="fill" size={19} />
      </button>

      <button
        type="button"
        onclick={onTogglePlay}
        aria-label={isPlaying ? 'Pause' : 'Play'}
        class="m3-fab w-12 h-12 flex items-center justify-center cursor-pointer"
      >
        <Icon name={isPlaying ? 'pause' : 'play'} weight="fill" size={21} />
      </button>

      <button
        type="button"
        onclick={onNext}
        aria-label="Next"
        class="m3-icon-btn w-9 h-9 flex items-center justify-center cursor-pointer"
        style="color: var(--md-sys-color-on-surface);"
      >
        <Icon name="skip-forward" weight="fill" size={19} />
      </button>

      <button
        type="button"
        onclick={onCycleRepeat}
        aria-label="Repeat"
        class="m3-icon-btn w-9 h-9 flex items-center justify-center relative cursor-pointer"
        style={repeatMode !== 'off' ? 'color: var(--md-sys-color-primary);' : ''}
      >
        <Icon name={repeatMode === 'one' ? 'repeat-once' : 'repeat'} weight={repeatMode !== 'off' ? 'bold' : 'regular'} size={18} />
        {#if repeatMode !== 'off'}
          <span
            class="absolute bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full"
            style="background-color: var(--md-sys-color-primary);"
          ></span>
        {/if}
      </button>
    </div>

    <!-- Right: Card Focus Actions & M3 Volume Slider -->
    <div class="w-[30%] min-w-[180px] flex items-center justify-end gap-2.5">
      <div class="hidden md:flex items-center gap-1 pr-2">
        <button
          type="button"
          onclick={() => (onToggleLyrics ? onToggleLyrics() : onFocusRightCard('lyrics'))}
          title={isLyricsOpen ? 'Tutup Lirik Layar Penuh' : 'Lirik Lagu'}
          aria-label="Lirik Lagu"
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
          style={isLyricsOpen ? 'color: var(--md-sys-color-primary);' : ''}
        >
          <Icon name="microphone-stage" weight={isLyricsOpen ? 'bold' : 'regular'} size={17} />
        </button>
        <button
          type="button"
          onclick={() => onFocusRightCard('queue')}
          title={rightCollapsed ? 'Expand Queue Card' : 'Focus Queue Card'}
          aria-label="Focus Queue Card"
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
          style={!rightCollapsed && rightPanelMode === 'queue' ? 'color: var(--md-sys-color-primary);' : ''}
        >
          <Icon name="queue" weight={!rightCollapsed && rightPanelMode === 'queue' ? 'bold' : 'regular'} size={17} />
        </button>
      </div>

      <div class="flex items-center gap-2 w-36 shrink-0">
        <button
          type="button"
          onclick={onToggleMute}
          aria-label={muted ? 'Unmute' : 'Mute'}
          class="m3-icon-btn w-8 h-8 shrink-0 flex items-center justify-center cursor-pointer"
        >
          <Icon name={displayVolume === 0 ? 'speaker-x' : displayVolume < 45 ? 'speaker-low' : 'speaker-high'} size={18} />
        </button>
        <input
          type="range"
          min="0"
          max="100"
          value={displayVolume}
          oninput={(e) => onVolumeChange(Number((e.target as HTMLInputElement).value))}
          aria-label="Volume"
          class="velora-range flex-1 min-w-0"
          style="background: linear-gradient(to right, var(--md-sys-color-primary) {displayVolume}%, var(--md-sys-color-surface-container-highest) {displayVolume}%);"
        />
      </div>
    </div>
  </div>
</footer>
