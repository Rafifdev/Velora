//go:build windows

package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"velora/internal/sources"
)

// Direct2D 1.0 & DirectWrite COM interop for crisp ClearType HD text rendering
// in the Windows Dynamic Island layered window.

var (
	diModD2D1                 = windows.NewLazySystemDLL("d2d1.dll")
	diProcD2D1CreateFactory   = diModD2D1.NewProc("D2D1CreateFactory")
	diModDWrite               = windows.NewLazySystemDLL("dwrite.dll")
	diProcDWriteCreateFactory = diModDWrite.NewProc("DWriteCreateFactory")
)

var (
	iidID2D1Factory = windows.GUID{
		Data1: 0x06152247,
		Data2: 0x6f50,
		Data3: 0x465a,
		Data4: [8]byte{0x92, 0x45, 0x11, 0x8b, 0xfd, 0x3b, 0x60, 0x07},
	}
	iidIDWriteFactory = windows.GUID{
		Data1: 0xb859ee5a,
		Data2: 0xd838,
		Data3: 0x4b5b,
		Data4: [8]byte{0xa2, 0xe8, 0x1a, 0xdc, 0x7d, 0x93, 0xdb, 0x48},
	}
	iidIDWriteFactory5 = windows.GUID{
		Data1: 0x958DB99A,
		Data2: 0xBE2A,
		Data3: 0x4F09,
		Data4: [8]byte{0xAF, 0x7D, 0x65, 0x18, 0x98, 0x03, 0xD1, 0xD3},
	}
)

const (
	d2dRenderTargetTypeSoftware = 1
	dxgiFormatB8G8R8A8UNORM     = 87
	d2dAlphaModeIgnore          = 3

	d2dTextAAModeClearType    = 1
	d2dTextAAModeGrayscale    = 2
	d2dAAModePerPrimitive     = 0
	d2dDrawTextOptNoSnap      = 0x00000001
	d2dDrawTextOptClip        = 0x00000002
	d2dDrawTextOptClipNoSnap  = d2dDrawTextOptClip | d2dDrawTextOptNoSnap

	dwFontWeightNormal   = 400
	dwFontWeightSemiBold = 600
	dwFontWeightBold     = 700
	dwFontStyleNormal    = 0
	dwFontStretchNormal  = 5

	dwTextAlignLeading  = 0
	dwTextAlignTrailing = 1
	dwTextAlignCenter   = 2

	dwParaAlignNear   = 0
	dwParaAlignCenter = 2

	dwWordWrapWrap   = 0
	dwWordWrapNoWrap = 1

	dwTrimGranularityNone = 0
	dwTrimGranularityChar = 1
)

type d2dPixelFormat struct {
	Format    uint32
	AlphaMode uint32
}

type d2dRenderTargetProps struct {
	Type        uint32
	PixelFormat d2dPixelFormat
	DpiX        float32
	DpiY        float32
	Usage       uint32
	MinLevel    uint32
}

type d2dRectF struct {
	Left   float32
	Top    float32
	Right  float32
	Bottom float32
}

type d2dColorF struct {
	R float32
	G float32
	B float32
	A float32
}

type dwriteTrimming struct {
	Granularity    uint32
	Delimiter      uint32
	DelimiterCount uint32
}

type dwriteTextMetrics struct {
	Left                             float32
	Top                              float32
	Width                            float32
	WidthIncludingTrailingWhitespace float32
	Height                           float32
	LayoutWidth                      float32
	LayoutHeight                     float32
	MaxBidiReorderingDepth           uint32
	LineCount                        uint32
}

type d2dFmtKey struct {
	familyKind uint8 // 0=Regular, 1=SemiBold, 2=Bold
	sizeQPx    int32 // round(sizePx * 4)
	align      uint8 // 0=Leading, 1=Trailing, 2=Center
	paraAlign  uint8 // 0=Near, 2=Center
	wrap       bool
	ellipsis   bool
}

func comVtblCall(obj uintptr, index int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	var stack [12]uintptr
	stack[0] = obj
	copy(stack[1:], args)
	r1, _, _ := syscall.SyscallN(fn, stack[:1+len(args)]...)
	return r1
}

func comQueryInterface(obj uintptr, iid *windows.GUID, out *uintptr) uintptr {
	return comVtblCall(obj, 0, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(out)))
}

func colorToD2D(c color.RGBA, alphaMult float64) d2dColorF {
	a := clamp01((float64(c.A) / 255.0) * alphaMult)
	return d2dColorF{
		R: float32(c.R) / 255.0,
		G: float32(c.G) / 255.0,
		B: float32(c.B) / 255.0,
		A: float32(a),
	}
}

