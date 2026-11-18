import { api, errText } from "../api";
import type { Snippet } from "../types";
import { app } from "./app.svelte";
import { tabs } from "./tabs.svelte";

type Form = { id: string; title: string; code: string };

class SnippetStore {
  items = $state<Snippet[]>([]);
  loaded = $state(false);
  open = $state(false);
  query = $state("");
  form = $state<Form | null>(null); // null means the list is showing
  busy = $state(false);
  formError = $state("");

  filtered = $derived.by(() => {
    const q = this.query.trim().toLowerCase();
    return q
      ? this.items.filter((s) =>
          `${s.title} ${s.code}`.toLowerCase().includes(q),
        )
      : this.items;
  });

  async load() {
    try {
      this.items = (await api.listSnippets()) ?? [];
      this.loaded = true;
    } catch (e) {
      app.toast("error", errText(e));
    }
  }

  /** Forget everything (the vault was locked). */
  clear() {
    this.items = [];
    this.loaded = false;
    this.open = false;
    this.query = "";
    this.form = null;
    this.formError = "";
  }

  toggle() {
    if (this.open) {
      this.close();
    } else {
      app.pickerOpen = false;
      this.open = true;
    }
  }

  close() {
    this.open = false;
    this.form = null;
    this.formError = "";
    this.query = "";
    tabs.focusActive();
  }

  startNew() {
    this.formError = "";
    this.form = { id: "", title: "", code: "" };
  }

  startEdit(s: Snippet) {
    this.formError = "";
    this.form = { id: s.id, title: s.title, code: s.code };
  }

  cancelForm() {
    this.form = null;
    this.formError = "";
  }

  async save() {
    const f = this.form;
    if (!f || this.busy) return;
    this.formError = "";
    if (!f.title.trim()) {
      this.formError = "Give the snippet a name.";
      return;
    }
    if (!f.code.trim()) {
      this.formError = "Enter a command or some code.";
      return;
    }

    this.busy = true;
    try {
      await api.saveSnippet(f.id, f.title, f.code);
      await this.load();
      app.toast("success", f.id ? "Snippet updated" : "Snippet saved");
      this.form = null;
    } catch (e) {
      this.formError = errText(e);
    } finally {
      this.busy = false;
    }
  }

  remove(s: Snippet) {
    app.confirm = {
      title: "Delete snippet?",
      message: `“${s.title}” will be permanently removed from your vault.`,
      action: "Delete",
      run: async () => {
        try {
          await api.deleteSnippet(s.id);
          await this.load();
          app.toast("success", "Snippet deleted");
        } catch (e) {
          app.toast("error", errText(e));
        }
      },
    };
  }

  /** Put the snippet on the active session's command line. It never presses Enter. */
  paste(s: Snippet) {
    const tab = tabs.items.find((t) => t.id === tabs.activeId);
    if (!tab || tab.status !== "connected") {
      app.toast("error", "Open a connected session first, then paste.");
      return;
    }
    tabs.handles.get(tab.id)?.paste(s.code);
    this.close();
    app.toast(
      "success",
      `Pasted “${s.title}”. Review it, then press Enter to run.`,
    );
  }
}

export const snippets = new SnippetStore();
