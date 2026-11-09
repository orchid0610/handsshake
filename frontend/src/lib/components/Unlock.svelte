<script lang="ts">
  import { app } from '../stores/app.svelte'
  import { api, errText } from '../api'
  import { focus } from '../actions'
  import Icon from './Icon.svelte'

  let mode = $state<'unlock' | 'recover'>('unlock')
  let pw = $state('')
  let pw2 = $state('')
  let rk = $state('')
  let error = $state('')
  let busy = $state(false)
  let saved = $state(false)
  let copied = $state(false)

  const view = $derived(
    app.recoveryKey ? 'recoveryKey'
      : app.vault === 'none' ? 'setup'
      : app.vault === 'locked' ? mode
      : 'loading',
  )

  const strength = $derived(pw.length === 0 ? 0 : pw.length < 8 ? 1 : pw.length < 12 ? 2 : pw.length < 16 ? 3 : 4)
  const strengthLabel = ['', 'Too short', 'Okay', 'Good', 'Strong']

  function go(m: 'unlock' | 'recover') {
    mode = m; pw = ''; pw2 = ''; rk = ''; error = ''
  }

  async function run(job: () => Promise<void>) {
    error = ''
    busy = true
    try { await job() } catch (e) { error = errText(e) } finally { busy = false }
  }

  const setup = () => run(async () => {
    if (pw.length < 8) throw 'Use at least 8 characters.'
    if (pw !== pw2) throw 'The passwords do not match.'
    app.recoveryKey = await api.createVault(pw)
    app.vault = 'unlocked'
    pw = ''; pw2 = ''
  })

  const unlock = () => run(async () => {
    await api.unlock(pw)
    pw = ''
    await app.finishUnlock()
  })

  const recover = () => run(async () => {
    if (!rk.trim()) throw 'Enter your recovery key.'
    if (pw.length < 8) throw 'Use at least 8 characters for the new password.'
    if (pw !== pw2) throw 'The passwords do not match.'
    await api.recover(rk, pw)
    pw = ''; pw2 = ''; rk = ''
    await app.finishUnlock()
    app.toast('success', 'Password reset. You are back in.')
  })

  async function copyKey() {
    try {
      await navigator.clipboard.writeText(app.recoveryKey)
      copied = true
      setTimeout(() => (copied = false), 2000)
    } catch {
      app.toast('error', 'Could not copy automatically. Select the key and copy it by hand.')
    }
  }

  function finishRecoveryScreen() {
    saved = false
    app.acknowledgeRecoveryKey()
  }
</script>

