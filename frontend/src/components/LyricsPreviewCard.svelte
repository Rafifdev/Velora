<script lang="ts">
  import { untrack } from 'svelte';
  import {
    computeLineSweepProgress,
    computeWordProgressList,
    wordSweepBackground
  } from '../lib/lyricsSweep';
  import { getPrefBool, getPrefValue } from '../lib/preferences';
  import { getPrefLang, tr } from '../lib/i18n';
  import type { Track, TrackLyrics } from '../lib/types';
  import EqIndicator from './EqIndicator.svelte';
  import Icon from './Icon.svelte';

  interface Props {
    currentTrack: Track | null;
    isPlaying: boolean;
    positionSec: number;
    lyricOffsetSec?: number;
    lyrics: TrackLyrics | null;
    lyricsLoading: boolean;
    preferences: Record<string, string>;
    collapsed?: boolean;
    lyricsExpanded?: boolean;
    cardEl?: HTMLElement | undefined;
    onSeek: (sec: number) => void;
    onOpenLyrics?: () => void;
  }

  let {
    currentTrack,
    isPlaying,
    positionSec,
    lyricOffsetSec = 0,
    lyrics,
    lyricsLoading,
    preferences,
    collapsed = false,
    lyricsExpanded = $bindable(false),
    cardEl = $bindable(),
    onSeek,
    onOpenLyrics
  }: Props = $props();

  let lyricsScrollEl: HTMLDivElement | undefined = $state();

  const lang = $derived(getPrefLang(preferences));
  const lyricsAutoScroll = $derived(getPrefBool(preferences, 'lyrics.autoScroll'));
  const lyricsTextSize = $derived(Number(getPrefValue(preferences, 'lyrics.textSize') || '0'));
  const lyricsSweep = $derived(getPrefBool(preferences, 'lyrics.sweep'));
  const lyricsBlurOthers = $derived(
    getPrefBool(preferences, 'lyrics.blurOthers') && getPrefValue(preferences, 'appearance.performance') !== 'lite'
  );
  const lyricsMiniLyrics = $derived(getPrefBool(preferences, 'lyrics.miniLyrics'));
  const lyricsPlainFallback = $derived(getPrefBool(preferences, 'lyrics.plainFallback'));

  const lyricsExpandedSizeClass = $derived(
    lyricsTextSize <= 0
      ? 'text-[15px] leading-[1.35]'
      : lyricsTextSize >= 2
        ? 'text-[20px] leading-[1.35]'
        : 'text-[18px] leading-[1.35]'
  );

  interface DisplayLyricLine {
    text: string;
    timeSec: number | null;
    isSynced: boolean;
  }

  const lyricLines = $derived.by((): DisplayLyricLine[] => {
    if (lyrics?.syncedLyrics && lyrics.syncedLyrics.length > 0) {
      return lyrics.syncedLyrics.map((l) => ({
        text: l.text,
        timeSec: l.timeSec,
        isSynced: true
      }));
    }
    if (lyricsPlainFallback && lyrics?.plainLyrics) {
      return lyrics.plainLyrics
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter((line) => line.length > 0)
        .map((text) => ({ text, timeSec: null, isSynced: false }));
    }
    return [];
  });

  let smoothPosSec = $state(0);
  let anchorPos = 0;
  let anchorAt = 0;

  $effect(() => {
    anchorPos = positionSec;
    anchorAt = performance.now();
    smoothPosSec = positionSec;
  });

  $effect(() => {
    if (collapsed || !isPlaying || !lyricsSweep || (lyrics?.syncedLyrics?.length ?? 0) === 0) {
      smoothPosSec = positionSec;
      return;
    }
    let rafId = 0;
    const step = (now: number) => {
      if (typeof document !== 'undefined' && document.hidden) {
        rafId = 0;
        return;
      }
      const dtSec = Math.max(0, Math.min(0.35, (now - anchorAt) / 1000));
      smoothPosSec = anchorPos + dtSec;
      rafId = requestAnimationFrame(step);
    };
    const onVisibility = () => {
      if (!document.hidden && !rafId) {
        rafId = requestAnimationFrame(step);
      }
    };
    if (typeof document === 'undefined' || !document.hidden) {
      rafId = requestAnimationFrame(step);
    }
    document?.addEventListener('visibilitychange', onVisibility);
    return () => {
      cancelAnimationFrame(rafId);
      document?.removeEventListener('visibilitychange', onVisibility);
    };
  });

  const activeSyncedIndex = $derived.by(() => {
    if (!lyrics?.syncedLyrics || lyrics.syncedLyrics.length === 0) return -1;
    let active = -1;
    const eff = smoothPosSec + 0.05 + lyricOffsetSec;
    for (let i = 0; i < lyrics.syncedLyrics.length; i++) {
      if (eff >= lyrics.syncedLyrics[i].timeSec) {
        active = i;
      } else {
        break;
      }
    }
    return active;
  });

  function getLineSweepProgress(idx: number): number {
    if (!lyricsSweep || idx !== activeSyncedIndex || !lyrics?.syncedLyrics) return 1;
    const cur = lyrics.syncedLyrics[idx];
    const next = lyrics.syncedLyrics[idx + 1];
    if (!cur) return 1;
    return computeLineSweepProgress(cur.text || '♪', cur.timeSec, next?.timeSec, smoothPosSec + lyricOffsetSec);
  }

  $effect(() => {
    const _expanded = lyricsExpanded;
    const idx = activeSyncedIndex;
    const el = lyricsScrollEl;
    if (!el || idx < 0 || !lyricsAutoScroll || collapsed) return;
    untrack(() => {
      const activeEl = el.querySelector(`[data-lyric-idx="${idx}"]`) as HTMLElement | null;
      if (activeEl) {
        const targetTop = Math.max(
          0,
          activeEl.offsetTop - el.clientHeight / 2 + activeEl.clientHeight / 2
        );
        el.scrollTo({ top: targetTop, behavior: 'smooth' });
      }
    });
  });