func (c *dynamicIslandController) initD2DText() {
	if c.d2dDCRT != 0 {
		return
	}
	if diModD2D1.Load() != nil || diModDWrite.Load() != nil {
		return
	}

	var d2dFac uintptr
	hr, _, _ := diProcD2D1CreateFactory.Call(
		0, // D2D1_FACTORY_TYPE_SINGLE_THREADED
		uintptr(unsafe.Pointer(&iidID2D1Factory)),
		0,
		uintptr(unsafe.Pointer(&d2dFac)),
	)
	if int32(hr) < 0 || d2dFac == 0 {
		return
	}
	c.d2dFactory = d2dFac

	rtProps := d2dRenderTargetProps{
		Type: d2dRenderTargetTypeSoftware,
		PixelFormat: d2dPixelFormat{
			Format:    dxgiFormatB8G8R8A8UNORM,
			AlphaMode: d2dAlphaModeIgnore,
		},
		DpiX: 96.0,
		DpiY: 96.0,
	}
	var dcRT uintptr
	hr = comVtblCall(d2dFac, 16, uintptr(unsafe.Pointer(&rtProps)), uintptr(unsafe.Pointer(&dcRT)))
	if int32(hr) < 0 || dcRT == 0 {
		c.destroyD2DText()
		return
	}
	c.d2dDCRT = dcRT
	comVtblCall(dcRT, 34, d2dTextAAModeClearType) // SetTextAntialiasMode

	initCol := d2dColorF{R: 1, G: 1, B: 1, A: 1}
	var brush uintptr
	hr = comVtblCall(dcRT, 8, uintptr(unsafe.Pointer(&initCol)), 0, uintptr(unsafe.Pointer(&brush)))
	if int32(hr) < 0 || brush == 0 {
		c.destroyD2DText()
		return
	}
	c.d2dBrush = brush

	var dwFac uintptr
	hr, _, _ = diProcDWriteCreateFactory.Call(
		0, // DWRITE_FACTORY_TYPE_SHARED
		uintptr(unsafe.Pointer(&iidIDWriteFactory)),
		uintptr(unsafe.Pointer(&dwFac)),
	)
	if int32(hr) < 0 || dwFac == 0 {
		c.destroyD2DText()
		return
	}
	c.dwFactory = dwFac

	c.initDWriteEmbeddedFonts()
	c.d2dFmtCache = make(map[d2dFmtKey]uintptr, 32)
}

func (c *dynamicIslandController) initDWriteEmbeddedFonts() {
	if c.dwFactory == 0 {
		return
	}
	var dwFac5 uintptr
	hr := comQueryInterface(c.dwFactory, &iidIDWriteFactory5, &dwFac5)
	if int32(hr) < 0 || dwFac5 == 0 {
		return
	}
	defer comRelease(dwFac5)

	var memLoader uintptr
	hr = comVtblCall(dwFac5, 44, uintptr(unsafe.Pointer(&memLoader))) // CreateInMemoryFontFileLoader
	if int32(hr) < 0 || memLoader == 0 {
		return
	}

	hr = comVtblCall(c.dwFactory, 13, memLoader) // RegisterFontFileLoader
	if int32(hr) < 0 {
		comRelease(memLoader)
		return
	}
	c.dwMemLoader = memLoader

	var fsb1 uintptr
	hr = comVtblCall(dwFac5, 43, uintptr(unsafe.Pointer(&fsb1))) // CreateFontSetBuilder(IDWriteFontSetBuilder1**)
	if int32(hr) < 0 || fsb1 == 0 {
		return
	}
	defer comRelease(fsb1)

	for _, ttf := range [][]byte{poppinsRegularTTF, poppinsSemiBoldTTF, poppinsBoldTTF} {
		if len(ttf) == 0 {
			continue
		}
		var fontFile uintptr
		// IDWriteInMemoryFontFileLoader::CreateInMemoryFontFileReference [4]
		hr = comVtblCall(
			memLoader,
			4,
			c.dwFactory,
			uintptr(unsafe.Pointer(&ttf[0])),
			uintptr(len(ttf)),
			0,
			uintptr(unsafe.Pointer(&fontFile)),
		)
		if int32(hr) >= 0 && fontFile != 0 {
			_ = comVtblCall(fsb1, 7, fontFile) // AddFontFile
			comRelease(fontFile)
		}
	}

	var fontSet uintptr
	hr = comVtblCall(fsb1, 6, uintptr(unsafe.Pointer(&fontSet))) // CreateFontSet
	if int32(hr) < 0 || fontSet == 0 {
		return
	}
	defer comRelease(fontSet)

	var fontColl1 uintptr
	hr = comVtblCall(dwFac5, 37, fontSet, uintptr(unsafe.Pointer(&fontColl1))) // CreateFontCollectionFromFontSet
	if int32(hr) >= 0 && fontColl1 != 0 {
		c.dwFontColl = fontColl1
	}
}

