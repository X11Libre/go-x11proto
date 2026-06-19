package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetFontPath(c *core.X11Conn, path []string) error {
	_, err := c.Send(&request.SetFontPathRequest{Path: path})
	return err
}
