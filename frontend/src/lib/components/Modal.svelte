<script lang="ts">
  import type { Snippet } from "svelte";
  import Icon from "./Icon.svelte";

  let {
    title,
    eyebrow = "",
    onclose,
    width = 460,
    children,
  }: {
    title: string;
    eyebrow?: string;
    onclose: () => void;
    width?: number;
    children: Snippet;
  } = $props();
</script>

<svelte:window onkeydown={(e) => e.key === "Escape" && onclose()} />

<div
  class="backdrop"
  role="presentation"
  onmousedown={(e) => {
    if (e.target === e.currentTarget) onclose();
  }}
>
  <div
    class="panel"
    role="dialog"
    aria-modal="true"
    aria-label={title}
    style:width="{width}px"
  >
    <header>
      <div>
        {#if eyebrow}<p class="eyebrow">{eyebrow}</p>{/if}
        <h2>{title}</h2>
      </div>
      <button class="close" onclick={onclose} aria-label="Close"
        ><Icon name="x" size={16} /></button
      >
    </header>
    <div class="body">{@render children()}</div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 500;
    display: grid;
    place-items: center;
    padding: 20px;
    background: rgba(4, 5, 8, 0.72);
    backdrop-filter: blur(8px);
    animation: fade 0.18s ease;
  }
  .panel {
    max-width: 100%;
    max-height: calc(100vh - 40px);
    display: flex;
    flex-direction: column;
    background: var(--surface);
    border: 1px solid var(--border-strong);
    border-radius: 16px;
    box-shadow: var(--shadow);
    animation: pop 0.22s ease;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 22px 22px 0;
  }
  .eyebrow {
    margin-bottom: 4px;
  }
  h2 {
    font-size: 19px;
    font-weight: 650;
    letter-spacing: -0.03em;
    color: var(--text-strong);
  }
  .close {
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    color: var(--text-dim);
    border: 1px solid var(--border-strong);
    border-radius: 9px;
    background: var(--surface-2);
    transition:
      color 0.15s,
      background 0.15s;
  }
  .close:hover {
    color: var(--text-strong);
    background: var(--surface-3);
  }
  .body {
    padding: 22px;
    overflow-y: auto;
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @keyframes pop {
    from {
      opacity: 0;
      transform: translateY(8px) scale(0.98);
    }
  }
</style>
