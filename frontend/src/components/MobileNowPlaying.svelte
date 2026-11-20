<script lang="ts">
  import { fly } from 'svelte/transition';
  import { cubicIn, cubicOut } from 'svelte/easing';
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import type { Track, TrackLyrics } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    open: boolean;
    currentTrack: Track | null;
    lyrics: TrackLyrics | null;
    queue: Track[];
    isPlaying: boolean;
    positionSec: number;
    durationSec: number;
    shuffle: boolean;
    repeatMode: 'off' | 'all' | 'one';
    mediaBaseUrl: string;
    onClose: () => void;
    onTogglePlay: () => void;
    onPrev: () => void;
    onNext: () => void;
    onSeek: (sec: number) => void;
    onToggleShuffle: () => void;
    onCycleRepeat: () => void;
    onToggleFavorite: (trackId: number) => void;
    onPlayTrack: (tr: Track, list?: Track[]) => void;
    onOpenLyrics: () => void;
    onOpenQueue: () => void;
  }

  let {
    open,
    currentTrack,
    lyrics,
    queue,
    isPlaying,
    positionSec,
    durationSec,
    shuffle,
    repeatMode,
    mediaBaseUrl,
    onClose,
    onTogglePlay,
    onPrev,
    onNext,
    onSeek,
    onToggleShuffle,
    onCycleRepeat,
    onToggleFavorite,
    onPlayTrack,
    onOpenLyrics,
    onOpenQueue
  }: Props = $props();

  const cover = $derived(currentTrack ? resolveCoverUrl(currentTrack.coverUrl, mediaBaseUrl) : '');
  const effectiveDuration = $derived(durationSec > 0 ? durationSec : currentTrack?.durationSec || 0);
  const progressPct = $derived(
    effectiveDuration > 0 ? Math.min(100, Math.max(0, (positionSec / effectiveDuration) * 100)) : 0
  );

  const previewLines = $derived.by(() => {
    if (!lyrics) return [];
    if (lyrics.syncedLyrics && lyrics.syncedLyrics.length > 0) {
      let activeIdx = 0;
      for (let i = 0; i < lyrics.syncedLyrics.length; i++) {
        if (lyrics.syncedLyrics[i].timeSec <= positionSec + 0.35) activeIdx = i;
        else break;
      }
      const start = Math.max(0, activeIdx - 1);
      return lyrics.syncedLyrics.slice(start, start + 5).map((l, i) => ({
        text: l.text || '♪',
        active: start + i === activeIdx
      }));
    }
    if (lyrics.plainLyrics) {
      return lyrics.plainLyrics
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean)
        .slice(0, 5)
        .map((text, idx) => ({ text, active: idx === 0 }));
    }
    return [];
  });

  const upcomingQueue = $derived.by(() => {
    if (!currentTrack) return queue;
    const idx = queue.findIndex(
      (t) =>
        (currentTrack.id > 0 && t.id === currentTrack.id) ||
        (currentTrack.externalId && t.externalId === currentTrack.externalId)
    );
    if (idx < 0) return queue.filter((t) => t.id !== currentTrack.id);
    return queue.slice(idx + 1);
  });

  const nextTrack = $derived(upcomingQueue[0] ?? null);
</script>

