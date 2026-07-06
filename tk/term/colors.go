package term

// ColorMode selects how a Color's value is interpreted.
type ColorMode uint8

const (
	ColorDefault ColorMode = iota // the Term's own default Fg/Bg
	ColorIndexed                  // Index into the 16- or 256-colour ANSI palette
	ColorRGB                      // true 24-bit colour (SGR 38/48;2;r;g;b)
)

// Color is a cell's foreground or background colour as requested by SGR. It
// is resolved to an actual X11 pixel value by a pixelResolver (see
// visual.go), gated by the Term's Type.Colors (see Type.clampColor).
type Color struct {
	Mode    ColorMode
	Index   uint8 // for ColorIndexed: 0-15 basic, 16-231 the 6x6x6 cube, 232-255 grayscale
	R, G, B uint8 // for ColorRGB
}

// ansi16 is the standard 16-colour ANSI palette (indices 0-15: the 8 basic
// colours, then their bright variants), matching xterm's default.
var ansi16 = [16][3]uint8{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0},
	{0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
	{92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// indexedRGB expands a full 0-255 xterm-256-colour index to RGB: 0-15 are the
// ansi16 table, 16-231 the 6x6x6 colour cube, 232-255 a 24-step grayscale
// ramp — the standard xterm-256color layout.
func indexedRGB(i uint8) (r, g, b uint8) {
	switch {
	case i < 16:
		c := ansi16[i]
		return c[0], c[1], c[2]
	case i < 232:
		i -= 16
		levels := [6]uint8{0, 95, 135, 175, 215, 255}
		return levels[i/36], levels[(i/6)%6], levels[i%6]
	default:
		v := 8 + (i-232)*10
		return v, v, v
	}
}

// nearestIndexed quantizes an RGB colour to the nearest palette index within
// the first `colors` entries of the 256-colour table (colors is normally 16
// or 256; see Type.clampColor).
func nearestIndexed(c Color, colors int) uint8 {
	if colors > 256 {
		colors = 256
	}
	best, bestD := uint8(0), int(^uint(0)>>1)
	for i := 0; i < colors; i++ {
		r, g, b := indexedRGB(uint8(i))
		d := sq(int(r)-int(c.R)) + sq(int(g)-int(c.G)) + sq(int(b)-int(c.B))
		if d < bestD {
			best, bestD = uint8(i), d
		}
	}
	return best
}

// nearestBasic maps any 256-palette index down to the nearest of the 16 basic
// ANSI colours, for a Type with Colors == 16.
func nearestBasic(i uint8) uint8 {
	r, g, b := indexedRGB(i)
	return nearestIndexed(Color{R: r, G: g, B: b}, 16)
}

func sq(x int) int { return x * x }
