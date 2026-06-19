package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func RecolorCursor(c *core.X11Conn, cursor base.CURSOR, fore, back [3]base.CARD16) error {
	_, err := c.Send(&request.RecolorCursorRequest{
		Cursor:  cursor,
		ForeRed: fore[0], ForeGreen: fore[1], ForeBlue: fore[2],
		BackRed: back[0], BackGreen: back[1], BackBlue: back[2],
	})
	return err
}
