package render

import "github.com/X11Libre/go-x11proto/proto/base"

// DirectFormat is RENDER's DIRECTFORMAT: channel shifts and masks.
type DirectFormat struct {
	RedShift, RedMask     base.CARD16
	GreenShift, GreenMask base.CARD16
	BlueShift, BlueMask   base.CARD16
	AlphaShift, AlphaMask base.CARD16
}

// PictFormInfo describes one server picture format (PICTFORMINFO).
type PictFormInfo struct {
	ID       PICTFORMAT
	Type     base.CARD8 // PictTypeIndexed / PictTypeDirect
	Depth    base.CARD8
	Direct   DirectFormat
	Colormap base.COLORMAP
}

// ---- QueryPictFormats ----

type QueryPictFormatsRequest struct {
	MajorOpcode base.CARD8
}

func (q *QueryPictFormatsRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorQueryPictFormats)
	return nil
}

// QueryPictFormatsReply carries the server's picture formats. Only the formats
// list is decoded (enough to pick a format by depth/channels); the screen,
// depth, visual and subpixel tables that follow it on the wire are not parsed.
type QueryPictFormatsReply struct {
	Formats []PictFormInfo

	NumScreens   base.CARD32
	NumDepths    base.CARD32
	NumVisuals   base.CARD32
	NumSubpixels base.CARD32
}

func (r *QueryPictFormatsReply) Parse(rr base.ReplyReader) error {
	numFormats := rr.CARD32()
	r.NumScreens = rr.CARD32()
	r.NumDepths = rr.CARD32()
	r.NumVisuals = rr.CARD32()
	r.NumSubpixels = rr.CARD32()
	rr.ReadBytes(4) // pad

	r.Formats = make([]PictFormInfo, 0, numFormats)
	for i := base.CARD32(0); i < numFormats; i++ {
		var f PictFormInfo
		f.ID = PICTFORMAT(rr.CARD32())
		f.Type = rr.CARD8()
		f.Depth = rr.CARD8()
		rr.ReadBytes(2) // pad
		f.Direct.RedShift = rr.CARD16()
		f.Direct.RedMask = rr.CARD16()
		f.Direct.GreenShift = rr.CARD16()
		f.Direct.GreenMask = rr.CARD16()
		f.Direct.BlueShift = rr.CARD16()
		f.Direct.BlueMask = rr.CARD16()
		f.Direct.AlphaShift = rr.CARD16()
		f.Direct.AlphaMask = rr.CARD16()
		f.Colormap = base.COLORMAP(rr.CARD32())
		if rr.LastError != nil {
			return rr.LastError
		}
		r.Formats = append(r.Formats, f)
	}
	return rr.LastError
}

// FindFormat returns the first Direct format of the given depth, requiring an
// alpha channel when withAlpha is true (and no alpha when false). It returns
// nil if none matches. depth 32 + alpha is the standard ARGB32; depth 24
// without alpha is the standard RGB24.
func (r *QueryPictFormatsReply) FindFormat(depth base.CARD8, withAlpha bool) *PictFormInfo {
	for i := range r.Formats {
		f := &r.Formats[i]
		if f.Type == PictTypeDirect && f.Depth == depth && (f.Direct.AlphaMask != 0) == withAlpha {
			return f
		}
	}
	return nil
}

// QueryPictFormats fetches the server's picture formats.
func (r *Render) QueryPictFormats() (*QueryPictFormatsReply, error) {
	reply, err := r.conn.SendAndWait(&QueryPictFormatsRequest{MajorOpcode: r.MajorOpcode()})
	if err != nil {
		return nil, err
	}
	rep := &QueryPictFormatsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
