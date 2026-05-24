package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type VisibilityEvent struct {
	GenericEvent
	Window base.WINDOW
	State  base.CARD8
}

func (e *VisibilityEvent) ReceiverWindow() base.WINDOW {
	return e.Window
}

func (e *VisibilityEvent) Parse(rbuf base.ReadBuffer) error {
	e.Window = rbuf.WINDOW()
	e.State = rbuf.CARD8()
	return rbuf.LastError
}

func ParseEvent_VisibilityNotify(gev GenericEvent, rbuf base.ReadBuffer) (*VisibilityEvent, error) {
	ev := VisibilityEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
