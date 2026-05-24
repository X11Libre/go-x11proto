package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func OpenFont(c *core.X11Conn, name string) (base.FONT, error) {
	fontid := base.FONT(c.NextResourceID())
	req := request.OpenFontRequest{
		FontID: fontid,
		Name:   name,
	}
	_, err := c.Send(&req)
	return fontid, err
}
