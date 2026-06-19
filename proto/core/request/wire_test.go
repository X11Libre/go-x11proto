package request

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

// --- little-endian wire builders for expected bytes ---

func u8(v byte) []byte { return []byte{v} }

func u16(v uint16) []byte { return []byte{byte(v), byte(v >> 8)} }

func u32(v uint32) []byte { return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)} }

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// pad rounds b up to a 4-byte boundary with zero bytes.
func pad(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}

// req builds the expected full request bytes: opcode, param0, the auto-computed
// length field (in 4-byte units) and the padded payload.
func req(opcode, param0 byte, payload []byte) []byte {
	p := pad(append([]byte{}, payload...))
	units := uint16(len(p)/4 + 1)
	return cat(u8(opcode), u8(param0), u16(units), p)
}

// encode serialises a request to its full wire bytes (little-endian).
func encode(t *testing.T, r base.Request) []byte {
	t.Helper()
	w := base.MakeRequestWriter(false)
	if err := r.WriteInto(&w); err != nil {
		t.Fatalf("WriteInto: %v", err)
	}
	return w.ToBytes()
}

// checkEncode asserts that r encodes exactly to want.
func checkEncode(t *testing.T, r base.Request, want []byte) {
	t.Helper()
	got := encode(t, r)
	if !bytes.Equal(got, want) {
		t.Errorf("encode %T:\n got  %v\n want %v", r, got, want)
	}
	if len(got)%4 != 0 {
		t.Errorf("encode %T: length %d not a multiple of 4", r, len(got))
	}
	// the length field (units) must match the actual size
	units := uint16(got[2]) | uint16(got[3])<<8
	if int(units)*4 != len(got) {
		t.Errorf("encode %T: length field %d units != %d bytes", r, units, len(got))
	}
}

// makeReply builds a ReplyReader as the connection would: data0 is the reply
// header byte, length the reply length field (extra 4-byte units), and tail the
// reply payload starting just after the 8-byte type/data0/seq/length header.
func makeReply(data0 byte, length uint32, tail []byte) base.ReplyReader {
	r := base.ReplyReader{Data0: base.CARD8(data0), Length: base.CARD32(length)}
	r.SetPayload(tail, false)
	return r
}

func checkReply(t *testing.T, got, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parse mismatch:\n got  %+v\n want %+v", got, want)
	}
}
