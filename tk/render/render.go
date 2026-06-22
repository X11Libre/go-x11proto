// Package render is the tk-layer abstraction over the RENDER extension
// (proto/ext/render). It mirrors the rest of tk: a per-connection Render handle
// and a Picture object that carries its connection, so callers manipulate
// pictures with methods instead of threading conn/xid through every call.
//
//	rdr, err := tkrender.Open(tkConn)
//	argb, _ := rdr.ARGB32()
//	pic, _ := rdr.NewPicture(pixmap.XID, argb, tkrender.PictureValues{})
//	pic.FillRect(tkrender.OpSrc, tkrender.Color{Alpha: 0xffff}, 0, 0, 64, 64)
//	defer pic.Free()
package render

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	pr "github.com/X11Libre/go-x11proto/proto/ext/render"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// Re-exported types and constants so simple callers need only this package.
type (
	Color         = pr.Color
	PictureValues = pr.PictureValues
	PICTFORMAT    = pr.PICTFORMAT
	PICTURE       = pr.PICTURE
	PictFormInfo  = pr.PictFormInfo
)

// Compositing operators.
const (
	OpClear       = pr.PictOpClear
	OpSrc         = pr.PictOpSrc
	OpDst         = pr.PictOpDst
	OpOver        = pr.PictOpOver
	OpOverReverse = pr.PictOpOverReverse
	OpIn          = pr.PictOpIn
	OpOut         = pr.PictOpOut
	OpAtop        = pr.PictOpAtop
	OpXor         = pr.PictOpXor
	OpAdd         = pr.PictOpAdd
)

// Picture-value mask bits and repeat modes (the common ones; the rest live in
// proto/ext/render).
const (
	CPRepeat         = pr.CPRepeat
	CPClipMask       = pr.CPClipMask
	CPComponentAlpha = pr.CPComponentAlpha

	RepeatNone    = pr.RepeatNone
	RepeatNormal  = pr.RepeatNormal
	RepeatPad     = pr.RepeatPad
	RepeatReflect = pr.RepeatReflect
)

// Render is the tk handle to the RENDER extension on a TkConn. It caches the
// picture-format list (it never changes for a connection).
type Render struct {
	Conn    *tk_core.TkConn
	ext     *pr.Render
	formats *pr.QueryPictFormatsReply
}

// Open negotiates RENDER on conn, returning an error if the extension is absent.
func Open(conn *tk_core.TkConn) (*Render, error) {
	ext, err := pr.Query(conn.X11Conn)
	if err != nil {
		return nil, err
	}
	return &Render{Conn: conn, ext: ext}, nil
}

// Version returns the server's RENDER protocol version.
func (r *Render) Version() (major, minor base.CARD32, err error) {
	return r.ext.QueryVersion()
}

// Formats returns the server's picture formats, fetching them once and caching.
func (r *Render) Formats() (*pr.QueryPictFormatsReply, error) {
	if r.formats == nil {
		f, err := r.ext.QueryPictFormats()
		if err != nil {
			return nil, err
		}
		r.formats = f
	}
	return r.formats, nil
}

// StandardFormat returns the id of the first Direct format of the given depth,
// requiring an alpha channel when alpha is true.
func (r *Render) StandardFormat(depth base.CARD8, alpha bool) (PICTFORMAT, error) {
	f, err := r.Formats()
	if err != nil {
		return 0, err
	}
	pf := f.FindFormat(depth, alpha)
	if pf == nil {
		return 0, fmt.Errorf("render: no depth-%d %s format", depth,
			map[bool]string{true: "ARGB", false: "RGB"}[alpha])
	}
	return pf.ID, nil
}

// ARGB32 returns the standard 32-bit format with alpha.
func (r *Render) ARGB32() (PICTFORMAT, error) { return r.StandardFormat(32, true) }

// RGB24 returns the standard 24-bit format without alpha.
func (r *Render) RGB24() (PICTFORMAT, error) { return r.StandardFormat(24, false) }

// Picture wraps a server-side PICTURE together with its Render handle.
type Picture struct {
	Render *Render
	XID    PICTURE
}

// NewPicture creates a picture for a drawable (any window or pixmap XID) in the
// given format. The returned Picture must be released with Free.
func (r *Render) NewPicture(drawable base.DRAWABLE, format PICTFORMAT, vals PictureValues) (*Picture, error) {
	pid, err := r.ext.CreatePicture(drawable, format, vals)
	if err != nil {
		return nil, err
	}
	return &Picture{Render: r, XID: pid}, nil
}

// PictureFor creates a picture for a tk Drawable (so it also works for a Window
// or Pixmap via their embedded Drawable).
func (r *Render) PictureFor(d tk_core.Drawable, format PICTFORMAT, vals PictureValues) (*Picture, error) {
	return r.NewPicture(d.XID, format, vals)
}

// Fill fills rects in the picture with color using operator op.
func (p *Picture) Fill(op byte, color Color, rects []base.Rectangle) error {
	return p.Render.ext.FillRectangles(op, p.XID, color, rects)
}

// FillRect fills a single rectangle.
func (p *Picture) FillRect(op byte, color Color, x, y base.INT16, width, height base.CARD16) error {
	return p.Fill(op, color, []base.Rectangle{{X: x, Y: y, Width: width, Height: height}})
}

// Composite draws src (and an optional mask, may be nil) onto this picture.
func (p *Picture) Composite(op byte, src, mask *Picture,
	srcX, srcY, maskX, maskY, dstX, dstY base.INT16, width, height base.CARD16) error {
	var m PICTURE
	if mask != nil {
		m = mask.XID
	}
	return p.Render.ext.Composite(op, src.XID, m, p.XID,
		srcX, srcY, maskX, maskY, dstX, dstY, width, height)
}

// Change updates picture attributes.
func (p *Picture) Change(vals PictureValues) error {
	return p.Render.ext.ChangePicture(p.XID, vals)
}

// Free releases the picture.
func (p *Picture) Free() error {
	return p.Render.ext.FreePicture(p.XID)
}
