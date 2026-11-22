<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    src?: string | null;
    fallbackIcon?: string;
    iconSize?: number;
    imgClass?: string;
    placeholderClass?: string;
    isVideoThumb?: boolean;
    loading?: 'lazy' | 'eager';
  }

  let {
    src = '',
    fallbackIcon = 'vinyl-record',
    iconSize = 36,
    imgClass = 'block w-full h-full object-cover object-center',
    placeholderClass = '',
    isVideoThumb = false,
    loading = 'lazy'
  }: Props = $props();

  let loaded = $state(false);
  let failed = $state(false);
  let imgEl = $state<HTMLImageElement | null>(null);

  const normalizedSrc = $derived((src || '').trim());

  $effect(() => {
    const _current = normalizedSrc;
    failed = false;
    if (!_current) {
      loaded = false;
      return;
    }
    if (imgEl && imgEl.complete) {
      if (imgEl.naturalWidth > 0) {
        loaded = true;
      } else {
        loaded = false;
        failed = true;
      }
    } else {
      loaded = false;
    }
  });
</script>

<div class="relative w-full h-full overflow-hidden">
  {#if !loaded || failed || !normalizedSrc}
    <div
      class="absolute inset-0 flex items-center justify-center {normalizedSrc && !failed ? 'animate-pulse' : ''} {placeholderClass}"
      style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-primary-container) 55%, var(--md-sys-color-surface-container-high)), var(--md-sys-color-surface-container-highest));"
      aria-hidden="true"
    >
      <Icon
        name={fallbackIcon}
        weight="fill"
        size={iconSize}
        style="color: var(--md-sys-color-on-surface-variant); opacity: 0.42;"
      />
    </div>
  {/if}

  {#if normalizedSrc && !failed}
    <img
      bind:this={imgEl}
      src={normalizedSrc}
      alt=""
      aria-hidden="true"
      {loading}
      decoding="async"
      onload={() => {
        loaded = true;
        failed = false;
      }}
      onerror={() => {
        loaded = false;
        failed = true;
      }}
      class="{imgClass} transition-opacity duration-200 {loaded ? 'opacity-100' : 'opacity-0'}"
      class:velora-yt-video-thumb={isVideoThumb}
    />
  {/if}
</div>
