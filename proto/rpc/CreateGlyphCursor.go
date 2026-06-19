package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// CreateGlyphCursor allocates a cursor id and creates a cursor from font glyphs;
// fore/back are {red, green, blue}.
func CreateGlyphCursor(c *core.X11Conn, sourceFont, maskFont base.FONT, sourceChar, maskChar base.CARD16, fore, back [3]base.CARD16) (base.CURSOR, error) {
	cid := base.CURSOR(c.NextResourceID())
	_, err := c.Send(&request.CreateGlyphCursorRequest{
		Cid: cid, SourceFont: sourceFont, MaskFont: maskFont,
		SourceChar: sourceChar, MaskChar: maskChar,
		ForeRed: fore[0], ForeGreen: fore[1], ForeBlue: fore[2],
		BackRed: back[0], BackGreen: back[1], BackBlue: back[2],
	})
	return cid, err
}
