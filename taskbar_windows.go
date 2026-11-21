//go:build windows

package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	wmCommand                      = 0x0111
	wmDwmSendIconicThumbnail       = 0x0323
	wmDwmSendIconicLivePreview     = 0x0326
	wmAppTaskbarInit               = 0x8000 + 101
	wmAppTaskbarSync               = 0x8000 + 102
	gwlpWndProc                    = -4
	thbnClicked                    = 0x1800
	thbIcon                        = 0x0002
	thbTooltip                     = 0x0004
	thbFlags                       = 0x0008
	thbfEnabled                    = 0x0000
	dwmwaForceIconicRepresentation = 7
	dwmwaHasIconicBitmap           = 10
	msgfltAllow                    = 1

	thumbBtnPrev   = 1001
	thumbBtnToggle = 1002
	thumbBtnNext   = 1003
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	ole32    = windows.NewLazySystemDLL("ole32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procEnumWindows                 = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId    = user32.NewProc("GetWindowThreadProcessId")
	procGetWindow                   = user32.NewProc("GetWindow")
	procGetParent                   = user32.NewProc("GetParent")
	procGetWindowTextW              = user32.NewProc("GetWindowTextW")
	procGetClientRect               = user32.NewProc("GetClientRect")
	procSetWindowLongPtrW           = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW             = user32.NewProc("CallWindowProcW")
	procPostMessageW                = user32.NewProc("PostMessageW")
	procRegisterWindowMessageW      = user32.NewProc("RegisterWindowMessageW")
	procChangeWindowMessageFilterEx = user32.NewProc("ChangeWindowMessageFilterEx")
	procCreateIconIndirect          = user32.NewProc("CreateIconIndirect")
	procDestroyIcon                 = user32.NewProc("DestroyIcon")

	procCreateDIBSection = gdi32.NewProc("CreateDIBSection")
	procCreateBitmap     = gdi32.NewProc("CreateBitmap")
	procDeleteObject     = gdi32.NewProc("DeleteObject")

	procCoCreateInstance = ole32.NewProc("CoCreateInstance")

	procDwmSetWindowAttribute         = dwmapi.NewProc("DwmSetWindowAttribute")
	procDwmInvalidateIconicBitmaps    = dwmapi.NewProc("DwmInvalidateIconicBitmaps")
	procDwmSetIconicThumbnail         = dwmapi.NewProc("DwmSetIconicThumbnail")
	procDwmSetIconicLivePreviewBitmap = dwmapi.NewProc("DwmSetIconicLivePreviewBitmap")

	procGetCurrentProcessId = kernel32.NewProc("GetCurrentProcessId")
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type thumbButton struct {
	DwMask  uint32
	IId     uint32
	IBitmap uint32
	HIcon   uintptr
	SzTip   [260]uint16
	DwFlags uint32
}

type iTaskbarList3Vtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	HrInit              uintptr
	AddTab              uintptr
	DeleteTab           uintptr
	ActivateTab         uintptr
	SetActiveAlt        uintptr
	MarkFullscreenWin   uintptr
	SetProgressValue    uintptr
	SetProgressState    uintptr
	RegisterTab         uintptr
	UnregisterTab       uintptr
	SetTabOrder         uintptr
	SetTabActive        uintptr
	ThumbBarAddButtons  uintptr
	ThumbBarUpdateBtns  uintptr
	ThumbBarSetImgList  uintptr
	SetOverlayIcon      uintptr
	SetThumbnailTooltip uintptr
	SetThumbnailClip    uintptr
}

type iTaskbarList3 struct {
	lpVtbl *iTaskbarList3Vtbl
}

type taskbarController struct {
	ctx        context.Context
	artworkDir string

	hwnd             uintptr
	origWndProc      uintptr
	wmTaskbarCreated uint32
	taskbarPtr       *iTaskbarList3
	thumbAdded       bool

	iconPrev  uintptr
	iconPlay  uintptr
	iconPause uintptr
	iconNext  uintptr

	mu        sync.Mutex
	isPlaying bool
	coverURL  string
	coverImg  image.Image
	iconicOn  bool
	fetchSeq  uint64
}

