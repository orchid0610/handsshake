<script lang="ts">
  import { onMount, onDestroy, tick } from "svelte";
  import { Terminal } from "@xterm/xterm";
  import { FitAddon } from "@xterm/addon-fit";
  import { SearchAddon, type ISearchOptions } from "@xterm/addon-search";
  import "@xterm/xterm/css/xterm.css";
  import { api, events, errText } from "../api";
  import { mix } from "../appTheme";
  import { app } from "../stores/app.svelte";
  import { settings } from "../stores/settings.svelte";
  import { tabs, type Tab } from "../stores/tabs.svelte";
  import Icon from "./Icon.svelte";

  let { tab, active }: { tab: Tab; active: boolean } = $props();

  let el: HTMLDivElement;
  let term: Terminal | undefined;
  let fit: FitAddon | undefined;
  let search: SearchAddon | undefined;
  let ro: ResizeObserver | undefined;
  let sid = ""; // id of the current backend session
  let attempt = 0;
  let unsub: Array<() => void> = [];
  let disposed = false;

  let cols = $state(80);
  let rows = $state(24);
  let latency = $state<number | null>(null);

  // ---- find bar ----
  let findOpen = $state(false);
  let findInput = $state<HTMLInputElement>();
  let query = $state("");
  let matchCase = $state(false);
  let resultText = $state("");

  function dropListeners() {
    unsub.forEach((f) => f());
    unsub = [];
  }

  // The highlight colours come from the current theme (the addon wants #rrggbb).
  function searchOptions(incremental: boolean): ISearchOptions {
    const ui = settings.theme.ui;
    return {
      caseSensitive: matchCase,
      incremental,
      decorations: {
        matchBackground: mix(ui.background, ui.accent, 0.3),
        matchBorder: mix(ui.background, ui.accent, 0.45),
        matchOverviewRuler: ui.accent,
        activeMatchBackground: mix(ui.background, ui.accent, 0.65),
        activeMatchBorder: ui.accent,
        activeMatchColorOverviewRuler: ui.path,
      },
    };
  }

  function runFind(incremental = false, backwards = false) {
    if (!search || !term) return;
    if (!query) {
      search.clearDecorations();
      term.clearSelection();
      resultText = "";
      return;
    }
    const found = backwards
      ? search.findPrevious(query, searchOptions(false))
      : search.findNext(query, searchOptions(incremental));
    if (!found) resultText = "No results";
  }

  async function openFind() {
    // Start from whatever is selected, if it's a single reasonable line.
    const sel = term?.getSelection() ?? "";
    if (sel && !sel.includes("\n") && sel.length <= 200) query = sel;
    findOpen = true;
    await tick();
    findInput?.focus();
    findInput?.select();
    if (query) runFind(true);
  }

  function closeFind() {
    findOpen = false;
    search?.clearDecorations();
    term?.clearSelection();
    resultText = "";
    term?.focus();
  }

  function onFindKey(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      closeFind();
    } else if (e.key === "Enter") {
      e.preventDefault();
      runFind(false, e.shiftKey);
    }
  }

  async function start() {
    if (!term) return;
    dropListeners();
    if (sid) api.disconnect(sid); // harmless if it no longer exists
    tabs.setStatus(tab.id, "connecting");
    latency = null;
    term.reset();

    const id = `${tab.id}-${++attempt}`;
    sid = id;

    // Subscribe BEFORE connecting so the very first prompt isn't missed.
    unsub.push(
      events.on(`term:data:${id}`, (b64: string) => {
        term?.write(Uint8Array.from(atob(b64), (c) => c.charCodeAt(0)));
      }),
      events.on(`term:closed:${id}`, () => {
        if (sid === id) tabs.setStatus(tab.id, "ended");
      }),
      events.on(
        `term:os:${id}`,
        (info: { id: string; name: string; like: string }) => {
          if (sid !== id) return;
          tabs.setOS(tab.id, info.id, info.like, info.name);
          app.refreshHosts(); // the Hosts page shows the logo and OS name too
        },
      ),
    );

    try {
      const info = await api.connect(id, tab.hostId, term.cols, term.rows);
      if (disposed || sid !== id) {
        api.disconnect(id);
        return;
      }
      tabs.setKeyType(tab.id, info.hostKeyType);
      if (tab.status === "connecting") tabs.setStatus(tab.id, "connected");
      app.refreshHosts(); // "last used" changed, so the Hosts page's recent list should too
      if (active) term.focus();
    } catch (e) {
      if (sid === id) tabs.setStatus(tab.id, "error", errText(e));
    }
  }

  onMount(async () => {
    // xterm measures character cells once; make sure the font is ready first.
    await document.fonts.load('13px "JetBrains Mono"').catch(() => {});
    if (disposed) return;

    const s = settings.value;
    const t = new Terminal({
      fontFamily: settings.fontStack,
      fontSize: s.fontSize,
      lineHeight: s.lineHeight,
      cursorStyle: s.cursorStyle,
      cursorBlink: s.cursorBlink,
      scrollback: s.scrollback,
      theme: settings.theme.colors,
    });
    const f = new FitAddon();
    t.loadAddon(f);

    const sa = new SearchAddon();
    t.loadAddon(sa);
    sa.onDidChangeResults(({ resultIndex, resultCount }) => {
      resultText =
        resultCount === 0
          ? "No results"
          : resultIndex < 0
            ? `${resultCount}+`
            : `${resultIndex + 1} of ${resultCount}`;
    });
    search = sa;

    t.open(el);
    f.fit();
    term = t;
    fit = f;
    cols = t.cols;
    rows = t.rows;
    tabs.register(tab.id, {
      paste: (text) => t.paste(text),
      focus: () => t.focus(),
      find: openFind,
    });

    t.onData((d) => {
      if (tab.status === "connected") api.write(sid, d).catch(() => {});
    });
    t.onResize(({ cols: c, rows: r }) => {
      cols = c;
      rows = r;
      if (tab.status === "connected") api.resize(sid, c, r);
    });

    // Ctrl+Shift+C copies the selection (plain Ctrl+C must reach the server).
    t.attachCustomKeyEventHandler((e) => {
      if (
        e.type === "keydown" &&
        e.ctrlKey &&
        e.shiftKey &&
        e.code === "KeyC"
      ) {
        const sel = t.getSelection();
        if (sel) navigator.clipboard.writeText(sel).catch(() => {});
        e.preventDefault();
        return false;
      }
      return true;
    });

    ro = new ResizeObserver(() => {
      if (active && el.clientWidth > 0 && el.clientHeight > 0) f.fit();
    });
    ro.observe(el);

    start();
  });

  // Apply settings changes to this terminal live.
  $effect(() => {
    // Read everything BEFORE the early return, so Svelte tracks it even on the first run.
    const {
      fontSize,
      lineHeight,
      cursorStyle,
      cursorBlink,
      scrollback,
      compactSpacing,
      showStatusBar,
    } = settings.value;
    void compactSpacing;
    void showStatusBar;
    const theme = settings.theme.colors;
    const family = settings.fontStack;
    if (!term) return;

    term.options.theme = theme;
    term.options.fontFamily = family;
    term.options.fontSize = fontSize;
    term.options.lineHeight = lineHeight;
    term.options.cursorStyle = cursorStyle;
    term.options.cursorBlink = cursorBlink;
    term.options.scrollback = scrollback;
    tick().then(() => {
      if (active) fit?.fit();
    }); // font and padding changes alter how many columns fit
  });

  // When this tab becomes the visible one, re-measure it and focus it.
  $effect(() => {
    if (active && term) {
      tick().then(() => {
        fit?.fit();
        if (!findOpen) term?.focus();
      });
    }
  });

  // Measure the real round-trip time every 10 seconds while this tab is visible and connected.
  $effect(() => {
    if (!active || tab.status !== "connected") return;
    let stopped = false;
    const poll = async () => {
      try {
        const ms = await api.latency(sid);
        if (!stopped) latency = ms;
      } catch {
        if (!stopped) latency = null;
      }
    };
    poll();
    const timer = setInterval(poll, 10_000);
    return () => {
      stopped = true;
      clearInterval(timer);
    };
  });

  onDestroy(() => {
    tabs.unregister(tab.id);
    disposed = true;
    dropListeners();
    if (sid) api.disconnect(sid);
    ro?.disconnect();
    term?.dispose();
  });
