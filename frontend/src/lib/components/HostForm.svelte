<script lang="ts">
  import { app } from '../stores/app.svelte'
  import { api, errText } from '../api'
  import { focus } from '../actions'
  import Modal from './Modal.svelte'
  import Icon from './Icon.svelte'

  const host = app.editor.hostId ? (app.hosts.find((h) => h.id === app.editor.hostId) ?? null) : null

  let f = $state({
    name: host?.name ?? '',
    address: host?.address ?? '',
    user: host?.user ?? '',
    auth: (host?.auth ?? 'password') as 'password' | 'key',
    password: '',
    keyPath: host?.keyPath ?? '',
    importKey: false,
    passphrase: '',
    clearPassphrase: false,
  })
  let showPw = $state(false)
  let busy = $state(false)
  let error = $state('')

  async function browse() {
    try {
      const p = await api.pickKeyFile()
      if (p) f.keyPath = p
    } catch (e) {
      error = errText(e)
    }
  }

  async function save() {
    error = ''
    busy = true
    try {
      await api.saveHost({ id: host?.id ?? '', ...f })
      await app.refreshHosts()
      app.toast('success', host ? 'Host updated' : 'Host saved')
      app.closeEditor()
    } catch (e) {
      error = errText(e)
    } finally {
      busy = false
    }
  }
</script>

<Modal title={host ? 'Edit host' : 'New host'} onclose={() => app.closeEditor()}>
  <form onsubmit={(e) => { e.preventDefault(); save() }}>
    <div class="field">
      <label for="name">Name</label>
      <input id="name" class="input" bind:value={f.name} placeholder="Production server" use:focus autocomplete="off" />
    </div>

    <div class="grid">
      <div class="field">
        <label for="addr">Address</label>
        <input id="addr" class="input" bind:value={f.address} placeholder="192.168.1.10 or host:2222" autocomplete="off" spellcheck="false" />
      </div>
      <div class="field">
        <label for="user">Username</label>
        <input id="user" class="input" bind:value={f.user} placeholder="ubuntu" autocomplete="off" spellcheck="false" />
      </div>
    </div>

    <div class="field">
      <label for="auth-seg">Authentication</label>
      <div id="auth-seg" class="seg" role="group">
        <button type="button" class:on={f.auth === 'password'} onclick={() => (f.auth = 'password')}>Password</button>
        <button type="button" class:on={f.auth === 'key'} onclick={() => (f.auth = 'key')}>Private key</button>
      </div>
    </div>

    {#if f.auth === 'password'}
      <div class="field">
        <label for="pw">Password</label>
        <div class="with-btn">
          <input id="pw" class="input" type={showPw ? 'text' : 'password'}
                 value={f.password} oninput={(e) => (f.password = e.currentTarget.value)}
                 placeholder={host?.hasPassword ? '•••••••• (leave empty to keep)' : ''} autocomplete="off" />
          <button type="button" class="icon-btn" onclick={() => (showPw = !showPw)} aria-label="Show or hide password">
            <Icon name={showPw ? 'eyeOff' : 'eye'} />
          </button>
        </div>
      </div>
    {:else}
      <div class="field">
        <label for="key">Private key file</label>
        <div class="with-btn">
          <input id="key" class="input" bind:value={f.keyPath} placeholder="~/.ssh/id_ed25519" autocomplete="off" spellcheck="false" />
          <button type="button" class="btn" onclick={browse}><Icon name="folder" /> Browse</button>
        </div>
        {#if host?.keyInVault && !f.keyPath}
          <span class="hint">A key is stored in your vault. Leave this empty to keep it.</span>
        {/if}
      </div>
      <label class="checkbox">
        <input type="checkbox" bind:checked={f.importKey} />
        <span>Store a copy of the key inside the encrypted vault, so it doesn't depend on this file.</span>
      </label>
      <div class="field">
        <label for="pp">Key passphrase <span class="opt">(optional)</span></label>
        <input id="pp" class="input" type="password" bind:value={f.passphrase}
               placeholder={host?.hasPassphrase ? '•••••••• (leave empty to keep)' : 'Only if the key is protected'} autocomplete="off" />
        {#if host?.hasPassphrase}
          <label class="checkbox"><input type="checkbox" bind:checked={f.clearPassphrase} /> Remove the saved passphrase</label>
        {/if}
      </div>
    {/if}

    {#if error}<p class="error-text">{error}</p>{/if}

    <div class="actions">
      <button type="button" class="btn" onclick={() => app.closeEditor()}>Cancel</button>
      <button class="btn btn-primary" disabled={busy}>{busy ? 'Saving…' : 'Save'}</button>
    </div>
  </form>
</Modal>

<style>
  form { display: grid; gap: 16px; }
  .grid { display: grid; grid-template-columns: 1.4fr 1fr; gap: 12px; }
  .opt { color: var(--text-faint); font-weight: 400; }
  .with-btn { display: flex; gap: 8px; align-items: center; }
  .seg { display: inline-grid; grid-auto-flow: column; gap: 3px; padding: 3px; background: var(--bg); border: 1px solid var(--border-strong); border-radius: 9px; }
  .seg button { height: 30px; padding: 0 16px; border: none; border-radius: 7px; background: transparent; color: var(--text-dim); font-weight: 500; cursor: pointer; }
  .seg button.on { background: var(--surface-3); color: var(--text); box-shadow: 0 1px 2px rgba(0, 0, 0, 0.4); }
  .actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 4px; }
</style>