<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    activeDownloadsCount?: number;
    downloadProgressPct?: number;
    iconSize?: number;
  }

  let {
    activeDownloadsCount = 0,
    downloadProgressPct = 0,
    iconSize = 20
  }: Props = $props();

  const ringRadius = 9.5;
  const ringCircumference = 2 * Math.PI * ringRadius;
  const clampedPct = $derived(Math.max(0, Math.min(100, downloadProgressPct)));
  const ringOffset = $derived(ringCircumference * (1 - clampedPct / 100));
</script>

{#if activeDownloadsCount > 0}
  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
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
    <text
      x="12"
      y="12"
      text-anchor="middle"
      dominant-baseline="central"
      fill="var(--md-sys-color-primary)"
      font-size={activeDownloadsCount > 9 ? '8.5' : '10'}
      font-weight="800"
      style="font-variant-numeric: tabular-nums;"
    >
      {activeDownloadsCount > 99 ? '99' : activeDownloadsCount}
    </text>
  </svg>
{:else}
  <Icon name="download" weight="regular" size={iconSize} />
{/if}
