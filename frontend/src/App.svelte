<script lang="ts">
  import { onMount } from "svelte";
  import { app } from "./lib/stores/app.svelte";
  import { tabs } from "./lib/stores/tabs.svelte";
  import { settings } from "./lib/stores/settings.svelte";
  import { applyTheme } from "./lib/appTheme";
  import Rail from "./lib/components/Rail.svelte";
  import TabBar from "./lib/components/TabBar.svelte";
  import TerminalView from "./lib/components/TerminalView.svelte";
  import EmptyPage from "./lib/components/EmptyPage.svelte";
  import HostForm from "./lib/components/HostForm.svelte";
  import HostKeyDialog from "./lib/components/HostKeyDialog.svelte";
  import ConfirmDialog from "./lib/components/ConfirmDialog.svelte";
  import ImportDialog from "./lib/components/ImportDialog.svelte";
  import Unlock from "./lib/components/Unlock.svelte";
  import Toasts from "./lib/components/Toasts.svelte";
  import HostsPage from "./lib/pages/HostsPage.svelte";
  import SettingsPage from "./lib/pages/SettingsPage.svelte";
  import { snippets } from "./lib/stores/snippets.svelte";
  import SnippetsDrawer from "./lib/components/SnippetsDrawer.svelte";
  import KeysPage from './lib/pages/KeysPage.svelte'

  onMount(() => {
    settings.load(); // settings hold no secrets, so this works while the vault is locked
    app.init();
  });

  // The chosen theme recolours the whole app.
  $effect(() => applyTheme(settings.theme));

  // Terminal typography as CSS variables (used by the status bar).
  $effect(() => {
    const root = document.documentElement.style;
    root.setProperty("--terminal-font-family", settings.fontStack);
    root.setProperty("--terminal-font-size", `${settings.value.fontSize}px`);
    root.setProperty(
      "--terminal-line-height",
      String(settings.value.lineHeight),
    );
  });

  // Auto-lock: runs only while the vault is open and a timeout is set.
  $effect(() => {
    const minutes = settings.value.autoLockMinutes;
    if (minutes <= 0 || app.gated) return;

    const limit = minutes * 60_000;
    let last = Date.now();
    const bump = () => {
      last = Date.now();
    };
    const kinds = ["keydown", "mousedown", "mousemove", "wheel", "touchstart"];
    kinds.forEach((k) => window.addEventListener(k, bump, true));

    // Checking the clock (not a countdown) also locks right after the laptop wakes from sleep.
    const timer = setInterval(() => {
      if (Date.now() - last >= limit) {
        app.lock();
        app.toast("info", "Vault locked after inactivity");
      }
    }, 5000);

    return () => {
      clearInterval(timer);
      kinds.forEach((k) => window.removeEventListener(k, bump, true));
    };
  });

  // Snippets live in the vault: load them when it opens, forget them when it locks.
  $effect(() => {
    if (app.gated) snippets.clear();
    else snippets.load();
  });

  // The drawer and the "+" picker never show together, and the drawer belongs to the terminal page.
  $effect(() => {
    if (app.pickerOpen || app.page !== "terminal") snippets.open = false;
  });

  function onKey(e: KeyboardEvent) {
    if (app.gated) return;
    const mod = e.ctrlKey || e.metaKey;
    const snippetKey = mod && e.shiftKey && e.code === "KeyS";
    const findKey =
      (e.ctrlKey && e.shiftKey && e.code === "KeyF") ||
      (e.metaKey && !e.shiftKey && e.code === "KeyF");
    const newTab =
      (e.ctrlKey && e.shiftKey && e.code === "KeyT") ||
      (e.metaKey && !e.shiftKey && e.code === "KeyT");
    const closeTab =
      (e.ctrlKey && e.shiftKey && e.code === "KeyW") ||
      (e.metaKey && !e.shiftKey && e.code === "KeyW");

    if (snippetKey) {
      e.preventDefault();
      e.stopPropagation();
      if (app.page === "terminal") snippets.toggle();
    } else if (findKey && app.page === "terminal" && tabs.activeId) {
      e.preventDefault();
      e.stopPropagation();
      snippets.open = false;
      tabs.handles.get(tabs.activeId)?.find();
    } else if (newTab) {
      e.preventDefault();
      e.stopPropagation();
      app.go("terminal");
      app.pickerOpen = !app.pickerOpen;
    } else if (closeTab) {
      e.preventDefault();
      e.stopPropagation();
      if (app.page === "terminal" && tabs.activeId) tabs.close(tabs.activeId);
    } else if (e.ctrlKey && e.key === "Tab") {
      e.preventDefault();
      e.stopPropagation();
      if (app.page === "terminal") tabs.cycle(e.shiftKey ? -1 : 1);
    } else if (mod && e.key === ",") {
      e.preventDefault();
      e.stopPropagation();
      app.go("settings");
    } else if (mod && e.key.toLowerCase() === "k" && app.page !== "terminal") {
      // Not on the terminal page: Ctrl+K belongs to programs like nano there.
      e.preventDefault();
      e.stopPropagation();
      app.focusSearch = true;
      app.go("hosts");
    }
  }
