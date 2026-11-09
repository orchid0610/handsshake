<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Terminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import '@xterm/xterm/css/xterm.css'
  import { ListHosts, AddHost, RemoveHost, Connect, Write, Resize, Disconnect } from '../wailsjs/go/main/App'
  import { EventsOn } from '../wailsjs/runtime/runtime'

  let hosts: any[] = []
  let error = ''
  let showForm = false
  let form = { name: '', address: '', user: '', keyPath: '' }

  let termEl: HTMLDivElement
  let term: Terminal | null = null
  let fit: FitAddon | null = null
  let activeId = ''
  let activeHost = ''
  let offData: (() => void) | null = null
  let offClosed: (() => void) | null = null
  let counter = 0
  let observer: ResizeObserver

  async function refresh() {
    hosts = (await ListHosts()) ?? []
  }

  function closeTerminal() {
    offData?.()
    offClosed?.()
    offData = offClosed = null
    term?.dispose()
    term = null
    activeId = ''
    activeHost = ''
  }

  async function connectTo(name: string) {
    error = ''
    if (activeId) {
      await Disconnect(activeId)
      closeTerminal()
    }

    const id = `term-${++counter}`
    term = new Terminal({
      fontFamily: 'JetBrains Mono, monospace',
      fontSize: 14,
      cursorBlink: true,
      theme: { background: '#0f1115' },
    })
    fit = new FitAddon()
    term.loadAddon(fit)
    term.open(termEl)
    fit.fit()

    // Subscribe BEFORE connecting so the first prompt isn't missed
    offData = EventsOn(`term:data:${id}`, (b64: string) => {
      const bytes = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0))
      term?.write(bytes)
    })
    offClosed = EventsOn(`term:closed:${id}`, () => closeTerminal())

    term.onData((d) => Write(id, d))
    term.onResize(({ cols, rows }) => Resize(id, cols, rows))

    try {
      await Connect(id, name, term.cols, term.rows)
      activeId = id
      activeHost = name
      term.focus()
    } catch (e) {
      error = String(e)
      closeTerminal()
    }
  }

  async function saveHost() {
    error = ''
    try {
      await AddHost(form.name, form.address, form.user, form.keyPath)
      form = { name: '', address: '', user: '', keyPath: '' }
      showForm = false
      await refresh()
    } catch (e) {
      error = String(e)
    }
  }

  async function removeHost(name: string) {
    await RemoveHost(name)
    await refresh()
  }

  onMount(() => {
    refresh()
    observer = new ResizeObserver(() => fit?.fit())
    observer.observe(termEl)
  })

  onDestroy(() => observer?.disconnect())
</script>

<main>
  <aside>
    <div class="head">
      <h1>Hosts</h1>
      <button class="icon" on:click={() => (showForm = !showForm)}>＋</button>
    </div>

    {#if showForm}
      <form on:submit={(e) => { e.preventDefault(); saveHost() }}>
        <input placeholder="Name" bind:value={form.name} />
        <input placeholder="Address (host:22)" bind:value={form.address} />
        <input placeholder="User" bind:value={form.user} />
        <input placeholder="Key path (/home/you/.ssh/key)" bind:value={form.keyPath} />
        <button type="submit">Save</button>
      </form>
    {/if}

    {#if error}<p class="error">{error}</p>{/if}

    <ul>
      {#each hosts as h (h.name)}
        <li class:active={h.name === activeHost}>
          <button class="host" on:click={() => connectTo(h.name)}>
            <strong>{h.name}</strong>
            <span>{h.user}@{h.address}</span>
          </button>
          <button class="icon" title="Remove" on:click={() => removeHost(h.name)}>✕</button>
        </li>
      {/each}
    </ul>
  </aside>

  <section>
    <div class="term" bind:this={termEl}></div>
    {#if !activeId}<div class="empty">Select a host to connect</div>{/if}
  </section>
</main>

<style>
  :global(body) { margin: 0; background: #0f1115; color: #e6e6e6; text-align: left;
    font-family: system-ui, sans-serif; }
  :global(#app) { height: 100vh; text-align: left; }
  main { display: flex; height: 100vh; }
  aside { width: 260px; box-sizing: border-box; padding: 14px; overflow-y: auto;
    background: #151821; border-right: 1px solid #23283a; }
  .head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
  h1 { margin: 0; font-size: 12px; letter-spacing: 0.1em; text-transform: uppercase; color: #8b93a7; }
  ul { list-style: none; margin: 0; padding: 0; }
  li { display: flex; align-items: center; border-radius: 8px; }
  li:hover, li.active { background: #1e2330; }
  .host { flex: 1; display: flex; flex-direction: column; align-items: flex-start; gap: 2px;
    padding: 8px 10px; background: none; border: none; color: inherit; cursor: pointer; text-align: left; }
  .host span { font-size: 12px; color: #8b93a7; }
  .icon { background: none; border: none; color: #8b93a7; cursor: pointer; font-size: 14px; padding: 4px 8px; }
  .icon:hover { color: #fff; }
  form { display: flex; flex-direction: column; gap: 6px; margin-bottom: 12px; }
  input { padding: 7px 9px; border-radius: 6px; border: 1px solid #2a3042;
    background: #0f1115; color: inherit; }
  form button { padding: 7px; border: none; border-radius: 6px; background: #3b82f6;
    color: white; cursor: pointer; }
  .error { color: #f87171; font-size: 12px; word-break: break-word; }
  section { position: relative; flex: 1; min-width: 0; }
  .term { position: absolute; inset: 0; padding: 8px; box-sizing: border-box; }
  .empty { position: absolute; inset: 0; display: grid; place-items: center;
    color: #5b6378; pointer-events: none; }
</style>