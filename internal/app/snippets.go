package app

import "handsshake/internal/vault"

func (a *App) ListSnippets() ([]vault.Snippet, error) { return a.vault.Snippets() }

// SaveSnippet adds a snippet (empty id) or updates one, and returns its id.
func (a *App) SaveSnippet(id, title, code string) (string, error) {
	return a.vault.PutSnippet(vault.Snippet{ID: id, Title: title, Code: code})
}

func (a *App) DeleteSnippet(id string) error { return a.vault.DeleteSnippet(id) }