</script>

<section
  bind:this={cardEl}
  class="group/lyrics rounded-lg p-4 relative overflow-hidden"
  style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
>
  <div class="flex items-center justify-between gap-2 mb-3">
    <span class="text-[15px] font-bold">{tr(lang, 'nav.lyrics')}</span>

    {#if onOpenLyrics}
      <button
        type="button"
        onclick={() => onOpenLyrics()}
        title={tr(lang, 'nav.expandLyrics')}
        aria-label={tr(lang, 'nav.expandLyrics')}
        class="w-8 h-8 sm:w-7 sm:h-7 rounded-full flex items-center justify-center opacity-75 hover:opacity-100 hover:bg-[var(--md-sys-color-state-hover)] transition-colors cursor-pointer"
      >
        <Icon name="arrows-out-simple" size={16} />
      </button>
    {/if}
  </div>

  <div class="relative">
    <div
      bind:this={lyricsScrollEl}
      class="overflow-hidden pr-1 space-y-2.5 pb-12"
      style="max-height: {lyricsExpanded ? '420px' : '152px'}; transition: max-height 380ms cubic-bezier(0.22, 1, 0.36, 1);"
    >
      {#if !currentTrack}
        <p class="text-[14px] font-semibold opacity-80 py-3">
          {tr(lang, 'nav.playSongForLyrics')}
        </p>
      {:else if lyricsLoading}
        <div class="py-6 flex items-center justify-center" aria-label={tr(lang, 'nav.loadingLyrics')}>
          <EqIndicator variant="loader" color="var(--md-sys-color-on-primary-container)" />
        </div>
      {:else if !lyricsExpanded && !lyricsMiniLyrics}
        <p class="text-[13px] font-medium opacity-75 py-2">
          {tr(lang, 'nav.clickExpandLyrics')}
        </p>
      {:else if lyricLines.length > 0}
        {#each lyricLines as line, idx (idx + '-' + line.text)}
          {@const active = idx === activeSyncedIndex}
          <button
            type="button"
            data-lyric-idx={idx}
            onclick={() => line.isSynced && line.timeSec !== null && onSeek(line.timeSec)}
            class="w-full text-left font-bold {lyricsExpandedSizeClass} origin-left transition-all duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] cursor-pointer {active
              ? 'opacity-100 scale-[1.02]'
              : lyricsBlurOthers && line.isSynced
                ? 'opacity-35 scale-[0.93] blur-[1.2px] hover:opacity-100 hover:blur-none'
                : 'opacity-35 scale-[0.93] hover:opacity-100'}"
          >
            {#if active && line.isSynced && lyricsSweep}
              {@const words = computeWordProgressList(line.text || '♪', getLineSweepProgress(idx))}
              {#each words as item, wIdx (wIdx)}
                <span
                  class="inline-block will-change-transform"
                  style="background-image: {wordSweepBackground(item.prog, 'var(--md-sys-color-on-primary-container)', 'color-mix(in srgb, var(--md-sys-color-on-primary-container) 34%, transparent)')}; -webkit-background-clip: text; background-clip: text; color: transparent; transform: translate3d(0, {(-1.0 * item.prog).toFixed(2)}px, 0);"
                >{item.word}</span>{wIdx < words.length - 1 ? ' ' : ''}
              {/each}
            {:else}
              {line.text}
            {/if}
          </button>
        {/each}
      {:else}
        <p class="text-[14px] font-semibold text-[var(--md-sys-color-on-primary-container)] opacity-75 py-4">
          {tr(lang, 'nav.noLyricsFor')} "{currentTrack.title}".
        </p>
      {/if}
    </div>

    <div
      class="pointer-events-none absolute inset-x-0 bottom-0 h-14 flex items-end justify-end"
      style="background: linear-gradient(to bottom, transparent 0%, var(--md-sys-color-primary-container) 88%);"
    >
      <button
        type="button"
        onclick={() => (lyricsExpanded = !lyricsExpanded)}
        style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
        class="text-[12px] font-bold px-4 py-1.5 rounded-full cursor-pointer opacity-100 sm:opacity-0 sm:translate-y-3 sm:pointer-events-none pointer-events-auto group-hover/lyrics:opacity-100 group-hover/lyrics:translate-y-0 group-hover/lyrics:pointer-events-auto transition-all duration-300 ease-out hover:scale-[1.03] active:scale-[0.98]"
      >
        {lyricsExpanded ? tr(lang, 'nav.seeLess') : tr(lang, 'nav.seeMore')}
      </button>
    </div>
  </div>
</section>
