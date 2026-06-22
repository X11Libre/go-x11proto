package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

func TestGCOps(t *testing.T) {
	c := connect(t)
	defer c.Close()
	gc := newGC(t, c)

	must(t, rpc.ChangeGC(c, &request.ChangeGCRequest{Gc: gc,
		ValueMask: request.GC_MASK_LINE_WIDTH | request.GC_MASK_FOREGROUND,
		LineWidth: 3, Foreground: c.DefaultWhitePixel()}), "ChangeGC")

	gc2 := newGC(t, c)
	must(t, rpc.CopyGC(c, gc, gc2, request.GC_MASK_LINE_WIDTH), "CopyGC")
	must(t, rpc.SetDashes(c, gc, 0, []base.CARD8{4, 2}), "SetDashes")
	must(t, rpc.SetClipRectangles(c, request.ClipOrderingUnsorted, gc, 0, 0,
		[]base.Rectangle{{X: 0, Y: 0, Width: 10, Height: 10}}), "SetClipRectangles")
	must(t, rpc.FreeGC(c, gc2), "FreeGC")
	must(t, rpc.FreeGC(c, gc), "FreeGC")
}

func TestDrawingPrimitives(t *testing.T) {
	c := connect(t)
	defer c.Close()
	pm := newPixmap(t, c, 100, 100)
	gc := newGC(t, c)

	must(t, rpc.PolyPoint(c, request.CoordModeOrigin, pm, gc, []base.Point{{X: 1, Y: 1}, {X: 2, Y: 2}}), "PolyPoint")
	must(t, rpc.PolyLine(c, request.CoordModeOrigin, pm, gc, []base.Point{{X: 0, Y: 0}, {X: 10, Y: 10}}), "PolyLine")
	must(t, rpc.PolySegment(c, pm, gc, []base.Segment{{X1: 0, Y1: 0, X2: 5, Y2: 5}}), "PolySegment")
	must(t, rpc.PolyRectangle(c, pm, gc, []base.Rectangle{{X: 1, Y: 1, Width: 8, Height: 8}}), "PolyRectangle")
	must(t, rpc.PolyArc(c, pm, gc, []base.Arc{{X: 0, Y: 0, Width: 10, Height: 10, Angle1: 0, Angle2: 360 * 64}}), "PolyArc")
	must(t, rpc.FillPoly(c, pm, gc, request.PolyShapeConvex, request.CoordModeOrigin,
		[]base.Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}}), "FillPoly")
	must(t, rpc.FillRect(c, pm, gc, 0, 0, 20, 20), "FillRect(PolyFillRectangle)")
	must(t, rpc.PolyFillArc(c, pm, gc, []base.Arc{{X: 0, Y: 0, Width: 10, Height: 10, Angle1: 0, Angle2: 180 * 64}}), "PolyFillArc")

	must(t, rpc.FreeGC(c, gc), "FreeGC")
	must(t, rpc.FreePixmap(c, pm), "FreePixmap")
}

func TestCopyAreaAndPlane(t *testing.T) {
	c := connect(t)
	defer c.Close()
	src := newPixmap(t, c, 50, 50)
	dst := newPixmap(t, c, 50, 50)
	gc := newGC(t, c)

	must(t, rpc.FillRect(c, src, gc, 0, 0, 50, 50), "FillRect")
	must(t, rpc.CopyArea(c, src, dst, gc, 0, 0, 0, 0, 50, 50), "CopyArea")
	must(t, rpc.CopyPlane(c, src, dst, gc, 0, 0, 0, 0, 50, 50, 1), "CopyPlane")

	must(t, rpc.FreeGC(c, gc), "FreeGC")
	must(t, rpc.FreePixmap(c, src), "FreePixmap src")
	must(t, rpc.FreePixmap(c, dst), "FreePixmap dst")
}

func TestPutGetImageRoundTrip(t *testing.T) {
	c := connect(t)
	defer c.Close()
	depth := screen(c).RootDepth
	if depth != 24 {
		t.Skipf("root depth %d not 24bpp; skipping pixel round-trip", depth)
	}
	const w, h = 2, 2
	pm := newPixmap(t, c, w, h)
	gc := newGC(t, c)

	// ZPixmap at depth 24 is 4 bytes/pixel on this server.
	in := []byte{
		0x11, 0x22, 0x33, 0x00, 0x44, 0x55, 0x66, 0x00,
		0x77, 0x88, 0x99, 0x00, 0xAA, 0xBB, 0xCC, 0x00,
	}
	must(t, rpc.PutImage(c, pm, gc, request.ImageFormatZPixmap, depth, w, h, 0, 0, in), "PutImage")

	out, err := rpc.GetImage(c, request.ImageFormatZPixmap, pm, 0, 0, w, h, 0xFFFFFFFF)
	must(t, err, "GetImage")
	if len(out.Data) != len(in) {
		t.Fatalf("GetImage data len = %d, want %d", len(out.Data), len(in))
	}
	// compare the three colour bytes of each pixel (4th is unused padding)
	for px := 0; px < w*h; px++ {
		for b := 0; b < 3; b++ {
			i := px*4 + b
			if out.Data[i] != in[i] {
				t.Errorf("pixel %d byte %d = %#x, want %#x", px, b, out.Data[i], in[i])
			}
		}
	}
	must(t, rpc.FreeGC(c, gc), "FreeGC")
	must(t, rpc.FreePixmap(c, pm), "FreePixmap")
}

func TestClearArea(t *testing.T) {
	c := connect(t)
	defer c.Close()
	w := createWin(t, c, request.CW_BACKGROUND_PIXEL,
		&request.CreateWindowRequest{Width: 100, Height: 100, BackPixel: c.DefaultWhitePixel()})
	must(t, rpc.MapWindow(c, w), "MapWindow")
	must(t, rpc.ClearArea(c, w, 10, 10, 20, 20, false), "ClearArea")
	must(t, rpc.DestroyWindow(c, w), "DestroyWindow")
}
