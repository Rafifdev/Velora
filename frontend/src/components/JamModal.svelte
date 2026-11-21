<script lang="ts">
  import { cubicIn, cubicOut } from 'svelte/easing';
  import { fade, fly, slide } from 'svelte/transition';
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
    in:fly={{ y: -8, duration: 180, easing: cubicOut }}
    out:fly={{ y: -8, duration: 150, easing: cubicIn }}
    class="velora-jam-popover absolute right-0 top-full mt-2 w-80 max-w-80 rounded-lg shadow-2xl p-5 flex flex-col gap-4 z-50 select-none overflow-hidden"
    style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
  >
    <div class="w-full flex items-center justify-between gap-3">
      <div class="min-w-0 space-y-0.5">
        <div class="text-xs font-semibold tracking-tight truncate">Velora Jam</div>
        <div class="text-[11px] truncate" style="color: var(--md-sys-color-on-surface-variant);">
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
        class="w-[46px] h-7 rounded-full relative transition-all duration-200 cursor-pointer shrink-0 flex items-center hover:brightness-110 disabled:opacity-50"
        style={status.running
          ? 'background-color: var(--md-sys-color-primary);'
          : 'background-color: var(--md-sys-color-surface-container-highest);'}
      >
        <span
          class="rounded-full transition-all duration-200 shadow-xs"
          style={status.running
            ? 'width: 20px; height: 20px; transform: translateX(22px); background-color: var(--md-sys-color-on-primary);'
            : 'width: 14px; height: 14px; transform: translateX(5px); background-color: var(--md-sys-color-on-surface-variant);'}
        ></span>
      </button>
    </div>

    {#if status.running}
      <div
        in:slide={{ duration: 240, easing: cubicOut }}
        out:slide={{ duration: 180, easing: cubicIn }}
        class="flex flex-col gap-4"
      >
        {#if status.qrDataUrl}
          <div
            in:fade={{ duration: 200, delay: 40 }}
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

        <div in:fade={{ duration: 200, delay: 60 }} class="w-full">
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
        in:fade={{ duration: 160 }}
        class="py-3 text-center space-y-1.5"
      >
        <p class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
          {tr(lang, 'nav.jamInactiveDesc')}
        </p>
      </div>
    {/if}

    {#if status.error}
      <div in:fade={{ duration: 150 }} class="w-full text-[11px] px-3 py-2 rounded-xl bg-red-500/15 text-red-400">
        {status.error}
      </div>
    {/if}
  </div>
{/if}

<style>
  .velora-jam-popover {
    transform-origin: top right;
  }
</style>
