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
	Drawable
	ParentXID base.WINDOW
	Parent    *Window
	Name      string
	X         base.INT16
	Y         base.INT16
	H         base.CARD16
	W         base.CARD16
	EventMask base.CARD32

	// Optional initial attributes. When the Set* flag is false the default is
	// used (white background, server-default border), so callers no longer need
	// a follow-up ChangeWindowAttributes just to set these.
	BackPixel      base.CARD32
	SetBackPixel   bool
	BorderPixel    base.CARD32
	SetBorderPixel bool

	// derived classes need to link themselves in here
	WinHandler WindowHandler
}

func (w *Window) Create() error {
	if !w.XID.Invalid() {
		base.MakeX11Error("foo")
		return fmt.Errorf("window already created XID %d\n", w.XID)
	}

	if w.ParentXID.Invalid() && w.Parent != nil {
		w.ParentXID = w.Parent.XID
	}

	if w.ParentXID.Invalid() {
		w.ParentXID = w.Conn.X11Conn.DefaultRoot()
	}

	spec := rpc.WindowSpec{
		Parent:         w.ParentXID,
		X:              int16(w.X),
		Y:              int16(w.Y),
		Width:          uint16(w.W),
		Height:         uint16(w.H),
		BorderWidth:    1, // preserve the previous default
		EventMask:      w.EventMask,
		SetBackPixel:   true,
		BackPixel:      w.Conn.X11Conn.DefaultWhitePixel(),
		SetBorderPixel: w.SetBorderPixel,
		BorderPixel:    w.BorderPixel,
	}
	if w.SetBackPixel {
		spec.BackPixel = w.BackPixel
	}
	xid, err := rpc.CreateWindow(w.Conn.X11Conn, spec)
	w.XID = xid

	if err != nil {
		return err
	}

	w.Drawable.Conn = w.Conn
	w.Drawable.XID = xid

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

func (w Window) Unmap() error {
	return rpc.UnmapWindow(w.Conn.X11Conn, w.XID)
}

func (w Window) SetName(n string) error {
	w.Name = n
	return rpc.SetWindowName(w.Conn.X11Conn, w.XID, n)
}
