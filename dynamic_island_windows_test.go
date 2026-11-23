package main

import (
	"image"
	"image/color"
	"math"
	"testing"
	"unsafe"
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
