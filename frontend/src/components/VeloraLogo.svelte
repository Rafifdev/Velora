<script lang="ts">
  interface Props {
    size?: number;
    class?: string;
  }

  let { size = 24, class: className = '' }: Props = $props();
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  viewBox="0 0 512 512"
  width={size}
  height={size}
  fill="none"
  shape-rendering="geometricPrecision"
  class={className}
  aria-hidden="true"
>
  <path
    d="M 88 176 A 188 188 0 0 0 88 288"
    stroke="#FF9E64"
    stroke-width="40"
    stroke-linecap="round"
  />
  <path
    d="M 424 176 A 188 188 0 0 1 424 288"
    stroke="#FF9E64"
    stroke-width="40"
    stroke-linecap="round"
  />
  <path
    d="M 205 128
       C 162 148, 140 188, 140 232
       C 140 286, 174 328, 256 384
       C 338 328, 372 286, 372 232
       C 372 188, 350 148, 307 128"
    stroke="currentColor"
    stroke-width="40"
    stroke-linecap="round"
    stroke-linejoin="round"
  />
  <circle cx="256" cy="232" r="64" fill="#FF9E64" />
</svg>
