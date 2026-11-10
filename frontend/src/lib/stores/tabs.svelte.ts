import type { HostInfo } from "../types";

export type TabStatus = "connecting" | "connected" | "ended" | "error";
export type Tab = {
  id: string;
  hostId: string;
  title: string;
  subtitle: string; // user@address
  os: string;
  osLike: string;
  osName: string;
  status: TabStatus;
  error: string;
};

class TabStore {
  items = $state<Tab[]>([]);
  activeId = $state("");
  private counter = 0;

  open(host: HostInfo) {
    const id = `t${++this.counter}`;
    this.items.push({
      id,
      hostId: host.id,
      title: host.name,
      subtitle: `${host.user}@${host.address}`,
      os: host.os,
      osLike: host.osLike,
      osName: "",
      status: "connecting",
      error: "",
    });
    this.activeId = id;
  }

  close(id: string) {
    const i = this.items.findIndex((t) => t.id === id);
    if (i < 0) return;
    this.items.splice(i, 1);
    if (this.activeId === id) {
      this.activeId = (this.items[i] ?? this.items[i - 1])?.id ?? "";
    }
  }

  activate(id: string) {
    this.activeId = id;
  }

  /** Switch to the next (1) or previous (-1) tab, wrapping around. */
  cycle(dir: 1 | -1) {
    if (this.items.length < 2) return;
    const i = this.items.findIndex((t) => t.id === this.activeId);
    this.activeId =
      this.items[(i + dir + this.items.length) % this.items.length].id;
  }

  setStatus(id: string, status: TabStatus, error = "") {
    const t = this.items.find((t) => t.id === id);
    if (t) {
      t.status = status;
      t.error = error;
    }
  }

  setOS(id: string, os: string, like: string, name: string) {
    const t = this.items.find((t) => t.id === id);
    if (t) {
      t.os = os;
      t.osLike = like;
      t.osName = name;
    }
  }
}

export const tabs = new TabStore();