var (
	globalTaskbar   *taskbarController
	globalTaskbarMu sync.Mutex

	pendingMu      sync.Mutex
	pendingTitle   string
	pendingArtist  string
	pendingCover   string
	pendingPlaying bool
	hasPending     bool
)

func (a *App) initTaskbar(ctx context.Context) {
	go func() {
		var hwnd uintptr
		for i := 0; i < 60; i++ {
			hwnd = findMainWindowHWND()
			if hwnd != 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if hwnd == 0 {
			return
		}

		ctrl := &taskbarController{
			ctx:        ctx,
			artworkDir: a.artworkDir,
			hwnd:       hwnd,
			iconPrev:   createGlyphIcon("prev"),
			iconPlay:   createGlyphIcon("play"),
			iconPause:  createGlyphIcon("pause"),
			iconNext:   createGlyphIcon("next"),
		}

		msgName, _ := syscall.UTF16PtrFromString("TaskbarButtonCreated")
		wmCreated, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(msgName)))
		ctrl.wmTaskbarCreated = uint32(wmCreated)

		_, _, _ = procChangeWindowMessageFilterEx.Call(hwnd, uintptr(ctrl.wmTaskbarCreated), msgfltAllow, 0)
		_, _, _ = procChangeWindowMessageFilterEx.Call(hwnd, wmCommand, msgfltAllow, 0)

		globalTaskbarMu.Lock()
		globalTaskbar = ctrl
		globalTaskbarMu.Unlock()

		cb := syscall.NewCallback(taskbarWndProc)
		orig, _, _ := procSetWindowLongPtrW.Call(hwnd, ^uintptr(3), cb) // GWLP_WNDPROC = -4
		ctrl.origWndProc = orig

		_, _, _ = procPostMessageW.Call(hwnd, wmAppTaskbarInit, 0, 0)

		pendingMu.Lock()
		pTitle, pArtist, pCover, pPlaying, pOk := pendingTitle, pendingArtist, pendingCover, pendingPlaying, hasPending
		pendingMu.Unlock()
		if pOk {
			a.UpdateTaskbarState(pTitle, pArtist, pCover, pPlaying)
		}

		time.Sleep(400 * time.Millisecond)
		_, _, _ = procPostMessageW.Call(hwnd, wmAppTaskbarInit, 0, 0)
		time.Sleep(1000 * time.Millisecond)
		_, _, _ = procPostMessageW.Call(hwnd, wmAppTaskbarInit, 0, 0)
	}()
}

func (a *App) shutdownTaskbar() {
	globalTaskbarMu.Lock()
	ctrl := globalTaskbar
	globalTaskbar = nil
	globalTaskbarMu.Unlock()
	if ctrl == nil {
		return
	}
	if ctrl.origWndProc != 0 && ctrl.hwnd != 0 {
		_, _, _ = procSetWindowLongPtrW.Call(ctrl.hwnd, ^uintptr(3), ctrl.origWndProc)
	}
	for _, ic := range []uintptr{ctrl.iconPrev, ctrl.iconPlay, ctrl.iconPause, ctrl.iconNext} {
		if ic != 0 {
			_, _, _ = procDestroyIcon.Call(ic)
		}
	}
}

