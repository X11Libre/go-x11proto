package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ChangePointerControl(c *core.X11Conn, accelNum, accelDenom, threshold base.INT16, doAccel, doThreshold bool) error {
	_, err := c.Send(&request.ChangePointerControlRequest{
		AccelerationNumerator: accelNum, AccelerationDenominator: accelDenom,
		Threshold: threshold, DoAcceleration: doAccel, DoThreshold: doThreshold,
	})
	return err
}
