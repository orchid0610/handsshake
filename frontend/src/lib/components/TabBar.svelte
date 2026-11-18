<script lang="ts">
  import { tick } from "svelte";
  import { app } from "../stores/app.svelte";
  import { tabs } from "../stores/tabs.svelte";
  import { keyLabel } from "../format";
  import type { HostInfo } from "../types";
  import OsBadge from "./OsBadge.svelte";
  import Icon from "./Icon.svelte";
  import { snippets } from "../stores/snippets.svelte";

  const states = {
    connecting: "Connecting…",
    connected: "Connected",
    ended: "Session ended",
    error: "Connection failed",
  };

  let addEl: HTMLDivElement | undefined;
  let menuEl: HTMLDivElement | undefined;
  let filterEl: HTMLInputElement | undefined;
  let filter = $state("");
  let cursor = $state(0);

  const snipShortcut = /Mac/i.test(navigator.platform) ? "⌘ ⇧ S" : "Ctrl ⇧ S";
  const showFilter = $derived(app.hosts.length > 5);
  const shown = $derived.by(() => {
    const q = filter.trim().toLowerCase();
    return q
      ? app.hosts.filter((h) =>
          `${h.name} ${h.user}@${h.address}`.toLowerCase().includes(q),
        )
      : app.hosts;
  });
  const live = $derived(
    new Set(
      tabs.items.filter((t) => t.status === "connected").map((t) => t.hostId),
    ),
  );
  const current = $derived(tabs.items.find((t) => t.id === tabs.activeId));

  // Opening the picker (button or Ctrl+Shift+T) resets it and focuses it for the keyboard.
  $effect(() => {
    if (!app.pickerOpen) return;
    filter = "";
    cursor = 0;
    tick().then(() => (showFilter ? filterEl : menuEl)?.focus());
  });

  function pick(h: HostInfo) {
    app.pickerOpen = false;
    tabs.open(h);
  }

  function addHost() {
    app.pickerOpen = false;
    app.openEditor();
  }

  function onMenuKey(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.stopPropagation();
      app.pickerOpen = false;
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      cursor = Math.min(shown.length - 1, cursor + 1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      cursor = Math.max(0, cursor - 1);
    } else if (e.key === "Enter" && shown[cursor]) {
      e.preventDefault();
      pick(shown[cursor]);
    }
  }

  function onWindowDown(e: MouseEvent) {
    if (app.pickerOpen && addEl && !addEl.contains(e.target as Node))
      app.pickerOpen = false;
  }
</script>

<svelte:window onmousedown={onWindowDown} />