func (a *App) UpdateTaskbarState(title string, artist string, coverURL string, isPlaying bool) {
	a.updateDynamicIslandState(title, artist, coverURL, isPlaying)

	pendingMu.Lock()
	pendingTitle = title
	pendingArtist = artist
	pendingCover = coverURL
	pendingPlaying = isPlaying
	hasPending = true
	pendingMu.Unlock()

	globalTaskbarMu.Lock()
	ctrl := globalTaskbar
	globalTaskbarMu.Unlock()
	if ctrl == nil || ctrl.hwnd == 0 {
		return
	}

	ctrl.mu.Lock()
	ctrl.isPlaying = isPlaying
	coverChanged := ctrl.coverURL != coverURL
	if coverChanged {
		ctrl.coverURL = coverURL
		ctrl.fetchSeq++
	}
	seq := ctrl.fetchSeq
	ctrl.mu.Unlock()

	_, _, _ = procPostMessageW.Call(ctrl.hwnd, wmAppTaskbarSync, 0, 0)

	if coverChanged {
		if strings.TrimSpace(coverURL) == "" {
			ctrl.mu.Lock()
			if ctrl.fetchSeq == seq {
				ctrl.coverImg = nil
			}
			ctrl.mu.Unlock()
			_, _, _ = procPostMessageW.Call(ctrl.hwnd, wmAppTaskbarSync, 1, 0)
			return
		}
		go func(u string, expectedSeq uint64) {
			img := loadCoverImage(u, ctrl.artworkDir)
			ctrl.mu.Lock()
			if ctrl.fetchSeq == expectedSeq {
				ctrl.coverImg = img
			}
			ctrl.mu.Unlock()
			_, _, _ = procPostMessageW.Call(ctrl.hwnd, wmAppTaskbarSync, 1, 0)
		}(coverURL, seq)
	}
}

func taskbarWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	globalTaskbarMu.Lock()
	ctrl := globalTaskbar
	globalTaskbarMu.Unlock()
	if ctrl == nil {
		return 0
	}

	m := uint32(msg)
	switch {
	case m == wmAppTaskbarInit || (ctrl.wmTaskbarCreated != 0 && m == ctrl.wmTaskbarCreated):
		ctrl.ensureThumbButtons()
		ctrl.syncThumbAndIconic(false)
		if m == wmAppTaskbarInit {
			return 0
		}
	case m == wmAppTaskbarSync:
		ctrl.ensureThumbButtons()
		ctrl.syncThumbAndIconic(wParam == 1)
		return 0
	case m == wmCommand:
		hi := uint16((wParam >> 16) & 0xFFFF)
		lo := uint16(wParam & 0xFFFF)
		if hi == thbnClicked {
			switch lo {
			case thumbBtnPrev:
				runtime.EventsEmit(ctrl.ctx, "taskbar:control", "prev")
				return 0
			case thumbBtnToggle:
				runtime.EventsEmit(ctrl.ctx, "taskbar:control", "toggle")
				return 0
			case thumbBtnNext:
				runtime.EventsEmit(ctrl.ctx, "taskbar:control", "next")
				return 0
			}
		}
	case m == wmDwmSendIconicThumbnail:
		maxW := int(int16((lParam >> 16) & 0xFFFF))
		maxH := int(int16(lParam & 0xFFFF))
		if maxW <= 0 {
			maxW = 200
		}
		if maxH <= 0 {
			maxH = 120
		}
		ctrl.mu.Lock()
		img := ctrl.coverImg
		ctrl.mu.Unlock()
		if img != nil {
			if hbmp := buildThumbnailBitmap(img, maxW, maxH); hbmp != 0 {
				_, _, _ = procDwmSetIconicThumbnail.Call(hwnd, hbmp, 0)
				_, _, _ = procDeleteObject.Call(hbmp)
				return 0
			}
		}
	case m == wmDwmSendIconicLivePreview:
		ctrl.mu.Lock()
		img := ctrl.coverImg
		ctrl.mu.Unlock()
		if img != nil {
			var rc rect
			_, _, _ = procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
			w := int(rc.Right - rc.Left)
			h := int(rc.Bottom - rc.Top)
			if w <= 0 || h <= 0 {
				w, h = 640, 360
			}
			if hbmp := buildThumbnailBitmap(img, w, h); hbmp != 0 {
				_, _, _ = procDwmSetIconicLivePreviewBitmap.Call(hwnd, hbmp, 0, 0)
				_, _, _ = procDeleteObject.Call(hbmp)
				return 0
			}
		}
	}

	ret, _, _ := procCallWindowProcW.Call(ctrl.origWndProc, hwnd, msg, wParam, lParam)
	return ret
}

