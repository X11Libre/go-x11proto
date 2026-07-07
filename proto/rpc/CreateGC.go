package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CreateGC1(c *core.X11Conn, fg base.CARD32, bg base.CARD32, font base.FONT) (base.GC, error) {
	return CreateGC(c, c.Setup.Screens[0].RootWindow, fg, bg, font)
}

// CreateGC creates a graphics context for an arbitrary drawable. Unlike
// CreateGC1 (always against the root window), this is required whenever the
// GC will draw into a pixmap whose depth differs from the root window's — a
// GC's depth must match its drawable's, so e.g. PutImage into an 8-bit alpha
// mask or a 32-bit ARGB pixmap needs its own GC created against that pixmap.
func CreateGC(c *core.X11Conn, drawable base.DRAWABLE, fg base.CARD32, bg base.CARD32, font base.FONT) (base.GC, error) {
	gcid := base.GC(c.NextResourceID())
	req := request.CreateGCRequest{
		Gcid:       gcid,
		Drawable:   drawable,
		ValueMask:  request.GC_MASK_FOREGROUND | request.GC_MASK_BACKGROUND,
		Foreground: fg,
		Background: bg,
		Font:       font,
	}

	if !font.Invalid() {
		req.ValueMask = req.ValueMask | request.GC_MASK_FONT
	}

	_, err := c.Send(&req)
	return req.Gcid, err
}
