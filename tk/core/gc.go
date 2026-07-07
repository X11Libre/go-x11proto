package core

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// GC wraps a server-side graphics-context resource together with its
// connection, so it can be changed and freed without threading the raw
// X11Conn / XID around.
type GC struct {
	Conn *TkConn
	XID  base.GC
}

// CreateGC1 creates a simple graphics context with the given foreground and
// background pixel values and font (font may be 0 for none). The returned GC
// must be released with Free.
func (tkc *TkConn) CreateGC1(fg, bg base.CARD32, font base.FONT) (*GC, error) {
	xid, err := rpc.CreateGC1(tkc.X11Conn, fg, bg, font)
	if err != nil {
		return nil, err
	}
	return &GC{Conn: tkc, XID: xid}, nil
}

// CreateGCFor creates a graphics context for an arbitrary drawable. Needed
// whenever the GC draws into something other than a root-depth window or
// pixmap (e.g. an 8-bit alpha mask or a 32-bit ARGB pixmap for RENDER
// compositing) — a GC's depth must match its drawable's.
func (tkc *TkConn) CreateGCFor(drawable base.DRAWABLE, fg, bg base.CARD32, font base.FONT) (*GC, error) {
	xid, err := rpc.CreateGC(tkc.X11Conn, drawable, fg, bg, font)
	if err != nil {
		return nil, err
	}
	return &GC{Conn: tkc, XID: xid}, nil
}

// Free releases the graphics context.
func (g *GC) Free() error {
	return rpc.FreeGC(g.Conn.X11Conn, g.XID)
}

// Change applies arbitrary GC value changes; the Gc field is filled in.
func (g *GC) Change(req *request.ChangeGCRequest) error {
	req.Gc = g.XID
	return rpc.ChangeGC(g.Conn.X11Conn, req)
}

// SetForeground changes the foreground pixel value.
func (g *GC) SetForeground(fg base.CARD32) error {
	return g.Change(&request.ChangeGCRequest{ValueMask: request.GC_MASK_FOREGROUND, Foreground: fg})
}

// SetBackground changes the background pixel value.
func (g *GC) SetBackground(bg base.CARD32) error {
	return g.Change(&request.ChangeGCRequest{ValueMask: request.GC_MASK_BACKGROUND, Background: bg})
}

// SetFont sets the font used for text-drawing requests (ImageText8 / PutText8).
func (g *GC) SetFont(font base.FONT) error {
	return g.Change(&request.ChangeGCRequest{ValueMask: request.GC_MASK_FONT, Font: font})
}
