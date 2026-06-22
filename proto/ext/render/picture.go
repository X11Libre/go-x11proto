package render

import "github.com/X11Libre/go-x11proto/proto/base"

// Picture value-list mask bits (CP*), in wire (ascending-bit) order.
const (
	CPRepeat           base.CARD32 = 1 << 0
	CPAlphaMap         base.CARD32 = 1 << 1
	CPAlphaXOrigin     base.CARD32 = 1 << 2
	CPAlphaYOrigin     base.CARD32 = 1 << 3
	CPClipXOrigin      base.CARD32 = 1 << 4
	CPClipYOrigin      base.CARD32 = 1 << 5
	CPClipMask         base.CARD32 = 1 << 6
	CPGraphicsExposure base.CARD32 = 1 << 7
	CPSubwindowMode    base.CARD32 = 1 << 8
	CPPolyEdge         base.CARD32 = 1 << 9
	CPPolyMode         base.CARD32 = 1 << 10
	CPDither           base.CARD32 = 1 << 11
	CPComponentAlpha   base.CARD32 = 1 << 12
)

// PictureValues is the optional value list for CreatePicture / ChangePicture.
// Set ValueMask to the OR of the CP* bits you provide; each value is sent as a
// 4-byte entry. Fields are CARD32 because every value occupies 4 bytes on the
// wire (XIDs, enums and the INT16 origins alike).
type PictureValues struct {
	ValueMask base.CARD32

	Repeat           base.CARD32
	AlphaMap         base.CARD32
	AlphaXOrigin     base.CARD32
	AlphaYOrigin     base.CARD32
	ClipXOrigin      base.CARD32
	ClipYOrigin      base.CARD32
	ClipMask         base.CARD32
	GraphicsExposure base.CARD32
	SubwindowMode    base.CARD32
	PolyEdge         base.CARD32
	PolyMode         base.CARD32
	Dither           base.CARD32
	ComponentAlpha   base.CARD32
}

func (v PictureValues) has(m base.CARD32) bool { return v.ValueMask&m == m }

// writeValues writes the value mask followed by the present values in bit order.
func (v PictureValues) writeValues(w *base.RequestWriter) {
	w.WriteCARD32(v.ValueMask)
	if v.has(CPRepeat) {
		w.WriteCARD32(v.Repeat)
	}
	if v.has(CPAlphaMap) {
		w.WriteCARD32(v.AlphaMap)
	}
	if v.has(CPAlphaXOrigin) {
		w.WriteCARD32(v.AlphaXOrigin)
	}
	if v.has(CPAlphaYOrigin) {
		w.WriteCARD32(v.AlphaYOrigin)
	}
	if v.has(CPClipXOrigin) {
		w.WriteCARD32(v.ClipXOrigin)
	}
	if v.has(CPClipYOrigin) {
		w.WriteCARD32(v.ClipYOrigin)
	}
	if v.has(CPClipMask) {
		w.WriteCARD32(v.ClipMask)
	}
	if v.has(CPGraphicsExposure) {
		w.WriteCARD32(v.GraphicsExposure)
	}
	if v.has(CPSubwindowMode) {
		w.WriteCARD32(v.SubwindowMode)
	}
	if v.has(CPPolyEdge) {
		w.WriteCARD32(v.PolyEdge)
	}
	if v.has(CPPolyMode) {
		w.WriteCARD32(v.PolyMode)
	}
	if v.has(CPDither) {
		w.WriteCARD32(v.Dither)
	}
	if v.has(CPComponentAlpha) {
		w.WriteCARD32(v.ComponentAlpha)
	}
}

// ---- CreatePicture ----

type CreatePictureRequest struct {
	MajorOpcode base.CARD8
	Pid         PICTURE
	Drawable    base.DRAWABLE
	Format      PICTFORMAT
	Values      PictureValues
}

func (q *CreatePictureRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorCreatePicture)
	w.WriteXID(q.Pid)
	w.WriteXID(q.Drawable)
	w.WriteXID(q.Format)
	q.Values.writeValues(w)
	return nil
}

// CreatePicture creates a picture for drawable in the given format, allocating
// and returning its id.
func (r *Render) CreatePicture(drawable base.DRAWABLE, format PICTFORMAT, vals PictureValues) (PICTURE, error) {
	pid := PICTURE(r.conn.NextResourceID())
	_, err := r.conn.Send(&CreatePictureRequest{
		MajorOpcode: r.MajorOpcode(),
		Pid:         pid,
		Drawable:    drawable,
		Format:      format,
		Values:      vals,
	})
	return pid, err
}

// ---- ChangePicture ----

type ChangePictureRequest struct {
	MajorOpcode base.CARD8
	Picture     PICTURE
	Values      PictureValues
}

func (q *ChangePictureRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorChangePicture)
	w.WriteXID(q.Picture)
	q.Values.writeValues(w)
	return nil
}

// ChangePicture updates picture attributes.
func (r *Render) ChangePicture(pic PICTURE, vals PictureValues) error {
	_, err := r.conn.Send(&ChangePictureRequest{
		MajorOpcode: r.MajorOpcode(),
		Picture:     pic,
		Values:      vals,
	})
	return err
}

// ---- FreePicture ----

type FreePictureRequest struct {
	MajorOpcode base.CARD8
	Picture     PICTURE
}

func (q *FreePictureRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorFreePicture)
	w.WriteXID(q.Picture)
	return nil
}

// FreePicture releases a picture.
func (r *Render) FreePicture(pic PICTURE) error {
	_, err := r.conn.Send(&FreePictureRequest{MajorOpcode: r.MajorOpcode(), Picture: pic})
	return err
}
