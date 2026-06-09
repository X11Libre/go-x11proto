package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type ReparentEvent struct {
	GenericEvent
	EventWindow      base.WINDOW
	TargetWindow     base.WINDOW
	ParentWindow     base.WINDOW
	X                base.INT16
	Y                base.INT16
	OverrideRedirect bool
}

func (e *ReparentEvent) ReceiverWindow() base.WINDOW {
	return e.EventWindow
}

func (e *ReparentEvent) Parse(rbuf base.ReadBuffer) error {
	e.EventWindow = rbuf.XID()
	e.TargetWindow = rbuf.XID()
	e.ParentWindow = rbuf.XID()
	e.X = rbuf.INT16()
	e.Y = rbuf.INT16()
	e.OverrideRedirect = rbuf.Bool()
	return rbuf.LastError
}

func ParseEvent_ReparentNotify(gev GenericEvent, rbuf base.ReadBuffer) (*ReparentEvent, error) {
	ev := ReparentEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
