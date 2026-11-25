<script lang="ts">
  import { formatDuration, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { DownloadProgress, Track } from '../lib/types';
  import CoverImage from '../components/CoverImage.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import Icon from '../components/Icon.svelte';

  interface Props {
    activeDownloads: DownloadProgress[];
    downloadedTracks: Track[];
    deletedTracks?: Track[];
    allTracks?: Track[];
    loading?: boolean;
    mediaBaseUrl: string;
    lang?: LangCode;
    isDummyActive?: boolean;
    onBack: () => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onCancelDownload?: (trackId: number) => void;
    onRemoveDownload?: (trackId: number) => void;
    onCopyLink?: (track: Track) => void;
    onToggleDummy?: () => void;
  }

  let {
    activeDownloads,
    downloadedTracks,
    deletedTracks = [],
    allTracks = [],
    loading = false,
    mediaBaseUrl,
    lang = 'en',
    isDummyActive = false,
    onBack,
    onPlayTrack,
    onOpenTrackAlbum,
    onCancelDownload,
    onRemoveDownload,
    onCopyLink,
    onToggleDummy
  }: Props = $props();

  let searchOpen = $state(false);
  let searchText = $state('');
  let previewSkeleton = $state(false);

  const totalCount = $derived(activeDownloads.length + downloadedTracks.length);

  function resolveDownloadCover(dl: DownloadProgress): string {
    const matched = allTracks.find(
      (t) =>
        (dl.trackId > 0 && t.id === dl.trackId) ||
        (dl.externalId && t.externalId === dl.externalId)
    );
    return matched ? resolveCoverUrl(matched.coverUrl, mediaBaseUrl, 96) : '';
  }

  function formatGroupDate(raw?: string): string {
    if (!raw) {
      return lang === 'id' ? 'Hari ini' : 'Today';
    }
    const iso = raw.includes('T') ? raw : raw.replace(' ', 'T') + 'Z';
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return lang === 'id' ? 'Hari ini' : 'Today';

    const now = new Date();
    const startToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const startTarget = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
    const diffDays = Math.round((startToday - startTarget) / 86400000);

    if (diffDays === 0) return lang === 'id' ? 'Hari ini' : 'Today';
    if (diffDays === 1) return lang === 'id' ? 'Kemarin' : 'Yesterday';

    return new Intl.DateTimeFormat(lang === 'id' ? 'id-ID' : 'en-US', {
      month: 'long',
      day: 'numeric',
      year: 'numeric'
    }).format(d);
  }

  type DownloadEntry = {
    track: Track;
    deleted: boolean;
    dateLabel: string;
    sortMs: number;
  };

  const filteredActive = $derived.by(() => {
    const q = searchText.trim().toLowerCase();
    if (!q) return activeDownloads;
    return activeDownloads.filter(
      (d) => d.title.toLowerCase().includes(q) || d.artist.toLowerCase().includes(q)
    );
  });

  const groupedHistory = $derived.by(() => {
    const q = searchText.trim().toLowerCase();
    const entries: DownloadEntry[] = [];

    for (const trItem of downloadedTracks) {
      if (
        q &&
        !trItem.title.toLowerCase().includes(q) &&
        !trItem.artistName.toLowerCase().includes(q)
      ) {
        continue;
      }
      const raw = trItem.downloadedAt || trItem.createdAt;
      const ms = raw ? Date.parse(raw.includes('T') ? raw : raw.replace(' ', 'T') + 'Z') : Date.now();
      entries.push({
        track: trItem,
        deleted: false,
        dateLabel: formatGroupDate(raw),
        sortMs: Number.isFinite(ms) ? ms : Date.now()
      });
    }

    for (const delItem of deletedTracks) {
      if (
        q &&
        !delItem.title.toLowerCase().includes(q) &&
        !delItem.artistName.toLowerCase().includes(q)
      ) {
        continue;
      }
      const raw = delItem.downloadedAt || delItem.createdAt;
      entries.push({
        track: delItem,
        deleted: true,
        dateLabel: formatGroupDate(raw),
        sortMs: Date.now()
      });
    }

    entries.sort((a, b) => b.sortMs - a.sortMs);

    const groups: { label: string; items: DownloadEntry[] }[] = [];
    const map = new Map<string, DownloadEntry[]>();
    for (const entry of entries) {
      let arr = map.get(entry.dateLabel);
      if (!arr) {
        arr = [];
        map.set(entry.dateLabel, arr);
        groups.push({ label: entry.dateLabel, items: arr });
      }
      arr.push(entry);
    }
    return groups;
  });
</script>

<div
  class="relative h-full w-full overflow-y-auto select-none flex flex-col"
  style="background-color: var(--md-sys-color-surface-container-low);"
>
  <header
    class="sticky top-0 z-20 px-3 pt-3 pb-2.5 border-b backdrop-blur-md space-y-2"
    style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-low) 90%, transparent); border-color: color-mix(in srgb, var(--md-sys-color-outline-variant) 35%, transparent);"
  >
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1.5 min-w-0">
        <button
          type="button"
          onclick={onBack}
          aria-label="Kembali"
          class="min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
          style="color: var(--md-sys-color-on-surface);"
        >
          <Icon name="arrow-left" size={22} />
        </button>

        <h1 class="text-xl font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
          {tr(lang, 'nav.downloaded')}
        </h1>

        <span
          class="px-2 py-0.5 rounded-full text-[11px] font-bold tabular-nums shrink-0"
          style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
        >
          {totalCount}
        </span>
      </div>

      <div class="flex items-center gap-1 shrink-0">
        {#if onToggleDummy}
          <button
            type="button"
            onclick={onToggleDummy}
            class="min-h-9 px-2.5 rounded-full text-[11px] font-semibold cursor-pointer transition-colors"
            style={isDummyActive
              ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
              : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);'}
          >
            Demo
          </button>
        {/if}

        <button
          type="button"
          onclick={() => (previewSkeleton = !previewSkeleton)}
          class="min-h-9 px-2.5 rounded-full text-[11px] font-semibold cursor-pointer transition-colors"
          style={previewSkeleton
            ? 'background-color: var(--md-sys-color-tertiary-container); color: var(--md-sys-color-on-tertiary-container);'
            : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);'}
        >
          Skeleton
        </button>

        <button
          type="button"
          onclick={() => {
            searchOpen = !searchOpen;
            if (!searchOpen) searchText = '';
          }}
          aria-label={tr(lang, 'nav.search')}
          class="min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
          style="color: {searchOpen ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface-variant)'};"
        >
          <Icon name={searchOpen ? 'x' : 'magnifying-glass'} size={20} />
        </button>
      </div>
    </div>

    {#if searchOpen}
      <div
        class="h-10 px-3 rounded-xl flex items-center gap-2 border"
        style="background-color: var(--md-sys-color-surface-container); border-color: color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent);"
      >
        <Icon name="magnifying-glass" size={16} style="color: var(--md-sys-color-on-surface-variant);" />
        <input
          type="search"
          bind:value={searchText}
          placeholder={lang === 'id' ? 'Cari riwayat unduhan...' : 'Search downloads...'}
          class="w-full bg-transparent text-sm outline-none"
          style="color: var(--md-sys-color-on-surface);"
        />
      </div>
    {/if}
  </header>

  <div class="px-4 py-4 pb-10 space-y-6 flex-1">
    {#if loading || previewSkeleton}
      {#each [0, 1] as groupIdx (groupIdx)}
        <section class="space-y-3 animate-pulse">
          <div class="h-4 w-28 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
          {#each [0, 1] as cardIdx (cardIdx)}
            <div
              class="relative overflow-clip rounded-2xl px-4 py-3.5 flex items-center gap-3.5 border"
              style="background-color: var(--md-sys-color-surface-container); border-color: color-mix(in srgb, var(--md-sys-color-outline-variant) 35%, transparent);"
            >
              <div class="w-11 h-11 rounded-lg shrink-0 bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="min-w-0 flex-1 space-y-2">
                <div class="h-3.5 w-3/5 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                <div class="h-2.5 w-2/5 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <div class="w-8 h-8 rounded-full bg-[var(--md-sys-color-surface-container-high)]"></div>
                <div class="w-8 h-8 rounded-full bg-[var(--md-sys-color-surface-container-high)]"></div>
              </div>
              {#if groupIdx === 0 && cardIdx === 0}
                <div class="absolute inset-x-0 bottom-0 h-1 bg-[var(--md-sys-color-primary)]/40"></div>
              {/if}
            </div>
          {/each}
        </section>
      {/each}
    {:else if filteredActive.length === 0 && groupedHistory.length === 0}
      <EmptyState
        icon="download"
        iconWeight="fill"
        iconSize={42}
        title={tr(lang, 'nav.noDownloads')}
        description={tr(lang, 'nav.noDownloadsDesc')}
      />
    {:else}
      {#if filteredActive.length > 0}
        <section class="space-y-2.5">
          <div class="flex items-center justify-between">
            <h2 class="text-sm font-medium" style="color: var(--md-sys-color-on-surface-variant);">
              {lang === 'id' ? 'Sedang Diunduh' : 'Downloading'} ({filteredActive.length})
            </h2>
          </div>

          <div class="space-y-2.5">
            {#each filteredActive as dl (dl.trackId)}
              {@const dlCover = resolveDownloadCover(dl)}
              {@const pct = Math.max(0, Math.min(100, Math.round(dl.progress || 0)))}
              <article
                class="relative overflow-clip rounded-2xl px-4 pt-3.5 pb-4 flex items-center gap-3.5 border"
                style="background-color: var(--md-sys-color-surface-container); border-color: color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent);"
              >
                <div
                  class="w-11 h-11 rounded-lg overflow-hidden shrink-0 flex items-center justify-center"
                  style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
                >
                  {#if dlCover}
                    <CoverImage src={dlCover} fallbackIcon="download" iconSize={20} />
                  {:else}
                    <Icon name="download" weight="bold" size={20} />
                  {/if}
                </div>

                <div class="min-w-0 flex-1">
                  <div class="text-sm font-semibold truncate" style="color: var(--md-sys-color-primary);">
                    {dl.title}
                  </div>
                  <div class="text-xs truncate mt-0.5 tabular-nums" style="color: {dl.status === 'failed' ? 'var(--md-sys-color-error)' : 'var(--md-sys-color-on-surface-variant)'};">
                    {#if dl.status === 'failed'}
                      {dl.artist} • {dl.error || 'Failed'}
                    {:else if dl.status === 'queued'}
                      {dl.artist} • {lang === 'id' ? 'Menunggu antrean...' : 'Queued...'}
                    {:else}
                      {dl.artist} • {pct}%
                    {/if}
                  </div>
                </div>

                {#if onCancelDownload}
                  <button
                    type="button"
                    onclick={() => onCancelDownload(dl.trackId)}
                    aria-label={tr(lang, 'nav.cancelDownload')}
                    title={tr(lang, 'nav.cancelDownload')}
                    class="min-w-10 min-h-10 rounded-full flex items-center justify-center shrink-0 cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                    style="color: var(--md-sys-color-on-surface-variant);"
                  >
                    <Icon name="x" size={18} />
                  </button>
                {/if}

                {#if dl.status !== 'failed'}
                  <progress
                    class="velora-chrome-progress absolute inset-x-0 bottom-0"
                    max="100"
                    value={dl.status === 'queued' ? undefined : pct}
                    aria-label="{dl.title} {pct}%"
                  ></progress>
                {/if}
              </article>
            {/each}
          </div>
        </section>
      {/if}

      {#each groupedHistory as group (group.label)}
        <section class="space-y-2.5">
          <h2 class="text-sm font-medium" style="color: var(--md-sys-color-on-surface-variant);">
            {group.label}
          </h2>

          <div class="space-y-2.5">
            {#each group.items as entry (`${entry.deleted ? 'del' : 'dl'}-${entry.track.id}`)}
              {@const item = entry.track}
              {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl, 96)}
              <article
                class="relative overflow-clip rounded-2xl px-4 py-3.5 flex items-center gap-3.5 border transition-colors"
                style="background-color: var(--md-sys-color-surface-container); border-color: color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent);"
              >
                <div
                  class="w-11 h-11 rounded-lg overflow-hidden shrink-0 flex items-center justify-center {entry.deleted ? 'opacity-45 grayscale' : ''}"
                  style="background-color: var(--md-sys-color-surface-container-highest);"
                >
                  <CoverImage src={cover} fallbackIcon="music-note" iconSize={20} isVideoThumb={isYTVideoThumb(cover)} />
                </div>

                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2 min-w-0">
                    <button
                      type="button"
                      disabled={entry.deleted}
                      onclick={() => onPlayTrack(item, downloadedTracks)}
                      class="text-sm font-semibold truncate text-left cursor-pointer disabled:cursor-default {entry.deleted ? 'line-through opacity-55' : 'underline decoration-1 underline-offset-2'}"
                      style="color: {entry.deleted ? 'var(--md-sys-color-on-surface-variant)' : 'var(--md-sys-color-primary)'};"
                    >
                      {item.title}
                    </button>
                    {#if entry.deleted}
                      <span class="text-xs font-medium shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                        Deleted
                      </span>
                    {/if}
                  </div>
                  <div class="text-xs truncate mt-0.5 tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
                    {item.artistName}{item.durationSec > 0 ? ` • ${formatDuration(item.durationSec)}` : ''}{item.format ? ` • ${item.format}` : ''}
                  </div>
                </div>

                <div class="flex items-center gap-0.5 shrink-0">
                  {#if onCopyLink}
                    <button
                      type="button"
                      onclick={() => onCopyLink(item)}
                      aria-label={tr(lang, 'nav.copyLink')}
                      title={tr(lang, 'nav.copyLink')}
                      class="min-w-10 min-h-10 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                      style="color: var(--md-sys-color-on-surface-variant);"
                    >
                      <Icon name="link" size={17} />
                    </button>
                  {/if}
                  {#if onOpenTrackAlbum && !entry.deleted}
                    <button
                      type="button"
                      onclick={() => onOpenTrackAlbum(item)}
                      aria-label={tr(lang, 'nav.goToAlbum')}
                      title={tr(lang, 'nav.goToAlbum')}
                      class="min-w-10 min-h-10 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                      style="color: var(--md-sys-color-on-surface-variant);"
                    >
                      <Icon name="folder-open" size={17} />
                    </button>
                  {/if}
                  {#if onRemoveDownload && !entry.deleted}
                    <button
                      type="button"
                      onclick={() => onRemoveDownload(item.id)}
                      aria-label={tr(lang, 'nav.removeDownload')}
                      title={tr(lang, 'nav.removeDownload')}
                      class="min-w-10 min-h-10 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                      style="color: var(--md-sys-color-on-surface-variant);"
                    >
                      <Icon name="x" size={17} />
                    </button>
                  {/if}
                </div>
              </article>
            {/each}
          </div>
        </section>
      {/each}
    {/if}
  </div>
</div>
