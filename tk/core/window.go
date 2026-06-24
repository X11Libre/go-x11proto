package core

// simple Window object
//
// note: if you derive from it / wanna receive window messages,
// you need to install a WinHandler

import (
	"fmt"
	"image"

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

// xaATOM / xaCARDINAL are predefined property types (XA_ATOM, XA_CARDINAL).
const (
	xaATOM     base.ATOM = 4
	xaCARDINAL base.ATOM = 6
)

// SetIconRGBA sets the window's icon via the EWMH _NET_WM_ICON property, which
// modern window managers and taskbars use. rgba is width*height pixels, 4 bytes
// each (R,G,B,A); the property stores them as 32-bit ARGB preceded by the size.
func (w Window) SetIconRGBA(width, height int, rgba []byte) error {
	if width <= 0 || height <= 0 || len(rgba) < width*height*4 {
		return fmt.Errorf("SetIconRGBA: bad dimensions or short data")
	}
	data := make([]base.CARD32, 0, 2+width*height)
	data = append(data, base.CARD32(width), base.CARD32(height))
	for i := 0; i < width*height; i++ {
		r := base.CARD32(rgba[i*4])
		g := base.CARD32(rgba[i*4+1])
		b := base.CARD32(rgba[i*4+2])
		a := base.CARD32(rgba[i*4+3])
		data = append(data, a<<24|r<<16|g<<8|b)
	}
	atom, err := w.Conn.InternAtom("_NET_WM_ICON")
	if err != nil {
		return err
	}
	return rpc.ChangeProperty32(w.Conn.X11Conn, 0 /*replace*/, w.XID, atom, xaCARDINAL, data)
}

// SetIcon sets the window icon from any image.Image.
func (w Window) SetIcon(img image.Image) error {
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	rgba := make([]byte, 0, width*height*4)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA() // 16-bit, premultiplied
			rgba = append(rgba, byte(r>>8), byte(g>>8), byte(bl>>8), byte(a>>8))
		}
	}
	return w.SetIconRGBA(width, height, rgba)
}

// EnableWMDelete advertises the WM_DELETE_WINDOW protocol on this (top-level)
// window, so the window manager sends a ClientMessage when the user closes it
// instead of killing the client connection (which would tear down the whole
// program). It returns the WM_DELETE_WINDOW atom; pass received events to
// IsWMDelete to recognise the close request and shut the window down gracefully.
func (w Window) EnableWMDelete() (base.ATOM, error) {
	proto, err := w.Conn.InternAtom("WM_PROTOCOLS")
	if err != nil {
		return 0, err
	}
	del, err := w.Conn.InternAtom("WM_DELETE_WINDOW")
	if err != nil {
		return 0, err
	}
	if err := rpc.ChangeProperty32(w.Conn.X11Conn, 0 /*replace*/, w.XID, proto, xaATOM,
		[]base.CARD32{base.CARD32(del)}); err != nil {
		return 0, err
	}
	return del, nil
}

// IsWMDelete reports whether ev is the WM_DELETE_WINDOW close request for the
// given atom (as returned by EnableWMDelete). It is false for del == 0.
func IsWMDelete(ev events.Event, del base.ATOM) bool {
	cm, ok := ev.(*events.ClientMessageEvent)
	return ok && del != 0 && base.ATOM(cm.Data[0]) == del
}

// ParentRelative is the special background-pixmap value that makes a window use
// its parent's background: the parent's background shows through wherever the
// window itself does not paint, giving a "transparent" overlay.
const ParentRelative base.PIXMAP = 1

// SetBackgroundPixmap sets the window's background to the given pixmap (or the
// special value ParentRelative). The change takes effect on the next repaint
// (e.g. after ClearArea).
func (w Window) SetBackgroundPixmap(pm base.PIXMAP) error {
	return rpc.SetWindowBackgroundPixmap(w.Conn.X11Conn, w.XID, pm)
}

// SetOverrideRedirect sets (or clears) the override-redirect attribute, which
// tells the window manager to leave the window alone (no reparenting/decoration)
// - used for popups, menus and tooltips. Call it before mapping.
func (w Window) SetOverrideRedirect(on bool) error {
	v := base.CARD32(0)
	if on {
		v = 1
	}
	return w.ChangeAttributes(&request.ChangeWindowAttributesRequest{
		ValueMask:        request.CW_OVERRIDE_REDIRECT,
		OverrideRedirect: v,
	})
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
