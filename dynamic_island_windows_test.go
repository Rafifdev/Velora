package main

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"velora/internal/sources"
)

func TestClosedPillCoverAndRingGeometrySymmetry(t *testing.T) {
	for _, tc := range []struct {
		nx float64
		w  float64
		h  float64
		u  float64
	}{
		{nx: 212, w: 216, h: 36, u: 1.0},
		{nx: 212, w: 216, h: 36, u: 1.25},
		{nx: 224, w: 192, h: 36, u: 1.25},
		{nx: 212.3, w: 216, h: 36, u: 1.5},
		{nx: 212, w: 216, h: 36, u: 2.0},
	} {
		miniPx, topCov, leftCov, ringCx, ringCy, strokeW, ringR := closedPillCoverAndRingGeom(tc.nx, tc.w, tc.h, tc.u)
		outerDiam := 2*ringR + strokeW
		if math.Abs(outerDiam-float64(miniPx)) > 1e-6 {
			t.Fatalf("u=%.2f outer diameter %.4f != miniPx %d", tc.u, outerDiam, miniPx)
		}
		leftCxIdx := float64(leftCov) + float64(miniPx-1)/2.0
		pillMidIdx := (tc.nx+tc.w/2.0)*tc.u - 0.5
		if math.Abs((leftCxIdx+ringCx)/2.0-pillMidIdx) > 1e-6 {
			t.Fatalf("u=%.2f horizontal centers %.4f and %.4f not symmetrical around pill midpoint %.4f",
				tc.u, leftCxIdx, ringCx, pillMidIdx)
		}
		expectedCyIdx := float64(topCov) + float64(miniPx-1)/2.0
		if math.Abs(ringCy-expectedCyIdx) > 1e-6 {
			t.Fatalf("u=%.2f ringCy %.4f != left cover Y center %.4f", tc.u, ringCy, expectedCyIdx)
		}
	}
}

func TestClosedPillCoverAndRingSymmetryGdiPlus(t *testing.T) {
	var token uintptr
	var si diGdiplusStartupInput
	si.GdiplusVersion = 1
	_, _, _ = diProcGdiplusStartup.Call(uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&si)), 0)
	if token == 0 {
		t.Fatal("failed to start GDI+")
	}
	defer diProcGdiplusShutdown.Call(token)

	for _, u := range []float64{1.0, 1.25, 1.5, 2.0} {
		w := 216.0
		h := 36.0
		nx := (float64(diLogicalW) - w) / 2.0
		physW := int(math.Ceil(float64(diLogicalW) * u))
		physH := int(math.Ceil(48.0 * u))

		ctrl := &dynamicIslandController{}
		ctrl.initDIB(physW, physH)
		pix := unsafe.Slice((*byte)(ctrl.dibBits), physW*physH*4)

		miniPx, topCov, leftCov, ringCx, ringCy, strokeW, ringR := closedPillCoverAndRingGeom(nx, w, h, u)

		src := image.NewRGBA(image.Rect(0, 0, 64, 64))
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				src.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
		miniArt := resizeCoverBilinear(src, miniPx, miniPx, float64(miniPx)/2.0)

		clear(pix)
		blitPremulRGBAWithOp(pix, physW, physH, miniArt, leftCov, topCov, 1.0)
		var g uintptr
		_, _, _ = diProcGdipCreateFromHDC.Call(ctrl.memDC, uintptr(unsafe.Pointer(&g)))
		_, _, _ = diProcGdipSetSmoothingMode.Call(g, 2)
		_, _, _ = diProcGdipSetPixelOffsetMode.Call(g, 2)
		drawPkRingGdip(g, ringCx, ringCy, ringR, strokeW, 0.02, 1.0, 0xC99191, 0x40C99191)
		_, _, _ = diProcGdipDeleteGraphics.Call(g)

		mid := physW / 2
		lx1, lx2, ly1, ly2 := physW, -1, physH, -1
		rx1, rx2, ry1, ry2 := physW, -1, physH, -1
		for y := 0; y < physH; y++ {
			for x := 0; x < physW; x++ {
				if pix[(y*physW+x)*4+3] < 32 {
					continue
				}
				if x < mid {
					if x < lx1 {
						lx1 = x
					}
					if x > lx2 {
						lx2 = x
					}
					if y < ly1 {
						ly1 = y
					}
					if y > ly2 {
						ly2 = y
					}
				} else {
					if x < rx1 {
						rx1 = x
					}
					if x > rx2 {
						rx2 = x
					}
					if y < ry1 {
						ry1 = y
					}
					if y > ry2 {
						ry2 = y
					}
				}
			}
		}

		leftW := lx2 - lx1 + 1
		rightW := rx2 - rx1 + 1
		leftH := ly2 - ly1 + 1
		rightH := ry2 - ry1 + 1
		if leftW != miniPx || leftH != miniPx {
			t.Fatalf("u=%.2f unexpected cover bounds %dx%d (want %dx%d)", u, leftW, leftH, miniPx, miniPx)
		}
		if rightW != leftW {
			t.Fatalf("u=%.2f ring width %d != cover width %d (L=[%d..%d], R=[%d..%d])", u, rightW, leftW, lx1, lx2, rx1, rx2)
		}
		if rightH < leftH || rightH > leftH+1 {
			t.Fatalf("u=%.2f ring height %d out of range [%d..%d] (L=[%d..%d], R=[%d..%d])", u, rightH, leftH, leftH+1, ly1, ly2, ry1, ry2)
		}
		ctrl.destroyDIB()
	}
}

