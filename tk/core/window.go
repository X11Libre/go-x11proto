package core

// simple Window object
//
// note: if you derive from it / wanna receive window messages,
// you need to install a WinHandler

import (
	"fmt"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

type WindowHandler interface {
	HandleWindowEvent(ev events.Event) bool
}

type Window struct {
	Conn      *TkConn
	XID       base.WINDOW
	ParentXID base.WINDOW
	Parent    *Window
	Name      string
	X         base.INT16
	Y         base.INT16
	H         base.CARD16
	W         base.CARD16
	EventMask base.CARD32

	// derived classes need to link themselves in here
	WinHandler WindowHandler
}

func (w *Window) Create() error {
	if w.XID != 0 {
		base.MakeX11Error("foo")
		return fmt.Errorf("window already created XID %d\n", w.XID)
	}

	if w.ParentXID == 0 && w.Parent != nil {
		w.ParentXID = w.Parent.XID
	}

	if w.ParentXID == 0 {
		w.ParentXID = w.Conn.X11Conn.DefaultRoot()
	}

	xid, err := rpc.CreateWindow1(w.Conn.X11Conn, w.ParentXID, int16(w.X), int16(w.Y), uint16(w.W), uint16(w.H), w.EventMask)
	w.XID = xid

	if err != nil {
		return err
	}

	w.Conn.X11Conn.RegisterWindowHandler(w.XID, w)

	if w.Name != "" {
		return w.SetName(w.Name)
	}

	return nil
}

func (w *Window) SetWindowHandler(h WindowHandler) {
	w.WinHandler = h
}

func (w *Window) HandleX11WindowEvent(window base.WINDOW, ev events.Event) bool {
	if w.WinHandler != nil {
		return w.WinHandler.HandleWindowEvent(ev)
	}
	return true
}

func (w Window) Map() error {
	return rpc.MapWindow(w.Conn.X11Conn, w.XID)
}

func (w Window) SetName(n string) error {
	w.Name = n
	return rpc.SetWindowName(w.Conn.X11Conn, w.XID, n)
}

func (w Window) DRAWABLE() base.DRAWABLE {
	return w.XID.Drawable()
}

func (w Window) FillRect(gc base.GC, x base.INT16, y base.INT16, width base.CARD16, height base.CARD16) error {
	return w.FillRects(
		gc,
		[]base.Rectangle{
			{X: x, Y: y, Width: width, Height: height},
		},
	)
}

func (w Window) FillRects(gc base.GC, rects []base.Rectangle) error {
	return rpc.FillRects(w.Conn.X11Conn, w.DRAWABLE(), gc, rects)
}

func (w Window) PutText8(gc base.GC, x base.INT16, y base.INT16, text string) error {
	return rpc.PutText8(w.Conn.X11Conn, w.DRAWABLE(), gc, x, y, text)
}
