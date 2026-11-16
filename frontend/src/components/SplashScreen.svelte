<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    onComplete: () => void;
    isDark?: boolean;
  }

  let { onComplete, isDark = true }: Props = $props();

  // Animasi dirender menjadi WebM ber-alpha untuk tema gelap & terang,
  // lalu diputar sinkron agar pergantian tema tetap mulus.
  let darkVideo = $state<HTMLVideoElement | null>(null);
  let lightVideo = $state<HTMLVideoElement | null>(null);
  let ready = $state(false);
  let exiting = $state(false);
  let finished = $state(false);

  function finishSplash() {
    if (exiting || finished) return;
    exiting = true;
    setTimeout(() => {
      finished = true;
      onComplete();
    }, 420);
  }

  onMount(() => {
    const videos = [darkVideo, lightVideo].filter((v): v is HTMLVideoElement => v !== null);
    const safety = setTimeout(finishSplash, 8000);

    const waitReady = (v: HTMLVideoElement) =>
      new Promise<void>((resolve, reject) => {
        if (v.readyState >= 3) return resolve();
        v.addEventListener('canplaythrough', () => resolve(), { once: true });
        v.addEventListener('error', () => reject(v.error), { once: true });
      });

    Promise.all(videos.map(waitReady))
      .then(() => {
        for (const v of videos) v.currentTime = 0;
        ready = true;
        return Promise.all(videos.map((v) => v.play()));
      })
      .catch(() => setTimeout(finishSplash, 300));

    // jaga kedua video tetap sinkron
    let rafId = 0;
    const sync = () => {
      const [a, b] = videos;
      if (a && b && Math.abs(a.currentTime - b.currentTime) > 0.04) b.currentTime = a.currentTime;
      rafId = requestAnimationFrame(sync);
    };
    rafId = requestAnimationFrame(sync);

    return () => {
      clearTimeout(safety);
      cancelAnimationFrame(rafId);
    };
  });
</script>

{#if !finished}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    onclick={finishSplash}
    class="splash-backdrop fixed inset-0 z-50 flex items-center justify-center select-none cursor-pointer"
    style="
      opacity: {exiting ? 0 : 1};
      transition: opacity 420ms cubic-bezier(0.4, 0, 0.2, 1);
    "
  >
    <div
      class="splash-stage relative"
      aria-label="Velora"
      style="
        opacity: {ready ? 1 : 0};
        transform: scale({exiting ? 1.04 : 1});
        filter: blur({exiting ? 6 : 0}px);
        transition: transform 420ms cubic-bezier(0.4, 0, 0.2, 1), filter 420ms ease, opacity 160ms ease;
      "
    >
      <video
        bind:this={darkVideo}
        src="/splash/velora-splash-dark.webm"
        class="splash-video"
        style="opacity: {isDark ? 1 : 0};"
        muted
        playsinline
        preload="auto"
        disablepictureinpicture
        onended={finishSplash}
      ></video>
      <video
        bind:this={lightVideo}
        src="/splash/velora-splash-light.webm"
        class="splash-video"
        style="opacity: {isDark ? 0 : 1};"
        muted
        playsinline
        preload="auto"
        disablepictureinpicture
      ></video>
    </div>
  </div>
{/if}

<style>
  .splash-backdrop {
    background-color: rgba(11, 12, 14, 0.56);
    backdrop-filter: blur(28px);
    -webkit-backdrop-filter: blur(28px);
  }
  :global([data-theme='light']) .splash-backdrop {
    background-color: rgba(255, 255, 255, 0.56);
  }
  .splash-stage {
    width: 320px;
    height: 368px;
  }
  .splash-video {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    pointer-events: none;
    transition: opacity 300ms ease;
  }
</style>
