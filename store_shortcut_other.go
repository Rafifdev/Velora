//go:build !windows

package main

func (a *App) ensureStoreDesktopShortcut() {}

func isStorePackage() bool {
	return false
}

func getMicrosoftStoreURL() string {
	return "ms-windows-store://pdp/?ProductId=9PN43QQ19SS9"
}
