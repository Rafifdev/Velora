<script lang="ts">
  import { tick } from 'svelte';
  import { cubicIn, cubicOut } from 'svelte/easing';
  import { fade, scale, slide } from 'svelte/transition';
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
    onAddToPlaylist?: (playlistId: number, trackId: number, track?: Track) => void;
    onCreatePlaylist?: (track?: Track) => void;
    onRemoveFromPlaylist?: (trackId: number) => void;
    onToggleFavorite?: (trackId: number, track?: Track) => void;
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
    onCreatePlaylist,
    onRemoveFromPlaylist,
    onToggleFavorite,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenTrackAlbum,
    onCopyLink
  }: Props = $props();

  let showPlaylistSubmenu = $state(false);
  let triggerEl: HTMLButtonElement | undefined = $state();
  let menuEl: HTMLDivElement | undefined = $state();
  let openUpward = $state(false);
  let menuPos = $state({
    top: 0,
    bottom: 0,
    right: 12,
    maxHeight: 360
  });

  function portal(node: HTMLElement) {
    if (typeof document !== 'undefined' && document.body && node.parentNode !== document.body) {
      document.body.appendChild(node);
    }
    return {
      destroy() {
        node.remove();
      }
    };
  }

  const estimatedMenuHeight = $derived.by(() => {
    let items = 1; // Copy link always shown
    if (onPlayNext) items++;
    if (onAddToQueue) items++;
    if ((onAddToPlaylist && availablePlaylists.length > 0) || onCreatePlaylist) items++;
    if (onRemoveFromPlaylist) items++;
    if (onToggleFavorite) items++;
    if ((track.isDownloaded && onRemoveDownload) || onDownloadTrack) items++;
    if (onOpenArtist && track.artistId > 0) items++;
    if (onOpenTrackAlbum) items++;
    const submenuCount = availablePlaylists.length + (onCreatePlaylist ? 1 : 0);
    const submenuExtra = showPlaylistSubmenu
      ? Math.min(submenuCount, 5) * 32 + 10
      : 0;
    return 24 + items * 34 + submenuExtra;
  });

  function getSafeBounds(): { top: number; bottom: number } {
    const vh = typeof window !== 'undefined' ? window.innerHeight : 800;
    const mainEl = triggerEl?.closest('main');
    if (mainEl) {
      const mr = mainEl.getBoundingClientRect();
      return {
        top: Math.max(8, mr.top + 6),
        bottom: Math.min(vh - 12, mr.bottom - 6)
      };
    }
    return { top: 8, bottom: vh - 88 };
  }

  function updateMenuPosition(redecideDirection = false) {
    if (!triggerEl || typeof window === 'undefined') return;
    const rect = triggerEl.getBoundingClientRect();
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    const bounds = getSafeBounds();
    const gap = 6;
    const menuWidth = 240;

    if (rect.bottom < bounds.top - 8 || rect.top > bounds.bottom + 8) {
      onClose();
      return;
    }

    const spaceBelow = Math.max(0, bounds.bottom - (rect.bottom + gap));
    const spaceAbove = Math.max(0, rect.top - gap - bounds.top);
    let flipUp = openUpward;
    if (redecideDirection) {
      flipUp = spaceBelow < Math.min(estimatedMenuHeight, 340) && spaceAbove > spaceBelow;
      openUpward = flipUp;
    }

    const avail = flipUp ? spaceAbove : spaceBelow;
    const maxHeight = Math.max(150, Math.floor(avail));
    const right = Math.max(8, Math.min(vw - menuWidth - 8, vw - rect.right));

    menuPos = {
      top: Math.max(bounds.top, Math.round(rect.bottom + gap)),
      bottom: Math.max(12, Math.round(vh - (rect.top - gap))),
      right: Math.round(right),
      maxHeight
    };
  }

  const availablePlaylists = $derived(
    excludePlaylistId != null ? playlists.filter((p) => p.id !== excludePlaylistId) : playlists
  );

  const hasBottomSection = $derived(
    Boolean((onOpenArtist && track.artistId > 0) || onOpenTrackAlbum || true)
  );

  $effect.pre(() => {
    if (open && triggerEl) {
      updateMenuPosition(true);
    }
  });

  $effect(() => {
    if (!open) {
      showPlaylistSubmenu = false;
      return;
    }
    void tick().then(() => {
      if (open) {
        updateMenuPosition(false);
      }
    });

    const handleOutsideClick = (e: MouseEvent) => {
      const target = e.target as Node | null;
      if (target && (menuEl?.contains(target) || triggerEl?.contains(target))) {
        return;
      }
      onClose();
    };
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    const handleScrollOrResize = (e: Event) => {
      const target = e.target as Node | null;
      if (target && menuEl?.contains(target)) {
        return;
      }
      updateMenuPosition();
    };

    window.addEventListener('click', handleOutsideClick, true);
    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('resize', handleScrollOrResize);
    window.addEventListener('scroll', handleScrollOrResize, true);
    return () => {
      window.removeEventListener('click', handleOutsideClick, true);
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('resize', handleScrollOrResize);
      window.removeEventListener('scroll', handleScrollOrResize, true);
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
    bind:this={triggerEl}
    type="button"
    onclick={(e) => {
      e.stopPropagation();
      if (!open) {
        updateMenuPosition(true);
      }
      onToggleOpen();
    }}
    aria-label={tr(lang, 'nav.trackActions')}
    aria-expanded={open}
    aria-haspopup="menu"
    class="w-7 h-7 flex items-center justify-center bg-transparent hover:bg-transparent cursor-pointer transition-colors {triggerClass} {open
      ? 'opacity-100 text-[var(--md-sys-color-on-surface)]'
      : 'text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)]'}"
  >
    <Icon name="dots-three-vertical" weight="bold" size={17} />
  </button>

  {#if open}
    <div use:portal class="fixed inset-0 z-[120] pointer-events-none">
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div
        bind:this={menuEl}
        in:scale={{ start: 0.94, opacity: 0, duration: 150, easing: cubicOut }}
        out:fade={{ duration: 95, easing: cubicIn }}
        onclick={(e) => e.stopPropagation()}
        role="menu"
        tabindex="-1"
        class="{open ? 'pointer-events-auto' : 'pointer-events-none'} fixed w-60 p-1.5 rounded-xl shadow-[0_14px_40px_rgba(0,0,0,0.48)] border border-[var(--md-sys-color-outline-variant)]/35 overflow-y-auto text-xs {openUpward
          ? 'origin-bottom-right'
          : 'origin-top-right'}"
        style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface); right: {menuPos.right}px; {openUpward
          ? `bottom: ${menuPos.bottom}px;`
          : `top: ${menuPos.top}px;`} max-height: {menuPos.maxHeight}px;"
      >
      {#if onPlayNext}
        <button
          type="button"
          onclick={() => {
            onPlayNext(track);
            onClose();
          }}
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
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
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="queue" size={15} />
          <span>{tr(lang, 'nav.addToQueue')}</span>
        </button>
      {/if}

      {#if (onAddToPlaylist && availablePlaylists.length > 0) || onCreatePlaylist}
        <div class="relative">
          <button
            type="button"
            onclick={() => (showPlaylistSubmenu = !showPlaylistSubmenu)}
            class="w-full min-h-10 sm:min-h-0 flex items-center justify-between gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] transition-colors duration-150 cursor-pointer text-left"
          >
            <span class="flex items-center gap-2.5">
              <Icon name="list-plus" size={15} />
              <span>{tr(lang, 'nav.addToPlaylist')}</span>
            </span>
            <Icon
              name="caret-right"
              size={12}
              class="transition-transform duration-150 {showPlaylistSubmenu ? 'rotate-90' : ''}"
            />
          </button>
          {#if showPlaylistSubmenu}
            <div transition:slide={{ duration: 140, easing: cubicOut }} class="mt-0.5 mb-1 pl-3 space-y-0.5 max-h-40 overflow-y-auto">
              {#if onCreatePlaylist}
                <button
                  type="button"
                  onclick={() => {
                    onCreatePlaylist(track);
                    onClose();
                  }}
                  class="w-full min-h-10 sm:min-h-0 flex items-center gap-2 text-left px-3 py-1.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] transition-colors duration-150 cursor-pointer font-semibold text-[var(--md-sys-color-primary)]"
                >
                  <Icon name="plus" size={13} />
                  <span class="truncate">{tr(lang, 'nav.createPlaylist')}</span>
                </button>
              {/if}
              {#if onAddToPlaylist}
                {#each availablePlaylists as otherPl (otherPl.id)}
                  <button
                    type="button"
                    onclick={() => {
                      onAddToPlaylist(otherPl.id, track.id, track);
                      onClose();
                    }}
                    class="w-full min-h-10 sm:min-h-0 text-left px-3 py-1.5 rounded-md hover:bg-[var(--md-sys-color-state-hover)] transition-colors duration-150 truncate cursor-pointer opacity-90"
                  >
                    {otherPl.name} {otherPl.isLocal ? `(${tr(lang, 'nav.localSource')})` : ''}
                  </button>
                {/each}
              {/if}
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
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="minus-circle" size={15} />
          <span>{tr(lang, 'nav.removeFromThisPlaylist')}</span>
        </button>
      {/if}

      {#if onToggleFavorite}
        <button
          type="button"
          onclick={() => {
            onToggleFavorite(track.id, track);
            onClose();
          }}
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
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
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
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
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
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
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
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
          class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] cursor-pointer text-left"
        >
          <Icon name="vinyl-record" size={15} />
          <span>{tr(lang, 'nav.goToAlbum')}</span>
        </button>
      {/if}

      <button
        type="button"
        onclick={copyTrackLink}
        class="w-full min-h-10 sm:min-h-0 flex items-center gap-2.5 px-3 py-2 rounded-lg hover:bg-[var(--md-sys-color-state-hover)] transition-colors duration-150 cursor-pointer text-left"
      >
        <Icon name="link" size={15} />
        <span>{tr(lang, 'nav.copyLink')}</span>
      </button>
      </div>
    </div>
  {/if}
</div>
