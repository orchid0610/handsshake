<script lang="ts">
  import { app } from "../stores/app.svelte";
  import { settings } from "../stores/settings.svelte";
  import { THEMES } from "../themes";
  import { api, errText } from "../api";
  import type { Settings } from "../types";
  import Icon from "../components/Icon.svelte";
  import Toggle from "../components/Toggle.svelte";

  const s = $derived(settings.value);
  const ui = $derived(settings.theme.ui);

  // If a hand-edited file holds a value that isn't in our list, still show it.
  const withCurrent = (base: number[], cur: number) =>
    base.includes(cur) ? base : [...base, cur].sort((a, b) => a - b);
  const lockChoices = $derived(
    withCurrent([0, 1, 5, 15, 30, 60], s.autoLockMinutes),
  );
  const scrollChoices = $derived(
    withCurrent([1000, 5000, 10000, 25000, 50000], s.scrollback),
  );
  const lockLabel = (m: number) =>
    m === 0
      ? "Never"
      : m < 60
        ? `After ${m} minute${m === 1 ? "" : "s"}`
        : m % 60 === 0
          ? `After ${m / 60} hour${m === 60 ? "" : "s"}`
          : `After ${m} minutes`;

  // ---- change master password ----
  let cur = $state("");
  let next = $state("");
  let again = $state("");
  let pwError = $state("");
  let busy = $state(false);

  async function changePw() {
    pwError = "";
    if (next.length < 8) {
      pwError = "Use at least 8 characters.";
      return;
    }
    if (next !== again) {
      pwError = "The new passwords do not match.";
      return;
    }
    busy = true;
    try {
      await api.changePassword(cur, next);
      cur = next = again = "";
      app.toast("success", "Master password changed");
    } catch (e) {
      pwError = errText(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="inner">
  <header>
    <div>
      <p class="eyebrow">Preferences</p>
      <h1 class="page-title">Terminal settings</h1>
      <p class="page-subtitle">
        Customize the look and feel of every SSH session.
      </p>
    </div>
    <button class="btn reset" onclick={() => settings.resetTerminal()}
      >Reset defaults</button
    >
  </header>

  <div class="layout">
    <div class="stack">
      <section class="card">
        <div class="head">
          <span class="icon"><Icon name="terminal" size={17} /></span>
          <div>
            <h2>Terminal theme</h2>
            <p>Choose a color palette for terminal sessions.</p>
          </div>
        </div>
        <div class="themes" role="radiogroup" aria-label="Terminal theme">
          {#each THEMES as t (t.id)}
            <button
              class="theme"
              class:on={s.theme === t.id}
              role="radio"
              aria-checked={s.theme === t.id}
              onclick={() => settings.update({ theme: t.id })}
            >
              <span class="swatch">
                <i style:background={t.ui.background}></i><i
                  style:background={t.ui.surface}
                ></i>
                <i style:background={t.ui.text}></i><i
                  style:background={t.ui.accent}
                ></i>
              </span>
              <span class="tname"
                ><strong>{t.name}</strong><small>{t.description}</small></span
              >
              <i class="dot"></i>
            </button>
          {/each}
        </div>
      </section>

      <section class="card">
        <div class="head">
          <span class="icon"><Icon name="user" size={17} /></span>
          <div>
            <h2>Typography</h2>
            <p>Adjust terminal readability and density.</p>
          </div>
        </div>

        <div class="setting first">
          <div class="label">
            <strong>Font family</strong><small
              >Applied to commands and terminal output.</small
            >
          </div>
          <div class="seg" role="group" aria-label="Font family">
            <button
              class:on={s.fontFamily === "jetbrains-mono"}
              onclick={() => settings.update({ fontFamily: "jetbrains-mono" })}
              >JetBrains Mono</button
            >
            <button
              class:on={s.fontFamily === "system"}
              onclick={() => settings.update({ fontFamily: "system" })}
              >System Mono</button
            >
          </div>
        </div>

        <div class="setting">
          <div class="label">
            <strong>Font size</strong><small>Controls terminal text size.</small
            >
          </div>
          <div class="range">
            <input
              type="range"
              min="10"
              max="18"
              step="0.5"
              value={s.fontSize}
              aria-label="Font size"
              oninput={(e) =>
                settings.update({ fontSize: Number(e.currentTarget.value) })}
            />
            <output>{s.fontSize}px</output>
          </div>
        </div>

        <div class="setting">
          <div class="label">
            <strong>Line height</strong><small
              >Changes vertical spacing between lines.</small
            >
          </div>
          <div class="range">
            <input
              type="range"
              min="1"
              max="2"
              step="0.05"
              value={s.lineHeight}
              aria-label="Line height"
              oninput={(e) =>
                settings.update({ lineHeight: Number(e.currentTarget.value) })}
            />
            <output>{s.lineHeight.toFixed(2)}</output>
          </div>
        </div>
      </section>

      <section class="card">
        <div class="head">
          <span class="icon"><Icon name="settings" size={17} /></span>
          <div>
            <h2>Session interface</h2>
            <p>Control the density of your terminal sessions.</p>
          </div>
        </div>
        <div class="setting first">
          <div class="label">
            <strong>Compact spacing</strong><small
              >Reduce padding around terminal output.</small
            >
          </div>
          <Toggle
            checked={s.compactSpacing}
            label="Compact spacing"
            onchange={(v) => settings.update({ compactSpacing: v })}
          />
        </div>
        <div class="setting">
          <div class="label">
            <strong>Show status bar</strong><small
              >Display encoding, dimensions, and latency.</small
            >
          </div>
          <Toggle
            checked={s.showStatusBar}
            label="Show status bar"
            onchange={(v) => settings.update({ showStatusBar: v })}
          />
        </div>
      </section>

      <section class="card">
        <div class="head">
          <span class="icon"><Icon name="activity" size={17} /></span>
          <div>
            <h2>Cursor &amp; history</h2>
            <p>How the cursor looks and how much output is kept.</p>
          </div>
        </div>
        <div class="setting first">
          <div class="label">
            <strong>Cursor style</strong><small
              >The shape of the cursor in a session.</small
            >
          </div>
          <div class="seg" role="group" aria-label="Cursor style">
            {#each ["block", "bar", "underline"] as c}
              <button
                class:on={s.cursorStyle === c}
                onclick={() =>
                  settings.update({
                    cursorStyle: c as Settings["cursorStyle"],
                  })}
              >
                {c[0].toUpperCase() + c.slice(1)}
              </button>
            {/each}
          </div>
        </div>
        <div class="setting">
          <div class="label">
            <strong>Blinking cursor</strong><small
              >Pulse the cursor while a session is focused.</small
            >
          </div>
          <Toggle
            checked={s.cursorBlink}
            label="Blinking cursor"
            onchange={(v) => settings.update({ cursorBlink: v })}
          />
        </div>
        <div class="setting">
          <div class="label">
            <strong>Scrollback</strong><small
              >How many lines of output each session remembers.</small
            >
          </div>
          <select
            class="input select"
            value={s.scrollback}
            aria-label="Scrollback"
            onchange={(e) =>
              settings.update({ scrollback: Number(e.currentTarget.value) })}
          >
            {#each scrollChoices as n}<option value={n}
                >{n.toLocaleString()} lines</option
              >{/each}
          </select>
        </div>
      </section>

      <section class="card">
        <div class="head">
          <span class="icon"><Icon name="shield" size={17} /></span>
          <div>
            <h2>Security</h2>
            <p>Protect your vault when you step away.</p>
          </div>
        </div>
        <div class="setting first">
          <div class="label">
            <strong>Auto-lock</strong><small
              >Locks the vault after no activity in this window. Open sessions
              stay connected.</small
            >
          </div>
          <select
            class="input select"
            value={s.autoLockMinutes}
            aria-label="Auto-lock"
            onchange={(e) =>
              settings.update({
                autoLockMinutes: Number(e.currentTarget.value),
              })}
          >
            {#each lockChoices as m}<option value={m}>{lockLabel(m)}</option
              >{/each}
          </select>
        </div>

        <form
          onsubmit={(e) => {
            e.preventDefault();
            changePw();
          }}
        >
          <strong>Change master password</strong>
          <input
            class="input"
            type="password"
            placeholder="Current password"
            bind:value={cur}
            autocomplete="off"
          />
          <input
            class="input"
            type="password"
            placeholder="New password"
            bind:value={next}
            autocomplete="off"
          />
          <input
            class="input"
            type="password"
            placeholder="Confirm new password"
            bind:value={again}
            autocomplete="off"
          />
          {#if pwError}<p class="error-text">{pwError}</p>{/if}
          <div class="row">
            <button class="btn" disabled={busy || !cur || !next}
              >{busy ? "Changing…" : "Change password"}</button
            >
            <span class="hint">Your recovery key keeps working.</span>
          </div>
        </form>
      </section>
    </div>

    <aside class="preview">
      <div class="pv-head">
        <div><span>Live preview</span><small>{settings.theme.name}</small></div>
        <span class="live"><i></i> Live</span>
      </div>

      <div
        class="term"
        class:compact={s.compactSpacing}
        style:background={ui.background}
        style:border-color={ui.border}
      >
        <div
          class="tab"
          style:background={ui.surface}
          style:color={ui.muted}
          style:border-color={ui.border}
        >
          <Icon name="terminal" size={13} /> Preview session
        </div>
        <div
          class="code"
          style:color={ui.text}
          style:font-family={settings.fontStack}
          style:font-size="{s.fontSize}px"
          style:line-height={s.lineHeight}
        >
          <p>
            <span style:color={ui.accent}>dev@server</span><span
              style:color={ui.path}>:~$</span
            > ssh production
          </p>
          <p style:color={ui.muted}>Connected to 203.0.113.8</p>
          <p>
            <span style:color={ui.accent}>root@production</span><span
              style:color={ui.path}>:~$</span
            > uptime
          </p>
          <p>10:24:08 up 42 days, 3 users</p>
          <p style:color={ui.accent}>System operational</p>
          <p>
            <span style:color={ui.accent}>root@production</span><span
              style:color={ui.path}>:~$</span
            >
            <span
              class="cursor {s.cursorStyle}"
              class:blink={s.cursorBlink}
              style:--c={ui.accent}
            ></span>
          </p>
        </div>
        {#if s.showStatusBar}
          <div
            class="foot"
            style:background={ui.surface}
            style:color={ui.muted}
            style:border-color={ui.border}
            style:font-family={settings.fontStack}
          >
            <span>UTF-8</span><span>38 ms</span>
          </div>
        {/if}
      </div>
      <p class="note">
        Changes are applied instantly to all terminal sessions.
      </p>
    </aside>
  </div>
</div>

<style>
  .inner {
    width: min(1080px, calc(100% - 88px));
    margin: 0 auto;
    padding: 48px 0 56px;
  }
  header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 24px;
    margin-bottom: 30px;
  }
  .eyebrow {
    margin-bottom: 4px;
  }
  .page-subtitle {
    margin-top: 7px;
  }
  .reset {
    height: 34px;
    font-size: 12px;
    color: var(--text-dim);
  }

  .layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 320px;
    align-items: start;
    gap: 18px;
  }
  .stack {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-width: 0;
  }

  .card,
  .preview {
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--card);
  }
  .card {
    padding: 20px;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 11px;
    margin-bottom: 20px;
  }
  .icon {
    display: grid;
    place-items: center;
    width: 34px;
    height: 34px;
    flex: none;
    color: var(--accent-text);
    border: 1px solid var(--accent-border);
    border-radius: 9px;
    background: var(--accent-soft);
  }
  .head h2 {
    font-size: 13px;
    font-weight: 600;
    color: var(--text);
    margin-bottom: 3px;
  }
  .head p {
    font-size: 11px;
    color: var(--text-faint);
  }

  .themes {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
  .theme {
    display: flex;
    align-items: center;
    gap: 11px;
    min-width: 0;
    padding: 12px;
    text-align: left;
    border: 1px solid var(--border);
    border-radius: 9px;
    background: var(--input-bg);
    transition:
      border-color 0.15s,
      background 0.15s;
  }
  .theme:hover {
    border-color: var(--border-strong);
    background: var(--surface-2);
  }
  .theme.on {
    border-color: var(--accent-border);
    background: var(--accent-soft);
  }
  .swatch {
    display: grid;
    grid-template-columns: 1fr 1fr;
    width: 38px;
    height: 38px;
    flex: none;
    border: 1px solid var(--border-strong);
    border-radius: 8px;
    overflow: hidden;
  }
  .tname {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .tname strong {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
  }
  .tname small {
    font-size: 11px;
    color: var(--text-faint);
  }
  .dot {
    width: 8px;
    height: 8px;
    margin-left: auto;
    flex: none;
    border: 1px solid var(--border-strong);
    border-radius: 50%;
  }
  .theme.on .dot {
    border-color: var(--accent-line);
    background: var(--accent-line);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .setting {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    min-height: 62px;
    border-top: 1px solid var(--border);
  }
  .setting.first {
    border-top: 0;
  }
  .label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .label strong,
  form > strong {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
  }
  .label small {
    font-size: 11px;
    color: var(--text-faint);
    line-height: 1.4;
  }

  .seg {
    display: flex;
    flex: none;
    padding: 3px;
    border: 1px solid var(--border-strong);
    border-radius: 8px;
    background: var(--input-bg);
  }
  .seg button {
    min-width: 76px;
    height: 28px;
    padding: 0 10px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-faint);
    font: 11px var(--font-mono);
  }
  .seg button.on {
    background: var(--accent-soft);
    color: var(--accent-text);
  }

  .range {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 240px;
    flex: none;
  }
  .range input {
    width: 100%;
    height: 3px;
    appearance: none;
    -webkit-appearance: none;
    border-radius: 4px;
    background: var(--border-strong);
  }
  .range input::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 14px;
    height: 14px;
    border: 2px solid var(--bg);
    border-radius: 50%;
    background: var(--accent-line);
    box-shadow: 0 0 0 2px var(--accent-border);
    cursor: pointer;
  }
  .range output {
    width: 46px;
    text-align: right;
    font: 11px var(--font-mono);
    color: var(--accent-text);
  }
  .select {
    width: auto;
    min-width: 170px;
    height: 34px;
    flex: none;
  }

  form {
    display: grid;
    gap: 10px;
    margin-top: 6px;
    padding-top: 18px;
    border-top: 1px solid var(--border);
  }
  form .btn {
    align-self: start;
  }

  .preview {
    position: sticky;
    top: 24px;
    padding: 14px;
  }
  .pv-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 2px 2px 13px;
  }
  .pv-head > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .pv-head span {
    font-size: 12px;
    font-weight: 600;
    color: var(--text);
  }
  .pv-head small {
    font-size: 10px;
    color: var(--text-faint);
  }
  .live {
    display: flex;
    align-items: center;
    gap: 5px;
    color: var(--accent-text) !important;
    font-size: 10px !important;
    font-weight: 500 !important;
  }
  .live i {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: var(--success);
    box-shadow: 0 0 7px rgba(75, 209, 155, 0.5);
  }

  .term {
    border: 1px solid;
    border-radius: 9px;
    overflow: hidden;
    box-shadow: 0 14px 32px rgba(0, 0, 0, 0.22);
  }
  .tab {
    display: flex;
    align-items: center;
    gap: 7px;
    height: 31px;
    padding: 0 10px;
    font-size: 10px;
    border-bottom: 1px solid;
  }
  .code {
    min-height: 225px;
    padding: 16px;
    transition: padding 0.18s;
    user-select: none;
  }
  .term.compact .code {
    padding: 10px;
  }
  .code p {
    margin: 0;
    min-height: 1em;
    white-space: nowrap;
    overflow: hidden;
  }
  .cursor {
    display: inline-block;
    width: 0.6em;
    height: 1.05em;
    vertical-align: text-bottom;
  }
  .cursor.block {
    background: var(--c);
  }
  .cursor.bar {
    width: 0;
    border-left: 2px solid var(--c);
  }
  .cursor.underline {
    height: 0;
    border-bottom: 2px solid var(--c);
    vertical-align: baseline;
  }
  .cursor.blink {
    animation: blink 1s step-end infinite;
  }
  @keyframes blink {
    50% {
      opacity: 0;
    }
  }
  .foot {
    display: flex;
    justify-content: space-between;
    padding: 7px 10px;
    font-size: 10px;
    border-top: 1px solid;
  }
  .note {
    margin: 12px 3px 2px;
    font-size: 10px;
    line-height: 1.5;
    color: var(--text-faint);
  }

  @media (max-width: 900px) {
    .inner {
      width: calc(100% - 40px);
      padding-top: 32px;
    }
    .layout {
      grid-template-columns: 1fr;
    }
    .preview {
      position: static;
    }
    .themes {
      grid-template-columns: 1fr;
    }
    .setting {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;
      padding: 14px 0;
    }
    .range {
      width: 100%;
    }
  }
</style>
