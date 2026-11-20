<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { api, formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import type { Track } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
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
    waveBar?: boolean;
    lang?: LangCode;
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
    onOpenMobileNowPlaying?: () => void;
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
    waveBar = true,
    lang = 'en',
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
    onToggleLyrics,
    onOpenMobileNowPlaying
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

<!-- Spotify-Style Floating Mini Player (Mobile Only) -->
<div class="sm:hidden px-2 pb-1.5 shrink-0 select-none">
  <div
    class="relative rounded-lg overflow-hidden shadow-lg"
    style="background-color: color-mix(in srgb, var(--md-sys-color-tertiary-container) 36%, var(--md-sys-color-surface-container-high));"
  >
    <div class="flex items-center justify-between gap-2.5 px-2.5 py-2">
      <button
        type="button"
        onclick={() => onOpenMobileNowPlaying?.()}
        class="flex-1 min-w-0 flex items-center gap-2.5 text-left cursor-pointer"
      >
        <div class="w-10 h-10 rounded-md overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)] flex items-center justify-center">
          {#if currentTrack && resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl)}
            {@const mCover = resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl)}
            <img
              src={mCover}
              alt={currentTrack.title}
              class="block w-full h-full object-cover object-center"
              class:velora-yt-video-thumb={isYTVideoThumb(mCover)}
            />
          {:else}
            <Icon name="music-note" size={18} />
          {/if}
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
            {currentTrack?.title || 'Belum ada lagu diputar'}
          </div>
          <div class="text-[11px] truncate" style="color: var(--md-sys-color-on-surface-variant);">
            {currentTrack?.artistName || 'Velora'}
          </div>
        </div>
      </button>

      <div class="flex items-center gap-1 shrink-0">
        <button
          type="button"
          disabled={!currentTrack}
          onclick={() => currentTrack && onToggleFavorite(currentTrack.id)}
          aria-label={tr(lang, 'nav.likeSong')}
          class="w-9 h-9 rounded-full flex items-center justify-center cursor-pointer disabled:opacity-40"
        >
          <Icon
            name="heart"
            weight={currentTrack?.isFavorite ? 'fill' : 'regular'}
            size={19}
            style={currentTrack?.isFavorite ? 'color: var(--md-sys-color-tertiary);' : 'color: var(--md-sys-color-on-surface);'}
          />
        </button>
        <button
          type="button"
          onclick={onTogglePlay}
          aria-label={isPlaying ? 'Jeda' : 'Putar'}
          class="w-9 h-9 rounded-full flex items-center justify-center cursor-pointer"
          style="color: var(--md-sys-color-on-surface);"
        >
          <Icon name={isPlaying ? 'pause' : 'play'} weight="fill" size={20} />
        </button>
      </div>
    </div>

    <!-- Bottom 2px thin progress bar (Spotify Mobile style) -->
    <div class="h-[2px] w-full bg-white/15">
      <div
        class="h-full transition-[width] duration-200"
        style="width: {progressPct}%; background-color: var(--md-sys-color-on-surface);"
      ></div>
    </div>
  </div>
</div>

<!-- Full-Width Flat Borderless Media Player Bar (Desktop & Tablet) -->
<footer
  class="hidden sm:block w-full shrink-0 select-none rounded-none overflow-visible relative z-20 transition-colors duration-300"
  style="background-color: var(--md-sys-color-surface-container-lowest);"
>
  <!-- Top Edge Full-Width Seekbar + Timestamps -->
  <div class="relative w-full">
    <div
      class="relative w-full py-1.5 -mt-1.5 cursor-pointer"
      onpointerdown={(e) => {
        const rect = e.currentTarget.getBoundingClientRect();
        if (rect.width <= 0) return;
        const ratio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
        onSeek(ratio * Math.max(1, effectiveDuration));
      }}
    >
      {#if waveBar && isPlaying}
        <div
          class="pointer-events-none absolute left-0 right-0 top-1/2 -translate-y-1/2 h-[9px] flex items-center"
          aria-hidden="true"
        >
          <div
            class="h-full overflow-hidden shrink-0"
            style="width: {progressPct}%;"
          >
            <svg
              class="velora-wave-svg h-full w-[calc(100%+24px)] block"
              preserveAspectRatio="none"
              style="color: var(--md-sys-color-primary);"
            >
              <defs>
                <pattern id="velora-wave-pat" width="24" height="9" patternUnits="userSpaceOnUse">
                  <path
                    d="M0 4.5 Q6 1 12 4.5 T24 4.5"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2.5"
                    stroke-linecap="round"
                  />
                </pattern>
              </defs>
              <rect width="100%" height="9" fill="url(#velora-wave-pat)" />
            </svg>
          </div>
          <div
            class="h-[4px] flex-1"
            style="background-color: var(--md-sys-color-surface-container-highest);"
          ></div>
        </div>
      {/if}
      <input
        type="range"
        min="0"
        max={Math.max(1, effectiveDuration)}
        step="0.5"
        value={Math.min(positionSec, Math.max(1, effectiveDuration))}
        oninput={(e) => onSeek(Number((e.target as HTMLInputElement).value))}
        aria-label="Seekbar durasi lagu"
        class="velora-seekbar w-full block relative z-10"
        style={waveBar && isPlaying
          ? 'background: transparent;'
          : `background: linear-gradient(to right, var(--md-sys-color-primary) ${progressPct}%, var(--md-sys-color-surface-container-highest) ${progressPct}%);`}
      />
    </div>
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
  <div class="h-[68px] px-3 sm:px-6 pb-2 flex items-center justify-between gap-2 sm:gap-4">
    <!-- Left: Track Cover & Info (1:1 Reference Style) -->
    <div class="flex-1 min-w-0 sm:flex-none sm:w-[30%] sm:min-w-[180px] flex items-center gap-2.5 sm:gap-3.5">
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
    <div class="flex items-center justify-center gap-1 sm:gap-3 shrink-0">
      <button
        type="button"
        disabled={!currentTrack}
        onclick={() => currentTrack && onToggleFavorite(currentTrack.id)}
        aria-label={tr(lang, 'nav.likeSong')}
        class="hidden sm:flex m3-icon-btn w-9 h-9 items-center justify-center cursor-pointer disabled:opacity-35"
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
        aria-label={tr(lang, 'nav.shuffle')}
        class="hidden sm:flex m3-icon-btn w-9 h-9 items-center justify-center relative cursor-pointer"
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
        aria-label={tr(lang, 'nav.previous')}
        class="m3-icon-btn w-9 h-9 flex items-center justify-center cursor-pointer"
        style="color: var(--md-sys-color-on-surface);"
      >
        <Icon name="skip-back" weight="fill" size={19} />
      </button>

      <button
        type="button"
        onclick={onTogglePlay}
        aria-label={isPlaying ? tr(lang, 'nav.pause') : tr(lang, 'nav.play')}
        class="m3-fab w-12 h-12 flex items-center justify-center cursor-pointer"
      >
        <Icon name={isPlaying ? 'pause' : 'play'} weight="fill" size={21} />
      </button>

      <button
        type="button"
        onclick={onNext}
        aria-label={tr(lang, 'nav.next')}
        class="m3-icon-btn w-9 h-9 flex items-center justify-center cursor-pointer"
        style="color: var(--md-sys-color-on-surface);"
      >
        <Icon name="skip-forward" weight="fill" size={19} />
      </button>

      <button
        type="button"
        onclick={onCycleRepeat}
        aria-label={tr(lang, 'nav.repeat')}
        class="hidden sm:flex m3-icon-btn w-9 h-9 items-center justify-center relative cursor-pointer"
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
    <div class="hidden sm:flex w-[30%] min-w-[180px] items-center justify-end gap-2.5">
      <div class="hidden md:flex items-center gap-1 pr-2">
        <button
          type="button"
          onclick={() => (onToggleLyrics ? onToggleLyrics() : onFocusRightCard('lyrics'))}
          title={isLyricsOpen ? tr(lang, 'nav.closeLyrics') : tr(lang, 'nav.songLyrics')}
          aria-label={tr(lang, 'nav.songLyrics')}
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
          style={isLyricsOpen ? 'color: var(--md-sys-color-primary);' : ''}
        >
          <Icon name="microphone-stage" weight={isLyricsOpen ? 'bold' : 'regular'} size={17} />
        </button>
        <button
          type="button"
          onclick={() => onFocusRightCard('queue')}
          title={rightCollapsed ? tr(lang, 'nav.expandQueue') : tr(lang, 'nav.focusQueue')}
          aria-label={tr(lang, 'nav.focusQueue')}
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
          style={!rightCollapsed && rightPanelMode === 'queue' ? 'color: var(--md-sys-color-primary);' : ''}
        >
          <Icon name="queue" weight={!rightCollapsed && rightPanelMode === 'queue' ? 'bold' : 'regular'} size={17} />
        </button>
        <button
          type="button"
          onclick={() => api.windowMinimize()}
          title={tr(lang, 'nav.miniPlayer')}
          aria-label={tr(lang, 'nav.minimizeIsland')}
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
        >
          <Icon name="picture-in-picture" size={17} />
        </button>
      </div>

      <div class="flex items-center gap-2 w-36 shrink-0">
        <button
          type="button"
          onclick={onToggleMute}
          aria-label={muted ? tr(lang, 'nav.unmute') : tr(lang, 'nav.mute')}
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
