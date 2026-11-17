import { themeById, type TerminalTheme } from "./themes";

type RGB = [number, number, number];

function parse(c: string): RGB {
  const n = parseInt(c.replace("#", ""), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}
function hex([r, g, b]: RGB): string {
  return (
    "#" +
    [r, g, b].map((v) => Math.round(v).toString(16).padStart(2, "0")).join("")
  );
}
/** Blend a toward b: t=0 gives a, t=1 gives b. */
export function mix(a: string, b: string, t: number): string {
  const x = parse(a),
    y = parse(b);
  return hex([
    x[0] + (y[0] - x[0]) * t,
    x[1] + (y[1] - x[1]) * t,
    x[2] + (y[2] - x[2]) * t,
  ]);
}
export function rgba(c: string, alpha: number): string {
  const [r, g, b] = parse(c);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

const WHITE = "#ffffff";
const BLACK = "#000000";

// Primary-button colours per theme (the emerald values are from the design).
const BUTTONS: Record<string, { top: string; bottom: string; border: string }> =
  {
    emerald: { top: "#238b61", bottom: "#156044", border: "#33976c" },
    forest: { top: "#2e8b4a", bottom: "#1b5f31", border: "#3a9d57" },
    graphite: { top: "#3d8a6c", bottom: "#295f49", border: "#4b9a7b" },
    amber: { top: "#a47625", bottom: "#6c4b15", border: "#bd8b33" },
  };

/** Every colour the app uses, derived from one terminal theme. */
export function appVars(t: TerminalTheme): Record<string, string> {
  const ui = t.ui;
  const btn = BUTTONS[t.id] ?? BUTTONS.emerald;
  return {
    "--bg": ui.background,
    "--term-bg": ui.background,
    "--rail-bg": mix(ui.surface, ui.background, 0.5),
    "--card": ui.surface,
    "--surface": mix(ui.surface, WHITE, 0.025),
    "--surface-2": mix(ui.surface, WHITE, 0.05),
    "--surface-3": mix(ui.surface, WHITE, 0.09),
    "--input-bg": mix(ui.background, BLACK, 0.3),
    "--border": ui.border,
    "--border-strong": mix(ui.border, WHITE, 0.1),
    "--text": mix(ui.text, WHITE, 0.3),
    "--text-strong": mix(ui.text, WHITE, 0.7),
    "--text-dim": mix(ui.text, ui.background, 0.3),
    "--text-faint": mix(ui.text, ui.background, 0.52),
    "--accent": btn.top,
    "--accent-hover": mix(btn.top, WHITE, 0.12),
    "--accent-text": ui.accent,
    "--accent-line": ui.accent,
    "--accent-soft": rgba(ui.accent, 0.14),
    "--accent-border": rgba(ui.accent, 0.3),
    "--btn-grad": `linear-gradient(180deg, ${btn.top}, ${btn.bottom})`,
    "--btn-grad-hover": `linear-gradient(180deg, ${mix(btn.top, WHITE, 0.1)}, ${mix(btn.bottom, WHITE, 0.1)})`,
    "--btn-border": btn.border,
    "--btn-shadow": `0 7px 20px ${rgba(btn.bottom, 0.4)}`,
    "--brand-gradient": `linear-gradient(145deg, ${btn.top}, ${btn.bottom})`,
    "--brand-glow": rgba(btn.top, 0.3),
    "--page-glow": `radial-gradient(circle at 58% -18%, ${rgba(ui.accent, 0.08)}, transparent 34%)`,
    "--tab-active": mix(ui.surface, ui.accent, 0.12),
    "--overlay": rgba(ui.background, 0.92),

    // the terminal's own colours
    "--terminal-bg": ui.background,
    "--terminal-surface": ui.surface,
    "--terminal-border": ui.border,
    "--terminal-text": ui.text,
    "--terminal-muted": ui.muted,
    "--terminal-accent": ui.accent,
    "--terminal-path": ui.path,
  };
}

export function applyTheme(t: TerminalTheme) {
  const root = document.documentElement.style;
  for (const [name, value] of Object.entries(appVars(t)))
    root.setProperty(name, value);
  try {
    localStorage.setItem("hs-theme", t.id);
  } catch {
    /* private mode etc.: the theme still applies */
  }
}

/** Called before the UI mounts, so a non-default theme doesn't flash emerald first. */
export function applySavedTheme() {
  let id = "emerald";
  try {
    id = localStorage.getItem("hs-theme") ?? id;
  } catch {
    /* ignore */
  }
  applyTheme(themeById(id));
}
