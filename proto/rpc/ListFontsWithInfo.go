package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// ListFontsWithInfo returns info for up to maxNames fonts matching pattern. The
// server answers with one reply per font followed by a terminating reply; this
// iterates the whole series (via SendAndIterate) and returns one entry per font
// (the terminator is consumed, not returned).
func ListFontsWithInfo(c *core.X11Conn, maxNames base.CARD16, pattern string) ([]request.ListFontsWithInfoReply, error) {
	it, err := c.SendAndIterate(&request.ListFontsWithInfoRequest{MaxNames: maxNames, Pattern: pattern})
	if err != nil {
		return nil, err
	}
	defer it.Close()

	var fonts []request.ListFontsWithInfoReply
	for {
		reply, err := it.Next()
		if err != nil {
			return nil, err
		}
		rep := request.ListFontsWithInfoReply{}
		if err := rep.Parse(*reply); err != nil {
			return nil, err
		}
		if rep.LastReply {
			break
		}
		fonts = append(fonts, rep)
	}
	return fonts, nil
}
