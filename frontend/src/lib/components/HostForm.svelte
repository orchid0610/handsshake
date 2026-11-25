<script lang="ts">
  import { app } from "../stores/app.svelte";
  import { api, errText } from "../api";
  import { focus } from "../actions";
  import { splitAddress, joinAddress } from "../address";
  import Modal from "./Modal.svelte";
  import Icon from "./Icon.svelte";

  const host = app.editor.hostId
    ? (app.hosts.find((h) => h.id === app.editor.hostId) ?? null)
    : null;
  const start = splitAddress(host?.address ?? "");

  let f = $state({
    name: host?.name ?? "",
    address: start.host,
    port: start.port,
    user: host?.user ?? "",
    auth: (host?.auth ?? "password") as "password" | "key",
    password: "",
    keyPath: host?.keyPath ?? "",
    importKey: false,
    passphrase: "",
    clearPassphrase: false,
    jumpHostId: host?.jumpHostId ?? "",
  });
  let showPw = $state(false);
  let busy = $state(false);
  let error = $state("");

  // A host can't connect through itself, or through a host that (eventually) connects through it.
  const byId = $derived(new Map(app.hosts.map((h) => [h.id, h])));
  function loopsBack(candidateId: string): boolean {
    if (!host) return false;
    let cur = byId.get(candidateId);
    for (let i = 0; cur && i < 10; i++) {
      if (cur.id === host.id) return true;
      cur = cur.jumpHostId ? byId.get(cur.jumpHostId) : undefined;
    }
    return false;
  }
  const choices = $derived(app.hosts.filter((h) => !loopsBack(h.id)));

  async function browse() {
    try {
      const p = await api.pickKeyFile();
      if (p) f.keyPath = p;
    } catch (e) {
      error = errText(e);
    }
  }

  async function save() {
    error = "";
    let address = f.address.trim();
    let port = f.port.trim() || "22";

    // Someone typed "host:2222" into the host field: honour it unless a port was set separately.
    const parsed = splitAddress(address);
    if (parsed.host !== address) {
      address = parsed.host;
      if (port === "22") port = parsed.port;
    }
    if (!address) {
      error = "Enter the server address.";
      return;
    }
    const n = Number(port);
    if (!Number.isInteger(n) || n < 1 || n > 65535) {
      error = "Port must be a number between 1 and 65535.";
      return;
    }

    busy = true;
    try {
      await api.saveHost({
        id: host?.id ?? "",
        name: f.name,
        address: joinAddress(address, String(n)),
        user: f.user,
        auth: f.auth,
        password: f.password,
        keyPath: f.keyPath,
        importKey: f.importKey,
        passphrase: f.passphrase,
        clearPassphrase: f.clearPassphrase,
        jumpHostId: f.jumpHostId,
      });
      await app.refreshHosts();
      app.toast("success", host ? "Host updated" : "Host saved");
      app.closeEditor();
    } catch (e) {
      error = errText(e);
    } finally {
      busy = false;
    }
  }
</script>

<Modal
  eyebrow="Secure shell"
  title={host ? "Edit connection" : "New connection"}
  onclose={() => app.closeEditor()}
