<script lang="ts">
  import { fly } from 'svelte/transition';
  import { PREFERENCE_APP_TABS, PREFERENCE_CATEGORIES } from '../lib/preferences';
  import { tr, type LangCode } from '../lib/i18n';
  import Icon from './Icon.svelte';

  interface Props {
    activeTab: string;
    search: string;
    savedToast?: string;
    lang?: LangCode;
    onBack: () => void;
  }

  let {
    activeTab = $bindable('appearance'),
    search = $bindable(''),
    savedToast = '',
    lang = 'en',
    onBack
  }: Props = $props();
</script>

<div
  in:fly={{ x: -12, duration: 220 }}
  aria-label="Settings navigation"
  class="w-full h-full min-h-0 flex flex-col justify-between select-none"
>
  <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain p-2.5 space-y-4">
    <!-- Top Header matching Sidebar.svelte -->
    <div class="flex items-center justify-between gap-2 px-1 pt-0.5">
      <div class="flex items-center gap-1.5 min-w-0">
        <button
          type="button"
          onclick={onBack}
          title={tr(lang, 'nav.back')}
          aria-label={tr(lang, 'nav.back')}
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
        >
          <Icon name="arrow-left" size={17} />
        </button>
        <span class="text-[13px] font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
          {tr(lang, 'nav.settings')}
        </span>
      </div>
      {#if savedToast}
        <span
          class="px-2 py-0.5 rounded-full text-[10px] font-semibold shrink-0"
          style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
        >
          {tr(lang, 'nav.saved')}
        </span>
      {/if}
    </div>

    <!-- M3 Search Field -->
    <div class="relative px-0.5 transition-transform duration-200 ease-[cubic-bezier(0.34,1.56,0.64,1)] focus-within:scale-[1.025] active:scale-[0.985]">
      <span
        class="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
        style="color: var(--md-sys-color-on-surface-variant);"
      >
        <Icon name="magnifying-glass" size={14} />
      </span>
      <input
        type="search"
        bind:value={search}
        placeholder={tr(lang, 'nav.searchSettings')}
        class="m3-input w-full h-9 pl-8 pr-3 rounded-md text-[12px] outline-none"
      />
    </div>

    <!-- Section 1: Settings Categories -->
    <div class="space-y-1">
      {#each PREFERENCE_CATEGORIES as cat (cat.id)}
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
          <Icon name={cat.icon} weight={active ? 'fill' : 'regular'} size={14} class="shrink-0" />
          <span class="truncate">{tr(lang, cat.titleKey, cat.title)}</span>
        </button>
      {/each}
    </div>

    <div class="border-t mx-2" style="border-color: var(--md-sys-color-outline-variant);"></div>

    <!-- Section 2: App Info -->
    <div class="space-y-1">
      {#each PREFERENCE_APP_TABS as tab (tab.id)}
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
          <Icon name={tab.icon} weight={active ? 'fill' : 'regular'} size={14} class="shrink-0" />
          <span class="truncate">{tr(lang, tab.titleKey, tab.title)}</span>
        </button>
      {/each}
    </div>
  </div>
</div>
