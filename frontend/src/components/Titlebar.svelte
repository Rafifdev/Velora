<script lang="ts">
  import { api } from '../lib/api';
  import type { FireStreakStatus, ViewRoute } from '../lib/types';
  import Icon from './Icon.svelte';
  import VeloraLogo from './VeloraLogo.svelte';

  interface Props {
    currentRoute: ViewRoute;
    canGoBack: boolean;
    canGoForward: boolean;
    searchQuery: string;
    fireStreak: FireStreakStatus;
    isDark: boolean;
    accentColor: string;
    onBack: () => void;
    onForward: () => void;
    onNavigate: (route: ViewRoute) => void;
    onSearch: (query: string) => void;
    onToggleTheme: () => void;
    onOpenStreakModal: () => void;
  }

  let {
    currentRoute,
    canGoBack,
    canGoForward,
    searchQuery,
    fireStreak,
    isDark,
    onBack,
    onForward,
    onNavigate,
    onSearch,
    onToggleTheme,
    onOpenStreakModal
  }: Props = $props();

  let localQuery = $state('');
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let isMaximized = $state(false);

  async function syncMaximizedState() {
    isMaximized = await api.windowIsMaximized();
  }

  async function handleToggleMaximize() {
    isMaximized = !isMaximized;
    api.windowMaximize();
    setTimeout(syncMaximizedState, 120);
  }

  $effect(() => {
    localQuery = searchQuery;
  });

  $effect(() => {
    void syncMaximizedState();
    window.addEventListener('resize', syncMaximizedState);
    return () => {
      window.removeEventListener('resize', syncMaximizedState);
    };
  });

  function handleInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    localQuery = val;
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      onSearch(val);
    }, 250);
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      if (debounceTimer) clearTimeout(debounceTimer);
      onSearch(localQuery);
    }
  }

  function clearSearch() {
    if (debounceTimer) clearTimeout(debounceTimer);
    localQuery = '';
    onSearch('');
  }
</script>

<!-- Borderless Spotify Top App Bar -->
<header
  class="wails-drag h-16 pl-3 pr-0 flex items-center justify-between gap-4 select-none shrink-0 transition-colors duration-300"
  style="background-color: var(--md-sys-color-surface-container-lowest);"
