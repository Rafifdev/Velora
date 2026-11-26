<script lang="ts">
  import { tr, type LangCode } from '../lib/i18n';
  import type { ViewRoute } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    title: string;
    lang?: LangCode;
    activeDownloadsCount?: number;
    downloadProgressPct?: number;
    onNavigate: (route: ViewRoute) => void;
  }

  let {
    title,
    lang = 'en',
    activeDownloadsCount = 0,
    downloadProgressPct = 0,
    onNavigate
  }: Props = $props();

  const ringRadius = 9.5;
  const ringCircumference = 2 * Math.PI * ringRadius;
  const clampedPct = $derived(Math.max(0, Math.min(100, downloadProgressPct)));
  const ringOffset = $derived(ringCircumference * (1 - clampedPct / 100));
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
      {#if activeDownloadsCount > 0}
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <circle
            cx="12"
            cy="12"
            r={ringRadius}
            stroke="currentColor"
            stroke-opacity="0.28"
            stroke-width="2.2"
          />
          <circle
            cx="12"
            cy="12"
            r={ringRadius}
            stroke="var(--md-sys-color-primary)"
            stroke-width="2.2"
            stroke-linecap="round"
            stroke-dasharray={ringCircumference}
            stroke-dashoffset={ringOffset}
            transform="rotate(-90 12 12)"
            style="transition: stroke-dashoffset 0.25s ease;"
          />
          <path
            d="M12 7.5V15.5M12 15.5L8.75 12.25M12 15.5L15.25 12.25"
            stroke="var(--md-sys-color-primary)"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      {:else}
        <Icon name="download" weight="regular" size={21} />
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