</script>

<div class="pane" class:hidden={!active}>
  <!-- The padding lives on .wrap; xterm measures .term, which has none. -->
  <div class="wrap" class:compact={settings.value.compactSpacing}>
    <div class="term" bind:this={el}></div>
  </div>

  {#if findOpen}
    <div class="find" role="search">
      <Icon name="search" size={14} />
      <input
        bind:this={findInput}
        bind:value={query}
        oninput={() => runFind(true)}
        onkeydown={onFindKey}
        placeholder="Find in terminal"
        aria-label="Find in terminal"
        spellcheck="false"
        autocomplete="off"
      />
      <span class="count" class:none={resultText === "No results"}
        >{resultText}</span
      >
      <button
        class="fbtn"
        class:on={matchCase}
        aria-pressed={matchCase}
        title="Match case"
        onclick={() => {
          matchCase = !matchCase;
          runFind(true);
        }}>Aa</button
      >
      <button
        class="fbtn"
        title="Previous match (Shift+Enter)"
        aria-label="Previous match"
        onclick={() => runFind(false, true)}
        ><Icon name="up" size={14} /></button
      >
      <button
        class="fbtn"
        title="Next match (Enter)"
        aria-label="Next match"
        onclick={() => runFind()}><Icon name="down" size={14} /></button
      >
      <button
        class="fbtn"
        title="Close (Esc)"
        aria-label="Close search"
        onclick={closeFind}><Icon name="x" size={14} /></button
      >
    </div>
  {/if}

  {#if settings.value.showStatusBar}
    <footer class="status">
      <span>UTF-8</span>
      <span>{rows} × {cols}</span>
      <span class="latency" class:off={latency === null}>
        <i></i>{latency === null ? "—" : `${latency} ms`}
      </span>
    </footer>
  {/if}

  {#if tab.status === "connecting"}
    <div class="overlay">
      <div class="spinner"></div>
      <p>Connecting to <strong>{tab.title}</strong>…</p>
    </div>
  {:else if tab.status === "error"}
    <div class="overlay">
      <span class="badge"><Icon name="alert" size={22} /></span>
      <h3>Couldn't connect</h3>
      <p class="msg selectable">{tab.error}</p>
      <div class="row">
        <button class="btn btn-primary" onclick={start}
          ><Icon name="refresh" /> Try again</button
        >
        <button class="btn" onclick={() => tabs.close(tab.id)}>Close tab</button
        >
      </div>
    </div>
  {:else if tab.status === "ended"}
    <div class="ended">
      <span>Session ended</span>
      <button class="btn btn-primary" onclick={start}
        ><Icon name="refresh" /> Reconnect</button
      >
      <button class="btn" onclick={() => tabs.close(tab.id)}>Close</button>
    </div>
  {/if}
</div>

<style>
  .pane {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    background: var(--terminal-bg);
  }
  .pane.hidden {
    display: none;
  }
  .wrap {
    flex: 1;
    min-height: 0;
    padding: 18px 10px 14px 22px;
  }
  .wrap.compact {
    padding: 10px 6px 8px 12px;
  }
  .term {
    width: 100%;
    height: 100%;
  }

  /* Below the snippets drawer (z-index 3 and 4), above the terminal. */
  .find {
    position: absolute;
    top: 10px;
    right: 18px;
    z-index: 2;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 36px;
    padding: 0 6px 0 11px;
    color: var(--text-faint);
    border: 1px solid var(--border-strong);
    border-radius: 9px;
    background: var(--surface);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
    animation: drop 0.14s ease;
  }
  @keyframes drop {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
  }
  .find input {
    width: 190px;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--text);
    font: inherit;
    font-size: 12.5px;
  }
  .find input::placeholder {
    color: var(--text-faint);
  }
  .count {
    min-width: 58px;
    text-align: right;
    font: 11px var(--font-mono);
    color: var(--text-dim);
    white-space: nowrap;
  }
  .count.none {
    color: var(--danger);
  }
  .fbtn {
    display: grid;
    place-items: center;
    min-width: 26px;
    height: 26px;
    padding: 0 6px;
    border: none;
    border-radius: 6px;
    background: none;
    color: var(--text-dim);
    font: 600 11px var(--font-ui);
    transition:
      background 0.12s,
      color 0.12s;
  }
  .fbtn:hover {
    background: var(--surface-3);
    color: var(--text);
  }
  .fbtn.on {
    background: var(--accent-soft);
    color: var(--accent-text);
  }

  .status {
    display: flex;
    align-items: center;
    gap: 17px;
    height: 28px;
    flex: none;
    padding: 0 13px;
    border-top: 1px solid var(--terminal-border);
    background: var(--terminal-surface);
    color: var(--terminal-muted);
    font: 10px var(--terminal-font-family, var(--font-mono));
  }
  .latency {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
  }
  .latency i {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--success);
  }
  .latency.off i {
    background: var(--terminal-muted);
  }

  .overlay {
    position: absolute;
    inset: 0;
    display: grid;
    align-content: center;
    justify-items: center;
    gap: 12px;
    padding: 30px;
    text-align: center;
    color: var(--text-dim);
    background: var(--overlay);
  }
  .overlay strong {
    color: var(--text);
  }
  h3 {
    font-size: 17px;
    color: var(--text-strong);
  }
  .msg {
    max-width: 460px;
    font-size: 13px;
    line-height: 1.55;
    word-break: break-word;
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
  .spinner {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    border: 2px solid var(--border-strong);
    border-top-color: var(--accent-line);
    animation: spin 0.8s linear infinite;
  }
  .ended {
    position: absolute;
    left: 50%;
    bottom: 46px;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 8px 8px 16px;
    background: var(--surface-2);
    border: 1px solid var(--border-strong);
    border-radius: 12px;
    box-shadow: var(--shadow);
    color: var(--text-dim);
  }
  .ended .btn {
    height: 30px;
    font-size: 13px;
  }
</style>
