//go:build windows && installer

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

//go:embed assets/Velora.exe
var veloraExeBytes []byte

//go:embed assets/icon.ico
var veloraIconBytes []byte

const (
	appName      = "Velora"
	appVersion   = "1.1.0"
	appPublisher = "Yoapipp"
	appURL       = "https://github.com/Rafifdev/Velora"
	uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Velora`
	startupKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procGetModuleHandleW        = kernel32.NewProc("GetModuleHandleW")
	procInitCommonControlsEx    = comctl32.NewProc("InitCommonControlsEx")
	procRegisterClassExW        = user32.NewProc("RegisterClassExW")
	procCreateWindowExW         = user32.NewProc("CreateWindowExW")
	procDefWindowProcW          = user32.NewProc("DefWindowProcW")
	procShowWindow              = user32.NewProc("ShowWindow")
	procUpdateWindow            = user32.NewProc("UpdateWindow")
	procGetMessageW             = user32.NewProc("GetMessageW")
	procTranslateMessage        = user32.NewProc("TranslateMessage")
	procDispatchMessageW        = user32.NewProc("DispatchMessageW")
	procPostQuitMessage         = user32.NewProc("PostQuitMessage")
	procSendMessageW            = user32.NewProc("SendMessageW")
	procSetWindowTextW          = user32.NewProc("SetWindowTextW")
	procGetWindowTextW          = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW    = user32.NewProc("GetWindowTextLengthW")
	procEnableWindow            = user32.NewProc("EnableWindow")
	procInvalidateRect          = user32.NewProc("InvalidateRect")
	procBeginPaint              = user32.NewProc("BeginPaint")
	procEndPaint                = user32.NewProc("EndPaint")
	procFillRect                = user32.NewProc("FillRect")
	procGetSystemMetrics        = user32.NewProc("GetSystemMetrics")
	procGetDpiForSystem         = user32.NewProc("GetDpiForSystem")
	procLoadCursorW             = user32.NewProc("LoadCursorW")
	procLoadImageW              = user32.NewProc("LoadImageW")
	procDrawIconEx              = user32.NewProc("DrawIconEx")
	procMessageBoxW             = user32.NewProc("MessageBoxW")
	procDestroyWindow           = user32.NewProc("DestroyWindow")
	procCreateFontIndirectW     = gdi32.NewProc("CreateFontIndirectW")
	procCreateSolidBrush        = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject            = gdi32.NewProc("DeleteObject")
	procSetBkMode               = gdi32.NewProc("SetBkMode")
	procSetBkColor              = gdi32.NewProc("SetBkColor")
	procSetTextColor            = gdi32.NewProc("SetTextColor")
	procGetStockObject          = gdi32.NewProc("GetStockObject")
	procSHBrowseForFolderW      = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW    = shell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree           = ole32.NewProc("CoTaskMemFree")
)

const (
	wmCreate       = 0x0001
	wmDestroy      = 0x0002
	wmPaint        = 0x000F
	wmClose        = 0x0010
	wmSetFont      = 0x0030
	wmCommand      = 0x0111
	wmCtlColorStc  = 0x0138
	wmAppStep      = 0x8001
	wmAppDone      = 0x8002
	wmAppErr       = 0x8003

	wsOverlapped  = 0x00000000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsMinBox      = 0x00020000
	wsVisible     = 0x10000000
	wsChild       = 0x40000000
	wsTabStop     = 0x00010000
	wsBorder      = 0x00800000
	esAutoHScroll = 0x0080

	bsPushButton      = 0x00000000
	bsDefPushButton   = 0x00000001
	bsAutoCheckBox    = 0x00000003
	bsAutoRadioButton = 0x00000009

	bmGetCheck = 0x00F0
	bmSetCheck = 0x00F1
	bstChecked = 1

	pbmSetRange32 = 0x0406
	pbmSetPos     = 0x0402

	swShow = 5
	swHide = 0

	mbOK              = 0x00000000
	mbYesNo           = 0x00000004
	mbIconError       = 0x00000010
	mbIconQuestion    = 0x00000020
	mbIconWarning     = 0x00000030
	mbIconInformation = 0x00000040
	idYes             = 6

	idBtnBack      = 101
	idBtnNext      = 102
	idBtnCancel    = 103
	idEditDir      = 104
	idBtnBrowse    = 105
	idChkDesktop   = 106
	idChkStartMnu  = 107
	idChkStartup   = 108
	idChkLaunch     = 109
	idRadRepair     = 110
	idRadUninstall  = 111
	idChkRemoveData = 112
)

type initCommonControlsExStruct struct {
	dwSize uint32
	dwICC  uint32
}

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type rect struct {
	left, top, right, bottom int32
}

type point struct {
	x, y int32
}

type msgStruct struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type logFontW struct {
	lfHeight         int32
	lfWidth          int32
	lfEscapement     int32
	lfOrientation    int32
	lfWeight         int32
	lfItalic         byte
	lfUnderline      byte
	lfStrikeOut      byte
	lfCharSet        byte
	lfOutPrecision   byte
	lfClipPrecision  byte
	lfQuality        byte
	lfPitchAndFamily byte
	lfFaceName       [32]uint16
}

type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

type wizardState struct {
	hwnd            uintptr
	dpi             int32
	page            int // 0=Welcome/Maintenance, 1=Location, 2=Tasks, 3=Installing/Finished
	installing      bool
	finished        bool
	isMaintenance   bool
	isUpdate        bool
	isUninstallMode bool
	uninstalling    bool
	removeUserData  bool
	existingVersion string
	installDir      string
	lastErr         string
	hIconBig        uintptr
	hFontTitle      uintptr
	hFontHdr        uintptr
	hFontBold       uintptr
	hFontBody       uintptr
	hBrushWhite     uintptr
	hBrushFooter    uintptr
	hBrushLine      uintptr

	lblHdrTitle uintptr
	lblHdrSub   uintptr

	lblWelTitle      uintptr
	lblWelBody       uintptr
	radRepair        uintptr
	lblRepairDesc    uintptr
	radUninstall     uintptr
	lblUninstallDesc uintptr
	chkRemoveData    uintptr

	lblDirIntro uintptr
	editDir     uintptr
	btnBrowse   uintptr
	lblDirSpace uintptr

	lblTaskIntro uintptr
	lblTaskGroup uintptr
	chkDesktop   uintptr
	chkStartMenu uintptr
	lblStartGrp  uintptr
	chkStartup   uintptr

	lblProgStatus uintptr
	progBar       uintptr
	lblDoneBody   uintptr
	chkLaunch     uintptr

	btnBack   uintptr
	btnNext   uintptr
	btnCancel uintptr
}

var wiz wizardState

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func scale(px int32) int32 {
	if wiz.dpi <= 0 {
		return px
	}
	return (px * wiz.dpi) / 96
}

func rgb(r, g, b byte) uintptr {
	return uintptr(uint32(r) | (uint32(g) << 8) | (uint32(b) << 16))
}

func messageBox(hwnd uintptr, title, text string, flags uint32) int {
	ret, _, _ := procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(title))), uintptr(flags))
	return int(ret)
}

func makeFont(sizePt int32, weight int32) uintptr {
	var lf logFontW
	dpi := wiz.dpi
	if dpi <= 0 {
		dpi = 96
	}
	// LOGFONT lfHeight = -MulDiv(pt, DPI, 72)
	lf.lfHeight = -(sizePt * dpi) / 72
	lf.lfWeight = weight
	lf.lfCharSet = 1
	lf.lfQuality = 5 // CLEARTYPE_QUALITY
	face, _ := syscall.UTF16FromString("Segoe UI")
	copy(lf.lfFaceName[:], face)
	h, _, _ := procCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&lf)))
	return h
}

func createCtrl(exStyle uint32, className, text string, style uint32, x, y, w, h int32, parent uintptr, id uintptr, font uintptr) uintptr {
	hwnd, _, _ := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(u16(className))),
		uintptr(unsafe.Pointer(u16(text))),
		uintptr(style),
		uintptr(scale(x)),
		uintptr(scale(y)),
		uintptr(scale(w)),
		uintptr(scale(h)),
		parent,
		id,
		0,
		0,
	)
	if hwnd != 0 && font != 0 {
		procSendMessageW.Call(hwnd, wmSetFont, font, 1)
	}
	return hwnd
}

func showCtrl(hwnd uintptr, visible bool) {
	if visible {
		procShowWindow.Call(hwnd, swShow)
	} else {
		procShowWindow.Call(hwnd, swHide)
	}
}

func setText(hwnd uintptr, text string) {
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(u16(text))))
}

func getText(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	buf := make([]uint16, length+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), length+1)
	return syscall.UTF16ToString(buf)
}

func isChecked(hwnd uintptr) bool {
	r, _, _ := procSendMessageW.Call(hwnd, bmGetCheck, 0, 0)
	return r == bstChecked
}

func setChecked(hwnd uintptr, checked bool) {
	v := uintptr(0)
	if checked {
		v = bstChecked
	}
	procSendMessageW.Call(hwnd, bmSetCheck, v, 0)
}

func defaultInstallDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		if home, err := os.UserHomeDir(); err == nil {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
	}
	return filepath.Join(localAppData, "Programs", appName)
}

func detectExistingInstall() (installed bool, dir string, ver string) {
	dir = defaultInstallDir()
	if k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE); err == nil {
		if loc, _, locErr := k.GetStringValue("InstallLocation"); locErr == nil && strings.TrimSpace(loc) != "" {
			dir = strings.TrimSpace(loc)
		}
		if v, _, vErr := k.GetStringValue("DisplayVersion"); vErr == nil {
			ver = strings.TrimSpace(v)
		}
		_ = k.Close()
	}
	if st, err := os.Stat(filepath.Join(dir, "Velora.exe")); err == nil && !st.IsDir() {
		return true, dir, ver
	}
	return false, defaultInstallDir(), ""
}

func shortcutPaths() (desktopLnk, startMenuLnk string) {
	home, _ := os.UserHomeDir()
	desktopDir := filepath.Join(home, "Desktop")
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	startMenuDir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs")
	return filepath.Join(desktopDir, appName+".lnk"), filepath.Join(startMenuDir, appName+".lnk")
}

func runHidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func createShortcut(lnkPath, targetPath, iconPath, workDir string) error {
	script := fmt.Sprintf(
		"$ws = New-Object -ComObject WScript.Shell; $s = $ws.CreateShortcut('%s'); $s.TargetPath = '%s'; $s.WorkingDirectory = '%s'; $s.IconLocation = '%s,0'; $s.Save()",
		escapePS(lnkPath), escapePS(targetPath), escapePS(workDir), escapePS(iconPath),
	)
	return runHidden("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
}

func stopRunningApp() {
	_ = runHidden("taskkill", "/F", "/IM", "Velora.exe")
}

func browseForFolder(hwnd uintptr) string {
	var display [260]uint16
	var bi browseInfoW
	bi.hwndOwner = hwnd
	bi.pszDisplayName = &display[0]
	bi.lpszTitle = u16("Select the destination folder where Velora should be installed:")
	bi.ulFlags = 0x00000001 | 0x00000040 // BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)
	var pathBuf [260]uint16
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))
	if ok == 0 {
		return ""
	}
	selected := syscall.UTF16ToString(pathBuf[:])
	if selected == "" {
		return ""
	}
	if !strings.EqualFold(filepath.Base(selected), appName) {
		selected = filepath.Join(selected, appName)
	}
	return selected
}

func syncWizardPage() {
	isP0 := wiz.page == 0
	isP1 := wiz.page == 1
	isP2 := wiz.page == 2
	isP3 := wiz.page == 3

	showCtrl(wiz.lblWelTitle, isP0)
	showCtrl(wiz.lblWelBody, isP0)
	showCtrl(wiz.radRepair, isP0 && wiz.isMaintenance && !wiz.isUninstallMode)
	showCtrl(wiz.lblRepairDesc, isP0 && wiz.isMaintenance && !wiz.isUninstallMode)
	showCtrl(wiz.radUninstall, isP0 && wiz.isMaintenance && !wiz.isUninstallMode)
	showCtrl(wiz.lblUninstallDesc, isP0 && wiz.isMaintenance && !wiz.isUninstallMode)
	showCtrl(wiz.chkRemoveData, isP0 && (wiz.isUninstallMode || wiz.isMaintenance))

	showCtrl(wiz.lblDirIntro, isP1)
	showCtrl(wiz.editDir, isP1)
	showCtrl(wiz.btnBrowse, isP1)
	showCtrl(wiz.lblDirSpace, isP1)

	showCtrl(wiz.lblTaskIntro, isP2)
	showCtrl(wiz.lblTaskGroup, isP2)
	showCtrl(wiz.chkDesktop, isP2)
	showCtrl(wiz.chkStartMenu, isP2)
	showCtrl(wiz.lblStartGrp, isP2)
	showCtrl(wiz.chkStartup, isP2)

	showCtrl(wiz.lblProgStatus, isP3)
	showCtrl(wiz.progBar, isP3 && !wiz.finished)
	showCtrl(wiz.lblDoneBody, isP3 && wiz.finished)
	showCtrl(wiz.chkLaunch, isP3 && wiz.finished && !wiz.uninstalling)
	showCtrl(wiz.btnBack, !wiz.isUninstallMode)

	switch wiz.page {
	case 0:
		if wiz.isUninstallMode {
			setText(wiz.lblHdrTitle, "Uninstall Velora")
			setText(wiz.lblHdrSub, "Remove Velora from your computer.")
			procEnableWindow.Call(wiz.btnBack, 0)
			setText(wiz.btnNext, "Uninstall")
		} else if wiz.isMaintenance {
			setText(wiz.lblHdrTitle, "Program Maintenance")
			setText(wiz.lblHdrSub, "Repair or remove Velora from your computer.")
			procEnableWindow.Call(wiz.btnBack, 0)
			if isChecked(wiz.radUninstall) {
				setText(wiz.btnNext, "Uninstall")
				procEnableWindow.Call(wiz.chkRemoveData, 1)
			} else {
				setText(wiz.btnNext, "Next >")
				procEnableWindow.Call(wiz.chkRemoveData, 0)
			}
		} else if wiz.isUpdate {
			setText(wiz.lblHdrTitle, "Update Velora")
			setText(wiz.lblHdrSub, fmt.Sprintf("Upgrade Velora from v%s to v%s", wiz.existingVersion, appVersion))
			procEnableWindow.Call(wiz.btnBack, 0)
			setText(wiz.btnNext, "Next >")
		} else {
			setText(wiz.lblHdrTitle, "Welcome to the Velora Setup Wizard")
			setText(wiz.lblHdrSub, fmt.Sprintf("Version %s — Desktop Music Player & Dynamic Island", appVersion))
			procEnableWindow.Call(wiz.btnBack, 0)
			setText(wiz.btnNext, "Next >")
		}
	case 1:
		setText(wiz.lblHdrTitle, "Select Destination Location")
		setText(wiz.lblHdrSub, "Where should Velora be installed?")
		procEnableWindow.Call(wiz.btnBack, 1)
		setText(wiz.btnNext, "Next >")
	case 2:
		setText(wiz.lblHdrTitle, "Select Additional Tasks")
		setText(wiz.lblHdrSub, "Which additional tasks should be performed?")
		procEnableWindow.Call(wiz.btnBack, 1)
		if wiz.isUpdate {
			setText(wiz.btnNext, "Update")
		} else if wiz.isMaintenance {
			setText(wiz.btnNext, "Reinstall")
		} else {
			setText(wiz.btnNext, "Install")
		}
	case 3:
		if !wiz.finished {
			if wiz.uninstalling {
				setText(wiz.lblHdrTitle, "Uninstalling")
				setText(wiz.lblHdrSub, "Please wait while Setup removes Velora and its data from your computer.")
			} else {
				setText(wiz.lblHdrTitle, "Installing")
				setText(wiz.lblHdrSub, "Please wait while Setup installs Velora on your computer.")
			}
			procEnableWindow.Call(wiz.btnBack, 0)
			procEnableWindow.Call(wiz.btnNext, 0)
			procEnableWindow.Call(wiz.btnCancel, 0)
		} else {
			if wiz.uninstalling {
				setText(wiz.lblHdrTitle, "Uninstallation Complete")
				setText(wiz.lblHdrSub, "Velora has been completely removed from your computer.")
				setText(wiz.lblDoneBody, "Setup has finished removing Velora and all of its application data, cache, downloads, and settings from your computer.\n\nClick Finish to exit Setup.")
			} else {
				setText(wiz.lblHdrTitle, "Completing the Velora Setup Wizard")
				setText(wiz.lblHdrSub, "Setup has finished installing Velora on your computer.")
				setText(wiz.lblDoneBody, "Setup has finished installing Velora on your computer. The application may be launched by selecting the installed shortcuts.\n\nClick Finish to exit Setup.")
			}
			procEnableWindow.Call(wiz.btnBack, 0)
			procEnableWindow.Call(wiz.btnNext, 1)
			setText(wiz.btnNext, "Finish")
			procEnableWindow.Call(wiz.btnCancel, 0)
		}
	}
	procInvalidateRect.Call(wiz.hwnd, 0, 1)
}

func performUninstallAsync(removeUserData bool) {
	sendStep := func(pct int, status string) {
		setText(wiz.lblProgStatus, status)
		procSendMessageW.Call(wiz.progBar, pbmSetPos, uintptr(pct), 0)
	}

	sendStep(15, "Stopping running application instances...")
	stopRunningApp()
	time.Sleep(180 * time.Millisecond)

	sendStep(40, "Removing shortcuts and startup entries...")
	desktopLnk, startMenuLnk := shortcutPaths()
	_ = os.Remove(desktopLnk)
	_ = os.Remove(startMenuLnk)
	if kRun, err := registry.OpenKey(registry.CURRENT_USER, startupKey, registry.SET_VALUE); err == nil {
		_ = kRun.DeleteValue(appName)
		_ = kRun.Close()
	}
	time.Sleep(160 * time.Millisecond)

	if removeUserData {
		sendStep(70, "Removing application data, cache, and downloads...")
		if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
			_ = os.RemoveAll(filepath.Join(cfgDir, appName))
			_ = os.RemoveAll(filepath.Join(cfgDir, appName+".exe.WebView2"))
		}
		if appData := os.Getenv("APPDATA"); appData != "" {
			_ = os.RemoveAll(filepath.Join(appData, appName))
			_ = os.RemoveAll(filepath.Join(appData, appName+".exe.WebView2"))
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			_ = os.RemoveAll(filepath.Join(localAppData, appName))
			_ = os.RemoveAll(filepath.Join(localAppData, appName+".exe.WebView2"))
		}
		time.Sleep(160 * time.Millisecond)
	}

	sendStep(90, "Removing application files and registry keys...")
	_ = registry.DeleteKey(registry.CURRENT_USER, uninstallKey)
	_ = os.Remove(filepath.Join(wiz.installDir, "Velora.exe"))
	_ = os.Remove(filepath.Join(wiz.installDir, "icon.ico"))
	cmd := exec.Command("cmd.exe", "/C", fmt.Sprintf(`ping 127.0.0.1 -n 2 >nul & rmdir /S /Q "%s"`, wiz.installDir))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
	time.Sleep(160 * time.Millisecond)

	sendStep(100, "Uninstallation complete.")
	procSendMessageW.Call(wiz.hwnd, wmAppDone, 0, 0)
}

func performInstallAsync(dir string, createDesktop, createStartMenu, runOnStartup bool) {
	sendStep := func(pct int, status string) {
		setText(wiz.lblProgStatus, status)
		procSendMessageW.Call(wiz.progBar, pbmSetPos, uintptr(pct), 0)
	}

	sendStep(15, "Stopping running application instances...")
	stopRunningApp()

	sendStep(30, "Creating installation directory...")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		wiz.lastErr = "Unable to create installation directory:\n" + err.Error()
		procSendMessageW.Call(wiz.hwnd, wmAppErr, 0, 0)
		return
	}

	targetExe := filepath.Join(dir, "Velora.exe")
	targetIcon := filepath.Join(dir, "icon.ico")
	targetUninst := filepath.Join(dir, "uninstall.exe")

	sendStep(55, "Extracting application files...")
	if err := os.WriteFile(targetExe, veloraExeBytes, 0o755); err != nil {
		wiz.lastErr = "Unable to write Velora.exe. Please make sure Velora is closed and try again.\n\nDetails: " + err.Error()
		procSendMessageW.Call(wiz.hwnd, wmAppErr, 0, 0)
		return
	}
	_ = os.WriteFile(targetIcon, veloraIconBytes, 0o644)

	if selfExe, err := os.Executable(); err == nil {
		if selfBytes, readErr := os.ReadFile(selfExe); readErr == nil {
			_ = os.WriteFile(targetUninst, selfBytes, 0o755)
		}
	}

	sendStep(78, "Creating shortcuts...")
	desktopLnk, startMenuLnk := shortcutPaths()
	if createDesktop {
		_ = createShortcut(desktopLnk, targetExe, targetIcon, dir)
	} else {
		_ = os.Remove(desktopLnk)
	}
	if createStartMenu {
		_ = createShortcut(startMenuLnk, targetExe, targetIcon, dir)
	}

	sendStep(92, "Registering application in Windows...")
	if k, _, regErr := registry.CreateKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE); regErr == nil {
		_ = k.SetStringValue("DisplayName", appName)
		_ = k.SetStringValue("DisplayVersion", appVersion)
		_ = k.SetStringValue("Publisher", appPublisher)
		_ = k.SetStringValue("URLInfoAbout", appURL)
		_ = k.SetStringValue("InstallDate", time.Now().Format("20060102"))
		_ = k.SetStringValue("InstallLocation", dir)
		_ = k.SetStringValue("DisplayIcon", targetIcon)
		_ = k.SetStringValue("UninstallString", fmt.Sprintf(`"%s" --uninstall`, targetUninst))
		_ = k.SetDWordValue("NoModify", 1)
		_ = k.SetDWordValue("NoRepair", 1)
		_ = k.SetDWordValue("EstimatedSize", uint32(len(veloraExeBytes)/1024))
		_ = k.Close()
	}

	if kRun, _, runErr := registry.CreateKey(registry.CURRENT_USER, startupKey, registry.SET_VALUE); runErr == nil {
		if runOnStartup {
			_ = kRun.SetStringValue(appName, fmt.Sprintf(`"%s"`, targetExe))
		} else {
			_ = kRun.DeleteValue(appName)
		}
		_ = kRun.Close()
	}

	sendStep(100, "Installation complete.")
	procSendMessageW.Call(wiz.hwnd, wmAppDone, 0, 0)
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCreate:
		wiz.hwnd = hwnd
		wiz.lblHdrTitle = createCtrl(0, "STATIC", "", wsChild|wsVisible, 24, 14, 440, 26, hwnd, 0, wiz.hFontHdr)
		wiz.lblHdrSub = createCtrl(0, "STATIC", "", wsChild|wsVisible, 24, 41, 440, 22, hwnd, 0, wiz.hFontBody)

		welTitle := "Welcome to the Velora Setup Wizard"
		welBody := fmt.Sprintf(
			"This wizard will guide you through the installation of %s %s on your computer.\n\n"+
				"Publisher: %s\n\n"+
				"It is recommended that you close any running instance of %s before continuing.\n\n"+
				"Click Next to continue, or Cancel to exit Setup.",
			appName, appVersion, appPublisher, appName,
		)
		welBodyH := int32(175)
		chkDataY := int32(288)
		if wiz.isUninstallMode {
			welTitle = "Welcome to the Velora Uninstall Wizard"
			welBody = fmt.Sprintf(
				"This wizard will remove %s %s from your computer.\n\n"+
					"Installed location:\n%s\n\n"+
					"Click Uninstall to proceed with the removal, or Cancel to exit Setup.",
				appName, appVersion, wiz.installDir,
			)
			welBodyH = 128
			chkDataY = 266
		} else if wiz.isMaintenance {
			welTitle = "Modify, Repair, or Remove Installation"
			welBody = fmt.Sprintf(
				"Setup has detected that %s %s is already installed on your computer.\n\nSelect the operation you would like to perform:",
				appName, appVersion,
			)
			welBodyH = 48
			chkDataY = 294
		} else if wiz.isUpdate {
			welTitle = fmt.Sprintf("Update %s to Version %s", appName, appVersion)
			welBody = fmt.Sprintf(
				"Setup has detected %s %s installed at:\n%s\n\n"+
					"This wizard will update %s to version %s. All of your playlists, downloads, and settings will be preserved.\n\n"+
					"Click Next to continue, or Cancel to exit Setup.",
				appName, wiz.existingVersion, wiz.installDir, appName, appVersion,
			)
		}
		wiz.lblWelTitle = createCtrl(0, "STATIC", welTitle, wsChild, 26, 90, 495, 32, hwnd, 0, wiz.hFontTitle)
		wiz.lblWelBody = createCtrl(0, "STATIC", welBody, wsChild, 26, 128, 495, welBodyH, hwnd, 0, wiz.hFontBody)
		wiz.radRepair = createCtrl(0, "BUTTON", "&Repair / Reinstall Velora", wsChild|wsTabStop|bsAutoRadioButton, 32, 182, 460, 24, hwnd, idRadRepair, wiz.hFontBold)
		wiz.lblRepairDesc = createCtrl(0, "STATIC", "Reinstall application files and shortcuts while keeping all of your playlists, downloads, and settings intact.", wsChild, 52, 206, 460, 34, hwnd, 0, wiz.hFontBody)
		wiz.radUninstall = createCtrl(0, "BUTTON", "&Remove (Uninstall) Velora", wsChild|wsTabStop|bsAutoRadioButton, 32, 244, 460, 24, hwnd, idRadUninstall, wiz.hFontBold)
		wiz.lblUninstallDesc = createCtrl(0, "STATIC", "Completely remove Velora from your computer.", wsChild, 52, 268, 460, 22, hwnd, 0, wiz.hFontBody)
		wiz.chkRemoveData = createCtrl(0, "BUTTON", "Also remove all &user data, playlists, downloads, cache, and settings", wsChild|wsTabStop|bsAutoCheckBox, 52, chkDataY, 465, 24, hwnd, idChkRemoveData, wiz.hFontBody)
		setChecked(wiz.radRepair, true)
		setChecked(wiz.radUninstall, false)
		setChecked(wiz.chkRemoveData, true)

		wiz.lblDirIntro = createCtrl(
			0, "STATIC",
			"Setup will install Velora into the following folder.\n\nTo continue, click Next. If you would like to select a different folder, click Browse.",
			wsChild, 26, 92, 495, 64, hwnd, 0, wiz.hFontBody,
		)
		wiz.editDir = createCtrl(0, "EDIT", wiz.installDir, wsChild|wsBorder|wsTabStop|esAutoHScroll, 26, 166, 392, 28, hwnd, idEditDir, wiz.hFontBody)
		wiz.btnBrowse = createCtrl(0, "BUTTON", "Browse...", wsChild|wsTabStop|bsPushButton, 428, 165, 94, 30, hwnd, idBtnBrowse, wiz.hFontBody)
		spaceMB := float64(len(veloraExeBytes)) / (1024.0 * 1024.0)
		wiz.lblDirSpace = createCtrl(
			0, "STATIC",
			fmt.Sprintf("At least %.1f MB of free disk space is required.", spaceMB+2.0),
			wsChild, 26, 282, 495, 24, hwnd, 0, wiz.hFontBody,
		)

		wiz.lblTaskIntro = createCtrl(
			0, "STATIC",
			"Select the additional tasks you would like Setup to perform while installing Velora, then click Install.",
			wsChild, 26, 92, 495, 44, hwnd, 0, wiz.hFontBody,
		)
		wiz.lblTaskGroup = createCtrl(0, "STATIC", "Additional shortcuts:", wsChild, 26, 142, 495, 22, hwnd, 0, wiz.hFontBold)
		wiz.chkDesktop = createCtrl(0, "BUTTON", "Create a &desktop shortcut", wsChild|wsTabStop|bsAutoCheckBox, 34, 168, 460, 26, hwnd, idChkDesktop, wiz.hFontBody)
		wiz.chkStartMenu = createCtrl(0, "BUTTON", "Create a &Start Menu shortcut", wsChild|wsTabStop|bsAutoCheckBox, 34, 198, 460, 26, hwnd, idChkStartMnu, wiz.hFontBody)
		wiz.lblStartGrp = createCtrl(0, "STATIC", "System startup:", wsChild, 26, 236, 495, 22, hwnd, 0, wiz.hFontBold)
		wiz.chkStartup = createCtrl(0, "BUTTON", "Launch Velora automatically when &Windows starts", wsChild|wsTabStop|bsAutoCheckBox, 34, 262, 460, 26, hwnd, idChkStartup, wiz.hFontBody)
		setChecked(wiz.chkDesktop, true)
		setChecked(wiz.chkStartMenu, true)
		setChecked(wiz.chkStartup, false)

		wiz.lblProgStatus = createCtrl(0, "STATIC", "Preparing installation...", wsChild, 26, 94, 495, 24, hwnd, 0, wiz.hFontBold)
		wiz.progBar = createCtrl(0, "msctls_progress32", "", wsChild, 26, 126, 495, 22, hwnd, 0, 0)
		procSendMessageW.Call(wiz.progBar, pbmSetRange32, 0, 100)
		wiz.lblDoneBody = createCtrl(
			0, "STATIC",
			"Setup has finished installing Velora on your computer. The application may be launched by selecting the installed shortcuts.\n\nClick Finish to exit Setup.",
			wsChild, 26, 130, 495, 68, hwnd, 0, wiz.hFontBody,
		)
		wiz.chkLaunch = createCtrl(0, "BUTTON", "&Launch Velora now", wsChild|wsTabStop|bsAutoCheckBox, 30, 210, 460, 26, hwnd, idChkLaunch, wiz.hFontBody)
		setChecked(wiz.chkLaunch, true)

		wiz.btnBack = createCtrl(0, "BUTTON", "< &Back", wsChild|wsVisible|wsTabStop|bsPushButton, 238, 346, 90, 30, hwnd, idBtnBack, wiz.hFontBody)
		wiz.btnNext = createCtrl(0, "BUTTON", "&Next >", wsChild|wsVisible|wsTabStop|bsDefPushButton, 334, 346, 90, 30, hwnd, idBtnNext, wiz.hFontBody)
		wiz.btnCancel = createCtrl(0, "BUTTON", "Cancel", wsChild|wsVisible|wsTabStop|bsPushButton, 434, 346, 90, 30, hwnd, idBtnCancel, wiz.hFontBody)

		syncWizardPage()
		return 0

	case wmCtlColorStc:
		hdc := wParam
		procSetBkColor.Call(hdc, rgb(255, 255, 255))
		procSetTextColor.Call(hdc, rgb(24, 28, 34))
		return wiz.hBrushWhite

	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			rcMain := rect{left: 0, top: 0, right: scale(560), bottom: scale(330)}
			procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rcMain)), wiz.hBrushWhite)

			rcHdrLine := rect{left: 0, top: scale(72), right: scale(560), bottom: scale(73)}
			procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rcHdrLine)), wiz.hBrushLine)

			if wiz.hIconBig != 0 {
				procDrawIconEx.Call(hdc, uintptr(scale(486)), uintptr(scale(16)), wiz.hIconBig, uintptr(scale(40)), uintptr(scale(40)), 0, 0, 0x0003)
			}

			rcFootLine := rect{left: 0, top: scale(330), right: scale(560), bottom: scale(331)}
			procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rcFootLine)), wiz.hBrushLine)
			rcFoot := rect{left: 0, top: scale(331), right: scale(560), bottom: scale(400)}
			procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rcFoot)), wiz.hBrushFooter)
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case wmCommand:
		id := int(wParam & 0xFFFF)
		switch id {
		case idRadRepair, idRadUninstall:
			syncWizardPage()
		case idBtnBrowse:
			if chosen := browseForFolder(hwnd); chosen != "" {
				wiz.installDir = chosen
				setText(wiz.editDir, chosen)
			}
		case idBtnBack:
			if wiz.page == 2 && (wiz.isMaintenance || wiz.isUpdate) {
				wiz.page = 0
				syncWizardPage()
			} else if wiz.page > 0 && wiz.page < 3 {
				wiz.page--
				syncWizardPage()
			}
		case idBtnNext:
			switch wiz.page {
			case 0:
				if wiz.isUninstallMode || (wiz.isMaintenance && isChecked(wiz.radUninstall)) {
					removeData := isChecked(wiz.chkRemoveData)
					confirmMsg := "Are you sure you want to remove Velora from your computer?"
					if removeData {
						confirmMsg = "Are you sure you want to completely remove Velora and all of its user data, playlists, downloads, cache, and settings?"
					}
					if messageBox(hwnd, "Velora Uninstall", confirmMsg, mbYesNo|mbIconWarning) != idYes {
						return 0
					}
					wiz.uninstalling = true
					wiz.removeUserData = removeData
					wiz.page = 3
					wiz.installing = true
					wiz.finished = false
					syncWizardPage()
					go performUninstallAsync(removeData)
				} else if wiz.isMaintenance || wiz.isUpdate {
					wiz.uninstalling = false
					wiz.page = 2
					syncWizardPage()
				} else {
					wiz.page = 1
					syncWizardPage()
				}
			case 1:
				dir := strings.TrimSpace(getText(wiz.editDir))
				if dir == "" {
					messageBox(hwnd, "Velora Setup", "Please specify a valid destination folder.", mbOK|mbIconWarning)
					return 0
				}
				wiz.installDir = dir
				wiz.page = 2
				syncWizardPage()
			case 2:
				createDesktop := isChecked(wiz.chkDesktop)
				createStart := isChecked(wiz.chkStartMenu)
				runStartup := isChecked(wiz.chkStartup)
				wiz.page = 3
				wiz.installing = true
				wiz.finished = false
				syncWizardPage()
				go performInstallAsync(wiz.installDir, createDesktop, createStart, runStartup)
			case 3:
				if wiz.finished {
					if !wiz.uninstalling && isChecked(wiz.chkLaunch) {
						targetExe := filepath.Join(wiz.installDir, "Velora.exe")
						_ = exec.Command(targetExe).Start()
					}
					procDestroyWindow.Call(hwnd)
				}
			}
		case idBtnCancel:
			if !wiz.installing {
				if messageBox(hwnd, "Exit Setup", "Setup is not complete. Are you sure you want to exit Setup?", mbYesNo|mbIconQuestion) == idYes {
					procDestroyWindow.Call(hwnd)
				}
			}
		}
		return 0

	case wmAppDone:
		wiz.installing = false
		wiz.finished = true
		if wiz.uninstalling {
			setText(wiz.lblProgStatus, "Velora has been completely removed from your computer.")
		} else {
			setText(wiz.lblProgStatus, "Velora v"+appVersion+" has been successfully installed.")
		}
		syncWizardPage()
		return 0

	case wmAppErr:
		wiz.installing = false
		messageBox(hwnd, "Velora Setup Error", wiz.lastErr, mbOK|mbIconError)
		wiz.page = 2
		syncWizardPage()
		return 0

	case wmClose:
		if wiz.installing {
			return 0
		}
		if !wiz.finished {
			if messageBox(hwnd, "Exit Setup", "Setup is not complete. If you exit now, Velora will not be installed.\n\nExit Setup?", mbYesNo|mbIconQuestion) != idYes {
				return 0
			}
		}
		procDestroyWindow.Call(hwnd)
		return 0

	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func runSilentInstall() {
	dir := defaultInstallDir()
	stopRunningApp()
	_ = os.MkdirAll(dir, 0o755)
	targetExe := filepath.Join(dir, "Velora.exe")
	targetIcon := filepath.Join(dir, "icon.ico")
	targetUninst := filepath.Join(dir, "uninstall.exe")
	_ = os.WriteFile(targetExe, veloraExeBytes, 0o755)
	_ = os.WriteFile(targetIcon, veloraIconBytes, 0o644)
	if selfExe, err := os.Executable(); err == nil {
		if selfBytes, readErr := os.ReadFile(selfExe); readErr == nil {
			_ = os.WriteFile(targetUninst, selfBytes, 0o755)
		}
	}
	desktopLnk, startMenuLnk := shortcutPaths()
	_ = createShortcut(desktopLnk, targetExe, targetIcon, dir)
	_ = createShortcut(startMenuLnk, targetExe, targetIcon, dir)
	if k, _, regErr := registry.CreateKey(registry.CURRENT_USER, uninstallKey, registry.SET_VALUE); regErr == nil {
		_ = k.SetStringValue("DisplayName", appName)
		_ = k.SetStringValue("DisplayVersion", appVersion)
		_ = k.SetStringValue("Publisher", appPublisher)
		_ = k.SetStringValue("URLInfoAbout", appURL)
		_ = k.SetStringValue("InstallDate", time.Now().Format("20060102"))
		_ = k.SetStringValue("InstallLocation", dir)
		_ = k.SetStringValue("DisplayIcon", targetIcon)
		_ = k.SetStringValue("UninstallString", fmt.Sprintf(`"%s" --uninstall`, targetUninst))
		_ = k.SetDWordValue("NoModify", 1)
		_ = k.SetDWordValue("NoRepair", 1)
		_ = k.SetDWordValue("EstimatedSize", uint32(len(veloraExeBytes)/1024))
		_ = k.Close()
	}
}

func runUninstall(silent bool) {
	if !silent {
		if messageBox(
			0,
			appName+" Uninstall",
			"Are you sure you want to completely remove "+appName+" and all of its application data, cache, downloads, and settings?",
			mbYesNo|mbIconWarning,
		) != idYes {
			return
		}
	}

	stopRunningApp()

	desktopLnk, startMenuLnk := shortcutPaths()
	_ = os.Remove(desktopLnk)
	_ = os.Remove(startMenuLnk)
	if kRun, err := registry.OpenKey(registry.CURRENT_USER, startupKey, registry.SET_VALUE); err == nil {
		_ = kRun.DeleteValue(appName)
		_ = kRun.Close()
	}

	if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
		_ = os.RemoveAll(filepath.Join(cfgDir, appName))
		_ = os.RemoveAll(filepath.Join(cfgDir, appName+".exe.WebView2"))
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		_ = os.RemoveAll(filepath.Join(appData, appName))
		_ = os.RemoveAll(filepath.Join(appData, appName+".exe.WebView2"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		_ = os.RemoveAll(filepath.Join(localAppData, appName))
		_ = os.RemoveAll(filepath.Join(localAppData, appName+".exe.WebView2"))
	}

	installLocation := defaultInstallDir()
	if k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE); err == nil {
		if loc, _, locErr := k.GetStringValue("InstallLocation"); locErr == nil && strings.TrimSpace(loc) != "" {
			installLocation = loc
		}
		_ = k.Close()
	}
	_ = registry.DeleteKey(registry.CURRENT_USER, uninstallKey)

	_ = os.Remove(filepath.Join(installLocation, "Velora.exe"))
	_ = os.Remove(filepath.Join(installLocation, "icon.ico"))
	cmd := exec.Command("cmd.exe", "/C", fmt.Sprintf(`ping 127.0.0.1 -n 2 >nul & rmdir /S /Q "%s"`, installLocation))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()

	if !silent {
		messageBox(0, appName+" Uninstall", appName+" and all of its data have been completely removed from your computer.", mbOK|mbIconInformation)
	}
}

func runGUIInstaller(forceUninstall bool) {
	runtime.LockOSThread()
	icex := initCommonControlsExStruct{dwSize: uint32(unsafe.Sizeof(initCommonControlsExStruct{})), dwICC: 0x000040FF}
	procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icex)))

	dpi, _, _ := procGetDpiForSystem.Call()
	if dpi == 0 {
		dpi = 96
	}
	wiz.dpi = int32(dpi)
	installed, existingDir, existingVer := detectExistingInstall()
	wiz.installDir = existingDir
	wiz.existingVersion = existingVer
	if forceUninstall {
		wiz.isUninstallMode = true
	} else if installed {
		if existingVer != "" && existingVer != appVersion {
			wiz.isUpdate = true
		} else {
			wiz.isMaintenance = true
		}
	}

	wiz.hFontTitle = makeFont(14, 600)
	wiz.hFontHdr = makeFont(11, 600)
	wiz.hFontBold = makeFont(10, 600)
	wiz.hFontBody = makeFont(10, 400)
	wiz.hBrushWhite, _, _ = procCreateSolidBrush.Call(rgb(255, 255, 255))
	wiz.hBrushFooter, _, _ = procCreateSolidBrush.Call(rgb(240, 240, 240))
	wiz.hBrushLine, _, _ = procCreateSolidBrush.Call(rgb(218, 220, 224))

	hInst, _, _ := procGetModuleHandleW.Call(0)
	wiz.hIconBig, _, _ = procLoadImageW.Call(hInst, 1, 1, uintptr(scale(40)), uintptr(scale(40)), 0)
	if wiz.hIconBig == 0 {
		wiz.hIconBig, _, _ = procLoadImageW.Call(hInst, 2, 1, uintptr(scale(40)), uintptr(scale(40)), 0)
	}

	cursor, _, _ := procLoadCursorW.Call(0, 32512) // IDC_ARROW
	clsName := u16("VeloraSetupWizardWnd")
	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     hInst,
		hIcon:         wiz.hIconBig,
		hCursor:       cursor,
		hbrBackground: wiz.hBrushWhite,
		lpszClassName: clsName,
		hIconSm:       wiz.hIconBig,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	winW := scale(556)
	winH := scale(422)
	scrW, _, _ := procGetSystemMetrics.Call(0)
	scrH, _, _ := procGetSystemMetrics.Call(1)
	winX := (int32(scrW) - winW) / 2
	winY := (int32(scrH) - winH) / 2

	winTitle := "Setup - Velora"
	if forceUninstall {
		winTitle = "Uninstall - Velora"
	}
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(clsName)),
		uintptr(unsafe.Pointer(u16(winTitle))),
		wsOverlapped|wsCaption|wsSysMenu|wsMinBox|wsVisible,
		uintptr(winX), uintptr(winY), uintptr(winW), uintptr(winH),
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)

	var m msgStruct
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func main() {
	silent := false
	uninstall := false
	for _, arg := range os.Args[1:] {
		low := strings.ToLower(strings.TrimSpace(arg))
		if low == "/s" || low == "-s" || low == "--silent" {
			silent = true
		} else if low == "--uninstall" || low == "/uninstall" {
			uninstall = true
		}
	}
	if uninstall {
		if silent {
			runUninstall(true)
		} else {
			runGUIInstaller(true)
		}
		return
	}
	if silent {
		runSilentInstall()
		return
	}
	runGUIInstaller(false)
}
