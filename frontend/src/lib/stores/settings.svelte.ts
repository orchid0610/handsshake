import { api, errText } from "../api";
import { themeById } from "../themes";
import type { Settings } from "../types";
import { app } from "./app.svelte";

export const DEFAULTS: Settings = {
  theme: "tokyo-night",
  fontFamily: "jetbrains-mono",
  fontSize: 13.5,
  lineHeight: 1.2,
  cursorStyle: "block",
  cursorBlink: true,
  scrollback: 5000,
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
      this.value = await api.getSettings();
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
