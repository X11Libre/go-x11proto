package events

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_code"
)

func TestSelectionNotifyEncodeParse(t *testing.T) {
	orig := &SelectionNotifyEvent{
		Timestamp: 0x11223344,
		Requestor: 0x00556677,
		Selection: 0x0a,
		Target:    0x0b,
		Property:  0x0c,
	}
	for _, be := range []bool{false, true} {
		raw := orig.Encode(be)
		ev, err := ParseEvent(raw[:], be)
		if err != nil {
			t.Fatalf("be=%v: ParseEvent: %v", be, err)
		}
		ne, ok := ev.(*SelectionNotifyEvent)
		if !ok {
			t.Fatalf("be=%v: got %T, want *SelectionNotifyEvent", be, ev)
		}
		if ne.Timestamp != orig.Timestamp || ne.Requestor != orig.Requestor ||
			ne.Selection != orig.Selection || ne.Target != orig.Target || ne.Property != orig.Property {
			t.Errorf("be=%v: round-trip mismatch: %+v vs %+v", be, ne, orig)
		}
		if ne.ReceiverWindow() != orig.Requestor {
			t.Errorf("be=%v: ReceiverWindow = %d, want %d", be, ne.ReceiverWindow(), orig.Requestor)
		}
	}
}

func buildEvent(be bool, code base.CARD8, fields ...base.CARD32) []byte {
	wb := base.MakeWriteBuffer(be)
	wb.WriteCARD8(code)
	wb.WriteCARD8(0)
	wb.WriteCARD16(0) // sequence
	for _, f := range fields {
		wb.WriteCARD32(f)
	}
	raw := make([]byte, 32)
	copy(raw, wb.PayloadBytes())
	return raw
}

func TestSelectionRequestParse(t *testing.T) {
	for _, be := range []bool{false, true} {
		raw := buildEvent(be, event_code.SelectionRequest,
			0x1111 /*time*/, 0x2222 /*owner*/, 0x3333 /*requestor*/, 0x44 /*sel*/, 0x55 /*target*/, 0x66 /*prop*/)
		ev, err := ParseEvent(raw, be)
		if err != nil {
			t.Fatalf("be=%v: %v", be, err)
		}
		re, ok := ev.(*SelectionRequestEvent)
		if !ok {
			t.Fatalf("be=%v: got %T", be, ev)
		}
		if re.Owner != 0x2222 || re.Requestor != 0x3333 || re.Selection != 0x44 ||
			re.Target != 0x55 || re.Property != 0x66 {
			t.Errorf("be=%v: %+v", be, re)
		}
		if re.ReceiverWindow() != re.Owner {
			t.Errorf("be=%v: ReceiverWindow = %d, want owner %d", be, re.ReceiverWindow(), re.Owner)
		}
	}
}

func TestSelectionClearParse(t *testing.T) {
	for _, be := range []bool{false, true} {
		raw := buildEvent(be, event_code.SelectionClear, 0x1234 /*time*/, 0xabcd /*owner*/, 0x42 /*sel*/)
		ev, err := ParseEvent(raw, be)
		if err != nil {
			t.Fatalf("be=%v: %v", be, err)
		}
		ce, ok := ev.(*SelectionClearEvent)
		if !ok {
			t.Fatalf("be=%v: got %T", be, ev)
		}
		if ce.Owner != 0xabcd || ce.Selection != 0x42 {
			t.Errorf("be=%v: %+v", be, ce)
		}
	}
}
