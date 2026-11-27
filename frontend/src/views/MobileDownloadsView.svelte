<script lang="ts">
  import { formatDuration, groupActiveDownloads, groupCompletedDownloads, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { DownloadProgress, GroupedCompletedDownload, GroupedDownloadItem, Track } from '../lib/types';
  import CoverImage from '../components/CoverImage.svelte';
  import EmptyState from '../components/EmptyState.svelte';
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
    onOpenDownloadedMusic?: (track: Track) => void;
    onOpenGroup?: (entry: GroupedCompletedDownload) => void;
    onCancelDownload?: (trackId: number) => void;
    onCancelDownloadGroup?: (groupKey: string, trackIds: number[]) => void;
    onPauseDownload?: (trackId: number) => void;
    onResumeDownload?: (trackId: number) => void;
    onPauseDownloadGroup?: (groupKey: string) => void;
    onResumeDownloadGroup?: (groupKey: string) => void;
    onRemoveDownload?: (trackId: number) => void;
    onRemoveDownloadGroup?: (tracks: Track[]) => void;
    onDismissDeletedEntry?: (entry: GroupedCompletedDownload) => void;
    onRetryFailedEntry?: (entry: GroupedCompletedDownload) => void;
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
    onOpenDownloadedMusic,
    onOpenGroup,
    onCancelDownload,
    onCancelDownloadGroup,
    onPauseDownload,
    onResumeDownload,
    onPauseDownloadGroup,
    onResumeDownloadGroup,
    onRemoveDownload,
    onRemoveDownloadGroup,
    onDismissDeletedEntry,
    onRetryFailedEntry,
    onCopyLink
  }: Props = $props();

  let categoryFilter = $state<'all' | 'music' | 'album' | 'playlist'>('all');
  let openActionKey = $state<string | null>(null);
  let hoveredCoverKey = $state<string | null>(null);

  function formatBytes(bytes: number): string {
    const b = Math.max(0, Number(bytes) || 0);
    if (b < 1024) return `${Math.round(b)} B`;
    if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
    if (b < 1024 * 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(1)} MB`;
    return `${(b / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  }

  function formatDownloadMetrics(dl: GroupedDownloadItem, pct: number): string {
    const speedStr = `${formatBytes(dl.speedBps)}/s`;
    const dlStr = formatBytes(dl.downloadedBytes);
    const sizePart =
      dl.totalBytes > 0 && dl.totalBytes >= dl.downloadedBytes
        ? `${dlStr} of ${formatBytes(dl.totalBytes)}`
        : dlStr;
    const statusText =
      dl.status === 'failed'
        ? dl.error || 'Failed'
        : dl.status === 'paused'
          ? tr(lang, 'nav.paused')
          : dl.status === 'queued'
            ? lang === 'id'
              ? 'Menunggu'
              : 'Queued'
            : dl.isGroup
              ? `${dl.completedCount}/${dl.totalCount} (${pct}%)`
              : `${pct}%`;
    return `${speedStr} - ${sizePart}, ${statusText}`;
  }

  function formatCollectionDuration(totalSec: number): string {
    const s = Math.max(0, Math.floor(totalSec || 0));
    const hours = Math.floor(s / 3600);
    const mins = Math.floor((s % 3600) / 60);
    if (hours > 0) {
      const hUnit = lang === 'id' ? 'j' : 'h';
      return mins > 0 ? `${hours}${hUnit} ${mins}m` : `${hours}${hUnit}`;
    }
    return formatDuration(s);
  }

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

<svelte:window onclick={() => (openActionKey = null)} />

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

  <div class="px-4 py-3 pb-10 space-y-8 flex-1 flex flex-col">
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
        iconWeight="bold"
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

          <div class="space-y-3">
            {#each filteredActive as dl (dl.key)}
              {@const dlCover = resolveDownloadCover(dl)}
              {@const pct = Math.max(0, Math.min(100, Math.round(dl.progress || 0)))}
              {@const fallbackIcon = dl.kind === 'playlist' ? 'playlist' : dl.kind === 'album' ? 'vinyl-record' : 'music-note'}
              <article
                class="rounded-xl px-3.5 py-3 flex items-start gap-3.5"
                style="background-color: var(--md-sys-color-surface-container);"
              >
                <button
                  type="button"
                  disabled={dl.status === 'failed'}
                  onmouseenter={() => (hoveredCoverKey = dl.key)}
                  onmouseleave={() => {
                    if (hoveredCoverKey === dl.key) hoveredCoverKey = null;
                  }}
                  onclick={() => {
                    if (dl.status === 'paused') {
                      if (dl.isGroup && onResumeDownloadGroup) {
                        onResumeDownloadGroup(dl.key);
                      } else {
                        onResumeDownload?.(dl.trackId);
                      }
                    } else if (dl.status === 'downloading' || dl.status === 'queued') {
                      if (dl.isGroup && onPauseDownloadGroup) {
                        onPauseDownloadGroup(dl.key);
                      } else {
                        onPauseDownload?.(dl.trackId);
                      }
                    }
                  }}
                  aria-label={dl.status === 'paused' ? tr(lang, 'nav.resumeDownload') : tr(lang, 'nav.pauseDownload')}
                  class="relative w-14 h-14 rounded-lg overflow-hidden shrink-0 flex items-center justify-center cursor-pointer disabled:cursor-default"
                  style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-primary);"
                >
                  <CoverImage src={dlCover} {fallbackIcon} iconSize={22} isVideoThumb={isYTVideoThumb(dlCover)} />
                  {#if dl.status === 'downloading' || dl.status === 'queued'}
                    <div class="absolute inset-0 bg-black/55 transition-colors flex items-center justify-center text-white">
                      {#if hoveredCoverKey === dl.key}
                        <Icon name="pause" size={20} weight="fill" />
                      {:else}
                        <div class="velora-dl-anim flex items-center justify-center">
                          <Icon name="arrow-down" size={20} weight="bold" />
                        </div>
                      {/if}
                    </div>
                  {:else if dl.status === 'paused'}
                    <div class="absolute inset-0 bg-black/65 transition-colors flex items-center justify-center text-white">
                      {#if hoveredCoverKey === dl.key}
                        <Icon name="play" size={20} weight="fill" />
                      {:else}
                        <Icon name="pause" size={20} weight="fill" />
                      {/if}
                    </div>
                  {/if}
                </button>

                <div class="min-w-0 flex-1 min-h-[56px] flex flex-col justify-between">
                  <div class="space-y-0.5">
                    <div class="flex items-start justify-between gap-2 min-w-0">
                      <div class="flex items-center gap-2 min-w-0 flex-1">
                        <div class="text-sm font-semibold leading-tight truncate min-w-0" style="color: var(--md-sys-color-on-surface);">
                          {dl.title}
                        </div>
                        {#if dl.status === 'paused'}
                          <span class="text-xs font-medium shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                            {tr(lang, 'nav.paused')}
                          </span>
                        {:else if dl.status === 'failed'}
                          <span class="text-xs font-medium shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                            Failed
                          </span>
                        {/if}
                      </div>

                      {#if dl.isGroup ? (onCancelDownloadGroup || onPauseDownloadGroup || onResumeDownloadGroup) : (onCancelDownload || onPauseDownload || onResumeDownload)}
                        <div class="relative shrink-0 flex items-center justify-end">
                          <button
                            type="button"
                            onclick={(e) => {
                              e.stopPropagation();
                              openActionKey = openActionKey === dl.key ? null : dl.key;
                            }}
                            aria-label={tr(lang, 'nav.trackActions')}
                            aria-expanded={openActionKey === dl.key}
                            aria-haspopup="menu"
                            class="w-8 h-8 -mt-1 -mr-1 flex items-center justify-center bg-transparent hover:bg-transparent cursor-pointer transition-colors {openActionKey === dl.key
                              ? 'opacity-100 text-[var(--md-sys-color-on-surface)]'
                              : 'opacity-65 hover:opacity-100 text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)]'}"
                          >
                            <Icon name="dots-three-vertical" weight="bold" size={17} />
                          </button>

                          {#if openActionKey === dl.key}
                            <!-- svelte-ignore a11y_click_events_have_key_events -->
                            <!-- svelte-ignore a11y_no_static_element_interactions -->
                            <div
                              onclick={(e) => e.stopPropagation()}
                              role="menu"
                              tabindex="-1"
                              class="absolute right-0 top-8 z-30 min-w-[152px] rounded-xl p-1.5 shadow-[0_14px_40px_rgba(0,0,0,0.48)] border border-[var(--md-sys-color-outline-variant)]/35 text-xs"
                              style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
                            >
                              {#if dl.status === 'paused'}
                                {#if dl.isGroup ? onResumeDownloadGroup : onResumeDownload}
                                  <button
                                    type="button"
                                    onclick={() => {
                                      openActionKey = null;
                                      if (dl.isGroup && onResumeDownloadGroup) {
                                        onResumeDownloadGroup(dl.key);
                                      } else {
                                        onResumeDownload?.(dl.trackId);
                                      }
                                    }}
                                    class="w-full px-3 py-2 rounded-lg text-xs font-medium flex items-center gap-2.5 text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
                                    style="color: var(--md-sys-color-on-surface);"
                                  >
                                    <Icon name="play" size={14} weight="fill" />
                                    <span>{tr(lang, 'nav.resumeDownload')}</span>
                                  </button>
                                {/if}
                              {:else if dl.status !== 'failed' && (dl.isGroup ? onPauseDownloadGroup : onPauseDownload)}
                                <button
                                  type="button"
                                  onclick={() => {
                                    openActionKey = null;
                                    if (dl.isGroup && onPauseDownloadGroup) {
                                      onPauseDownloadGroup(dl.key);
                                    } else {
                                      onPauseDownload?.(dl.trackId);
                                    }
                                  }}
                                  class="w-full px-3 py-2 rounded-lg text-xs font-medium flex items-center gap-2.5 text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
                                  style="color: var(--md-sys-color-on-surface);"
                                >
                                  <Icon name="pause" size={14} weight="fill" />
                                  <span>{tr(lang, 'nav.pauseDownload')}</span>
                                </button>
                              {/if}

                              {#if dl.isGroup ? onCancelDownloadGroup : onCancelDownload}
                                <button
                                  type="button"
                                  onclick={() => {
                                    openActionKey = null;
                                    if (dl.isGroup && onCancelDownloadGroup) {
                                      onCancelDownloadGroup(dl.key, dl.trackIds);
                                    } else {
                                      onCancelDownload?.(dl.trackId);
                                    }
                                  }}
                                  class="w-full px-3 py-2 rounded-lg text-xs font-medium flex items-center gap-2.5 text-left cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
                                  style="color: var(--md-sys-color-on-surface);"
                                >
                                  <Icon name="x" size={14} />
                                  <span>{tr(lang, 'nav.cancel')}</span>
                                </button>
                              {/if}
                            </div>
                          {/if}
                        </div>
                      {/if}
                    </div>

                    <div class="text-xs leading-tight truncate tabular-nums" style="color: {dl.status === 'failed' ? 'var(--md-sys-color-error)' : 'var(--md-sys-color-on-surface-variant)'};">
                      {formatDownloadMetrics(dl, pct)}
                    </div>
                  </div>

                  {#if dl.status !== 'failed'}
                    <progress
                      class="velora-chrome-progress w-full rounded-full mt-1.5"
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

      {#if filteredActive.length === 0 && groupedHistory.length === 0}
        <EmptyState
          icon="download"
          iconWeight="bold"
          iconSize={42}
          title={tr(lang, 'nav.noDownloads')}
          description={tr(lang, 'nav.noDownloadsDesc')}
        />
      {/if}

      {#each groupedHistory as group (group.label)}
        <section class="space-y-3 pt-1">
          <h2 class="text-sm font-medium" style="color: var(--md-sys-color-on-surface-variant);">
            {group.label}
          </h2>

          <div class="space-y-3">
            {#each group.items as entry (entry.key)}
              {@const item = entry.primaryTrack}
              {@const cover = resolveCoverUrl(entry.coverUrl, mediaBaseUrl, 96)}
              {@const fallbackIcon = entry.kind === 'playlist' ? 'playlist' : entry.kind === 'album' ? 'vinyl-record' : 'music-note'}
              <article
                class="rounded-xl px-3.5 py-3 flex items-start gap-3.5 transition-colors"
                style="background-color: var(--md-sys-color-surface-container);"
              >
                <div
                  class="w-12 h-12 rounded-lg overflow-hidden shrink-0 flex items-center justify-center {entry.deleted ? 'opacity-45 grayscale' : ''}"
                  style="background-color: var(--md-sys-color-surface-container-highest);"
                >
                  <CoverImage src={cover} {fallbackIcon} iconSize={20} isVideoThumb={isYTVideoThumb(cover)} />
                </div>

                <div class="min-w-0 flex-1 space-y-0.5">
                  <div class="flex items-start justify-between gap-2 min-w-0">
                    <div class="flex items-center gap-2 min-w-0 flex-1">
                      <button
                        type="button"
                        disabled={entry.deleted}
                        onclick={() => onPlayTrack(item, entry.isGroup ? entry.tracks : downloadedTracks)}
                        class="text-sm font-semibold leading-tight truncate text-left cursor-pointer disabled:cursor-default {entry.deleted ? 'line-through opacity-55' : 'hover:underline decoration-1 underline-offset-2'}"
                        style="color: {entry.deleted ? 'var(--md-sys-color-on-surface-variant)' : 'var(--md-sys-color-on-surface)'};"
                      >
                        <span class="truncate">{entry.title}</span>
                      </button>
                      {#if entry.deleted}
                        <span class="text-xs font-medium shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                          {entry.deletedReason === 'canceled' ? 'Canceled' : entry.deletedReason === 'failed' ? 'Failed' : 'Deleted'}
                        </span>
                      {/if}
                    </div>

                    <div class="flex items-center gap-1 shrink-0 -mt-1">
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
                      {#if !entry.deleted && ((entry.isGroup && onOpenGroup) || (!entry.isGroup && (onOpenDownloadedMusic || onOpenTrackAlbum)))}
                        <button
                          type="button"
                          onclick={() => {
                            if (entry.isGroup && onOpenGroup) {
                              onOpenGroup(entry);
                            } else if (onOpenDownloadedMusic) {
                              onOpenDownloadedMusic(item);
                            } else if (onOpenTrackAlbum) {
                              onOpenTrackAlbum(item);
                            }
                          }}
                          aria-label={entry.kind === 'playlist' ? tr(lang, 'nav.playlist') : entry.kind === 'album' ? tr(lang, 'nav.goToAlbum') : tr(lang, 'nav.music')}
                          title={entry.kind === 'playlist' ? tr(lang, 'nav.playlist') : entry.kind === 'album' ? tr(lang, 'nav.goToAlbum') : tr(lang, 'nav.music')}
                          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
                        >
                          <Icon name="folder-open" size={16} />
                        </button>
                      {/if}
                      {#if entry.deleted}
                        {#if entry.deletedReason === 'failed' && onRetryFailedEntry}
                          <button
                            type="button"
                            onclick={() => onRetryFailedEntry(entry)}
                            aria-label={lang === 'id' ? 'Coba lagi' : 'Retry'}
                            title={lang === 'id' ? 'Coba lagi' : 'Retry'}
                            class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
                          >
                            <Icon name="arrow-clockwise" size={16} />
                          </button>
                        {/if}
                        {#if onDismissDeletedEntry}
                          <button
                            type="button"
                            onclick={() => onDismissDeletedEntry(entry)}
                            aria-label={tr(lang, 'nav.close')}
                            title={tr(lang, 'nav.close')}
                            class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
                          >
                            <Icon name="x" size={16} />
                          </button>
                        {/if}
                      {:else if entry.isGroup ? onRemoveDownloadGroup : onRemoveDownload}
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
                      {@const kindLabel = entry.kind === 'album' ? tr(lang, 'nav.album') : tr(lang, 'nav.playlist')}
                      {kindLabel} • {entry.tracks.length} {tr(lang, 'nav.songs').toLowerCase()}{entry.totalDurationSec > 0 ? ` • ${formatCollectionDuration(entry.totalDurationSec)}` : ''}
                    {:else}
                      {tr(lang, 'nav.music')} • {item.artistName}{item.durationSec > 0 ? ` • ${formatDuration(item.durationSec)}` : ''}{item.format ? ` • ${item.format}` : ''}
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

<style>
  @keyframes velora-dl-drop {
    0%, 100% {
      transform: translateY(-2.5px);
    }
    50% {
      transform: translateY(3px);
    }
  }

  .velora-dl-anim {
    animation: velora-dl-drop 1.1s ease-in-out infinite;
  }
</style>
