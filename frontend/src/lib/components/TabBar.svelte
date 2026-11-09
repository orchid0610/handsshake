<script lang="ts">
  import { tabs } from '../stores/tabs.svelte'
  import Icon from './Icon.svelte'
</script>

<div class="bar" role="tablist">
  {#each tabs.items as t (t.id)}
    <div class="tab" class:active={t.id === tabs.activeId}>
      <button class="label" role="tab" aria-selected={t.id === tabs.activeId}
              onclick={() => tabs.activate(t.id)}
              onauxclick={(e) => { if (e.button === 1) tabs.close(t.id) }}>
        <span class="dot {t.status}"></span>
        <span class="title">{t.title}</span>
      </button>
      <button class="icon-btn x" onclick={() => tabs.close(t.id)} aria-label="Close {t.title}"><Icon name="x" size={13} /></button>
    </div>
  {/each}
</div>

<style>
  .bar { display: flex; align-items: flex-end; gap: 2px; height: 42px; padding: 8px 10px 0; background: var(--surface); border-bottom: 1px solid var(--border); overflow-x: auto; overflow-y: hidden; }
  .bar::-webkit-scrollbar { height: 0; }
  .tab { display: flex; align-items: center; min-width: 120px; max-width: 220px; height: 34px; border-radius: 9px 9px 0 0; color: var(--text-dim); }
  .tab:hover { background: var(--surface-2); }
  .tab.active { background: var(--term-bg); color: var(--text); box-shadow: inset 0 2px 0 var(--accent); }
  .label { display: flex; align-items: center; gap: 9px; flex: 1; min-width: 0; height: 100%; padding: 0 4px 0 13px; border: none; background: none; text-align: left; cursor: pointer; }
  .title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
  .x { margin-right: 5px; width: 22px; height: 22px; }
  .dot { flex: none; width: 7px; height: 7px; border-radius: 50%; background: var(--text-faint); }
  .dot.connected { background: var(--success); }
  .dot.error { background: var(--danger); }
  .dot.connecting { background: var(--warning); animation: pulse 1s ease-in-out infinite; }
  @keyframes pulse { 50% { opacity: 0.3; } }
</style>