>
  <form
    onsubmit={(e) => {
      e.preventDefault();
      save();
    }}
  >
    <div class="grid">
      <div class="field full">
        <label for="name">Name</label>
        <input
          id="name"
          class="input"
          bind:value={f.name}
          placeholder="Production server"
          use:focus
          autocomplete="off"
        />
      </div>

      <div class="field full">
        <label for="addr">Host</label>
        <input
          id="addr"
          class="input mono"
          bind:value={f.address}
          placeholder="hostname or IP address"
          autocomplete="off"
          spellcheck="false"
        />
      </div>

      <div class="field">
        <label for="user">Username</label>
        <input
          id="user"
          class="input mono"
          bind:value={f.user}
          placeholder="ubuntu"
          autocomplete="off"
          spellcheck="false"
        />
      </div>
      <div class="field">
        <label for="port">Port</label>
        <input
          id="port"
          class="input mono"
          bind:value={f.port}
          inputmode="numeric"
          autocomplete="off"
        />
      </div>

      <div class="field full">
        <label for="auth-seg">Authentication</label>
        <div id="auth-seg" class="seg" role="group">
          <button
            type="button"
            class:on={f.auth === "password"}
            onclick={() => (f.auth = "password")}
          >
            <Icon name="lock" size={14} /> Password
          </button>
          <button
            type="button"
            class:on={f.auth === "key"}
            onclick={() => (f.auth = "key")}
          >
            <Icon name="key" size={14} /> Private key
          </button>
        </div>
      </div>

      {#if f.auth === "password"}
        <div class="field full">
          <label for="pw">Password</label>
          <div class="with-btn">
            <input
              id="pw"
              class="input mono"
              type={showPw ? "text" : "password"}
              value={f.password}
              oninput={(e) => (f.password = e.currentTarget.value)}
              placeholder={host?.hasPassword
                ? "•••••••• (leave empty to keep)"
                : ""}
              autocomplete="off"
            />
            <button
              type="button"
              class="icon-btn"
              onclick={() => (showPw = !showPw)}
              aria-label="Show or hide password"
            >
              <Icon name={showPw ? "eyeOff" : "eye"} />
            </button>
          </div>
        </div>
      {:else}
        <div class="field full">
          <label for="key">Private key file</label>
          <div class="with-btn">
            <input
              id="key"
              class="input mono"
              bind:value={f.keyPath}
              placeholder="~/.ssh/id_ed25519"
              autocomplete="off"
              spellcheck="false"
            />
            <button type="button" class="btn" onclick={browse}
              ><Icon name="folder" /> Browse</button
            >
          </div>
          {#if host?.keyInVault && !f.keyPath}
            <span class="hint"
              >A key is stored in your vault. Leave this empty to keep it.</span
            >
          {/if}
        </div>
        <label class="checkbox full">
          <input type="checkbox" bind:checked={f.importKey} />
          <span
            >Store a copy of the key inside the encrypted vault, so it doesn't
            depend on this file.</span
          >
        </label>
        <div class="field full">
          <label for="pp"
            >Key passphrase <span class="opt">(optional)</span></label
          >
          <input
            id="pp"
            class="input mono"
            type="password"
            bind:value={f.passphrase}
            placeholder={host?.hasPassphrase
              ? "•••••••• (leave empty to keep)"
              : "Only if the key is protected"}
            autocomplete="off"
          />
          {#if host?.hasPassphrase}
            <label class="checkbox"
              ><input type="checkbox" bind:checked={f.clearPassphrase} /> Remove
              the saved passphrase</label
            >
          {/if}
        </div>
      {/if}

      {#if choices.length > 0}
        <div class="field full">
          <label for="jump"
            >Connect through <span class="opt">(optional)</span></label
          >
          <select id="jump" class="input" bind:value={f.jumpHostId}>
            <option value="">Connect directly</option>
            {#each choices as h (h.id)}
              <option value={h.id}>{h.name} · {h.user}@{h.address}</option>
            {/each}
          </select>
          <span class="hint"
            >Use another saved host as a jump host (bastion). It signs in first,
            then reaches this server from there.</span
          >
        </div>
      {/if}
    </div>

    {#if error}<p class="error-text">{error}</p>{/if}

    <footer>
      <button type="button" class="text-btn" onclick={() => app.closeEditor()}
        >Cancel</button
      >
      <button class="btn btn-primary" disabled={busy}>
        <Icon name="terminal" size={16} />
        {busy ? "Saving…" : "Save host"}
      </button>
    </footer>
  </form>
</Modal>

<style>
  .grid {
    display: grid;
    grid-template-columns: 1fr 110px;
    gap: 15px 12px;
  }
  .full {
    grid-column: 1 / -1;
  }
  .mono {
    font-family: var(--font-mono);
    font-size: 12.5px;
  }
  .opt {
    color: var(--text-faint);
    font-weight: 400;
  }
  .with-btn {
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .seg {
    display: inline-grid;
    grid-auto-flow: column;
    gap: 3px;
    padding: 3px;
    justify-self: start;
    background: var(--input-bg);
    border: 1px solid var(--border-strong);
    border-radius: 9px;
  }
  .seg button {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    height: 30px;
    padding: 0 14px;
    border: none;
    border-radius: 7px;
    background: transparent;
    color: var(--text-dim);
    font-weight: 500;
  }
  .seg button.on {
    background: var(--accent-soft);
    color: var(--accent-text);
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.22);
  }
  form {
    display: grid;
    gap: 4px;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 10px;
    margin-top: 22px;
    padding-top: 18px;
    border-top: 1px solid var(--border);
  }
  .text-btn {
    padding: 0 12px;
    border: none;
    background: none;
    color: var(--text-dim);
    font-size: 13px;
  }
  .text-btn:hover {
    color: var(--text);
  }
  .error-text {
    margin-top: 12px;
  }
</style>
