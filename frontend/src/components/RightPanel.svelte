<script lang="ts">
  import { untrack } from 'svelte';
  import { animate } from 'motion';
  import { fade, fly, slide } from 'svelte/transition';
  import { isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import { getPrefBool, getPrefValue } from '../lib/nuclearPrefs';
  import type { Track, TrackLyrics } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    width: number;
    collapsed: boolean;
    mode?: 'now-playing' | 'queue';
    currentTrack: Track | null;
    queue: Track[];
    queueSourceTitle?: string;
    isPlaying: boolean;
    positionSec: number;
    lyrics: TrackLyrics | null;
    lyricsLoading: boolean;
    preferences: Record<string, string>;
    mediaBaseUrl: string;
    accentColor: string;
    focusCardSignal?: { card: 'album' | 'lyrics' | 'queue'; seq: number } | null;
    onChangeMode?: (mode: 'now-playing' | 'queue') => void;
    onToggleCollapse: () => void;
    onResizeWidth: (newWidth: number) => void;
    onResizeEnd: (finalWidth: number) => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onRemoveFromQueue: (trackId: number) => void;
    onClearQueue: () => void;
    onToggleFavorite: (trackId: number) => void;
    onSeek: (sec: number) => void;
    onOpenTrackDetail?: (track: Track) => void;
    onOpenLyrics?: () => void;
  }

  let {
    width,
    collapsed,
    mode = 'now-playing',
    currentTrack,
    queue,
    queueSourceTitle = '',
    isPlaying: _isPlaying,
    positionSec,
    lyrics,
    lyricsLoading,
    preferences,
    mediaBaseUrl,
    focusCardSignal = null,
    onChangeMode,
    onToggleCollapse,
    onResizeWidth,
    onResizeEnd,
    onPlayTrack,
    onRemoveFromQueue: _onRemoveFromQueue,
    onClearQueue: _onClearQueue,
    onToggleFavorite,
    onSeek,
    onOpenTrackDetail,
    onOpenLyrics
  }: Props = $props();

  let panelScrollEl: HTMLElement | undefined = $state();
  let albumCardEl: HTMLElement | undefined = $state();
  let lyricsCardEl: HTMLElement | undefined = $state();
  let lyricsScrollEl: HTMLElement | undefined = $state();
  let queueCardEl: HTMLElement | undefined = $state();
  let collapseIconEl: HTMLElement | undefined = $state();
  let prevCollapsed: boolean | undefined;
  let flipAnim: { stop?: () => void } | undefined;
  let isResizing = $state(false);
  let lyricsExpanded = $state(false);
  let queueExpanded = $state(false);
  let lyricsTall = $state(false);

  const MIN_WIDTH = 280;
  const MAX_WIDTH = 520;
  const COLLAPSED_WIDTH = 36;
  const COLLAPSED_HOVER_WIDTH = 72;
  let collapsedHover = $state(false);

  const lyricsAutoScroll = $derived(getPrefBool(preferences, 'lyrics.autoScroll'));
  const lyricsTextSize = $derived(Number(getPrefValue(preferences, 'lyrics.textSize') || '1'));

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
    if (lyrics?.plainLyrics) {
      return lyrics.plainLyrics
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter((line) => line.length > 0)
        .map((text) => ({ text, timeSec: null, isSynced: false }));
    }
    return [];
  });

  const activeSyncedIndex = $derived.by(() => {
    if (!lyrics?.syncedLyrics || lyrics.syncedLyrics.length === 0) return -1;
    let active = -1;
    for (let i = 0; i < lyrics.syncedLyrics.length; i++) {
      if (positionSec + 0.25 >= lyrics.syncedLyrics[i].timeSec) {
        active = i;
      } else {
        break;
      }
    }
    return active;
  });

  const previewLines = $derived.by(() => {
    if (lyricLines.length === 0) return [];
    const start = activeSyncedIndex >= 0 ? activeSyncedIndex : 0;
    return lyricLines.slice(start, start + 3);
  });

  const upcomingQueue = $derived.by(() => {
    if (queue.length === 0) return [];
    if (!currentTrack) return queue;
    const idx = queue.findIndex((t) => t.id === currentTrack.id);
    if (idx >= 0) {
      return [...queue.slice(idx + 1), ...queue.slice(0, idx)];
    }
    return queue.filter((t) => t.id !== currentTrack.id);
  });

  const nextTrack = $derived(upcomingQueue[0] ?? null);

  $effect(() => {
    const nextCollapsed = collapsed;
    const targetEl = collapseIconEl;
    if (prevCollapsed === undefined) {
      prevCollapsed = nextCollapsed;
      return;
    }
    if (prevCollapsed !== nextCollapsed && targetEl) {
      prevCollapsed = nextCollapsed;
      flipAnim?.stop?.();
      const reducedMotion =
        typeof window !== 'undefined' &&
        window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      if (!reducedMotion) {
        flipAnim = animate(
          targetEl,
          { scaleX: nextCollapsed ? [-1, 1] : [1, -1] },
          { duration: 0.28, ease: [0.22, 1, 0.36, 1] }
        );
      }
    }
  });

  let lastFocusSeq = -1;
  $effect(() => {
    const sig = focusCardSignal;
    if (!sig || sig.seq === lastFocusSeq) return;
    lastFocusSeq = sig.seq;
    untrack(() => focusSection(sig.card));
  });

  $effect(() => {
    const _expanded = lyricsExpanded;
    if (!lyricsAutoScroll || activeSyncedIndex < 0 || !lyricsScrollEl) return;
    const activeEl = lyricsScrollEl.querySelector(`[data-lyric-idx="${activeSyncedIndex}"]`) as HTMLElement | null;
    if (activeEl) {
      const targetTop = Math.max(
        0,
        activeEl.offsetTop - lyricsScrollEl.clientHeight / 2 + activeEl.clientHeight / 2
      );
      lyricsScrollEl.scrollTo({ top: targetTop, behavior: 'smooth' });
    }
  });

  function scrollLyricsToActive() {
    if (!lyricsScrollEl || activeSyncedIndex < 0) return;
    const activeEl = lyricsScrollEl.querySelector(`[data-lyric-idx="${activeSyncedIndex}"]`) as HTMLElement | null;
    if (activeEl) {
      const targetTop = Math.max(
        0,
        activeEl.offsetTop - lyricsScrollEl.clientHeight / 2 + activeEl.clientHeight / 2
      );
      lyricsScrollEl.scrollTo({ top: targetTop, behavior: 'smooth' });
    }
  }

  function focusSection(card: 'album' | 'lyrics' | 'queue') {
    if (collapsed) {
      onToggleCollapse();
    }
    if (card === 'lyrics') lyricsExpanded = true;
    if (card === 'queue') queueExpanded = true;

    setTimeout(() => {
      const target =
        card === 'album' ? albumCardEl : card === 'lyrics' ? lyricsCardEl : queueCardEl;
      if (target && panelScrollEl) {
        panelScrollEl.scrollTo({
          top: card === 'album' ? 0 : Math.max(0, target.offsetTop - 8),
          behavior: card === 'album' ? 'auto' : 'smooth'
        });
        if (card !== 'album') {
          animate(target, { scale: [0.985, 1] }, { duration: 0.28, ease: [0.2, 0, 0, 1] });
        }
      }
    }, 120);
  }

  function startResize(e: PointerEvent) {
    if (collapsed) return;
    e.preventDefault();
    isResizing = true;
    const startX = e.clientX;
    const startW = width;

    function onMove(ev: PointerEvent) {
      const next = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, startW + (startX - ev.clientX)));
      onResizeWidth(next);
    }

    function onUp(ev: PointerEvent) {
      isResizing = false;
      const finalW = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, startW + (startX - ev.clientX)));
      onResizeEnd(finalW);
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    }

    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  }