func (c *dynamicIslandController) destroyD2DText() {
	for _, fmtPtr := range c.d2dFmtCache {
		comRelease(fmtPtr)
	}
	c.d2dFmtCache = nil

	if c.dwFontColl != 0 {
		comRelease(c.dwFontColl)
		c.dwFontColl = 0
	}
	if c.dwMemLoader != 0 && c.dwFactory != 0 {
		_ = comVtblCall(c.dwFactory, 14, c.dwMemLoader) // UnregisterFontFileLoader
	}
	if c.dwMemLoader != 0 {
		comRelease(c.dwMemLoader)
		c.dwMemLoader = 0
	}
	if c.dwFactory != 0 {
		comRelease(c.dwFactory)
		c.dwFactory = 0
	}
	if c.d2dBrush != 0 {
		comRelease(c.d2dBrush)
		c.d2dBrush = 0
	}
	if c.d2dDCRT != 0 {
		comRelease(c.d2dDCRT)
		c.d2dDCRT = 0
	}
	if c.d2dFactory != 0 {
		comRelease(c.d2dFactory)
		c.d2dFactory = 0
	}
	c.d2dAlphaScratch = nil
}

func (c *dynamicIslandController) getDWriteTextFormat(
	familyKind uint8,
	sizePx float64,
	align uint8,
	paraAlign uint8,
	wrap bool,
	ellipsis bool,
) uintptr {
	if c.dwFactory == 0 {
		return 0
	}
	if sizePx < 6.0 {
		sizePx = 6.0
	}
	sizeQ := int32(math.Round(sizePx * 4.0))
	key := d2dFmtKey{
		familyKind: familyKind,
		sizeQPx:    sizeQ,
		align:      align,
		paraAlign:  paraAlign,
		wrap:       wrap,
		ellipsis:   ellipsis,
	}
	if ptr, ok := c.d2dFmtCache[key]; ok && ptr != 0 {
		return ptr
	}

	// Keep cache bounded if DPI changes across monitors
	if len(c.d2dFmtCache) > 96 {
		for k, ptr := range c.d2dFmtCache {
			comRelease(ptr)
			delete(c.d2dFmtCache, k)
		}
	}

	famName := "Poppins"
	weight := uint32(dwFontWeightNormal)
	switch familyKind {
	case 1:
		if c.dwFontColl != 0 {
			// Embedded Poppins-SemiBold.ttf has Name ID 1 = "Poppins SemiBold", weight 400
			famName = "Poppins SemiBold"
			weight = dwFontWeightNormal
		} else {
			famName = "Segoe UI"
			weight = dwFontWeightSemiBold
		}
	case 2:
		if c.dwFontColl != 0 {
			famName = "Poppins"
			weight = dwFontWeightBold
		} else {
			famName = "Segoe UI"
			weight = dwFontWeightBold
		}
	default:
		if c.dwFontColl == 0 {
			famName = "Segoe UI"
		}
	}

	famU16, _ := windows.UTF16PtrFromString(famName)
	locU16, _ := windows.UTF16PtrFromString("en-US")
	actualSize := float32(sizeQ) / 4.0

	var tf uintptr
	hr := comVtblCall(
		c.dwFactory,
		15, // CreateTextFormat
		uintptr(unsafe.Pointer(famU16)),
		c.dwFontColl,
		uintptr(weight),
		dwFontStyleNormal,
		dwFontStretchNormal,
		uintptr(math.Float32bits(actualSize)),
		uintptr(unsafe.Pointer(locU16)),
		uintptr(unsafe.Pointer(&tf)),
	)
	if int32(hr) < 0 || tf == 0 {
		return 0
	}

	_ = comVtblCall(tf, 3, uintptr(align))     // SetTextAlignment
	_ = comVtblCall(tf, 4, uintptr(paraAlign)) // SetParagraphAlignment
	if wrap {
		_ = comVtblCall(tf, 5, dwWordWrapWrap) // SetWordWrapping
	} else {
		_ = comVtblCall(tf, 5, dwWordWrapNoWrap)
	}
	if ellipsis {
		var sign uintptr
		if int32(comVtblCall(c.dwFactory, 20, tf, uintptr(unsafe.Pointer(&sign)))) >= 0 && sign != 0 {
			trim := dwriteTrimming{Granularity: dwTrimGranularityChar}
			_ = comVtblCall(tf, 9, uintptr(unsafe.Pointer(&trim)), sign) // SetTrimming
			comRelease(sign)
		}
	}

	c.d2dFmtCache[key] = tf
	return tf
}