func (c *taskbarController) ensureTaskbarCOM() bool {
	if c.taskbarPtr != nil {
		return true
	}
	_ = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
	clsid := ole.NewGUID("{56FDF344-FD6D-11d0-958A-006097C9A090}")
	iid := ole.NewGUID("{EA1AFB91-9E28-4B86-90E9-9E9F8A5EEFAF}")
	var ptr *iTaskbarList3
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(clsid)),
		0,
		1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&ptr)),
	)
	if hr != 0 || ptr == nil {
		return false
	}
	hrInit, _, _ := syscall.SyscallN(ptr.lpVtbl.HrInit, uintptr(unsafe.Pointer(ptr)))
	if hrInit != 0 {
		return false
	}
	c.taskbarPtr = ptr
	return true
}

func (c *taskbarController) buildThumbButtons(playing bool) [3]thumbButton {
	var btns [3]thumbButton
	btns[0] = thumbButton{
		DwMask:  thbIcon | thbTooltip | thbFlags,
		IId:     thumbBtnPrev,
		HIcon:   c.iconPrev,
		DwFlags: thbfEnabled,
	}
	copyTip(&btns[0].SzTip, "Previous")

	toggleIcon := c.iconPlay
	toggleTip := "Play"
	if playing {
		toggleIcon = c.iconPause
		toggleTip = "Pause"
	}
	btns[1] = thumbButton{
		DwMask:  thbIcon | thbTooltip | thbFlags,
		IId:     thumbBtnToggle,
		HIcon:   toggleIcon,
		DwFlags: thbfEnabled,
	}
	copyTip(&btns[1].SzTip, toggleTip)

	btns[2] = thumbButton{
		DwMask:  thbIcon | thbTooltip | thbFlags,
		IId:     thumbBtnNext,
		HIcon:   c.iconNext,
		DwFlags: thbfEnabled,
	}
	copyTip(&btns[2].SzTip, "Next")
	return btns
}

func (c *taskbarController) ensureThumbButtons() {
	if c.thumbAdded || !c.ensureTaskbarCOM() {
		return
	}
	c.mu.Lock()
	playing := c.isPlaying
	c.mu.Unlock()

	btns := c.buildThumbButtons(playing)
	hr, _, _ := syscall.SyscallN(
		c.taskbarPtr.lpVtbl.ThumbBarAddButtons,
		uintptr(unsafe.Pointer(c.taskbarPtr)),
		c.hwnd,
		3,
		uintptr(unsafe.Pointer(&btns[0])),
	)
	if hr == 0 {
		c.thumbAdded = true
	}
}

func (c *taskbarController) syncThumbAndIconic(_ bool) {
	c.mu.Lock()
	playing := c.isPlaying
	wasIconic := c.iconicOn
	c.mu.Unlock()

	if c.thumbAdded && c.taskbarPtr != nil {
		btns := c.buildThumbButtons(playing)
		_, _, _ = syscall.SyscallN(
			c.taskbarPtr.lpVtbl.ThumbBarUpdateBtns,
			uintptr(unsafe.Pointer(c.taskbarPtr)),
			c.hwnd,
			3,
			uintptr(unsafe.Pointer(&btns[0])),
		)
	}

	if wasIconic {
		var val int32
		_, _, _ = procDwmSetWindowAttribute.Call(c.hwnd, dwmwaForceIconicRepresentation, uintptr(unsafe.Pointer(&val)), 4)
		_, _, _ = procDwmSetWindowAttribute.Call(c.hwnd, dwmwaHasIconicBitmap, uintptr(unsafe.Pointer(&val)), 4)
		c.mu.Lock()
		c.iconicOn = false
		c.mu.Unlock()
	}
}

func copyTip(dst *[260]uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	n := len(u)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:n], u[:n])
}

