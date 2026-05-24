package base

type Rectangle struct {
	X      INT16
	Y      INT16
	Width  CARD16
	Height CARD16
}

func (r *Rectangle) Parse(rb *ReadBuffer) error {
	r.X = rb.INT16()
	r.Y = rb.INT16()
	r.Width = rb.CARD16()
	r.Height = rb.CARD16()
	return rb.LastError
}

func (r Rectangle) WriteInto(rw *RequestWriter) error {
	rw.WriteINT16(r.X)
	rw.WriteINT16(r.Y)
	rw.WriteCARD16(r.Width)
	rw.WriteCARD16(r.Height)
	return nil
}
