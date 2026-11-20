<script lang="ts">
  import type { FireStreakStatus } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    open: boolean;
    fireStreak: FireStreakStatus;
    onClose: () => void;
    onOpenFullPage: () => void;
  }

  let { open, fireStreak, onClose, onOpenFullPage }: Props = $props();

  let dialogEl: HTMLDialogElement | undefined = $state();

  $effect(() => {
    if (!dialogEl) return;
    if (open && !dialogEl.open) {
      dialogEl.showModal();
    } else if (!open && dialogEl.open) {
      dialogEl.close();
    }
  });

  function handleClose() {
    if (dialogEl) dialogEl.returnValue = '';
    onClose();
  }
</script>

<!-- Google Material Design 3 Dialog -->
<dialog
  bind:this={dialogEl}
  onclose={handleClose}
  closedby="any"
  aria-labelledby="streak-dialog-title"
  class="m-auto w-full max-w-lg rounded-lg shadow-2xl p-6"
  style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
>
  <div class="space-y-5">
    <div class="flex items-start justify-between gap-4">
      <div class="flex items-center gap-3.5">
        <div
          class="w-12 h-12 rounded-md flex items-center justify-center"
          style="background-color: var(--md-sys-color-tertiary-container); color: var(--md-sys-color-on-tertiary-container);"
        >
          <Icon name="flame" weight="fill" size={26} />
        </div>
        <div>
          <h2 id="streak-dialog-title" class="text-xl font-semibold">Fire Streak Harian</h2>
          <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
            Dengarkan musik tiap hari untuk menjaga streak tetap aktif
          </p>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-3 gap-3">
      <div class="m3-card p-3.5">
        <div class="text-xs flex items-center gap-1" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="flame" weight="fill" size={14} class="text-orange-400" /> Streak
        </div>
        <div class="text-2xl font-semibold text-orange-400 mt-1 tabular-nums">{fireStreak.currentStreak} Hari</div>
      </div>
      <div class="m3-card p-3.5">
        <div class="text-xs flex items-center gap-1" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="trophy" weight="fill" size={14} class="text-amber-400" /> Rekor
        </div>
        <div class="text-2xl font-semibold mt-1 tabular-nums">{fireStreak.longestStreak} Hari</div>
      </div>
      <div class="m3-card p-3.5">
        <div class="text-xs flex items-center gap-1" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="calendar" weight="fill" size={14} style="color: var(--md-sys-color-primary);" /> Hari Ini
        </div>
        <div class="text-2xl font-semibold mt-1 tabular-nums" style="color: var(--md-sys-color-primary);">
          {fireStreak.progressPercent}%
        </div>
      </div>
    </div>

    <div class="m3-card p-4 space-y-3">
      <div>
        <div class="flex justify-between text-xs mb-1.5">
          <span style="color: var(--md-sys-color-on-surface-variant);">Target Menit Harian</span>
          <span class="font-semibold tabular-nums">{fireStreak.todayMinutes} / {fireStreak.goalMinutes} menit</span>
        </div>
        <div
          class="h-2 rounded-full overflow-hidden"
          style="background-color: var(--md-sys-color-surface-container-highest);"
        >
          <div
            class="h-full rounded-full bg-orange-400 transition-all"
            style="width: {Math.min(100, (fireStreak.todayMinutes / Math.max(1, fireStreak.goalMinutes)) * 100)}%"
          ></div>
        </div>
      </div>

      <div>
        <div class="flex justify-between text-xs mb-1.5">
          <span style="color: var(--md-sys-color-on-surface-variant);">Target Lagu Harian</span>
          <span class="font-semibold tabular-nums">{fireStreak.todayTracks} / {fireStreak.goalTracks} lagu</span>
        </div>
        <div
          class="h-2 rounded-full overflow-hidden"
          style="background-color: var(--md-sys-color-surface-container-highest);"
        >
          <div
            class="h-full rounded-full transition-all"
            style="background-color: var(--md-sys-color-primary); width: {Math.min(100, (fireStreak.todayTracks / Math.max(1, fireStreak.goalTracks)) * 100)}%"
          ></div>
        </div>
      </div>
    </div>

    <div class="flex items-center justify-end gap-2">
      <button
        type="button"
        onclick={() => dialogEl?.close()}
        class="m3-btn-outlined px-5 py-2 text-xs cursor-pointer"
      >
        Tutup
      </button>
      <button
        type="button"
        onclick={() => {
          dialogEl?.close();
          onOpenFullPage();
        }}
        class="m3-btn-filled px-5 py-2 text-xs cursor-pointer"
      >
        Detail Statistik
      </button>
    </div>
  </div>
</dialog>
