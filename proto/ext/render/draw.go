package render

import "github.com/X11Libre/go-x11proto/proto/base"

// ---- Composite ----

type CompositeRequest struct {
	MajorOpcode   base.CARD8
	Op            byte
	Src           PICTURE
	Mask          PICTURE // 0 = None
	Dst           PICTURE
	SrcX, SrcY    base.INT16
	MaskX, MaskY  base.INT16
	DstX, DstY    base.INT16
	Width, Height base.CARD16
}

func (q *CompositeRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorComposite)
	w.WriteCARD8(base.CARD8(q.Op))
	w.Pad() // 3 bytes
	w.WriteXID(q.Src)
	w.WriteXID(q.Mask)
	w.WriteXID(q.Dst)
	w.WriteINT16(q.SrcX)
	w.WriteINT16(q.SrcY)
	w.WriteINT16(q.MaskX)
	w.WriteINT16(q.MaskY)
	w.WriteINT16(q.DstX)
	w.WriteINT16(q.DstY)
	w.WriteCARD16(q.Width)
	w.WriteCARD16(q.Height)
	return nil
}

// Composite combines src (and optional mask) into dst with operator op. Pass
// mask = 0 for no mask.
func (r *Render) Composite(op byte, src, mask, dst PICTURE,
	srcX, srcY, maskX, maskY, dstX, dstY base.INT16, width, height base.CARD16) error {
	_, err := r.conn.Send(&CompositeRequest{
		MajorOpcode: r.MajorOpcode(),
		Op:          op,
		Src:         src, Mask: mask, Dst: dst,
		SrcX: srcX, SrcY: srcY,
		MaskX: maskX, MaskY: maskY,
		DstX: dstX, DstY: dstY,
		Width: width, Height: height,
	})
	return err
}

// ---- FillRectangles ----

type FillRectanglesRequest struct {
	MajorOpcode base.CARD8
	Op          byte
	Dst         PICTURE
	Color       Color
	Rects       []base.Rectangle
}

func (q *FillRectanglesRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorFillRectangles)
	w.WriteCARD8(base.CARD8(q.Op))
	w.Pad() // 3 bytes
	w.WriteXID(q.Dst)
	w.WriteCARD16(q.Color.Red)
	w.WriteCARD16(q.Color.Green)
	w.WriteCARD16(q.Color.Blue)
	w.WriteCARD16(q.Color.Alpha)
	for _, rect := range q.Rects {
		rect.WriteInto(w)
	}
	return nil
}

// FillRectangles fills rects in dst with the given color using operator op.
func (r *Render) FillRectangles(op byte, dst PICTURE, color Color, rects []base.Rectangle) error {
	if len(rects) == 0 {
		return nil
	}
	_, err := r.conn.Send(&FillRectanglesRequest{
		MajorOpcode: r.MajorOpcode(),
		Op:          op,
		Dst:         dst,
		Color:       color,
		Rects:       rects,
	})
	return err
}
