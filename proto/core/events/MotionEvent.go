package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type MotionEvent struct {
	GenericEvent
	Type        base.CARD8
	Serial      base.CARD32
	RootWindow  base.WINDOW
	EventWindow base.WINDOW
	ChildWindow base.WINDOW
	RootX       base.CARD16
	RootY       base.CARD16
	EventX      base.CARD16
	EventY      base.CARD16
	State       base.CARD16
	SameScreen  base.CARD8
}

func (e *MotionEvent) ReceiverWindow() base.WINDOW {
	return e.EventWindow
}

func (e *MotionEvent) Parse(rbuf base.ReadBuffer) error {
	e.Type = e.GenericEvent.Detail
	e.Serial = rbuf.CARD32()
	e.RootWindow = rbuf.WINDOW()
	e.EventWindow = rbuf.WINDOW()
	e.ChildWindow = rbuf.WINDOW()
	e.RootX = rbuf.CARD16()
	e.RootY = rbuf.CARD16()
	e.EventX = rbuf.CARD16()
	e.EventY = rbuf.CARD16()
	e.State = rbuf.CARD16()
	e.SameScreen = rbuf.CARD8()
	return rbuf.LastError
}

func ParseEvent_MotionNotify(gev GenericEvent, rbuf base.ReadBuffer) (*MotionEvent, error) {
	ev := MotionEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
