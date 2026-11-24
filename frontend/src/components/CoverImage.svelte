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
  let retryCount = $state(0);
  let retryNonce = $state(0);
  let imgEl = $state<HTMLImageElement | null>(null);
  let retryTimer: ReturnType<typeof setTimeout> | null = null;

  const normalizedSrc = $derived((src || '').trim());
  const effectiveSrc = $derived.by(() => {
    if (!normalizedSrc) return '';
    if (retryNonce === 0) return normalizedSrc;
    const sep = normalizedSrc.includes('?') ? '&' : '?';
    return `${normalizedSrc}${sep}_vr=${retryNonce}`;
  });

  function clearRetryTimer() {
    if (retryTimer) {
      clearTimeout(retryTimer);
      retryTimer = null;
    }
  }

  $effect(() => {
    const _current = normalizedSrc;
    clearRetryTimer();
    failed = false;
    retryCount = 0;
    retryNonce = 0;
    if (!_current) {
      loaded = false;
      return;
    }
    if (imgEl && imgEl.complete && imgEl.naturalWidth > 0) {
      loaded = true;
    } else {
      loaded = false;
    }
    return () => clearRetryTimer();
  });

  function handleError() {
    loaded = false;
    if (retryCount < 2 && normalizedSrc && !normalizedSrc.startsWith('data:')) {
      const nextRetry = retryCount + 1;
      retryCount = nextRetry;
      clearRetryTimer();
      retryTimer = setTimeout(() => {
        retryNonce = nextRetry;
      }, nextRetry * 700);
      return;
    }
    failed = true;
  }
</script>

<div class="relative w-full h-full overflow-hidden">
  {#if !loaded || failed || !effectiveSrc}
    <div
      class="absolute inset-0 flex items-center justify-center {effectiveSrc && !failed ? 'animate-pulse' : ''} {placeholderClass}"
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

  {#if effectiveSrc && !failed}
    <img
      bind:this={imgEl}
      src={effectiveSrc}
      alt=""
      aria-hidden="true"
      {loading}
      decoding="async"
      referrerpolicy="no-referrer"
      onload={() => {
        clearRetryTimer();
        loaded = true;
        failed = false;
      }}
      onerror={handleError}
      class="{imgClass} transition-opacity duration-200 {loaded ? 'opacity-100' : 'opacity-0'}"
      class:velora-yt-video-thumb={isVideoThumb}
    />
  {/if}
</div>
