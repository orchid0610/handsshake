<script lang="ts">
  import { onMount } from 'svelte'
  import { app } from '../stores/app.svelte'
  import { api, events } from '../api'
  import Modal from './Modal.svelte'
  import Icon from './Icon.svelte'

  const current = $derived(app.prompts[0])

  onMount(() => events.on('hostkey:prompt', (p: any) => app.prompts.push(p)))

  function answer(trust: boolean) {
    if (!current) return
    api.answerHostKey(current.promptId, trust)
    app.prompts.shift()
  }
</script>

{#if current}
  <Modal title="Verify server identity" width={500} onclose={() => answer(false)}>
    <div class="intro">
      <span class="badge"><Icon name="shield" size={20} /></span>
      <p>
        You're connecting to <strong class="selectable">{current.host}</strong> for the first time.
        Check that the fingerprint below matches your server before trusting it.
      </p>
    </div>

    <div class="fp">
      <span class="type">{current.keyType}</span>
      <code class="selectable">{current.fingerprint}</code>
    </div>

    <p class="hint">
      To see the real fingerprint, run this on the server:
      <code class="inline selectable">ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub</code>
    </p>

    <div class="actions">
      <button class="btn" onclick={() => answer(false)}>Cancel</button>
      <button class="btn btn-primary" onclick={() => answer(true)}>Trust and connect</button>
    </div>
  </Modal>
{/if}

<style>
  .intro { display: flex; gap: 14px; align-items: flex-start; color: var(--text-dim); line-height: 1.55; }
  .intro strong { color: var(--text); }
  .badge {
    flex: none; display: grid; place-items: center; width: 38px; height: 38px;
    border-radius: 10px; background: var(--accent-soft); color: var(--accent);
  }
  .fp {
    margin: 18px 0 14px; padding: 12px 14px; background: var(--bg);
    border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
  }
  .type { display: block; font-size: 11px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-faint); margin-bottom: 4px; }
  code { font-family: var(--font-mono); font-size: 13px; word-break: break-all; }
  code.inline { display: block; margin-top: 6px; padding: 8px 10px; background: var(--bg); border-radius: 6px; color: var(--text-dim); }
  .actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
</style>