func (c *dynamicIslandController) measureDWriteText(txt string, tf uintptr, maxW, maxH float64) (float64, float64) {
	if c.dwFactory == 0 || tf == 0 || txt == "" {
		return 0, 0
	}
	u16, err := windows.UTF16FromString(txt)
	if err != nil || len(u16) <= 1 {
		return 0, 0
	}
	var layout uintptr
	hr := comVtblCall(
		c.dwFactory,
		18, // CreateTextLayout
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)-1),
		tf,
		uintptr(math.Float32bits(float32(maxW))),
		uintptr(math.Float32bits(float32(maxH))),
		uintptr(unsafe.Pointer(&layout)),
	)
	if int32(hr) < 0 || layout == 0 {
		return 0, 0
	}
	defer comRelease(layout)

	var m dwriteTextMetrics
	hr = comVtblCall(layout, 60, uintptr(unsafe.Pointer(&m))) // GetMetrics
	if int32(hr) < 0 {
		return 0, 0
	}
	return float64(m.WidthIncludingTrailingWhitespace), float64(m.Height)
}

func (c *dynamicIslandController) drawD2DStr(txt string, tf uintptr, col color.RGBA, alphaMult, px, py, pw, ph float64, noSnap bool) {
	if txt == "" || tf == 0 || c.d2dDCRT == 0 || c.d2dBrush == 0 || alphaMult <= 0.01 {
		return
	}
	u16, err := windows.UTF16FromString(txt)
	if err != nil || len(u16) <= 1 {
		return
	}
	dcol := colorToD2D(col, alphaMult)
	comVtblCall(c.d2dBrush, 8, uintptr(unsafe.Pointer(&dcol))) // SetColor
	rc := d2dRectF{
		Left:   float32(px),
		Top:    float32(py),
		Right:  float32(px + pw),
		Bottom: float32(py + ph),
	}
	opts := uintptr(d2dDrawTextOptClip)
	if noSnap {
		opts = d2dDrawTextOptClipNoSnap
	}
	comVtblCall(
		c.d2dDCRT,
		27, // DrawText
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)-1),
		tf,
		uintptr(unsafe.Pointer(&rc)),
		c.d2dBrush,
		opts,
		0, // DWRITE_MEASURING_MODE_NATURAL
	)
}

func (c *dynamicIslandController) pushD2DClip(px, py, pw, ph float64) {
	rc := d2dRectF{
		Left:   float32(px),
		Top:    float32(py),
		Right:  float32(px + pw),
		Bottom: float32(py + ph),
	}
	comVtblCall(c.d2dDCRT, 45, uintptr(unsafe.Pointer(&rc)), d2dAAModePerPrimitive)
}

func (c *dynamicIslandController) popD2DClip() {
	comVtblCall(c.d2dDCRT, 46)
}

