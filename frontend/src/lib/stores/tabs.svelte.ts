import type { HostInfo } from '../types'

export type TabStatus = 'connecting' | 'connected' | 'ended' | 'error'
export type Tab = { id: string; hostId: string; title: string; status: TabStatus; error: string }

class TabStore {
  items = $state<Tab[]>([])
  activeId = $state('')
  private counter = 0

  open(host: HostInfo) {
    const id = `t${++this.counter}`
    this.items.push({ id, hostId: host.id, title: host.name, status: 'connecting', error: '' })
    this.activeId = id
  }

  close(id: string) {
    const i = this.items.findIndex((t) => t.id === id)
    if (i < 0) return
    this.items.splice(i, 1)
    if (this.activeId === id) {
      this.activeId = (this.items[i] ?? this.items[i - 1])?.id ?? ''
    }
  }

  activate(id: string) {
    this.activeId = id
  }

  setStatus(id: string, status: TabStatus, error = '') {
    const t = this.items.find((t) => t.id === id)
    if (t) {
      t.status = status
      t.error = error
    }
  }
}

export const tabs = new TabStore()