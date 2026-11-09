<script lang="ts">
  import { app } from '../stores/app.svelte'
  import Icon from './Icon.svelte'
</script>

<div class="stack" aria-live="polite">
  {#each app.toasts as t (t.id)}
    <div class="toast {t.kind}">
      <span class="text selectable">{t.text}</span>
      <button class="icon-btn" onclick={() => app.dismiss(t.id)} aria-label="Dismiss"><Icon name="x" size={14} /></button>
    </div>
  {/each}
</div>

<style>
  .stack {
    position: fixed; right: 18px; bottom: 18px; z-index: 2000;
    display: flex; flex-direction: column; gap: 8px; max-width: 380px;
  }
  .toast {
    display: flex; align-items: flex-start; gap: 10px; padding: 10px 8px 10px 14px;
    background: var(--surface-2); border: 1px solid var(--border-strong);
    border-left: 3px solid var(--accent); border-radius: var(--radius-sm);
    box-shadow: var(--shadow); animation: slide 0.18s ease-out;
  }
  .toast.success { border-left-color: var(--success); }
  .toast.error { border-left-color: var(--danger); }
  .text { flex: 1; font-size: 13px; padding-top: 3px; word-break: break-word; }
  @keyframes slide { from { opacity: 0; transform: translateX(12px); } }
</style>