package font

import (
	"image"

	"github.com/X11Libre/go-x11proto/proto/base"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/xpm"
)

// glyphLumFull is the master coverage value treated as "fully lit": coverage at
// or above it maps to the full tint colour.
const glyphLumFull = 205

// Digits is a fixed-width numeric font ("0".."9") rendered from greyscale
// master glyph images. Each master's alpha channel is its coverage; Build
// scales the masters for a resolution and tints them, compositing over black
// into opaque server-side pixmaps that Draw blits with CopyArea. It complements
// the bitmap Glyphs font: Glyphs draws labels, Digits draws the counters.
type Digits struct {
	tk      *tk_core.TkConn
	masters [10]image.Image
	pix     [10]*tk_core.Pixmap
	adv     int // cell width / advance
	cellH   int // cell height
}

// NewDigits returns a digit font backed by the given master images (slice index
// is the digit value). A nil master renders as a blank cell.
func NewDigits(tk *tk_core.TkConn, masters [10]image.Image) *Digits {
	return &Digits{tk: tk, masters: masters}
}

// Advance returns the per-digit cell width set by the last Build.
func (d *Digits) Advance() int { return d.adv }

// Build (re)creates the digit pixmaps for the given cell width (adv), pixel
// scale and tint colour, releasing any previously built pixmaps first. The
// glyph is 7 C64 rows tall, centred in an adv x (8*scale) cell.
func (d *Digits) Build(adv, scale int, tint [3]byte) {
	d.Free()
	d.adv = adv
	d.cellH = 8 * scale
	cw, ch := adv, 8*scale
	gh := 7 * scale // glyph height (7 C64 rows)
	for digit := 0; digit < 10; digit++ {
		m := d.masters[digit]
		if m == nil {
			continue
		}
		// content bounding box from the alpha channel
		mb := m.Bounds()
		minx, miny, maxx, maxy := mb.Max.X, mb.Max.Y, mb.Min.X-1, mb.Min.Y-1
		for y := mb.Min.Y; y < mb.Max.Y; y++ {
			for x := mb.Min.X; x < mb.Max.X; x++ {
				if _, _, _, a := m.At(x, y).RGBA(); a>>8 >= 16 {
					if x < minx {
						minx = x
					}
					if y < miny {
						miny = y
					}
					if x > maxx {
						maxx = x
					}
					if y > maxy {
						maxy = y
					}
				}
			}
		}
		if maxx < minx {
			continue
		}
		bw, bh := maxx-minx+1, maxy-miny+1
		gw := bw * gh / bh // preserve aspect (so '4' stays a touch wider)
		if gw > cw {
			gw = cw
		}
		xoff := (cw - gw) / 2

		px := make([]byte, cw*ch*4)
		for i := 3; i < len(px); i += 4 {
			px[i] = 0xFF // opaque black background
		}
		for ty := 0; ty < gh; ty++ {
			for tx := 0; tx < gw; tx++ {
				sx0 := minx + tx*bw/gw
				sx1 := minx + (tx+1)*bw/gw
				sy0 := miny + ty*bh/gh
				sy1 := miny + (ty+1)*bh/gh
				if sx1 <= sx0 {
					sx1 = sx0 + 1
				}
				if sy1 <= sy0 {
					sy1 = sy0 + 1
				}
				var sa, n int
				for yy := sy0; yy < sy1; yy++ {
					for xx := sx0; xx < sx1; xx++ {
						_, _, _, a := m.At(xx, yy).RGBA()
						sa += int(a >> 8)
						n++
					}
				}
				if n == 0 {
					n = 1
				}
				cov := sa / n // averaged alpha = glyph coverage
				o := (ty*cw + xoff + tx) * 4
				px[o] = tintByte(tint[0], cov)
				px[o+1] = tintByte(tint[1], cov)
				px[o+2] = tintByte(tint[2], cov)
				px[o+3] = 0xFF
			}
		}
		img := &xpm.Image{Width: cw, Height: ch, Data: px}
		pm, err := img.Upload(d.tk.X11Conn, d.tk.X11Conn.DefaultRoot())
		if err != nil {
			continue
		}
		d.pix[digit] = &tk_core.Pixmap{Drawable: tk_core.Drawable{Conn: d.tk, XID: pm}}
	}
}

// Draw blits the digits of text onto target, advancing one fixed-width cell per
// character. Non-digit runes are skipped (they still consume a cell).
func (d *Digits) Draw(target base.DRAWABLE, gc base.GC, x, y base.INT16, text string) {
	for i, c := range text {
		if c < '0' || c > '9' {
			continue
		}
		pm := d.pix[c-'0']
		if pm == nil {
			continue
		}
		pm.CopyArea(target, gc,
			0, 0, x+base.INT16(i*d.adv), y,
			base.CARD16(d.adv), base.CARD16(d.cellH))
	}
}

// Free releases the built pixmaps.
func (d *Digits) Free() {
	for i := range d.pix {
		if d.pix[i] != nil {
			d.pix[i].Free()
			d.pix[i] = nil
		}
	}
}

// tintByte scales tint channel t by coverage cov (0..glyphLumFull -> 0..t).
func tintByte(t byte, cov int) byte {
	v := int(t) * cov / glyphLumFull
	if v > 255 {
		v = 255
	}
	return byte(v)
}