func (c *dynamicIslandController) drawVeloraTextD2D(
	pix []byte,
	physW, physH int,
	nx, w, h, u float64,
	mode string,
	contentOp float64,
	mapX, mapXStatic, mapY, mapW, mapH func(float64) float64,
	title, artist string,
	pos, dur float64,
	playing bool,
	lyricsOpen bool,
	vol int,
	muted bool,
	eqPreset string,
	streakDays int,
	audioDev string,
	connType string,
	btBat int,
	sysBat int,
	isBt bool,
	audioOutputs []windowsAudioEndpoint,
	hpListOpen bool,
	hpListProg float64,
	lyricsLoading bool,
	lyrics []sources.LyricLine,
	lyricIdx int,
	prevLyricIdx int,
	lyricAnimProg float64,
	lyricCurY float64,
	lyricManualOffsetDp float64,
	hoverLyric int,
	closedLyricFrom string,
	closedLyricTo string,
	closedLyricProg float64,
	qTitle string,
	qIndex int,
	qScrollDp float64,
	qItems []DynamicIslandQueueItem,
	pal veloraBrandPalette,
	marqueeElapsed float64,
	diLang string,
	isScrolling bool,
) bool {
	if c.d2dDCRT == 0 || c.dwFactory == 0 || c.d2dBrush == 0 {
		return false
	}
	if contentOp <= 0.02 {
		return true
	}
	c.mu.Lock()
	showStatus := c.showStatus
	showCover := c.showCover
	showArtist := c.showArtist
	showLyric := c.showLyric
	c.mu.Unlock()
	diStr := getDIStrings(diLang)

	// Preserve existing alpha mask inside pix because D2D1_ALPHA_MODE_IGNORE
	// may zero alpha bytes on the bound DIB rect during EndDraw.
	numPx := physW * physH
	if len(pix) < numPx*4 {
		return false
	}
	if cap(c.d2dAlphaScratch) < numPx {
		c.d2dAlphaScratch = make([]byte, numPx)
	} else {
		c.d2dAlphaScratch = c.d2dAlphaScratch[:numPx]
	}
	for i := 0; i < numPx; i++ {
		c.d2dAlphaScratch[i] = pix[i*4+3]
	}

	bindRect := windows.Rect{Left: 0, Top: 0, Right: int32(physW), Bottom: int32(physH)}
	hr := comVtblCall(c.d2dDCRT, 57, c.memDC, uintptr(unsafe.Pointer(&bindRect))) // BindDC
	if int32(hr) < 0 {
		return false
	}

	if isScrolling {
		comVtblCall(c.d2dDCRT, 34, d2dTextAAModeGrayscale)
	} else {
		comVtblCall(c.d2dDCRT, 34, d2dTextAAModeClearType)
	}
	comVtblCall(c.d2dDCRT, 48) // BeginDraw
	defer func() {
		var t1, t2 uint64
		endHr := comVtblCall(c.d2dDCRT, 49, uintptr(unsafe.Pointer(&t1)), uintptr(unsafe.Pointer(&t2))) // EndDraw
		// Restore alpha mask for UpdateLayeredWindow (ClearType RGB stays intact)
		for i := 0; i < numPx; i++ {
			if a := c.d2dAlphaScratch[i]; a != 0 {
				pix[i*4+3] = a
			}
		}
		// D2DERR_RECREATE_TARGET = 0x8899000C
		if uint32(endHr) == 0x8899000C {
			c.destroyD2DText()
		}
	}()

	f155Bold := c.getDWriteTextFormat(2, 15.5*u, dwTextAlignLeading, dwParaAlignNear, false, true)
	f15Semi := c.getDWriteTextFormat(1, 15.0*u, dwTextAlignLeading, dwParaAlignNear, false, true)
	f145Bold := c.getDWriteTextFormat(2, 14.5*u, dwTextAlignLeading, dwParaAlignNear, false, true)
	f135Semi := c.getDWriteTextFormat(1, 13.5*u, dwTextAlignLeading, dwParaAlignNear, false, true)
	f135Bold := c.getDWriteTextFormat(2, 13.5*u, dwTextAlignLeading, dwParaAlignNear, false, true)
	f135SemiCenter := c.getDWriteTextFormat(1, 13.5*u, dwTextAlignCenter, dwParaAlignNear, false, true)
	f15SemiCenter := c.getDWriteTextFormat(1, 15.0*u, dwTextAlignCenter, dwParaAlignNear, false, true)
	f115Reg := c.getDWriteTextFormat(0, 11.5*u, dwTextAlignLeading, dwParaAlignNear, false, true)
	f11Reg := c.getDWriteTextFormat(0, 11.0*u, dwTextAlignLeading, dwParaAlignNear, false, true)
	fNoTrim155Bold := c.getDWriteTextFormat(2, 15.5*u, dwTextAlignLeading, dwParaAlignNear, false, false)
	fNoTrim145Bold := c.getDWriteTextFormat(2, 14.5*u, dwTextAlignLeading, dwParaAlignNear, false, false)
	fNoTrim115Reg := c.getDWriteTextFormat(0, 11.5*u, dwTextAlignLeading, dwParaAlignNear, false, false)
	fNoTrim11Reg := c.getDWriteTextFormat(0, 11.0*u, dwTextAlignLeading, dwParaAlignNear, false, false)

	drawMarquee := func(txt string, tfEllipsis, tfFull uintptr, col color.RGBA, colAlpha float64, px, py, pw, ph float64) {
		if txt == "" || tfFull == 0 {
			return
		}
		textW, _ := c.measureDWriteText(txt, tfFull, 4096.0*u, ph+8.0*u)
		if textW <= pw+1.0*u {
			c.drawD2DStr(txt, tfEllipsis, col, colAlpha, px, py, pw, ph, false)
			return
		}

		overflowPx := textW - pw + 4.0*u
		speedPx := 28.0 * u
		scrollDur := overflowPx / speedPx
		const (
			pauseStart = 2.0
			pauseEnd   = 1.5
			fadeOutDur = 0.35
			fadeInDur  = 0.35
		)
		cycleDur := pauseStart + scrollDur + pauseEnd + fadeOutDur + fadeInDur
		tCycle := math.Mod(marqueeElapsed, cycleDur)

		offPx := 0.0
		fadeMult := 1.0
		if tCycle < pauseStart {
			offPx = 0.0
		} else if tCycle < pauseStart+scrollDur {
			p := (tCycle - pauseStart) / scrollDur
			offPx = overflowPx * p
		} else if tCycle < pauseStart+scrollDur+pauseEnd {
			offPx = overflowPx
		} else if tCycle < pauseStart+scrollDur+pauseEnd+fadeOutDur {
			offPx = overflowPx
			p := (tCycle - pauseStart - scrollDur - pauseEnd) / fadeOutDur
			fadeMult = 1.0 - nusaCssEase(p)
		} else {
			offPx = 0.0
			p := (tCycle - pauseStart - scrollDur - pauseEnd - fadeOutDur) / fadeInDur
			fadeMult = nusaCssEase(p)
		}

		c.pushD2DClip(px, py, pw, ph)
		c.drawD2DStr(txt, tfFull, col, colAlpha*fadeMult, px-offPx, py, textW+12.0*u, ph, true)
		c.popD2DClip()
	}

	timeStr := getWindowsFormattedTime()

	switch mode {
	case diModeResting, diModeClosed:
		miniPx := maxInt(21, int(math.Round(20.8*u)))
		topCov := int(math.Round((h/2.0)*u - float64(miniPx)/2.0))
		cyExact := float64(topCov) + float64(miniPx)/2.0
		if showLyric && playing && lyricsOpen && len(lyrics) > 0 {
			txtHeightPx := float64(miniPx) * 0.72
			fClosed := c.getDWriteTextFormat(1, txtHeightPx, dwTextAlignCenter, dwParaAlignCenter, false, true)
			c.pushD2DClip((nx+35.0)*u, 0, (w-70.0)*u, h*u)
			maxBoxW := (560.0 - 80.0) * u
			boxH := txtHeightPx * 2.0
			if closedLyricProg < 0.99 && closedLyricFrom != "" && closedLyricFrom != closedLyricTo {
				outOp := contentOp * (1.0 - closedLyricProg)
				outY := cyExact - 3.0*u*closedLyricProg
				if outOp > 0.02 {
					c.drawD2DStr(closedLyricFrom, fClosed, pal.TextMain, outOp, (nx+w/2.0)*u-maxBoxW/2.0, outY-boxH/2.0, maxBoxW, boxH, true)
				}
			}
			inTxt := closedLyricTo
			if inTxt == "" {
				act := c.currentLyricIndexLocked()
				if act >= 0 && act < len(lyrics) {
					inTxt = strings.TrimSpace(lyrics[act].Text)
				}
			}
			if inTxt != "" {
				inOp := contentOp
				inY := cyExact
				if closedLyricProg < 0.99 && closedLyricFrom != "" && closedLyricFrom != closedLyricTo {
					inOp = contentOp * closedLyricProg
					inY = cyExact + 3.0*u*(1.0-closedLyricProg)
				}
				c.drawD2DStr(inTxt, fClosed, pal.TextMain, inOp, (nx+w/2.0)*u-maxBoxW/2.0, inY-boxH/2.0, maxBoxW, boxH, true)
			}
			c.popD2DClip()
		} else {
			txtH := float64(miniPx) * 0.80
			fClock := c.getDWriteTextFormat(1, txtH, dwTextAlignCenter, dwParaAlignCenter, false, false)
			maxBoxW := (w - 80.0) * u
			boxH := txtH * 2.0
			c.drawD2DStr(timeStr, fClock, pal.TextMain, contentOp, (nx+w/2.0)*u-maxBoxW/2.0, cyExact-boxH/2.0, maxBoxW, boxH, false)
		}

	case diModeHeadphones:
		c.drawD2DStr(timeStr, f135Semi, pal.TextMain, contentOp, mapXStatic(22), mapY(8), mapW(140), 22*u, false)

		outputs := audioOutputs
		if len(outputs) == 0 {
			bat := -1
			if isBt && btBat >= 0 && btBat <= 100 {
				bat = btBat
			}
			outputs = []windowsAudioEndpoint{{
				Name:     audioDev,
				ConnType: connType,
				IsActive: true,
				BatPct:   bat,
			}}
		}
		activeDev := outputs[0]
		for _, d := range outputs {
			if d.IsActive {
				activeDev = d
				break
			}
		}
		c.drawD2DStr(activeDev.Name, f15Semi, pal.TextMain, contentOp, mapX(68), mapY(37), mapW(290), 24*u, false)
		subTxt := diStr.Connected
		if activeDev.BatPct >= 0 && activeDev.BatPct <= 100 {
			subTxt = fmt.Sprintf("%s · %d%%", diStr.Connected, activeDev.BatPct)
		}
		c.drawD2DStr(subTxt, f115Reg, pal.TextVariant, 0.85*contentOp, mapX(68), mapY(60), mapW(290), 18*u, false)

		others := diOtherAudioIndices(outputs)
		if hpListProg > 0.01 && len(others) > 0 {
			c.pushD2DClip(mapX(14), mapY(96), mapW(392), math.Max(0, mapH(h-114)))
			for idx, origIdx := range others {
				dev := outputs[origIdx]
				rowDelay := float64(idx) * 0.12
				rowProg := clamp01((hpListProg - rowDelay) / (1.0 - rowDelay))
				rowEase := nusaCssEase(rowProg)
				if rowEase <= 0.01 {
					continue
				}
				slideY := (1.0 - rowEase) * 10.0
				rowOp := contentOp * rowEase
				cy := 68.0 + float64(idx+1)*56.0 + slideY

				c.drawD2DStr(dev.Name, f15Semi, pal.TextVariant, 0.75*rowOp, mapX(68), mapY(cy-21), mapW(290), 24*u, true)
				subRow := diStr.Available
				if dev.BatPct >= 0 && dev.BatPct <= 100 {
					subRow = fmt.Sprintf("%s · %d%%", diStr.Available, dev.BatPct)
				}
				c.drawD2DStr(subRow, f115Reg, pal.TextVariant, 0.52*rowOp, mapX(68), mapY(cy+2), mapW(290), 18*u, true)
			}
			c.popD2DClip()
		}

	case diModeMusic:
		if showStatus {
			c.drawD2DStr(timeStr, f135Semi, pal.TextMain, contentOp, mapXStatic(22), mapY(8), mapW(140), 22*u, false)
		}
		mTextX := 78.0
		mTextW := 238.0
		if !showCover {
			mTextX = 22.0
			mTextW = 294.0
		}
		drawMarquee(title, f155Bold, fNoTrim155Bold, pal.TextMain, contentOp, mapX(mTextX), mapY(33), mapW(mTextW), 24*u)
		if showArtist {
			drawMarquee(artist, f115Reg, fNoTrim115Reg, pal.TextVariant, 0.85*contentOp, mapX(mTextX), mapY(56), mapW(mTextW), 18*u)
		}

	case diModeLyrics:
		if showStatus {
			c.drawD2DStr(timeStr, f135Semi, pal.TextMain, contentOp, mapXStatic(22), mapY(8), mapW(140), 22*u, false)
		}
		lTextX := 76.0
		lTextW := 164.0
		if !showCover {
			lTextX = 22.0
			lTextW = 218.0
		}
		drawMarquee(title, f145Bold, fNoTrim145Bold, pal.TextMain, contentOp, mapX(lTextX), mapY(33), mapW(lTextW), 22*u)
		if showArtist {
			drawMarquee(artist, f11Reg, fNoTrim11Reg, pal.TextVariant, 0.85*contentOp, mapX(lTextX), mapY(56), mapW(lTextW), 18*u)
		}

		onPrimCont := color.RGBA{R: 0xD8, G: 0xF3, B: 0xFF, A: 255}
		if len(lyrics) == 0 {
			msgTitle := diStr.NoLyrics
			if lyricsLoading {
				msgTitle = diStr.LoadingLyrics
			}
			c.drawD2DStr(msgTitle, f135Semi, onPrimCont, 0.78*contentOp, mapX(262), mapY(82), mapW(230), 22*u, false)
			return true
		}

		fLyrWrap := c.getDWriteTextFormat(2, 13.8*u, dwTextAlignLeading, dwParaAlignNear, true, false)
		wrapW := mapW(234)
		lineTops := make([]float64, len(lyrics))
		lineHeights := make([]float64, len(lyrics))
		curTopDp := 0.0
		for i, line := range lyrics {
			hDp := 22.0
			if fLyrWrap != 0 && line.Text != "" {
				_, mh := c.measureDWriteText(line.Text, fLyrWrap, wrapW, 400.0*u)
				if mh > 1.0 {
					hDp = mh/u + 2.0
				}
			}
			lineTops[i] = curTopDp
			lineHeights[i] = hDp
			curTopDp += hDp + 6.0
		}

		activeTargetDp := 0.0
		if lyricIdx >= 0 && lyricIdx < len(lyrics) {
			activeTargetDp = lineTops[lyricIdx] + lineHeights[lyricIdx]/2.0
		} else if len(lyrics) > 0 {
			activeTargetDp = lineHeights[0] / 2.0
		}
		scrollCenterDp := activeTargetDp
		if prevLyricIdx >= 0 && prevLyricIdx < len(lyrics) && prevLyricIdx != lyricIdx && lyricAnimProg < 0.99 {
			prevCenter := lineTops[prevLyricIdx] + lineHeights[prevLyricIdx]/2.0
			scrollCenterDp = prevCenter + (activeTargetDp-prevCenter)*lyricAnimProg
		}
		maxScrollOff := math.Max(0.0, curTopDp-activeTargetDp)
		minScrollOff := -activeTargetDp
		clampedManual := math.Max(minScrollOff, math.Min(maxScrollOff, lyricManualOffsetDp))
		scrollCenterDp += clampedManual

		c.pushD2DClip(mapX(256), mapY(39), mapW(242), mapH(106.5))
		for i, line := range lyrics {
			yTop := 92.0 + lineTops[i] - scrollCenterDp
			yBot := yTop + lineHeights[i]
			if yBot < 38 || yTop > 148 {
				continue
			}
			lineCenterY := (yTop + yBot) / 2.0
			relMask := (lineCenterY - 39.0) / 106.5
			maskAlpha := 1.0
			if relMask < 0 {
				maskAlpha = 0.25
			} else if relMask < 0.14 {
				maskAlpha = 0.25 + 0.75*(relMask/0.14)
			} else if relMask > 1.0 {
				maskAlpha = 0.25
			} else if relMask > 0.86 {
				maskAlpha = 0.25 + 0.75*((1.0-relMask)/0.14)
			}

			opFactor := 0.35
			if i == lyricIdx {
				if prevLyricIdx >= 0 && prevLyricIdx != lyricIdx && lyricAnimProg < 0.99 {
					opFactor = 0.35 + 0.65*lyricAnimProg
				} else {
					opFactor = 1.0
				}
			} else if i == prevLyricIdx && prevLyricIdx != lyricIdx && lyricAnimProg < 0.99 {
				opFactor = 1.0 - 0.65*lyricAnimProg
			}
			if i == hoverLyric && opFactor < 0.95 {
				opFactor = 0.95
			}

			c.drawD2DStr(line.Text, fLyrWrap, onPrimCont, opFactor*contentOp*maskAlpha, mapX(260), mapY(yTop), wrapW, lineHeights[i]*u+4*u, true)
		}
		c.popD2DClip()

	case diModeQueue:
		c.drawD2DStr(timeStr, f135Semi, pal.TextMain, contentOp, mapXStatic(22), mapY(8), mapW(140), 22*u, false)

		hdrTitle := qTitle
		if hdrTitle == "" || strings.EqualFold(hdrTitle, "Queue") || strings.EqualFold(hdrTitle, "Antrean") {
			hdrTitle = diStr.QueueTitle
		}
		c.drawD2DStr(hdrTitle, f15SemiCenter, pal.TextMain, contentOp, mapX(72), mapY(42), mapW(276), 24*u, false)

		if len(qItems) == 0 {
			c.drawD2DStr(diStr.EmptyQueue, f135SemiCenter, pal.TextVariant, 0.85*contentOp, mapX(40), mapY(136), mapW(340), 22*u, false)
			return true
		}

		const qViewTop = 78.0
		const qViewBot = 242.0
		c.pushD2DClip(mapX(14), mapY(qViewTop), mapW(382), mapH(qViewBot-qViewTop))
		firstIdx := maxInt(0, int(math.Floor(qScrollDp/54.0))-1)
		lastIdx := minInt(len(qItems)-1, int(math.Ceil((qScrollDp+164.0)/54.0)))
		for itemIdx := firstIdx; itemIdx <= lastIdx; itemIdx++ {
			item := qItems[itemIdx]
			cy := 104.0 + float64(itemIdx)*54.0 - qScrollDp
			titleCol := pal.TextMain
			if itemIdx == qIndex {
				titleCol = pal.Accent
			}
			c.drawD2DStr(item.Title, f135Bold, titleCol, contentOp, mapX(74), mapY(cy-18), mapW(278), 22*u, true)
			drawMarquee(item.Artist, f11Reg, fNoTrim11Reg, pal.TextVariant, 0.85*contentOp, mapX(74), mapY(cy+2), mapW(278), 18*u)
		}
		c.popD2DClip()
	}

	return true
}
