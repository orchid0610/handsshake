import { tick } from "svelte";
import type { HostInfo } from "../types";
import { app } from "./app.svelte";

export type TabStatus = "connecting" | "connected" | "ended" | "error";
export type Tab = {
  id: string;
  hostId: string;
  title: string;
  subtitle: string; // user@address
  os: string;
  osLike: string;
  osName: string;
  keyType: string; // host key algorithm the server presented
  status: TabStatus;
  error: string;
};

/** What other parts of the UI may ask a tab's terminal to do. */
export type TermHandle = {
  paste: (text: string) => void;
  focus: () => void;
  find: () => void
};

class TabStore {
  items = $state<Tab[]>([]);
  activeId = $state("");
  private counter = 0;

  // Plain (not reactive) on purpose: terminals register themselves here.
  handles = new Map<string, TermHandle>();

  open(host: HostInfo) {
    const id = `t${++this.counter}`;
    this.items.push({
      id,
      hostId: host.id,
      title: host.name,
      subtitle: `${host.user}@${host.address}`,
      os: host.os,
      osLike: host.osLike,
      osName: host.osName,
      keyType: "",
      status: "connecting",
      error: "",
    });
    this.activeId = id;
    app.page = "terminal";
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

  setKeyType(id: string, keyType: string) {
    const t = this.items.find((t) => t.id === id);
    if (t) t.keyType = keyType;
  }

  register(id: string, handle: TermHandle) {
    this.handles.set(id, handle);
  }

  unregister(id: string) {
    this.handles.delete(id);
  }

  /** Give keyboard focus back to the visible terminal (after a drawer or dialog closes). */
  focusActive() {
    tick().then(() => this.handles.get(this.activeId)?.focus());
  }
}

export const tabs = new TabStore();