{#if open}
  <div
    in:fly={{ y: 480, duration: 280, easing: cubicOut }}
    out:fly={{ y: 480, duration: 220, easing: cubicIn }}
    class="fixed inset-0 z-50 overflow-y-auto px-5 pt-4 pb-10 select-none sm:hidden"
    style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary-container) 70%, var(--md-sys-color-background)) 0%, var(--md-sys-color-background) 42%); color: var(--md-sys-color-on-surface);"
  >
    <div class="max-w-md mx-auto space-y-6">
      <!-- Top Header -->
      <div class="flex items-center justify-between gap-3">
        <button
          type="button"
          onclick={onClose}
          aria-label="Tutup Now Playing"
          class="m3-icon-btn w-10 h-10 flex items-center justify-center cursor-pointer"
        >
          <Icon name="caret-down" weight="bold" size={22} />
        </button>

        <div class="min-w-0 text-center">
          <div class="text-[10px] font-bold uppercase tracking-widest opacity-75">Playing from</div>
          <div class="text-xs font-bold truncate max-w-[200px]">
            {currentTrack?.albumTitle || 'Velora'}
          </div>
        </div>

        <button
          type="button"
          onclick={() => {
            onClose();
            onOpenQueue();
          }}
          aria-label="Buka Queue"
          class="m3-icon-btn w-10 h-10 flex items-center justify-center cursor-pointer"
        >
          <Icon name="dots-three-vertical" weight="bold" size={20} />
        </button>
      </div>

      <!-- Large Album Artwork -->
      <div class="w-full aspect-square rounded-xl overflow-hidden shadow-2xl bg-[var(--md-sys-color-surface-container-high)] flex items-center justify-center">
        {#if cover}
          <img
            src={cover}
            alt={currentTrack?.title || 'Cover'}
            class="block w-full h-full object-cover object-center"
            class:velora-yt-video-thumb={isYTVideoThumb(cover)}
          />
        {:else}
          <Icon name="music-note" size={56} style="opacity: 0.45;" />
        {/if}
      </div>

      <!-- Track Meta + Favorite -->
      <div class="space-y-4">
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0 flex-1">
            <div class="text-xl font-bold tracking-tight truncate">
              {currentTrack?.title || 'Belum ada lagu diputar'}
            </div>
            <div class="text-sm truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
              {currentTrack?.artistName || 'Velora'}
            </div>
          </div>

          <button
            type="button"
            disabled={!currentTrack}
            onclick={() => currentTrack && onToggleFavorite(currentTrack.id)}
            aria-label="Sukai lagu"
            class="w-10 h-10 rounded-full flex items-center justify-center shrink-0 cursor-pointer disabled:opacity-40"
          >
            <Icon
              name={currentTrack?.isFavorite ? 'check-circle' : 'plus-circle'}
              weight={currentTrack?.isFavorite ? 'fill' : 'regular'}
              size={28}
              style={currentTrack?.isFavorite ? 'color: var(--md-sys-color-tertiary);' : 'color: var(--md-sys-color-on-surface);'}
            />
          </button>
        </div>

        <!-- Seek Bar -->
        <div class="space-y-1.5">
          <input
            type="range"
            min="0"
            max={effectiveDuration > 0 ? effectiveDuration : 100}
            step="0.5"
            value={Math.min(positionSec, effectiveDuration || 100)}
            oninput={(e) => onSeek(Number((e.target as HTMLInputElement).value))}
            aria-label="Seek position"
            class="velora-range w-full block"
            style="background: linear-gradient(to right, var(--md-sys-color-on-surface) {progressPct}%, var(--md-sys-color-surface-container-highest) {progressPct}%);"
          />
          <div class="flex items-center justify-between text-[11px] tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
            <span>{formatDuration(positionSec)}</span>
            <span>{formatDuration(effectiveDuration)}</span>
          </div>
        </div>

        <!-- Spotify-style Main Transport Controls -->
        <div class="flex items-center justify-between px-1">
          <button
            type="button"
            onclick={onToggleShuffle}
            aria-label="Shuffle"
            class="w-10 h-10 flex items-center justify-center cursor-pointer"
            style={shuffle ? 'color: var(--md-sys-color-tertiary);' : 'color: var(--md-sys-color-on-surface-variant);'}
          >
            <Icon name="shuffle" weight={shuffle ? 'bold' : 'regular'} size={22} />
          </button>

          <button
            type="button"
            onclick={onPrev}
            aria-label="Sebelumnya"
            class="w-12 h-12 flex items-center justify-center cursor-pointer"
          >
            <Icon name="skip-back" weight="fill" size={28} />
          </button>

          <button
            type="button"
            onclick={onTogglePlay}
            aria-label={isPlaying ? 'Jeda' : 'Putar'}
            class="w-16 h-16 rounded-full flex items-center justify-center shadow-xl cursor-pointer transition-transform active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            <Icon name={isPlaying ? 'pause' : 'play'} weight="fill" size={28} />
          </button>

          <button
            type="button"
            onclick={onNext}
            aria-label="Berikutnya"
            class="w-12 h-12 flex items-center justify-center cursor-pointer"
          >
            <Icon name="skip-forward" weight="fill" size={28} />
          </button>

          <button
            type="button"
            onclick={onCycleRepeat}
            aria-label="Repeat"
            class="w-10 h-10 flex items-center justify-center cursor-pointer"
            style={repeatMode !== 'off' ? 'color: var(--md-sys-color-tertiary);' : 'color: var(--md-sys-color-on-surface-variant);'}
          >
            <Icon name={repeatMode === 'one' ? 'repeat-once' : 'repeat'} weight={repeatMode !== 'off' ? 'bold' : 'regular'} size={22} />
          </button>
        </div>

        <!-- Action Icons Row -->
        <div class="flex items-center justify-between pt-1">
          <button
            type="button"
            onclick={() => {
              onClose();
              onOpenLyrics();
            }}
            aria-label="Buka Lirik"
            class="w-9 h-9 rounded-full flex items-center justify-center cursor-pointer"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="microphone-stage" size={19} />
          </button>
          <button
            type="button"
            onclick={() => {
              onClose();
              onOpenQueue();
            }}
            aria-label="Buka Queue"
            class="w-9 h-9 rounded-full flex items-center justify-center cursor-pointer"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="queue" size={19} />
          </button>
        </div>
      </div>

      <!-- Lyrics Preview Card (Spotify style) -->
      <section
        class="rounded-2xl p-4 space-y-3"
        style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-tertiary-container) 82%, #6d4c1d), color-mix(in srgb, var(--md-sys-color-tertiary-container) 55%, #3d2a0d));"
      >
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-bold text-white">Lyrics preview</h3>
        </div>
        {#if previewLines.length > 0}
          <div class="space-y-1.5">
            {#each previewLines as line, idx (idx)}
              <p class="text-base leading-snug font-bold {line.active ? 'text-white' : 'text-white/70'}">
                {line.text}
              </p>
            {/each}
          </div>
        {:else}
          <p class="text-sm text-white/75">Lirik belum tersedia untuk lagu ini.</p>
        {/if}
        <button
          type="button"
          onclick={() => {
            onClose();
            onOpenLyrics();
          }}
          class="h-8 px-4 rounded-full text-xs font-bold bg-white text-black cursor-pointer"
        >
          Show lyrics
        </button>
      </section>

      <!-- Antrean Berikutnya (Sama seperti Desktop RightPanel) -->
      <section
        class="rounded-2xl p-4"
        style="background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface);"
      >
        <div class="flex items-center justify-between gap-2 mb-3.5">
          <span class="text-[15px] font-bold">Antrean berikutnya</span>
          <button
            type="button"
            onclick={() => {
              onClose();
              onOpenQueue();
            }}
            class="text-[13px] font-bold text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)] hover:underline cursor-pointer transition-colors shrink-0"
          >
            Buka antrean
          </button>
        </div>

        {#if !nextTrack}
          <p class="text-xs font-medium text-[var(--md-sys-color-on-surface-variant)] py-2">
            Antrean berikutnya masih kosong.
          </p>
        {:else}
          {@const nextCover = resolveCoverUrl(nextTrack.coverUrl, mediaBaseUrl)}
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
        {/if}
      </section>
    </div>
  </div>
{/if}
