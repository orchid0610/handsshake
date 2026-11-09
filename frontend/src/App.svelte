<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Terminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import '@xterm/xterm/css/xterm.css'
  import { ListHosts, AddHost, RemoveHost, Connect, Write, Resize, Disconnect } from '../wailsjs/go/main/App'
  import { EventsOn } from '../wailsjs/runtime/runtime'

  type Tab = {
    id: string
    host: string
    term: Terminal
    fit: FitAddon
    el: HTMLDivElement
    off: (() => void)[]
  }

  let hosts: any[] = []
  let error = ''
  let showForm = false
  let form = { name: '', address: '', user: '', keyPath: '' }

  let tabs: Tab[] = []
  let activeId = ''
  let termHost: HTMLDivElement
  let counter = 0
  let observer: ResizeObserver

  $: activeHost = tabs.find((t) => t.id === activeId)?.host ?? ''

  async function refresh() {
    hosts = (await ListHosts()) ?? []
  }

  // Show one tab's terminal, hide the others.
  function activate(id: string) {
    activeId = id
    for (const t of tabs) t.el.style.display = t.id === id ? 'block' : 'none'
    const t = tabs.find((t) => t.id === id)
    if (t) {
      t.fit.fit() // must be visible to measure correctly
      t.term.focus()
    }
  }

  // Tear down one tab completely.
  function removeTab(id: string) {
    const t = tabs.find((t) => t.id === id)
    if (!t) return
    t.off.forEach((f) => f())
    t.term.dispose()
    t.el.remove()
    Disconnect(id) // safe to call twice; Go ignores unknown ids
    tabs = tabs.filter((x) => x.id !== id)
    if (activeId === id) {
      if (tabs.length) activate(tabs[tabs.length - 1].id)
      else activeId = ''
    }
  }

  async function openTab(host: string) {
    error = ''
    const id = `term-${++counter}`

    // Each tab gets its own container element, stacked on top of each other.
    const el = document.createElement('div')
    el.style.cssText = 'position:absolute;inset:0;padding:8px;box-sizing:border-box;display:none'
    termHost.appendChild(el)

    const term = new Terminal({
      fontFamily: 'JetBrains Mono, monospace',
      fontSize: 14,
      cursorBlink: true,
      theme: { background: '#0f1115' },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(el)

    const tab: Tab = { id, host, term, fit, el, off: [] }
    tabs = [...tabs, tab]
    activate(id)

    // Subscribe BEFORE connecting so the first prompt isn't missed.
    tab.off.push(
      EventsOn(`term:data:${id}`, (b64: string) => {
        term.write(Uint8Array.from(atob(b64), (c) => c.charCodeAt(0)))
      }),
      EventsOn(`term:closed:${id}`, () => removeTab(id)),
    )
    term.onData((d) => Write(id, d))
    term.onResize(({ cols, rows }) => Resize(id, cols, rows))

    try {
      await Connect(id, host, term.cols, term.rows)
    } catch (e) {
      error = String(e)
      removeTab(id)
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
    observer = new ResizeObserver(() => {
      tabs.find((t) => t.id === activeId)?.fit.fit()
    })
    observer.observe(termHost)
  })

  onDestroy(() => {
    observer?.disconnect()
    tabs.forEach((t) => Disconnect(t.id))
  })
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
          <button class="host" on:click={() => openTab(h.name)}>
            <strong>{h.name}</strong>
            <span>{h.user}@{h.address}</span>
          </button>
          <button class="icon" title="Remove" on:click={() => removeHost(h.name)}>✕</button>
        </li>
      {/each}
    </ul>
  </aside>

  <section>
    <div class="tabs">
      {#each tabs as t (t.id)}
        <div class="tab" class:active={t.id === activeId}>
          <button class="label" on:click={() => activate(t.id)}>{t.host}</button>
          <button class="x" title="Close" on:click={() => removeTab(t.id)}>✕</button>
        </div>
      {/each}
    </div>
    <div class="termhost" bind:this={termHost}></div>
    {#if tabs.length === 0}<div class="empty">Select a host to connect</div>{/if}
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

  section { flex: 1; min-width: 0; display: flex; flex-direction: column; position: relative; }
  .tabs { display: flex; gap: 2px; padding: 6px 8px 0; background: #151821;
    border-bottom: 1px solid #23283a; overflow-x: auto; min-height: 34px; box-sizing: border-box; }
  .tab { display: flex; align-items: center; border-radius: 8px 8px 0 0; background: #1a1e2b; }
  .tab.active { background: #0f1115; }
  .label { background: none; border: none; color: #aab1c5; padding: 6px 4px 6px 12px;
    cursor: pointer; font-size: 13px; }
  .tab.active .label { color: #fff; }
  .x { background: none; border: none; color: #6b7389; cursor: pointer; padding: 6px 10px 6px 4px; font-size: 12px; }
  .x:hover { color: #f87171; }
  .termhost { position: relative; flex: 1; min-height: 0; }
  .empty { position: absolute; inset: 34px 0 0 0; display: grid; place-items: center;
    color: #5b6378; pointer-events: none; }
</style>