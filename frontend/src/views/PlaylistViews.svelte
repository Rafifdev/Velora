<script lang="ts">
  import { formatDuration, formatTotalMinutes, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Playlist, PlaylistDetail, Track, YTMusicTrack } from '../lib/types';
  import Icon from '../components/Icon.svelte';

  interface Props {
    mode: 'playlists' | 'local-playlists' | 'playlist-detail';
    playlists: Playlist[];
    detail: PlaylistDetail | null;
    allTracks: Track[];
    ytTop100Tracks?: YTMusicTrack[];
    ytTop100Loading?: boolean;
    onRefreshRecommendations?: () => void;
    currentTrackId: number | null;
    mediaBaseUrl: string;
    accentColor: string;
    onOpenPlaylist: (id: number) => void;
    onPlayPlaylist?: (id: number) => void;
    onCreatePlaylist: () => void;
    onRenamePlaylist: (id: number, name: string, description: string, color: string) => void;
    onDeletePlaylist: (id: number) => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onPlayYTMusic?: (yt: YTMusicTrack) => void;
    onAddTrack: (playlistId: number, trackId: number) => void;
    onAddYTMusicTrack?: (playlistId: number, yt: YTMusicTrack) => void;
    onRemoveTrack: (playlistId: number, trackId: number) => void;
    onReorderTracks: (playlistId: number, trackIds: number[]) => void;
    onToggleFavorite: (trackId: number) => void;
    onAddToQueue?: (track: Track) => void;
    onOpenArtist?: (artistId: number) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onExportM3U: (playlistId: number) => void;
    onImportM3U: () => void;
  }

  let {
    mode,
    playlists,
    detail,
    allTracks,
    ytTop100Tracks = [],
    ytTop100Loading = false,
    onRefreshRecommendations,
    currentTrackId,
    mediaBaseUrl,
    accentColor,
    onOpenPlaylist,
    onPlayPlaylist,
    onCreatePlaylist,
    onRenamePlaylist,
    onDeletePlaylist,
    onPlayTrack,
    onPlayYTMusic,
    onAddTrack,
    onAddYTMusicTrack,
    onRemoveTrack,
    onReorderTracks,
    onToggleFavorite,
    onAddToQueue,
    onOpenArtist,
    onOpenTrackAlbum,
    onExportM3U,
    onImportM3U
  }: Props = $props();

  let editing = $state(false);
  let editDialogEl: HTMLDialogElement | undefined = $state();
  let editName = $state('');
  let editDesc = $state('');
  let dragIndex = $state<number | null>(null);
  let tabFilter = $state<'all' | 'local' | 'ytmusic'>('all');
  let playlistSearch = $state('');
  let recSeed = $state(Math.floor(Math.random() * 100000));
  let openActionTrackId = $state<number | null>(null);
  let showPlaylistSubmenu = $state(false);
  let failedRecCovers = $state<Record<string, boolean>>({});

  $effect(() => {
    if (!editDialogEl) return;
    if (editing && !editDialogEl.open) {
      editDialogEl.showModal();
    } else if (!editing && editDialogEl.open) {
      editDialogEl.close();
    }
  });

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
    return new Intl.DateTimeFormat('id-ID', {
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
      const q = playlistSearch.trim().toLowerCase();
      if (!q) return true;
      return p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q);
    })
  );

  function playShuffledPlaylist(list: Track[]) {
    if (list.length === 0) return;
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

{#if mode === 'playlist-detail' && detail}
  {@const pl = detail.playlist}
  {@const firstTrackCover = detail.tracks[0] ? resolveCoverUrl(detail.tracks[0].coverUrl, mediaBaseUrl) : ''}
  {@const plCover = resolveCoverUrl(pl.coverUrl, mediaBaseUrl) || firstTrackCover}
  {@const heroTint = pl.coverColor || 'var(--md-sys-color-primary)'}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    onclick={() => {
      openActionTrackId = null;
      showPlaylistSubmenu = false;
    }}
    role="presentation"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <!-- Hero Banner -->
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, {heroTint} 30%, var(--md-sys-color-surface-container-high)) 0%, color-mix(in srgb, {heroTint} 16%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <button
            type="button"
            onclick={startEdit}
            aria-label="Edit detail playlist"
            class="w-28 h-28 rounded-lg overflow-hidden shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)] flex items-center justify-center cursor-pointer"
            style="background: linear-gradient(135deg, color-mix(in srgb, {heroTint} 55%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, {heroTint} 85%, var(--md-sys-color-surface-container-highest)) 100%);"
          >
            {#if plCover}
              <img src={plCover} alt={pl.name} class="block w-full h-full object-cover object-center" />
            {:else}
              <Icon name="playlist" weight="fill" size={44} class="text-white drop-shadow-sm" />
            {/if}
          </button>

          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
              Playlist
            </div>

            <button type="button" onclick={startEdit} class="block text-left cursor-pointer group">
              <h1 class="text-3xl font-extrabold tracking-tight group-hover:underline decoration-2 underline-offset-4" style="color: var(--md-sys-color-on-surface);">
                {pl.name}
              </h1>
            </button>

            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {#if pl.description}{pl.description} • {/if}{detail.tracks.length} lagu • {formatTotalMinutes(pl.totalSec)}
            </p>
          </div>
        </div>

        <!-- Action Buttons di Sisi Kanan Sejajar Teks -->
        <div class="mb-[4px] flex items-center gap-2.5">
          <button
            type="button"
            disabled={detail.tracks.length === 0}
            onclick={() => detail.tracks[0] && onPlayTrack(detail.tracks[0], detail.tracks)}
            title="Putar Playlist"
            aria-label="Putar Playlist"
            class="w-11 h-11 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-primary)] text-[var(--md-sys-color-on-primary)] inline-flex items-center justify-center cursor-pointer transition-transform duration-150 hover:scale-[1.04] active:scale-95 disabled:opacity-40"
          >
            <Icon name="play" weight="fill" size={18} />
          </button>

          <button
            type="button"
            disabled={detail.tracks.length === 0}
            onclick={() => playShuffledPlaylist(detail.tracks)}
            title="Acak (Shuffle)"
            aria-label="Acak (Shuffle)"
            class="w-11 h-11 rounded-full inline-flex items-center justify-center cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] transition-colors disabled:opacity-40"
            style="color: var(--md-sys-color-on-surface);"
          >
            <Icon name="shuffle" size={18} />
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
              <Icon name="download-simple" size={18} />
            </button>
          {/if}
        </div>
      </div>
    </section>

    <!-- Body with Smooth Fade -->
    <div
      class="relative z-10 px-6 sm:px-7 pt-5 pb-10 flex-1 flex flex-col gap-6"
      style="background: linear-gradient(180deg, color-mix(in srgb, {heroTint} 12%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, {heroTint} 6%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, {heroTint} 2%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      <!-- Playlist Track Table (tanpa empty state jika kosong) -->
      {#if detail.tracks.length > 0}
        <div class="space-y-1">
          <!-- Column Header: #, Judul, Album, Tanggal ditambahkan, Durasi, Aksi -->
          <div
            class="-mx-2 grid grid-cols-[28px_minmax(0,1fr)_52px_36px] md:grid-cols-[28px_minmax(0,1.5fr)_minmax(0,1fr)_136px_52px_36px] items-center gap-3 px-2 py-2 text-[11px] font-semibold uppercase tracking-wider"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <span class="text-center">#</span>
            <span>Judul</span>
            <span class="hidden md:block">Album</span>
            <span class="hidden md:block">Ditambahkan</span>
            <span class="flex justify-end pr-1">
              <Icon name="clock" size={14} />
            </span>
            <span></span>
          </div>

          <div class="space-y-0.5" role="list">
            {#each detail.tracks as tr, idx (tr.id)}
              {@const cover = resolveCoverUrl(tr.coverUrl, mediaBaseUrl)}
              {@const active = tr.id === currentTrackId}
              {@const menuOpen = openActionTrackId === tr.id}
              <div
                role="listitem"
                draggable={true}
                ondragstart={() => (dragIndex = idx)}
                ondragover={(e) => e.preventDefault()}
                ondrop={() => handleDrop(idx)}
                class="group relative -mx-2 grid grid-cols-[28px_minmax(0,1fr)_52px_36px] md:grid-cols-[28px_minmax(0,1.5fr)_minmax(0,1fr)_136px_52px_36px] items-center gap-3 px-2 py-2 rounded-lg transition-colors duration-150 {active
                  ? 'bg-[var(--md-sys-color-state-active)]'
                  : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                style="color: var(--md-sys-color-on-surface);"
              >
                <span class="text-center text-xs font-medium tabular-nums opacity-70 {active ? 'text-[var(--md-sys-color-primary)] opacity-100 font-bold' : ''}">
                  {idx + 1}
                </span>

                <button
                  type="button"
                  onclick={() => onPlayTrack(tr, detail.tracks)}
                  class="flex items-center gap-3 min-w-0 text-left cursor-pointer"
                >
                  {#if cover}
                    <img src={cover} alt={tr.title} class="block w-10 h-10 rounded-md object-cover object-center shrink-0" />
                  {:else}
                    <div
                      class="w-10 h-10 rounded-md flex items-center justify-center shrink-0"
                      style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
                    >
                      <Icon name="vinyl-record" weight="fill" size={16} style="color: var(--md-sys-color-on-primary-container); opacity: 0.6;" />
                    </div>
                  {/if}
                  <div class="min-w-0 pr-4">
                    <div class="text-xs font-bold line-clamp-2 whitespace-normal leading-snug {active ? 'text-[var(--md-sys-color-primary)]' : ''}">{tr.title}</div>
                    <div class="text-[11px] truncate opacity-75 mt-0.5">
                      {tr.artistName}
                    </div>
                  </div>
                </button>

                <div class="hidden md:block text-xs truncate opacity-75">
                  {tr.albumTitle || '—'}
                </div>

                <div class="hidden md:block text-xs truncate tabular-nums opacity-75">
                  {formatAddedDate(tr.createdAt)}
                </div>

                <span class="text-[11px] font-medium tabular-nums text-right opacity-75 pr-1">
                  {formatDuration(tr.durationSec)}
                </span>

                <!-- Action Column (...) -->
                <div class="relative flex items-center justify-end">
                  <button
                    type="button"
                    onclick={(e) => {
                      e.stopPropagation();
                      if (menuOpen) {
                        openActionTrackId = null;
                        showPlaylistSubmenu = false;
                      } else {
                        openActionTrackId = tr.id;
                        showPlaylistSubmenu = false;
                      }
                    }}
                    aria-label="Aksi lagu"
                    class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer transition-opacity {menuOpen ? 'opacity-100 bg-[var(--md-sys-color-surface-container-high)]' : 'opacity-65 group-hover:opacity-100'}"
                  >
                    <Icon name="dots-three-vertical" weight="bold" size={18} />
                  </button>

                  {#if menuOpen}
                    <div
                      onclick={(e) => e.stopPropagation()}
                      role="menu"
                      tabindex="-1"
                      class="absolute right-0 top-9 z-40 w-60 p-1.5 rounded-xl shadow-2xl text-xs"
                      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
                    >
                      {#if playlists.some((other) => other.id !== pl.id)}
                        <div class="relative">
                          <button
                            type="button"
                            onclick={() => (showPlaylistSubmenu = !showPlaylistSubmenu)}
                            class="w-full flex items-center justify-between gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                          >
                            <span class="flex items-center gap-2.5">
                              <Icon name="plus" size={15} />
                              <span>Tambah ke playlist</span>
                            </span>
                            <Icon name="caret-right" size={12} />
                          </button>
                          {#if showPlaylistSubmenu}
                            <div class="mt-0.5 mb-1 pl-3 space-y-0.5 max-h-36 overflow-y-auto">
                              {#each playlists.filter((other) => other.id !== pl.id) as otherPl (otherPl.id)}
                                <button
                                  type="button"
                                  onclick={() => {
                                    onAddTrack(otherPl.id, tr.id);
                                    openActionTrackId = null;
                                    showPlaylistSubmenu = false;
                                  }}
                                  class="w-full text-left px-3 py-1.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] truncate cursor-pointer opacity-90"
                                >
                                  {otherPl.name}
                                </button>
                              {/each}
                            </div>
                          {/if}
                        </div>
                      {/if}

                      <button
                        type="button"
                        onclick={() => {
                          onRemoveTrack(pl.id, tr.id);
                          openActionTrackId = null;
                        }}
                        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                      >
                        <Icon name="minus-circle" size={15} />
                        <span>Hapus dari playlist ini</span>
                      </button>

                      <button
                        type="button"
                        onclick={() => {
                          onToggleFavorite(tr.id);
                          openActionTrackId = null;
                        }}
                        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                      >
                        <Icon
                          name="heart"
                          weight={tr.isFavorite ? 'fill' : 'regular'}
                          size={15}
                          class={tr.isFavorite ? 'text-[var(--md-sys-color-tertiary)]' : ''}
                        />
                        <span>{tr.isFavorite ? 'Hapus dari Lagu yang Disukai' : 'Tambahkan ke Lagu yang Disukai'}</span>
                      </button>

                      {#if onAddToQueue}
                        <button
                          type="button"
                          onclick={() => {
                            onAddToQueue(tr);
                            openActionTrackId = null;
                          }}
                          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                        >
                          <Icon name="queue" size={15} />
                          <span>Tambahkan ke antrean</span>
                        </button>
                      {/if}

                      {#if (onOpenArtist && tr.artistId > 0) || onOpenTrackAlbum}
                        <div class="my-1 h-px opacity-15" style="background-color: var(--md-sys-color-on-surface);"></div>
                      {/if}

                      {#if onOpenArtist && tr.artistId > 0}
                        <button
                          type="button"
                          onclick={() => {
                            onOpenArtist(tr.artistId);
                            openActionTrackId = null;
                          }}
                          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                        >
                          <Icon name="user" size={15} />
                          <span>Buka artis</span>
                        </button>
                      {/if}

                      {#if onOpenTrackAlbum}
                        <button
                          type="button"
                          onclick={() => {
                            onOpenTrackAlbum(tr);
                            openActionTrackId = null;
                          }}
                          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
                        >
                          <Icon name="vinyl-record" size={15} />
                          <span>Ke album</span>
                        </button>
                      {/if}
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Recommended Section: 10 Lagu Acak dari Top 100 YouTube Music -->
      <section class="pt-2 space-y-3">
        <div class="flex items-end justify-between gap-3">
          <h2 class="text-base font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            Direkomendasikan
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
            Cari lainnya
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
                  {rec.album || 'Single'}
                </div>

                <button
                  type="button"
                  onclick={() => onAddYTMusicTrack?.(pl.id, rec)}
                  class="h-8 px-4 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-bold cursor-pointer transition-colors"
                  style="color: var(--md-sys-color-on-surface);"
                >
                  Tambahkan
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    </div>
  </div>

  <!-- Modal Edit Detail (mengikuti layout referensi) -->
  <dialog
    bind:this={editDialogEl}
    onclose={() => (editing = false)}
    closedby="any"
    aria-labelledby="edit-playlist-title"
    class="m-auto w-full max-w-lg rounded-xl shadow-2xl p-6"
    style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
  >
    <div class="flex items-center justify-between mb-5">
      <h2 id="edit-playlist-title" class="text-xl font-extrabold tracking-tight">
        Edit detail
      </h2>
      <button
        type="button"
        onclick={() => (editing = false)}
        aria-label="Tutup"
        class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer"
      >
        <Icon name="x" size={18} />
      </button>
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
            placeholder="Nama playlist"
            class="w-full h-10 px-3.5 rounded-md text-sm font-semibold outline-none bg-[var(--md-sys-color-surface-container)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
          />
          <textarea
            rows="4"
            bind:value={editDesc}
            placeholder="Tambahkan deskripsi opsional"
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
          Batal
        </button>
        <button
          type="submit"
          class="h-10 px-7 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
          style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
        >
          Simpan
        </button>
      </div>
    </form>
  </dialog>
{:else}
  <!-- Playlists Overview -->
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <!-- Hero Banner -->
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
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
            <h1 class="text-3xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Playlists
            </h1>
            <p class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
              {playlists.length} playlist • Local (.m3u) & YouTube Music
            </p>
          </div>
        </div>

        <div class="mb-[4px] flex flex-wrap items-center gap-2">
          <button
            type="button"
            onclick={onImportM3U}
            class="h-9 px-4 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
          >
            <Icon name="upload-simple" size={14} /> Import .M3U
          </button>
          <button
            type="button"
            onclick={onCreatePlaylist}
            class="h-9 px-4 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold flex items-center gap-1.5 cursor-pointer hover:brightness-105"
          >
            <Icon name="plus" weight="bold" size={14} /> Buat Playlist
          </button>
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-10 flex-1 flex flex-col gap-5"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-primary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          {#each [
            { id: 'all', label: 'Semua' },
            { id: 'local', label: 'Local Playlists' },
            { id: 'ytmusic', label: 'YouTube Music' }
          ] as tab (tab.id)}
            <button
              type="button"
              onclick={() => (tabFilter = tab.id as 'all' | 'local' | 'ytmusic')}
              class="px-4 py-1.5 rounded-full text-xs font-semibold cursor-pointer transition-colors"
              style={tabFilter === tab.id
                ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
                : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
            >
              {tab.label}
            </button>
          {/each}
        </div>

        <div class="relative w-60">
          <span
            class="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <Icon name="magnifying-glass" size={14} />
          </span>
          <input
            type="search"
            bind:value={playlistSearch}
            placeholder="Filter playlists..."
            class="w-full h-9 pl-9 pr-3.5 rounded-full text-xs outline-none bg-[var(--md-sys-color-surface-container-high)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
          />
        </div>
      </div>

      {#if visiblePlaylists.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
            <Icon name={playlistSearch.trim() ? 'magnifying-glass' : 'playlist'} weight="fill" size={38} />
          </div>
          <h2 class="text-2xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            {playlistSearch.trim() ? 'Playlist tidak ditemukan' : 'Belum ada playlist'}
          </h2>
          <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            {playlistSearch.trim()
              ? `Tidak ada playlist yang cocok dengan "${playlistSearch.trim()}".`
              : 'Buat playlist baru untuk mengumpulkan lagu favoritmu.'}
          </p>
          <button
            type="button"
            onclick={() => (playlistSearch.trim() ? (playlistSearch = '') : onCreatePlaylist())}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            {playlistSearch.trim() ? 'Reset filter' : 'Buat playlist'}
          </button>
        </div>
      {:else}
      <div class="velora-card-grid">
        {#each visiblePlaylists as pl (pl.id)}
          {@const cover = resolveCoverUrl(pl.coverUrl, mediaBaseUrl)}
          <div class="group w-[172px] flex flex-col justify-start items-stretch self-start p-2.5 rounded-lg transition-colors duration-200 hover:bg-[var(--md-sys-color-state-hover)]">
            <div
              class="velora-card-artwork mb-2.5 flex items-center justify-center"
              style="background: linear-gradient(135deg, {pl.coverColor || 'var(--md-sys-color-primary-container)'}, var(--md-sys-color-surface-container-highest));"
            >
              <button
                type="button"
                onclick={() => onOpenPlaylist(pl.id)}
                aria-label="Buka {pl.name}"
                class="velora-card-artwork-hit cursor-pointer"
              >
                {#if cover}
                  <img
                    src={cover}
                    alt={pl.name}
                    loading="lazy"
                    decoding="async"
                    class:velora-yt-video-thumb={isYTVideoThumb(cover)}
                  />
                {:else}
                  <Icon name="vinyl-record" weight="fill" size={44} style="color: var(--md-sys-color-on-primary-container); opacity: 0.55;" />
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
                {pl.description || `${pl.isLocal ? 'Local Playlist' : 'YT Music'} • ${pl.trackCount} lagu`}
              </div>
            </button>
          </div>
        {/each}
      </div>
    {/if}
    </div>
  </div>
{/if}