func findMainWindowHWND() uintptr {
	pid, _, _ := procGetCurrentProcessId.Call()
	var found uintptr
	var fallback uintptr
	cb := syscall.NewCallback(func(hwnd, lParam uintptr) uintptr {
		var wPid uint32
		_, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&wPid)))
		if uintptr(wPid) != pid {
			return 1
		}
		parent, _, _ := procGetParent.Call(hwnd)
		if parent != 0 {
			return 1
		}
		owner, _, _ := procGetWindow.Call(hwnd, 4) // GW_OWNER = 4
		if owner != 0 {
			return 1
		}
		var buf [128]uint16
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		title := syscall.UTF16ToString(buf[:n])
		if title == "Velora" {
			found = hwnd
			return 0
		}
		if fallback == 0 && title != "" {
			fallback = hwnd
		}
		return 1
	})
	_, _, _ = procEnumWindows.Call(cb, 0)
	if found != 0 {
		return found
	}
	return fallback
}

func loadCoverImage(coverURL string, artworkDir string) image.Image {
	u := strings.TrimSpace(coverURL)
	if u == "" {
		return nil
	}

	if idx := strings.Index(u, "/artwork/"); idx >= 0 && artworkDir != "" {
		rel := strings.TrimPrefix(u[idx+len("/artwork/"):], "/")
		if q := strings.IndexByte(rel, '?'); q >= 0 {
			rel = rel[:q]
		}
		if rel != "" {
			if data, err := os.ReadFile(filepath.Join(artworkDir, filepath.FromSlash(rel))); err == nil {
				if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
					return img
				}
			}
		}
	}

	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		if data, err := os.ReadFile(u); err == nil {
			if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
				return img
			}
		}
		return nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "VeloraMusicPlayer/1.0")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

func buildThumbnailBitmap(src image.Image, w, h int) uintptr {
	if w <= 0 || h <= 0 || src == nil {
		return 0
	}
	bi := bitmapInfo{
		BmiHeader: bitmapInfoHeader{
			BiSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			BiWidth:       int32(w),
			BiHeight:      -int32(h), // top-down DIB
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: 0, // BI_RGB
		},
	}
	var bits unsafe.Pointer
	hbmp, _, _ := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbmp == 0 || bits == nil {
		return 0
	}

	pixels := unsafe.Slice((*uint32)(bits), w*h)

	sb := src.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	if sw <= 0 || sh <= 0 {
		return hbmp
	}

	bgBuf := make([]uint32, w*h)
	scaleCoverW := w
	scaleCoverH := w * sh / sw
	if scaleCoverH < h {
		scaleCoverH = h
		scaleCoverW = h * sw / sh
	}
	bgOffX := (scaleCoverW - w) / 2
	bgOffY := (scaleCoverH - h) / 2
	for y := 0; y < h; y++ {
		sy := sb.Min.Y + ((y+bgOffY)*sh)/scaleCoverH
		if sy >= sb.Max.Y {
			sy = sb.Max.Y - 1
		}
		for x := 0; x < w; x++ {
			sx := sb.Min.X + ((x+bgOffX)*sw)/scaleCoverW
			if sx >= sb.Max.X {
				sx = sb.Max.X - 1
			}
			c := color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA)
			bgBuf[y*w+x] = (uint32(c.R) << 16) | (uint32(c.G) << 8) | uint32(c.B)
		}
	}

	tmpBuf := make([]uint32, w*h)
	const radius = 10
	for pass := 0; pass < 2; pass++ {
		for y := 0; y < h; y++ {
			row := y * w
			for x := 0; x < w; x++ {
				var sr, sg, sbSum, cnt uint32
				x0 := x - radius
				if x0 < 0 {
					x0 = 0
				}
				x1 := x + radius
				if x1 >= w {
					x1 = w - 1
				}
				for kx := x0; kx <= x1; kx++ {
					p := bgBuf[row+kx]
					sr += (p >> 16) & 0xFF
					sg += (p >> 8) & 0xFF
					sbSum += p & 0xFF
					cnt++
				}
				tmpBuf[row+x] = ((sr / cnt) << 16) | ((sg / cnt) << 8) | (sbSum / cnt)
			}
		}
		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				var sr, sg, sbSum, cnt uint32
				y0 := y - radius
				if y0 < 0 {
					y0 = 0
				}
				y1 := y + radius
				if y1 >= h {
					y1 = h - 1
				}
				for ky := y0; ky <= y1; ky++ {
					p := tmpBuf[ky*w+x]
					sr += (p >> 16) & 0xFF
					sg += (p >> 8) & 0xFF
					sbSum += p & 0xFF
					cnt++
				}
				bgBuf[y*w+x] = ((sr / cnt) << 16) | ((sg / cnt) << 8) | (sbSum / cnt)
			}
		}
	}

	for i, p := range bgBuf {
		r := (((p >> 16) & 0xFF) * 48) / 100
		g := (((p >> 8) & 0xFF) * 48) / 100
		b := ((p & 0xFF) * 48) / 100
		pixels[i] = (uint32(0xFF) << 24) | (r << 16) | (g << 8) | b
	}

	targetW := w
	targetH := targetW * sh / sw
	if targetH > h {
		targetH = h
		targetW = targetH * sw / sh
	}
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}
	offX := (w - targetW) / 2
	offY := (h - targetH) / 2

	for dy := 0; dy < targetH; dy++ {
		sy := sb.Min.Y + (dy*sh)/targetH
		rowOffset := (offY + dy) * w
		for dx := 0; dx < targetW; dx++ {
			sx := sb.Min.X + (dx*sw)/targetW
			c := color.NRGBAModel.Convert(src.At(sx, sy)).(color.NRGBA)
			pixels[rowOffset+offX+dx] = (uint32(0xFF) << 24) | (uint32(c.R) << 16) | (uint32(c.G) << 8) | uint32(c.B)
		}
	}

	return hbmp
}

