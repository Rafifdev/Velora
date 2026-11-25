<script lang="ts">
  import { isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { Album, Artist, Playlist, Track, ViewRoute } from '../lib/types';
  import CoverImage from '../components/CoverImage.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import EqIndicator from '../components/EqIndicator.svelte';
  import Icon from '../components/Icon.svelte';
  import MobileTopHeader from '../components/MobileTopHeader.svelte';
  import TrackActionMenu from '../components/TrackActionMenu.svelte';

  export type LibraryFilter = 'all' | 'music' | 'albums' | 'artists' | 'downloaded';

  interface Props {
    tracks: Track[];
    albums: Album[];
    artists: Artist[];
    playlists: Playlist[];
    initialFilter?: LibraryFilter;
    currentTrackId: number | null;
    currentExternalId?: string | null;
    isPlaying: boolean;
    shuffle?: boolean;
    mediaBaseUrl: string;
    activeDownloadsCount?: number;
    lang?: LangCode;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onTogglePlay?: () => void;
    onPlayShuffled?: (list: Track[], sourceTitle?: string, sourceKey?: string) => void;
    onPlayNext?: (track: Track) => void;
    onAddToQueue?: (track: Track) => void;
    onAddToPlaylist?: (playlistId: number, trackId: number, track?: Track) => void;
    onCreatePlaylistTrack?: (track?: Track) => void;
    onToggleFavoriteTrack: (id: number, track?: Track) => void;
    onToggleFavoriteAlbum: (id: number) => void;
    onToggleFavoriteArtist: (id: number) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenAlbum: (albumId: number) => void;
    onPlayAlbum?: (albumId: number) => void;
    onOpenArtist: (artistId: number) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onCopyLink?: (track: Track) => void;
    onNavigate: (route: ViewRoute) => void;
  }

  let {
    tracks,
    albums,
    artists,
    playlists,
    initialFilter = 'all',
    currentTrackId,
    currentExternalId = null,
    isPlaying,
    mediaBaseUrl,
    activeDownloadsCount = 0,
    lang = 'en',
    onPlayTrack,
    onTogglePlay,
    onPlayShuffled,
    onPlayNext,
    onAddToQueue,
    onAddToPlaylist,
    onCreatePlaylistTrack,
    onToggleFavoriteTrack,
    onToggleFavoriteAlbum,
    onToggleFavoriteArtist,
    onDownloadTrack,
    onRemoveDownload,
    onOpenAlbum,
    onPlayAlbum,
    onOpenArtist,
    onOpenTrackAlbum,
    onCopyLink,
    onNavigate
  }: Props = $props();

  let activeFilter = $state<LibraryFilter>('all');
  let filterQuery = $state('');
  let openMenuKey = $state<string | null>(null);

  $effect(() => {
    activeFilter = initialFilter || 'all';
  });

  const normalizedQuery = $derived(filterQuery.trim().toLowerCase());

  const likedTracks = $derived.by(() => {
    const base = tracks.filter((t) => t.isFavorite);
    if (!normalizedQuery) return base;
    return base.filter(
      (t) =>
        t.title.toLowerCase().includes(normalizedQuery) ||
        t.artistName.toLowerCase().includes(normalizedQuery) ||
        t.albumTitle.toLowerCase().includes(normalizedQuery)
    );
  });

  const downloadedTracks = $derived.by(() => {
    const base = tracks.filter((t) => t.isDownloaded);
    if (!normalizedQuery) return base;
    return base.filter(
      (t) =>
        t.title.toLowerCase().includes(normalizedQuery) ||
        t.artistName.toLowerCase().includes(normalizedQuery) ||
        t.albumTitle.toLowerCase().includes(normalizedQuery)
    );
  });

  const favAlbums = $derived.by(() => {
    const base = albums.filter((a) => a.isFavorite);
    if (!normalizedQuery) return base;
    return base.filter(
      (a) =>
        a.title.toLowerCase().includes(normalizedQuery) ||
        a.artistName.toLowerCase().includes(normalizedQuery)
    );
  });

  const favArtists = $derived.by(() => {
    const base = artists.filter((a) => a.isFavorite);
    if (!normalizedQuery) return base;
    return base.filter((a) => a.name.toLowerCase().includes(normalizedQuery));
  });

  const filterPills = $derived([
    { id: 'all' as const, label: tr(lang, 'nav.all'), count: null },
    { id: 'music' as const, label: tr(lang, 'nav.favoriteMusic'), count: likedTracks.length },
    { id: 'downloaded' as const, label: tr(lang, 'nav.downloaded'), count: downloadedTracks.length },
    { id: 'albums' as const, label: tr(lang, 'nav.favoriteAlbums'), count: favAlbums.length },
    { id: 'artists' as const, label: tr(lang, 'nav.favoriteArtists'), count: favArtists.length }
  ]);

  const totalMatches = $derived(
    likedTracks.length + downloadedTracks.length + favAlbums.length + favArtists.length
  );

  function isTrackActive(trItem: Track): boolean {
    if (currentTrackId !== null && trItem.id === currentTrackId) return true;
    if (currentExternalId && trItem.externalId && trItem.externalId === currentExternalId) return true;
    return false;
  }

  function playShuffleList(list: Track[], sourceTitle: string, sourceKey: string) {
    if (list.length === 0) return;
    if (onPlayShuffled) {
      onPlayShuffled(list, sourceTitle, sourceKey);
      return;
    }
    const idx = Math.floor(Math.random() * list.length);
    onPlayTrack(list[idx], list);
  }
</script>

<div
  class="relative h-full w-full overflow-y-auto select-none flex flex-col"
  style="background-color: var(--md-sys-color-surface-container-low);"
  onclick={() => {
    openMenuKey = null;
  }}
  role="presentation"
>
  <MobileTopHeader
    title={tr(lang, 'nav.library')}
    {lang}
    {activeDownloadsCount}
    {onNavigate}
  />

  <div class="px-4 pt-2 pb-2 space-y-2.5 sticky top-0 z-20 backdrop-blur-md" style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-low) 88%, transparent);">
    <div
      class="h-11 px-3.5 rounded-xl flex items-center gap-2.5 border"
      style="background-color: var(--md-sys-color-surface-container); border-color: color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent);"
    >
      <Icon name="magnifying-glass" size={18} style="color: var(--md-sys-color-on-surface-variant);" />
      <input
        type="search"
        bind:value={filterQuery}
        placeholder={lang === 'id' ? 'Cari di Library...' : 'Search in Library...'}
        aria-label={lang === 'id' ? 'Cari di Library' : 'Search in Library'}
        class="w-full bg-transparent text-sm outline-none placeholder:opacity-65"
        style="color: var(--md-sys-color-on-surface);"
      />
      {#if filterQuery}
        <button
          type="button"
          onclick={() => (filterQuery = '')}
          aria-label="Clear search"
          class="min-w-8 min-h-8 rounded-full flex items-center justify-center cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
        >
          <Icon name="x" size={15} />
        </button>
      {/if}
    </div>

    <div class="flex flex-nowrap items-center gap-2 overflow-x-auto no-scrollbar pb-0.5">
      {#each filterPills as pill (pill.id)}
        {@const active = activeFilter === pill.id}
        <button
          type="button"
          onclick={(e) => {
            (e.currentTarget as HTMLElement).scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });
            activeFilter = pill.id;
          }}
          class="shrink-0 whitespace-nowrap min-h-9 px-3.5 py-1.5 rounded-full text-xs font-semibold flex items-center gap-1.5 cursor-pointer transition-colors duration-150"
          style={active
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
        >
          <span>{pill.label}</span>
          {#if pill.count !== null}
            <span class="opacity-70 tabular-nums text-[11px]">{pill.count}</span>
          {/if}
        </button>
      {/each}
    </div>
  </div>

  <div class="px-4 pt-2 pb-10 space-y-6 flex-1">
    {#if (activeFilter === 'all' && totalMatches === 0) ||
      (activeFilter === 'music' && likedTracks.length === 0) ||
      (activeFilter === 'downloaded' && downloadedTracks.length === 0) ||
      (activeFilter === 'albums' && favAlbums.length === 0) ||
      (activeFilter === 'artists' && favArtists.length === 0)}
      <EmptyState
        icon={activeFilter === 'downloaded'
          ? 'download'
          : activeFilter === 'albums'
            ? 'vinyl-record'
            : activeFilter === 'artists'
              ? 'user'
              : 'books'}
        iconWeight="fill"
        iconSize={40}
        title={activeFilter === 'downloaded'
          ? tr(lang, 'nav.noDownloads')
          : activeFilter === 'albums'
            ? tr(lang, 'nav.noFavAlbums')
            : activeFilter === 'artists'
              ? tr(lang, 'nav.noFavArtists')
              : tr(lang, 'nav.noFavSongs')}
        description={activeFilter === 'downloaded'
          ? tr(lang, 'nav.noDownloadsDesc')
          : activeFilter === 'albums'
            ? tr(lang, 'nav.noFavAlbumsDesc')
            : activeFilter === 'artists'
              ? tr(lang, 'nav.noFavArtistsDesc')
              : tr(lang, 'nav.noFavSongsDesc')}
      >
        <button
          type="button"
          onclick={() => onNavigate({ name: 'home' })}
          class="mt-5 min-h-11 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform active:scale-95"
          style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
        >
          {tr(lang, 'nav.exploreSongs')}
        </button>
      </EmptyState>
    {:else}
      {#if (activeFilter === 'all' || activeFilter === 'music') && likedTracks.length > 0}
        {@const visibleMusic = activeFilter === 'all' ? likedTracks.slice(0, 6) : likedTracks}
        <section class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span
                class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0"
                style="background-color: var(--md-sys-color-tertiary-container); color: var(--md-sys-color-on-tertiary-container);"
              >
                <Icon name="heart" weight="fill" size={15} />
              </span>
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.favoriteMusic')}
              </h2>
              <span class="text-xs font-semibold tabular-nums opacity-65">{likedTracks.length}</span>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <button
                type="button"
                onclick={() => playShuffleList(likedTracks, tr(lang, 'nav.favoriteMusic'), 'liked-songs')}
                aria-label={tr(lang, 'nav.shuffle')}
                class="min-w-9 min-h-9 rounded-full flex items-center justify-center cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                <Icon name="shuffle" size={18} />
              </button>
              <button
                type="button"
                onclick={() => likedTracks[0] && onPlayTrack(likedTracks[0], likedTracks)}
                aria-label={tr(lang, 'nav.playAll')}
                class="min-w-9 min-h-9 rounded-full flex items-center justify-center cursor-pointer shadow-sm"
                style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
              >
                <Icon name="play" weight="fill" size={16} />
              </button>
              {#if activeFilter === 'all' && likedTracks.length > 6}
                <button
                  type="button"
                  onclick={() => (activeFilter = 'music')}
                  class="min-h-9 px-2.5 text-xs font-bold cursor-pointer hover:underline"
                  style="color: var(--md-sys-color-primary);"
                >
                  {tr(lang, 'nav.showAll')}
                </button>
              {/if}
            </div>
          </div>

          <div class="space-y-1">
            {#each visibleMusic as item (item.id)}
              {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl, 96)}
              {@const active = isTrackActive(item)}
              {@const mKey = `lib-music-${item.id}`}
              <div
                class="min-h-14 px-2 py-1.5 rounded-xl flex items-center gap-3 transition-colors {active
                  ? 'bg-[var(--md-sys-color-state-active)]'
                  : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
              >
                <button
                  type="button"
                  onclick={() => (active && onTogglePlay ? onTogglePlay() : onPlayTrack(item, likedTracks))}
                  class="min-w-0 flex-1 flex items-center gap-3 text-left cursor-pointer"
                >
                  <div class="w-12 h-12 rounded-lg overflow-hidden shrink-0 relative bg-[var(--md-sys-color-surface-container-high)]">
                    <CoverImage src={cover} fallbackIcon="music-note" iconSize={20} isVideoThumb={isYTVideoThumb(cover)} />
                    {#if active && isPlaying}
                      <div class="absolute inset-0 bg-black/45 flex items-center justify-center text-white">
                        <EqIndicator color="currentColor" />
                      </div>
                    {/if}
                  </div>
                  <div class="min-w-0 flex-1">
                    <div
                      class="text-sm font-semibold truncate"
                      style="color: {active ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface)'};"
                    >
                      {item.title}
                    </div>
                    <div class="flex items-center gap-1.5 text-xs truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      {#if item.isDownloaded}
                        <Icon name="check-circle" weight="fill" size={13} class="text-[var(--md-sys-color-primary)] shrink-0" />
                      {/if}
                      <span class="truncate">{item.artistName}</span>
                    </div>
                  </div>
                </button>

                <button
                  type="button"
                  onclick={() => onToggleFavoriteTrack(item.id, item)}
                  aria-label={tr(lang, 'nav.likeSong')}
                  class="min-w-10 min-h-10 rounded-full flex items-center justify-center cursor-pointer velora-heart-btn is-loved"
                  style="color: var(--md-sys-color-tertiary);"
                >
                  <Icon name="heart" weight="fill" size={18} />
                </button>

                <TrackActionMenu
                  track={item}
                  open={openMenuKey === mKey}
                  {playlists}
                  {lang}
                  triggerClass="!w-10 !h-10"
                  onToggleOpen={() => (openMenuKey = openMenuKey === mKey ? null : mKey)}
                  onClose={() => { if (openMenuKey === mKey) openMenuKey = null; }}
                  {onPlayNext}
                  {onAddToQueue}
                  {onAddToPlaylist}
                  onCreatePlaylist={onCreatePlaylistTrack}
                  onToggleFavorite={onToggleFavoriteTrack}
                  {onDownloadTrack}
                  {onRemoveDownload}
                  {onOpenArtist}
                  {onOpenTrackAlbum}
                  {onCopyLink}
                />
              </div>
            {/each}
          </div>
        </section>
      {/if}

      {#if (activeFilter === 'all' || activeFilter === 'downloaded') && downloadedTracks.length > 0}
        {@const visibleDl = activeFilter === 'all' ? downloadedTracks.slice(0, 6) : downloadedTracks}
        <section class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span
                class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0"
                style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
              >
                <Icon name="download" weight="fill" size={15} />
              </span>
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.downloaded')}
              </h2>
              <span class="text-xs font-semibold tabular-nums opacity-65">{downloadedTracks.length}</span>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <button
                type="button"
                onclick={() => playShuffleList(downloadedTracks, tr(lang, 'nav.downloaded'), 'downloaded')}
                aria-label={tr(lang, 'nav.shuffle')}
                class="min-w-9 min-h-9 rounded-full flex items-center justify-center cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                <Icon name="shuffle" size={18} />
              </button>
              <button
                type="button"
                onclick={() => downloadedTracks[0] && onPlayTrack(downloadedTracks[0], downloadedTracks)}
                aria-label={tr(lang, 'nav.playAll')}
                class="min-w-9 min-h-9 rounded-full flex items-center justify-center cursor-pointer shadow-sm"
                style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
              >
                <Icon name="play" weight="fill" size={16} />
              </button>
              {#if activeFilter === 'all' && downloadedTracks.length > 6}
                <button
                  type="button"
                  onclick={() => (activeFilter = 'downloaded')}
                  class="min-h-9 px-2.5 text-xs font-bold cursor-pointer hover:underline"
                  style="color: var(--md-sys-color-primary);"
                >
                  {tr(lang, 'nav.showAll')}
                </button>
              {/if}
            </div>
          </div>

          <div class="space-y-1">
            {#each visibleDl as item (item.id)}
              {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl, 96)}
              {@const active = isTrackActive(item)}
              {@const dKey = `lib-dl-${item.id}`}
              <div
                class="min-h-14 px-2 py-1.5 rounded-xl flex items-center gap-3 transition-colors {active
                  ? 'bg-[var(--md-sys-color-state-active)]'
                  : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
              >
                <button
                  type="button"
                  onclick={() => (active && onTogglePlay ? onTogglePlay() : onPlayTrack(item, downloadedTracks))}
                  class="min-w-0 flex-1 flex items-center gap-3 text-left cursor-pointer"
                >
                  <div class="w-12 h-12 rounded-lg overflow-hidden shrink-0 relative bg-[var(--md-sys-color-surface-container-high)]">
                    <CoverImage src={cover} fallbackIcon="music-note" iconSize={20} isVideoThumb={isYTVideoThumb(cover)} />
                    {#if active && isPlaying}
                      <div class="absolute inset-0 bg-black/45 flex items-center justify-center text-white">
                        <EqIndicator color="currentColor" />
                      </div>
                    {/if}
                  </div>
                  <div class="min-w-0 flex-1">
                    <div
                      class="text-sm font-semibold truncate"
                      style="color: {active ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface)'};"
                    >
                      {item.title}
                    </div>
                    <div class="flex items-center gap-1.5 text-xs truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                      <Icon name="check-circle" weight="fill" size={13} class="text-[var(--md-sys-color-primary)] shrink-0" />
                      <span class="truncate">{item.artistName}</span>
                    </div>
                  </div>
                </button>

                <TrackActionMenu
                  track={item}
                  open={openMenuKey === dKey}
                  {playlists}
                  {lang}
                  triggerClass="!w-10 !h-10"
                  onToggleOpen={() => (openMenuKey = openMenuKey === dKey ? null : dKey)}
                  onClose={() => { if (openMenuKey === dKey) openMenuKey = null; }}
                  {onPlayNext}
                  {onAddToQueue}
                  {onAddToPlaylist}
                  onCreatePlaylist={onCreatePlaylistTrack}
                  onToggleFavorite={onToggleFavoriteTrack}
                  {onDownloadTrack}
                  {onRemoveDownload}
                  {onOpenArtist}
                  {onOpenTrackAlbum}
                  {onCopyLink}
                />
              </div>
            {/each}
          </div>
        </section>
      {/if}

      {#if (activeFilter === 'all' || activeFilter === 'albums') && favAlbums.length > 0}
        <section class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span
                class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0"
                style="background-color: var(--md-sys-color-secondary-container); color: var(--md-sys-color-on-secondary-container);"
              >
                <Icon name="vinyl-record" weight="fill" size={15} />
              </span>
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.favoriteAlbums')}
              </h2>
              <span class="text-xs font-semibold tabular-nums opacity-65">{favAlbums.length}</span>
            </div>
          </div>

          <div class="velora-card-grid-7">
            {#each favAlbums as al (al.id)}
              {@const cover = resolveCoverUrl(al.coverUrl, mediaBaseUrl, 256)}
              <div class="group w-full min-w-0 flex flex-col p-1.5 rounded-xl relative transition-colors hover:bg-[var(--md-sys-color-state-hover)]">
                <div class="velora-card-artwork mb-1.5 relative flex items-center justify-center shadow-sm" style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));">
                  <button
                    type="button"
                    onclick={() => onOpenAlbum(al.id)}
                    aria-label={al.title}
                    class="velora-card-artwork-hit cursor-pointer"
                  >
                    <CoverImage src={cover} fallbackIcon="vinyl-record" iconSize={34} isVideoThumb={isYTVideoThumb(cover)} />
                  </button>
                  <button
                    type="button"
                    onclick={() => (onPlayAlbum ? onPlayAlbum(al.id) : onOpenAlbum(al.id))}
                    aria-label="Play {al.title}"
                    class="absolute bottom-1.5 right-1.5 w-8 h-8 rounded-full shadow-md flex items-center justify-center cursor-pointer"
                    style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
                  >
                    <Icon name="play" weight="fill" size={14} />
                  </button>
                  <button
                    type="button"
                    onclick={() => onToggleFavoriteAlbum(al.id)}
                    aria-label="Unfavorite {al.title}"
                    class="absolute top-1.5 right-1.5 w-7 h-7 rounded-full backdrop-blur-xs flex items-center justify-center cursor-pointer"
                    style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-highest) 80%, transparent); color: var(--md-sys-color-tertiary);"
                  >
                    <Icon name="heart" weight="fill" size={13} />
                  </button>
                </div>
                <button type="button" onclick={() => onOpenAlbum(al.id)} class="w-full text-left cursor-pointer">
                  <div class="text-[11px] font-bold truncate" style="color: var(--md-sys-color-on-surface);">{al.title}</div>
                  <div class="text-[10px] truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">{al.artistName}</div>
                </button>
              </div>
            {/each}
          </div>
        </section>
      {/if}

      {#if (activeFilter === 'all' || activeFilter === 'artists') && favArtists.length > 0}
        <section class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span
                class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0"
                style="background-color: var(--md-sys-color-tertiary-container); color: var(--md-sys-color-on-tertiary-container);"
              >
                <Icon name="user" weight="fill" size={15} />
              </span>
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.favoriteArtists')}
              </h2>
              <span class="text-xs font-semibold tabular-nums opacity-65">{favArtists.length}</span>
            </div>
          </div>

          <div class="velora-card-grid-7">
            {#each favArtists as ar (ar.id)}
              {@const avatar = resolveCoverUrl(ar.avatarUrl, mediaBaseUrl, 256)}
              <div class="group w-full min-w-0 flex flex-col p-1.5 rounded-xl relative transition-colors hover:bg-[var(--md-sys-color-state-hover)]">
                <div class="relative mb-1.5">
                  <div
                    class="velora-card-artwork velora-artist-avatar !rounded-full flex items-center justify-center shadow-sm"
                    style="background: linear-gradient(135deg, var(--md-sys-color-primary-container), var(--md-sys-color-surface-container-highest));"
                  >
                    <button
                      type="button"
                      onclick={() => onOpenArtist(ar.id)}
                      aria-label={ar.name}
                      class="velora-card-artwork-hit cursor-pointer"
                    >
                      <CoverImage src={avatar} fallbackIcon="user" iconSize={34} />
                    </button>
                  </div>
                  <button
                    type="button"
                    onclick={() => onToggleFavoriteArtist(ar.id)}
                    aria-label="Unfavorite {ar.name}"
                    class="absolute bottom-0.5 right-0.5 w-7 h-7 rounded-full shadow-sm flex items-center justify-center cursor-pointer"
                    style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-highest) 88%, transparent); color: var(--md-sys-color-tertiary);"
                  >
                    <Icon name="heart" weight="fill" size={13} />
                  </button>
                </div>
                <button type="button" onclick={() => onOpenArtist(ar.id)} class="w-full text-center cursor-pointer">
                  <div class="text-[11px] font-bold truncate" style="color: var(--md-sys-color-on-surface);">{ar.name}</div>
                </button>
              </div>
            {/each}
          </div>
        </section>
      {/if}
    {/if}
  </div>
</div>
