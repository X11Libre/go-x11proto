package xpm

import (
	"bytes"
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

func DecodeBytes(data []byte) (*Image, error) {
	return Decode(bytes.NewReader(data))
}

// maxPutImageData is the maximum data payload for a PutImage request.
// X11 length field is CARD16, max 65535 → max request = 65535*4 = 262140,
// header = 24 bytes, so max data = 262116.
const maxPutImageData = 262116

// bppForDepth looks up the server's bits-per-pixel for the given depth
// from the connection setup pixmap formats.
func bppForDepth(c *core.X11Conn, depth base.CARD8) int {
	for _, pf := range c.Setup.PixmapFormats {
		if pf.Depth == depth {
			return int(pf.BitsPerPixel)
		}
	}
	return int((depth+7)/8) * 8
}

func zPixmapBytes(img *Image, bpp int, imageByteOrder base.CARD8) []byte {
	bpc := bpp / 8
	scanline := ((img.Width*bpp + 31) / 32) * 4
	out := make([]byte, scanline*img.Height)
	for y := 0; y < img.Height; y++ {
		dst := out[y*scanline:]
		for x := 0; x < img.Width; x++ {
			si := (y*img.Width + x) * 4
			di := x * bpc
			r, g, b := img.Data[si+0], img.Data[si+1], img.Data[si+2]
			switch bpc {
			case 1:
				v := (int(r)*30 + int(g)*59 + int(b)*11) / 100
				dst[di] = byte(v)
			case 2:
				hi, lo := b, r
				if imageByteOrder == 0 {
					lo, hi = r, b
				}
				dst[di+0] = lo
				dst[di+1] = hi
			case 3:
				if imageByteOrder == 0 {
					dst[di+0] = b
					dst[di+1] = g
					dst[di+2] = r
				} else {
					dst[di+0] = r
					dst[di+1] = g
					dst[di+2] = b
				}
			case 4:
				if imageByteOrder == 0 {
					dst[di+0] = b
					dst[di+1] = g
					dst[di+2] = r
					dst[di+3] = 0
				} else {
					dst[di+0] = r
					dst[di+1] = g
					dst[di+2] = b
					dst[di+3] = 0
				}
			}
		}
	}
	return out
}

func (img *Image) Upload(c *core.X11Conn, drawable base.DRAWABLE) (base.PIXMAP, error) {
	depth := c.Setup.Screens[0].RootDepth
	bpp := bppForDepth(c, depth)
	scanline := ((img.Width*bpp + 31) / 32) * 4
	maxLines := maxPutImageData / scanline

	pixmap, err := rpc.CreatePixmap(c, depth, drawable, base.CARD16(img.Width), base.CARD16(img.Height))
	if err != nil {
		return 0, fmt.Errorf("xpm: CreatePixmap: %w", err)
	}

	gcid, err := rpc.CreateGC1(c, c.DefaultBlackPixel(), c.DefaultWhitePixel(), 0)
	if err != nil {
		return 0, fmt.Errorf("xpm: CreateGC1: %w", err)
	}

	data := zPixmapBytes(img, bpp, c.Setup.ImageByteOrder)
	for y := 0; y < img.Height; y += maxLines {
		n := img.Height - y
		if n > maxLines {
			n = maxLines
		}
		chunk := data[y*scanline : (y+n)*scanline]
		err = rpc.PutImage(c, pixmap, gcid, 2, depth,
			base.CARD16(img.Width), base.CARD16(n),
			base.INT16(0), base.INT16(y),
			chunk)
		if err != nil {
			return 0, fmt.Errorf("xpm: PutImage row %d: %w", y, err)
		}
	}

	return pixmap, nil
}
