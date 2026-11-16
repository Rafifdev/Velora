<script lang="ts">
  import { animate } from 'motion';
  import type { ViewRoute } from '../lib/types';
  import Icon from './Icon.svelte';
  import PreferencesSidebar from './PreferencesSidebar.svelte';

  interface Props {
    currentRoute: ViewRoute;
    width: number;
    collapsed: boolean;
    accentColor: string;
    favoriteAlbumsCount?: number;
    favoriteTracksCount?: number;
    favoriteArtistsCount?: number;
    playlistsCount?: number;
    sourcesCount?: number;
    isPreferencesMode?: boolean;
    activePrefTab?: string;
    prefSearch?: string;
    prefSavedToast?: string;
    onBackFromPreferences?: () => void;
    onClearCache?: () => void;
    onNavigate: (route: ViewRoute) => void;
    onToggleCollapse: () => void;
    onResizeWidth: (newWidth: number) => void;
    onResizeEnd: (finalWidth: number) => void;
  }

  let {
    currentRoute,
    width,
    collapsed,
    favoriteAlbumsCount = 0,
    favoriteTracksCount = 0,
    favoriteArtistsCount = 0,
    playlistsCount = 0,
    sourcesCount = 0,
    isPreferencesMode = false,
    activePrefTab = $bindable('appearance'),
    prefSearch = $bindable(''),
    prefSavedToast = '',
    onBackFromPreferences,
    onClearCache,
    onNavigate,
    onToggleCollapse,
    onResizeWidth,
    onResizeEnd
  }: Props = $props();

  let collapseIconEl: HTMLElement | undefined = $state();
  let prevCollapsed: boolean | undefined;
  let flipAnim: { stop?: () => void } | undefined;
  let isResizing = $state(false);
  const prefActive = $derived(currentRoute.name === 'preferences');

  const MIN_WIDTH = 212;
  const MAX_WIDTH = 380;
  const COLLAPSED_WIDTH = 64;

  type MenuEntry = { id: ViewRoute['name']; label: string; icon: string; count?: number };

  const flatMenuItems = $derived<MenuEntry[]>([
    { id: 'home', label: 'Home', icon: 'house' },
    { id: 'favorite-albums', label: 'Favorite albums', icon: 'vinyl-record' },
    { id: 'liked-songs', label: 'Favorite music', icon: 'heart' },
    { id: 'favorite-artists', label: 'Favorite artists', icon: 'users' },
    { id: 'playlists', label: 'Playlists', icon: 'playlist', count: playlistsCount },
    { id: 'history', label: 'Listening history', icon: 'clock-counter-clockwise' },
    { id: 'sources', label: 'Sources', icon: 'plugs', count: sourcesCount }
  ]);

  function isActive(name: string): boolean {
    if (name === 'home') return currentRoute.name === 'home';
    if (name === 'favorite-albums') {
      return currentRoute.name === 'favorite-albums' || currentRoute.name === 'album-detail';
    }
    if (name === 'favorite-artists') {
      return currentRoute.name === 'favorite-artists' || currentRoute.name === 'artist-detail';
    }
    if (name === 'playlists') {
      return (
        currentRoute.name === 'playlists' ||
        currentRoute.name === 'local-playlists' ||
        currentRoute.name === 'playlist-detail'
      );
    }
    if (name === 'history') {
      return currentRoute.name === 'history' || currentRoute.name === 'listening-stats';
    }
    return currentRoute.name === name;
  }

  $effect(() => {
    const nextCollapsed = collapsed;
    const targetEl = collapseIconEl;
    if (prevCollapsed === undefined) {
      prevCollapsed = nextCollapsed;
      return;
    }
    if (prevCollapsed !== nextCollapsed && targetEl) {
      prevCollapsed = nextCollapsed;
      flipAnim?.stop?.();
      const reducedMotion =
        typeof window !== 'undefined' &&
        window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      if (!reducedMotion) {
        flipAnim = animate(
          targetEl,
          { scaleX: nextCollapsed ? [1, -1] : [-1, 1] },
          { duration: 0.34, ease: [0.16, 1, 0.3, 1] }
        );
      }
    }
  });

  function startResize(e: PointerEvent) {
    if (collapsed) return;
    e.preventDefault();
    isResizing = true;
    const startX = e.clientX;
    const startW = width;

    function onMove(ev: PointerEvent) {
      const next = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, startW + (ev.clientX - startX)));
      onResizeWidth(next);
    }

    function onUp(ev: PointerEvent) {
      isResizing = false;
      const finalW = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, startW + (ev.clientX - startX)));
      onResizeEnd(finalW);
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
    }

    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  }
</script>

<!-- Borderless Navigation Drawer / Morphing Icon Rail -->
<aside
  aria-label="Left Menu Section"
  class="relative shrink-0 h-full min-h-0 flex flex-col select-none overflow-visible rounded-lg"
  style="width: {isPreferencesMode ? width : collapsed ? COLLAPSED_WIDTH : width}px; background-color: var(--md-sys-color-surface-container-low); transition: {isResizing ? 'none' : 'width 380ms cubic-bezier(0.16, 1, 0.3, 1), background-color 300ms ease'};"
