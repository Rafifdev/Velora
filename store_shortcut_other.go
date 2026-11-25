//go:build !windows

package main

func (a *App) ensureStoreDesktopShortcut() {}

func getMicrosoftStoreURL() string {
	return ""
}
