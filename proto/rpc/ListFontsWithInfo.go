package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// ListFontsWithInfo returns font info. NOTE: the server replies with one reply
// per font plus a terminating reply; the current connection layer delivers a
// single reply per request, so this returns only the first reply. Full
// enumeration needs multi-reply support (FIXME).
func ListFontsWithInfo(c *core.X11Conn, maxNames base.CARD16, pattern string) (*request.ListFontsWithInfoReply, error) {
	reply, err := c.SendAndWait(&request.ListFontsWithInfoRequest{MaxNames: maxNames, Pattern: pattern})
	if err != nil {
		return nil, err
	}
	rep := &request.ListFontsWithInfoReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
