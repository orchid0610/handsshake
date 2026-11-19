export type VaultState = "none" | "locked" | "unlocked";

export interface HostInfo {
  id: string;
  name: string;
  address: string;
  user: string;
  auth: "password" | "key";
  keyPath: string;
  hasPassword: boolean;
  keyInVault: boolean;
  hasPassphrase: boolean;
  os: string;
  osLike: string;
  osName: string;
  lastUsed: number;
}

export interface HostInput {
  id: string;
  name: string;
  address: string;
  user: string;
  auth: "password" | "key";
  password: string;
  keyPath: string;
  importKey: boolean;
  passphrase: string;
  clearPassphrase: boolean;
}

export interface Settings {
  theme: string;
  fontFamily: "jetbrains-mono" | "system";
  fontSize: number;
  lineHeight: number;
  cursorStyle: "block" | "bar" | "underline";
  cursorBlink: boolean;
  scrollback: number;
  autoLockMinutes: number;
  compactSpacing: boolean;
  showStatusBar: boolean;
}

export interface ImportCandidate {
  alias: string;
  address: string;
  user: string;
  keyPath: string;
  keyStatus: "ok" | "encrypted" | "missing" | "invalid" | "none";
  status: "new" | "exists" | "unsupported";
  note: string;
}

export interface ImportPreview {
  path: string;
  candidates: ImportCandidate[];
  warnings: string[];
}

export interface ConnectInfo {
  hostKeyType: string;
}

export interface Snippet {
  id: string;
  title: string;
  code: string;
}

export interface KeyInfo {
  id: string
  name: string
  source: 'file' | 'vault'
  path: string
  type: string
  fingerprint: string
  encrypted: boolean
  status: 'ok' | 'locked' | 'missing' | 'invalid'
  usedBy: string[]
}