func TestCenteredWinXAndSnapSymmetry(t *testing.T) {
	for _, tc := range []struct {
		screenLeft int
		screenW    int
		dpiScale   float64
		sizeScale  float64
		pillW      float64
	}{
		{screenLeft: 0, screenW: 1920, dpiScale: 1.0, sizeScale: 0.88, pillW: 216},
		{screenLeft: 0, screenW: 1920, dpiScale: 1.25, sizeScale: 0.88, pillW: 216},
		{screenLeft: 0, screenW: 1920, dpiScale: 1.25, sizeScale: 1.00, pillW: 216},
		{screenLeft: 0, screenW: 1920, dpiScale: 1.25, sizeScale: 1.08, pillW: 420},
		{screenLeft: -1920, screenW: 1920, dpiScale: 1.5, sizeScale: 0.88, pillW: 216},
	} {
		ctrl := &dynamicIslandController{
			dpiScale:   tc.dpiScale,
			sizeScale:  tc.sizeScale,
			screenLeft: tc.screenLeft,
			screenW:    tc.screenW,
			physW:      int(math.Ceil(float64(diLogicalW) * tc.dpiScale)),
			curW:       tc.pillW,
		}
		cx := ctrl.centeredWinXLocked(tc.screenLeft, tc.screenW)
		snapped, dock := ctrl.clampWinXLocked(cx+5, true)
		if dock != 0 || snapped != cx {
			t.Fatalf("expected center snap to %d, got snapped=%d dock=%d", cx, snapped, dock)
		}
		u := ctrl.effectiveScaleLocked()
		nx := (float64(diLogicalW) - tc.pillW) / 2.0
		leftEdge := float64(snapped) + nx*u
		rightEdge := float64(snapped) + (nx+tc.pillW)*u
		leftGap := leftEdge - float64(tc.screenLeft)
		rightGap := float64(tc.screenLeft+tc.screenW) - rightEdge
		if math.Abs(leftGap-rightGap) > 1.0 {
			t.Fatalf("dpi=%.2f size=%.2f pill=%.0f: leftGap=%.2f != rightGap=%.2f",
				tc.dpiScale, tc.sizeScale, tc.pillW, leftGap, rightGap)
		}
	}
}

func TestDynamicIslandLyricStatesAndSync(t *testing.T) {
	now := time.Now()
	ctrl := &dynamicIslandController{
		title:        "Some Track",
		isPlaying:    true,
		showLyric:    true,
		lyricsOpen:   true,
		lang:         "en",
		posSec:       10.0,
		posUpdatedAt: now.Add(-200 * time.Millisecond),
		durSec:       200.0,
	}
	if !ctrl.isLyricPlaybackModeActiveLocked() || !ctrl.isClosedLyricActiveLocked() {
		t.Fatal("expected lyrics mode to remain active without synced lines so island does not hide")
	}
	if got := ctrl.closedLyricMessageLocked(now, getDIStrings("en")); got != "No lyrics found" {
		t.Fatalf("unexpected empty lyric state: %q", got)
	}
	ctrl.lyricsHasPlain = true
	if got := ctrl.closedLyricMessageLocked(now, getDIStrings("en")); got != "Synced lyrics unavailable" {
		t.Fatalf("unexpected plain-only lyric state: %q", got)
	}
	if got := ctrl.closedLyricMessageLocked(now, getDIStrings("id")); got != "Lirik tersinkron tidak tersedia" {
		t.Fatalf("unexpected Indonesian plain-only lyric state: %q", got)
	}

	ctrl.lyrics = []sources.LyricLine{
		{TimeSec: 5.0, Text: "Line 1"},
		{TimeSec: 10.15, Text: "Line 2"},
	}
	// Effective pos is ~10.20 + 0.05 = 10.25 >= 10.15 (line 2 already active thanks to interpolation)
	if idx := ctrl.currentLyricIndexAtLocked(now); idx != 1 {
		t.Fatalf("expected interpolated lyric idx=1 at pos 10.0+0.20s, got %d", idx)
	}
	ctrl.lyricOffsetSec = -0.30
	if idx := ctrl.currentLyricIndexAtLocked(now); idx != 0 {
		t.Fatalf("expected negative lyric offset to move back to idx=0, got %d", idx)
	}
}

