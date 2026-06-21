package core

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// Pixmap wraps a server-side pixmap resource. It embeds Drawable, so all the
// drawing, copy and query methods are available on it directly.
type Pixmap struct {
	Drawable
}

// CreatePixmap allocates a pixmap of the given depth and size. ref is any
// drawable on the target screen (e.g. the root or a window) and selects the
// screen the pixmap is created on. The returned Pixmap must be released with
// Free.
func (tkc *TkConn) CreatePixmap(depth base.CARD8, ref base.DRAWABLE, width, height base.CARD16) (*Pixmap, error) {
	xid, err := rpc.CreatePixmap(tkc.X11Conn, depth, ref, width, height)
	if err != nil {
		return nil, err
	}
	return &Pixmap{Drawable{Conn: tkc, XID: xid}}, nil
}

// Free releases the pixmap.
func (p *Pixmap) Free() error {
	return rpc.FreePixmap(p.Conn.X11Conn, p.XID)
}
