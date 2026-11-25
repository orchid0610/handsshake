<script lang="ts">
  import { tick } from "svelte";
  import { app } from "../stores/app.svelte";
  import { tabs } from "../stores/tabs.svelte";
  import { api, errText } from "../api";
  import { ago } from "../format";
  import type { HostInfo } from "../types";
  import Icon from "../components/Icon.svelte";
  import OsBadge from "../components/OsBadge.svelte";

  let search: HTMLInputElement | undefined;

  const shortcut = /Mac/i.test(navigator.platform) ? "⌘ K" : "Ctrl K";

  // A host counts as connected while any of its sessions is open.
  const connected = $derived(
    new Set(
      tabs.items.filter((t) => t.status === "connected").map((t) => t.hostId),
    ),
  );

  // Recent sessions are the hosts you connected to most recently.
  const recents = $derived(
    app.hosts
      .filter((h) => h.lastUsed > 0)
      .sort((a, b) => b.lastUsed - a.lastUsed)
      .slice(0, 5),
  );

  // Ctrl+K from another page sets this flag, then switches here.
  $effect(() => {
    if (app.focusSearch) {
      app.focusSearch = false;
      tick().then(() => {
        search?.focus();
        search?.select();
      });
    }
  });

  const jumpName = (h: HostInfo) =>
    h.jumpHostId
      ? (app.hosts.find((x) => x.id === h.jumpHostId)?.name ?? "")
      : "";
  const meta = (h: HostInfo) =>
    [h.osName || h.user, jumpName(h) && `via ${jumpName(h)}`]
      .filter(Boolean)
      .join(" · ");

  function askDelete(h: HostInfo) {
    const users = app.hosts.filter((x) => x.jumpHostId === h.id).length;
    app.confirm = {
      title: "Delete host?",
      message:
        `“${h.name}” and its saved credentials will be permanently removed from your vault.` +
        (users > 0
          ? ` ${users} other host${users === 1 ? "" : "s"} connect${users === 1 ? "s" : ""} through it and will connect directly instead.`
          : ""),
      action: "Delete",
      run: async () => {
        try {
          await api.deleteHost(h.id);
          await app.refreshHosts();
          app.toast("success", "Host deleted");
        } catch (e) {
          app.toast("error", errText(e));
        }
      },
    };
  }

  function onSearchKey(e: KeyboardEvent) {
    if (e.key === "Enter" && app.filtered[0]) tabs.open(app.filtered[0]);
    if (e.key === "Escape") {
      app.query = "";
      search?.blur();
    }
  }
</script>

