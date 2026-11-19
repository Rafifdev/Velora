<script lang="ts">
  import { fly } from 'svelte/transition';
  import { NUCLEAR_APP_TABS, NUCLEAR_CATEGORIES } from '../lib/nuclearPrefs';
  import Icon from './Icon.svelte';

  interface Props {
    activeTab: string;
    search: string;
    savedToast?: string;
    onBack: () => void;
  }

  let {
    activeTab = $bindable('general'),
    search = $bindable(''),
    savedToast = '',
    onBack
  }: Props = $props();
</script>

<div
  in:fly={{ x: -12, duration: 220 }}
  aria-label="Preferences navigation"
  class="w-full h-full min-h-0 flex flex-col justify-between select-none"
>
  <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain p-2.5 space-y-4">
    <!-- Top Header matching Sidebar.svelte -->
    <div class="flex items-center justify-between gap-2 px-1 pt-0.5">
      <div class="flex items-center gap-1.5 min-w-0">
        <button
          type="button"
          onclick={onBack}
          title="Kembali"
          aria-label="Kembali dari Preferences"
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
        >
          <Icon name="arrow-left" size={17} />
        </button>
        <span class="text-[13px] font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
          Preferences
        </span>
      </div>
      {#if savedToast}
        <span
          class="px-2 py-0.5 rounded-full text-[10px] font-semibold shrink-0"
          style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
        >
          Tersimpan
        </span>
      {/if}
    </div>

    <!-- M3 Search Field -->
    <div class="relative px-0.5">
      <span
        class="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
        style="color: var(--md-sys-color-on-surface-variant);"
      >
        <Icon name="magnifying-glass" size={14} />
      </span>
      <input
        type="search"
        bind:value={search}
        placeholder="Cari pengaturan..."
        class="m3-input w-full h-9 pl-8 pr-3 rounded-md text-[12px] outline-none"
      />
    </div>

    <!-- Section 1: Pengaturan -->
    <div class="space-y-1">
      <div
        class="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider"
        style="color: var(--md-sys-color-on-surface-variant); opacity: 0.78;"
      >
        Pengaturan
      </div>
      {#each NUCLEAR_CATEGORIES as cat (cat.id)}
        {@const active = !search.trim() && activeTab === cat.id}
        <button
          type="button"
          onclick={() => {
            search = '';
            activeTab = cat.id;
          }}
          class="w-full h-10 flex items-center gap-2.5 px-3 rounded-md text-[13px] cursor-pointer transition-colors duration-200 text-left {active
            ? 'font-semibold'
            : 'font-medium hover:bg-[var(--md-sys-color-state-hover)]'}"
          style={active
            ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
            : 'color: var(--md-sys-color-on-surface-variant);'}
        >
          <Icon name={cat.icon} weight={active ? 'fill' : 'regular'} size={18} class="shrink-0" />
          <span class="truncate">{cat.title}</span>
        </button>
      {/each}
    </div>

    <!-- Section 2: Aplikasi -->
    <div class="space-y-1">
      <div
        class="px-3 pb-1 text-[10px] font-semibold uppercase tracking-wider"
        style="color: var(--md-sys-color-on-surface-variant); opacity: 0.78;"
      >
        Aplikasi
      </div>
      {#each NUCLEAR_APP_TABS as tab (tab.id)}
        {@const active = !search.trim() && activeTab === tab.id}
        <button
          type="button"
          onclick={() => {
            search = '';
            activeTab = tab.id;
          }}
          class="w-full h-10 flex items-center gap-2.5 px-3 rounded-md text-[13px] cursor-pointer transition-colors duration-200 text-left {active
            ? 'font-semibold'
            : 'font-medium hover:bg-[var(--md-sys-color-state-hover)]'}"
          style={active
            ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
            : 'color: var(--md-sys-color-on-surface-variant);'}
        >
          <Icon name={tab.icon} weight={active ? 'fill' : 'regular'} size={18} class="shrink-0" />
          <span class="truncate">{tab.title}</span>
        </button>
      {/each}
    </div>
  </div>
</div>
