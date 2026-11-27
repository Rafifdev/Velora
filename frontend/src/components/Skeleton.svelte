<script lang="ts">
  interface Props {
    variant?: 'box' | 'card-grid' | 'track-rows' | 'download-list';
    count?: number;
    circle?: boolean;
    gridClass?: string;
    class?: string;
    style?: string;
  }

  let {
    variant = 'box',
    count = 5,
    circle = false,
    gridClass = 'grid grid-cols-2 md:grid-cols-4 lg:grid-cols-5 gap-3',
    class: className = 'h-4 w-full rounded bg-[var(--md-sys-color-surface-container-highest)]',
    style
  }: Props = $props();

  const items = $derived(Array.from({ length: Math.max(1, count) }, (_, i) => i));
  const titleWidths = [62, 48, 70, 54, 66, 44, 58, 60];
</script>

{#if variant === 'card-grid'}
  <div class="{gridClass} animate-pulse" aria-busy="true">
    {#each items as idx (idx)}
      <div class="w-full min-w-0 flex flex-col gap-2.5 p-2.5 rounded-xl">
        <div
          class="w-full aspect-square {circle ? 'rounded-full' : 'rounded-lg'} bg-[var(--md-sys-color-surface-container-high)]"
        ></div>
        <div class="h-3.5 w-3/4 {circle ? 'mx-auto' : ''} rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
        {#if !circle}
          <div class="h-2.5 w-1/2 rounded bg-[var(--md-sys-color-surface-container)]"></div>
        {/if}
      </div>
    {/each}
  </div>
{:else if variant === 'track-rows'}
  <div class="space-y-1.5 animate-pulse" aria-busy="true">
    {#each items as idx (idx)}
      {@const w = titleWidths[idx % titleWidths.length]}
      <div class="-mx-2 grid grid-cols-[minmax(0,1fr)_52px_32px] sm:grid-cols-[32px_minmax(0,1fr)_52px_32px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(120px,0.7fr)_minmax(120px,0.7fr)_56px_32px] items-center gap-4 px-3 py-2.5 rounded-lg">
        <div class="hidden sm:flex justify-center">
          <div class="w-4 h-3 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
        </div>
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-10 h-10 rounded-md bg-[var(--md-sys-color-surface-container-highest)] shrink-0"></div>
          <div class="space-y-1.5 min-w-0 flex-1">
            <div class="h-3.5 rounded bg-[var(--md-sys-color-surface-container-highest)]" style="width: {w}%;"></div>
            <div class="h-2.5 w-28 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
          </div>
        </div>
        <div class="hidden md:block">
          <div class="h-3 w-14 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
        </div>
        <div class="hidden md:block">
          <div class="h-3 w-20 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
        </div>
        <div class="flex justify-end pr-1">
          <div class="h-3 w-8 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
        </div>
        <div></div>
      </div>
    {/each}
  </div>
{:else if variant === 'download-list'}
  <div class="space-y-3 animate-pulse" aria-busy="true">
    {#each items as idx (idx)}
      <div
        class="rounded-xl px-3.5 py-3 flex items-center gap-3.5"
        style="background-color: var(--md-sys-color-surface-container);"
      >
        <div class="w-12 h-12 rounded-lg shrink-0 bg-[var(--md-sys-color-surface-container-highest)]"></div>
        <div class="min-w-0 flex-1 space-y-2">
          <div class="h-3.5 w-3/5 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
          <div class="h-2.5 w-2/5 rounded bg-[var(--md-sys-color-surface-container-high)]"></div>
          {#if idx === 0}
            <div class="h-1 w-full rounded-full bg-[var(--md-sys-color-primary)]/40"></div>
          {/if}
        </div>
      </div>
    {/each}
  </div>
{:else}
  <div class="animate-pulse {className}" {style} aria-hidden="true"></div>
{/if}
