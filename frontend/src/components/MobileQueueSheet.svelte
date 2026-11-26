<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { cubicIn, cubicOut } from 'svelte/easing';
  import type { Snippet } from 'svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    children: Snippet;
  }

  let { open, onClose, children }: Props = $props();

  let snapPct = $state<70 | 90>(70);
  let currentHeightPct = $state(70);
  let isDragging = $state(false);
  let startY = 0;
  let startPct = 70;

  $effect(() => {
    if (open) {
      snapPct = 70;
      currentHeightPct = 70;
      isDragging = false;
    }
  });

  function onHandlePointerDown(e: PointerEvent) {
    e.preventDefault();
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    isDragging = true;
    startY = e.clientY;
    startPct = snapPct;
    currentHeightPct = snapPct;
  }

  function onHandlePointerMove(e: PointerEvent) {
    if (!isDragging) return;
    const vh = typeof window !== 'undefined' && window.innerHeight > 0 ? window.innerHeight : 800;
    const deltaPct = ((startY - e.clientY) / vh) * 100;
    currentHeightPct = Math.max(12, Math.min(94, startPct + deltaPct));
  }

  function onHandlePointerUp(e: PointerEvent) {
    if (!isDragging) return;
    isDragging = false;
    try {
      (e.currentTarget as HTMLElement).releasePointerCapture(e.pointerId);
    } catch {}

    const draggedDownPct = startPct - currentHeightPct;
    const draggedUpPct = currentHeightPct - startPct;

    if (startPct === 70) {
      if (draggedUpPct >= 7) {
        snapPct = 90;
        currentHeightPct = 90;
      } else if (draggedDownPct >= 12) {
        onClose();
      } else {
        snapPct = 70;
        currentHeightPct = 70;
      }
    } else {
      if (draggedDownPct >= 34) {
        onClose();
      } else if (draggedDownPct >= 8) {
        snapPct = 70;
        currentHeightPct = 70;
      } else {
        snapPct = 90;
        currentHeightPct = 90;
      }
    }
  }
</script>

{#if open}
  <div class="sm:hidden fixed inset-0 z-[60] flex flex-col justify-end select-none">
    <button
      type="button"
      aria-label="Tutup Queue"
      onclick={onClose}
      in:fade={{ duration: 200 }}
      out:fade={{ duration: 180 }}
      class="absolute inset-0 bg-black/55 backdrop-blur-[2px] border-0 cursor-default"
    ></button>

    <div
      role="dialog"
      aria-modal="true"
      aria-label="Queue"
      in:fly={{ y: typeof window !== 'undefined' ? window.innerHeight * 0.75 : 540, duration: 280, opacity: 1, easing: cubicOut }}
      out:fly={{ y: typeof window !== 'undefined' ? window.innerHeight * 0.75 : 540, duration: 220, opacity: 1, easing: cubicIn }}
      class="relative z-10 w-full rounded-t-3xl shadow-2xl flex flex-col overflow-hidden"
      style="height: {currentHeightPct}dvh; background-color: var(--md-sys-color-surface-container-low); padding-bottom: env(safe-area-inset-bottom, 0px); transition: {isDragging ? 'none' : 'height 320ms cubic-bezier(0.22, 1, 0.36, 1)'};"
    >
      <div
        role="slider"
        tabindex="0"
        aria-label="Geser tinggi Queue"
        aria-valuemin={0}
        aria-valuemax={90}
        aria-valuenow={Math.round(currentHeightPct)}
        onpointerdown={onHandlePointerDown}
        onpointermove={onHandlePointerMove}
        onpointerup={onHandlePointerUp}
        onpointercancel={onHandlePointerUp}
        class="w-full pt-3 pb-2 flex items-center justify-center shrink-0 touch-none cursor-grab active:cursor-grabbing"
      >
        <div
          class="w-10 h-1 rounded-full transition-colors"
          style="background-color: color-mix(in srgb, var(--md-sys-color-on-surface-variant) 45%, transparent);"
        ></div>
      </div>

      <div class="flex-1 min-h-0 flex flex-col">
        {@render children()}
      </div>
    </div>
  </div>
{/if}
