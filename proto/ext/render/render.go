// Package render implements a useful subset of the X RENDER extension
// (compositing, picture formats, solid fills). It follows the same conventions
// as the core protocol: request structs with WriteInto, reply structs with
// Parse, and a per-connection handle (Render) carrying the runtime-assigned
// major opcode.
//
//	rdr, err := render.Query(conn)
//	if err != nil { ... }
//	fmts, _ := rdr.QueryPictFormats()
//	argb := fmts.FindFormat(32, true)
//	pic, _ := rdr.CreatePicture(pixmap, argb.ID, render.PictureValues{})
//	rdr.FillRectangles(render.PictOpSrc, pic, render.Color{Alpha: 0xffff}, rects)
package render

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
)

// ExtName is the wire name queried via QueryExtension.
const ExtName = "RENDER"

// Protocol version this code targets.
const (
	VersionMajor base.CARD32 = 0
	VersionMinor base.CARD32 = 11
)

// Minor opcodes (the request's data byte for an extension request).
const (
	MinorQueryVersion     base.CARD8 = 0
	MinorQueryPictFormats base.CARD8 = 1
	MinorCreatePicture    base.CARD8 = 4
	MinorChangePicture    base.CARD8 = 5
	MinorFreePicture      base.CARD8 = 7
	MinorComposite        base.CARD8 = 8
	MinorFillRectangles   base.CARD8 = 26
)

// NumErrors is the count of RENDER error codes (BadPictFormat, BadPicture,
// BadPictOp, BadGlyphSet, BadGlyph), used when registering them.
const NumErrors = 5

// Resource-id aliases (all XIDs).
type (
	PICTURE    = base.XID
	PICTFORMAT = base.XID
)

// Picture format types.
const (
	PictTypeIndexed base.CARD8 = 0
	PictTypeDirect  base.CARD8 = 1
)

// Compositing operators (PICTOP).
const (
	PictOpClear       byte = 0
	PictOpSrc         byte = 1
	PictOpDst         byte = 2
	PictOpOver        byte = 3
	PictOpOverReverse byte = 4
	PictOpIn          byte = 5
	PictOpInReverse   byte = 6
	PictOpOut         byte = 7
	PictOpOutReverse  byte = 8
	PictOpAtop        byte = 9
	PictOpAtopReverse byte = 10
	PictOpXor         byte = 11
	PictOpAdd         byte = 12
	PictOpSaturate    byte = 13
)

// Repeat modes (the CPRepeat picture value).
const (
	RepeatNone    base.CARD32 = 0
	RepeatNormal  base.CARD32 = 1
	RepeatPad     base.CARD32 = 2
	RepeatReflect base.CARD32 = 3
)

// Color is a RENDER COLOR: four 16-bit channels.
type Color struct {
	Red, Green, Blue, Alpha base.CARD16
}

// Render is the per-connection handle to the RENDER extension.
type Render struct {
	conn *core.X11Conn
	ext  *core.Extension
}

// Query negotiates RENDER on c, returning an error if it is not present. It
// registers RENDER's error codes for nicer error reporting. RENDER defines no
// events.
func Query(c *core.X11Conn) (*Render, error) {
	ext, err := c.QueryExtension(ExtName)
	if err != nil {
		return nil, err
	}
	if !ext.Present {
		return nil, fmt.Errorf("render: %s extension not available", ExtName)
	}
	c.RegisterExtensionErrors(ext.FirstError, NumErrors, "RENDER")
	return &Render{conn: c, ext: ext}, nil
}

// MajorOpcode is the server-assigned request opcode for RENDER.
func (r *Render) MajorOpcode() base.CARD8 { return r.ext.MajorOpcode }

// ---- QueryVersion ----

type QueryVersionRequest struct {
	MajorOpcode base.CARD8
	ClientMajor base.CARD32
	ClientMinor base.CARD32
}

func (q *QueryVersionRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorQueryVersion)
	w.WriteCARD32(q.ClientMajor)
	w.WriteCARD32(q.ClientMinor)
	return nil
}

type QueryVersionReply struct {
	Major base.CARD32
	Minor base.CARD32
}

func (r *QueryVersionReply) Parse(rr base.ReplyReader) error {
	r.Major = rr.CARD32()
	r.Minor = rr.CARD32()
	return rr.LastError
}

// QueryVersion negotiates the protocol version; the returned values are the
// server's supported version (<= what we asked for).
func (r *Render) QueryVersion() (major, minor base.CARD32, err error) {
	reply, err := r.conn.SendAndWait(&QueryVersionRequest{
		MajorOpcode: r.MajorOpcode(),
		ClientMajor: VersionMajor,
		ClientMinor: VersionMinor,
	})
	if err != nil {
		return 0, 0, err
	}
	var rep QueryVersionReply
	if err := rep.Parse(*reply); err != nil {
		return 0, 0, err
	}
	return rep.Major, rep.Minor, nil
}
