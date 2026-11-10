<script lang="ts">
  import { tabs } from "../stores/tabs.svelte";
  import OsBadge from "./OsBadge.svelte";
  import Icon from "./Icon.svelte";

  const label = {
    connecting: "Connecting…",
    connected: "Connected",
    ended: "Session ended",
    error: "Connection failed",
  };
</script>

<div class="bar" role="tablist">
  {#each tabs.items as t (t.id)}
    <div
      class="tab"
      class:active={t.id === tabs.activeId}
      class:dim={t.status === "ended" || t.status === "error"}
      title="{t.subtitle}{t.osName ? ' · ' + t.osName : ''} · {label[t.status]}"
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
        <span class="icon">
          <OsBadge os={t.os} like={t.osLike} size={20} />
          <i class="dot {t.status}"></i>
        </span>
        <span class="title">{t.title}</span>
      </button>
      <button
        class="x"
        onclick={() => tabs.close(t.id)}
        aria-label="Close {t.title}"
      >
        <Icon name="x" size={12} />
      </button>
    </div>
  {/each}
</div>

<style>
  .bar {
    display: flex;
    align-items: flex-end;
    gap: 4px;
    height: 44px;
    padding: 8px 10px 0;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
    overflow-x: auto;
    overflow-y: hidden;
  }
  .bar::-webkit-scrollbar {
    height: 0;
  }

  .tab {
    --tab-bg: var(--surface);
    position: relative;
    display: flex;
    align-items: center;
    flex: none;
    min-width: 140px;
    max-width: 240px;
    height: 36px;
    border-radius: 10px 10px 0 0;
    color: var(--text-dim);
    transition:
      background 0.12s,
      color 0.12s;
  }
  .tab:hover {
    --tab-bg: var(--surface-2);
    background: var(--tab-bg);
  }
  .tab.active {
    --tab-bg: var(--term-bg);
    background: var(--tab-bg);
    color: var(--text);
  }
  .tab.active::after {
    content: "";
    position: absolute;
    top: 0;
    left: 14px;
    right: 14px;
    height: 2px;
    border-radius: 0 0 2px 2px;
    background: var(--accent);
  }
  .tab.dim .title {
    opacity: 0.55;
  }

  .label {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0 2px 0 12px;
    border: none;
    background: none;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }
  .title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }

  .icon {
    position: relative;
    flex: none;
    display: grid;
  }
  .dot {
    position: absolute;
    right: -4px;
    bottom: -4px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    border: 2px solid var(--tab-bg);
    background: var(--text-faint);
  }
  .dot.connected {
    background: var(--success);
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
    width: 22px;
    height: 22px;
    margin-right: 8px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-faint);
    cursor: pointer;
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
</style>