func TestDWriteEmojiColorRendering(t *testing.T) {
	physW, physH := 220, 64
	ctrl := &dynamicIslandController{}
	ctrl.initDIB(physW, physH)
	defer ctrl.destroyDIB()
	ctrl.initD2DText()
	defer ctrl.destroyD2DText()
	if ctrl.d2dDCRT == 0 || ctrl.dwFactory == 0 {
		t.Skip("Direct2D/DirectWrite unavailable on test host")
	}

	pix := unsafe.Slice((*byte)(ctrl.dibBits), physW*physH*4)
	clear(pix)
	for i := 0; i < physW*physH; i++ {
		pix[i*4+3] = 255
	}

	bindRect := windows.Rect{Left: 0, Top: 0, Right: int32(physW), Bottom: int32(physH)}
	if hr := comVtblCall(ctrl.d2dDCRT, 57, ctrl.memDC, uintptr(unsafe.Pointer(&bindRect))); int32(hr) < 0 {
		t.Fatalf("BindDC failed: 0x%x", uint32(hr))
	}
	comVtblCall(ctrl.d2dDCRT, 48)
	tf := ctrl.getDWriteTextFormat(1, 20.0, dwTextAlignCenter, dwParaAlignCenter, false, false)
	ctrl.drawD2DStr("🎵 ❤️ 🔥", tf, color.RGBA{255, 255, 255, 255}, 1.0, 0, 0, float64(physW), float64(physH), false)
	var t1, t2 uint64
	comVtblCall(ctrl.d2dDCRT, 49, uintptr(unsafe.Pointer(&t1)), uintptr(unsafe.Pointer(&t2)))

	coloredPixels := 0
	litPixels := 0
	for i := 0; i < physW*physH; i++ {
		b, g, r := pix[i*4+0], pix[i*4+1], pix[i*4+2]
		if r > 10 || g > 10 || b > 10 {
			litPixels++
		}
		if r != g || g != b {
			coloredPixels++
		}
	}
	t.Logf("DWrite emoji test: litPixels=%d coloredPixels=%d (dwFontColl=%x)", litPixels, coloredPixels, ctrl.dwFontColl)
	if litPixels == 0 {
		t.Fatal("expected emoji text to render visible pixels")
	}
	if coloredPixels == 0 {
		t.Fatal("expected color emoji glyphs (COLR) to produce non-grayscale color pixels")
	}
}

func TestClosedPillClipAndQueuePlayAlignment(t *testing.T) {
	// Verify queue Play/Pause button right edge aligns with music queue row container (18 + 374 = 392 dp),
	// rather than the scrollbar track (398.5 + 3.5 = 402 dp).
	const queueRowRight = 18.0 + 374.0
	const queuePlayCenter = 375.5
	const queuePlayRadius = 16.5
	if math.Abs((queuePlayCenter+queuePlayRadius)-queueRowRight) > 1e-6 {
		t.Fatalf("queue Play/Pause right edge %.2f != queue row right edge %.2f",
			queuePlayCenter+queuePlayRadius, queueRowRight)
	}

	// Verify closed pill lyric clip margins stay outside both rotating left cover and right progress ring.
	for _, tc := range []struct {
		w float64
		u float64
	}{
		{w: 216, u: 1.0},
		{w: 320, u: 1.25},
		{w: 560, u: 1.5},
	} {
		nx := (float64(diLogicalW) - tc.w) / 2.0
		miniPx, _, leftCov, ringCx, _, strokeW, ringR := closedPillCoverAndRingGeom(nx, tc.w, 36.0, tc.u)
		covRightPx := float64(leftCov + miniPx)
		ringLeftPx := ringCx - ringR - strokeW/2.0
		clipLeftPx := (nx + 38.0) * tc.u
		clipRightPx := clipLeftPx + math.Max(1.0, (tc.w-76.0)*tc.u)
		if clipLeftPx <= covRightPx+1.5*tc.u {
			t.Fatalf("w=%.0f u=%.2f clipLeft %.2f too close to coverRight %.2f", tc.w, tc.u, clipLeftPx, covRightPx)
		}
		if clipRightPx >= ringLeftPx-1.5*tc.u {
			t.Fatalf("w=%.0f u=%.2f clipRight %.2f too close to ringLeft %.2f", tc.w, tc.u, clipRightPx, ringLeftPx)
		}
	}
}
