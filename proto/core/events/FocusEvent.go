package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type FocusEvent struct {
	GenericEvent
	Reason base.CARD8
	Window base.WINDOW
	Mode   base.CARD8
}

func (e *FocusEvent) ReceiverWindow() base.WINDOW {
	return e.Window
}

func (e *FocusEvent) Parse(rbuf base.ReadBuffer) error {
	e.Reason = e.GenericEvent.Detail
	e.Window = rbuf.WINDOW()
	e.Mode = rbuf.CARD8()
	return rbuf.LastError
}

func ParseEvent_FocusNotify(gev GenericEvent, rbuf base.ReadBuffer) (*FocusEvent, error) {
	ev := FocusEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}

type FocusInEvent struct {
	FocusEvent
}

type FocusOutEvent struct {
	FocusEvent
}

func ParseEvent_FocusInNotify(gev GenericEvent, rbuf base.ReadBuffer) (*FocusInEvent, error) {
	ev := FocusInEvent{FocusEvent: FocusEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}

func ParseEvent_FocusOutNotify(gev GenericEvent, rbuf base.ReadBuffer) (*FocusOutEvent, error) {
	ev := FocusOutEvent{FocusEvent: FocusEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}
