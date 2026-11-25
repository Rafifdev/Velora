//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32Shortcut          = windows.NewLazySystemDLL("kernel32.dll")
	procGetCurrentPackageFamName = modKernel32Shortcut.NewProc("GetCurrentPackageFamilyName")

	modShell32Shortcut         = windows.NewLazySystemDLL("shell32.dll")
	procSHGetKnownFolderPath   = modShell32Shortcut.NewProc("SHGetKnownFolderPath")
	modOle32Shortcut           = windows.NewLazySystemDLL("ole32.dll")
	procCoTaskMemFreeShortcut  = modOle32Shortcut.NewProc("CoTaskMemFree")
	folderIDDesktop            = windows.GUID{Data1: 0xB4BFCC3A, Data2: 0xDB2C, Data3: 0x424C, Data4: [8]byte{0xB0, 0x29, 0x7F, 0xE9, 0x9A, 0x87, 0xC6, 0x41}}
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

func getMicrosoftStoreURL() string {
	if pfn, ok := getCurrentPackageFamilyName(); ok {
		return "ms-windows-store://pdp/?PFN=" + pfn
	}
	return "ms-windows-store://updates"
}

func getDesktopDir() string {
	var ppszPath uintptr
	ret, _, _ := procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&folderIDDesktop)),
		0,
		0,
		uintptr(unsafe.Pointer(&ppszPath)),
	)
	if ret == 0 && ppszPath != 0 {
		defer procCoTaskMemFreeShortcut.Call(ppszPath)
		path := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ppszPath)))
		if strings.TrimSpace(path) != "" {
			return path
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "Desktop")
	}
	return ""
}

func escapePSLiteral(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func (a *App) ensureStoreDesktopShortcut() {
	if a.store == nil {
		return
	}
	if prefs, err := a.store.GetPreferences(); err == nil {
		if prefs["store.desktopShortcutCreated"] == "true" {
			return
		}
	}
	pfn, ok := getCurrentPackageFamilyName()
	if !ok {
		return
	}
	desktopDir := getDesktopDir()
	if desktopDir == "" {
		return
	}
	lnkPath := filepath.Join(desktopDir, "Velora.lnk")
	if _, err := os.Stat(lnkPath); err == nil {
		_ = a.store.SetPreference("store.desktopShortcutCreated", "true")
		return
	}

	iconPath := ""
	if len(a.trayIcon) > 0 && a.dataDir != "" {
		candidate := filepath.Join(a.dataDir, "velora.ico")
		if err := os.WriteFile(candidate, a.trayIcon, 0o644); err == nil {
			iconPath = candidate
		}
	}

	appTarget := fmt.Sprintf("shell:AppsFolder\\%s!Velora", pfn)
	script := fmt.Sprintf(
		"$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%s'); $s.TargetPath = 'explorer.exe'; $s.Arguments = '%s'; $s.Description = 'Velora Player - Dynamic Island'; ",
		escapePSLiteral(lnkPath),
		escapePSLiteral(appTarget),
	)
	if iconPath != "" {
		script += fmt.Sprintf("$s.IconLocation = '%s,0'; ", escapePSLiteral(iconPath))
	}
	script += "$s.Save()"

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err == nil {
		_ = a.store.SetPreference("store.desktopShortcutCreated", "true")
	}
}
