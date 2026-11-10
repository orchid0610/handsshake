<script lang="ts">
  import { iconFor, glyphFor } from "../osIcons";
  import Icon from "./Icon.svelte";

  let {
    os = "",
    like = "",
    size = 20,
    title = "",
  }: {
    os?: string;
    like?: string;
    size?: number;
    title?: string;
  } = $props();

  const icon = $derived(os ? iconFor(os, like) : null);
</script>

{#if icon}
  <span
    class="badge"
    {title}
    style:width="{size}px"
    style:height="{size}px"
    style:background={"#" + icon.hex}
  >
    <svg
      viewBox="0 0 24 24"
      width={size * 0.62}
      height={size * 0.62}
      fill={glyphFor(icon.hex)}
      aria-hidden="true"
    >
      <path d={icon.path} />
    </svg>
  </span>
{:else}
  <span
    class="badge generic"
    {title}
    style:width="{size}px"
    style:height="{size}px"
  >
    <Icon name="terminal" size={Math.round(size * 0.6)} />
  </span>
{/if}

<style>
  .badge {
    display: inline-grid;
    place-items: center;
    flex: none;
    border-radius: 28%;
  }
  .generic {
    background: var(--surface-3);
    color: var(--text-dim);
  }
</style>
