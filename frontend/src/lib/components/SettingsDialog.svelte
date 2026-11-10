<script lang="ts">
  import { app } from "../stores/app.svelte";
  import { settings } from "../stores/settings.svelte";
  import { THEMES } from "../themes";
  import { api, errText } from "../api";
  import Modal from "./Modal.svelte";
  import Icon from "./Icon.svelte";

  const s = $derived(settings.value);
  const th = $derived(settings.theme.colors);

  // If the file was hand-edited to a value that isn't in our list, still show it.
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
        ? `${m} minute${m === 1 ? "" : "s"}`
        : m % 60 === 0
          ? `${m / 60} hour${m === 60 ? "" : "s"}`
          : `${m} minutes`;

  const stepFont = (d: number) =>
    settings.update({
      fontSize: Math.min(28, Math.max(9, Math.round((s.fontSize + d) * 2) / 2)),
    });

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

<Modal title="Settings" width={580} onclose={() => (app.settingsOpen = false)}>
  <section>
    <h3>Appearance</h3>

    <div class="themes" role="radiogroup" aria-label="Colour theme">
      {#each THEMES as t (t.id)}
        <button
          class="theme"
          class:on={s.theme === t.id}
          role="radio"
          aria-checked={s.theme === t.id}
          onclick={() => settings.update({ theme: t.id })}
        >
          <span class="swatch" style:background={t.colors.background}>
            {#each [t.colors.red, t.colors.green, t.colors.yellow, t.colors.blue, t.colors.magenta, t.colors.cyan] as c}
              <i style:background={c}></i>
            {/each}
          </span>
          <span class="tname">{t.name}</span>
        </button>
      {/each}
    </div>

    <div
      class="preview"
      style:background={th.background}
      style:color={th.foreground}
      style:font-family={settings.fontStack}
      style:font-size="{s.fontSize}px"
      style:line-height={s.lineHeight}
    >
      <div>
        <span style:color={th.green}>you@server</span>:<span
          style:color={th.blue}>~/projects</span
        >$ ls -la
      </div>
      <div>
        <span style:color={th.blue}>drwxr-xr-x</span> 4 you you 4096
        <span style:color={th.yellow}>Nov 10</span>
        <span style:color={th.cyan}>handsshake</span>
      </div>
      <div>
        <span style:color={th.red}>error:</span>
        <span style:color={th.magenta}>permission denied</span>
        <span
          class="cursor {s.cursorStyle}"
          class:blink={s.cursorBlink}
          style:--c={th.cursor}
        ></span>
      </div>
    </div>
  </section>

  <section>
    <h3>Terminal</h3>

    <div class="setting">
      <label for="font">Font</label>
      <select
        id="font"
        class="input"
        value={s.fontFamily}
        onchange={(e) =>
          settings.update({ fontFamily: e.currentTarget.value as any })}
      >
        <option value="jetbrains-mono">JetBrains Mono</option>
        <option value="system">System monospace</option>
      </select>
    </div>

    <div class="setting">
      <label for="size">Font size</label>
      <div class="stepper">
        <button
          class="icon-btn"
          onclick={() => stepFont(-0.5)}
          aria-label="Smaller"><Icon name="minus" size={14} /></button
        >
        <output id="size">{s.fontSize}px</output>
        <button
          class="icon-btn"
          onclick={() => stepFont(0.5)}
          aria-label="Larger"><Icon name="plus" size={14} /></button
        >
      </div>
    </div>

    <div class="setting">
      <label for="lh">Line height</label>
      <div class="slider">
        <input
          id="lh"
          type="range"
          min="1"
          max="1.6"
          step="0.05"
          value={s.lineHeight}
          oninput={(e) =>
            settings.update({ lineHeight: Number(e.currentTarget.value) })}
        />
        <output>{s.lineHeight.toFixed(2)}</output>
      </div>
    </div>

    <div class="setting">
      <label for="cursor-seg">Cursor</label>
      <div class="cursorctl">
        <div id="cursor-seg" class="seg" role="group">
          {#each ["block", "bar", "underline"] as c}
            <button
              class:on={s.cursorStyle === c}
              onclick={() => settings.update({ cursorStyle: c as any })}
            >
              {c[0].toUpperCase() + c.slice(1)}
            </button>
          {/each}
        </div>
        <label class="checkbox">
          <input
            type="checkbox"
            checked={s.cursorBlink}
            onchange={(e) =>
              settings.update({ cursorBlink: e.currentTarget.checked })}
          />
          Blink
        </label>
      </div>
    </div>

    <div class="setting">
      <label for="sb">Scrollback</label>
      <select
        id="sb"
        class="input"
        value={s.scrollback}
        onchange={(e) =>
          settings.update({ scrollback: Number(e.currentTarget.value) })}
      >
        {#each scrollChoices as n}<option value={n}
            >{n.toLocaleString()} lines</option
          >{/each}
      </select>
    </div>
  </section>

  <section>
    <h3>Security</h3>

    <div class="setting">
      <label for="lock">Auto-lock</label>
      <select
        id="lock"
        class="input"
        value={s.autoLockMinutes}
        onchange={(e) =>
          settings.update({ autoLockMinutes: Number(e.currentTarget.value) })}
      >
        {#each lockChoices as m}<option value={m}
            >{m === 0 ? "Never" : "After " + lockLabel(m)}</option
          >{/each}
      </select>
    </div>
    <p class="hint">
      Locks the vault after no keyboard or mouse activity in this window. Open
      sessions stay connected.
    </p>

    <form
      onsubmit={(e) => {
        e.preventDefault();
        changePw();
      }}
    >
      <h4>Change master password</h4>
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
</Modal>

<style>
  section {
    display: grid;
    gap: 14px;
    padding-bottom: 22px;
    margin-bottom: 22px;
    border-bottom: 1px solid var(--border);
  }
  section:last-child {
    margin-bottom: 0;
    padding-bottom: 0;
    border-bottom: none;
  }
  h3 {
    font-size: 11.5px;
    font-weight: 600;
    letter-spacing: 0.09em;
    text-transform: uppercase;
    color: var(--text-faint);
  }
  h4 {
    font-size: 13px;
    font-weight: 600;
    margin-bottom: 2px;
  }

  .themes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }
  .theme {
    display: grid;
    gap: 8px;
    padding: 8px;
    text-align: left;
    cursor: pointer;
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    transition:
      border-color 0.12s,
      background 0.12s;
  }
  .theme:hover {
    background: var(--surface-3);
  }
  .theme.on {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }
  .swatch {
    display: flex;
    align-items: center;
    gap: 4px;
    height: 28px;
    padding: 0 8px;
    border-radius: 6px;
  }
  .swatch i {
    width: 10px;
    height: 10px;
    border-radius: 50%;
  }
  .tname {
    font-size: 12.5px;
    font-weight: 500;
  }

  .preview {
    padding: 12px 14px;
    border-radius: var(--radius);
    border: 1px solid var(--border-strong);
    white-space: pre;
    overflow: hidden;
    user-select: none;
  }
  .cursor {
    display: inline-block;
    width: 0.6em;
    height: 1.1em;
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
    animation: blink 1.1s steps(1) infinite;
  }
  @keyframes blink {
    50% {
      opacity: 0;
    }
  }

  .setting {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }
  .setting > label {
    font-size: 13.5px;
    color: var(--text-dim);
  }
  select.input {
    width: auto;
    min-width: 190px;
  }

  .stepper {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 3px;
    background: var(--bg);
    border: 1px solid var(--border-strong);
    border-radius: 9px;
  }
  .stepper output {
    min-width: 56px;
    text-align: center;
    font-variant-numeric: tabular-nums;
    font-weight: 500;
  }
  .slider {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .slider input {
    width: 170px;
    accent-color: var(--accent);
  }
  .slider output {
    min-width: 36px;
    font-variant-numeric: tabular-nums;
    color: var(--text-dim);
  }

  .cursorctl {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  .seg {
    display: inline-grid;
    grid-auto-flow: column;
    gap: 3px;
    padding: 3px;
    background: var(--bg);
    border: 1px solid var(--border-strong);
    border-radius: 9px;
  }
  .seg button {
    height: 28px;
    padding: 0 12px;
    border: none;
    border-radius: 7px;
    background: transparent;
    color: var(--text-dim);
    font-weight: 500;
    cursor: pointer;
  }
  .seg button.on {
    background: var(--surface-3);
    color: var(--text);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.4);
  }
  .checkbox {
    align-items: center;
  }

  form {
    display: grid;
    gap: 10px;
    margin-top: 8px;
    padding-top: 18px;
    border-top: 1px dashed var(--border-strong);
  }
  form .btn {
    align-self: start;
  }
</style>
