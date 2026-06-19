package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetScreenSaver(c *core.X11Conn, timeout, interval base.INT16, preferBlanking, allowExposures base.CARD8) error {
	_, err := c.Send(&request.SetScreenSaverRequest{
		Timeout: timeout, Interval: interval, PreferBlanking: preferBlanking, AllowExposures: allowExposures,
	})
	return err
}
