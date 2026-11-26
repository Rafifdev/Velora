<script lang="ts">
  import { fly } from 'svelte/transition';
  import { cubicIn, cubicOut } from 'svelte/easing';
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import { veloraMarquee } from '../lib/marquee';
  import type { Playlist, Track, TrackLyrics, YTMusicTrack } from '../lib/types';
  import CoverImage from './CoverImage.svelte';
  import Icon from './Icon.svelte';
  import LyricsPreviewCard from './LyricsPreviewCard.svelte';
  import TrackActionMenu from './TrackActionMenu.svelte';

  interface Props {
    open: boolean;
    currentTrack: Track | null;
    lyrics: TrackLyrics | null;
    lyricsLoading?: boolean;
    lyricOffsetSec?: number;
    queue: Track[];
    queueIndex?: number;
    allTracks?: Track[];
    ytRecommendedTracks?: YTMusicTrack[];
    playlists?: Playlist[];
    preferences?: Record<string, string>;
    isPlaying: boolean;
    positionSec: number;
    durationSec: number;
    shuffle: boolean;
    repeatMode: 'off' | 'all' | 'one';
    mediaBaseUrl: string;
    lang?: LangCode;
    onClose: () => void;
    onTogglePlay: () => void;
    onPrev: () => void;
    onNext: () => void;
    onSeek: (sec: number) => void;
    onToggleShuffle: () => void;
    onCycleRepeat: () => void;
    onToggleFavorite: (trackId: number, track?: Track) => void;
    onPlayTrack: (tr: Track, list?: Track[]) => void;
    onPlayYTMusic?: (yt: YTMusicTrack) => void;
    onPlayNext?: (track: Track) => void;
    onAddToQueue?: (track: Track) => void;
    onAddToPlaylist?: (playlistId: number, trackId: number, track?: Track) => void;
    onCreatePlaylist?: (track?: Track) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist?: (artistId: number) => void;
    onOpenArtistDetail?: (track: Track) => void;
    onOpenTrackDetail?: (track: Track) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onOpenLyrics: () => void;
    onResyncLyrics?: () => void;
    onOpenQueue: () => void;
    onOpenDevices?: () => void;
  }

  let {
    open,
    currentTrack,
    lyrics,
    lyricsLoading = false,
    lyricOffsetSec = 0,
    queue,
    queueIndex = -1,
    allTracks = [],
    ytRecommendedTracks = [],
    playlists = [],
    preferences = {},
    isPlaying,
    positionSec,
    durationSec,
    shuffle,
    repeatMode,
    mediaBaseUrl,
    lang = 'en',
    onClose,
    onTogglePlay,
    onPrev,
    onNext,
    onSeek,
    onToggleShuffle,
    onCycleRepeat,
    onToggleFavorite,
    onPlayTrack,
    onPlayYTMusic,
    onPlayNext,
    onAddToQueue,
    onAddToPlaylist,
    onCreatePlaylist,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenArtistDetail,
    onOpenTrackDetail,
    onOpenTrackAlbum,
    onOpenLyrics,
    onResyncLyrics,
    onOpenQueue,
    onOpenDevices
  }: Props = $props();

  let menuOpen = $state(false);

  const cover = $derived.by(() => {
    if (!currentTrack) return '';
    const raw =
      currentTrack.coverUrl ||
      (/^[A-Za-z0-9_-]{11}$/.test(currentTrack.externalId ?? '')
        ? `https://i.ytimg.com/vi/${currentTrack.externalId}/mqdefault.jpg`
        : '');
    return resolveCoverUrl(raw, mediaBaseUrl);
  });
  const effectiveDuration = $derived(durationSec > 0 ? durationSec : currentTrack?.durationSec || 0);
  const progressPct = $derived(
    effectiveDuration > 0 ? Math.min(100, Math.max(0, (positionSec / effectiveDuration) * 100)) : 0
  );

  function shelfEdgeFade(node: HTMLElement) {
    const update = () => {
      const hasOverflow = node.scrollWidth > node.clientWidth + 4;
      const atStart = node.scrollLeft <= 4;
      const atEnd = node.scrollLeft + node.clientWidth >= node.scrollWidth - 4;
      node.dataset.fadeStart = hasOverflow && !atStart ? 'true' : 'false';
      node.dataset.fadeEnd = hasOverflow && !atEnd ? 'true' : 'false';
    };
    update();
    node.addEventListener('scroll', update, { passive: true });
    const ro = new ResizeObserver(update);
    ro.observe(node);
    return {
      destroy() {
        node.removeEventListener('scroll', update);
        ro.disconnect();
      }
    };
  }

  type RecMusicCard =
    | { kind: 'track'; key: string; title: string; subtitle: string; cover: string; track: Track }
    | { kind: 'yt'; key: string; title: string; subtitle: string; cover: string; yt: YTMusicTrack };

  const recommendedMusic = $derived.by<RecMusicCard[]>(() => {
    const cards: RecMusicCard[] = [];
    const seen = new Set<string>();
    const curTitle = (currentTrack?.title ?? '').trim().toLowerCase();
    const curArtist = (currentTrack?.artistName ?? '').trim().toLowerCase();
    const curExt = (currentTrack?.externalId ?? '').trim();

    const addTrack = (t: Track) => {
      const key = `${t.title.trim().toLowerCase()}|${t.artistName.trim().toLowerCase()}`;
      if (!t.title.trim() || seen.has(key)) return;
      if (currentTrack && ((t.id > 0 && t.id === currentTrack.id) || (curExt && t.externalId === curExt) || (curTitle && key === `${curTitle}|${curArtist}`))) {
        return;
      }
      seen.add(key);
      const raw =
        t.coverUrl ||
        (/^[A-Za-z0-9_-]{11}$/.test(t.externalId ?? '')
          ? `https://i.ytimg.com/vi/${t.externalId}/mqdefault.jpg`
          : '');
      cards.push({
        kind: 'track',
        key: `rec-tr-${t.id}-${key}`,
        title: t.title,
        subtitle: t.artistName,
        cover: resolveCoverUrl(raw, mediaBaseUrl, 256),
        track: t
      });
    };

    if (curArtist) {
      for (const t of allTracks) {
        if (t.artistName.trim().toLowerCase() === curArtist) addTrack(t);
        if (cards.length >= 12) return cards;
      }
    }
    for (const yt of ytRecommendedTracks) {
      const key = `${yt.title.trim().toLowerCase()}|${yt.artist.trim().toLowerCase()}`;
      if (!yt.title.trim() || seen.has(key)) continue;
      if (curExt && yt.videoId === curExt) continue;
      seen.add(key);
      const raw =
        yt.coverUrl ||
        (/^[A-Za-z0-9_-]{11}$/.test(yt.videoId ?? '')
          ? `https://i.ytimg.com/vi/${yt.videoId}/mqdefault.jpg`
          : '');
      cards.push({
        kind: 'yt',
        key: `rec-yt-${yt.videoId || key}`,
        title: yt.title,
        subtitle: yt.artist,
        cover: resolveCoverUrl(raw, mediaBaseUrl, 256),
        yt
      });
      if (cards.length >= 12) return cards;
    }
    for (const t of [...allTracks, ...queue]) {
      addTrack(t);
      if (cards.length >= 12) return cards;
    }
    return cards;
  });
