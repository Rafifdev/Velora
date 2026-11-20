<script lang="ts">
  import type { JamStatus } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    open: boolean;
    status: JamStatus;
    loading?: boolean;
    onClose?: () => void;
    onToggleJam: () => void;
  }

  let { open = false, status, loading = false, onToggleJam }: Props = $props();

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
      // ignore clipboard errors
    }
  }
</script>

{#if open}
  <div
    role="dialog"
    aria-label="Velora Jam Remote"
    class="velora-jam-popover absolute right-0 top-full mt-2 w-80 max-w-80 rounded-lg shadow-2xl p-5 flex flex-col gap-4 z-50 select-none"
    style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
  >
    <div class="w-full flex items-center justify-between gap-3">
      <div class="min-w-0 space-y-0.5">
        <div class="text-xs font-semibold tracking-tight truncate">Velora Jam</div>
        <div class="text-[11px] truncate" style="color: var(--md-sys-color-on-surface-variant);">
          {status.running ? `${status.connectedClients} perangkat terhubung` : 'Remote via jaringan'}
        </div>
      </div>

      <button
        type="button"
        role="switch"
        aria-checked={status.running}
        aria-label={status.running ? 'Matikan Velora Jam' : 'Aktifkan Velora Jam'}
        title={status.running ? 'Matikan Velora Jam' : 'Aktifkan Velora Jam'}
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
      {#if status.qrDataUrl}
        <div class="flex flex-col items-center gap-3">
          <div class="p-2.5 rounded-lg bg-white shadow-sm">
            <img src={status.qrDataUrl} alt="Velora Jam QR Code" class="w-[192px] h-[192px] block" />
          </div>
          <span class="text-[11px] text-center leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
            Scan QR Code dengan kamera HP di jaringan yang sama
          </span>
        </div>
      {/if}

      <div class="w-full">
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
            title="Salin Remote URL"
            class="m3-icon-btn w-7 h-7 flex items-center justify-center shrink-0 cursor-pointer"
          >
            <Icon name={copied ? 'check' : 'copy'} size={14} />
          </button>
        </div>
      </div>
    {:else}
      <div class="py-3 text-center space-y-1.5">
        <p class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
          Aktifkan Velora Jam untuk mengontrol pemutaran, antrean, dan pencarian lagu langsung dari browser HP.
        </p>
      </div>
    {/if}

    {#if status.error}
      <div class="w-full text-[11px] px-3 py-2 rounded-xl bg-red-500/15 text-red-400">
        {status.error}
      </div>
    {/if}
  </div>
{/if}

<style>
  .velora-jam-popover {
    transform-origin: top right;
    animation: veloraJamPopoverIn 160ms cubic-bezier(0.22, 1, 0.36, 1) both;
  }
  @keyframes veloraJamPopoverIn {
    from {
      opacity: 0;
      transform: translateY(-6px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }
</style>