</script>

<svelte:window onkeydowncapture={onKey} />

<div class="shell">
  <Rail />

  <main>
    <!-- Always mounted: hiding this page must never drop a live session. -->
    <section class="page terminal" class:hidden={app.page !== "terminal"}>
      <div class="workspace">
        <TabBar />
        <div class="window">
          {#each tabs.items as tab (tab.id)}
            <TerminalView
              {tab}
              active={tab.id === tabs.activeId && app.page === "terminal"}
            />
          {/each}
          {#if tabs.items.length === 0}
            <EmptyPage
              icon="terminal"
              eyebrow="Sessions"
              title="No open sessions"
              text="Pick a host to start a session, or press Ctrl+Shift+T."
              action="Open a session"
              onaction={() => (app.pickerOpen = true)}
            />
          {/if}
          <SnippetsDrawer />
        </div>
      </div>
    </section>

    {#if app.page === "hosts"}
      <section class="page glow"><HostsPage /></section>
    {:else if app.page === "settings"}
      <section class="page glow"><SettingsPage /></section>
    {:else if app.page === "sftp"}
      <section class="page glow">
        <EmptyPage
          icon="folder"
          eyebrow="File transfer"
          title="SFTP"
          tag="Coming soon"
          text="Connect to a host to browse and transfer files securely."
          action="Choose a host"
          disabled
        />
      </section>
    {:else if app.page === "keys"}
      <section class="page glow"><KeysPage /></section>
    {/if}
  </main>
</div>

{#if app.editor.open}<HostForm />{/if}
{#if app.importOpen}<ImportDialog />{/if}
<HostKeyDialog />
<ConfirmDialog />
{#if app.gated}<Unlock />{/if}
<Toasts />

<style>
  .shell {
    display: grid;
    grid-template-columns: var(--rail-w) minmax(0, 1fr);
    height: 100vh;
    background: var(--bg);
  }
  main {
    position: relative;
    min-width: 0;
    min-height: 0;
  }
  .page {
    position: absolute;
    inset: 0;
  }
  .page.hidden {
    display: none;
  }
  .page.glow {
    background: var(--page-glow), var(--bg);
    overflow: auto;
  }

  /* the inset, rounded terminal frame from the design */
  .terminal {
    background: var(--bg);
  }
  .workspace {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 10px 12px 12px;
  }
  .window {
    position: relative;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    border: 1px solid var(--terminal-border);
    border-radius: 0 0 9px 9px;
    background: var(--terminal-bg);
    box-shadow: 0 18px 45px rgba(0, 0, 0, 0.16);
  }
</style>
