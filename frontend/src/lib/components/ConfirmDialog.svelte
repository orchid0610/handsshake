<script lang="ts">
  import { app } from '../stores/app.svelte'
  import Modal from './Modal.svelte'
</script>

{#if app.confirm}
  {@const c = app.confirm}
  <Modal title={c.title} width={400} onclose={() => (app.confirm = null)}>
    <p class="msg">{c.message}</p>
    <div class="actions">
      <button class="btn" onclick={() => (app.confirm = null)}>Cancel</button>
      <button class="btn btn-danger" onclick={() => { const run = c.run; app.confirm = null; run() }}>
        {c.action}
      </button>
    </div>
  </Modal>
{/if}

<style>
  .msg { color: var(--text-dim); line-height: 1.55; }
  .actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
</style>