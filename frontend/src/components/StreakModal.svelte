<script lang="ts">
  import type { FireStreakStatus } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
  import Icon from './Icon.svelte';
  import ModalDialog from './ModalDialog.svelte';

  interface Props {
    open: boolean;
    fireStreak: FireStreakStatus;
    lang?: LangCode;
    onClose: () => void;
    onOpenFullPage: () => void;
  }

  let { open, fireStreak, lang = 'en', onClose, onOpenFullPage }: Props = $props();
</script>

<ModalDialog
  {open}
  {onClose}
  ariaLabelledby="streak-dialog-title"
  maxWidthClass="max-w-lg"
  class="p-6"
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
          <h2 id="streak-dialog-title" class="text-xl font-semibold">{tr(lang, 'nav.dailyStreak')}</h2>
          <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.streakSubtitle')}
          </p>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-3 gap-3">
      <div class="m3-card p-3.5">
        <div class="text-xs flex items-center gap-1" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="flame" weight="fill" size={14} class="text-orange-400" /> {tr(lang, 'nav.streak')}
        </div>
        <div class="text-2xl font-semibold text-orange-400 mt-1 tabular-nums">{fireStreak.currentStreak} {tr(lang, 'nav.days')}</div>
      </div>
      <div class="m3-card p-3.5">
        <div class="text-xs flex items-center gap-1" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="trophy" weight="fill" size={14} class="text-amber-400" /> {tr(lang, 'nav.record')}
        </div>
        <div class="text-2xl font-semibold mt-1 tabular-nums">{fireStreak.longestStreak} {tr(lang, 'nav.days')}</div>
      </div>
      <div class="m3-card p-3.5">
        <div class="text-xs flex items-center gap-1" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="calendar" weight="fill" size={14} style="color: var(--md-sys-color-primary);" /> {tr(lang, 'nav.today')}
        </div>
        <div class="text-2xl font-semibold mt-1 tabular-nums" style="color: var(--md-sys-color-primary);">
          {fireStreak.progressPercent}%
        </div>
      </div>
    </div>

    <div class="m3-card p-4 space-y-3">
      <div>
        <div class="flex justify-between text-xs mb-1.5">
          <span style="color: var(--md-sys-color-on-surface-variant);">{tr(lang, 'nav.dailyMinutesGoal')}</span>
          <span class="font-semibold tabular-nums">{fireStreak.todayMinutes} / {fireStreak.goalMinutes} {tr(lang, 'nav.minutes')}</span>
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
          <span style="color: var(--md-sys-color-on-surface-variant);">{tr(lang, 'nav.dailyTracksGoal')}</span>
          <span class="font-semibold tabular-nums">{fireStreak.todayTracks} / {fireStreak.goalTracks} {tr(lang, 'nav.songs').toLowerCase()}</span>
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
        onclick={onClose}
        class="m3-btn-outlined px-5 py-2 text-xs cursor-pointer"
      >
        {tr(lang, 'nav.close')}
      </button>
      <button
        type="button"
        onclick={() => {
          onClose();
          onOpenFullPage();
        }}
        class="m3-btn-filled px-5 py-2 text-xs cursor-pointer"
      >
        {tr(lang, 'nav.statsDetail')}
      </button>
    </div>
  </div>
</ModalDialog>
