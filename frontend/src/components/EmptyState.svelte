<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from './Icon.svelte';

  interface Props {
    icon: string;
    iconWeight?: 'regular' | 'bold' | 'fill';
    iconSize?: number;
    title: string;
    description?: string;
    titleClass?: string;
    descClass?: string;
    class?: string;
    children?: Snippet;
  }

  let {
    icon,
    iconWeight = 'bold',
    iconSize = 38,
    title,
    description = '',
    titleClass = 'text-2xl font-bold tracking-tight',
    descClass = 'mt-2 text-[13px] max-w-sm mx-auto',
    class: className = 'flex-1 flex flex-col items-center justify-center text-center py-12 px-4',
    children
  }: Props = $props();
</script>

<div class={className}>
  <div class="mb-4" style="color: var(--md-sys-color-on-surface);">
    <Icon name={icon} weight={iconWeight} size={iconSize} />
  </div>
  <h2 class={titleClass} style="color: var(--md-sys-color-on-surface);">
    {title}
  </h2>
  {#if description}
    <p class={descClass} style="color: var(--md-sys-color-on-surface-variant);">
      {description}
    </p>
  {/if}
  {#if children}
    {@render children()}
  {/if}
</div>