>
  {#if isPreferencesMode}
    <div class="w-full h-full min-h-0 flex flex-col overflow-hidden rounded-lg">
      <PreferencesSidebar
        bind:activeTab={activePrefTab}
        bind:search={prefSearch}
        savedToast={prefSavedToast}
        onBack={() => onBackFromPreferences?.()}
      />
    </div>
  {:else}
  <div class="w-full h-full min-h-0 flex flex-col overflow-hidden rounded-lg p-2.5">
    <!-- Persistent Top Row: constant justify-between + px-1 so toggle button glides smoothly without layout jump -->
    <div class="flex items-center justify-between px-1 pb-2 w-full shrink-0">
      <span
        class="text-[11px] font-semibold uppercase tracking-wider truncate overflow-hidden transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] {collapsed
          ? 'max-w-0 opacity-0 -translate-x-2 pointer-events-none pl-0'
          : 'max-w-[140px] opacity-100 translate-x-0 pl-2'}"
        style="color: var(--md-sys-color-on-surface-variant);"
      >
        Library
      </span>
      <button
        type="button"
        onclick={onToggleCollapse}
        title={collapsed ? 'Expand Menu' : 'Collapse Menu'}
        aria-label={collapsed ? 'Expand Left Menu' : 'Collapse Menu'}
        class="m3-icon-btn w-9 h-9 shrink-0 flex items-center justify-center cursor-pointer transition-transform duration-200 active:scale-90"
      >
        <span
          bind:this={collapseIconEl}
          class="inline-flex items-center justify-center"
          style="transform: scaleX({collapsed ? -1 : 1});"
        >
          <Icon name="sidebar-simple" size={18} />
        </span>
      </button>
    </div>

    <div class="flex-1 min-h-0 flex flex-col justify-between w-full">
      <!-- Unified Morphing Menu List (Fixed icon X-anchor so icons never jitter) -->
      <nav class="flex-1 min-h-0 overflow-y-auto overscroll-contain flex flex-col gap-1 pt-1" aria-label="Menu Utama">
        {#each flatMenuItems as item (item.id)}
          {@const active = isActive(item.id)}
          <button
            type="button"
            onclick={() => onNavigate({ name: item.id } as ViewRoute)}
            title={collapsed ? item.label : undefined}
            aria-label={item.label}
            class="group relative w-full h-10 flex items-center justify-start px-3 rounded-md text-[13px] transition-colors duration-200 cursor-pointer text-left {active
              ? 'font-semibold'
              : 'font-medium hover:bg-[var(--md-sys-color-state-hover)]'}"
            style={active
              ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
              : 'color: var(--md-sys-color-on-surface-variant);'}
          >
            <Icon
              name={item.icon}
              weight={active ? 'fill' : 'regular'}
              size={19}
              class="shrink-0"
            />
            <span
              class="truncate overflow-hidden whitespace-nowrap transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] {collapsed
                ? 'max-w-0 opacity-0 -translate-x-2 ml-0'
                : 'max-w-[150px] opacity-100 translate-x-0 ml-2.5'}"
            >
              {item.label}
            </span>
          </button>
        {/each}
      </nav>

      <!-- Bottom Pinned Preferences Item -->
      <div class="pt-3 shrink-0">
        <button
          type="button"
          onclick={() => onNavigate({ name: 'preferences' })}
          title={collapsed ? 'Preferences' : undefined}
          aria-label="Preferences"
          class="group relative w-full h-10 flex items-center justify-start px-3 rounded-md text-[13px] transition-colors duration-200 cursor-pointer text-left {prefActive
            ? 'font-semibold'
            : 'font-medium hover:bg-[var(--md-sys-color-state-hover)]'}"
          style={prefActive
            ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
            : 'color: var(--md-sys-color-on-surface-variant);'}
        >
          <Icon
            name="gear"
            weight={prefActive ? 'fill' : 'regular'}
            size={19}
            class="shrink-0"
          />
          <span
            class="truncate overflow-hidden whitespace-nowrap transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] {collapsed
              ? 'max-w-0 opacity-0 -translate-x-2 ml-0'
              : 'max-w-[150px] opacity-100 translate-x-0 ml-2.5'}"
          >
            Preferences
          </span>
        </button>
      </div>
    </div>
  </div>
  {/if}

  {#if !collapsed || isPreferencesMode}
    <!-- Spotify Centered Gap Resize Divider -->
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize left menu"
      onpointerdown={startResize}
      class="group absolute top-0 -right-2 w-2 h-full cursor-col-resize z-30 flex items-center justify-center"
    >
      <div
        class="w-[2px] h-[calc(100%-12px)] rounded-full transition-colors duration-150 {isResizing
          ? 'bg-white'
          : 'bg-transparent group-hover:bg-[#727272]'}"
      ></div>
    </div>
  {/if}
</aside>
