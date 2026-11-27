//go:build windows

package main

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32Shortcut          = windows.NewLazySystemDLL("kernel32.dll")
	procGetCurrentPackageFamName = modKernel32Shortcut.NewProc("GetCurrentPackageFamilyName")
)

func getCurrentPackageFamilyName() (string, bool) {
	if procGetCurrentPackageFamName.Find() != nil {
		return "", false
	}
	var length uint32
	ret, _, _ := procGetCurrentPackageFamName.Call(uintptr(unsafe.Pointer(&length)), 0)
	// ERROR_INSUFFICIENT_BUFFER = 122
	if ret != 122 || length == 0 {
		return "", false
	}
	buf := make([]uint16, length)
	ret, _, _ = procGetCurrentPackageFamName.Call(uintptr(unsafe.Pointer(&length)), uintptr(unsafe.Pointer(&buf[0])))
	if ret != 0 {
		return "", false
	}
	pfn := strings.TrimSpace(windows.UTF16ToString(buf))
	return pfn, pfn != ""
}

func isStorePackage() bool {
	_, ok := getCurrentPackageFamilyName()
	return ok
}

func getMicrosoftStoreURL() string {
	return "ms-windows-store://pdp/?ProductId=9PN43QQ19SS9"
}

// Desktop shortcut lifecycle for MSIX builds is declared via desktop7:Shortcut
// in AppxManifest.xml so Windows creates and removes Velora.lnk automatically.
func (a *App) ensureStoreDesktopShortcut() {}
