<script lang="ts">
  import { extractDominantColor, formatDuration, formatTotalMinutes, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Playlist, PlaylistDetail, Track, YTMusicTrack } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
  import CoverImage from '../components/CoverImage.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import EqIndicator from '../components/EqIndicator.svelte';
  import Icon from '../components/Icon.svelte';
  import ModalDialog from '../components/ModalDialog.svelte';
  import TrackActionMenu from '../components/TrackActionMenu.svelte';

  interface Props {
    mode: 'playlists' | 'local-playlists' | 'playlist-detail';
    playlists: Playlist[];
    playlistsLoading?: boolean;
    detail: PlaylistDetail | null;
    detailLoading?: boolean;
    allTracks: Track[];
    ytTop100Tracks?: YTMusicTrack[];
    ytTop100Loading?: boolean;
    onRefreshRecommendations?: () => void;
    currentTrackId: number | null;
    isPlaying?: boolean;
    shuffle?: boolean;
    queueSourceKey?: string | null;
    onTogglePlay?: () => void;
    onToggleShuffle?: () => void;
    mediaBaseUrl: string;
    accentColor: string;
    onOpenPlaylist: (id: number) => void;
    onPlayPlaylist?: (id: number) => void;
    onCreatePlaylist: () => void;
    onRenamePlaylist: (id: number, name: string, description: string, color: string) => void;
    onDeletePlaylist: (id: number) => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onPlayShuffled?: (list: Track[], sourceTitle?: string, sourceKey?: string) => void;
    onPlayYTMusic?: (yt: YTMusicTrack) => void;
    onAddTrack: (playlistId: number, trackId: number) => void;
    onAddYTMusicTrack?: (playlistId: number, yt: YTMusicTrack) => void;
    onRemoveTrack: (playlistId: number, trackId: number) => void;
    onReorderTracks: (playlistId: number, trackIds: number[]) => void;
    onToggleFavorite: (trackId: number) => void;
    onPlayNext?: (track: Track) => void;
    onAddToQueue?: (track: Track) => void;
    onCopyLink?: (track: Track) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist?: (artistId: number) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onExportM3U: (playlistId: number) => void;
    onImportM3U: () => void;
    onImportSpotify?: (url: string) => Promise<void>;
    lang?: LangCode;
  }

  let {
    mode,
    playlists,
    playlistsLoading = false,
    detail,
    detailLoading = false,
    allTracks,
    ytTop100Tracks = [],
    ytTop100Loading = false,
    onRefreshRecommendations,
    currentTrackId,
    isPlaying = false,
    shuffle = false,
    queueSourceKey = null,
    onTogglePlay,
    onToggleShuffle,
    mediaBaseUrl,
    accentColor,
    onOpenPlaylist,
    onPlayPlaylist,
    onCreatePlaylist,
    onRenamePlaylist,
    onDeletePlaylist,
    onPlayTrack,
    onPlayShuffled,
    onPlayYTMusic,
    onAddTrack,
    onAddYTMusicTrack,
    onRemoveTrack,
    onReorderTracks,
    onToggleFavorite,
    onPlayNext,
    onAddToQueue,
    onCopyLink,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenTrackAlbum,
    onExportM3U,
    onImportM3U,
    onImportSpotify,
    lang = 'en'
  }: Props = $props();

  let editing = $state(false);
  let editName = $state('');
  let editDesc = $state('');
  let dragIndex = $state<number | null>(null);
  let tabFilter = $state<'all' | 'local' | 'ytmusic'>('all');
  let recSeed = $state(Math.floor(Math.random() * 100000));
  let openActionTrackId = $state<number | null>(null);
  let failedRecCovers = $state<Record<string, boolean>>({});
  let importModalOpen = $state(false);
  let confirmDeleteOpen = $state(false);
  let spotifyUrl = $state('');
  let spotifyImporting = $state(false);
  let spotifyError = $state('');
  let dynamicCoverColor = $state<string | null>(null);

  function isColorTooDark(color: string | null | undefined): boolean {
    if (!color) return false;
    const c = color.trim();
    let r = -1;
    let g = -1;
    let b = -1;
    const rgbMatch = /^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i.exec(c);
    if (rgbMatch) {
      r = Number(rgbMatch[1]);
      g = Number(rgbMatch[2]);
      b = Number(rgbMatch[3]);
    } else {
      const hex = c.replace(/^#/, '');
      if (/^[0-9a-fA-F]{6}$/.test(hex)) {
        r = parseInt(hex.slice(0, 2), 16);
        g = parseInt(hex.slice(2, 4), 16);
        b = parseInt(hex.slice(4, 6), 16);
      }
    }
    if (r < 0 || g < 0 || b < 0) return false;
    return 0.2126 * r + 0.7152 * g + 0.0722 * b < 110;
  }

  $effect(() => {
    if (mode !== 'playlist-detail' || !detail) {
      dynamicCoverColor = null;
      return;
    }
    const firstCover = detail.tracks[0] ? resolveCoverUrl(detail.tracks[0].coverUrl, mediaBaseUrl) : '';
    const targetCover = firstCover || resolveCoverUrl(detail.playlist.coverUrl, mediaBaseUrl);
    if (!targetCover) {
      dynamicCoverColor = null;
      return;
    }
    let cancelled = false;
    void extractDominantColor(targetCover).then((color) => {
      if (!cancelled) {
        dynamicCoverColor = color;
      }
    });
    return () => {
      cancelled = true;
    };
  });

  async function handleSpotifyImportSubmit(e: Event) {
    e.preventDefault();
    if (!spotifyUrl.trim() || !onImportSpotify || spotifyImporting) return;
    spotifyError = '';
    spotifyImporting = true;
    try {
      await onImportSpotify(spotifyUrl.trim());
      importModalOpen = false;
      spotifyUrl = '';
    } catch (err: any) {
      spotifyError = err?.message || String(err) || 'Gagal mengimpor playlist Spotify.';
    } finally {
      spotifyImporting = false;
    }
  }

  const existingExternalIds = $derived(
    new Set(detail ? detail.tracks.map((t) => t.externalId).filter(Boolean) : [])
  );

  const recommendedTracks = $derived.by(() => {
    const pool = ytTop100Tracks.filter((yt) => !existingExternalIds.has(yt.videoId));
    if (pool.length <= 10) return pool;
    const copy = [...pool];
    let s = recSeed || 1;
    for (let i = copy.length - 1; i > 0; i--) {
      s = (s * 1664525 + 1013904223) >>> 0;
      const j = s % (i + 1);
      [copy[i], copy[j]] = [copy[j], copy[i]];
    }
    return copy.slice(0, 10);
  });

  function formatAddedDate(raw: string): string {
    if (!raw) return '—';
    const d = new Date(raw.includes('T') ? raw : raw.replace(' ', 'T') + 'Z');
    if (Number.isNaN(d.getTime())) return raw;
    return new Intl.DateTimeFormat(lang === 'id' ? 'id-ID' : lang, {
      day: 'numeric',
      month: 'short',
      year: 'numeric'
    }).format(d);
  }

  const visiblePlaylists = $derived(
    playlists.filter((p) => {
      if (mode === 'local-playlists' || tabFilter === 'local') {
        if (!p.isLocal) return false;
      } else if (tabFilter === 'ytmusic' && p.isLocal) {
        return false;
      }
      return true;
    })
  );

  function playShuffledPlaylist(list: Track[]) {
    if (list.length === 0) return;
    const isPlaylistQueueActive =
      Boolean(detail) &&
      ((queueSourceKey && queueSourceKey === `local-playlist:${detail!.playlist.id}`) ||
        (currentTrackId !== null && detail!.tracks.some((t) => t.id === currentTrackId)));
    if (shuffle && isPlaylistQueueActive && onToggleShuffle) {
      onToggleShuffle();
      return;
    }
    if (onPlayShuffled && detail) {
      onPlayShuffled(list, detail.playlist.name, `local-playlist:${detail.playlist.id}`);
      return;
    }
    const copy = [...list];
    for (let i = copy.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [copy[i], copy[j]] = [copy[j], copy[i]];
    }
    onPlayTrack(copy[0], copy);
  }

  function startEdit() {
    if (!detail) return;
    editName = detail.playlist.name;
    editDesc = detail.playlist.description;
    editing = true;
  }

  function saveEdit(e: Event) {
    e.preventDefault();
    if (!detail || !editName.trim()) return;
    onRenamePlaylist(detail.playlist.id, editName.trim(), editDesc.trim(), detail.playlist.coverColor);
    editing = false;
  }

  function moveTrack(index: number, delta: number) {
    if (!detail) return;
    const nextIdx = index + delta;
    if (nextIdx < 0 || nextIdx >= detail.tracks.length) return;
    const copy = [...detail.tracks];
    const [item] = copy.splice(index, 1);
    copy.splice(nextIdx, 0, item);
    onReorderTracks(
      detail.playlist.id,
      copy.map((t) => t.id)
    );
  }

  function handleDrop(targetIndex: number) {
    if (!detail || dragIndex === null || dragIndex === targetIndex) {
      dragIndex = null;
      return;
    }
    const copy = [...detail.tracks];
    const [item] = copy.splice(dragIndex, 1);
    copy.splice(targetIndex, 0, item);
    dragIndex = null;
    onReorderTracks(
      detail.playlist.id,
      copy.map((t) => t.id)
    );
  }
</script>

{#if mode === 'playlist-detail' && (detailLoading || !detail)}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col animate-pulse"
    style="background-color: var(--md-sys-color-surface-container-low);"
  >
    <section
      class="relative z-10 shrink-0 px-6 sm:px-10 pt-8 pb-8"
      style="background: linear-gradient(180deg, var(--md-sys-color-surface-container-high) 0%, var(--md-sys-color-surface-container-low) 100%);"
    >
      <div class="flex flex-col sm:flex-row sm:items-end gap-6">
        <div class="w-44 h-44 sm:w-52 sm:h-52 rounded-2xl bg-[var(--md-sys-color-surface-container-highest)] shrink-0"></div>
        <div class="min-w-0 flex-1 space-y-2.5">
          <div class="h-5 w-24 rounded-full bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="h-10 w-64 max-w-full rounded-lg bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="h-4 w-44 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
          <div class="pt-1 flex items-center gap-3">
            <div class="w-11 h-11 rounded-full bg-[var(--md-sys-color-surface-container-highest)]"></div>
            <div class="w-11 h-11 rounded-full bg-[var(--md-sys-color-surface-container-high)]"></div>
          </div>
        </div>
      </div>
    </section>

    <div class="relative z-10 px-6 sm:px-10 pt-4 pb-10 flex-1 flex flex-col gap-3">
      <div
        class="-mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] gap-4 px-3 py-2 text-[11px] font-bold uppercase tracking-wider"
        style="color: var(--md-sys-color-on-surface-variant);"
      >
        <div class="text-center">#</div>
        <div>{tr(lang, 'nav.titleCol')}</div>
        <div class="hidden md:block">{tr(lang, 'nav.playsCol')}</div>
        <div class="hidden md:block">{tr(lang, 'nav.addedCol')}</div>
        <div class="flex justify-end pr-1">
          <Icon name="clock" size={14} />
        </div>
        <div></div>
      </div>
      <div class="space-y-1.5">
        {#each [62, 48, 70, 54, 66, 44, 58] as titleW, idx (idx)}
          <div class="-mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] gap-4 px-3 py-2 items-center rounded-lg">
            <div class="flex justify-center">
              <div class="w-4 h-3 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
            </div>
            <div class="flex items-center gap-3 min-w-0">
              <div class="w-10 h-10 rounded-md bg-[var(--md-sys-color-surface-container-highest)] shrink-0"></div>
              <div class="space-y-1.5 min-w-0 flex-1">
                <div class="h-3.5 rounded bg-[var(--md-sys-color-surface-container-highest)]" style="width: {titleW}%;"></div>
                <div class="h-2.5 w-28 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
            </div>
            <div class="hidden md:block">
              <div class="h-3 w-14 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
            </div>
            <div class="hidden md:block">
              <div class="h-3 w-20 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
            </div>
            <div class="flex justify-end pr-1">
              <div class="h-3 w-8 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
            </div>
            <div></div>
          </div>
        {/each}
      </div>
    </div>
  </div>
{:else if mode === 'playlist-detail' && detail}
  {@const pl = detail.playlist}
  {@const firstTrackCover = detail.tracks[0] ? resolveCoverUrl(detail.tracks[0].coverUrl, mediaBaseUrl) : ''}
  {@const plCover = resolveCoverUrl(pl.coverUrl, mediaBaseUrl) || firstTrackCover}
  {@const rawTint = dynamicCoverColor || pl.coverColor}
  {@const heroTint = rawTint || 'var(--md-sys-color-primary)'}
  {@const heroAccent = isColorTooDark(rawTint) ? 'var(--md-sys-color-primary)' : heroTint}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    onclick={() => {
      openActionTrackId = null;
    }}
    role="presentation"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <section class="relative shrink-0 overflow-hidden px-6 sm:px-10 pt-8 pb-8">
      {#if plCover}
        <img
          src={plCover}
          alt=""
          aria-hidden="true"
          class="pointer-events-none absolute inset-0 w-full h-full object-cover scale-125 blur-3xl opacity-35"
        />
      {/if}
      <div
        class="pointer-events-none absolute inset-0"
        style="background: linear-gradient(180deg, color-mix(in srgb, {heroTint} 22%, rgba(11, 12, 14, 0.35)) 0%, var(--md-sys-color-surface-container-low) 100%);"
      ></div>

      <div class="relative z-10 flex flex-col sm:flex-row sm:items-end gap-6">
        <button
          type="button"
          onclick={startEdit}
          aria-label="Edit detail playlist"
          class="w-44 h-44 sm:w-52 sm:h-52 rounded-2xl overflow-hidden shrink-0 shadow-2xl flex items-center justify-center cursor-pointer transition-transform hover:scale-[1.02]"
          style={plCover
            ? 'background-color: var(--md-sys-color-surface-container-highest);'
            : `background: linear-gradient(135deg, color-mix(in srgb, ${heroTint} 55%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, ${heroTint} 85%, var(--md-sys-color-surface-container-highest)) 100%);`}
        >
          {#if plCover}
            <CoverImage
              src={plCover}
              fallbackIcon="playlist"
              iconSize={56}
              isVideoThumb={isYTVideoThumb(plCover)}
              loading="eager"
            />
          {:else}
            <Icon name="playlist" weight="fill" size={56} class="text-white drop-shadow-sm" />
          {/if}
        </button>

        <div class="min-w-0 flex-1 space-y-2">
          <div class="flex flex-wrap items-center gap-2 text-xs font-semibold">
            <span
              class="px-2.5 py-0.5 rounded-full"
              style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
            >
              {pl.isLocal ? tr(lang, 'nav.localPlaylist') : tr(lang, 'nav.savedPlaylist')}
            </span>
          </div>

          <button type="button" onclick={startEdit} class="block max-w-full text-left cursor-pointer group">
            <h1
              class="{pl.name.length > 44
                ? 'text-lg sm:text-xl lg:text-[24px]'
                : pl.name.length > 24
                  ? 'text-xl sm:text-2xl lg:text-[28px]'
                  : 'text-2xl sm:text-3xl lg:text-[38px]'} font-bold tracking-tight leading-[1.18] line-clamp-2 group-hover:underline decoration-2 underline-offset-4"
              style="color: var(--md-sys-color-on-surface);"
            >
              {pl.name}
            </h1>
          </button>

          <div
            class="flex flex-wrap items-center gap-x-2.5 gap-y-1 text-sm"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            {#if pl.description}
              <span class="font-medium" style="color: var(--md-sys-color-on-surface);">{pl.description}</span>
              <span>•</span>
            {/if}
            <span class="tabular-nums">{detail.tracks.length} {tr(lang, 'nav.songs').toLowerCase()}</span>
            <span>•</span>
            <span class="tabular-nums">{formatTotalMinutes(pl.totalSec)}</span>
          </div>

          <div class="pt-1 flex flex-wrap items-center gap-3">
            <button
              type="button"
              disabled={detail.tracks.length === 0}
              onclick={() => detail.tracks[0] && onPlayTrack(detail.tracks[0], detail.tracks)}
              title={tr(lang, 'nav.playAll')}
              aria-label={tr(lang, 'nav.playAll')}
              class="w-11 h-11 rounded-full border-0 outline-none shadow-none inline-flex items-center justify-center cursor-pointer transition-transform duration-150 hover:scale-[1.04] active:scale-95 disabled:opacity-40"
              style="background-color: {heroAccent}; color: var(--md-sys-color-on-primary);"
            >
              <Icon name="play" weight="fill" size={20} />
            </button>

            <button
              type="button"
              disabled={detail.tracks.length === 0}
              onclick={() => playShuffledPlaylist(detail.tracks)}
              title={tr(lang, 'nav.shuffle')}
              aria-label={tr(lang, 'nav.shuffle')}
              aria-pressed={shuffle}
              class="w-11 h-11 rounded-full relative inline-flex items-center justify-center cursor-pointer transition-colors hover:brightness-110 disabled:opacity-40"
              style={shuffle
                ? `background-color: color-mix(in srgb, ${heroAccent} 24%, var(--md-sys-color-surface-container-high)); color: ${heroAccent};`
                : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
            >
              <Icon name="shuffle" weight={shuffle ? 'bold' : 'regular'} size={18} />
              {#if shuffle}
                <span
                  class="absolute bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full"
                  style="background-color: {heroAccent};"
                ></span>
              {/if}
            </button>

            {#if pl.isLocal}
              <button
                type="button"
                onclick={() => onExportM3U(pl.id)}
                title="Export .M3U"
                aria-label="Export .M3U"
                class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] transition-colors"
                style="color: var(--md-sys-color-on-surface);"
              >
                <Icon name="download" size={18} />
              </button>
            {/if}

            <button
              type="button"
              onclick={() => (confirmDeleteOpen = true)}
              title={tr(lang, 'nav.deletePlaylist')}
              aria-label={tr(lang, 'nav.deletePlaylist')}
              class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-rose-500/20 hover:text-rose-400 transition-colors"
              style="color: var(--md-sys-color-on-surface);"
            >
              <Icon name="trash" size={18} />
            </button>
          </div>
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-10 pt-2 pb-10 flex-1 flex flex-col gap-6"
      style="background-color: var(--md-sys-color-surface-container-low);"
    >
      {#if detail.tracks.length > 0}
        <div class="space-y-1">
          <div
            class="-mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2 text-[11px] font-semibold uppercase tracking-wider"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <span class="text-center">#</span>
            <span>{tr(lang, 'nav.titleCol')}</span>
            <span class="hidden md:block">{tr(lang, 'nav.playsCol')}</span>
            <span class="hidden md:block">{tr(lang, 'nav.addedCol')}</span>
            <span class="flex justify-end pr-1">
              <Icon name="clock" size={14} />
            </span>
            <span></span>
          </div>

          <div class="space-y-0.5" role="list">
            {#each detail.tracks as item, idx (item.id)}
              {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
              {@const active = item.id === currentTrackId}
              {@const menuOpen = openActionTrackId === item.id}
              <div
                role="listitem"
                draggable={true}
                ondragstart={() => (dragIndex = idx)}
                ondragover={(e) => e.preventDefault()}
                ondrop={() => handleDrop(idx)}
                class="group relative -mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2 rounded-lg transition-colors duration-150 {active
                  ? 'bg-[var(--md-sys-color-state-active)]'
                  : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                style="color: var(--md-sys-color-on-surface);"
              >
                <div class="flex items-center justify-center">
                  {#if active && isPlaying}
                    <EqIndicator color={heroAccent} class="group-hover:hidden" />
                    <button
                      type="button"
                      onclick={() => (onTogglePlay ? onTogglePlay() : onPlayTrack(item, detail.tracks))}
                      aria-label="{tr(lang, 'nav.pause')} {item.title}"
                      class="hidden group-hover:flex items-center justify-center cursor-pointer"
                      style="color: {heroAccent};"
                    >
                      <Icon name="pause" weight="fill" size={14} />
                    </button>
                  {:else if active && !isPlaying}
                    <button
                      type="button"
                      onclick={() => (onTogglePlay ? onTogglePlay() : onPlayTrack(item, detail.tracks))}
                      aria-label="{tr(lang, 'nav.play')} {item.title}"
                      class="flex items-center justify-center cursor-pointer"
                      style="color: {heroAccent};"
                    >
                      <Icon name="play" weight="fill" size={14} />
                    </button>
                  {:else}
                    <span class="text-xs font-medium tabular-nums group-hover:hidden opacity-65">
                      {idx + 1}
                    </span>
                    <button
                      type="button"
                      onclick={() => onPlayTrack(item, detail.tracks)}
                      aria-label="{tr(lang, 'nav.play')} {item.title}"
                      class="hidden group-hover:flex items-center justify-center cursor-pointer"
                      style="color: {heroAccent};"
                    >
                      <Icon name="play" weight="fill" size={14} />
                    </button>
                  {/if}
                </div>

                <button
                  type="button"
                  onclick={() => (active && onTogglePlay ? onTogglePlay() : onPlayTrack(item, detail.tracks))}
                  class="flex items-center gap-3 min-w-0 text-left cursor-pointer"
                >
                  <div class="w-10 h-10 rounded-md overflow-hidden shrink-0">
                    <CoverImage src={cover} fallbackIcon="vinyl-record" iconSize={16} />
                  </div>
                  <div class="min-w-0 pr-4">
                    <div
                      class="text-xs font-bold line-clamp-2 whitespace-normal leading-snug"
                      style={active ? `color: color-mix(in srgb, ${heroAccent} 82%, #ffffff);` : ''}
                    >{item.title}</div>
                    <div class="text-[11px] truncate opacity-75 mt-0.5">
                      {item.artistName}
                    </div>
                  </div>
                </button>

                <div class="hidden md:block text-xs tabular-nums truncate opacity-75">
                  {item.globalPlays || '-'}
                </div>

                <div class="hidden md:block text-xs truncate tabular-nums opacity-75">
                  {formatAddedDate(item.createdAt)}
                </div>

                <span class="text-[11px] font-medium tabular-nums text-right opacity-75 pr-1">
                  {formatDuration(item.durationSec)}
                </span>

                <TrackActionMenu
                  track={item}
                  open={menuOpen}
                  {playlists}
                  excludePlaylistId={pl.id}
                  {lang}
                  triggerClass="w-8 h-8 {menuOpen ? 'opacity-100' : 'opacity-65 hover:opacity-100'}"
                  onToggleOpen={() => (openActionTrackId = menuOpen ? null : item.id)}
                  onClose={() => (openActionTrackId = null)}
                  {onPlayNext}
                  {onAddToQueue}
                  onAddToPlaylist={onAddTrack}
                  onRemoveFromPlaylist={(trackId) => onRemoveTrack(pl.id, trackId)}
                  {onToggleFavorite}
                  {onDownloadTrack}
                  {onRemoveDownload}
                  {onOpenArtist}
                  {onOpenTrackAlbum}
                  {onCopyLink}
                />
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <section class="pt-2 space-y-3">
        <div class="flex items-end justify-between gap-3">
          <h2 class="text-base font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            {tr(lang, 'nav.recommendedSongs')}
          </h2>
          <button
            type="button"
            onclick={() => {
              if (onRefreshRecommendations) {
                onRefreshRecommendations();
              } else {
                recSeed = Math.floor(Math.random() * 1000000) + 1;
              }
            }}
            class="text-xs font-bold cursor-pointer hover:underline"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            {tr(lang, 'nav.findMore')}
          </button>
        </div>

        {#if ytTop100Loading}
          <div class="space-y-1">
            {#each Array(10) as _, i (i)}
              <div class="-mx-2 grid grid-cols-[minmax(0,1fr)_auto] md:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)_auto] items-center gap-3 px-2 py-2 rounded-lg animate-pulse">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="w-10 h-10 rounded-md shrink-0 bg-[var(--md-sys-color-surface-container-high)]"></div>
                  <div class="space-y-1.5 flex-1 min-w-0">
                    <div class="h-3 w-2/3 max-w-[200px] rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
                    <div class="h-2.5 w-1/3 max-w-[120px] rounded bg-[var(--md-sys-color-surface-container-high)]/70"></div>
                  </div>
                </div>
                <div class="hidden md:block h-3 w-36 rounded bg-[var(--md-sys-color-surface-container-high)]/70"></div>
                <div class="h-8 w-24 rounded-full bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
            {/each}
          </div>
        {:else if recommendedTracks.length > 0}
          <div class="space-y-1">
            {#each recommendedTracks as rec (rec.videoId)}
              {@const recCover = resolveCoverUrl(rec.coverUrl, mediaBaseUrl)}
              <div class="-mx-2 grid grid-cols-[minmax(0,1fr)_auto] md:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)_auto] items-center gap-3 px-2 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] transition-colors">
                <button
                  type="button"
                  onclick={() => onPlayYTMusic?.(rec)}
                  class="flex items-center gap-3 min-w-0 text-left cursor-pointer"
                >
                  {#if recCover && !failedRecCovers[rec.videoId]}
                    <img
                      src={recCover}
                      alt={rec.title}
                      loading="lazy"
                      decoding="async"
                      onerror={() => {
                        failedRecCovers = { ...failedRecCovers, [rec.videoId]: true };
                      }}
                      class="w-10 h-10 rounded-md object-cover object-center shrink-0 bg-[var(--md-sys-color-surface-container-high)]"
                    />
                  {:else}
                    <div
                      class="w-10 h-10 rounded-md flex items-center justify-center shrink-0"
                      style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
                    >
                      <Icon name="music-notes" weight="fill" size={16} style="color: var(--md-sys-color-on-primary-container); opacity: 0.6;" />
                    </div>
                  {/if}
                  <div class="min-w-0 pr-4">
                    <div class="text-xs font-bold line-clamp-2 whitespace-normal leading-snug" style="color: var(--md-sys-color-on-surface);">{rec.title}</div>
                    <div class="text-[11px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      {rec.artist}
                    </div>
                  </div>
                </button>

                <div class="hidden md:block text-xs truncate" style="color: var(--md-sys-color-on-surface-variant);">
                  {rec.album && rec.album !== 'YouTube Music' && rec.album !== 'Single' ? rec.album : rec.title}
                </div>

                <button
                  type="button"
                  onclick={() => onAddYTMusicTrack?.(pl.id, rec)}
                  class="h-8 px-4 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-bold cursor-pointer transition-colors"
                  style="color: var(--md-sys-color-on-surface);"
                >
                  {tr(lang, 'nav.add')}
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    </div>
  </div>

  <ModalDialog
    open={editing}
    onClose={() => (editing = false)}
    ariaLabelledby="edit-playlist-title"
    maxWidthClass="max-w-lg"
    class="p-6"
  >
    <div class="mb-5">
      <h2 id="edit-playlist-title" class="text-xl font-bold tracking-tight">
        {tr(lang, 'nav.editDetails')}
      </h2>
    </div>

    <form onsubmit={saveEdit} class="space-y-4">
      <div class="grid grid-cols-1 sm:grid-cols-[144px_1fr] gap-4 items-stretch">
        <div
          class="w-36 h-36 rounded-lg overflow-hidden shrink-0 shadow-lg flex items-center justify-center mx-auto sm:mx-0"
          style="background: linear-gradient(135deg, color-mix(in srgb, {heroTint} 55%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, {heroTint} 85%, var(--md-sys-color-surface-container-highest)) 100%);"
        >
          {#if plCover}
            <img src={plCover} alt={pl.name} class="block w-full h-full object-cover object-center" />
          {:else}
            <Icon name="playlist" weight="fill" size={48} class="text-white drop-shadow-sm" />
          {/if}
        </div>

        <div class="flex flex-col gap-3 min-w-0">
          <input
            type="text"
            bind:value={editName}
            required
            placeholder={tr(lang, 'nav.playlistNamePlaceholder')}
            class="w-full h-10 px-3.5 rounded-md text-sm font-semibold outline-none bg-[var(--md-sys-color-surface-container)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
          />
          <textarea
            rows="4"
            bind:value={editDesc}
            placeholder={tr(lang, 'nav.playlistDescOptionalPlaceholder')}
            class="w-full flex-1 px-3.5 py-2.5 rounded-md text-xs outline-none resize-none bg-[var(--md-sys-color-surface-container)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
          ></textarea>
        </div>
      </div>

      <div class="flex items-center justify-end gap-2 pt-1">
        <button
          type="button"
          onclick={() => (editing = false)}
          class="h-10 px-5 rounded-full bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer"
        >
          {tr(lang, 'nav.cancel')}
        </button>
        <button
          type="submit"
          class="h-10 px-7 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
          style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
        >
          {tr(lang, 'nav.save')}
        </button>
      </div>
    </form>
  </ModalDialog>
{:else}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <section
      class="relative z-10 shrink-0 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary) 30%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 74%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)] flex items-center justify-center"
            style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-primary) 45%, #0077b6) 0%, color-mix(in srgb, var(--md-sys-color-primary) 82%, #38bdf8) 52%, color-mix(in srgb, var(--md-sys-color-on-primary-container) 88%, #ffffff) 100%);"
          >
            <Icon name="playlist" weight="fill" size={44} class="text-white drop-shadow-sm" />
          </div>

          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
              Collection • Playlists
            </div>
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              {tr(lang, 'nav.playlists')}
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {playlists.length} playlist • {tr(lang, 'nav.localPlaylists')} (.m3u) & {tr(lang, 'nav.savedPlaylists')}
            </p>
          </div>
        </div>

        <div class="mb-[4px] flex flex-wrap items-center gap-2">
          <button
            type="button"
            onclick={() => {
              spotifyError = '';
              spotifyUrl = '';
              importModalOpen = true;
            }}
            class="h-9 px-4 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
          >
            <Icon name="upload-simple" size={14} /> {tr(lang, 'nav.importPlaylist')}
          </button>
          <button
            type="button"
            onclick={onCreatePlaylist}
            class="h-9 px-4 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold flex items-center gap-1.5 cursor-pointer hover:brightness-105"
          >
            <Icon name="plus" weight="bold" size={14} /> {tr(lang, 'nav.createPlaylist')}
          </button>
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-10 flex-1 flex flex-col gap-5"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-primary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      <div class="flex flex-wrap items-center gap-2">
        {#each [
          { id: 'all', label: tr(lang, 'nav.all') },
          { id: 'ytmusic', label: tr(lang, 'nav.savedPlaylists') },
          { id: 'local', label: tr(lang, 'nav.localPlaylists') }
        ] as tab (tab.id)}
          <button
            type="button"
            onclick={() => (tabFilter = tab.id as 'all' | 'local' | 'ytmusic')}
            class="px-4 py-1.5 rounded-full text-xs font-semibold cursor-pointer transition-all hover:brightness-110"
            style={tabFilter === tab.id
              ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
              : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
          >
            {tab.label}
          </button>
        {/each}
      </div>

      {#if playlistsLoading && playlists.length === 0}
        <div class="velora-playlist-grid-7 animate-pulse">
          {#each Array.from({ length: 14 }) as _, idx (idx)}
            <div class="w-full min-w-0 flex flex-col gap-2.5 p-2.5">
              <div class="velora-card-artwork bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-3.5 w-3/4 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-2.5 w-1/2 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
            </div>
          {/each}
        </div>
      {:else if visiblePlaylists.length === 0}
        <EmptyState
          icon="playlist"
          iconWeight="fill"
          iconSize={38}
          title={tr(lang, 'nav.noPlaylists')}
          description={tr(lang, 'nav.noPlaylistsDesc')}
        >
          <button
            type="button"
            onclick={() => onCreatePlaylist()}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            {tr(lang, 'nav.createPlaylist')}
          </button>
        </EmptyState>
      {:else}
      <div class="velora-playlist-grid-7">
        {#each visiblePlaylists as pl (pl.id)}
          {@const cover = resolveCoverUrl(pl.coverUrl, mediaBaseUrl)}
          {@const cardTint = pl.coverColor || 'var(--md-sys-color-primary)'}
          <div class="group w-full min-w-0 flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]">
            <div
              class="velora-card-artwork mb-2.5 flex items-center justify-center"
              style={cover
                ? 'background-color: var(--md-sys-color-surface-container-high);'
                : `background: linear-gradient(135deg, color-mix(in srgb, ${cardTint} 55%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, ${cardTint} 85%, var(--md-sys-color-surface-container-highest)) 100%);`}
            >
              <button
                type="button"
                onclick={() => onOpenPlaylist(pl.id)}
                aria-label="Buka {pl.name}"
                class="velora-card-artwork-hit flex items-center justify-center cursor-pointer"
              >
                {#if cover}
                  <CoverImage
                    src={cover}
                    fallbackIcon="playlist"
                    iconSize={44}
                    isVideoThumb={isYTVideoThumb(cover)}
                  />
                {:else}
                  <Icon name="playlist" weight="fill" size={44} class="text-white/95 drop-shadow-[0_2px_8px_rgba(0,0,0,0.45)]" />
                {/if}
              </button>
              <button
                type="button"
                onclick={() => (onPlayPlaylist ? onPlayPlaylist(pl.id) : onOpenPlaylist(pl.id))}
                aria-label="Putar {pl.name}"
                class="absolute bottom-2 right-2 w-10 h-10 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] flex items-center justify-center cursor-pointer opacity-0 translate-y-2 group-hover:opacity-100 group-hover:translate-y-0 transition-all duration-200 hover:scale-105"
              >
                <Icon name="play" weight="fill" size={18} />
              </button>
            </div>
            <button type="button" onclick={() => onOpenPlaylist(pl.id)} class="w-full flex flex-col justify-start items-stretch text-left cursor-pointer">
              <div class="text-xs font-bold truncate" style="color: var(--md-sys-color-on-surface);">
                {pl.name}
              </div>
              <div class="text-[11px] leading-snug line-clamp-2 mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                {pl.description || `${pl.isLocal ? tr(lang, 'nav.localPlaylists') : tr(lang, 'nav.savedPlaylist')} • ${pl.trackCount} ${tr(lang, 'nav.songs').toLowerCase()}`}
              </div>
            </button>
          </div>
        {/each}
      </div>
    {/if}
    </div>
  </div>
{/if}

<ModalDialog
  open={importModalOpen}
  onClose={() => (importModalOpen = false)}
  ariaLabelledby="import-playlist-title"
  maxWidthClass="max-w-md"
  class="p-6"
>
  <div class="mb-5">
    <h2 id="import-playlist-title" class="text-lg font-bold tracking-tight">
      {tr(lang, 'nav.importPlaylist')}
    </h2>
  </div>

  <div class="space-y-5">
    <form onsubmit={handleSpotifyImportSubmit} class="space-y-2.5">
      <label for="spotify-url-input" class="block text-xs font-bold" style="color: var(--md-sys-color-on-surface);">
        {tr(lang, 'nav.spotifyLink')}
      </label>
      <input
        id="spotify-url-input"
        type="url"
        bind:value={spotifyUrl}
        required
        disabled={spotifyImporting}
        placeholder="https://open.spotify.com/playlist/..."
        class="m3-input w-full h-11 px-4 rounded-md text-xs disabled:opacity-60"
      />
      {#if spotifyError}
        <p class="text-[11px] text-rose-400">{spotifyError}</p>
      {/if}
      <button
        type="submit"
        disabled={spotifyImporting || !spotifyUrl.trim()}
        class="m3-btn-filled w-full h-10 px-5 text-xs font-bold flex items-center justify-center gap-2 cursor-pointer disabled:opacity-50"
      >
        {#if spotifyImporting}
          <EqIndicator variant="loader" color="currentColor" scale={0.75} />
          <span>{tr(lang, 'nav.importingSongs')}</span>
        {:else}
          <Icon name="link" size={15} />
          <span>{tr(lang, 'nav.importFromSpotify')}</span>
        {/if}
      </button>
    </form>

    <div class="flex items-center gap-3">
      <div class="flex-1 h-px opacity-15" style="background-color: var(--md-sys-color-on-surface);"></div>
      <span class="text-[11px] font-semibold uppercase tracking-wider opacity-60">{tr(lang, 'nav.or')}</span>
      <div class="flex-1 h-px opacity-15" style="background-color: var(--md-sys-color-on-surface);"></div>
    </div>

    <div class="space-y-2">
      <div class="text-xs font-bold" style="color: var(--md-sys-color-on-surface);">
        {tr(lang, 'nav.localFile')}
      </div>
      <button
        type="button"
        disabled={spotifyImporting}
        onclick={() => {
          importModalOpen = false;
          onImportM3U();
        }}
        class="m3-btn-tonal w-full h-10 px-5 text-xs font-semibold flex items-center justify-center gap-2 cursor-pointer disabled:opacity-50"
      >
        <Icon name="folder-open" size={16} />
        <span>{tr(lang, 'nav.chooseFile')}</span>
      </button>
    </div>
  </div>
</ModalDialog>

<ModalDialog
  open={confirmDeleteOpen}
  onClose={() => (confirmDeleteOpen = false)}
  ariaLabelledby="delete-playlist-title"
  maxWidthClass="max-w-sm"
  class="p-6"
>
  <div class="space-y-3">
    <h2 id="delete-playlist-title" class="text-base font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
      {tr(lang, 'nav.deletePlaylist')}
    </h2>
    <p class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
      {tr(lang, 'nav.deletePlaylistDesc')}
    </p>
    <div class="pt-2 flex items-center justify-end gap-2">
      <button
        type="button"
        onclick={() => (confirmDeleteOpen = false)}
        class="h-9 px-4 rounded-full bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer"
      >
        {tr(lang, 'nav.cancel')}
      </button>
      <button
        type="button"
        onclick={() => {
          const pid = detail?.playlist.id;
          confirmDeleteOpen = false;
          if (pid) onDeletePlaylist(pid);
        }}
        class="h-9 px-5 rounded-full bg-rose-500 hover:bg-rose-600 text-white text-xs font-bold cursor-pointer transition-colors"
      >
        {tr(lang, 'nav.delete')}
      </button>
    </div>
  </div>
</ModalDialog>
