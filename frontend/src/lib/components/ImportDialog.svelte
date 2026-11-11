<script lang="ts">
  import { onMount } from "svelte";
  import { app } from "../stores/app.svelte";
  import { api, errText } from "../api";
  import type { ImportCandidate, ImportPreview } from "../types";
  import Modal from "./Modal.svelte";
  import Icon from "./Icon.svelte";

  let preview = $state<ImportPreview | null>(null);
  let loading = $state(true);
  let loadError = $state("");
  let error = $state("");
  let picked = $state<Record<string, boolean>>({});
  let copyKeys = $state(false);
  let busy = $state(false);

  const importable = $derived(
    preview?.candidates.filter((c) => c.status === "new") ?? [],
  );
  const count = $derived(importable.filter((c) => picked[c.alias]).length);
  const allOn = $derived(importable.length > 0 && count === importable.length);
  const anyKeys = $derived(
    importable.some(
      (c) =>
        picked[c.alias] &&
        (c.keyStatus === "ok" || c.keyStatus === "encrypted"),
    ),
  );

  const base = (p: string) => p.split("/").pop() ?? p;

  function chip(c: ImportCandidate): {
    text: string;
    tone: "ok" | "warn" | "bad";
  } {
    switch (c.keyStatus) {
      case "ok":
        return { text: base(c.keyPath), tone: "ok" };
      case "encrypted":
        return { text: `${base(c.keyPath)} · needs passphrase`, tone: "warn" };
      case "missing":
        return { text: `${base(c.keyPath)} · file not found`, tone: "bad" };
      case "invalid":
        return { text: `${base(c.keyPath)} · not a private key`, tone: "bad" };
      default:
        return {
          text: "No key found · add a password after importing",
          tone: "warn",
        };
    }
  }

  async function load(path: string) {
    loading = true;
    loadError = "";
    error = "";
    try {
      const pv = await api.previewImport(path);
      preview = pv;
      picked = Object.fromEntries(
        pv.candidates
          .filter((c) => c.status === "new")
          .map((c) => [c.alias, true]),
      );
    } catch (e) {
      preview = null;
      loadError = errText(e);
    } finally {
      loading = false;
    }
  }

  async function choose() {
    try {
      const p = await api.pickConfigFile();
      if (p) await load(p);
    } catch (e) {
      loadError = errText(e);
    }
  }

  function toggleAll() {
    const on = !allOn;
    picked = Object.fromEntries(importable.map((c) => [c.alias, on]));
  }

  async function run() {
    if (!preview) return;
    error = "";
    busy = true;
    try {
      const aliases = importable
        .filter((c) => picked[c.alias])
        .map((c) => c.alias);
      const n = await api.importHosts(preview.path, aliases, copyKeys);
      await app.refreshHosts();
      app.toast("success", `Imported ${n} host${n === 1 ? "" : "s"}`);
      app.importOpen = false;
    } catch (e) {
      error = errText(e);
    } finally {
      busy = false;
    }
  }

  onMount(() => load(""));
</script>

<Modal
  title="Import from SSH config"
  width={640}
  onclose={() => (app.importOpen = false)}
