<script lang="ts">
  import { tick } from "svelte";
  import { app } from "../stores/app.svelte";
  import { snippets } from "../stores/snippets.svelte";
  import Icon from "./Icon.svelte";

  let searchEl = $state<HTMLDivElement>();
  let titleEl = $state<HTMLDivElement>();

  // Put the cursor in the right field when the drawer or its form opens.
  $effect(() => {
    if (!snippets.open) return;
    const editing = snippets.form !== null;
    tick().then(() => (editing ? titleEl : searchEl)?.focus());
  });

  function onKey(e: KeyboardEvent) {
    if (!snippets.open || e.key !== "Escape" || app.confirm) return; // a confirm dialog gets the first Escape
    if (snippets.form) snippets.cancelForm();
    else snippets.close();
  }

  function onCodeKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      snippets.save();
    }
  }

  const lineCount = (code: string) => code.split("\n").length;
</script>

<svelte:window onkeydown={onKey} />

{#if snippets.open}
  <button
    class="scrim"
    tabindex="-1"
    aria-label="Close snippets"
    onclick={() => snippets.close()}
  ></button>

  <aside class="drawer" id="snippets-drawer" aria-label="Saved snippets">
    <header>
      <div class="who">
        <span class="hicon"><Icon name="code" size={16} /></span>
        <div>
          <h2>Snippets</h2>
          <p>{snippets.items.length} saved · encrypted in your vault</p>
        </div>
      </div>
      <button
        class="icon-btn"
        onclick={() => snippets.close()}
        aria-label="Close snippets"><Icon name="x" size={16} /></button
      >
    </header>

    {#if snippets.form}
      {@const f = snippets.form}
      <form
        onsubmit={(e) => {
          e.preventDefault();
          snippets.save();
        }}
      >
        <div class="fhead">
          <div>
            <h3>{f.id ? "Edit snippet" : "New snippet"}</h3>
            <p>Save a command or a reusable block of code.</p>
          </div>
          <button
            type="button"
            class="icon-btn"
            onclick={() => snippets.cancelForm()}
            aria-label="Cancel"><Icon name="x" size={15} /></button
          >
        </div>

        <div class="field">
          <label for="sn-title">Name</label>
          <input
            id="sn-title"
            class="input mono"
            bind:this={titleEl}
            bind:value={f.title}
            maxlength="80"
            placeholder="e.g. Restart web service"
            autocomplete="off"
            spellcheck="false"
          />
        </div>
        <div class="field">
          <label for="sn-code">Command or code</label>
          <textarea
            id="sn-code"
            class="input code"
            bind:value={f.code}
            onkeydown={onCodeKey}
            rows="8"
            placeholder="sudo systemctl restart nginx"
            spellcheck="false"></textarea>
          <span class="hint"
            >Pasting never runs it: you press Enter yourself. Ctrl+Enter saves.</span
          >
        </div>

        {#if snippets.formError}<p class="error-text">
            {snippets.formError}
          </p>{/if}

        <div class="actions">
          <button
            type="button"
            class="text-btn"
            onclick={() => snippets.cancelForm()}>Cancel</button
          >
          <button class="btn btn-primary" disabled={snippets.busy}>
            {snippets.busy ? "Saving…" : f.id ? "Save changes" : "Save snippet"}
          </button>
        </div>
      </form>
    {:else}
      <div class="tools">
        <label class="search">
          <Icon name="search" size={14} />
          <input
            bind:this={searchEl}
            bind:value={snippets.query}
            placeholder="Search snippets…"
            spellcheck="false"
            autocomplete="off"
          />
        </label>
        <button class="btn btn-primary new" onclick={() => snippets.startNew()}>
          <Icon name="plus" size={14} /> New
        </button>
      </div>

      <div class="list">
        {#each snippets.filtered as s (s.id)}
          <article class="card">
            <div class="ctitle">
              <h3>{s.title}</h3>
              <div class="cbtns">
                {#if lineCount(s.code) > 1}<span class="chip"
                    >{lineCount(s.code)} lines</span
                  >{/if}
                <button
                  class="icon-btn"
                  onclick={() => snippets.startEdit(s)}
                  aria-label="Edit {s.title}"
                  title="Edit"><Icon name="pencil" size={13} /></button
                >
                <button
                  class="icon-btn del"
                  onclick={() => snippets.remove(s)}
                  aria-label="Delete {s.title}"
                  title="Delete"><Icon name="trash" size={13} /></button
                >
              </div>
            </div>
            <pre class="selectable">{s.code}</pre>
            <button class="paste" onclick={() => snippets.paste(s)}>
              <Icon name="copy" size={13} /> Paste to terminal
            </button>
          </article>
        {:else}
          {#if snippets.loaded && snippets.items.length === 0}
            <div class="empty">
              <span class="eicon"><Icon name="code" size={20} /></span>
              <strong>No snippets yet</strong>
              <span
                >Save a command you use often, then paste it into any session
                with one click.</span
              >
              <button class="btn" onclick={() => snippets.startNew()}
                ><Icon name="plus" size={14} /> New snippet</button
              >
            </div>
          {:else if snippets.loaded}
            <div class="empty">
              <span class="eicon"><Icon name="search" size={20} /></span>
              <strong>No snippets found</strong>
              <span>Try another search or save a new snippet.</span>
            </div>
          {/if}
        {/each}
      </div>
    {/if}
  </aside>
{/if}

<style>
  .scrim {
    position: absolute;
    inset: 0;
    z-index: 3;
    padding: 0;
    border: none;
    cursor: default;
    background: rgba(0, 0, 0, 0.5);
    animation: fade 0.16s ease;
  }
  .drawer {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    z-index: 4;
    display: flex;
    flex-direction: column;
    width: min(390px, calc(100% - 24px));
    background: var(--card);
    color: var(--text);
    border-left: 1px solid var(--accent-border);
    box-shadow: -24px 0 60px rgba(0, 0, 0, 0.4);
    animation: slide 0.2s ease;
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @keyframes slide {
    from {
      opacity: 0;
      transform: translateX(18px);
    }
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 68px;
    padding: 14px 16px;
    border-bottom: 1px solid var(--border);
  }
  .who {
    display: flex;
    align-items: center;
    gap: 11px;
  }
  .hicon {
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    color: var(--accent-text);
    border: 1px solid var(--accent-border);
    border-radius: 9px;
    background: var(--accent-soft);
  }
  h2 {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-strong);
  }
  header p,
  .fhead p {
    margin-top: 4px;
    font-size: 11px;
    color: var(--text-faint);
  }

  .tools {
    display: flex;
    gap: 8px;
    padding: 12px 14px;
    border-bottom: 1px solid var(--border);
  }
  .search {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
    height: 35px;
    padding: 0 10px;
    color: var(--text-faint);
    background: var(--input-bg);
    border: 1px solid var(--border-strong);
    border-radius: 8px;
    transition: border-color 0.15s;
  }
  .search:focus-within {
    border-color: var(--accent-border);
  }
  .search input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--text);
    font: inherit;
    font-size: 12px;
  }
  .search input::placeholder {
    color: var(--text-faint);
  }
  .new {
    height: 35px;
    padding: 0 12px;
    font-size: 12px;
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
    min-height: 0;
    padding: 12px 14px 20px;
    overflow-y: auto;
  }
  .card {
    padding: 12px;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: var(--surface);
    transition: border-color 0.18s;
  }
  .card:hover {
    border-color: var(--accent-border);
  }
  .ctitle {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
  }
  .ctitle h3 {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-strong);
  }
  .cbtns {
    display: flex;
    align-items: center;
    gap: 2px;
    flex: none;
  }
  .cbtns .icon-btn {
    width: 25px;
    height: 25px;
  }
  .cbtns .del:hover {
    color: var(--danger);
  }
  .chip {
    margin-right: 4px;
    padding: 1px 7px;
    border-radius: 99px;
    font-size: 10px;
    color: var(--text-faint);
    background: var(--surface-3);
  }
  pre {
    max-height: 96px;
    margin: 10px 0;
    padding: 10px;
    overflow: auto;
    border: 1px solid var(--border);
    border-radius: 7px;
    background: var(--input-bg);
    color: var(--text-dim);
    font: 11px/1.6 var(--font-mono);
    white-space: pre-wrap;
    word-break: break-word;
  }
  .paste {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    width: 100%;
    height: 32px;
    color: var(--accent-text);
    border: 1px solid var(--accent-border);
    border-radius: 7px;
    background: var(--accent-soft);
    font-size: 11px;
    font-weight: 600;
    transition: background 0.15s;
  }
  .paste:hover {
    background: var(--accent-border);
  }

  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    min-height: 200px;
    padding: 0 20px;
    text-align: center;
    color: var(--text-faint);
    font-size: 12px;
    line-height: 1.5;
  }
  .empty strong {
    font-size: 12px;
    color: var(--text-dim);
  }
  .eicon {
    display: grid;
    place-items: center;
    width: 40px;
    height: 40px;
    margin-bottom: 4px;
    color: var(--accent-text);
    border: 1px solid var(--accent-border);
    border-radius: 11px;
    background: var(--accent-soft);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 15px;
    padding: 16px;
    overflow-y: auto;
  }
  .fhead {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
  }
  .fhead h3 {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-strong);
  }
  .mono,
  .code {
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .code {
    height: auto;
    min-height: 170px;
    padding: 10px;
    line-height: 1.6;
    resize: vertical;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 8px;
    padding-top: 3px;
  }
  .text-btn {
    height: 35px;
    padding: 0 12px;
    border: none;
    background: none;
    color: var(--text-dim);
    font-size: 12px;
  }
  .text-btn:hover {
    color: var(--text);
  }
</style>
