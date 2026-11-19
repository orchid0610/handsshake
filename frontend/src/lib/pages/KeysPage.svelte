<script lang="ts">
  import { app } from "../stores/app.svelte";
  import { api, errText } from "../api";
  import type { KeyInfo } from "../types";
  import Icon from "../components/Icon.svelte";
  import EmptyPage from "../components/EmptyPage.svelte";

  let keys = $state<KeyInfo[]>([]);
  let loaded = $state(false);
  let token = 0; // the newest request wins, if two overlap

  async function load() {
    const mine = ++token;
    try {
      const list = (await api.listKeys()) ?? [];
      if (mine === token) {
        keys = list;
        loaded = true;
      }
    } catch (e) {
      if (mine === token) app.toast("error", errText(e));
    }
  }

  // Runs when the page opens, and again whenever the host list changes.
  $effect(() => {
    void app.hosts;
    load();
  });

  async function copy(fp: string) {
    try {
      await navigator.clipboard.writeText(fp);
      app.toast("success", "Fingerprint copied");
    } catch {
      app.toast(
        "error",
        "Could not copy automatically. Select the fingerprint and copy it by hand.",
      );
    }
  }

  type Tone = "ok" | "warn" | "bad";

  function getStatusInfo(k: KeyInfo): { text: string; tone: Tone } {
    switch (k.status) {
      case "locked":
        return { text: "Passphrase needed", tone: "warn" };
      case "missing":
        return { text: "File not found", tone: "bad" };
      case "invalid":
        return { text: "Not a valid key", tone: "bad" };
      default:
        return { text: k.encrypted ? "Passphrase saved" : "Ready", tone: "ok" };
    }
  }

  function hint(k: KeyInfo): string {
    switch (k.status) {
      case "locked":
        return "This key is passphrase-protected. Edit the host and save the passphrase, or connecting will fail.";
      case "missing":
        return "The key file is no longer there. Edit the host to point it at the right file.";
      case "invalid":
        return "This file isn't a private key that SSH can read.";
      default:
        return "";
    }
  }
</script>

{#if loaded && keys.length === 0}
  <EmptyPage
    icon="key"
    eyebrow="Authentication"
    title="Keys"
    text="Hosts that sign in with a private key will show their keys here."
    action="Go to hosts"
    onaction={() => app.go("hosts")}
  />
{:else if loaded}
  <div class="inner">
    <header>
      <p class="eyebrow">Authentication</p>
      <h1 class="page-title">Keys</h1>
      <p class="page-subtitle">The SSH keys your hosts use to sign in.</p>
    </header>

    <div class="section-title">
      <h2>Keys in use</h2>
      <span>{keys.length} key{keys.length === 1 ? "" : "s"}</span>
    </div>

    <ul>
      {#each keys as k (k.id)}
        {@const st = getStatusInfo(k)}
        <li class="card">
          <span class="kicon"><Icon name="key" size={19} /></span>
          <div class="body">
            <div class="top">
              <strong>{k.name}</strong>
              {#if k.type}<span class="chip">{k.type}</span>{/if}
              <span class="state {st.tone}"><i></i>{st.text}</span>
            </div>

            <p class="where">
              {#if k.source === "vault"}<Icon name="lock" size={12} /> Stored in
                your encrypted vault
              {:else}<span class="selectable">{k.path}</span>{/if}
            </p>

            {#if k.fingerprint}
              <div class="fp">
                <code class="selectable">{k.fingerprint}</code>
                <button
                  class="icon-btn"
                  onclick={() => copy(k.fingerprint)}
                  aria-label="Copy fingerprint"
                  title="Copy fingerprint"
                >
                  <Icon name="copy" size={14} />
                </button>
              </div>
            {:else}
              <p class="nofp">Fingerprint unavailable.</p>
            {/if}

            {#if hint(k)}<p class="hintline {st.tone}">{hint(k)}</p>{/if}

            <p class="used">
              <span>Used by</span>
              {#each k.usedBy.slice(0, 4) as name}<em>{name}</em>{/each}
              {#if k.usedBy.length > 4}<em class="more"
                  >+{k.usedBy.length - 4} more</em
                >{/if}
            </p>
          </div>
        </li>
      {/each}
    </ul>

    <p class="security">
      <Icon name="shield" size={15} /> Private keys never leave this device. Only
      fingerprints are shown here.
    </p>
  </div>
{/if}

<style>
  .inner {
    width: min(1080px, calc(100% - 88px));
    margin: 0 auto;
    padding: 52px 0 34px;
  }
  header {
    margin-bottom: 34px;
  }
  .eyebrow {
    margin-bottom: 4px;
  }
  .page-subtitle {
    margin-top: 7px;
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

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 10px;
  }
  .card {
    display: flex;
    gap: 16px;
    padding: 18px;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--card);
    transition: border-color 0.18s;
  }
  .card:hover {
    border-color: var(--accent-border);
  }
  .kicon {
    display: grid;
    place-items: center;
    flex: none;
    width: 40px;
    height: 40px;
    border-radius: 11px;
    color: var(--accent-text);
    background: var(--accent-soft);
    border: 1px solid var(--accent-border);
  }
  .body {
    display: grid;
    gap: 8px;
    min-width: 0;
    flex: 1;
  }
  .top {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
  }
  .top strong {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-strong);
  }
  .chip {
    padding: 2px 8px;
    border-radius: 99px;
    font: 500 11px var(--font-mono);
    color: var(--text-dim);
    background: var(--surface-3);
  }

  .state {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
    font-size: 11px;
  }
  .state i {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }
  .state.ok {
    color: var(--accent-text);
  }
  .state.ok i {
    background: var(--success);
    box-shadow: 0 0 8px rgba(75, 209, 155, 0.45);
  }
  .state.warn {
    color: var(--warning);
  }
  .state.warn i {
    background: var(--warning);
  }
  .state.bad {
    color: var(--danger);
  }
  .state.bad i {
    background: var(--danger);
  }

  .where {
    display: flex;
    align-items: center;
    gap: 6px;
    font: 11.5px var(--font-mono);
    color: var(--text-faint);
  }
  .fp {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }
  .fp code {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: 11.5px var(--font-mono);
    color: var(--text-dim);
  }
  .nofp {
    font-size: 12px;
    color: var(--text-faint);
  }
  .hintline {
    font-size: 12px;
    line-height: 1.5;
  }
  .hintline.warn {
    color: var(--warning);
  }
  .hintline.bad {
    color: var(--danger);
  }

  .used {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
    font-size: 11px;
    color: var(--text-faint);
  }
  .used span {
    margin-right: 2px;
  }
  .used em {
    padding: 2px 8px;
    border-radius: 6px;
    font-style: normal;
    color: var(--text-dim);
    background: var(--surface-2);
    border: 1px solid var(--border);
  }
  .used em.more {
    background: none;
    border-style: dashed;
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
    .state {
      margin-left: 0;
    }
  }
</style>