>
  {#if loading}
    <div class="center"><div class="spinner"></div></div>
  {:else if loadError}
    <div class="center">
      <span class="badge"><Icon name="alert" size={22} /></span>
      <p class="msg selectable">{loadError}</p>
      <button class="btn" onclick={choose}
        ><Icon name="folder" /> Choose another file…</button
      >
    </div>
  {:else if preview}
    <div class="source">
      <span class="path selectable" title={preview.path}>{preview.path}</span>
      <button class="btn small" onclick={choose}>Choose file…</button>
    </div>

    {#if preview.candidates.length === 0}
      <p class="empty">
        No named hosts were found in this file. Wildcard entries like <code
          >Host *</code
        > are skipped.
      </p>
    {:else}
      <div class="bar">
        <button
          class="link"
          onclick={toggleAll}
          disabled={importable.length === 0}
        >
          {allOn ? "Select none" : "Select all"}
        </button>
        <span class="hint"
          >{count} of {importable.length} importable selected</span
        >
      </div>

      <ul>
        {#each preview.candidates as c (c.alias)}
          {@const usable = c.status === "new"}
          <li>
            <label class="item" class:off={!usable}>
              <input
                type="checkbox"
                bind:checked={picked[c.alias]}
                disabled={!usable}
              />
              <span class="body">
                <span class="top">
                  <strong>{c.alias}</strong>
                  <span class="addr selectable">{c.user}@{c.address}</span>
                </span>
                {#if usable}
                  {@const k = chip(c)}
                  <span class="chip {k.tone}">
                    <Icon
                      name={c.keyStatus === "none" ? "lock" : "key"}
                      size={12}
                    />
                    {k.text}
                  </span>
                {:else}
                  <span class="note">{c.note}</span>
                {/if}
              </span>
            </label>
          </li>
        {/each}
      </ul>

      <label class="checkbox copy">
        <input type="checkbox" bind:checked={copyKeys} />
        <span>
          Copy private keys into the encrypted vault. The vault then has
          everything it needs, even if the key files are later moved or deleted.
          {#if copyKeys && anyKeys}<em
              >The keys will also stay on disk where they are.</em
            >{/if}
        </span>
      </label>
    {/if}

    {#if preview.warnings.length > 0}
      <details class="warn">
        <summary
          >{preview.warnings.length} note{preview.warnings.length === 1
            ? ""
            : "s"} from reading the file</summary
        >
        <ul class="plain">
          {#each preview.warnings as w}<li>{w}</li>{/each}
        </ul>
      </details>
    {/if}

    {#if error}<p class="error-text">{error}</p>{/if}

    <div class="actions">
      <button class="btn" onclick={() => (app.importOpen = false)}
        >Cancel</button
      >
      <button
        class="btn btn-primary"
        disabled={busy || count === 0}
        onclick={run}
      >
        {busy
          ? "Importing…"
          : count === 0
            ? "Import"
            : `Import ${count} host${count === 1 ? "" : "s"}`}
      </button>
    </div>
  {/if}
</Modal>

<style>
  .center {
    display: grid;
    justify-items: center;
    gap: 14px;
    padding: 26px 10px;
    text-align: center;
  }
  .spinner {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    border: 2px solid var(--border-strong);
    border-top-color: var(--accent);
    animation: spin 0.8s linear infinite;
  }
  .badge {
    display: grid;
    place-items: center;
    width: 46px;
    height: 46px;
    border-radius: 14px;
    background: var(--danger-soft);
    color: var(--danger);
  }
  .msg {
    max-width: 440px;
    font-size: 13px;
    line-height: 1.55;
    color: var(--text-dim);
    word-break: break-word;
  }

  .source {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 14px;
  }
  .path {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--font-mono);
    font-size: 12.5px;
    color: var(--text-dim);
  }
  .small {
    height: 30px;
    padding: 0 12px;
    font-size: 13px;
  }
  .empty {
    padding: 18px 4px;
    color: var(--text-dim);
    line-height: 1.55;
  }
  code {
    font-family: var(--font-mono);
    font-size: 12.5px;
  }

  .bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }
  .link {
    background: none;
    border: none;
    padding: 2px 0;
    color: var(--accent);
    font-size: 13px;
    cursor: pointer;
  }
  .link:hover:not(:disabled) {
    text-decoration: underline;
  }
  .link:disabled {
    color: var(--text-faint);
    cursor: default;
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    max-height: 330px;
    overflow-y: auto;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
  }
  li + li {
    border-top: 1px solid var(--border);
  }
  .item {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 11px 14px;
    cursor: pointer;
  }
  .item:hover {
    background: var(--surface-2);
  }
  .item.off {
    cursor: default;
    opacity: 0.6;
  }
  .item.off:hover {
    background: none;
  }
  .item input {
    margin-top: 3px;
    accent-color: var(--accent);
  }
  .body {
    display: grid;
    gap: 5px;
    min-width: 0;
  }
  .top {
    display: flex;
    align-items: baseline;
    gap: 10px;
    min-width: 0;
  }
  .addr {
    color: var(--text-faint);
    font-size: 12.5px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .note {
    font-size: 12.5px;
    color: var(--text-faint);
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    justify-self: start;
    padding: 2px 9px;
    border-radius: 99px;
    font-size: 12px;
  }
  .chip.ok {
    background: var(--surface-3);
    color: var(--text-dim);
  }
  .chip.warn {
    background: rgba(251, 191, 36, 0.12);
    color: var(--warning);
  }
  .chip.bad {
    background: var(--danger-soft);
    color: var(--danger);
  }

  .copy {
    margin: 16px 0 4px;
    line-height: 1.5;
  }
  .copy em {
    display: block;
    margin-top: 4px;
    font-style: normal;
    color: var(--text-faint);
  }
  .warn {
    margin-top: 12px;
    font-size: 12.5px;
    color: var(--text-faint);
  }
  .warn summary {
    cursor: pointer;
  }
  .plain {
    max-height: none;
    margin-top: 6px;
    padding-left: 16px;
    list-style: disc;
    border: none;
    background: none;
  }
  .plain li {
    border: none;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    margin-top: 18px;
  }
</style>
