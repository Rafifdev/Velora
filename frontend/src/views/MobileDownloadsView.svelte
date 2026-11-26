<script lang="ts">
  import { formatDuration, groupActiveDownloads, groupCompletedDownloads, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { DownloadProgress, GroupedCompletedDownload, GroupedDownloadItem, Track } from '../lib/types';
  import CoverImage from '../components/CoverImage.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import EqIndicator from '../components/EqIndicator.svelte';
  import Icon from '../components/Icon.svelte';
  import Skeleton from '../components/Skeleton.svelte';

  interface Props {
    activeDownloads: DownloadProgress[];
    downloadedTracks: Track[];
    deletedTracks?: Track[];
    allTracks?: Track[];
    loading?: boolean;
    mediaBaseUrl: string;
    lang?: LangCode;
    onBack: () => void;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onCancelDownload?: (trackId: number) => void;
    onCancelDownloadGroup?: (groupKey: string, trackIds: number[]) => void;
    onRemoveDownload?: (trackId: number) => void;
    onRemoveDownloadGroup?: (tracks: Track[]) => void;
    onCopyLink?: (track: Track) => void;
  }

  let {
    activeDownloads,
    downloadedTracks,
    deletedTracks = [],
    allTracks = [],
    loading = false,
    mediaBaseUrl,
    lang = 'en',
    onBack,
    onPlayTrack,
    onOpenTrackAlbum,
    onCancelDownload,
    onCancelDownloadGroup,
    onRemoveDownload,
    onRemoveDownloadGroup,
    onCopyLink
  }: Props = $props();

  let categoryFilter = $state<'all' | 'music' | 'album' | 'playlist'>('all');

  function resolveDownloadCover(dl: GroupedDownloadItem): string {
    if (dl.coverUrl) {
      return resolveCoverUrl(dl.coverUrl, mediaBaseUrl, 96);
    }
    const matched =
      allTracks.find(
        (t) =>
          (dl.trackId > 0 && t.id === dl.trackId) ||
          (dl.externalId && t.externalId === dl.externalId)
      ) ?? allTracks.find((t) => Boolean(t.coverUrl));
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

  const filteredActive = $derived(groupActiveDownloads(activeDownloads));

  const allCompletedEntries = $derived(
    groupCompletedDownloads(downloadedTracks, deletedTracks, activeDownloads)
  );

  const categoryCounts = $derived({
    all: allCompletedEntries.length,
    music: allCompletedEntries.filter((e) => e.kind === 'music').length,
    album: allCompletedEntries.filter((e) => e.kind === 'album').length,
    playlist: allCompletedEntries.filter((e) => e.kind === 'playlist').length
  });

  const groupedHistory = $derived.by(() => {
    const visible =
      categoryFilter === 'all'
        ? allCompletedEntries
        : allCompletedEntries.filter((e) => e.kind === categoryFilter);

    const groups: { label: string; items: GroupedCompletedDownload[] }[] = [];
    const map = new Map<string, GroupedCompletedDownload[]>();
    for (const entry of visible) {
      const label = formatGroupDate(entry.downloadedAt);
      let arr = map.get(label);
      if (!arr) {
        arr = [];
        map.set(label, arr);
        groups.push({ label, items: arr });
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
    class="sticky top-0 z-20 px-3 pt-3 pb-2.5 backdrop-blur-md space-y-2"
    style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-low) 90%, transparent);"
  >
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1.5 min-w-0">
        <button
          type="button"
          onclick={onBack}
          aria-label="Kembali"
          class="velora-press min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
          style="color: var(--md-sys-color-on-surface);"
        >
          <Icon name="arrow-left" size={22} />
        </button>

        <h1 class="text-xl font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
          {tr(lang, 'nav.downloaded')}
        </h1>
      </div>
    </div>
  </header>

  <div class="px-4 py-3 pb-10 space-y-6 flex-1">
    {#if loading}
      {#each [0, 1] as groupIdx (groupIdx)}
        <section class="space-y-3">
          <Skeleton class="h-4 w-28 rounded bg-[var(--md-sys-color-surface-container-highest)]" />
          <Skeleton variant="download-list" count={2} />
        </section>
      {/each}
    {:else if filteredActive.length === 0 && allCompletedEntries.length === 0}
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
          <h2 class="text-sm font-medium" style="color: var(--md-sys-color-on-surface-variant);">
            {lang === 'id' ? 'Sedang Diunduh' : 'Downloading'} ({filteredActive.length})
          </h2>

          <div class="space-y-2.5">
            {#each filteredActive as dl (dl.key)}
              {@const dlCover = resolveDownloadCover(dl)}
              {@const pct = Math.max(0, Math.min(100, Math.round(dl.progress || 0)))}
              {@const fallbackIcon = dl.kind === 'playlist' ? 'playlist' : dl.kind === 'album' ? 'vinyl-record' : 'music-note'}
              <article
                class="rounded-xl px-3.5 py-3 flex items-center gap-3.5"
                style="background-color: var(--md-sys-color-surface-container);"
              >
                <div
                  class="relative w-12 h-12 rounded-lg overflow-hidden shrink-0 flex items-center justify-center"
                  style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-primary);"
                >
                  <CoverImage src={dlCover} {fallbackIcon} iconSize={20} isVideoThumb={isYTVideoThumb(dlCover)} />
                  {#if dl.status === 'downloading' || dl.status === 'queued'}
                    <div class="absolute inset-0 bg-black/40 pointer-events-none flex items-center justify-center">
                      <div class="velora-download-wave absolute inset-0 pointer-events-none"></div>
                      <div class="relative z-10 text-white">
                        <EqIndicator color="currentColor" />
                      </div>
                    </div>
                  {/if}
                </div>

                <div class="min-w-0 flex-1 space-y-1">
                  <div class="flex items-center justify-between gap-2 min-w-0">
                    <div class="text-sm font-semibold leading-tight truncate min-w-0" style="color: var(--md-sys-color-primary);">
                      {dl.title}
                    </div>

                    {#if dl.isGroup ? onCancelDownloadGroup : onCancelDownload}
                      <button
                        type="button"
                        onclick={() =>
                          dl.isGroup && onCancelDownloadGroup
                            ? onCancelDownloadGroup(dl.key, dl.trackIds)
                            : onCancelDownload?.(dl.trackId)}
                        aria-label={tr(lang, 'nav.cancelDownload')}
                        title={tr(lang, 'nav.cancelDownload')}
                        class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
                      >
                        <Icon name="x" size={16} />
                      </button>
                    {/if}
                  </div>

                  <div class="text-xs leading-tight truncate tabular-nums" style="color: {dl.status === 'failed' ? 'var(--md-sys-color-error)' : 'var(--md-sys-color-on-surface-variant)'};">
                    {#if dl.isGroup}
                      {@const kindLabel = dl.kind === 'album' ? tr(lang, 'nav.album') : tr(lang, 'nav.playlist')}
                      {#if dl.status === 'failed'}
                        {kindLabel} • {dl.error || 'Failed'}
                      {:else if dl.status === 'queued'}
                        {dl.completedCount} / {dl.totalCount} {tr(lang, 'nav.songs').toLowerCase()} • {lang === 'id' ? 'Menunggu antrean...' : 'Queued...'}
                      {:else}
                        {dl.completedCount} / {dl.totalCount} {tr(lang, 'nav.songs').toLowerCase()} • {pct}%
                      {/if}
                    {:else}
                      {#if dl.status === 'failed'}
                        {dl.subtitle} • {dl.error || 'Failed'}
                      {:else if dl.status === 'queued'}
                        {dl.subtitle} • {lang === 'id' ? 'Menunggu antrean...' : 'Queued...'}
                      {:else}
                        {dl.subtitle} • {pct}%
                      {/if}
                    {/if}
                  </div>

                  {#if dl.status !== 'failed'}
                    <progress
                      class="velora-chrome-progress w-full rounded-full mt-0.5"
                      max="100"
                      value={dl.status === 'queued' && !dl.isGroup ? undefined : pct}
                      aria-label="{dl.title} {pct}%"
                    ></progress>
                  {/if}
                </div>
              </article>
            {/each}
          </div>
        </section>
      {/if}

      {#if allCompletedEntries.length > 0}
        <div class="flex flex-wrap items-center gap-2">
          {#each [
            { id: 'all', label: tr(lang, 'nav.all'), count: categoryCounts.all },
            { id: 'music', label: tr(lang, 'nav.music'), count: categoryCounts.music },
            { id: 'album', label: tr(lang, 'nav.album'), count: categoryCounts.album },
            { id: 'playlist', label: tr(lang, 'nav.playlist'), count: categoryCounts.playlist }
          ] as pill (pill.id)}
            {@const active = categoryFilter === pill.id}
            <button
              type="button"
              onclick={() => (categoryFilter = pill.id as 'all' | 'music' | 'album' | 'playlist')}
              class="px-3.5 py-1.5 rounded-full text-xs font-semibold inline-flex items-center gap-1.5 cursor-pointer transition-colors"
              style={active
                ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
                : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
            >
              <span>{pill.label}</span>
              <span class="opacity-70 tabular-nums">{pill.count}</span>
            </button>
          {/each}
        </div>
      {/if}

      {#each groupedHistory as group (group.label)}
        <section class="space-y-2.5">
          <h2 class="text-sm font-medium" style="color: var(--md-sys-color-on-surface-variant);">
            {group.label}
          </h2>

          <div class="space-y-2.5">
            {#each group.items as entry (entry.key)}
              {@const item = entry.primaryTrack}
              {@const cover = resolveCoverUrl(entry.coverUrl, mediaBaseUrl, 96)}
              {@const fallbackIcon = entry.kind === 'playlist' ? 'playlist' : entry.kind === 'album' ? 'vinyl-record' : 'music-note'}
              <article
                class="rounded-xl px-3.5 py-3 flex items-center gap-3.5 transition-colors"
                style="background-color: var(--md-sys-color-surface-container);"
              >
                <div
                  class="w-12 h-12 rounded-lg overflow-hidden shrink-0 flex items-center justify-center {entry.deleted ? 'opacity-45 grayscale' : ''}"
                  style="background-color: var(--md-sys-color-surface-container-highest);"
                >
                  <CoverImage src={cover} {fallbackIcon} iconSize={20} isVideoThumb={isYTVideoThumb(cover)} />
                </div>

                <div class="min-w-0 flex-1 space-y-1">
                  <div class="flex items-center justify-between gap-2 min-w-0">
                    <div class="flex items-center gap-2 min-w-0 flex-1">
                      <button
                        type="button"
                        disabled={entry.deleted}
                        onclick={() => onPlayTrack(item, entry.isGroup ? entry.tracks : downloadedTracks)}
                        class="min-h-11 -my-2.5 flex items-center text-sm font-semibold leading-tight truncate text-left cursor-pointer disabled:cursor-default {entry.deleted ? 'line-through opacity-55' : 'underline decoration-1 underline-offset-2'}"
                        style="color: {entry.deleted ? 'var(--md-sys-color-on-surface-variant)' : 'var(--md-sys-color-primary)'};"
                      >
                        <span class="truncate">{entry.title}</span>
                      </button>
                      <span
                        class="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wide shrink-0"
                        style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface-variant);"
                      >
                        {entry.kind === 'album' ? tr(lang, 'nav.album') : entry.kind === 'playlist' ? tr(lang, 'nav.playlist') : tr(lang, 'nav.music')}
                      </span>
                      {#if entry.deleted}
                        <span class="text-xs font-medium shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                          Deleted
                        </span>
                      {/if}
                    </div>

                    <div class="flex items-center gap-1 shrink-0">
                      {#if !entry.isGroup && onCopyLink}
                        <button
                          type="button"
                          onclick={() => onCopyLink(item)}
                          aria-label={tr(lang, 'nav.copyLink')}
                          title={tr(lang, 'nav.copyLink')}
                          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
                        >
                          <Icon name="link" size={16} />
                        </button>
                      {/if}
                      {#if !entry.isGroup && onOpenTrackAlbum && !entry.deleted}
                        <button
                          type="button"
                          onclick={() => onOpenTrackAlbum(item)}
                          aria-label={tr(lang, 'nav.goToAlbum')}
                          title={tr(lang, 'nav.goToAlbum')}
                          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
                        >
                          <Icon name="folder-open" size={16} />
                        </button>
                      {/if}
                      {#if !entry.deleted && (entry.isGroup ? onRemoveDownloadGroup : onRemoveDownload)}
                        <button
                          type="button"
                          onclick={() =>
                            entry.isGroup && onRemoveDownloadGroup
                              ? onRemoveDownloadGroup(entry.tracks)
                              : onRemoveDownload?.(item.id)}
                          aria-label={tr(lang, 'nav.removeDownload')}
                          title={tr(lang, 'nav.removeDownload')}
                          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
                        >
                          <Icon name="x" size={16} />
                        </button>
                      {/if}
                    </div>
                  </div>

                  <div class="text-xs leading-tight truncate tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
                    {#if entry.isGroup}
                      {entry.tracks.length} {tr(lang, 'nav.songs').toLowerCase()}{entry.totalDurationSec > 0 ? ` • ${formatDuration(entry.totalDurationSec)}` : ''}
                    {:else}
                      {item.artistName}{item.durationSec > 0 ? ` • ${formatDuration(item.durationSec)}` : ''}{item.format ? ` • ${item.format}` : ''}
                    {/if}
                  </div>
                </div>
              </article>
            {/each}
          </div>
        </section>
      {/each}
    {/if}
  </div>
</div>