>
  <!-- Left: Navigation History Circular Icon Buttons -->
  <div class="flex items-center gap-3 shrink-0">
    <div class="wails-no-drag flex items-center gap-1">
      <button
        type="button"
        onclick={onBack}
        disabled={!canGoBack}
        aria-label="Kembali"
        class="m3-icon-btn w-10 h-10 rounded-full flex items-center justify-center disabled:opacity-35 disabled:pointer-events-none cursor-pointer"
      >
        <Icon name="caret-left" weight="bold" size={18} />
      </button>
      <button
        type="button"
        onclick={onForward}
        disabled={!canGoForward}
        aria-label="Maju"
        class="m3-icon-btn w-10 h-10 rounded-full flex items-center justify-center disabled:opacity-35 disabled:pointer-events-none cursor-pointer"
      >
        <Icon name="caret-right" weight="bold" size={18} />
      </button>
    </div>
  </div>

  <!-- Center: 1:1 Spotify Home Circle Button + Docked Pill Search Bar -->
  <div class="wails-no-drag flex-1 max-w-xl mx-auto flex items-center gap-2">
    <button
      type="button"
      onclick={() => onNavigate({ name: 'home' })}
      title="Home"
      aria-label="Home"
      class="w-12 h-12 rounded-full flex items-center justify-center shrink-0 cursor-pointer transition-all duration-200 hover:scale-105 hover:brightness-125"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
    >
      <Icon name="house" weight={currentRoute.name === 'home' ? 'fill' : 'regular'} size={22} />
    </button>

    <div
      class="flex-1 h-12 rounded-full px-3.5 flex items-center gap-3 transition-colors duration-200 has-[input:hover]:brightness-110"
      style="background-color: var(--md-sys-color-surface-container-high);"
    >
      <Icon
        name="magnifying-glass"
        weight="regular"
        size={22}
        class="shrink-0"
        style="color: var(--md-sys-color-on-surface-variant);"
      />
      <input
        type="search"
        value={localQuery}
        oninput={handleInput}
        onkeydown={handleKeyDown}
        placeholder="Apa yang ingin kamu putar?"
        class="w-full h-full bg-transparent text-[14px] font-normal outline-none border-none"
        style="color: var(--md-sys-color-on-surface);"
      />
      {#if localQuery.length > 0}
        <button
          type="button"
          onclick={clearSearch}
          title="Hapus pencarian"
          aria-label="Hapus pencarian"
          class="shrink-0 flex items-center justify-center cursor-pointer transition-opacity hover:opacity-80"
          style="color: #ffffff;"
        >
          <Icon name="x" weight="regular" size={22} />
        </button>
      {/if}
      <span class="w-[1px] h-6 shrink-0" style="background-color: rgba(114, 114, 114, 0.55);"></span>
      <button
        type="button"
        onclick={onOpenStreakModal}
        title="Fire Streak Hari Ini ({fireStreak.currentStreak} Hari)"
        aria-label="Fire Streak {fireStreak.currentStreak}"
        class="h-8 px-2.5 rounded-full flex items-center gap-1.5 shrink-0 text-[13px] font-bold cursor-pointer transition-all duration-200 hover:brightness-125"
        style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
      >
        <Icon name="flame" weight="fill" size={18} style="color: var(--md-sys-color-tertiary);" />
        <span class="tabular-nums">{fireStreak.currentStreak}</span>
      </button>
    </div>
  </div>

  <!-- Right: Taller Unified Controls (44px Action Circles) + Native Windows Caption Controls -->
  <div class="wails-no-drag flex items-center gap-1.5 shrink-0 self-stretch">
    <button
      type="button"
      onclick={() => onNavigate({ name: 'preferences' })}
      title="Nuclear Jam Remote"
      aria-label="Remote QR & Integrations"
      class="w-11 h-11 rounded-full flex items-center justify-center cursor-pointer transition-all duration-200 hover:brightness-125"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);"
    >
      <Icon name="qr-code" size={19} />
    </button>

    <button
      type="button"
      onclick={onToggleTheme}
      title={isDark ? 'Switch to Material Light' : 'Switch to Material Dark'}
      aria-label="Toggle Theme"
      class="w-11 h-11 rounded-full flex items-center justify-center cursor-pointer transition-all duration-200 hover:brightness-125"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);"
    >
      <Icon name={isDark ? 'sun' : 'moon'} weight="bold" size={19} />
    </button>

    <div class="flex items-stretch self-stretch ml-2">
      <button
        type="button"
        onclick={() => api.windowMinimize()}
        title="Minimize"
        aria-label="Minimize"
        class="win-caption-btn"
      >
        <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
          <line x1="0" y1="5.5" x2="10" y2="5.5" stroke="currentColor" stroke-width="1" />
        </svg>
      </button>
      <button
        type="button"
        onclick={handleToggleMaximize}
        title={isMaximized ? 'Restore Down' : 'Maximize'}
        aria-label={isMaximized ? 'Restore Down' : 'Maximize'}
        class="win-caption-btn"
      >
        {#if isMaximized}
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
            <rect x="0.5" y="2.5" width="7" height="7" stroke="currentColor" stroke-width="1" />
            <path d="M2.5 2.5V0.5H9.5V7.5H7.5" stroke="currentColor" stroke-width="1" />
          </svg>
        {:else}
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
            <rect x="0.5" y="0.5" width="9" height="9" stroke="currentColor" stroke-width="1" />
          </svg>
        {/if}
      </button>
      <button
        type="button"
        onclick={() => api.windowClose()}
        title="Close"
        aria-label="Close"
        class="win-caption-btn win-caption-btn-close"
      >
        <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
          <path d="M1 1L9 9M9 1L1 9" stroke="currentColor" stroke-width="1.15" stroke-linecap="butt" />
        </svg>
      </button>
    </div>
  </div>
</header>
