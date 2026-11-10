<script lang="ts">
  import { onMount } from "svelte";
  import { app } from "./lib/stores/app.svelte";
  import { tabs } from "./lib/stores/tabs.svelte";
  import Sidebar from "./lib/components/Sidebar.svelte";
  import TabBar from "./lib/components/TabBar.svelte";
  import TerminalView from "./lib/components/TerminalView.svelte";
  import HostForm from "./lib/components/HostForm.svelte";
  import HostKeyDialog from "./lib/components/HostKeyDialog.svelte";
  import ConfirmDialog from "./lib/components/ConfirmDialog.svelte";
  import Unlock from "./lib/components/Unlock.svelte";
  import Toasts from "./lib/components/Toasts.svelte";
  import Icon from "./lib/components/Icon.svelte";

  onMount(() => app.init());

  function onKey(e: KeyboardEvent) {
    if (e.ctrlKey && e.key === "Tab") {
      e.preventDefault();
      e.stopPropagation();
      tabs.cycle(e.shiftKey ? -1 : 1);
    }
  }
</script>
<svelte:window onkeydowncapture={onKey} />
<div class="shell">
  <Sidebar />
  <main>
    {#if tabs.items.length > 0}<TabBar />{/if}
    <div class="stage">
      {#each tabs.items as tab (tab.id)}
        <TerminalView {tab} active={tab.id === tabs.activeId} />
      {/each}
      {#if tabs.items.length === 0}
        <div class="welcome">
          <span class="big"><Icon name="terminal" size={30} /></span>
          <h2>No open sessions</h2>
          <p>
            Pick a host from the sidebar to connect, or press <kbd>Ctrl K</kbd> to
            search.
          </p>
        </div>
      {/if}
    </div>
  </main>
</div>

{#if app.editor.open}<HostForm />{/if}
<HostKeyDialog />
<ConfirmDialog />
{#if app.gated}<Unlock />{/if}
<Toasts />

<style>
  .shell {
    display: grid;
    grid-template-columns: var(--sidebar-w) 1fr;
    height: 100vh;
  }
  main {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
    background: var(--term-bg);
  }
  .stage {
    position: relative;
    flex: 1;
    min-height: 0;
  }
  .welcome {
    position: absolute;
    inset: 0;
    display: grid;
    align-content: center;
    justify-items: center;
    gap: 10px;
    color: var(--text-faint);
    text-align: center;
  }
  .big {
    display: grid;
    place-items: center;
    width: 64px;
    height: 64px;
    margin-bottom: 6px;
    border-radius: 20px;
    background: var(--surface);
    border: 1px solid var(--border);
  }
  .welcome h2 {
    font-size: 17px;
    font-weight: 600;
    color: var(--text-dim);
  }
  kbd {
    font: 500 11px var(--font-ui);
    padding: 2px 6px;
    border: 1px solid var(--border-strong);
    border-radius: 5px;
  }
</style>
