import { api, errText } from "../api";
import type { HostInfo, VaultState } from "../types";

export type Toast = {
  id: number;
  kind: "success" | "error" | "info";
  text: string;
};
export type HostKeyPrompt = {
  promptId: string;
  host: string;
  keyType: string;
  fingerprint: string;
};
export type Confirm = {
  title: string;
  message: string;
  action: string;
  run: () => void;
};

class AppStore {
  vault = $state<VaultState | "loading">("loading");
  recoveryKey = $state(""); // non-empty only while the "save your recovery key" screen is up
  hosts = $state<HostInfo[]>([]);
  query = $state("");
  toasts = $state<Toast[]>([]);
  prompts = $state<HostKeyPrompt[]>([]);
  confirm = $state<Confirm | null>(null);
  editor = $state<{ open: boolean; hostId: string | null }>({
    open: false,
    hostId: null,
  });
  settingsOpen = $state(false);
  importOpen = $state(false);

  private nextToast = 1;

  filtered = $derived.by(() => {
    const q = this.query.trim().toLowerCase();
    if (!q) return this.hosts;
    return this.hosts.filter((h) =>
      `${h.name} ${h.user}@${h.address}`.toLowerCase().includes(q),
    );
  });

  /** The lock screen shows whenever the vault isn't open, or the recovery key is pending. */
  get gated() {
    return this.vault !== "unlocked" || this.recoveryKey !== "";
  }

  async init() {
    try {
      this.vault = await api.vaultState();
      if (this.vault === "unlocked") await this.refreshHosts();
    } catch (e) {
      this.toast("error", errText(e));
    }
  }

  async refreshHosts() {
    try {
      this.hosts = (await api.listHosts()) ?? [];
    } catch (e) {
      this.toast("error", errText(e));
    }
  }

  async finishUnlock() {
    this.vault = "unlocked";
    await this.refreshHosts();
  }

  async acknowledgeRecoveryKey() {
    this.recoveryKey = "";
    await this.refreshHosts();
  }

  async lock() {
    await api.lock();
    this.vault = "locked";
    this.hosts = [];
    this.query = "";
    this.editor = { open: false, hostId: null };
    this.confirm = null;
    this.settingsOpen = false
    this.importOpen = false
    
  }

  openEditor(hostId: string | null = null) {
    this.editor = { open: true, hostId };
  }
  closeEditor() {
    this.editor = { open: false, hostId: null };
  }

  toast(kind: Toast["kind"], text: string) {
    const id = this.nextToast++;
    this.toasts.push({ id, kind, text });
    setTimeout(() => this.dismiss(id), kind === "error" ? 7000 : 3500);
  }
  dismiss(id: number) {
    this.toasts = this.toasts.filter((t) => t.id !== id);
  }
}

export const app = new AppStore();
