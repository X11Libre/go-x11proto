package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type KeymapEvent struct {
	GenericEvent
	Keys []base.CARD8
}

func (e *KeymapEvent) ReceiverWindow() base.WINDOW {
	return 0
}

func (e *KeymapEvent) Parse(rbuf base.ReadBuffer) error {
	rbuf.Reset()
	rbuf.CARD8() // skip the event code
	e.Detail = 0
	e.Sequence = 0
	e.IsClient = false
	e.Keys = []base.CARD8{
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(),
		rbuf.CARD8(), // 31 total
	}
	return rbuf.LastError
}

func ParseEvent_KeymapNotify(gev GenericEvent, rbuf base.ReadBuffer) (*KeymapEvent, error) {
	ev := KeymapEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
