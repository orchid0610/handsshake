package app

import "handsshake/internal/settings"

// GetSettings works while the vault is locked: settings hold no secrets.
func (a *App) GetSettings() settings.Settings { return a.settings.Get() }

// SaveSettings stores the settings and returns the sanitized values actually kept.
func (a *App) SaveSettings(s settings.Settings) (settings.Settings, error) {
	return a.settings.Set(s)
}