</script>

{#if open}
  <div
    in:fly={{ y: typeof window !== 'undefined' ? window.innerHeight : 760, duration: 280, opacity: 1, easing: cubicOut }}
    out:fly={{ y: typeof window !== 'undefined' ? window.innerHeight : 760, duration: 230, opacity: 1, easing: cubicIn }}
    class="fixed inset-0 z-50 overflow-y-auto px-5 pt-[max(1rem,env(safe-area-inset-top))] pb-10 select-none sm:hidden"
    style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-tertiary-container) 70%, var(--md-sys-color-background)) 0%, var(--md-sys-color-background) 42%); color: var(--md-sys-color-on-surface); padding-bottom: calc(2.5rem + env(safe-area-inset-bottom, 0px));"
  >
    <div class="max-w-md mx-auto space-y-5">
      <div class="flex items-center justify-between gap-3">
        <button
          type="button"
          onclick={onClose}
          aria-label="Tutup Now Playing"
          class="m3-icon-btn min-w-11 min-h-11 flex items-center justify-center cursor-pointer"
        >
          <Icon name="caret-down" weight="bold" size={22} />
        </button>

        <div class="min-w-0 flex-1 text-center">
          <div class="text-[10px] font-bold uppercase tracking-widest opacity-75">{tr(lang, 'nav.playingFrom')}</div>
          <div
            use:veloraMarquee={{ text: currentTrack?.albumTitle || 'Velora', disabled: !open }}
            class="overflow-hidden text-xs font-bold max-w-[220px] mx-auto"
          >
            <span class="inline-block whitespace-nowrap">
              {currentTrack?.albumTitle || 'Velora'}
            </span>
          </div>
        </div>

        {#if currentTrack}
          <TrackActionMenu
            track={currentTrack}
            open={menuOpen}
            {playlists}
            {lang}
            triggerClass="m3-icon-btn !w-11 !h-11"
            onToggleOpen={() => (menuOpen = !menuOpen)}
            onClose={() => (menuOpen = false)}
            {onPlayNext}
            {onAddToQueue}
            {onAddToPlaylist}
            {onCreatePlaylist}
            {onToggleFavorite}
            {onDownloadTrack}
            {onRemoveDownload}
            onOpenArtist={onOpenArtist ? (artistId) => {
              onClose();
              onOpenArtist(artistId);
            } : undefined}
            onOpenTrackAlbum={onOpenTrackAlbum ? (t) => {
              onClose();
              onOpenTrackAlbum(t);
            } : undefined}
          />
        {:else}
          <button
            type="button"
            onclick={() => {
              onClose();
              onOpenQueue();
            }}
            aria-label={tr(lang, 'nav.openQueue')}
            class="m3-icon-btn min-w-11 min-h-11 flex items-center justify-center cursor-pointer"
          >
            <Icon name="dots-three-vertical" weight="bold" size={20} />
          </button>
        {/if}
      </div>

      <div class="w-full max-w-[320px] mx-auto aspect-square rounded-2xl overflow-hidden shadow-2xl bg-[var(--md-sys-color-surface-container-high)] flex items-center justify-center">
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

      <div class="space-y-4">
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0 flex-1">
            <button
              type="button"
              disabled={!currentTrack || !onOpenTrackDetail}
              onclick={() => {
                if (currentTrack && onOpenTrackDetail) {
                  onClose();
                  onOpenTrackDetail(currentTrack);
                }
              }}
              use:veloraMarquee={{ text: currentTrack?.title ?? '', disabled: !open }}
              class="w-full overflow-hidden text-left text-xl font-bold tracking-tight cursor-pointer disabled:cursor-default"
            >
              <span class="inline-block whitespace-nowrap">
                {currentTrack?.title || tr(lang, 'nav.noSongPlaying')}
              </span>
            </button>
            <button
              type="button"
              disabled={!currentTrack || !onOpenArtistDetail}
              onclick={() => {
                if (currentTrack && onOpenArtistDetail) {
                  onClose();
                  onOpenArtistDetail(currentTrack);
                }
              }}
              use:veloraMarquee={{ text: currentTrack?.artistName ?? '', disabled: !open }}
              class="w-full overflow-hidden text-left text-sm mt-0.5 cursor-pointer disabled:cursor-default"
              style="color: var(--md-sys-color-on-surface-variant);"
            >
              <span class="inline-block whitespace-nowrap">
                {currentTrack?.artistName || 'Velora'}
              </span>
            </button>
          </div>

          <button
            type="button"
            disabled={!currentTrack}
            onclick={() => currentTrack && onToggleFavorite(currentTrack.id, currentTrack)}
            aria-label={tr(lang, 'nav.likeSong')}
            class="-mr-2.5 min-w-11 min-h-11 rounded-full flex items-center justify-center shrink-0 cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)] disabled:opacity-40 velora-heart-btn {currentTrack?.isFavorite ? 'is-loved' : ''}"
          >
            <Icon
              name="heart"
              weight={currentTrack?.isFavorite ? 'fill' : 'regular'}
              size={24}
              style={currentTrack?.isFavorite ? 'color: var(--md-sys-color-tertiary);' : 'color: var(--md-sys-color-on-surface);'}
            />
          </button>
        </div>

        <div class="space-y-2.5">
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

        <div class="flex items-center justify-between px-1">
          <button
            type="button"
            onclick={onToggleShuffle}
            aria-label="Shuffle"
            aria-pressed={shuffle}
            class="min-w-11 min-h-11 rounded-full relative flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            style={shuffle ? 'color: var(--md-sys-color-primary);' : 'color: var(--md-sys-color-on-surface-variant);'}
          >
            <Icon name="shuffle" weight={shuffle ? 'bold' : 'regular'} size={22} />
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
            aria-label="Sebelumnya"
            class="min-w-12 min-h-12 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
          >
            <Icon name="skip-back" weight="fill" size={28} />
          </button>

          <button
            type="button"
            onclick={onTogglePlay}
            aria-label={isPlaying ? 'Jeda' : 'Putar'}
            class="w-16 h-16 rounded-full flex items-center justify-center shadow-xl cursor-pointer transition-transform hover:scale-[1.04] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            <Icon name={isPlaying ? 'pause' : 'play'} weight="fill" size={28} />
          </button>

          <button
            type="button"
            onclick={onNext}
            aria-label="Berikutnya"
            class="min-w-12 min-h-12 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
          >
            <Icon name="skip-forward" weight="fill" size={28} />
          </button>

          <button
            type="button"
            onclick={onCycleRepeat}
            aria-label="Repeat"
            class="min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            style={repeatMode !== 'off' ? 'color: var(--md-sys-color-tertiary);' : 'color: var(--md-sys-color-on-surface-variant);'}
          >
            <Icon name={repeatMode === 'one' ? 'repeat-once' : 'repeat'} weight={repeatMode !== 'off' ? 'bold' : 'regular'} size={22} />
          </button>
        </div>

        <div class="flex items-center justify-between pt-1">
          <button
            type="button"
            onclick={() => {
              onClose();
              onOpenLyrics();
            }}
            aria-label="Buka Lirik"
            class="min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="microphone-stage" size={20} />
          </button>
          <div class="flex items-center gap-1">
            {#if onOpenDevices}
              <button
                type="button"
                onclick={() => {
                  onClose();
                  onOpenDevices();
                }}
                aria-label={tr(lang, 'nav.audioOutput')}
                class="velora-press min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                <Icon name="speaker-hifi" size={20} />
              </button>
            {/if}
            <button
              type="button"
              onclick={() => {
                onClose();
                onOpenQueue();
              }}
              aria-label="Buka Queue"
              class="velora-press min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
              style="color: var(--md-sys-color-on-surface-variant);"
            >
              <Icon name="queue" size={20} />
            </button>
          </div>
        </div>
      </div>

      <LyricsPreviewCard
        {currentTrack}
        {isPlaying}
        {positionSec}
        {lyricOffsetSec}
        {lyrics}
        {lyricsLoading}
        {preferences}
        collapsed={!open}
        {onSeek}
        onOpenLyrics={() => {
          onClose();
          onOpenLyrics();
        }}
        onResync={onResyncLyrics}
      />

      {#if recommendedMusic.length > 0}
        <section class="pt-1">
          <h2 class="text-base font-bold mb-3" style="color: var(--md-sys-color-on-surface);">
            {lang === 'id' ? 'Rekomendasi Musik' : 'Recommended Music'}
          </h2>
          <div
            use:shelfEdgeFade
            class="flex items-start gap-3.5 overflow-x-auto no-scrollbar velora-shelf-scroll pb-2"
          >
            {#each recommendedMusic as card (card.key)}
              <button
                type="button"
                onclick={() => {
                  if (card.kind === 'track') {
                    onPlayTrack(card.track, allTracks.length > 0 ? allTracks : queue);
                  } else if (onPlayYTMusic) {
                    onPlayYTMusic(card.yt);
                  }
                }}
                class="velora-press w-32 shrink-0 flex flex-col items-stretch justify-start self-start text-left group cursor-pointer"
              >
                <div
                  class="w-full aspect-square rounded-lg overflow-hidden mb-2 shadow-md"
                  style="background-color: var(--md-sys-color-surface-container-highest);"
                >
                  <CoverImage
                    src={card.cover}
                    fallbackIcon="music-note"
                    iconSize={30}
                    imgClass="block w-full h-full object-cover object-center transition-transform duration-300 group-hover:scale-[1.03]"
                    isVideoThumb={isYTVideoThumb(card.cover)}
                  />
                </div>
                <div class="text-[13px] font-bold leading-snug line-clamp-2 break-words whitespace-normal" style="color: var(--md-sys-color-on-surface);">
                  {card.title}
                </div>
                <div class="text-[11px] leading-snug truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                  {card.subtitle}
                </div>
              </button>
            {/each}
          </div>
        </section>
      {/if}
    </div>
  </div>
{/if}
