/** Split a stored "host:port" address ([::1]:22 style for IPv6). */
export function splitAddress(addr: string): { host: string; port: string } {
  const a = addr.trim()
  if (a.startsWith('[')) {
    const end = a.indexOf(']')
    if (end > 0) return { host: a.slice(1, end), port: a.slice(end + 2) || '22' }
  }
  const i = a.lastIndexOf(':')
  if (i > 0 && a.indexOf(':') === i) return { host: a.slice(0, i), port: a.slice(i + 1) || '22' }
  return { host: a, port: '22' }
}

export function joinAddress(host: string, port: string): string {
  const h = host.trim()
  const p = port.trim() || '22'
  return h.includes(':') ? `[${h}]:${p}` : `${h}:${p}`
}