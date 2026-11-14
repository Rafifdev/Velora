<script lang="ts">
  import { formatDuration, formatTotalMinutes, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import type { Playlist, PlaylistDetail, Track } from '../lib/types';
  import Icon from '../components/Icon.svelte';

  interface Props {
    mode: 'playlists' | 'local-playlists' | 'playlist-detail';
    playlists: Playlist[];
    detail: PlaylistDetail | null;
    allTracks: Track[];
    currentTrackId: number | null;
    mediaBaseUrl: string;
    accentColor: string;
    onOpenPlaylist: (id: number) => void;
    onPlayPlaylist?: (id: number) => void;
    onCreatePlaylist: (isLocal: boolean) => void;
    onRenamePlaylist: (id: number, name: string, description: string, color: string) => void;
    onDeletePlaylist: (id: number) => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onAddTrack: (playlistId: number, trackId: number) => void;
    onRemoveTrack: (playlistId: number, trackId: number) => void;
    onReorderTracks: (playlistId: number, trackIds: number[]) => void;
    onToggleFavorite: (trackId: number) => void;
    onExportM3U: (playlistId: number) => void;
    onImportM3U: () => void;
  }

  let {
    mode,
    playlists,
    detail,
    allTracks,
    currentTrackId,
    mediaBaseUrl,
    accentColor,
    onOpenPlaylist,
    onPlayPlaylist,
    onCreatePlaylist,
    onRenamePlaylist,
    onDeletePlaylist,
    onPlayTrack,
    onAddTrack,
    onRemoveTrack,
    onReorderTracks,
    onToggleFavorite,
    onExportM3U,
    onImportM3U
  }: Props = $props();

  let editing = $state(false);
  let editName = $state('');
  let editDesc = $state('');
  let showAddPicker = $state(false);
  let dragIndex = $state<number | null>(null);
  let tabFilter = $state<'all' | 'local' | 'ytmusic'>('all');
  let playlistSearch = $state('');

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
  {@const plCover = resolveCoverUrl(pl.coverUrl, mediaBaseUrl)}
  <div class="relative min-h-full" use:svelteMotionEnter={{ y: 8 }}>
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-60"
      style="background: linear-gradient(180deg, rgba(29, 185, 84, 0.14) 0%, transparent 100%);"
    ></div>

    <div class="relative z-10 p-6 space-y-6">
      <div class="flex flex-wrap items-end gap-5">
        <div
          class="w-32 h-32 rounded-lg overflow-hidden flex items-center justify-center shrink-0"
          style="background: linear-gradient(135deg, {pl.coverColor || 'var(--md-sys-color-primary-container)'}, var(--md-sys-color-surface-container-highest));"
        >
          {#if plCover}
            <img src={plCover} alt={pl.name} class="block w-full h-full object-cover object-center" />
          {:else}
            <Icon name="vinyl-record" weight="fill" size={48} style="color: var(--md-sys-color-on-primary-container); opacity: 0.6;" />
          {/if}
        </div>

        <div class="space-y-1.5 flex-1 min-w-[220px]">
          <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">
            {pl.isLocal ? 'Local Playlist' : 'YouTube Music Playlist'}
          </div>

          {#if editing}
            <form onsubmit={saveEdit} class="space-y-2.5 max-w-md pt-1">
              <input
                type="text"
                bind:value={editName}
                class="w-full px-3.5 py-2 rounded-md text-base font-semibold outline-none bg-[var(--md-sys-color-surface-container-high)] text-[var(--md-sys-color-on-surface)]"
              />
              <input
                type="text"
                bind:value={editDesc}
                placeholder="Deskripsi..."
                class="w-full px-3.5 py-1.5 rounded-md text-xs outline-none bg-[var(--md-sys-color-surface-container-high)] text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)]"
              />
              <div class="flex gap-2">
                <button
                  type="submit"
                  class="px-4 py-1.5 rounded-full bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold cursor-pointer"
                >
                  Simpan
                </button>
                <button
                  type="button"
                  onclick={() => (editing = false)}
                  class="px-4 py-1.5 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer"
                >
                  Batal
                </button>
              </div>
            </form>
          {:else}
            <h1 class="text-3xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              {pl.name}
            </h1>
            <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
              {pl.description || 'Koleksi playlist Velora'}
            </p>
          {/if}

          <div class="text-xs tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
            {detail.tracks.length} lagu • {formatTotalMinutes(pl.totalSec)}
          </div>
        </div>
      </div>

      <!-- Action Toolbar -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            disabled={detail.tracks.length === 0}
            onclick={() => detail.tracks[0] && onPlayTrack(detail.tracks[0], detail.tracks)}
            class="h-10 px-5 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold flex items-center gap-1.5 cursor-pointer hover:brightness-105 disabled:opacity-40"
          >
            <Icon name="play" weight="fill" size={14} /> Play Playlist
          </button>

          <button
            type="button"
            disabled={detail.tracks.length === 0}
            onclick={() => playShuffledPlaylist(detail.tracks)}
            class="h-10 px-4 rounded-full text-xs font-semibold flex items-center gap-1.5 cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] disabled:opacity-40"
          >
            <Icon name="shuffle" size={14} /> Shuffle
          </button>

          <button
            type="button"
            onclick={() => (showAddPicker = !showAddPicker)}
            class="h-10 px-4 rounded-full text-xs font-semibold flex items-center gap-1.5 cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)]"
          >
            <Icon name="plus" weight="bold" size={14} /> Tambah Lagu
          </button>

          <button
            type="button"
            onclick={startEdit}
            class="h-10 px-4 rounded-full text-xs font-semibold flex items-center gap-1.5 cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)]"
          >
            <Icon name="pencil-simple" size={14} /> Edit
          </button>

          {#if pl.isLocal}
            <button
              type="button"
              onclick={() => onExportM3U(pl.id)}
              class="h-10 px-4 rounded-full text-xs font-semibold flex items-center gap-1.5 cursor-pointer bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)]"
            >
              <Icon name="download-simple" size={14} /> Export .M3U
            </button>
          {/if}
        </div>

        <button
          type="button"
          onclick={() => onDeletePlaylist(pl.id)}
          class="h-10 px-4 rounded-full text-xs font-medium flex items-center gap-1.5 text-rose-400 cursor-pointer transition-colors hover:bg-rose-500/20"
          style="background-color: rgba(244, 63, 94, 0.12);"
        >
          <Icon name="trash" size={14} /> Hapus Playlist
        </button>
      </div>

      {#if showAddPicker}
        <div class="p-5 space-y-3 rounded-lg" style="background-color: var(--md-sys-color-surface-container-low);">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold" style="color: var(--md-sys-color-on-surface);">Pilih Lagu dari Library</span>
            <button
              type="button"
              onclick={() => (showAddPicker = false)}
              class="px-3 py-1 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs cursor-pointer"
            >
              Tutup
            </button>
          </div>
          <div class="max-h-52 overflow-y-auto overscroll-contain space-y-1 pr-1">
            {#each allTracks as t (t.id)}
              <div class="flex items-center justify-between px-3 py-2 rounded-md hover:bg-[var(--md-sys-color-state-hover)] transition-colors">
                <div class="min-w-0 pr-3">
                  <div class="text-xs font-semibold truncate">{t.title}</div>
                  <div class="text-[11px] truncate" style="color: var(--md-sys-color-on-surface-variant);">
                    {t.artistName} • {t.format}
                  </div>
                </div>
                <button
                  type="button"
                  onclick={() => onAddTrack(pl.id, t.id)}
                  class="px-3 py-1 rounded-full bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold cursor-pointer shrink-0"
                >
                  + Add
                </button>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      {#if detail.tracks.length === 0}
        <div class="p-8 text-center text-xs rounded-lg" style="background-color: var(--md-sys-color-surface-container-low); color: var(--md-sys-color-on-surface-variant);">
          Playlist ini masih kosong. Klik "Tambah Lagu" di atas untuk mengisi daftar putar.
        </div>
      {:else}
        <div class="space-y-0.5" role="list">
          {#each detail.tracks as tr, idx (tr.id)}
            {@const cover = resolveCoverUrl(tr.coverUrl, mediaBaseUrl)}
            {@const active = tr.id === currentTrackId}
            <div
              role="listitem"
              draggable={true}
              ondragstart={() => (dragIndex = idx)}
              ondragover={(e) => e.preventDefault()}
              ondrop={() => handleDrop(idx)}
              class="group flex items-center justify-between gap-3 px-3.5 py-2 rounded-lg transition-colors duration-150 {active
                ? 'bg-[var(--md-sys-color-state-active)]'
                : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
              style="color: var(--md-sys-color-on-surface);"
            >
              <button
                type="button"
                onclick={() => onPlayTrack(tr, detail.tracks)}
                class="flex items-center gap-3 min-w-0 flex-1 text-left cursor-pointer"
              >
                <span class="w-5 text-center text-xs font-medium tabular-nums opacity-70 {active ? 'text-[var(--md-sys-color-primary)]' : ''}">
                  {idx + 1}
                </span>
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
                <div class="min-w-0">
                  <div class="text-xs font-bold truncate {active ? 'text-[var(--md-sys-color-primary)]' : ''}">{tr.title}</div>
                  <div class="text-[11px] truncate opacity-75">
                    {tr.artistName} • {tr.albumTitle}
                  </div>
                </div>
              </button>

              <div class="flex items-center gap-1">
                <button
                  type="button"
                  onclick={() => moveTrack(idx, -1)}
                  disabled={idx === 0}
                  title="Geser naik"
                  class="m3-icon-btn w-7 h-7 flex items-center justify-center disabled:opacity-25 cursor-pointer"
                >
                  <Icon name="arrow-up" weight="bold" size={13} />
                </button>
                <button
                  type="button"
                  onclick={() => moveTrack(idx, 1)}
                  disabled={idx === detail.tracks.length - 1}
                  title="Geser turun"
                  class="m3-icon-btn w-7 h-7 flex items-center justify-center disabled:opacity-25 cursor-pointer"
                >
                  <Icon name="arrow-down" weight="bold" size={13} />
                </button>
                <button
                  type="button"
                  onclick={() => onToggleFavorite(tr.id)}
                  aria-label="Sukai lagu"
                  class="m3-icon-btn w-7 h-7 flex items-center justify-center cursor-pointer {tr.isFavorite ? 'text-[var(--md-sys-color-tertiary)]' : ''}"
                >
                  <Icon
                    name="heart"
                    weight={tr.isFavorite ? 'fill' : 'regular'}
                    size={15}
                  />
                </button>
                <span class="text-[11px] font-medium tabular-nums w-10 text-right opacity-75">
                  {formatDuration(tr.durationSec)}
                </span>
                <button
                  type="button"
                  onclick={() => onRemoveTrack(pl.id, tr.id)}
                  title="Hapus dari playlist"
                  class="m3-icon-btn w-7 h-7 flex items-center justify-center text-rose-400 cursor-pointer"
                >
                  <Icon name="trash" size={14} />
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{:else}
  <!-- Nuclear Playlists Overview (Borderless) -->
  <div class="p-6 space-y-6" use:svelteMotionEnter={{ y: 8 }}>
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-primary)]">Collection • Playlists</div>
        <h1 class="text-2xl font-bold tracking-tight mt-0.5" style="color: var(--md-sys-color-on-surface);">
          Playlists
        </h1>
        <p class="text-xs mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
          Gabungan Local Playlists (.m3u) & YouTube Music Playlists
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onclick={onImportM3U}
          class="h-9 px-4 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
        >
          <Icon name="upload-simple" size={14} /> Import .M3U
        </button>
        <button
          type="button"
          onclick={() => onCreatePlaylist(true)}
          class="h-9 px-4 rounded-full bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
        >
          <Icon name="plus" weight="bold" size={14} /> Local Playlist
        </button>
        <button
          type="button"
          onclick={() => onCreatePlaylist(false)}
          class="h-9 px-4 rounded-full border-0 outline-none shadow-none bg-[var(--md-sys-color-tertiary)] text-[var(--md-sys-color-on-tertiary)] text-xs font-bold flex items-center gap-1.5 cursor-pointer hover:brightness-105"
        >
          <Icon name="plus" weight="bold" size={14} /> YT Playlist
        </button>
      </div>
    </div>

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
              : 'background-color: rgba(128, 128, 128, 0.16); color: var(--md-sys-color-on-surface);'}
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
      <div class="p-10 text-center space-y-2 rounded-lg" style="background-color: var(--md-sys-color-surface-container-low);">
        <Icon name="playlist" weight="fill" size={32} class="mx-auto opacity-50" />
        <p class="text-sm font-semibold">Belum ada playlist di kategori ini</p>
        <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
          Buat playlist baru atau impor file .m3u untuk mengoleksi daftar putarmu.
        </p>
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
{/if}
