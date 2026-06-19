package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetModifierMapping(c *core.X11Conn, keycodesPerModifier base.CARD8, keycodes []base.CARD8) (base.CARD8, error) {
	reply, err := c.SendAndWait(&request.SetModifierMappingRequest{KeycodesPerModifier: keycodesPerModifier, Keycodes: keycodes})
	if err != nil {
		return 0, err
	}
	rep := &request.SetModifierMappingReply{}
	if err := rep.Parse(*reply); err != nil {
		return 0, err
	}
	return rep.Status, nil
}
