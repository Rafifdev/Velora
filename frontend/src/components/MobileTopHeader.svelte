<script lang="ts">
  import { tr, type LangCode } from '../lib/i18n';
  import type { ViewRoute } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    title: string;
    lang?: LangCode;
    activeDownloadsCount?: number;
    onNavigate: (route: ViewRoute) => void;
  }

  let {
    title,
    lang = 'en',
    activeDownloadsCount = 0,
    onNavigate
  }: Props = $props();
</script>

<div class="sm:hidden flex items-center justify-between gap-2 px-4 pt-3 pb-1">
  <h1
    class="text-2xl font-bold tracking-tight truncate min-w-0 flex-1"
    style="color: var(--md-sys-color-on-surface);"
  >
    {title}
  </h1>

  <div class="flex items-center gap-0.5 shrink-0">
    <button
      type="button"
      onclick={() => onNavigate({ name: 'history' })}
      aria-label={tr(lang, 'nav.history')}
      title={tr(lang, 'nav.history')}
      class="min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
      style="color: var(--md-sys-color-on-surface-variant);"
    >
      <Icon name="clock-counter-clockwise" size={21} />
    </button>

    <button
      type="button"
      onclick={() => onNavigate({ name: 'listening-stats' })}
      aria-label={tr(lang, 'nav.stats')}
      title={tr(lang, 'nav.stats')}
      class="min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
      style="color: var(--md-sys-color-on-surface-variant);"
    >
      <Icon name="chart-bar" size={21} />
    </button>

    <button
      type="button"
      onclick={() => onNavigate({ name: 'downloads' })}
      aria-label={tr(lang, 'nav.downloaded')}
      title={tr(lang, 'nav.downloaded')}
      class="min-w-11 min-h-11 rounded-full flex items-center justify-center relative cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
      style="color: {activeDownloadsCount > 0
        ? 'var(--md-sys-color-primary)'
        : 'var(--md-sys-color-on-surface-variant)'};"
    >
      <Icon
        name="download"
        weight={activeDownloadsCount > 0 ? 'bold' : 'regular'}
        size={21}
      />
      {#if activeDownloadsCount > 0}
        <span
          class="min-w-[18px] h-[18px] px-1 rounded-full text-[10px] font-extrabold leading-none flex items-center justify-center absolute top-1 right-1 shadow-sm tabular-nums"
          style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
        >
          {activeDownloadsCount > 99 ? '99+' : activeDownloadsCount}
        </span>
      {/if}
    </button>

    <button
      type="button"
      onclick={() => onNavigate({ name: 'preferences' })}
      aria-label={tr(lang, 'nav.settings')}
      title={tr(lang, 'nav.settings')}
      class="min-w-11 min-h-11 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
      style="color: var(--md-sys-color-on-surface-variant);"
    >
      <Icon name="gear" size={21} />
    </button>
  </div>
</div>