</script>

<!-- Collapsible Right Panel with Spotify 1:1 Now Playing, Lyrics, and Queue Cards (Internal Scroll Only) -->
<aside
  aria-label="Right Panel Section"
  onmouseenter={() => {
    if (collapsed) collapsedHover = true;
  }}
  onmouseleave={() => {
    collapsedHover = false;
  }}
  class="relative shrink-0 h-full min-h-0 flex flex-col select-none overflow-visible {collapsed
    ? 'rounded-l-lg rounded-r-none'
    : 'rounded-lg'}"
  style="width: {collapsed ? (collapsedHover ? COLLAPSED_HOVER_WIDTH : COLLAPSED_WIDTH) : width}px; background-color: var(--md-sys-color-surface-container-low); transition: {isResizing ? 'none' : 'width 320ms cubic-bezier(0.16, 1, 0.3, 1), background-color 300ms ease'};"
>
  {#if !collapsed}
    <!-- Spotify Centered Gap Resize Divider -->
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize right panel"
      onpointerdown={startResize}
      class="group absolute top-0 -left-2 w-2 h-full cursor-col-resize z-30 flex items-center justify-center"
    >
      <div
        class="w-[2px] h-[calc(100%-12px)] rounded-full transition-colors duration-150 {isResizing
          ? 'bg-white'
          : 'bg-transparent group-hover:bg-[#727272]'}"
      ></div>
    </div>
  {/if}

  <div class="relative w-full h-full min-h-0 overflow-hidden {collapsed ? 'rounded-l-lg rounded-r-none' : 'rounded-lg'}">
    <!-- EXPANDED / PEEK LAYER: Cross-slides & peeks cleanly on collapsed hover -->
    <div
      aria-hidden={collapsed}
      class="absolute inset-y-0 left-0 flex flex-col transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] {collapsed
        ? collapsedHover
          ? 'opacity-55 translate-x-0 scale-100 pointer-events-none'
          : 'opacity-0 translate-x-6 scale-[0.97] pointer-events-none'
        : 'opacity-100 translate-x-0 scale-100 pointer-events-auto delay-75'}"
      style="width: {collapsed ? `${width}px` : '100%'}; {collapsed ? '' : 'right: 0;'}"
    >
    <!-- Header (Fixed inside RightPanel) -->
    <div class="px-3.5 py-2.5 flex items-center justify-between gap-2 shrink-0">
      <div class="flex items-center gap-2 min-w-0">
        <button
          type="button"
          onclick={onToggleCollapse}
          title="Collapse Right Panel"
          aria-label="Collapse Right Panel"
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
        >
          <span
            bind:this={collapseIconEl}
            class="inline-flex items-center justify-center"
            style="transform: scaleX(-1);"
          >
            <Icon name="sidebar-simple" size={18} />
          </span>
        </button>
        <span class="text-xs font-semibold tracking-wide truncate" style="color: var(--md-sys-color-on-surface-variant);">
          {mode === 'queue' ? 'Antrean' : 'Sedang Diputar'}
        </span>
      </div>
      {#if mode === 'queue' && onChangeMode}
        <button
          type="button"
          onclick={() => onChangeMode('now-playing')}
          title="Kembali ke Sedang Diputar"
          aria-label="Kembali ke Sedang Diputar"
          class="m3-icon-btn w-7 h-7 flex items-center justify-center cursor-pointer shrink-0"
        >
          <Icon name="x" size={15} />
        </button>
      {/if}
    </div>

    <!-- Internal Scroll Container Inside RightPanel -->
    <div bind:this={panelScrollEl} class="flex-1 min-h-0 overflow-y-auto overscroll-contain px-3 pb-3 space-y-3">
      {#if mode === 'queue'}
        <div
          in:fly={{ x: 24, duration: 280 }}
          class="space-y-4 pt-1"
        >
          <!-- Sekarang memutar -->
          <section class="space-y-2">
            <h3 class="px-1.5 text-[14px] font-bold" style="color: var(--md-sys-color-on-surface);">
              Sekarang memutar
            </h3>
            {#if currentTrack}
              {@const activeCover = resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl)}
              <div class="flex items-center gap-3 p-1.5 rounded-lg bg-[var(--md-sys-color-surface-container)]">
                <div class="relative w-11 h-11 rounded-md overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)]">
                  {#if activeCover}
                    <img
                      src={activeCover}
                      alt={currentTrack.title}
                      class="block w-full h-full object-cover object-center"
                      class:velora-yt-video-thumb={isYTVideoThumb(activeCover)}
                    />
                  {:else}
                    <div class="w-full h-full flex items-center justify-center text-[var(--md-sys-color-on-surface-variant)] opacity-45">
                      <Icon name="music-note" size={16} />
                    </div>
                  {/if}
                </div>
                <div class="min-w-0 flex-1">
                  <div class="text-[14px] font-semibold truncate leading-snug" style="color: var(--md-sys-color-primary);">
                    {currentTrack.title}
                  </div>
                  <div class="text-[12px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    {currentTrack.artistName}
                  </div>
                </div>
              </div>
            {:else}
              <p class="px-1.5 text-xs" style="color: var(--md-sys-color-on-surface-variant);">
                Belum ada lagu yang diputar.
              </p>
            {/if}
          </section>

          <!-- Berikutnya dari: Playlist atau Algoritma -->
          <section class="space-y-2">
            <h3 class="px-1.5 text-[14px] font-bold leading-snug" style="color: var(--md-sys-color-on-surface);">
              {queueSourceTitle
                ? `Berikutnya dari: ${queueSourceTitle}`
                : 'Berikutnya: Rekomendasi untuk Anda'}
            </h3>
            {#if upcomingQueue.length === 0}
              <p class="px-1.5 text-xs py-2" style="color: var(--md-sys-color-on-surface-variant);">
                Antrean berikutnya masih kosong.
              </p>
            {:else}
              <div class="space-y-1">
                {#each upcomingQueue as qt (`${qt.id}-${qt.externalId || qt.title}`)}
                  {@const qCover = resolveCoverUrl(qt.coverUrl, mediaBaseUrl)}
                  <button
                    type="button"
                    onclick={() => onPlayTrack(qt, queue)}
                    class="w-full flex items-center gap-3 p-1.5 rounded-lg text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)] transition-colors"
                  >
                    <div class="relative w-11 h-11 rounded-md overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)]">
                      {#if qCover}
                        <img
                          src={qCover}
                          alt={qt.title}
                          loading="lazy"
                          decoding="async"
                          class="block w-full h-full object-cover object-center"
                          class:velora-yt-video-thumb={isYTVideoThumb(qCover)}
                        />
                      {:else}
                        <div class="w-full h-full flex items-center justify-center text-[var(--md-sys-color-on-surface-variant)] opacity-45">
                          <Icon name="music-note" size={16} />
                        </div>
                      {/if}
                    </div>

                    <div class="min-w-0 flex-1">
                      <div class="text-[14px] font-semibold truncate leading-snug" style="color: var(--md-sys-color-on-surface);">
                        {qt.title}
                      </div>
                      <div class="text-[12px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                        {qt.artistName}
                      </div>
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </section>
        </div>
      {:else}
      <div in:fly={{ x: -16, duration: 240 }} class="space-y-3">
      <!-- CARD 1: FULL-BLEED ALBUM & TRACK CARD -->
      <section
        bind:this={albumCardEl}
        class="rounded-lg overflow-hidden"
        style="background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface);"
      >
        {#if currentTrack}
          {@const cover = resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl)}
          <div class="relative w-full aspect-square overflow-clip bg-[var(--md-sys-color-surface-container-highest)]">
            {#if cover}
              <img
                src={cover}
                alt={currentTrack.title}
                class="absolute inset-0 block w-full h-full object-cover object-center"
                class:velora-yt-video-thumb={isYTVideoThumb(cover)}
              />
            {:else}
              <div class="w-full h-full flex items-center justify-center text-[var(--md-sys-color-on-surface-variant)] opacity-40">
                <Icon name="music-note" weight="fill" size={44} />
              </div>
            {/if}
          </div>

          <div class="p-3.5 flex items-center justify-between gap-3">
            <div class="min-w-0 flex-1">
              <button
                type="button"
                onclick={() => onOpenTrackDetail?.(currentTrack)}
                class="w-full text-left text-[16px] font-bold leading-snug truncate hover:underline cursor-pointer"
                title="Lihat Detail Musik"
              >
                {currentTrack.title}
              </button>
              <div class="text-[13px] font-medium text-[var(--md-sys-color-on-surface-variant)] truncate mt-0.5">
                {currentTrack.artistName}
              </div>
            </div>

            <button
              type="button"
              onclick={() => onToggleFavorite(currentTrack.id)}
              aria-label="Favorite track"
              class="w-8 h-8 rounded-full flex items-center justify-center shrink-0 cursor-pointer hover:bg-[var(--md-sys-color-state-hover)] transition-colors"
            >
              <Icon
                name="heart"
                weight={currentTrack.isFavorite ? 'fill' : 'regular'}
                size={18}
                style={currentTrack.isFavorite ? 'color: var(--md-sys-color-tertiary);' : 'color: var(--md-sys-color-on-surface-variant);'}
              />
            </button>
          </div>
        {:else}
          <div class="py-8 px-4 text-center space-y-2">
            <Icon name="disc" size={28} class="text-[var(--md-sys-color-on-surface-variant)] opacity-35" />
            <div class="text-xs font-medium text-[var(--md-sys-color-on-surface-variant)]">
              Belum ada lagu diputar
            </div>
          </div>
        {/if}
      </section>

      <!-- CARD 2: SPOTIFY 1:1 LYRICS CARD (Single card, smooth max-height expand/collapse) -->
      <section
        bind:this={lyricsCardEl}
        class="group/lyrics rounded-lg p-4 relative overflow-hidden"
        style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
      >
        <div class="flex items-center justify-between gap-2 mb-3">
          <span class="text-[15px] font-bold">Lirik</span>

          <button
            type="button"
            onclick={() => onOpenLyrics?.()}
            title="Perluas lirik"
            aria-label="Perluas lirik"
            class="w-7 h-7 rounded-full flex items-center justify-center opacity-75 hover:opacity-100 hover:bg-[var(--md-sys-color-state-hover)] transition-colors cursor-pointer"
          >
            <Icon name="arrows-out-simple" size={16} />
          </button>
        </div>

        <div class="relative">
          <div
            bind:this={lyricsScrollEl}
            class="overflow-hidden pr-1 space-y-2.5 pb-12"
            style="max-height: {lyricsExpanded ? '420px' : '152px'}; transition: max-height 380ms cubic-bezier(0.22, 1, 0.36, 1);"
          >
            {#if !currentTrack}
              <p class="text-[14px] font-semibold opacity-80 py-3">
                Putar lagu untuk menampilkan lirik.
              </p>
            {:else if lyricsLoading}
              <div class="py-6 flex items-center justify-center" aria-label="Memuat lirik">
                <span class="velora-eq-loader" style="--loader-color: var(--md-sys-color-on-primary-container);" aria-hidden="true">
                  <span class="velora-eq-bar"></span>
                  <span class="velora-eq-bar"></span>
                  <span class="velora-eq-bar"></span>
                  <span class="velora-eq-bar"></span>
                  <span class="velora-eq-bar"></span>
                </span>
              </div>
            {:else if lyricLines.length > 0}
              {#each lyricLines as line, idx (idx + '-' + line.text)}
                {@const active = idx === activeSyncedIndex}
                <button
                  type="button"
                  data-lyric-idx={idx}
                  onclick={() => line.isSynced && line.timeSec !== null && onSeek(line.timeSec)}
                  class="w-full text-left font-bold {lyricsExpandedSizeClass} origin-left transition-all duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] cursor-pointer {active
                    ? 'opacity-100 scale-[1.02]'
                    : 'opacity-35 scale-[0.93] hover:opacity-100'}"
                >
                  {line.text}
                </button>
              {/each}
            {:else}
              <p class="text-[14px] font-semibold text-[var(--md-sys-color-on-primary-container)] opacity-75 py-4">
                Lirik tidak ditemukan untuk "{currentTrack.title}".
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
              class="text-[12px] font-bold px-4 py-1.5 rounded-full cursor-pointer opacity-0 translate-y-3 pointer-events-none group-hover/lyrics:opacity-100 group-hover/lyrics:translate-y-0 group-hover/lyrics:pointer-events-auto transition-all duration-300 ease-out hover:scale-[1.03] active:scale-[0.98]"
            >
              {lyricsExpanded ? 'Lihat lebih sedikit' : 'Lihat lebih banyak'}
            </button>
          </div>
        </div>
      </section>

      <!-- CARD 3: SPOTIFY 1:1 QUEUE CARD (Antrean berikutnya) -->
      <section
        bind:this={queueCardEl}
        class="rounded-lg p-4"
        style="background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface);"
      >
        <div class="flex items-center justify-between gap-2 mb-3.5">
          <span class="text-[15px] font-bold">Antrean berikutnya</span>
          <button
            type="button"
            onclick={() => (onChangeMode ? onChangeMode('queue') : (queueExpanded = !queueExpanded))}
            class="text-[13px] font-bold text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)] hover:underline cursor-pointer transition-colors shrink-0"
          >
            Buka antrean
          </button>
        </div>

        {#if !nextTrack}
          <p class="text-xs font-medium text-[var(--md-sys-color-on-surface-variant)] py-2">
            Antrean berikutnya masih kosong.
          </p>
        {:else if !queueExpanded}
          {@const nextCover = resolveCoverUrl(nextTrack.coverUrl, mediaBaseUrl)}
          <div transition:fade={{ duration: 160 }}>
            <button
              type="button"
              onclick={() => onPlayTrack(nextTrack, queue)}
              class="w-full flex items-center gap-3 p-1 -m-1 rounded-md text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)] transition-colors"
            >
              <div class="w-12 h-12 rounded-md overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)]">
                {#if nextCover}
                  <img
                    src={nextCover}
                    alt={nextTrack.title}
                    class="block w-full h-full object-cover object-center"
                  />
                {:else}
                  <div class="w-full h-full flex items-center justify-center text-[var(--md-sys-color-on-surface-variant)] opacity-45">
                    <Icon name="music-note" size={18} />
                  </div>
                {/if}
              </div>

              <div class="min-w-0 flex-1">
                <div class="text-[15px] font-semibold truncate leading-snug">
                  {nextTrack.title}
                </div>
                <div class="text-[13px] text-[var(--md-sys-color-on-surface-variant)] truncate mt-0.5">
                  {nextTrack.artistName}
                </div>
              </div>
            </button>
          </div>
        {:else}
          <div transition:slide={{ duration: 240 }} class="max-h-60 overflow-y-auto overscroll-contain pr-1 space-y-2">
            {#each upcomingQueue as qt (qt.id)}
              {@const qCover = resolveCoverUrl(qt.coverUrl, mediaBaseUrl)}
              <button
                type="button"
                onclick={() => onPlayTrack(qt, queue)}
                class="w-full flex items-center gap-3 p-1.5 rounded-md text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)] transition-colors"
              >
                <div class="w-11 h-11 rounded-md overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)]">
                  {#if qCover}
                    <img
                      src={qCover}
                      alt={qt.title}
                      class="block w-full h-full object-cover object-center"
                    />
                  {:else}
                    <div class="w-full h-full flex items-center justify-center text-[var(--md-sys-color-on-surface-variant)] opacity-45">
                      <Icon name="music-note" size={16} />
                    </div>
                  {/if}
                </div>

                <div class="min-w-0 flex-1">
                  <div class="text-[14px] font-semibold truncate leading-snug">
                    {qt.title}
                  </div>
                  <div class="text-[12px] text-[var(--md-sys-color-on-surface-variant)] truncate mt-0.5">
                    {qt.artistName}
                  </div>
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </section>
      </div>
      {/if}
    </div>
    </div>

    <!-- COLLAPSED OVERLAY LAYER: Transparent gradient fade revealing peeked content + centered "<" chevron -->
    <div
      aria-hidden={!collapsed}
      class="absolute inset-0 z-10 flex items-center justify-center transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] {collapsed
        ? 'opacity-100 pointer-events-auto'
        : 'opacity-0 pointer-events-none'}"
    >
      <div
        class="pointer-events-none absolute inset-0 transition-opacity duration-300 {collapsedHover
          ? 'opacity-100'
          : 'opacity-0'}"
        style="background: linear-gradient(to right, rgba(0, 0, 0, 0.25) 0%, rgba(0, 0, 0, 0.72) 100%);"
      ></div>
      <button
        type="button"
        onclick={onToggleCollapse}
        tabindex={collapsed ? 0 : -1}
        title="Expand Right Panel"
        aria-label="Expand Right Panel"
        class="relative z-10 w-full h-full flex items-center justify-center cursor-pointer transition-colors {collapsedHover
          ? 'text-white'
          : 'text-[var(--md-sys-color-on-surface-variant)]'} active:scale-95"
      >
        <span
          bind:this={collapseIconEl}
          class="inline-flex items-center justify-center"
          style="transform: scaleX(1);"
        >
          <Icon name="caret-left" weight="bold" size={18} />
        </span>
      </button>
    </div>
  </div>
</aside>
