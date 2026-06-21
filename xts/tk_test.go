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

	must(t, d.FillRects(gc, nil), "Drawable.FillRects(empty)") // no-op, no round-trip
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

// TestTkGC exercises the tk GC resource wrapper.
func TestTkGC(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	gc, err := tk.CreateGC1(c.DefaultWhitePixel(), c.DefaultBlackPixel(), 0)
	must(t, err, "TkConn.CreateGC1")
	must(t, gc.SetForeground(c.DefaultBlackPixel()), "GC.SetForeground")
	must(t, gc.SetBackground(c.DefaultWhitePixel()), "GC.SetBackground")
	must(t, gc.Change(&request.ChangeGCRequest{
		ValueMask: request.GC_MASK_LINE_WIDTH, LineWidth: 2,
	}), "GC.Change")

	// usable for drawing
	pm, err := tk.CreatePixmap(screen(c).RootDepth, c.DefaultRoot(), 32, 32)
	must(t, err, "TkConn.CreatePixmap")
	must(t, pm.FillRect(gc.XID, 0, 0, 16, 16), "Pixmap.FillRect")

	must(t, pm.Free(), "Pixmap.Free")
	must(t, gc.Free(), "GC.Free")
}

// TestTkPixmap exercises the tk Pixmap wrapper and its embedded Drawable methods.
func TestTkPixmap(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	depth := screen(c).RootDepth

	src, err := tk.CreatePixmap(depth, c.DefaultRoot(), 40, 40)
	must(t, err, "CreatePixmap src")
	dst, err := tk.CreatePixmap(depth, c.DefaultRoot(), 40, 40)
	must(t, err, "CreatePixmap dst")
	gc := newGC(t, c)

	// drawing + copy via the embedded Drawable
	must(t, src.FillRect(gc, 0, 0, 40, 40), "Pixmap.FillRect")
	must(t, src.CopyArea(dst.XID, gc, 0, 0, 0, 0, 40, 40), "Pixmap.CopyArea")

	g, err := src.GetGeometry()
	must(t, err, "Pixmap.GetGeometry")
	if g.Width != 40 || g.Height != 40 {
		t.Errorf("pixmap geometry = %dx%d, want 40x40", g.Width, g.Height)
	}

	must(t, rpc.FreeGC(c, gc), "FreeGC")
	must(t, src.Free(), "src.Free")
	must(t, dst.Free(), "dst.Free")
}

// TestTkInternAtom verifies TkConn.InternAtom resolves and caches atoms.
func TestTkInternAtom(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	a1, err := tk.InternAtom("WM_NAME")
	must(t, err, "InternAtom WM_NAME")
	if a1 == 0 {
		t.Fatal("WM_NAME interned to 0")
	}
	a2, err := tk.InternAtom("WM_NAME") // cached
	must(t, err, "InternAtom WM_NAME (cached)")
	if a1 != a2 {
		t.Errorf("cached atom = %d, want %d", a2, a1)
	}
	name, err := rpc.GetAtomName(c, a1)
	must(t, err, "GetAtomName")
	if name != "WM_NAME" {
		t.Errorf("GetAtomName = %q, want WM_NAME", name)
	}
}

// TestTkSetBackgroundPixmap exercises Window.SetBackgroundPixmap.
func TestTkSetBackgroundPixmap(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	w := tk_core.Window{
		Drawable: tk_core.Drawable{Conn: &tk}, ParentXID: c.DefaultRoot(),
		W: 64, H: 64, SetBackPixel: true, BackPixel: c.DefaultBlackPixel(),
	}
	must(t, w.Create(), "Window.Create")
	pm, err := tk.CreatePixmap(screen(c).RootDepth, c.DefaultRoot(), 64, 64)
	must(t, err, "CreatePixmap")
	gc := newGC(t, c)
	must(t, pm.FillRect(gc, 0, 0, 64, 64), "Pixmap.FillRect")

	must(t, w.SetBackgroundPixmap(pm.XID), "Window.SetBackgroundPixmap")
	must(t, w.Map(), "Window.Map")
	must(t, w.ClearArea(0, 0, 0, 0, false), "Window.ClearArea")

	must(t, rpc.FreeGC(c, gc), "FreeGC")
	must(t, pm.Free(), "Pixmap.Free")
	must(t, w.Destroy(), "Window.Destroy")
}