func createGlyphIcon(kind string) uintptr {
	const size = 32
	bi := bitmapInfo{
		BmiHeader: bitmapInfoHeader{
			BiSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			BiWidth:       size,
			BiHeight:      -size,
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: 0,
		},
	}
	var bits unsafe.Pointer
	colorBmp, _, _ := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if colorBmp == 0 || bits == nil {
		return 0
	}
	maskBmp, _, _ := procCreateBitmap.Call(size, size, 1, 1, 0)

	pixels := unsafe.Slice((*uint32)(bits), size*size)

	inside := func(x, y float64) bool {
		switch kind {
		case "play":
			if x < 10 || x > 24 {
				return false
			}
			halfH := 9.0 * (24.0 - x) / 14.0
			return math.Abs(y-16.0) <= halfH
		case "pause":
			if y < 7 || y > 25 {
				return false
			}
			return (x >= 8 && x <= 13) || (x >= 19 && x <= 24)
		case "prev":
			if y < 8 || y > 24 {
				return false
			}
			if x >= 7 && x <= 10 {
				return true
			}
			if x >= 12 && x <= 24 {
				halfH := 8.0 * (x - 12.0) / 12.0
				return math.Abs(y-16.0) <= halfH
			}
			return false
		case "next":
			if y < 8 || y > 24 {
				return false
			}
			if x >= 22 && x <= 25 {
				return true
			}
			if x >= 8 && x <= 20 {
				halfH := 8.0 * (20.0 - x) / 12.0
				return math.Abs(y-16.0) <= halfH
			}
			return false
		}
		return false
	}

	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			hits := 0
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					fx := float64(px) + (float64(sx)+0.5)/4.0
					fy := float64(py) + (float64(sy)+0.5)/4.0
					if inside(fx, fy) {
						hits++
					}
				}
			}
			alpha := uint32(hits * 255 / 16)
			pixels[py*size+px] = (alpha << 24) | (alpha << 16) | (alpha << 8) | alpha
		}
	}

	ii := iconInfo{
		FIcon:    1,
		HbmMask:  maskBmp,
		HbmColor: colorBmp,
	}
	hicon, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	_, _, _ = procDeleteObject.Call(colorBmp)
	if maskBmp != 0 {
		_, _, _ = procDeleteObject.Call(maskBmp)
	}
	return hicon
}