<div class="inner">
  <header>
    <div>
      <h1 class="page-title">Your hosts</h1>
      <p class="page-subtitle">Connect to and manage your secure servers.</p>
    </div>
    <div class="head-actions">
      <button
        class="btn"
        onclick={() => (app.importOpen = true)}
        title="Import from an SSH config file"
      >
        <Icon name="download" size={16} /> Import
      </button>
      <button class="btn btn-primary" onclick={() => app.openEditor()}>
        <Icon name="plus" size={16} /> New host
      </button>
    </div>
  </header>

  <label class="search">
    <Icon name="search" size={16} />
    <input
      bind:this={search}
      bind:value={app.query}
      placeholder="Search by name, hostname, or IP…"
      spellcheck="false"
      autocomplete="off"
      onkeydown={onSearchKey}
    />
    <kbd>{shortcut}</kbd>
  </label>

  {#if app.hosts.length === 0}
    <div class="blank">
      <span class="blank-icon"><Icon name="server" size={23} /></span>
      <h2>No hosts yet</h2>
      <p>
        Add a server to connect to it, or import the hosts you already have in <code
          >~/.ssh/config</code
        >.
      </p>
      <div class="head-actions">
        <button class="btn" onclick={() => (app.importOpen = true)}
          ><Icon name="download" size={16} /> Import from SSH config</button
        >
        <button class="btn btn-primary" onclick={() => app.openEditor()}
          ><Icon name="plus" size={16} /> Add your first host</button
        >
      </div>
    </div>
  {:else}
    <section>
      <div class="section-title">
        <h2>All hosts</h2>
        <span>
          {#if app.query.trim()}{app.filtered.length} of {app.hosts.length} saved
            connections
          {:else}{app.hosts.length} saved connection{app.hosts.length === 1
              ? ""
              : "s"}{/if}
        </span>
      </div>

      {#if app.filtered.length === 0}
        <p class="none">No hosts match “{app.query}”.</p>
      {:else}
        <div class="grid">
          {#each app.filtered as h (h.id)}
            <div class="tile">
              <button
                class="tile-main"
                onclick={() => tabs.open(h)}
                aria-label="Connect to {h.name}"
              >
                <span class="top">
                  {#if h.os}
                    <OsBadge os={h.os} like={h.osLike} size={37} />
                  {:else}
                    <span class="symbol"><Icon name="server" size={19} /></span>
                  {/if}
                  {#if connected.has(h.id)}<span class="pill"
                      ><i></i>Connected</span
                    >{/if}
                </span>
                <span class="info">
                  <strong>{h.name}</strong>
                  <small>{h.user}@{h.address}</small>
                </span>
                <span class="meta">
                  <span class="os">{meta(h)}</span>
                  <span class="go"
                    >Connect <Icon name="chevron" size={13} /></span
                  >
                </span>
              </button>
              <span class="actions">
                <button
                  class="icon-btn"
                  onclick={() => app.openEditor(h.id)}
                  aria-label="Edit {h.name}"
                  ><Icon name="pencil" size={14} /></button
                >
                <button
                  class="icon-btn"
                  onclick={() => askDelete(h)}
                  aria-label="Delete {h.name}"
                  ><Icon name="trash" size={14} /></button
                >
              </span>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    {#if recents.length > 0}
      <section class="recent">
        <div class="section-title">
          <h2>Recent sessions</h2>
          <span>Pick up where you left off</span>
        </div>
        <div class="table">
          {#each recents as h (h.id)}
            <button class="row" onclick={() => tabs.open(h)}>
              <span class="ricon"><Icon name="terminal" size={16} /></span>
              <span class="rname"
                ><strong>{h.name}</strong><small>{h.user}@{h.address}</small
                ></span
              >
              <span class="ros">{h.osName}</span>
              <time>{ago(h.lastUsed)}</time>
              <Icon name="chevron" size={15} />
            </button>
          {/each}
        </div>
      </section>
    {/if}
  {/if}

  <p class="security">
    <Icon name="key" size={15} /> Credentials are encrypted and stored only on this
    device.
  </p>
</div>

<style>
  .inner {
    width: min(1080px, calc(100% - 88px));
    margin: 0 auto;
    padding: 52px 0 34px;
  }
  header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 24px;
  }
  .page-subtitle {
    margin-top: 7px;
  }
  .head-actions {
    display: flex;
    gap: 10px;
  }

  .search {
    display: flex;
    align-items: center;
    gap: 9px;
    width: min(520px, 100%);
    height: 42px;
    margin: 30px 0 38px;
    padding: 0 12px;
    color: var(--text-faint);
    background: var(--card);
    border: 1px solid var(--border);
    border-radius: 9px;
    transition:
      border-color 0.15s,
      box-shadow 0.15s;
  }
  .search:focus-within {
    border-color: var(--accent-border);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }
  .search input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--text);
    font: inherit;
    font-size: 13.5px;
  }
  .search input::placeholder {
    color: var(--text-faint);
  }
  kbd {
    padding: 2px 6px;
    border: 1px solid var(--border-strong);
    border-radius: 5px;
    background: var(--surface-2);
    color: var(--text-faint);
    font: 500 10px var(--font-ui);
  }

  .section-title {
    display: flex;
    align-items: baseline;
    gap: 10px;
    margin-bottom: 12px;
  }
  .section-title h2 {
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
  }
  .section-title span {
    font-size: 11px;
    color: var(--text-faint);
  }
  .none {
    padding: 24px 4px;
    color: var(--text-dim);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(290px, 1fr));
    gap: 12px;
  }
  .tile {
    position: relative;
  }
  .tile-main {
    display: flex;
    flex-direction: column;
    width: 100%;
    min-height: 168px;
    padding: 16px;
    text-align: left;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--card);
    color: inherit;
    transition:
      transform 0.18s,
      border-color 0.18s,
      background 0.18s,
      box-shadow 0.18s;
  }
  .tile-main:hover {
    transform: translateY(-1px);
    border-color: var(--accent-border);
    background: var(--surface-2);
    box-shadow: 0 12px 28px rgba(0, 0, 0, 0.16);
  }
  .top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 37px;
  }
  .symbol {
    display: grid;
    place-items: center;
    width: 37px;
    height: 37px;
    border-radius: 10px;
    color: var(--accent-text);
    background: var(--accent-soft);
    border: 1px solid var(--accent-border);
  }
  .pill {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: var(--accent-text);
    transition: opacity 0.15s;
  }
  .pill i {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--success);
    box-shadow: 0 0 8px rgba(75, 209, 155, 0.45);
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 20px;
    min-width: 0;
  }
  .info strong {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-strong);
  }
  .info small {
    font: 11px var(--font-mono);
    color: var(--text-faint);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: auto;
    padding-top: 15px;
    border-top: 1px solid var(--border);
    font-size: 11px;
    color: var(--text-faint);
  }
  .os {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .go {
    display: flex;
    align-items: center;
    gap: 2px;
    flex: none;
    color: var(--accent-text);
  }

  .actions {
    position: absolute;
    top: 14px;
    right: 14px;
    display: none;
    gap: 2px;
  }
  .tile:hover .actions,
  .tile:focus-within .actions {
    display: flex;
  }
  .tile:hover .pill,
  .tile:focus-within .pill {
    opacity: 0;
  }

  .recent {
    margin-top: 38px;
  }
  .table {
    border: 1px solid var(--border);
    border-radius: 10px;
    background: var(--card);
    overflow: hidden;
  }
  .row {
    display: grid;
    grid-template-columns:
      32px minmax(150px, 1.5fr) minmax(80px, 1fr) minmax(100px, 0.8fr)
      16px;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 58px;
    padding: 8px 16px;
    border: none;
    background: none;
    color: var(--text-faint);
    text-align: left;
    transition: background 0.15s;
  }
  .row + .row {
    border-top: 1px solid var(--border);
  }
  .row:hover {
    background: var(--surface-2);
  }
  .ricon {
    display: grid;
    place-items: center;
    width: 30px;
    height: 30px;
    border-radius: 8px;
    color: var(--accent-text);
    background: var(--accent-soft);
  }
  .rname {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }
  .rname strong {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
  }
  .rname small,
  time {
    font: 10.5px var(--font-mono);
    color: var(--text-faint);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ros {
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .blank {
    display: grid;
    justify-items: center;
    gap: 12px;
    padding: 60px 20px;
    text-align: center;
  }
  .blank-icon {
    display: grid;
    place-items: center;
    width: 50px;
    height: 50px;
    color: var(--accent-text);
    border: 1px solid var(--accent-border);
    border-radius: 14px;
    background: var(--accent-soft);
  }
  .blank h2 {
    font-size: 17px;
    font-weight: 600;
    color: var(--text-strong);
  }
  .blank p {
    max-width: 380px;
    color: var(--text-dim);
    font-size: 13px;
    line-height: 1.6;
  }
  code {
    font-family: var(--font-mono);
    font-size: 12px;
  }

  .security {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    margin-top: 30px;
    font-size: 11px;
    color: var(--text-faint);
  }
  .security :global(svg) {
    color: var(--accent-line);
  }

  @media (max-width: 760px) {
    .inner {
      width: calc(100% - 40px);
      padding-top: 32px;
    }
    header {
      flex-direction: column;
      align-items: flex-start;
    }
    .row {
      grid-template-columns: 32px minmax(120px, 1fr) 16px;
    }
    .ros,
    time {
      display: none;
    }
  }
</style>
