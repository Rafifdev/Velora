<script lang="ts">
  import { tr, type LangCode } from '../lib/i18n';
  import type { Playlist, Track } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    track: Track;
    open: boolean;
    playlists?: Playlist[];
    excludePlaylistId?: number | null;
    lang?: LangCode;
    triggerClass?: string;
    onToggleOpen: () => void;
    onClose: () => void;
    onPlayNext?: (track: Track) => void;
    onAddToQueue?: (track: Track) => void;
    onAddToPlaylist?: (playlistId: number, trackId: number) => void;
    onRemoveFromPlaylist?: (trackId: number) => void;
    onToggleFavorite?: (trackId: number) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist?: (artistId: number) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onCopyLink?: (track: Track) => void;
  }

  let {
    track,
    open,
    playlists = [],
    excludePlaylistId = null,
    lang = 'en',
    triggerClass = '',
    onToggleOpen,
    onClose,
    onPlayNext,
    onAddToQueue,
    onAddToPlaylist,
    onRemoveFromPlaylist,
    onToggleFavorite,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenTrackAlbum,
    onCopyLink
  }: Props = $props();

  let showPlaylistSubmenu = $state(false);

  const availablePlaylists = $derived(
    excludePlaylistId != null ? playlists.filter((p) => p.id !== excludePlaylistId) : playlists
  );

  const hasBottomSection = $derived(
    Boolean((onOpenArtist && track.artistId > 0) || onOpenTrackAlbum || true)
  );

  $effect(() => {
    if (!open) {
      showPlaylistSubmenu = false;
      return;
    }
    const handleDocClick = () => {
      onClose();
    };
    window.addEventListener('click', handleDocClick);
    return () => {
      window.removeEventListener('click', handleDocClick);
    };
  });

  function copyTrackLink() {
    if (onCopyLink) {
      onCopyLink(track);
      onClose();
      return;
    }
    const link = track.externalId
      ? `https://music.youtube.com/watch?v=${track.externalId}`
      : `${track.title}${track.artistName ? ` — ${track.artistName}` : ''}`;
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(link);
    }
    onClose();
  }
</script>

<div class="relative flex items-center justify-end">
  <button
    type="button"
    onclick={(e) => {
      e.stopPropagation();
      onToggleOpen();
    }}
    aria-label={tr(lang, 'nav.trackActions')}
    class="w-7 h-7 flex items-center justify-center bg-transparent hover:bg-transparent cursor-pointer transition-colors {triggerClass} {open
      ? 'opacity-100 text-[var(--md-sys-color-on-surface)]'
      : 'text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)]'}"
  >
    <Icon name="dots-three-vertical" weight="bold" size={17} />
  </button>

  {#if open}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      onclick={(e) => e.stopPropagation()}
      role="menu"
      tabindex="-1"
      class="absolute right-0 top-9 z-40 w-60 p-1.5 rounded-xl shadow-2xl text-xs"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
    >
      {#if onPlayNext}
        <button
          type="button"
          onclick={() => {
            onPlayNext(track);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="skip-forward" size={15} />
          <span>{tr(lang, 'nav.playNext')}</span>
        </button>
      {/if}

      {#if onAddToQueue}
        <button
          type="button"
          onclick={() => {
            onAddToQueue(track);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="queue" size={15} />
          <span>{tr(lang, 'nav.addToQueue')}</span>
        </button>
      {/if}

      {#if onAddToPlaylist && availablePlaylists.length > 0}
        <div class="relative">
          <button
            type="button"
            onclick={() => (showPlaylistSubmenu = !showPlaylistSubmenu)}
            class="w-full flex items-center justify-between gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
          >
            <span class="flex items-center gap-2.5">
              <Icon name="list-plus" size={15} />
              <span>{tr(lang, 'nav.addToPlaylist')}</span>
            </span>
            <Icon name="caret-right" size={12} />
          </button>
          {#if showPlaylistSubmenu}
            <div class="mt-0.5 mb-1 pl-3 space-y-0.5 max-h-36 overflow-y-auto">
              {#each availablePlaylists as otherPl (otherPl.id)}
                <button
                  type="button"
                  onclick={() => {
                    onAddToPlaylist(otherPl.id, track.id);
                    onClose();
                  }}
                  class="w-full text-left px-3 py-1.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] truncate cursor-pointer opacity-90"
                >
                  {otherPl.name} {otherPl.isLocal ? `(${tr(lang, 'nav.localSource')})` : ''}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      {#if onRemoveFromPlaylist}
        <button
          type="button"
          onclick={() => {
            onRemoveFromPlaylist(track.id);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="minus-circle" size={15} />
          <span>{tr(lang, 'nav.removeFromThisPlaylist')}</span>
        </button>
      {/if}

      {#if onToggleFavorite}
        <button
          type="button"
          onclick={() => {
            onToggleFavorite(track.id);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon
            name="heart"
            weight={track.isFavorite ? 'fill' : 'regular'}
            size={15}
            class={track.isFavorite ? 'text-[var(--md-sys-color-tertiary)]' : ''}
          />
          <span>{track.isFavorite ? tr(lang, 'nav.removeFromLikedSongs') : tr(lang, 'nav.addToLikedSongs')}</span>
        </button>
      {/if}

      {#if track.isDownloaded && onRemoveDownload}
        <button
          type="button"
          onclick={() => {
            onRemoveDownload(track.id);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="check-circle" weight="fill" size={15} class="text-[var(--md-sys-color-primary)]" />
          <span>{tr(lang, 'nav.removeDownload')}</span>
        </button>
      {:else if onDownloadTrack}
        <button
          type="button"
          onclick={() => {
            onDownloadTrack(track);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="download" size={15} />
          <span>{tr(lang, 'nav.downloadOffline')}</span>
        </button>
      {/if}

      {#if hasBottomSection}
        <div class="my-1 h-px opacity-15" style="background-color: var(--md-sys-color-on-surface);"></div>
      {/if}

      {#if onOpenArtist && track.artistId > 0}
        <button
          type="button"
          onclick={() => {
            onOpenArtist(track.artistId);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="user" size={15} />
          <span>{tr(lang, 'nav.openArtist')}</span>
        </button>
      {/if}

      {#if onOpenTrackAlbum}
        <button
          type="button"
          onclick={() => {
            onOpenTrackAlbum(track);
            onClose();
          }}
          class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="vinyl-record" size={15} />
          <span>{tr(lang, 'nav.goToAlbum')}</span>
        </button>
      {/if}

      <button
        type="button"
        onclick={copyTrackLink}
        class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
      >
        <Icon name="link" size={15} />
        <span>{tr(lang, 'nav.copyLink')}</span>
      </button>
    </div>
  {/if}
</div>
