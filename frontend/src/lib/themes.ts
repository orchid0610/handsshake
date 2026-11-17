import type { ITheme } from "@xterm/xterm";

/** The colours the app's own chrome (tab bar, status bar, preview) takes from the theme. */
export interface ThemeUI {
  background: string;
  surface: string;
  border: string;
  text: string;
  muted: string;
  accent: string;
  path: string;
}

export interface TerminalTheme {
  id: string;
  name: string;
  description: string;
  ui: ThemeUI;
  colors: ITheme; // handed to xterm
}

function alpha(hex: string, a: number): string {
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${n >> 16}, ${(n >> 8) & 255}, ${n & 255}, ${a})`;
}

function theme(
  id: string,
  name: string,
  description: string,
  ui: ThemeUI,
  ansi: ITheme,
): TerminalTheme {
  return {
    id,
    name,
    description,
    ui,
    colors: {
      background: ui.background,
      foreground: ui.text,
      cursor: ui.accent,
      cursorAccent: ui.background,
      selectionBackground: alpha(ui.accent, 0.28),
      ...ansi,
    },
  };
}

export const THEMES: TerminalTheme[] = [
  theme(
    "emerald",
    "Deep emerald",
    "Balanced and calm",
    {
      background: "#090d0b",
      surface: "#0e1411",
      border: "#1d2923",
      text: "#c2ccc7",
      muted: "#55615b",
      accent: "#5fc493",
      path: "#8abda5",
    },
    {
      black: "#16201b",
      red: "#e0707a",
      green: "#5fc493",
      yellow: "#d9bd7a",
      blue: "#6fa8d6",
      magenta: "#b79ad6",
      cyan: "#6fc5c0",
      white: "#b9c4be",
      brightBlack: "#55615b",
      brightRed: "#ef8a93",
      brightGreen: "#7fd9ab",
      brightYellow: "#ead08f",
      brightBlue: "#8bbde6",
      brightMagenta: "#cbb0e6",
      brightCyan: "#8ad8d3",
      brightWhite: "#e3ebe7",
    },
  ),
  theme(
    "forest",
    "Forest",
    "Deep and organic",
    {
      background: "#07100b",
      surface: "#0b1710",
      border: "#193225",
      text: "#c5d2c8",
      muted: "#516057",
      accent: "#82ce8b",
      path: "#b2c985",
    },
    {
      black: "#0f1a13",
      red: "#d9777f",
      green: "#82ce8b",
      yellow: "#cdb978",
      blue: "#6fb0c9",
      magenta: "#b89fc9",
      cyan: "#6fc2a8",
      white: "#b9c7bd",
      brightBlack: "#516057",
      brightRed: "#ea9097",
      brightGreen: "#9fe0a7",
      brightYellow: "#e0cd8e",
      brightBlue: "#8cc6dc",
      brightMagenta: "#cdb7dc",
      brightCyan: "#8cd6c0",
      brightWhite: "#e2ece5",
    },
  ),
  theme(
    "graphite",
    "Graphite",
    "Neutral and focused",
    {
      background: "#0b0d10",
      surface: "#111419",
      border: "#242932",
      text: "#c5c9d0",
      muted: "#555d68",
      accent: "#80b89d",
      path: "#91aeb6",
    },
    {
      black: "#171a20",
      red: "#e07b86",
      green: "#80b89d",
      yellow: "#d5b97e",
      blue: "#7aa7d8",
      magenta: "#b59ad6",
      cyan: "#7fc0cf",
      white: "#b9bec7",
      brightBlack: "#555d68",
      brightRed: "#ee949d",
      brightGreen: "#9ccfb5",
      brightYellow: "#e6cd96",
      brightBlue: "#96bde6",
      brightMagenta: "#c9b2e6",
      brightCyan: "#9ad3df",
      brightWhite: "#e4e7ec",
    },
  ),
  theme(
    "amber",
    "Warm amber",
    "Classic terminal",
    {
      background: "#100d08",
      surface: "#17120b",
      border: "#302719",
      text: "#d2c9b9",
      muted: "#665d50",
      accent: "#d0a85d",
      path: "#b9bd83",
    },
    {
      black: "#1c160d",
      red: "#d96f5f",
      green: "#a7b86a",
      yellow: "#d0a85d",
      blue: "#7c9fbf",
      magenta: "#b58aa0",
      cyan: "#78b3a8",
      white: "#cfc5b2",
      brightBlack: "#665d50",
      brightRed: "#e88a7b",
      brightGreen: "#bccb82",
      brightYellow: "#e3be78",
      brightBlue: "#96b7d3",
      brightMagenta: "#caa2b8",
      brightCyan: "#93c8be",
      brightWhite: "#efe6d4",
    },
  ),
];

/** Unknown ids (say, from an older settings file) fall back to the first theme. */
export function themeById(id: string): TerminalTheme {
  return THEMES.find((t) => t.id === id) ?? THEMES[0];
}
