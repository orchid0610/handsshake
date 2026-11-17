import { api, errText } from "../api";
import { THEMES, themeById } from "../themes";
import type { Settings } from "../types";
import { app } from "./app.svelte";

export const DEFAULTS: Settings = {
  theme: "emerald",
  fontFamily: "jetbrains-mono",
  fontSize: 13,
  lineHeight: 1.4,
  cursorStyle: "block",
  cursorBlink: true,
  scrollback: 5000,
  compactSpacing: false,
  showStatusBar: true,
  autoLockMinutes: 0,
};

export const FONT_STACKS: Record<Settings["fontFamily"], string> = {
  "jetbrains-mono": '"JetBrains Mono", ui-monospace, monospace',
  system:
    'ui-monospace, "DejaVu Sans Mono", "Liberation Mono", Menlo, Consolas, monospace',
};

class SettingsStore {
  value = $state<Settings>({ ...DEFAULTS });
  theme = $derived(themeById(this.value.theme));
  fontStack = $derived(
    FONT_STACKS[this.value.fontFamily] ?? FONT_STACKS["jetbrains-mono"],
  );

  private timer: ReturnType<typeof setTimeout> | undefined;
  private dirty = false;

  async load() {
    try {
      const s = await api.getSettings();
      // A theme from an older version no longer exists: show the default as selected.
      if (!THEMES.some((t) => t.id === s.theme)) s.theme = DEFAULTS.theme;
      this.value = s;
    } catch (e) {
      app.toast("error", errText(e));
    }
  }

  /** Apply a change right away, and save it shortly after (so dragging a slider doesn't hammer the disk). */
  update(patch: Partial<Settings>) {
    this.value = { ...this.value, ...patch };
    this.dirty = true;
    clearTimeout(this.timer);
    this.timer = setTimeout(() => this.persist(), 400);
  }

  /** Back to defaults for everything about how the terminal looks. Security settings are left alone. */
  resetTerminal() {
    const {
      theme,
      fontFamily,
      fontSize,
      lineHeight,
      cursorStyle,
      cursorBlink,
      scrollback,
      compactSpacing,
      showStatusBar,
    } = DEFAULTS;
    this.update({
      theme,
      fontFamily,
      fontSize,
      lineHeight,
      cursorStyle,
      cursorBlink,
      scrollback,
      compactSpacing,
      showStatusBar,
    });
  }

  private async persist() {
    this.dirty = false;
    try {
      const saved = await api.saveSettings({ ...this.value });
      if (!this.dirty) this.value = saved; // Go may have clamped something; show what was really kept
    } catch (e) {
      app.toast("error", errText(e));
    }
  }
}

export const settings = new SettingsStore();
