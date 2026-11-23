<script lang="ts">
  import { cubicIn, cubicOut } from 'svelte/easing';
  import { fade, fly } from 'svelte/transition';
  import type { JamStatus } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
  import Icon from './Icon.svelte';

  interface Props {
    open: boolean;
    status: JamStatus;
    loading?: boolean;
    lang?: LangCode;
    onClose?: () => void;
    onToggleJam: () => void;
  }

  let { open = false, status, loading = false, lang = 'en', onToggleJam }: Props = $props();

  let copied = $state(false);
  let onHeight = $state(0);
  let offHeight = $state(0);
  let stageHeight = $derived(status.running ? onHeight : offHeight);

  async function copyUrl(text: string) {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      copied = true;
      setTimeout(() => {
        copied = false;
      }, 1500);
    } catch {
    }
  }
</script>

{#if open}
  <div
    role="dialog"
    aria-label="Velora Jam Remote"
    in:fly={{ y: -8, duration: 220, easing: cubicOut }}
    out:fly={{ y: -6, duration: 160, easing: cubicIn }}
    class="velora-jam-popover absolute right-0 top-full mt-2 w-80 max-w-80 rounded-lg shadow-2xl p-5 flex flex-col gap-4 z-50 select-none overflow-hidden"
    style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
  >
    <div class="w-full flex items-center justify-between gap-3">
      <div class="min-w-0 space-y-0.5">
        <div class="text-xs font-semibold tracking-tight truncate">Velora Jam</div>
        <div class="text-[11px] truncate transition-colors duration-250" style="color: var(--md-sys-color-on-surface-variant);">
          {status.running ? `${status.connectedClients} ${tr(lang, 'nav.connectedDevices')}` : tr(lang, 'nav.remoteViaNetwork')}
        </div>
      </div>

      <button
        type="button"
        role="switch"
        aria-checked={status.running}
        aria-label={status.running ? tr(lang, 'nav.turnOffJam') : tr(lang, 'nav.turnOnJam')}
        title={status.running ? tr(lang, 'nav.turnOffJam') : tr(lang, 'nav.turnOnJam')}
        disabled={loading}
        onclick={onToggleJam}
        class="velora-jam-switch w-[46px] h-7 rounded-full relative cursor-pointer shrink-0 flex items-center hover:brightness-110 disabled:opacity-50"
        style={status.running
          ? 'background-color: var(--md-sys-color-primary);'
          : 'background-color: var(--md-sys-color-surface-container-highest);'}
      >
        <span
          class="velora-jam-knob rounded-full shadow-xs"
          style={status.running
            ? 'width: 20px; height: 20px; transform: translateX(22px); background-color: var(--md-sys-color-on-primary);'
            : 'width: 14px; height: 14px; transform: translateX(5px); background-color: var(--md-sys-color-on-surface-variant);'}
        ></span>
      </button>
    </div>

    <div
      class="velora-jam-stage"
      style={stageHeight > 0 ? `height: ${stageHeight}px;` : ''}
    >
      {#if status.running}
        <div
          bind:clientHeight={onHeight}
          in:fly={{ y: 10, duration: 280, delay: 40, easing: cubicOut }}
          out:fade={{ duration: 170, easing: cubicIn }}
          class="velora-jam-pane flex flex-col gap-4"
        >
          {#if status.qrDataUrl}
            <div
              in:fade={{ duration: 220, delay: 50, easing: cubicOut }}
              class="flex flex-col items-center gap-3"
            >
              <div class="p-2.5 rounded-lg bg-white shadow-sm">
                <img src={status.qrDataUrl} alt="Velora Jam QR Code" class="w-[192px] h-[192px] block" />
              </div>
              <span class="text-[11px] text-center leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
                {tr(lang, 'nav.scanJamQrHint')}
              </span>
            </div>
          {/if}

          <div in:fade={{ duration: 220, delay: 75, easing: cubicOut }} class="w-full">
            <div
              class="flex items-center gap-2 px-3 py-2 rounded-md"
              style="background-color: var(--md-sys-color-surface-container);"
            >
              <div class="min-w-0 flex-1">
                <div class="text-[10px] font-bold uppercase tracking-wider" style="color: var(--md-sys-color-primary);">
                  Remote URL
                </div>
                <div class="text-[11px] font-mono truncate select-all mt-0.5">{status.remoteUrl}</div>
              </div>
              <button
                type="button"
                onclick={() => copyUrl(status.remoteUrl)}
                title={tr(lang, 'nav.copyRemoteUrl')}
                class="m3-icon-btn w-7 h-7 flex items-center justify-center shrink-0 cursor-pointer"
              >
                <Icon name={copied ? 'check' : 'copy'} size={14} />
              </button>
            </div>
          </div>
        </div>
      {:else}
        <div
          bind:clientHeight={offHeight}
          in:fly={{ y: -6, duration: 240, delay: 40, easing: cubicOut }}
          out:fade={{ duration: 140, easing: cubicIn }}
          class="velora-jam-pane py-3 text-center space-y-1.5"
        >
          <p class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.jamInactiveDesc')}
          </p>
        </div>
      {/if}
    </div>

    {#if status.error}
      <div in:fade={{ duration: 180 }} class="w-full text-[11px] px-3 py-2 rounded-xl bg-red-500/15 text-red-400">
        {status.error}
      </div>
    {/if}
  </div>
{/if}

<style>
  .velora-jam-popover {
    transform-origin: top right;
    will-change: transform, opacity;
  }

  .velora-jam-switch {
    transition: background-color 280ms cubic-bezier(0.22, 1, 0.36, 1), filter 200ms ease, opacity 200ms ease;
  }

  .velora-jam-knob {
    transition:
      transform 300ms cubic-bezier(0.22, 1, 0.36, 1),
      width 280ms cubic-bezier(0.22, 1, 0.36, 1),
      height 280ms cubic-bezier(0.22, 1, 0.36, 1),
      background-color 260ms cubic-bezier(0.22, 1, 0.36, 1);
    will-change: transform, width, height;
  }

  .velora-jam-stage {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    align-items: start;
    overflow: hidden;
    transition: height 320ms cubic-bezier(0.22, 1, 0.36, 1);
    will-change: height;
  }

  .velora-jam-pane {
    grid-area: 1 / 1;
    min-width: 0;
  }
</style>
