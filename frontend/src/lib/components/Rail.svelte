<script lang="ts">
  import { app, type Page } from "../stores/app.svelte";
  import { tabs } from "../stores/tabs.svelte";
  import Icon, { type IconName } from "./Icon.svelte";

  const items: { page: Page; label: string; icon: IconName }[] = [
    { page: "hosts", label: "Hosts", icon: "server" },
    { page: "terminal", label: "Terminal", icon: "terminal" },
    { page: "sftp", label: "SFTP", icon: "folder" },
    { page: "keys", label: "Keys", icon: "key" },
  ];
</script>

<aside class="rail">
  <button
    class="brand"
    onclick={() => app.go("hosts")}
    aria-label="HandSSHake home"
    title="HandSSHake"
  >
    <Icon name="terminal" size={21} />
  </button>

  <nav aria-label="Primary navigation">
    {#each items as it (it.page)}
      <button
        class="item"
        class:active={app.page === it.page}
        onclick={() => app.go(it.page)}
        title={it.label}
        aria-label={it.label}
        aria-current={app.page === it.page ? "page" : undefined}
      >
        <Icon name={it.icon} size={18} />
        <span>{it.label}</span>
        {#if it.page === "terminal" && tabs.items.length > 0}<i
            class="live"
            title="Sessions open"
          ></i>{/if}
      </button>
    {/each}
  </nav>

  <div class="bottom">
    <button
      class="item"
      class:active={app.page === "settings"}
      onclick={() => app.go("settings")}
      title="Settings (Ctrl+,)"
      aria-label="Settings"
      aria-current={app.page === "settings" ? "page" : undefined}
    >
      <Icon name="settings" size={18} />
      <span>Settings</span>
    </button>
    <button
      class="lock"
      onclick={() => app.lock()}
      title="Lock vault"
      aria-label="Lock vault"
    >
      <Icon name="lock" size={16} />
    </button>
  </div>
</aside>

<style>
  .rail {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 18px 10px;
    background: var(--rail-bg);
    border-right: 1px solid var(--border);
  }
  .brand {
    display: grid;
    place-items: center;
    width: 40px;
    height: 40px;
    border: none;
    border-radius: 11px;
    color: #fff;
    background: var(--brand-gradient);
    box-shadow:
      0 8px 24px var(--brand-glow),
      inset 0 1px rgba(255, 255, 255, 0.18);
  }
  nav {
    display: flex;
    flex-direction: column;
    gap: 7px;
    width: 100%;
    margin-top: 38px;
  }
  .bottom {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
    width: 100%;
    margin-top: auto;
  }

  .item {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 5px;
    width: 100%;
    height: 54px;
    border: none;
    border-radius: 10px;
    background: none;
    color: var(--text-faint);
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.02em;
    transition:
      color 0.15s,
      background 0.15s;
  }
  .item:hover {
    color: var(--text);
    background: var(--surface-2);
  }
  .item.active {
    color: var(--accent-text);
    background: var(--accent-soft);
  }
  .item.active::before {
    content: "";
    position: absolute;
    left: -10px;
    width: 2px;
    height: 25px;
    border-radius: 2px;
    background: var(--accent-line);
    box-shadow: 0 0 10px var(--accent-border);
  }
  .live {
    position: absolute;
    top: 9px;
    right: 17px;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--success);
    box-shadow: 0 0 8px rgba(75, 209, 155, 0.6);
  }

  .lock {
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    border: 1px solid var(--border-strong);
    border-radius: 10px;
    color: var(--text-dim);
    background: var(--surface-2);
    transition:
      color 0.15s,
      border-color 0.15s,
      background 0.15s;
  }
  .lock:hover {
    color: var(--text-strong);
    background: var(--surface-3);
  }
</style>
