package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// CreateCursor allocates a cursor id and creates the cursor; fore/back are
// {red, green, blue}.
func CreateCursor(c *core.X11Conn, source, mask base.PIXMAP, fore, back [3]base.CARD16, x, y base.CARD16) (base.CURSOR, error) {
	cid := base.CURSOR(c.NextResourceID())
	_, err := c.Send(&request.CreateCursorRequest{
		Cid: cid, Source: source, Mask: mask,
		ForeRed: fore[0], ForeGreen: fore[1], ForeBlue: fore[2],
		BackRed: back[0], BackGreen: back[1], BackBlue: back[2],
		X: x, Y: y,
	})
	return cid, err
}
