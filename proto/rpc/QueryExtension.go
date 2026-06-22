package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

type ExtensionInfo struct {
	Name        string
	Present     bool
	MajorOpcode base.CARD8
	FirstEvent  base.CARD8
	FirstError  base.CARD8
}

func QueryExtension(c *core.X11Conn, name string) (ExtensionInfo, error) {
	req := request.QueryExtensionRequest{
		Name: name,
	}

	reply, err := c.SendAndWait(&req)
	if err != nil {
		return ExtensionInfo{}, err
	}

	rep := request.QueryExtensionReply{}
	if err := rep.Parse(*reply); err != nil {
		return ExtensionInfo{}, err
	}

	return ExtensionInfo{
		Name:        name,
		Present:     rep.Present,
		MajorOpcode: rep.MajorOpcode,
		FirstEvent:  rep.FirstEvent,
		FirstError:  rep.FirstError,
	}, nil
}
