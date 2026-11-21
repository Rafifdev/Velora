<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    ariaLabelledby?: string;
    ariaDescribedby?: string;
    maxWidthClass?: string;
    class?: string;
    style?: string;
    children: Snippet;
  }

  let {
    open,
    onClose,
    ariaLabelledby,
    ariaDescribedby,
    maxWidthClass = 'max-w-md',
    class: className = 'p-6',
    style = 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);',
    children
  }: Props = $props();

  let dialogEl: HTMLDialogElement | undefined = $state();

  $effect(() => {
    if (!dialogEl) return;
    if (open && !dialogEl.open) {
      dialogEl.showModal();
    } else if (!open && dialogEl.open) {
      dialogEl.close();
    }
  });
</script>

<dialog
  bind:this={dialogEl}
  onclose={onClose}
  closedby="any"
  aria-labelledby={ariaLabelledby}
  aria-describedby={ariaDescribedby}
  class="m-auto w-full {maxWidthClass} rounded-xl shadow-2xl {className}"
  {style}
>
  {@render children()}
</dialog>
