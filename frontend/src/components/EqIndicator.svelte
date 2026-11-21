<script lang="ts">
  interface Props {
    variant?: 'playing' | 'loader';
    color?: string;
    scale?: number;
    class?: string;
  }

  let { variant = 'playing', color = '', scale, class: className = '' }: Props = $props();

  const scaleStyle = $derived(scale ? `transform: scale(${scale});` : '');
</script>

{#if variant === 'loader'}
  <span
    class="velora-eq-loader {className}"
    style="{color ? `--loader-color: ${color};` : ''}{scaleStyle}"
    aria-hidden="true"
  >
    <span class="velora-eq-bar"></span>
    <span class="velora-eq-bar"></span>
    <span class="velora-eq-bar"></span>
    <span class="velora-eq-bar"></span>
    <span class="velora-eq-bar"></span>
  </span>
{:else}
  <span
    class="velora-playing-eq {className}"
    style="{color ? `color: ${color};` : ''}{scaleStyle}"
    aria-hidden="true"
  >
    <span></span>
    <span></span>
    <span></span>
    <span></span>
  </span>
{/if}
