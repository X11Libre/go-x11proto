package xembed

import (
	"encoding/binary"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_code"
)

// buildCM builds a ClientMessageEvent the way sendXEmbed does, but with a
// caller-chosen atom id, so the wire payload can be checked without a server.
func buildCM(atom base.ATOM, win base.WINDOW, data [5]base.CARD32) *events.ClientMessageEvent {
	return &events.ClientMessageEvent{
		Window:      win,
		MessageType: atom,
		Format:      32,
		Data:        data,
	}
}

// TestSendXEmbedWireFormat pins the little-endian wire encoding of an
// XEMBED_EMBEDDED_NOTIFY ClientMessage: event code 33, format 32, window, then
// the five payload words (timestamp deliberately 0 = CurrentTime).
func TestSendXEmbedWireFormat(t *testing.T) {
	at := base.ATOM(0x1234)
	win := base.WINDOW(0x000003e8)
	data := [5]base.CARD32{0, 0x3e8, MsgEmbeddedNotify, Version, 0}

	ev := buildCM(at, win, data).Encode(false) // little-endian

	if code := ev[0]; code != byte(event_code.ClientMessage) {
		t.Errorf("event code = %#x, want %#x", code, byte(event_code.ClientMessage))
	}
	if format := ev[1]; format != 32 {
		t.Errorf("format = %d, want 32", format)
	}
	// Window at bytes 4..8.
	if got := base.WINDOW(binary.LittleEndian.Uint32(ev[4:8])); got != win {
		t.Errorf("window = %#x, want %#x", got, win)
	}
	// MessageType (the _XEMBED atom) at bytes 8..12.
	if got := base.ATOM(binary.LittleEndian.Uint32(ev[8:12])); got != at {
		t.Errorf("message type = %#x, want %#x", got, at)
	}
	// Data[0] timestamp and Data[2] message id.
	if got := binary.LittleEndian.Uint32(ev[12:16]); got != 0 {
		t.Errorf("data[0] timestamp = %d, want 0 (CurrentTime)", got)
	}
	if got := binary.LittleEndian.Uint32(ev[20:24]); got != MsgEmbeddedNotify {
		t.Errorf("data[2] message = %d, want %d", got, MsgEmbeddedNotify)
	}
}

// TestMessageParsing verifies the embedder's Message() round-trips the id and
// detail values out of a received _XEMBED ClientMessage.
func TestMessageParsing(t *testing.T) {
	at, win, client := base.ATOM(0x99), base.WINDOW(0x2000), base.WINDOW(0x3000)
	c := &Conn{at: at}
	data := [5]base.CARD32{base.CARD32(client), 0, MsgFocusNext, FocusFirst, 0}
	ev := buildCM(at, win, data)

	id, d1, d2, ok := c.Message(ev)
	if !ok {
		t.Fatalf("Message(%v): not a _XEMBED message", ev)
	}
	if id != MsgFocusNext {
		t.Errorf("id = %d, want %d", id, MsgFocusNext)
	}
	if d1 != FocusFirst {
		t.Errorf("detail1 = %d, want %d", d1, FocusFirst)
	}
	if d2 != 0 {
		t.Errorf("detail2 = %d, want 0", d2)
	}

	// A non-_XEMBED ClientMessage must be rejected.
	other := buildCM(base.ATOM(0x7f), win, data)
	if _, _, _, ok := c.Message(other); ok {
		t.Error("Message() accepted a non-_XEMBED ClientMessage")
	}
}
