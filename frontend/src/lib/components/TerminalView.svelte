<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte'
  import { Terminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import '@xterm/xterm/css/xterm.css'
  import { api, events, errText } from '../api'
  import { tabs, type Tab } from '../stores/tabs.svelte'
  import Icon from './Icon.svelte'

  let { tab, active }: { tab: Tab; active: boolean } = $props()

  const THEME = {
    background: '#0d0f14', foreground: '#c9d1e0', cursor: '#7aa2f7', cursorAccent: '#0d0f14',
    selectionBackground: 'rgba(122, 162, 247, 0.28)',
    black: '#1b1f2b', red: '#f7768e', green: '#9ece6a', yellow: '#e0af68',
    blue: '#7aa2f7', magenta: '#bb9af7', cyan: '#7dcfff', white: '#a9b1d6',
    brightBlack: '#414868', brightRed: '#ff899d', brightGreen: '#b9f27c', brightYellow: '#ffc777',
    brightBlue: '#9ab8ff', brightMagenta: '#c9afff', brightCyan: '#a4daff', brightWhite: '#d5dbf0',
  }

  let el: HTMLDivElement
  let term: Terminal | undefined
  let fit: FitAddon | undefined
  let ro: ResizeObserver | undefined
  let sid = '' // id of the current backend session
  let attempt = 0
  let unsub: Array<() => void> = []
  let disposed = false

  function dropListeners() {
    unsub.forEach((f) => f())
    unsub = []
  }

  async function start() {
    if (!term) return
    dropListeners()
    if (sid) api.disconnect(sid) // harmless if it no longer exists
    tabs.setStatus(tab.id, 'connecting')
    term.reset()

    const id = `${tab.id}-${++attempt}`
    sid = id

    // Subscribe BEFORE connecting so the very first prompt isn't missed.
    unsub.push(
      events.on(`term:data:${id}`, (b64: string) => {
        term?.write(Uint8Array.from(atob(b64), (c) => c.charCodeAt(0)))
      }),
      events.on(`term:closed:${id}`, () => {
        if (sid === id) tabs.setStatus(tab.id, 'ended')
      }),
    )

    try {
      await api.connect(id, tab.hostId, term.cols, term.rows)
      if (disposed || sid !== id) { api.disconnect(id); return }
      if (tab.status === 'connecting') tabs.setStatus(tab.id, 'connected')
      if (active) term.focus()
    } catch (e) {
      if (sid === id) tabs.setStatus(tab.id, 'error', errText(e))
    }
  }

  onMount(async () => {
    // xterm measures character cells once; make sure the font is ready first.
    await document.fonts.load('13.5px "JetBrains Mono"').catch(() => {})
    if (disposed) return

    const t = new Terminal({
      fontFamily: '"JetBrains Mono", ui-monospace, monospace',
      fontSize: 13.5,
      lineHeight: 1.2,
      cursorBlink: true,
      scrollback: 5000,
      theme: THEME,
    })
    const f = new FitAddon()
    t.loadAddon(f)
    t.open(el)
    f.fit()
    term = t
    fit = f

    t.onData((d) => { if (tab.status === 'connected') api.write(sid, d).catch(() => {}) })
    t.onResize(({ cols, rows }) => { if (tab.status === 'connected') api.resize(sid, cols, rows) })

    // Ctrl+Shift+C copies the selection (plain Ctrl+C must reach the server).
    t.attachCustomKeyEventHandler((e) => {
      if (e.type === 'keydown' && e.ctrlKey && e.shiftKey && e.code === 'KeyC') {
        const sel = t.getSelection()
        if (sel) navigator.clipboard.writeText(sel).catch(() => {})
        e.preventDefault()
        return false
      }
      return true
    })

    ro = new ResizeObserver(() => { if (active) f.fit() })
    ro.observe(el)

    start()
  })

  // When this tab becomes the visible one, re-measure it and focus it.
  $effect(() => {
    if (active && term) {
      tick().then(() => { fit?.fit(); term?.focus() })
    }
  })

  onDestroy(() => {
    disposed = true
    dropListeners()
    if (sid) api.disconnect(sid)
    ro?.disconnect()
    term?.dispose()
  })
</script>

<div class="pane" class:hidden={!active}>
  <div class="term" bind:this={el}></div>

  {#if tab.status === 'connecting'}
    <div class="overlay">
      <div class="spinner"></div>
      <p>Connecting to <strong>{tab.title}</strong>…</p>
    </div>
  {:else if tab.status === 'error'}
    <div class="overlay">
      <span class="badge"><Icon name="alert" size={22} /></span>
      <h3>Couldn't connect</h3>
      <p class="msg selectable">{tab.error}</p>
      <div class="row">
        <button class="btn btn-primary" onclick={start}><Icon name="refresh" /> Try again</button>
        <button class="btn" onclick={() => tabs.close(tab.id)}>Close tab</button>
      </div>
    </div>
  {:else if tab.status === 'ended'}
    <div class="ended">
      <span>Session ended</span>
      <button class="btn btn-primary" onclick={start}><Icon name="refresh" /> Reconnect</button>
      <button class="btn" onclick={() => tabs.close(tab.id)}>Close</button>
    </div>
  {/if}
</div>

<style>
  .pane { position: absolute; inset: 0; background: var(--term-bg); }
  .hidden { display: none; }
  .term { position: absolute; inset: 0; padding: 10px 4px 6px 14px; }
  .overlay {
    position: absolute; inset: 0; display: grid; align-content: center; justify-items: center; gap: 12px;
    padding: 30px; text-align: center; color: var(--text-dim); background: rgba(13, 15, 20, 0.92);
  }
  .overlay strong { color: var(--text); }
  h3 { font-size: 17px; color: var(--text); }
  .msg { max-width: 460px; font-size: 13px; line-height: 1.55; word-break: break-word; }
  .badge { display: grid; place-items: center; width: 46px; height: 46px; border-radius: 14px; background: var(--danger-soft); color: var(--danger); }
  .spinner { width: 24px; height: 24px; border-radius: 50%; border: 2px solid var(--border-strong); border-top-color: var(--accent); animation: spin 0.8s linear infinite; }
  .ended {
    position: absolute; left: 50%; bottom: 22px; transform: translateX(-50%);
    display: flex; align-items: center; gap: 12px; padding: 8px 8px 8px 16px;
    background: var(--surface-2); border: 1px solid var(--border-strong); border-radius: 12px; box-shadow: var(--shadow);
    color: var(--text-dim);
  }
  .ended .btn { height: 30px; font-size: 13px; }
</style>