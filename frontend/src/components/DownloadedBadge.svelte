<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    title?: string;
    class?: string;
  }

  let { title = 'Downloaded', class: className = 'top-1.5 right-1.5 sm:top-2 sm:right-2' }: Props = $props();
</script>

<span
  {title}
  aria-label={title}
  class="pointer-events-none absolute z-20 w-6 h-6 rounded-full flex items-center justify-center shadow-md {className}"
  style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
>
  <Icon name="check" weight="bold" size={13} />
</span>
