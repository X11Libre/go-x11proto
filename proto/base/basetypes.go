package base

type CARD64 uint64
type CARD32 uint32
type CARD16 uint16
type CARD8 uint8

type INT16 int16

type CARD8s []CARD8
type BOOL CARD8

type ATOM CARD32
type XID CARD32

func (x XID) Invalid() bool {
	return x == 0
}

type DRAWABLE = XID
type WINDOW = DRAWABLE
type PIXMAP = DRAWABLE
type COLORMAP = XID
type VISUAL = XID
type FONT = XID
type GC = XID
type CURSOR = XID

func bool2BOOL(b bool) BOOL {
	if b {
		return 1
	}
	return 0
}
