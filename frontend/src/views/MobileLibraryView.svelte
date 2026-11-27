<script lang="ts">
  import { groupCompletedDownloads, isYTVideoThumb, resolveCoverUrl } from '../lib/api';
  import { tr, type LangCode } from '../lib/i18n';
  import type { Album, Artist, GroupedCompletedDownload, Playlist, Track, ViewRoute } from '../lib/types';
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
    downloadProgressPct?: number;
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
    onRemoveDownloadGroup?: (tracks: Track[]) => void;
    onOpenGroup?: (entry: GroupedCompletedDownload) => void;
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
    downloadProgressPct = 0,
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
    onRemoveDownloadGroup,
    onOpenGroup,
    onOpenAlbum,
    onPlayAlbum,
    onOpenArtist,
    onOpenTrackAlbum,
    onCopyLink,
    onNavigate
  }: Props = $props();

  let activeFilter = $state<LibraryFilter>('all');
  let openMenuKey = $state<string | null>(null);

  $effect(() => {
    activeFilter = initialFilter || 'all';
  });

  const likedTracks = $derived(tracks.filter((t) => t.isFavorite));
  const downloadedTracks = $derived(tracks.filter((t) => t.isDownloaded));
  const groupedDownloaded = $derived(groupCompletedDownloads(downloadedTracks, [], []));
  const favAlbums = $derived(albums.filter((a) => a.isFavorite));
  const favArtists = $derived(artists.filter((a) => a.isFavorite));

  const filterPills = $derived([
    { id: 'all' as const, label: tr(lang, 'nav.all') },
    { id: 'music' as const, label: tr(lang, 'nav.favoriteMusic') },
    { id: 'albums' as const, label: tr(lang, 'nav.favoriteAlbums') },
    { id: 'artists' as const, label: tr(lang, 'nav.favoriteArtists') },
    { id: 'downloaded' as const, label: tr(lang, 'nav.downloaded') }
  ]);

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

  const totalMatches = $derived(
    likedTracks.length + downloadedTracks.length + favAlbums.length + favArtists.length
  );

  function isTrackActive(trItem: Track): boolean {
    if (currentTrackId !== null && trItem.id === currentTrackId) return true;
    if (currentExternalId && trItem.externalId && trItem.externalId === currentExternalId) return true;
    return false;
  }

  function resolveAlbumCover(al: Album): string {
    const raw =
      al.coverUrl ||
      tracks.find((t) => al.id > 0 && t.albumId === al.id && Boolean(t.coverUrl))?.coverUrl ||
      tracks.find((t) => al.title && t.albumTitle === al.title && Boolean(t.coverUrl))?.coverUrl ||
      tracks.find((t) => al.artistName && t.artistName === al.artistName && Boolean(t.coverUrl))?.coverUrl ||
      '';
    return resolveCoverUrl(raw, mediaBaseUrl, 256);
  }

  function resolveArtistAvatar(ar: Artist): string {
    const raw =
      ar.avatarUrl ||
      tracks.find((t) => ar.id > 0 && t.artistId === ar.id && Boolean(t.coverUrl))?.coverUrl ||
      tracks.find((t) => ar.name && t.artistName === ar.name && Boolean(t.coverUrl))?.coverUrl ||
      albums.find((a) => ar.id > 0 && a.artistId === ar.id && Boolean(a.coverUrl))?.coverUrl ||
      '';
    return resolveCoverUrl(raw, mediaBaseUrl, 256);
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
    {downloadProgressPct}
    {onNavigate}
  />

  <div class="px-4 pt-2 pb-2 sticky top-0 z-20 backdrop-blur-md" style="background-color: color-mix(in srgb, var(--md-sys-color-surface-container-low) 88%, transparent);">
    <div
      use:shelfEdgeFade
      class="flex flex-nowrap items-center gap-2 overflow-x-auto no-scrollbar velora-shelf-scroll pb-0.5"
    >
      {#each filterPills as pill (pill.id)}
        {@const active = activeFilter === pill.id}
        <button
          type="button"
          onclick={(e) => {
            (e.currentTarget as HTMLElement).scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });
            activeFilter = pill.id;
          }}
          class="velora-press shrink-0 whitespace-nowrap px-3.5 py-1.5 rounded-full text-xs font-semibold flex items-center cursor-pointer transition-colors duration-150"
          style={active
            ? 'background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);'
            : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);'}
        >
          <span>{pill.label}</span>
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
            <div class="min-w-0">
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.favoriteMusic')}
              </h2>
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
                class="velora-press min-h-14 px-2 py-1.5 rounded-xl flex items-center gap-3 transition-colors {active
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

      {#if (activeFilter === 'all' || activeFilter === 'downloaded') && groupedDownloaded.length > 0}
        {@const visibleDl = activeFilter === 'all' ? groupedDownloaded.slice(0, 6) : groupedDownloaded}
        <section class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <div class="min-w-0">
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.downloaded')}
              </h2>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              {#if activeFilter === 'all' && groupedDownloaded.length > 6}
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
            {#each visibleDl as entry (entry.key)}
              {@const item = entry.primaryTrack}
              {@const cover = resolveCoverUrl(entry.coverUrl, mediaBaseUrl, 96)}
              {#if entry.isGroup}
                {@const active = entry.tracks.some((t) => isTrackActive(t))}
                {@const fallbackIcon = entry.kind === 'playlist' ? 'playlist' : 'vinyl-record'}
                <div
                  class="velora-press min-h-14 px-2 py-1.5 rounded-xl flex items-center gap-3 transition-colors {active
                    ? 'bg-[var(--md-sys-color-state-active)]'
                    : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                >
                  <button
                    type="button"
                    onclick={() => (onOpenGroup ? onOpenGroup(entry) : onPlayTrack(item, entry.tracks))}
                    class="min-w-0 flex-1 flex items-center gap-3 text-left cursor-pointer"
                  >
                    <div class="w-12 h-12 rounded-lg overflow-hidden shrink-0 relative bg-[var(--md-sys-color-surface-container-high)]">
                      <CoverImage src={cover} {fallbackIcon} iconSize={20} isVideoThumb={isYTVideoThumb(cover)} />
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
                        {entry.title}
                      </div>
                      <div class="flex items-center gap-1.5 text-xs truncate mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                        <Icon name="check-circle" weight="fill" size={13} class="text-[var(--md-sys-color-primary)] shrink-0" />
                        <span class="truncate">
                          {entry.kind === 'playlist' ? tr(lang, 'nav.playlist') : tr(lang, 'nav.album')} · {entry.tracks.length} {tr(lang, 'nav.songsCount')}
                        </span>
                      </div>
                    </div>
                  </button>

                  <div class="flex items-center gap-0.5 shrink-0">
                    <button
                      type="button"
                      onclick={() => (active && onTogglePlay ? onTogglePlay() : onPlayTrack(item, entry.tracks))}
                      aria-label={tr(lang, 'nav.play')}
                      class="w-9 h-9 rounded-full flex items-center justify-center cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
                      style="color: var(--md-sys-color-on-surface-variant);"
                    >
                      <Icon name={active && isPlaying ? 'pause' : 'play'} weight="fill" size={15} />
                    </button>
                    {#if onRemoveDownloadGroup}
                      <button
                        type="button"
                        onclick={() => onRemoveDownloadGroup(entry.tracks)}
                        title={tr(lang, 'nav.removeDownload')}
                        aria-label={tr(lang, 'nav.removeDownload')}
                        class="w-9 h-9 rounded-full flex items-center justify-center cursor-pointer hover:bg-[var(--md-sys-color-state-hover)]"
                        style="color: var(--md-sys-color-on-surface-variant);"
                      >
                        <Icon name="trash" size={15} />
                      </button>
                    {/if}
                  </div>
                </div>
              {:else}
                {@const active = isTrackActive(item)}
                {@const dKey = `lib-dl-${item.id}`}
                <div
                  class="velora-press min-h-14 px-2 py-1.5 rounded-xl flex items-center gap-3 transition-colors {active
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
              {/if}
            {/each}
          </div>
        </section>
      {/if}

      {#if (activeFilter === 'all' || activeFilter === 'albums') && favAlbums.length > 0}
        <section class="space-y-2.5">
          <div class="flex items-center justify-between gap-2">
            <div class="min-w-0">
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.favoriteAlbums')}
              </h2>
            </div>
          </div>

          <div class="velora-card-grid-7">
            {#each favAlbums as al (al.id)}
              {@const cover = resolveAlbumCover(al)}
              <div class="velora-press group w-full min-w-0 flex flex-col p-1.5 rounded-xl relative transition-colors hover:bg-[var(--md-sys-color-state-hover)]">
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
            <div class="min-w-0">
              <h2 class="text-lg font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
                {tr(lang, 'nav.favoriteArtists')}
              </h2>
            </div>
          </div>

          <div class="velora-card-grid-7">
            {#each favArtists as ar (ar.id)}
              {@const avatar = resolveArtistAvatar(ar)}
              <div class="velora-press group w-full min-w-0 flex flex-col p-1.5 rounded-xl relative transition-colors hover:bg-[var(--md-sys-color-state-hover)]">
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
