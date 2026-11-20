<script lang="ts">
  import { fade } from 'svelte/transition';
  import type { Track, TrackLyrics } from '../lib/types';
  import { resolveCoverUrl } from '../lib/api';
  import { getPrefBool, getPrefValue } from '../lib/preferences';
  import Icon from './Icon.svelte';

  let {
    track = null,
    lyrics = null,
    lyricsLoading = false,
    positionSec = 0,
    mediaBaseUrl = '',
    preferences = {},
    onSeek,
    onResync,
    onClose
  }: {
    track: Track | null;
    lyrics: TrackLyrics | null;
    lyricsLoading?: boolean;
    positionSec?: number;
    mediaBaseUrl?: string;
    preferences?: Record<string, string>;
    onSeek: (sec: number) => void;
    onResync?: () => void;
    onClose: () => void;
  } = $props();

  let scrollEl: HTMLDivElement | undefined = $state();
  let lineEls: Array<HTMLElement | null> = $state([]);
  let autoSync = $state(true);
  let offsetSec = $state(0);

  const lyricsTextSize = $derived(Number(getPrefValue(preferences, 'lyrics.textSize') || '1'));
  const lyricsAutoScroll = $derived(getPrefBool(preferences, 'lyrics.autoScroll'));
  const lyricsSweep = $derived(getPrefBool(preferences, 'lyrics.sweep'));
  const lyricsBlurOthers = $derived(
    getPrefBool(preferences, 'lyrics.blurOthers') && getPrefValue(preferences, 'appearance.performance') !== 'lite'
  );
  const lyricsNudgeStepSec = $derived(Math.max(0.25, Number(getPrefValue(preferences, 'lyrics.nudgeMs') || '500') / 1000));
  const lyricsPlainFallback = $derived(getPrefBool(preferences, 'lyrics.plainFallback'));

  const syncedSizeClass = $derived(
    lyricsTextSize <= 0
      ? 'text-xl sm:text-2xl md:text-3xl'
      : lyricsTextSize >= 2
        ? 'text-3xl sm:text-4xl md:text-5xl'
        : 'text-2xl sm:text-3xl md:text-4xl'
  );
  const plainSizeClass = $derived(
    lyricsTextSize <= 0
      ? 'text-lg sm:text-xl md:text-2xl'
      : lyricsTextSize >= 2
        ? 'text-2xl sm:text-3xl md:text-4xl'
        : 'text-xl sm:text-2xl md:text-3xl'
  );

  let cover = $derived(track ? resolveCoverUrl(track.coverUrl, mediaBaseUrl) : '');

  let syncedLines = $derived(lyrics?.syncedLyrics ?? []);
  let hasSynced = $derived(syncedLines.length > 0);
  let plainLines = $derived(
    lyricsPlainFallback
      ? (lyrics?.plainLyrics ?? '')
          .split(/\r?\n/)
          .map((l) => l.trimEnd())
      : []
  );

  let effectiveTime = $derived(positionSec + 0.25 + offsetSec);

  let activeSyncedIndex = $derived.by(() => {
    if (!hasSynced) return -1;
    let idx = -1;
    for (let i = 0; i < syncedLines.length; i++) {
      if (syncedLines[i].timeSec <= effectiveTime) idx = i;
      else break;
    }
    return idx;
  });

  function getLineSweepPct(idx: number): number {
    if (!lyricsSweep || idx !== activeSyncedIndex || !hasSynced) return 100;
    const cur = syncedLines[idx];
    const next = syncedLines[idx + 1];
    const start = cur.timeSec;
    const end = next ? Math.max(start + 0.6, next.timeSec) : start + 4.0;
    const pct = ((effectiveTime - start) / (end - start)) * 100;
    return Math.max(8, Math.min(100, pct));
  }

  function scrollToActiveLine(behavior: ScrollBehavior = 'smooth') {
    const idx = activeSyncedIndex;
    const container = scrollEl;
    if (!container || idx < 0) return;
    const el = lineEls[idx];
    if (!el) return;
    const targetTop = Math.max(
      0,
      el.offsetTop - container.clientHeight * 0.42 + el.clientHeight / 2
    );
    container.scrollTo({ top: targetTop, behavior });
  }

  $effect(() => {
    const _trackId = track?.id;
    autoSync = lyricsAutoScroll;
    offsetSec = 0;
  });

  $effect(() => {
    autoSync = lyricsAutoScroll;
  });

  $effect(() => {
    const idx = activeSyncedIndex;
    if (!lyricsAutoScroll || !autoSync || idx < 0) return;
    scrollToActiveLine('smooth');
  });

  function handleSyncClick() {
    autoSync = true;
    if (hasSynced) {
      scrollToActiveLine('smooth');
    } else {
      onResync?.();
    }
  }
