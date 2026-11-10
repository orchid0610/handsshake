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