<script lang="ts">
  import { fade } from 'svelte/transition';
  import type { Track, TrackLyrics } from '../lib/types';
  import { resolveCoverUrl } from '../lib/api';
  import Icon from './Icon.svelte';

  let {
    track = null,
    lyrics = null,
    lyricsLoading = false,
    positionSec = 0,
    mediaBaseUrl = '',
    onSeek,
    onClose
  }: {
    track: Track | null;
    lyrics: TrackLyrics | null;
    lyricsLoading?: boolean;
    positionSec?: number;
    mediaBaseUrl?: string;
    onSeek: (sec: number) => void;
    onClose: () => void;
  } = $props();

  let scrollEl: HTMLDivElement | undefined = $state();
  let lineEls: Array<HTMLElement | null> = $state([]);

  let cover = $derived(track ? resolveCoverUrl(track.coverUrl, mediaBaseUrl) : '');

  let syncedLines = $derived(lyrics?.syncedLines ?? []);
  let hasSynced = $derived(syncedLines.length > 0);
  let plainLines = $derived(
    (lyrics?.plainLyrics ?? '')
      .split(/\r?\n/)
      .map((l) => l.trimEnd())
  );

  let activeSyncedIndex = $derived.by(() => {
    if (!hasSynced) return -1;
    const t = positionSec + 0.15;
    let idx = -1;
    for (let i = 0; i < syncedLines.length; i++) {
      if (syncedLines[i].timeSec <= t) idx = i;
      else break;
    }
    return idx;
  });

  $effect(() => {
    const idx = activeSyncedIndex;
    const container = scrollEl;
    if (!container || idx < 0) return;
    const el = lineEls[idx];
    if (!el) return;
    const targetTop = Math.max(
      0,
      el.offsetTop - container.clientHeight * 0.42 + el.clientHeight / 2
    );
    container.scrollTo({ top: targetTop, behavior: 'smooth' });
  });
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
    style="background: linear-gradient(180deg, rgba(10, 12, 16, 0.58) 0%, rgba(10, 12, 16, 0.78) 60%, rgba(10, 12, 16, 0.90) 100%);"
  ></div>

  <!-- Top Header -->
  <header class="relative z-10 flex items-center justify-between gap-4 px-8 pt-6 pb-4 shrink-0">
    <div class="flex items-center gap-3.5 min-w-0">
      {#if cover}
        <img
          src={cover}
          alt={track?.title ?? 'Album cover'}
          class="w-12 h-12 rounded-lg object-cover shadow-lg shrink-0"
        />
      {:else}
        <div
          class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0 bg-white/10 text-white/70"
        >
          <Icon name="music-notes" size={22} />
        </div>
      {/if}
      <div class="min-w-0">
        <p class="text-[11px] font-semibold uppercase tracking-widest text-white/60">
          Lirik Lagu
        </p>
        <h1 class="text-lg font-bold text-white truncate">
          {track?.title ?? 'Tidak ada lagu diputar'}
        </h1>
        {#if track?.artistName}
          <p class="text-xs text-white/70 truncate">{track.artistName}</p>
        {/if}
      </div>
    </div>

    <button
      type="button"
      onclick={onClose}
      class="w-9 h-9 rounded-full flex items-center justify-center bg-white/10 hover:bg-white/20 text-white transition-colors duration-200 cursor-pointer shrink-0"
      title="Tutup layar lirik"
      aria-label="Tutup layar lirik"
    >
      <Icon name="x" size={18} />
    </button>
  </header>

  <!-- Main Full-Screen Lyrics Area -->
  <div
    bind:this={scrollEl}
    class="relative z-10 flex-1 min-h-0 overflow-y-auto px-8 sm:px-14 md:px-24 pb-20 pt-6"
  >
    {#if !track}
      <div class="h-full flex flex-col items-center justify-center text-center gap-3 text-white/65">
        <Icon name="microphone-stage" size={40} />
        <p class="text-base font-medium">Putar lagu untuk melihat liriknya di sini.</p>
      </div>
    {:else if lyricsLoading}
      <div class="h-full flex flex-col items-center justify-center text-center gap-3 text-white/65">
        <Icon name="spinner-gap" size={32} class="animate-spin" />
        <p class="text-sm font-medium">Memuat lirik…</p>
      </div>
    {:else if hasSynced}
      <div class="max-w-4xl mx-auto flex flex-col gap-5 py-12">
        {#each syncedLines as line, idx (idx + '-' + line.timeSec)}
          {@const active = idx === activeSyncedIndex}
          <button
            bind:this={lineEls[idx]}
            type="button"
            onclick={() => onSeek(line.timeSec)}
            class="text-left text-2xl sm:text-3xl md:text-4xl font-bold leading-snug tracking-tight transition-all duration-300 cursor-pointer text-white {active
              ? 'opacity-100 scale-[1.01] origin-left'
              : 'opacity-35 hover:opacity-100'}"
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
              class="text-xl sm:text-2xl md:text-3xl font-bold leading-relaxed text-white opacity-70 hover:opacity-100 transition-opacity duration-300"
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
