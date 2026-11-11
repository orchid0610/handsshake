export type VaultState = 'none' | 'locked' | 'unlocked'

export interface HostInfo {
  id: string
  name: string
  address: string
  user: string
  auth: 'password' | 'key'
  keyPath: string
  hasPassword: boolean
  keyInVault: boolean
  hasPassphrase: boolean
  os: string
  osLike: string
}

export interface HostInput {
  id: string
  name: string
  address: string
  user: string
  auth: 'password' | 'key'
  password: string
  keyPath: string
  importKey: boolean
  passphrase: string
  clearPassphrase: boolean
}


export interface Settings {
  theme: string
  fontFamily: 'jetbrains-mono' | 'system'
  fontSize: number
  lineHeight: number
  cursorStyle: 'block' | 'bar' | 'underline'
  cursorBlink: boolean
  scrollback: number
  autoLockMinutes: number
}

export interface ImportCandidate {
  alias: string
  address: string
  user: string
  keyPath: string
  keyStatus: 'ok' | 'encrypted' | 'missing' | 'invalid' | 'none'
  status: 'new' | 'exists' | 'unsupported'
  note: string
}

export interface ImportPreview {
  path: string
  candidates: ImportCandidate[]
  warnings: string[]
}