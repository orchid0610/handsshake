import * as Go from "../../wailsjs/go/app/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { HostInfo, HostInput, VaultState, Settings, ImportPreview } from "./types";

/** Wails rejects with a plain string; turn anything into readable text. */
export function errText(e: unknown): string {
  if (typeof e === "string") return e;
  if (e instanceof Error) return e.message;
  return String(e);
}

export const api = {
  vaultState: () => Go.VaultState() as Promise<VaultState>,
  createVault: (password: string) => Go.CreateVault(password),
  unlock: (password: string) => Go.Unlock(password),
  recover: (recoveryKey: string, newPassword: string) =>
    Go.RecoverVault(recoveryKey, newPassword),
  lock: () => Go.LockVault(),

  listHosts: () => Go.ListHosts() as Promise<HostInfo[]>,
  saveHost: (h: HostInput) => Go.SaveHost(h as any),
  deleteHost: (id: string) => Go.DeleteHost(id),
  pickKeyFile: () => Go.PickKeyFile(),

  connect: (id: string, hostId: string, cols: number, rows: number) =>
    Go.Connect(id, hostId, cols, rows),
  write: (id: string, data: string) => Go.Write(id, data),
  resize: (id: string, cols: number, rows: number) => Go.Resize(id, cols, rows),
  disconnect: (id: string) => Go.Disconnect(id),
  answerHostKey: (promptId: string, trust: boolean) =>
    Go.AnswerHostKey(promptId, trust),
  getSettings: () => Go.GetSettings() as unknown as Promise<Settings>,
  saveSettings: (s: Settings) =>
    Go.SaveSettings(s as any) as unknown as Promise<Settings>,
  changePassword: (current: string, next: string) =>
    Go.ChangePassword(current, next),
  pickConfigFile: () => Go.PickConfigFile(),
  previewImport: (path: string) =>
    Go.PreviewSSHConfig(path) as unknown as Promise<ImportPreview>,
  importHosts: (path: string, aliases: string[], copyKeys: boolean) =>
    Go.ImportHosts(path, aliases, copyKeys),
};

export const events = { on: EventsOn };
