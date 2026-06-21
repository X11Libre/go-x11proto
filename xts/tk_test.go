package xts

import (
	"testing"

	tk_core "github.com/X11Libre/go-x11proto/tk/core"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// TestTkWindowOps exercises the tk Window operations against a live server.
func TestTkWindowOps(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	w := tk_core.Window{
		Drawable:     tk_core.Drawable{Conn: &tk},
		ParentXID:    c.DefaultRoot(),
		W:            100,
		H:            80,
		SetBackPixel: true,
		BackPixel:    c.DefaultBlackPixel(),
	}
	must(t, w.Create(), "Window.Create")
	must(t, w.Map(), "Window.Map")

	must(t, w.Move(10, 20), "Window.Move")
	must(t, w.Resize(120, 90), "Window.Resize")
	g, err := w.GetGeometry()
	must(t, err, "Window.GetGeometry")
	if g.X != 10 || g.Y != 20 || g.Width != 120 || g.Height != 90 {
		t.Errorf("geometry = (%d,%d %dx%d), want (10,20 120x90)", g.X, g.Y, g.Width, g.Height)
	}
	must(t, w.MoveResize(0, 0, 64, 64), "Window.MoveResize")

	a, err := w.GetAttributes()
	must(t, err, "Window.GetAttributes")
	if a.Class != 1 {
		t.Errorf("class = %d, want 1", a.Class)
	}
	tr, err := w.QueryTree()
	must(t, err, "Window.QueryTree")
	if tr.Parent != c.DefaultRoot() {
		t.Errorf("parent = %d, want root", tr.Parent)
	}

	must(t, w.Raise(), "Window.Raise")
	must(t, w.Lower(), "Window.Lower")
	must(t, w.ChangeAttributes(&request.ChangeWindowAttributesRequest{
		ValueMask: request.CW_BACKGROUND_PIXEL, BackPixel: c.DefaultWhitePixel(),
	}), "Window.ChangeAttributes")
	must(t, w.ClearArea(0, 0, 10, 10, false), "Window.ClearArea")

	// drawing on the window via the embedded Drawable
	gc := newGC(t, c)
	must(t, w.PolyLine(gc, request.CoordModeOrigin, []base.Point{{X: 0, Y: 0}, {X: 50, Y: 50}}), "Window.PolyLine")
	must(t, w.FillRect(gc, 0, 0, 20, 20), "Window.FillRect")
	must(t, rpc.FreeGC(c, gc), "FreeGC")

	must(t, w.Unmap(), "Window.Unmap")
	must(t, w.Destroy(), "Window.Destroy")
}

// TestTkDrawableOps exercises the tk Drawable operations on a pixmap.
func TestTkDrawableOps(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	pm := newPixmap(t, c, 64, 64)
	d := tk_core.Drawable{Conn: &tk, XID: pm}
	gc := newGC(t, c)

	must(t, d.PolyPoint(gc, request.CoordModeOrigin, []base.Point{{X: 1, Y: 1}}), "Drawable.PolyPoint")
	must(t, d.PolySegment(gc, []base.Segment{{X1: 0, Y1: 0, X2: 5, Y2: 5}}), "Drawable.PolySegment")
	must(t, d.PolyRectangle(gc, []base.Rectangle{{X: 0, Y: 0, Width: 10, Height: 10}}), "Drawable.PolyRectangle")
	must(t, d.PolyArc(gc, []base.Arc{{Width: 10, Height: 10, Angle2: 360 * 64}}), "Drawable.PolyArc")
	must(t, d.FillPoly(gc, request.PolyShapeConvex, request.CoordModeOrigin,
		[]base.Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 5, Y: 10}}), "Drawable.FillPoly")
	must(t, d.PolyFillArc(gc, []base.Arc{{Width: 10, Height: 10, Angle2: 180 * 64}}), "Drawable.PolyFillArc")

	dst := newPixmap(t, c, 64, 64)
	must(t, d.CopyArea(dst, gc, 0, 0, 0, 0, 32, 32), "Drawable.CopyArea")
	must(t, d.CopyPlane(dst, gc, 0, 0, 0, 0, 32, 32, 1), "Drawable.CopyPlane")

	g, err := d.GetGeometry()
	must(t, err, "Drawable.GetGeometry")
	if g.Width != 64 || g.Height != 64 {
		t.Errorf("pixmap geometry = %dx%d, want 64x64", g.Width, g.Height)
	}

	must(t, rpc.FreeGC(c, gc), "FreeGC")
	must(t, rpc.FreePixmap(c, dst), "FreePixmap dst")
	must(t, rpc.FreePixmap(c, pm), "FreePixmap")
}
