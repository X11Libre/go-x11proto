package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// Colour component flags (do-red/green/blue) for StoreColors / StoreNamedColor.
const (
	ColorFlagDoRed   = 0x01
	ColorFlagDoGreen = 0x02
	ColorFlagDoBlue  = 0x04
)

// COLORITEM
type ColorItem struct {
	Pixel base.CARD32
	Red   base.CARD16
	Green base.CARD16
	Blue  base.CARD16
	Flags base.CARD8
}

type StoreColorsRequest struct {
	Colormap base.COLORMAP
	Items    []ColorItem
}

func (r *StoreColorsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.StoreColors)
	writer.WriteXID(r.Colormap)
	for _, it := range r.Items {
		writer.WriteCARD32(it.Pixel)
		writer.WriteCARD16(it.Red)
		writer.WriteCARD16(it.Green)
		writer.WriteCARD16(it.Blue)
		writer.WriteCARD8(it.Flags)
		writer.WriteCARD8(0) // unused
	}
	return nil
}
