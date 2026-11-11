<script lang="ts">
  import { app } from "../stores/app.svelte";
  import { tabs } from "../stores/tabs.svelte";
  import { api, errText } from "../api";
  import type { HostInfo } from "../types";
  import Icon from "./Icon.svelte";
  import OsBadge from "./OsBadge.svelte";

  let search: HTMLInputElement;

  const hue = (s: string) => {
    let h = 0;
    for (const c of s) h = (h * 31 + c.charCodeAt(0)) % 360;
    return h;
  };
  const initial = (s: string) => (s.trim()[0] ?? "?").toUpperCase();
  const isLive = (id: string) =>
    tabs.items.some((t) => t.hostId === id && t.status === "connected");

  function askDelete(h: HostInfo) {
    app.confirm = {
      title: "Delete host?",
      message: `“${h.name}” and its saved credentials will be permanently removed from your vault.`,
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

  function onWindowKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      search?.focus();
      search?.select();
    }
  }
</script>

<svelte:window onkeydown={onWindowKey} />

<aside>
  <div class="head">
    <div class="brand">
      <span class="mark"><Icon name="terminal" size={15} /></span>
      <span>HandSSHake</span>
    </div>
    <span class="headbtns">
      <button
        class="icon-btn"
        onclick={() => (app.importOpen = true)}
        title="Import from SSH config"
        aria-label="Import hosts"
      >
        <Icon name="download" size={16} />
      </button>
      <button
        class="btn btn-primary add"
        onclick={() => app.openEditor()}
        title="Add host"
      >
        <Icon name="plus" size={15} /> New
      </button>
    </span>
  </div>

  <div class="search">
    <Icon name="search" size={15} />
    <input
      bind:this={search}
      bind:value={app.query}
      placeholder="Search hosts"
      spellcheck="false"
      autocomplete="off"
      onkeydown={(e) => {
        if (e.key === "Enter" && app.filtered[0]) tabs.open(app.filtered[0]);
      }}
    />
    <kbd>Ctrl K</kbd>
  </div>

  <ul>
    {#each app.filtered as h (h.id)}
      <li class="host">
        <button
          class="main"
          onclick={() => tabs.open(h)}
          title="Connect to {h.name}"
        >
          {#if h.os}
            <OsBadge os={h.os} like={h.osLike} size={34} />
          {:else}
            <span
              class="avatar"
              style:background={`hsl(${hue(h.name)} 42% 26%)`}
              style:color={`hsl(${hue(h.name)} 85% 82%)`}
            >
              {initial(h.name)}
            </span>
          {/if}
          <span class="meta">
            <span class="name">{h.name}</span>
            <span class="addr">{h.user}@{h.address}</span>
          </span>
          {#if isLive(h.id)}<span class="live" title="Connected"></span>{/if}
        </button>
        <span class="actions">
          <button
            class="icon-btn"
            onclick={() => app.openEditor(h.id)}
            aria-label="Edit {h.name}"><Icon name="pencil" size={14} /></button
          >
          <button
            class="icon-btn"
            onclick={() => askDelete(h)}
            aria-label="Delete {h.name}"><Icon name="trash" size={14} /></button
          >
        </span>
      </li>
    {/each}
  </ul>

  {#if app.hosts.length === 0}
    <div class="empty">
      <Icon name="server" size={26} />
      <p>No hosts yet</p>
      <button class="btn" onclick={() => app.openEditor()}
        >Add your first host</button
      >
      <button class="btn" onclick={() => (app.importOpen = true)}>Import from ~/.ssh/config</button>
    </div>
  {:else if app.filtered.length === 0}
    <div class="empty"><p>No hosts match “{app.query}”</p></div>
  {/if}

  <div class="foot">
    <span class="count"
      >{app.hosts.length} {app.hosts.length === 1 ? "host" : "hosts"}</span
    >
    <span class="footbtns">
      <button
        class="icon-btn"
        onclick={() => (app.settingsOpen = true)}
        title="Settings (Ctrl+,)"
        aria-label="Settings"
      >
        <Icon name="sliders" size={16} />
      </button>
      <button class="btn lockbtn" onclick={() => app.lock()}
        ><Icon name="lock" size={14} /> Lock</button
      >
    </span>
  </div>
</aside>

<style>
  aside {
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: var(--surface);
    border-right: 1px solid var(--border);
  }
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 14px 10px 16px;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 9px;
    font-weight: 650;
    letter-spacing: -0.01em;
  }
  .mark {
    display: grid;
    place-items: center;
    width: 26px;
    height: 26px;
    border-radius: 8px;
    color: #fff;
    background: linear-gradient(135deg, #5b7cfa, #8b5cf6);
  }
  .add {
    height: 30px;
    padding: 0 11px;
    font-size: 13px;
  }

  .search {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 4px 14px 10px;
    padding: 0 10px;
    height: 36px;
    color: var(--text-faint);
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    transition:
      border-color 0.12s,
      box-shadow 0.12s;
  }
  .search:focus-within {
    border-color: var(--accent);
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
  }
  .search input::placeholder {
    color: var(--text-faint);
  }
  kbd {
    font: 500 10.5px var(--font-ui);
    color: var(--text-faint);
    padding: 2px 6px;
    border: 1px solid var(--border-strong);
    border-radius: 5px;
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0 8px;
    flex: 1;
    overflow-y: auto;
  }
  .host {
    position: relative;
    border-radius: var(--radius-sm);
  }
  .host:hover {
    background: var(--surface-2);
  }
  .main {
    display: flex;
    align-items: center;
    gap: 11px;
    width: 100%;
    padding: 8px 10px;
    border: none;
    background: none;
    text-align: left;
    cursor: pointer;
    border-radius: var(--radius-sm);
  }
  .avatar {
    flex: none;
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    border-radius: 10px;
    font-weight: 650;
    font-size: 14px;
  }
  .meta {
    display: grid;
    min-width: 0;
  }
  .name {
    font-weight: 550;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .addr {
    font-size: 12px;
    color: var(--text-faint);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .live {
    margin-left: auto;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--success);
    box-shadow: 0 0 8px var(--success);
  }
  .actions {
    position: absolute;
    right: 6px;
    top: 50%;
    transform: translateY(-50%);
    display: none;
    gap: 2px;
    padding-left: 20px;
    background: linear-gradient(90deg, transparent, var(--surface-2) 30%);
  }
  .host:hover .actions {
    display: flex;
  }
  .host:hover .live {
    display: none;
  }

  .empty {
    display: grid;
    justify-items: center;
    gap: 10px;
    padding: 30px 20px;
    color: var(--text-faint);
    text-align: center;
  }
  .foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    border-top: 1px solid var(--border);
  }
  .count {
    font-size: 12px;
    color: var(--text-faint);
  }
  .lockbtn {
    height: 30px;
    padding: 0 11px;
    font-size: 13px;
  }

  .footbtns {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .headbtns {
    display: flex;
    align-items: center;
    gap: 4px;
  }
</style>
