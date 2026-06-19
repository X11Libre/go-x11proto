package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ChangeKeyboardMapping(c *core.X11Conn, firstKeycode, keysymsPerKeycode, keycodeCount base.CARD8, keysyms []base.CARD32) error {
	_, err := c.Send(&request.ChangeKeyboardMappingRequest{
		FirstKeycode: firstKeycode, KeysymsPerKeycode: keysymsPerKeycode,
		KeycodeCount: keycodeCount, Keysyms: keysyms,
	})
	return err
}
