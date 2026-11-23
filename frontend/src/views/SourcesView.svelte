<script lang="ts">
  import { formatDuration, resolveCoverUrl } from '../lib/api';
  import type { Source, YTMusicTrack } from '../lib/types';
  import Icon from '../components/Icon.svelte';

  interface Props {
    sources: Source[];
    ytResults: YTMusicTrack[];
    ytLoading: boolean;
    statusMessage: string;
    accentColor: string;
    onUploadFiles: () => void;
    onAddFolder: () => void;
    onRescanSource: (id: number) => void;
    onRemoveSource: (id: number) => void;
    onSearchYTMusic: (q: string) => void;
    onSaveYTMusic: (yt: YTMusicTrack) => void;
    onPlayYTMusic: (yt: YTMusicTrack) => void;
  }

  let {
    sources,
    ytResults,
    ytLoading,
    statusMessage,
    accentColor: _accentColor,
    onUploadFiles,
    onAddFolder,
    onRescanSource,
    onRemoveSource,
    onSearchYTMusic,
    onSaveYTMusic,
    onPlayYTMusic
  }: Props = $props();

  let ytQuery = $state('');
  let dragOver = $state(false);
</script>

<div class="px-5 py-4 space-y-6">
  <div class="flex flex-wrap items-end justify-between gap-4">
    <div class="flex items-center gap-3.5 min-w-0">
      <div
        class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0"
        style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
      >
        <Icon name="plugs-connected" weight="bold" size={24} />
      </div>
      <div class="min-w-0">
        <div class="text-[11px] font-semibold uppercase tracking-[0.12em] text-[var(--md-sys-color-on-surface-variant)] opacity-85">
          Stream & Library Hub
        </div>
        <h1 class="text-2xl font-bold tracking-tight text-[var(--md-sys-color-on-surface)]">Sources & Providers</h1>
        <p class="text-xs text-[var(--md-sys-color-on-surface-variant)] mt-0.5">
          Kelola Stream Providers, Metadata Providers, dan folder musik lokal
        </p>
      </div>
    </div>

    <div class="flex items-center gap-2 text-xs text-[var(--md-sys-color-on-surface-variant)] tabular-nums">
      <span class="px-3 py-1 rounded-full bg-[var(--md-sys-color-surface-container-high)]">
        {sources.length} active sources
      </span>
    </div>
  </div>

  {#if statusMessage}
    <div
      class="px-4 py-2.5 rounded-lg text-xs font-medium flex items-center gap-2.5 text-[var(--md-sys-color-on-primary-container)]"
      style="background-color: var(--md-sys-color-primary-container);"
    >
      <Icon name="check-circle" weight="fill" size={16} />
      <span>{statusMessage}</span>
    </div>
  {/if}

  <section class="space-y-3">
    <h2 class="text-xs font-semibold uppercase tracking-[0.08em] text-[var(--md-sys-color-on-surface-variant)]">
      Stream & Metadata Providers
    </h2>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <div class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container)] space-y-2">
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2.5 min-w-0">
            <div class="w-8 h-8 rounded-md flex items-center justify-center bg-rose-500/20 text-rose-400 shrink-0">
              <Icon name="youtube-logo" weight="fill" size={18} />
            </div>
            <span class="text-xs font-bold text-[var(--md-sys-color-on-surface)] truncate">YouTube Music (InnerTube)</span>
          </div>
          <span class="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-[var(--md-sys-color-primary-container)] text-[var(--md-sys-color-on-primary-container)] shrink-0">
            Active
          </span>
        </div>
        <p class="text-[11px] text-[var(--md-sys-color-on-surface-variant)] leading-relaxed">
          Streaming langsung katalog YouTube Music (OPUS / AAC) serta Home Feed & pencarian real-time.
        </p>
      </div>

      <div class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container)] space-y-2">
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2.5 min-w-0">
            <div class="w-8 h-8 rounded-md flex items-center justify-center bg-[var(--md-sys-color-primary-container)] text-[var(--md-sys-color-on-primary-container)] shrink-0">
              <Icon name="folder-open" weight="fill" size={18} />
            </div>
            <span class="text-xs font-bold text-[var(--md-sys-color-on-surface)] truncate">Local Library Scanner</span>
          </div>
          <span class="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-[var(--md-sys-color-primary-container)] text-[var(--md-sys-color-on-primary-container)] shrink-0">
            Active
          </span>
        </div>
        <p class="text-[11px] text-[var(--md-sys-color-on-surface-variant)] leading-relaxed">
          Mengindeks MP3, FLAC, M4A, AAC, OPUS, OGG, WAV beserta metadata tag dan cover art offline.
        </p>
      </div>

      <div class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container)] space-y-2">
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2.5 min-w-0">
            <div class="w-8 h-8 rounded-md flex items-center justify-center bg-indigo-400/20 text-indigo-300 shrink-0">
              <Icon name="text-align-left" weight="bold" size={18} />
            </div>
            <span class="text-xs font-bold text-[var(--md-sys-color-on-surface)] truncate">LRCLIB Synced Lyrics</span>
          </div>
          <span class="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-[var(--md-sys-color-primary-container)] text-[var(--md-sys-color-on-primary-container)] shrink-0">
            Active
          </span>
        </div>
        <p class="text-[11px] text-[var(--md-sys-color-on-surface-variant)] leading-relaxed">
          Sinkronisasi lirik LRC time-coded otomatis untuk panel Now Playing sebelah kanan.
        </p>
      </div>
    </div>
  </section>

  <section class="grid grid-cols-1 lg:grid-cols-2 gap-4">
    <div
      ondragover={(e) => {
        e.preventDefault();
        dragOver = true;
      }}
      ondragleave={() => (dragOver = false)}
      ondrop={(e) => {
        e.preventDefault();
        dragOver = false;
        onUploadFiles();
      }}
      role="region"
      aria-label="Zona Upload File Audio Lokal"
      class="rounded-lg p-6 flex flex-col items-center justify-center text-center transition-colors"
      style={dragOver ? 'background-color: var(--md-sys-color-primary-container);' : 'background-color: var(--md-sys-color-surface-container);'}
    >
      <div
        class="w-12 h-12 rounded-lg flex items-center justify-center mb-3 text-[var(--md-sys-color-on-primary-container)]"
        style="background-color: var(--md-sys-color-primary-container);"
      >
        <Icon name="cloud-arrow-up" weight="bold" size={24} />
      </div>
      <h2 class="text-sm font-bold text-[var(--md-sys-color-on-surface)]">Local Files & Mounted Folders</h2>
      <p class="text-xs max-w-sm mt-1 text-[var(--md-sys-color-on-surface-variant)]">
        Drag file ke sini atau pilih manual untuk menambah koleksi MP3, FLAC, M4A, AAC, OPUS, OGG, WAV.
      </p>
      <div class="mt-4 flex flex-wrap justify-center gap-2">
        <button
          type="button"
          onclick={onUploadFiles}
          class="h-9 px-4 rounded-full text-xs font-bold flex items-center gap-1.5 cursor-pointer transition-transform hover:scale-[1.03] active:scale-95"
          style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
        >
          <Icon name="upload-simple" weight="bold" size={14} /> Pilih File Audio
        </button>
        <button
          type="button"
          onclick={onAddFolder}
          class="h-9 px-4 rounded-full text-xs font-semibold text-[var(--md-sys-color-on-surface)] bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] flex items-center gap-1.5 cursor-pointer transition-colors"
        >
          <Icon name="folder-plus" size={14} /> Scan Folder Musik
        </button>
      </div>
    </div>

    <div class="rounded-lg p-5 space-y-3 bg-[var(--md-sys-color-surface-container-low)]">
      <div class="flex items-center justify-between">
        <h2 class="text-sm font-bold text-[var(--md-sys-color-on-surface)]">Connected Sources ({sources.length})</h2>
      </div>
      <div class="space-y-1.5 max-h-60 overflow-y-auto overscroll-contain pr-1">
        {#each sources as src (src.id)}
          <div class="flex items-center justify-between gap-3 px-3 py-2.5 rounded-md bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-high)] transition-colors">
            <div class="min-w-0">
              <div class="text-xs font-semibold truncate text-[var(--md-sys-color-on-surface)]">{src.name}</div>
              <div class="text-[11px] truncate mt-0.5 text-[var(--md-sys-color-on-surface-variant)]">
                {src.type === 'ytmusic' ? 'YouTube Music InnerTube' : src.path || 'Velora AppData Uploads'} • {src.trackCount} lagu
              </div>
            </div>
            <div class="flex items-center gap-1.5 shrink-0">
              {#if src.type === 'local_folder'}
                <button
                  type="button"
                  onclick={() => onRescanSource(src.id)}
                  title="Scan Ulang Folder"
                  class="w-8 h-8 rounded-full flex items-center justify-center bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-[var(--md-sys-color-on-surface)] opacity-85 cursor-pointer transition-colors"
                >
                  <Icon name="arrows-clockwise" size={14} />
                </button>
              {/if}
              {#if src.type !== 'ytmusic'}
                <button
                  type="button"
                  onclick={() => onRemoveSource(src.id)}
                  title="Hapus Sumber"
                  class="w-8 h-8 rounded-full flex items-center justify-center bg-[var(--md-sys-color-surface-container-high)] hover:bg-rose-500/20 text-rose-400 cursor-pointer transition-colors"
                >
                  <Icon name="trash" size={14} />
                </button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <section class="rounded-lg p-5 space-y-4 bg-[var(--md-sys-color-surface-container-low)]">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-md flex items-center justify-center shrink-0 bg-rose-500/20 text-rose-400">
          <Icon name="youtube-logo" weight="fill" size={20} />
        </div>
        <div>
          <h2 class="text-sm font-bold text-[var(--md-sys-color-on-surface)]">YouTube Music Explorer</h2>
          <p class="text-xs text-[var(--md-sys-color-on-surface-variant)]">Cari, simpan, atau putar langsung dari YouTube Music</p>
        </div>
      </div>

      <form
        onsubmit={(e) => {
          e.preventDefault();
          onSearchYTMusic(ytQuery);
        }}
        class="flex items-center gap-2 w-full sm:w-auto"
      >
        <div class="relative flex-1 sm:w-64">
          <span class="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none text-[var(--md-sys-color-on-surface-variant)] opacity-75">
            <Icon name="magnifying-glass" size={14} />
          </span>
          <input
            type="search"
            bind:value={ytQuery}
            placeholder="Cari lagu di YouTube Music"
            class="w-full h-9 pl-9 pr-4 rounded-full text-xs text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)] placeholder:opacity-65 bg-[var(--md-sys-color-surface-container-high)] focus:bg-[var(--md-sys-color-surface-container-highest)] outline-none transition-colors"
          />
        </div>
        <button
          type="submit"
          class="px-4 h-9 rounded-full text-xs font-bold cursor-pointer transition-transform hover:scale-[1.03] active:scale-95"
          style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
        >
          {ytLoading ? 'Mencari...' : 'Cari'}
        </button>
      </form>
    </div>

    {#if ytResults.length === 0}
      <div class="py-6 text-center text-xs text-[var(--md-sys-color-on-surface-variant)] opacity-75">
        Ketik judul lagu atau nama artis di atas untuk mencari langsung di YouTube Music.
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-1.5">
        {#each ytResults as yt (yt.videoId)}
          <div class="group flex items-center gap-3 px-3 py-2 rounded-lg bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-high)] transition-colors">
            <img src={resolveCoverUrl(yt.coverUrl)} alt={yt.title} class="block w-11 h-11 rounded-md object-cover object-center shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="text-xs font-semibold truncate text-[var(--md-sys-color-on-surface)] group-hover:text-[var(--md-sys-color-primary)] transition-colors">
                {yt.title}
              </div>
              <div class="text-[11px] truncate mt-0.5 text-[var(--md-sys-color-on-surface-variant)]">
                {yt.artist} • {formatDuration(yt.durationSec)}
              </div>
            </div>
            <button
              type="button"
              onclick={() => onSaveYTMusic(yt)}
              title="Tambahkan ke Library"
              class="w-8 h-8 rounded-full flex items-center justify-center bg-[var(--md-sys-color-surface-container-high)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-[var(--md-sys-color-on-surface)] cursor-pointer transition-colors"
            >
              <Icon name="plus" weight="bold" size={14} />
            </button>
            <button
              type="button"
              onclick={() => onPlayYTMusic(yt)}
              title="Putar"
              class="w-8 h-8 rounded-full flex items-center justify-center cursor-pointer transition-transform hover:scale-105 active:scale-95"
              style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
            >
              <Icon name="play" weight="fill" size={14} />
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </section>
</div>