<div class="gate">
  <div class="card">
    <div class="logo"><Icon name="shield" size={26} /></div>

    {#if view === 'loading'}
      <div class="spinner"></div>

    {:else if view === 'setup'}
      <h1>Create your vault</h1>
      <p class="sub">Your hosts and credentials are encrypted with a master password. It never leaves this computer.</p>
      <form onsubmit={(e) => { e.preventDefault(); setup() }}>
        <div class="field">
          <label for="pw">Master password</label>
          <input id="pw" class="input" type="password" bind:value={pw} use:focus autocomplete="off" />
          {#if pw}
            <div class="meter s{strength}"><i></i><i></i><i></i><i></i></div>
            <span class="hint">{strengthLabel[strength]}. A phrase of several words works well.</span>
          {/if}
        </div>
        <div class="field">
          <label for="pw2">Confirm password</label>
          <input id="pw2" class="input" type="password" bind:value={pw2} autocomplete="off" />
        </div>
        <p class="warn">There is no "forgot password" button. You'll get a recovery key next. Keep it safe.</p>
        {#if error}<p class="error-text">{error}</p>{/if}
        <button class="btn btn-primary btn-block" disabled={busy}>{busy ? 'Creating…' : 'Create vault'}</button>
      </form>

    {:else if view === 'recoveryKey'}
      <h1>Save your recovery key</h1>
      <p class="sub">If you ever forget your master password, this key is the <strong>only</strong> way back in. It is shown once and can't be retrieved later.</p>
      <div class="keybox selectable">
        {#each app.recoveryKey.split('-') as part}<span>{part}</span>{/each}
      </div>
      <button class="btn btn-block" onclick={copyKey}>
        <Icon name={copied ? 'check' : 'copy'} /> {copied ? 'Copied' : 'Copy to clipboard'}
      </button>
      <label class="checkbox ack">
        <input type="checkbox" bind:checked={saved} />
        I've saved my recovery key somewhere safe, such as a password manager.
      </label>
      <button class="btn btn-primary btn-block" disabled={!saved} onclick={finishRecoveryScreen}>Continue</button>

    {:else if view === 'unlock'}
      <h1>Welcome back</h1>
      <p class="sub">Enter your master password to unlock your vault.</p>
      <form onsubmit={(e) => { e.preventDefault(); unlock() }}>
        <div class="field">
          <label for="upw">Master password</label>
          <input id="upw" class="input" type="password" bind:value={pw} use:focus autocomplete="off" />
        </div>
        {#if error}<p class="error-text">{error}</p>{/if}
        <button class="btn btn-primary btn-block" disabled={busy || !pw}>{busy ? 'Unlocking…' : 'Unlock'}</button>
      </form>
      <button class="link" onclick={() => go('recover')}>Forgot your password? Use your recovery key</button>

    {:else if view === 'recover'}
      <h1>Reset with recovery key</h1>
      <p class="sub">Enter the recovery key you saved, then choose a new master password.</p>
      <form onsubmit={(e) => { e.preventDefault(); recover() }}>
        <div class="field">
          <label for="rk">Recovery key</label>
          <input id="rk" class="input mono" bind:value={rk} use:focus placeholder="XXXX-XXXX-XXXX-…" autocomplete="off" spellcheck="false" />
        </div>
        <div class="field">
          <label for="npw">New master password</label>
          <input id="npw" class="input" type="password" bind:value={pw} autocomplete="off" />
        </div>
        <div class="field">
          <label for="npw2">Confirm new password</label>
          <input id="npw2" class="input" type="password" bind:value={pw2} autocomplete="off" />
        </div>
        {#if error}<p class="error-text">{error}</p>{/if}
        <button class="btn btn-primary btn-block" disabled={busy}>{busy ? 'Working…' : 'Reset password'}</button>
      </form>
      <button class="link" onclick={() => go('unlock')}>Back to unlock</button>
    {/if}
  </div>
</div>

<style>
  .gate {
    position: fixed; inset: 0; z-index: 1000; display: grid; place-items: center; padding: 20px;
    background:
      radial-gradient(900px 500px at 50% -10%, rgba(91, 124, 250, 0.16), transparent 70%),
      var(--bg);
  }
  .card {
    width: 100%; max-width: 410px; padding: 34px 30px 28px;
    display: grid; gap: 16px; justify-items: stretch;
    background: var(--surface); border: 1px solid var(--border-strong);
    border-radius: 20px; box-shadow: var(--shadow);
  }
  .logo {
    justify-self: center; display: grid; place-items: center; width: 54px; height: 54px;
    border-radius: 16px; color: #fff;
    background: linear-gradient(135deg, #5b7cfa, #8b5cf6);
    box-shadow: 0 8px 24px rgba(91, 124, 250, 0.35);
  }
  h1 { text-align: center; font-size: 20px; font-weight: 650; letter-spacing: -0.01em; }
  .sub { text-align: center; color: var(--text-dim); font-size: 13.5px; line-height: 1.55; }
  .sub strong { color: var(--text); }
  form { display: grid; gap: 14px; margin-top: 4px; }
  .warn {
    padding: 10px 12px; font-size: 12.5px; line-height: 1.5; color: var(--warning);
    background: rgba(251, 191, 36, 0.08); border: 1px solid rgba(251, 191, 36, 0.2); border-radius: var(--radius-sm);
  }
  .mono { font-family: var(--font-mono); font-size: 13px; }
  .meter { display: grid; grid-template-columns: repeat(4, 1fr); gap: 4px; }
  .meter i { height: 3px; border-radius: 3px; background: var(--border-strong); }
  .meter.s1 i:nth-child(-n + 1) { background: var(--danger); }
  .meter.s2 i:nth-child(-n + 2) { background: var(--warning); }
  .meter.s3 i:nth-child(-n + 3) { background: #a3e635; }
  .meter.s4 i { background: var(--success); }
  .keybox {
    display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px 6px; padding: 14px;
    font-family: var(--font-mono); font-size: 13px; text-align: center; letter-spacing: 0.03em;
    background: var(--bg); border: 1px dashed var(--border-strong); border-radius: var(--radius);
  }
  .ack { margin: 2px 0; }
  .link { background: none; border: none; color: var(--text-dim); font-size: 13px; cursor: pointer; justify-self: center; padding: 2px; }
  .link:hover { color: var(--text); text-decoration: underline; }
  .spinner {
    justify-self: center; width: 22px; height: 22px; border-radius: 50%;
    border: 2px solid var(--border-strong); border-top-color: var(--accent); animation: spin 0.8s linear infinite;
  }
</style>