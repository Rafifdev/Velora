<script lang="ts">
  interface Props {
    name: string;
    weight?: 'regular' | 'bold' | 'fill';
    size?: number;
    class?: string;
    style?: string;
  }

  let {
    name,
    weight = 'regular',
    size = 18,
    class: className = '',
    style = ''
  }: Props = $props();

  const weightClass = $derived(weight === 'regular' ? 'ph' : `ph-${weight}`);
  const iconClass = $derived(name.startsWith('ph-') ? name : `ph-${name}`);
</script>

<i
  class="{weightClass} {iconClass} inline-flex items-center justify-center leading-none select-none {className}"
  style="font-size: {size}px; {style}"
  aria-hidden="true"
></i>
