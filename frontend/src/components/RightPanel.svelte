<script lang="ts">
  import { untrack } from 'svelte';
  import { animate } from 'motion';
  import { cubicOut } from 'svelte/easing';
  import { fade, fly, slide } from 'svelte/transition';
  import { isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { veloraMarquee } from '../lib/marquee';
  import { svelteMotionEnter } from '../lib/motion';
  import { getPrefValue } from '../lib/preferences';
  import { getPrefLang, tr } from '../lib/i18n';
  import type { AudioOutputDevice, Track, TrackLyrics } from '../lib/types';
  import EmptyState from './EmptyState.svelte';
  import EqIndicator from './EqIndicator.svelte';
  import Icon from './Icon.svelte';
  import LyricsPreviewCard from './LyricsPreviewCard.svelte';

  interface Props {
    width: number;
    collapsed: boolean;
    mode?: 'now-playing' | 'queue' | 'devices';
    currentTrack: Track | null;
    queue: Track[];
    queueIndex?: number;
    queueSourceTitle?: string;
    audioOutputs?: AudioOutputDevice[];
    switchingOutputId?: string | null;
    isPlaying: boolean;
    positionSec: number;
    lyricOffsetSec?: number;
    lyrics: TrackLyrics | null;
    lyricsLoading: boolean;
    preferences: Record<string, string>;
    mediaBaseUrl: string;
    accentColor: string;
    embeddedMobile?: boolean;
    focusCardSignal?: { card: 'album' | 'lyrics' | 'queue'; seq: number } | null;
    onChangeMode?: (mode: 'now-playing' | 'queue' | 'devices') => void;
    onSelectOutputDevice?: (deviceId: string) => void;
    onRefreshAudioOutputs?: () => void;
    onToggleCollapse: () => void;
    onResizeWidth: (newWidth: number) => void;
    onResizeEnd: (finalWidth: number) => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onRemoveFromQueue: (trackId: number) => void;
    onClearQueue: () => void;
    onToggleFavorite: (trackId: number) => void;
    onSeek: (sec: number) => void;
    onOpenTrackDetail?: (track: Track) => void;
    onOpenArtistDetail?: (track: Track) => void;
    onOpenLyrics?: () => void;
    onResyncLyrics?: () => void;
    onOpenQueueSource?: () => void;
  }

  let {
    width,
    collapsed,
    mode = 'now-playing',
    currentTrack,
    queue,
    queueIndex = -1,
    queueSourceTitle = '',
    audioOutputs = [],
    switchingOutputId = null,
    isPlaying,
    positionSec,
    lyricOffsetSec = 0,
    lyrics,
    lyricsLoading,
    preferences,
    mediaBaseUrl,
    embeddedMobile = false,
    focusCardSignal = null,
    onChangeMode,
    onSelectOutputDevice,
    onRefreshAudioOutputs,
    onToggleCollapse,
    onResizeWidth,
    onResizeEnd,
    onPlayTrack,
    onRemoveFromQueue: _onRemoveFromQueue,
    onClearQueue: _onClearQueue,
    onToggleFavorite,
    onSeek,
    onOpenTrackDetail,
    onOpenArtistDetail,
    onOpenLyrics,
    onResyncLyrics,
    onOpenQueueSource
  }: Props = $props();

  let panelScrollEl: HTMLElement | undefined = $state();
  let albumCardEl: HTMLElement | undefined = $state();
  let lyricsCardEl: HTMLElement | undefined = $state();
  let queueCardEl: HTMLElement | undefined = $state();
  let collapseIconEl: HTMLElement | undefined = $state();
  let prevCollapsed: boolean | undefined;
  let flipAnim: { stop?: () => void } | undefined;
  let isResizing = $state(false);
  let lyricsExpanded = $state(false);
  let queueExpanded = $state(false);

  const MIN_WIDTH = 280;
  const MAX_WIDTH = 520;
  const COLLAPSED_WIDTH = 36;
  const COLLAPSED_HOVER_WIDTH = 72;
  let collapsedHover = $state(false);

  const lang = $derived(getPrefLang(preferences));

  function outputDeviceIcon(connType: string): string {
    if (connType === 'bluetooth' || connType === 'wired') return 'headphones';
    return 'speaker-high';
  }

  const activeOutputDev = $derived(
    audioOutputs.find((d) => d.isActive) ?? audioOutputs[0] ?? null
  );
  const otherOutputDevs = $derived(
    audioOutputs.filter((d) => !d.isActive && (!activeOutputDev || d.id !== activeOutputDev.id))
  );

  const upcomingQueue = $derived.by(() => {
    if (queue.length === 0) return [];
    if (!currentTrack) return queue;
    const idx =
      queueIndex >= 0 && queueIndex < queue.length && queue[queueIndex].id === currentTrack.id
        ? queueIndex
        : queue.findIndex((t) => t.id === currentTrack.id);
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

<aside
  aria-label="Right Panel Section"
  use:svelteMotionEnter={{ y: embeddedMobile ? 0 : 8 }}
  onmouseenter={() => {
    if (collapsed) collapsedHover = true;
  }}
  onmouseleave={() => {
    collapsedHover = false;
  }}
  class="relative shrink-0 h-full min-h-0 flex flex-col select-none overflow-visible {embeddedMobile
    ? 'w-full rounded-none'
    : collapsed
      ? 'rounded-l-lg rounded-r-none'
      : 'rounded-lg'}"
  style="width: {embeddedMobile ? '100%' : `${collapsed ? (collapsedHover ? COLLAPSED_HOVER_WIDTH : COLLAPSED_WIDTH) : width}px`}; background-color: {embeddedMobile ? 'transparent' : 'var(--md-sys-color-surface-container-low)'}; transition: {isResizing || embeddedMobile ? 'none' : 'width 360ms cubic-bezier(0.22, 1, 0.36, 1), background-color 300ms ease'};"
>
  {#if !collapsed && !embeddedMobile}
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
    <div
      aria-hidden={collapsed}
      class="absolute inset-y-0 left-0 flex flex-col transition-all duration-[360ms] ease-[cubic-bezier(0.22,1,0.36,1)] {collapsed
        ? collapsedHover
          ? 'opacity-55 translate-x-0 scale-100 pointer-events-none'
          : 'opacity-0 translate-x-10 scale-[0.98] pointer-events-none'
        : 'opacity-100 translate-x-0 scale-100 pointer-events-auto delay-75'}"
      style="width: {collapsed ? `${width}px` : '100%'}; {collapsed ? '' : 'right: 0;'}"
    >
    <div class="px-3.5 py-2.5 flex items-center justify-between gap-2 shrink-0">
      <div class="flex items-center gap-2 min-w-0">
        <button
          type="button"
          onclick={onToggleCollapse}
          title={tr(lang, 'nav.collapseRightPanel')}
          aria-label={tr(lang, 'nav.collapseRightPanel')}
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
          {mode === 'queue'
            ? tr(lang, 'nav.queue')
            : mode === 'devices'
              ? tr(lang, 'nav.connectDevice')
              : tr(lang, 'nav.nowPlaying')}
        </span>
      </div>
      {#if (mode === 'queue' || mode === 'devices') && onChangeMode}
        <button
          type="button"
          onclick={() => onChangeMode('now-playing')}
          title={tr(lang, 'nav.backToNowPlaying')}
          aria-label={tr(lang, 'nav.backToNowPlaying')}
          class="m3-icon-btn w-7 h-7 flex items-center justify-center cursor-pointer shrink-0"
        >
          <Icon name="x" size={15} />
        </button>
      {/if}
    </div>

    <div class="relative flex-1 min-h-0 flex flex-col">
      <div
        class="pointer-events-none absolute inset-x-0 top-0 h-5 z-10"
        style="background: linear-gradient(to bottom, var(--md-sys-color-surface-container-low) 0%, transparent 100%);"
      ></div>
      <div bind:this={panelScrollEl} class="flex-1 min-h-0 overflow-y-auto overflow-x-hidden overscroll-contain px-3 pt-2 pb-4 flex flex-col">
      {#key mode}
      <div
        in:fly={{ x: mode === 'now-playing' ? -20 : 44, duration: 280, easing: cubicOut }}
        class="flex-1 flex flex-col min-h-0"
      >
      {#if mode === 'devices'}
        <div class="flex-1 flex flex-col justify-between gap-6 pt-1">
          <div class="space-y-5">
            {#if activeOutputDev}
              <div
                class="flex items-center gap-3 p-3 rounded-xl"
                style="background-color: var(--md-sys-color-surface-container);"
              >
                <div
                  class="w-9 h-9 rounded-full flex items-center justify-center shrink-0"
                  style="color: var(--md-sys-color-primary);"
                >
                  <Icon name={outputDeviceIcon(activeOutputDev.connType)} weight="bold" size={22} />
                </div>
                <div class="min-w-0 flex-1">
                  <div
                    class="text-[14px] font-bold truncate leading-snug"
                    style="color: var(--md-sys-color-primary);"
                  >
                    {activeOutputDev.name}
                  </div>
                  <div
                    class="text-[12px] truncate mt-0.5 flex items-center gap-1.5"
                    style="color: var(--md-sys-color-on-surface-variant);"
                  >
                    <span>{tr(lang, 'nav.thisComputer')}</span>
                    {#if activeOutputDev.batPct >= 0 && activeOutputDev.batPct <= 100}
                      <span>· {activeOutputDev.batPct}%</span>
                    {/if}
                  </div>
                </div>
              </div>
            {/if}

            {#if otherOutputDevs.length > 0}
              <section class="space-y-2">
                <h3 class="px-1.5 text-[13px] font-bold" style="color: var(--md-sys-color-on-surface);">
                  {tr(lang, 'nav.otherDevices')}
                </h3>
                <div class="space-y-1">
                  {#each otherOutputDevs as dev (dev.id)}
                    {@const isSwitching = switchingOutputId === dev.id}
                    <button
                      type="button"
                      disabled={isSwitching}
                      onclick={() => onSelectOutputDevice?.(dev.id)}
                      class="w-full flex items-center gap-3 px-2.5 py-2.5 rounded-xl text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)] transition-colors disabled:opacity-60"
                    >
                      <div
                        class="w-9 h-9 rounded-full flex items-center justify-center shrink-0"
                        style="color: var(--md-sys-color-on-surface-variant);"
                      >
                        <Icon name={outputDeviceIcon(dev.connType)} size={20} />
                      </div>
                      <div class="min-w-0 flex-1">
                        <div class="text-[14px] font-semibold truncate leading-snug" style="color: var(--md-sys-color-on-surface);">
                          {dev.name}
                        </div>
                        <div class="text-[12px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                          {tr(lang, 'nav.availableDevice')}
                          {#if dev.batPct >= 0 && dev.batPct <= 100}
                            · {dev.batPct}%
                          {/if}
                        </div>
                      </div>
                    </button>
                  {/each}
                </div>
              </section>
            {:else}
              <section class="px-1.5 space-y-4">
                <h3 class="text-[15px] font-bold leading-snug" style="color: var(--md-sys-color-on-surface);">
                  {tr(lang, 'nav.noOtherDevicesFound')}
                </h3>

                <div class="space-y-4">
                  <div class="flex items-start gap-3">
                    <span class="mt-0.5 shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                      <Icon name="wifi-high" size={19} />
                    </span>
                    <div class="min-w-0">
                      <div class="text-[13px] font-bold leading-snug" style="color: var(--md-sys-color-on-surface);">
                        {tr(lang, 'nav.checkWifi')}
                      </div>
                      <p class="text-[12px] leading-relaxed mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                        {tr(lang, 'nav.checkWifiDesc')}
                      </p>
                    </div>
                  </div>

                  <div class="flex items-start gap-3">
                    <span class="mt-0.5 shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                      <Icon name="devices" size={19} />
                    </span>
                    <div class="min-w-0">
                      <div class="text-[13px] font-bold leading-snug" style="color: var(--md-sys-color-on-surface);">
                        {tr(lang, 'nav.playFromAnotherDevice')}
                      </div>
                      <p class="text-[12px] leading-relaxed mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                        {tr(lang, 'nav.playFromAnotherDeviceDesc')}
                      </p>
                    </div>
                  </div>

                  <div class="flex items-start gap-3">
                    <span class="mt-0.5 shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                      <Icon name="power" size={19} />
                    </span>
                    <div class="min-w-0">
                      <div class="text-[13px] font-bold leading-snug" style="color: var(--md-sys-color-on-surface);">
                        {tr(lang, 'nav.restartSpeaker')}
                      </div>
                      <p class="text-[12px] leading-relaxed mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                        {tr(lang, 'nav.restartSpeakerDesc')}
                      </p>
                    </div>
                  </div>
                </div>
              </section>
            {/if}
          </div>

          <button
            type="button"
            onclick={() => onRefreshAudioOutputs?.()}
            title={tr(lang, 'nav.scanAgain')}
            class="px-1.5 py-2 rounded-lg flex items-center justify-between gap-2 text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)] transition-colors"
          >
            <span class="text-[13px] font-bold truncate" style="color: var(--md-sys-color-on-surface);">
              {tr(lang, 'nav.dontSeeDevice')}
            </span>
            <span class="shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
              <Icon name="arrows-clockwise" size={17} />
            </span>
          </button>
        </div>
      {:else if mode === 'queue'}
        {#if !currentTrack && upcomingQueue.length === 0}
          <EmptyState
            icon="queue"
            title={tr(lang, 'nav.emptyQueue')}
            description={tr(lang, 'nav.emptyQueueDesc')}
            titleClass="text-lg font-bold tracking-tight"
            descClass="mt-1.5 text-[13px] max-w-[230px] mx-auto"
          />
        {:else}
        <div class="space-y-4 pt-1">
          <section class="space-y-2">
            <h3 class="px-1.5 text-[14px] font-bold" style="color: var(--md-sys-color-on-surface);">
              {tr(lang, 'nav.nowPlaying')}
            </h3>
            {#if currentTrack}
              {@const activeCover = resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl, 96)}
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
                {tr(lang, 'nav.noSongPlaying')}
              </p>
            {/if}
          </section>

          <section class="space-y-2">
            <h3 class="px-1.5 text-[14px] font-bold leading-snug" style="color: var(--md-sys-color-on-surface);">
              {#if queueSourceTitle}
                {tr(lang, 'nav.nextFrom')}
                {#if onOpenQueueSource}
                  <button
                    type="button"
                    onclick={onOpenQueueSource}
                    title={queueSourceTitle}
                    class="hover:underline underline-offset-2 cursor-pointer text-left inline font-bold transition-opacity hover:opacity-85"
                    style="color: var(--md-sys-color-on-surface);"
                  >
                    {queueSourceTitle}
                  </button>
                {:else}
                  {queueSourceTitle}
                {/if}
              {:else}
                {tr(lang, 'nav.upNext')}
              {/if}
            </h3>
            {#if upcomingQueue.length === 0}
              <p class="px-1.5 text-xs py-2" style="color: var(--md-sys-color-on-surface-variant);">
                {tr(lang, 'nav.emptyQueue')}
              </p>
            {:else}
              <div class="space-y-1">
                {#each upcomingQueue as qt, qIdx (`${qIdx}-${qt.id}-${qt.externalId || qt.title}`)}
                  {@const qCover = resolveCoverUrl(qt.coverUrl, mediaBaseUrl, 96)}
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
        {/if}
      {:else if !currentTrack}
        <EmptyState
          icon="music-note"
          title={tr(lang, 'nav.noSongPlaying')}
          description={tr(lang, 'nav.playSongForLyrics')}
          titleClass="text-lg font-bold tracking-tight"
          descClass="mt-1.5 text-[13px] max-w-[230px] mx-auto"
        />
      {:else}
      <div in:fly={{ x: -16, duration: 240 }} class="space-y-3">
      <section
        bind:this={albumCardEl}
        class="rounded-lg overflow-hidden"
        style="background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface);"
      >
        {#if currentTrack}
          {@const cover = resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl, 544)}
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
                use:veloraMarquee={{ text: currentTrack.title, disabled: collapsed }}
                class="w-full overflow-hidden text-left text-[16px] font-bold leading-snug hover:underline cursor-pointer"
                title={tr(lang, 'nav.viewTrackDetails')}
              >
                <span class="inline-block whitespace-nowrap">
                  {currentTrack.title}
                </span>
              </button>
              <button
                type="button"
                onclick={() => onOpenArtistDetail?.(currentTrack)}
                use:veloraMarquee={{ text: currentTrack.artistName, disabled: collapsed }}
                class="w-full overflow-hidden text-left text-[13px] font-medium text-[var(--md-sys-color-on-surface-variant)] mt-0.5 hover:underline cursor-pointer"
                title={tr(lang, 'nav.viewArtistPage')}
              >
                <span class="inline-block whitespace-nowrap">
                  {currentTrack.artistName}
                </span>
              </button>
            </div>

            <button
              type="button"
              onclick={() => onToggleFavorite(currentTrack.id)}
              aria-label={tr(lang, 'nav.likeSong')}
              class="w-8 h-8 rounded-full flex items-center justify-center shrink-0 cursor-pointer hover:bg-[var(--md-sys-color-state-hover)] velora-heart-btn {currentTrack.isFavorite ? 'is-loved' : ''}"
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
              {tr(lang, 'nav.noSongPlaying')}
            </div>
          </div>
        {/if}
      </section>

      <LyricsPreviewCard
        bind:cardEl={lyricsCardEl}
        bind:lyricsExpanded={lyricsExpanded}
        {currentTrack}
        {isPlaying}
        {positionSec}
        {lyricOffsetSec}
        {lyrics}
        {lyricsLoading}
        {preferences}
        {collapsed}
        {onSeek}
        {onOpenLyrics}
        onResync={onResyncLyrics}
      />

      <section
        bind:this={queueCardEl}
        class="rounded-lg p-4"
        style="background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface);"
      >
        <div class="flex items-center justify-between gap-2 mb-3.5">
          <span class="text-[15px] font-bold">{tr(lang, 'nav.upNext')}</span>
          <button
            type="button"
            onclick={() => (onChangeMode ? onChangeMode('queue') : (queueExpanded = !queueExpanded))}
            class="text-[13px] font-bold text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)] hover:underline cursor-pointer transition-colors shrink-0"
          >
            {tr(lang, 'nav.openQueue')}
          </button>
        </div>

        {#if !nextTrack}
          <p class="text-xs font-medium text-[var(--md-sys-color-on-surface-variant)] py-2">
            {tr(lang, 'nav.emptyQueue')}
          </p>
        {:else if !queueExpanded}
          {@const nextCover = resolveCoverUrl(nextTrack.coverUrl, mediaBaseUrl, 96)}
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
            {#each upcomingQueue as qt, qIdx (`${qIdx}-${qt.id}-${qt.externalId || qt.title}`)}
              {@const qCover = resolveCoverUrl(qt.coverUrl, mediaBaseUrl, 96)}
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
      {/key}
      </div>
      <div
        class="pointer-events-none absolute inset-x-0 bottom-0 h-7 z-10"
        style="background: linear-gradient(to top, var(--md-sys-color-surface-container-low) 0%, transparent 100%);"
      ></div>
    </div>
    </div>

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
        title={tr(lang, 'nav.expandRightPanel')}
        aria-label={tr(lang, 'nav.expandRightPanel')}
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
