package core

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
)

func TestIsWMDelete(t *testing.T) {
	const del base.ATOM = 1234
	cm := &events.ClientMessageEvent{}
	cm.Data[0] = base.CARD32(del)

	if !IsWMDelete(cm, del) {
		t.Error("matching WM_DELETE_WINDOW client message should report true")
	}
	// wrong atom
	if IsWMDelete(cm, 5678) {
		t.Error("different atom must not match")
	}
	// del == 0 (protocol not enabled) never matches
	zero := &events.ClientMessageEvent{}
	if IsWMDelete(zero, 0) {
		t.Error("del==0 must never match")
	}
	// a non-ClientMessage event
	if IsWMDelete(&events.ExposeEvent{}, del) {
		t.Error("non-ClientMessage event must not match")
	}
}
