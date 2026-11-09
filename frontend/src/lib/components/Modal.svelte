<script lang="ts">
  import type { Snippet } from 'svelte'
  import Icon from './Icon.svelte'

  let { title, onclose, width = 460, children }: {
    title: string
    onclose: () => void
    width?: number
    children: Snippet
  } = $props()
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="backdrop" role="presentation"
     onmousedown={(e) => { if (e.target === e.currentTarget) onclose() }}>
  <div class="panel" role="dialog" aria-modal="true" aria-label={title} style:width="{width}px">
    <header>
      <h2>{title}</h2>
      <button class="icon-btn" onclick={onclose} aria-label="Close"><Icon name="x" /></button>
    </header>
    <div class="body">{@render children()}</div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0; z-index: 500; display: grid; place-items: center;
    background: rgba(5, 7, 10, 0.65); backdrop-filter: blur(3px);
    animation: fade 0.12s ease-out;
  }
  .panel {
    max-width: calc(100vw - 32px); max-height: calc(100vh - 48px);
    display: flex; flex-direction: column;
    background: var(--surface); border: 1px solid var(--border-strong);
    border-radius: var(--radius-lg); box-shadow: var(--shadow);
    animation: pop 0.14s ease-out;
  }
  header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 16px 16px 0 22px;
  }
  h2 { font-size: 16px; font-weight: 600; }
  .body { padding: 18px 22px 22px; overflow-y: auto; }
  @keyframes fade { from { opacity: 0; } }
  @keyframes pop { from { opacity: 0; transform: translateY(6px) scale(0.985); } }
</style>