<div class="bar">
  <div class="tabs" role="tablist">
    {#each tabs.items as t (t.id)}
      <div
        class="tab"
        class:active={t.id === tabs.activeId}
        title="{t.subtitle}{t.osName ? ' · ' + t.osName : ''} · {states[
          t.status
        ]}"
      >
        <button
          class="label"
          role="tab"
          aria-selected={t.id === tabs.activeId}
          onclick={() => tabs.activate(t.id)}
          onauxclick={(e) => {
            if (e.button === 1) tabs.close(t.id);
          }}
        >
          <OsBadge os={t.os} like={t.osLike} size={18} />
          <span class="title">{t.title}</span>
          <i class="dot {t.status}"></i>
        </button>
        <button
          class="x"
          onclick={() => tabs.close(t.id)}
          aria-label="Close {t.title}"
        >
          <Icon name="x" size={14} />
        </button>
      </div>
    {/each}
  </div>

  <div class="add" bind:this={addEl}>
    <button
      class="plus"
      class:on={app.pickerOpen}
      onclick={() => (app.pickerOpen = !app.pickerOpen)}
      aria-label="New session"
      aria-expanded={app.pickerOpen}
      title="New session (Ctrl+Shift+T)"
    >
      <Icon name="plus" size={16} />
    </button>

    {#if app.pickerOpen}
      <div
        class="picker"
        bind:this={menuEl}
        tabindex="-1"
        role="menu"
        onkeydown={onMenuKey}
      >
        <div class="heading">
          <span>Open a new session</span>
          <small
            >{app.hosts.length} host{app.hosts.length === 1 ? "" : "s"}</small
          >
        </div>

        {#if showFilter}
          <input
            class="filter"
            bind:this={filterEl}
            bind:value={filter}
            oninput={() => (cursor = 0)}
            placeholder="Filter hosts…"
            spellcheck="false"
            autocomplete="off"
          />
        {/if}

        <div class="list">
          {#each shown as h, i (h.id)}
            <button
              class="host"
              class:cursor={i === cursor}
              role="menuitem"
              onclick={() => pick(h)}
              onmouseenter={() => (cursor = i)}
            >
              {#if h.os}
                <OsBadge os={h.os} like={h.osLike} size={30} />
              {:else}
                <span class="symbol"><Icon name="server" size={15} /></span>
              {/if}
              <span class="meta">
                <strong>{h.name}</strong>
                <small>{h.user}@{h.address}</small>
              </span>
              <i
                class="state"
                class:live={live.has(h.id)}
                title={live.has(h.id) ? "Connected" : ""}
              ></i>
            </button>
          {:else}
            <p class="none">
              {app.hosts.length === 0 ? "No hosts yet." : "No hosts match."}
            </p>
          {/each}
        </div>

        <button class="new" onclick={addHost}
          ><Icon name="plus" size={14} /> Add a new host</button
        >
      </div>
    {/if}
  </div>

  <span class="spacer"></span>
  <button
    class="snip"
    class:on={snippets.open}
    onclick={() => snippets.toggle()}
    aria-expanded={snippets.open}
    aria-controls="snippets-drawer"
    title="Snippets ({snipShortcut})"
  >
    <Icon name="code" size={14} />
    <span>Snippets</span>
    <kbd>{snipShortcut}</kbd>
  </button>
  {#if current?.keyType}
    <span
      class="info"
      title="Host key algorithm the server used to prove its identity"
    >
      <Icon name="key" size={13} /> SSH · {keyLabel(current.keyType)}
    </span>
  {/if}
</div>

<style>
  .bar {
    position: relative;
    z-index: 20;
    display: flex;
    align-items: stretch;
    height: 38px;
    border: 1px solid var(--terminal-border);
    border-bottom: 0;
    border-radius: 9px 9px 0 0;
    background: var(--terminal-surface);
  }
  /* The tab list scrolls on its own, so the picker (outside it) is never clipped. */
  .tabs {
    display: flex;
    min-width: 0;
    overflow-x: auto;
    overflow-y: hidden;
    border-radius: 8px 0 0 0;
  }
  .tabs::-webkit-scrollbar {
    height: 0;
  }

  .tab {
    position: relative;
    display: flex;
    align-items: center;
    flex: none;
    min-width: 170px;
    max-width: 240px;
    color: var(--text-faint);
    border-right: 1px solid var(--terminal-border);
    transition:
      background 0.15s,
      color 0.15s;
  }
  .tab:hover {
    background: var(--surface-2);
    color: var(--text);
  }
  .tab.active {
    background: var(--tab-active);
    color: var(--text);
  }
  .tab.active::after {
    content: "";
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 1px;
    background: var(--accent-line);
    box-shadow: 0 -2px 8px var(--accent-border);
  }
  .label {
    display: flex;
    align-items: center;
    gap: 9px;
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0 4px 0 12px;
    border: none;
    background: none;
    color: inherit;
    text-align: left;
  }
  .title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    font-weight: 500;
  }
  .dot {
    flex: none;
    margin-left: auto;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--text-faint);
  }
  .dot.connected {
    background: var(--success);
    box-shadow: 0 0 8px rgba(75, 209, 155, 0.45);
  }
  .dot.error {
    background: var(--danger);
  }
  .dot.connecting {
    background: var(--warning);
    animation: pulse 1s ease-in-out infinite;
  }
  @keyframes pulse {
    50% {
      opacity: 0.35;
    }
  }
  .x {
    display: grid;
    place-items: center;
    flex: none;
    width: 22px;
    height: 22px;
    margin-right: 8px;
    border: none;
    border-radius: 6px;
    background: none;
    color: var(--text-faint);
    opacity: 0;
    transition:
      opacity 0.12s,
      background 0.12s,
      color 0.12s;
  }
  .tab:hover .x,
  .tab.active .x {
    opacity: 1;
  }
  .x:hover {
    background: var(--surface-3);
    color: var(--danger);
  }

  .add {
    position: relative;
    flex: none;
    display: flex;
  }
  .plus {
    display: grid;
    place-items: center;
    width: 40px;
    border: none;
    border-right: 1px solid var(--terminal-border);
    background: none;
    color: var(--text-faint);
    transition:
      color 0.15s,
      background 0.15s;
  }
  .plus:hover {
    color: var(--text);
    background: var(--surface-2);
  }
  .plus.on {
    color: var(--accent-text);
    background: var(--accent-soft);
  }

  .picker {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    width: 290px;
    padding: 7px;
    outline: none;
    border: 1px solid var(--border-strong);
    border-radius: 10px;
    background: var(--surface);
    box-shadow: 0 22px 60px rgba(0, 0, 0, 0.52);
    animation: drop 0.16s ease;
  }
  @keyframes drop {
    from {
      opacity: 0;
      transform: translateY(-4px) scale(0.98);
    }
  }
  .heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 9px 10px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text);
  }
  .heading small {
    font-size: 10px;
    font-weight: 500;
    color: var(--text-faint);
  }
  .filter {
    width: calc(100% - 4px);
    height: 32px;
    margin: 0 2px 6px;
    padding: 0 10px;
    border-radius: 7px;
    border: 1px solid var(--border-strong);
    background: var(--input-bg);
    color: var(--text);
    font: inherit;
    font-size: 12px;
  }
  .filter:focus {
    outline: none;
    border-color: var(--accent-border);
  }
  .list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 300px;
    overflow-y: auto;
  }
  .host {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px;
    border: none;
    border-radius: 8px;
    background: none;
    color: inherit;
    text-align: left;
  }
  .host.cursor {
    background: var(--surface-2);
  }
  .symbol {
    display: grid;
    place-items: center;
    flex: none;
    width: 30px;
    height: 30px;
    border-radius: 8px;
    color: var(--accent-text);
    background: var(--accent-soft);
    border: 1px solid var(--accent-border);
  }
  .meta {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }
  .meta strong {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
  }
  .meta small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: 10px var(--font-mono);
    color: var(--text-faint);
  }
  .state {
    flex: none;
    margin-left: auto;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--border-strong);
  }
  .state.live {
    background: var(--success);
    box-shadow:
      0 0 0 3px rgba(75, 209, 155, 0.08),
      0 0 10px rgba(75, 209, 155, 0.25);
  }
  .none {
    padding: 14px 9px;
    font-size: 12px;
    color: var(--text-faint);
  }
  .new {
    display: flex;
    align-items: center;
    gap: 7px;
    width: 100%;
    margin-top: 6px;
    padding: 10px 9px 6px;
    border: none;
    border-top: 1px solid var(--border);
    background: none;
    text-align: left;
    font-size: 11px;
    font-weight: 600;
    color: var(--accent-text);
  }
  .new:hover {
    color: var(--text-strong);
  }

  .spacer {
    flex: 1;
  }
  .info {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 14px;
    font: 10px var(--font-mono);
    letter-spacing: 0.04em;
    color: var(--terminal-muted);
  }

  .snip {
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 0 12px;
    border: none;
    border-left: 1px solid var(--terminal-border);
    background: none;
    color: var(--terminal-muted);
    font-size: 11px;
    font-weight: 600;
    transition:
      color 0.15s,
      background 0.15s;
  }
  .snip:hover,
  .snip.on {
    color: var(--accent-text);
    background: var(--accent-soft);
  }
  .snip kbd {
    padding: 2px 5px;
    border: 1px solid var(--terminal-border);
    border-radius: 5px;
    background: var(--terminal-bg);
    color: var(--terminal-muted);
    font: 500 9px var(--font-ui);
  }
</style>
