package core

// simple Window object
//
// note: if you derive from it / wanna receive window messages,
// you need to install a WinHandler

import (
	"fmt"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/request"
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
	// used (white background), so callers no longer need a follow-up
	// ChangeWindowAttributes just to set these. BorderWidth defaults to 0.
	BorderWidth    base.CARD16
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
		BorderWidth:    uint16(w.BorderWidth),
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

func (w Window) Destroy() error {
	return rpc.DestroyWindow(w.Conn.X11Conn, w.XID)
}

func (w Window) ClearArea(x, y base.INT16, width, height base.CARD16, exposures bool) error {
	return rpc.ClearArea(w.Conn.X11Conn, w.XID, x, y, width, height, exposures)
}

func (w Window) MapSubwindows() error {
	return rpc.MapSubwindows(w.Conn.X11Conn, w.XID)
}

func (w Window) UnmapSubwindows() error {
	return rpc.UnmapSubwindows(w.Conn.X11Conn, w.XID)
}

func (w Window) Reparent(parent base.WINDOW, x, y base.INT16) error {
	return rpc.ReparentWindow(w.Conn.X11Conn, w.XID, parent, x, y)
}

func (w Window) CirculateUp() error {
	return rpc.CirculateWindow(w.Conn.X11Conn, request.CirculateRaiseLowest, w.XID)
}

func (w Window) CirculateDown() error {
	return rpc.CirculateWindow(w.Conn.X11Conn, request.CirculateLowerHighest, w.XID)
}

func (w Window) GetAttributes() (*request.GetWindowAttributesReply, error) {
	return rpc.GetWindowAttributes(w.Conn.X11Conn, w.XID)
}

func (w Window) QueryTree() (*request.QueryTreeReply, error) {
	return rpc.QueryTree(w.Conn.X11Conn, w.XID)
}

// ChangeAttributes applies the given attribute changes; Window is filled in.
func (w Window) ChangeAttributes(req *request.ChangeWindowAttributesRequest) error {
	req.Window = w.XID
	return rpc.ChangeWindowAttributes(w.Conn.X11Conn, req)
}

// Configure applies the given geometry/stacking changes; Window is filled in.
func (w Window) Configure(req *request.ConfigureWindowRequest) error {
	req.Window = w.XID
	return rpc.ConfigureWindow(w.Conn.X11Conn, req)
}

func (w Window) Move(x, y base.INT16) error {
	return w.Configure(&request.ConfigureWindowRequest{
		ValueMask: request.CONFIG_WINDOW_X | request.CONFIG_WINDOW_Y, X: x, Y: y,
	})
}

func (w Window) Resize(width, height base.CARD16) error {
	return w.Configure(&request.ConfigureWindowRequest{
		ValueMask: request.CONFIG_WINDOW_WIDTH | request.CONFIG_WINDOW_HEIGHT, Width: width, Height: height,
	})
}

func (w Window) MoveResize(x, y base.INT16, width, height base.CARD16) error {
	return w.Configure(&request.ConfigureWindowRequest{
		ValueMask: request.CONFIG_WINDOW_X | request.CONFIG_WINDOW_Y |
			request.CONFIG_WINDOW_WIDTH | request.CONFIG_WINDOW_HEIGHT,
		X: x, Y: y, Width: width, Height: height,
	})
}

func (w Window) Raise() error {
	return w.Configure(&request.ConfigureWindowRequest{
		ValueMask: request.CONFIG_WINDOW_STACK_MODE, StackMode: request.StackModeAbove,
	})
}

func (w Window) Lower() error {
	return w.Configure(&request.ConfigureWindowRequest{
		ValueMask: request.CONFIG_WINDOW_STACK_MODE, StackMode: request.StackModeBelow,
	})
}
