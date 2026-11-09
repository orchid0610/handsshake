package app

// VaultState tells the UI which screen to show: "none" (first run: create a
// vault), "locked" (ask for the password) or "unlocked".
func (a *App) VaultState() string {
	switch {
	case !a.vault.Exists():
		return "none"
	case a.vault.IsUnlocked():
		return "unlocked"
	default:
		return "locked"
	}
}

// CreateVault makes a new vault and returns the recovery key, which the UI
// must show exactly once. (The vault is already unlocked afterwards; it is
// the UI's job to hold the user on the "save your recovery key" screen.)
func (a *App) CreateVault(password string) (string, error) {
	return a.vault.Create(password)
}

func (a *App) Unlock(password string) error { return a.vault.Unlock(password) }

func (a *App) RecoverVault(recoveryKey, newPassword string) error {
	return a.vault.UnlockWithRecovery(recoveryKey, newPassword)
}

func (a *App) ChangePassword(current, next string) error {
	return a.vault.ChangePassword(current, next)
}

// LockVault forgets all secrets. Open terminals keep running: they have
// already authenticated and no longer need the credentials.
func (a *App) LockVault() { a.vault.Lock() }