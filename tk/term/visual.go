package term

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
)

// visualClassTrueColor and visualClassDirectColor are the X11 core protocol's
// visual class numbers (core §2.1) for the two classes that let a pixel value
// be computed directly from RGB via bit masks, with no colormap involved.
const (
	visualClassTrueColor   = 4
	visualClassDirectColor = 5
)

// pixelResolver turns a Color into an X11 pixel value for GC.SetForeground,
// for the connection's root visual. It supports TrueColor/DirectColor
// visuals directly via the visual's R/G/B masks — the overwhelmingly common
// case on every modern X server. On any other visual class (indexed
// PseudoColor/StaticColor, effectively unseen today) it degrades to the
// connection's plain black/white pixels, ignoring the requested colour
// entirely: a real palette there would need per-colour AllocColor calls and
// colormap lifetime management, which this package does not implement.
type pixelResolver struct {
	trueColor              bool
	rShift, gShift, bShift uint
	rBits, gBits, bBits    uint
	black, white           base.CARD32
}

func newPixelResolver(conn *core.X11Conn) pixelResolver {
	pr := pixelResolver{black: conn.DefaultBlackPixel(), white: conn.DefaultWhitePixel()}
	screen := conn.Setup.Screens[0]
	for _, d := range screen.Depths {
		for _, v := range d.Visuals {
			if v.Id != screen.RootVisual {
				continue
			}
			if v.Class == visualClassTrueColor || v.Class == visualClassDirectColor {
				pr.trueColor = true
				pr.rShift, pr.rBits = maskShiftBits(v.RedMask)
				pr.gShift, pr.gBits = maskShiftBits(v.GreenMask)
				pr.bShift, pr.bBits = maskShiftBits(v.BlueMask)
			}
			return pr
		}
	}
	return pr
}

// maskShiftBits decomposes a contiguous colour-channel bitmask (as found in
// XSetupScreenVisual.RedMask/GreenMask/BlueMask) into its shift and width.
func maskShiftBits(mask base.CARD32) (shift, bits uint) {
	m := uint32(mask)
	for m != 0 && m&1 == 0 {
		shift++
		m >>= 1
	}
	for m&1 == 1 {
		bits++
		m >>= 1
	}
	return
}

// scale8 maps an 8-bit channel value onto a channel of the given bit width.
func scale8(v uint8, bits uint) uint32 {
	if bits == 0 {
		return 0
	}
	if bits >= 8 {
		return uint32(v) << (bits - 8)
	}
	return uint32(v) >> (8 - bits)
}

// Pixel resolves c to a pixel value. isBg selects which of the resolver's
// two monochrome fallbacks (black/white) a ColorDefault or a
// non-TrueColor-visual colour degrades to.
func (pr pixelResolver) Pixel(c Color, isBg bool) base.CARD32 {
	if c.Mode == ColorDefault {
		if isBg {
			return pr.white
		}
		return pr.black
	}
	if !pr.trueColor {
		if isBg {
			return pr.white
		}
		return pr.black
	}
	var r, g, b uint8
	if c.Mode == ColorRGB {
		r, g, b = c.R, c.G, c.B
	} else {
		r, g, b = indexedRGB(c.Index)
	}
	return base.CARD32(scale8(r, pr.rBits)<<pr.rShift |
		scale8(g, pr.gBits)<<pr.gShift |
		scale8(b, pr.bBits)<<pr.bShift)
}