</script>

<div
  in:fade={{ duration: 220 }}
  class="relative h-full w-full rounded-xl overflow-hidden flex flex-col select-none"
  style="background-color: var(--md-sys-color-surface-container);"
>
  <!-- Blurred Album Cover Backdrop -->
  {#if cover}
    <img
      src={cover}
      alt=""
      aria-hidden="true"
      class="absolute inset-0 w-full h-full object-cover scale-125 blur-3xl opacity-45 pointer-events-none"
    />
  {/if}
  <div
    class="absolute inset-0 pointer-events-none"
    style="background: linear-gradient(180deg, rgba(10, 12, 16, 0.84) 0%, rgba(10, 12, 16, 0.68) 18%, rgba(10, 12, 16, 0.72) 55%, rgba(10, 12, 16, 0.88) 88%, rgba(10, 12, 16, 0.94) 100%);"
  ></div>

  <!-- Top Header -->
  <header class="relative z-10 flex items-center justify-between gap-4 px-8 pt-6 pb-4 shrink-0">
    <div class="flex items-center gap-4 min-w-0">
      {#if cover}
        <img
          src={cover}
          alt={track?.title ?? 'Album cover'}
          class="w-16 h-16 rounded-lg object-cover shadow-lg shrink-0"
        />
      {:else}
        <div
          class="w-16 h-16 rounded-lg flex items-center justify-center shrink-0 bg-white/10 text-white/70"
        >
          <Icon name="music-notes" size={26} />
        </div>
      {/if}
      <div class="min-w-0 flex flex-col justify-center">
        <p class="text-[11px] leading-[14px] font-semibold uppercase tracking-widest text-white/60 mb-1.5">
          Lirik Lagu
        </p>
        <h1 class="text-lg leading-[24px] font-bold text-white truncate">
          {track?.title ?? 'Tidak ada lagu diputar'}
        </h1>
        {#if track?.artistName}
          <p class="text-xs leading-[18px] text-white/70 truncate mt-0.5">{track.artistName}</p>
        {/if}
      </div>
    </div>

    <div class="flex items-center gap-2 shrink-0">
      {#if hasSynced}
        <div class="flex items-center gap-1 bg-white/10 rounded-full px-1.5 py-1">
          <button
            type="button"
            onclick={() => (offsetSec = Number((offsetSec - lyricsNudgeStepSec).toFixed(2)))}
            class="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold text-white/80 hover:text-white hover:bg-white/15 cursor-pointer"
            title="Mundurkan lirik {lyricsNudgeStepSec}s"
            aria-label="Mundurkan lirik"
          >
            -
          </button>
          {#if offsetSec !== 0}
            <button
              type="button"
              onclick={() => (offsetSec = 0)}
              class="px-1.5 text-[11px] font-semibold tabular-nums text-white/85 hover:text-white cursor-pointer"
              title="Reset geser waktu lirik"
            >
              {offsetSec > 0 ? `+${offsetSec}s` : `${offsetSec}s`}
            </button>
          {/if}
          <button
            type="button"
            onclick={() => (offsetSec = Number((offsetSec + lyricsNudgeStepSec).toFixed(2)))}
            class="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold text-white/80 hover:text-white hover:bg-white/15 cursor-pointer"
            title="Majukan lirik {lyricsNudgeStepSec}s"
            aria-label="Majukan lirik"
          >
            +
          </button>
        </div>
      {/if}
      <button
        type="button"
        onclick={handleSyncClick}
        class="w-9 h-9 rounded-full flex items-center justify-center transition-colors duration-200 cursor-pointer {hasSynced && autoSync
          ? 'bg-white/25 text-white'
          : 'bg-white/10 hover:bg-white/20 text-white/80 hover:text-white'}"
        title={hasSynced ? 'Sinkronkan lirik dengan lagu' : 'Cari ulang lirik tersinkron'}
        aria-label="Sinkronisasi lirik"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <rect x="3" y="9" width="2.4" height="6" rx="1.2" fill="currentColor" />
          <rect x="7.4" y="6" width="2.4" height="12" rx="1.2" fill="currentColor" />
          <rect x="11.8" y="3" width="2.4" height="18" rx="1.2" fill="currentColor" />
          <rect x="16.2" y="6" width="2.4" height="12" rx="1.2" fill="currentColor" />
          <rect x="20.6" y="9" width="2.4" height="6" rx="1.2" fill="currentColor" />
        </svg>
      </button>

      <button
        type="button"
        onclick={onClose}
        class="w-9 h-9 rounded-full flex items-center justify-center bg-white/10 hover:bg-white/20 text-white transition-colors duration-200 cursor-pointer"
        title="Tutup layar lirik"
        aria-label="Tutup layar lirik"
      >
        <Icon name="x" size={18} />
      </button>
    </div>
  </header>

  <!-- Main Full-Screen Lyrics Area with Ultra-Smooth Top & Bottom Mask Fade -->
  <div class="relative z-10 flex-1 min-h-0">
    <div
      bind:this={scrollEl}
      onwheel={() => {
        if (hasSynced) autoSync = false;
      }}
      ontouchstart={() => {
        if (hasSynced) autoSync = false;
      }}
      class="h-full overflow-y-auto px-8 sm:px-14 md:px-24 pb-24 pt-10"
      style="-webkit-mask-image: linear-gradient(180deg, transparent 0%, rgba(0,0,0,0.45) 5%, black 14%, black 84%, rgba(0,0,0,0.45) 94%, transparent 100%); mask-image: linear-gradient(180deg, transparent 0%, rgba(0,0,0,0.45) 5%, black 14%, black 84%, rgba(0,0,0,0.45) 94%, transparent 100%);"
    >
    {#if !track}
      <div class="h-full flex flex-col items-center justify-center text-center gap-3 text-white/65">
        <Icon name="microphone-stage" size={40} />
        <p class="text-base font-medium">Putar lagu untuk melihat liriknya di sini.</p>
      </div>
    {:else if lyricsLoading}
      <div class="h-full flex flex-col items-center justify-center text-center gap-3 text-white/65">
        <span class="velora-eq-loader" style="--loader-color: rgba(255, 255, 255, 0.85);" aria-hidden="true">
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
          <span class="velora-eq-bar"></span>
        </span>
        <p class="text-sm font-medium">Memuat lirik…</p>
      </div>
    {:else if hasSynced}
      <div class="max-w-4xl mx-auto flex flex-col gap-5 py-12">
        {#each syncedLines as line, idx (idx + '-' + line.timeSec)}
          {@const active = idx === activeSyncedIndex}
          {@const sweepPct = getLineSweepPct(idx)}
          <button
            bind:this={lineEls[idx]}
            type="button"
            onclick={() => onSeek(line.timeSec)}
            class="text-left {syncedSizeClass} font-bold leading-snug tracking-tight origin-left transition-all duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] cursor-pointer text-white {active
              ? 'opacity-100 scale-[1.03]'
              : lyricsBlurOthers
                ? 'opacity-35 scale-[0.92] blur-[1.5px] hover:opacity-90 hover:blur-none'
                : 'opacity-35 scale-[0.92] hover:opacity-90'}"
            style={active && lyricsSweep
              ? `background-image: linear-gradient(90deg, #ffffff ${sweepPct}%, rgba(255,255,255,0.42) ${sweepPct}%); -webkit-background-clip: text; background-clip: text; color: transparent;`
              : ''}
          >
            {line.text || '♪'}
          </button>
        {/each}
      </div>
    {:else if plainLines.some((l) => l.length > 0)}
      <div class="max-w-4xl mx-auto flex flex-col gap-4 py-10">
        {#each plainLines as line, idx (idx)}
          {#if line.trim().length === 0}
            <div class="h-4"></div>
          {:else}
            <p
              class="{plainSizeClass} font-bold leading-relaxed text-white opacity-70 hover:opacity-100 transition-opacity duration-300"
            >
              {line}
            </p>
          {/if}
        {/each}
      </div>
    {:else}
      <div class="h-full flex flex-col items-center justify-center text-center gap-3 text-white/65">
        <Icon name="microphone-slash" size={36} />
        <p class="text-base font-medium">Lirik belum tersedia untuk lagu ini.</p>
      </div>
    {/if}
    </div>
  </div>
</div>
