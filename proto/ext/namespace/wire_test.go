package namespace

import (
	"bytes"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

// --- little-endian wire builders (mirroring proto/core/request tests) ---

func u8(v byte) []byte    { return []byte{v} }
func u16(v uint16) []byte { return []byte{byte(v), byte(v >> 8)} }
func u32(v uint32) []byte { return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)} }

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func pad(b []byte) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return b
}

// req builds expected request bytes: major opcode, minor opcode, auto length,
// padded payload.
func req(opcode, minor byte, payload []byte) []byte {
	p := pad(append([]byte{}, payload...))
	units := uint16(len(p)/4 + 1)
	return cat(u8(opcode), u8(minor), u16(units), p)
}

func encode(t *testing.T, r base.Request) []byte {
	t.Helper()
	w := base.MakeRequestWriter(false)
	if err := r.WriteInto(&w); err != nil {
		t.Fatalf("WriteInto: %v", err)
	}
	return w.ToBytes()
}

func checkEncode(t *testing.T, r base.Request, want []byte) {
	t.Helper()
	got := encode(t, r)
	if !bytes.Equal(got, want) {
		t.Errorf("encode %T:\n got  %v\n want %v", r, got, want)
	}
	if len(got)%4 != 0 {
		t.Errorf("encode %T: length %d not a multiple of 4", r, len(got))
	}
	units := uint16(got[2]) | uint16(got[3])<<8
	if int(units)*4 != len(got) {
		t.Errorf("encode %T: length field %d units != %d bytes", r, units, len(got))
	}
}

// makeReply builds a ReplyReader as the connection would: data0 is the reply
// header byte and tail the payload from just after the 8-byte header.
func makeReply(data0 byte, tail []byte) *base.ReplyReader {
	r := &base.ReplyReader{Data0: base.CARD8(data0)}
	r.SetPayload(tail, false)
	return r
}

const op = 200 // a stand-in major opcode for encode tests

func TestEncodeQueryVersion(t *testing.T) {
	checkEncode(t, &QueryVersionRequest{MajorOpcode: op, ClientMajor: 1, ClientMinor: 0},
		req(op, 0, cat(u16(1), u16(0))))
}

func TestEncodeListNamespaces(t *testing.T) {
	checkEncode(t, &ListNamespacesRequest{MajorOpcode: op}, req(op, 1, nil))
}

func TestEncodeCreateNamespace(t *testing.T) {
	checkEncode(t, &CreateNamespaceRequest{MajorOpcode: op, Capabilities: CapAll, Attributes: AttrTransient, Name: "web"},
		req(op, 2, cat(u32(uint32(CapAll)), u32(uint32(AttrTransient)), u16(3), u16(0), []byte("web"))))
}

func TestEncodeDeleteNamespace(t *testing.T) {
	checkEncode(t, &DeleteNamespaceRequest{MajorOpcode: op, OnClients: DeleteKillClients, Name: "web"},
		req(op, 3, cat(u8(1), u8(0), u16(3), []byte("web"))))
}

func TestEncodeQueryNamespace(t *testing.T) {
	checkEncode(t, &QueryNamespaceRequest{MajorOpcode: op, Name: "abcd"},
		req(op, 4, cat(u16(4), u16(0), []byte("abcd"))))
}

func TestEncodeSetNamespaceFlags(t *testing.T) {
	checkEncode(t, &SetNamespaceFlagsRequest{MajorOpcode: op, ValueMask: CapInput, Values: CapInput, Name: "x"},
		req(op, 5, cat(u32(uint32(CapInput)), u32(uint32(CapInput)), u16(1), u16(0), []byte("x"))))
}

func TestEncodeAddAuthToken(t *testing.T) {
	// name "ns" (2 -> pad to 4), proto "X" (1 -> pad to 4), data {0xde,0xad,0xbe} (3 -> pad to 4)
	checkEncode(t, &AddAuthTokenRequest{MajorOpcode: op, Name: "ns", Proto: "X", Data: []byte{0xde, 0xad, 0xbe}},
		req(op, 6, cat(u16(2), u16(1), u16(3), u16(0),
			pad([]byte("ns")), pad([]byte("X")), pad([]byte{0xde, 0xad, 0xbe}))))
}

func TestEncodeRemoveAuthToken(t *testing.T) {
	checkEncode(t, &RemoveAuthTokenRequest{MajorOpcode: op, TokenHandle: 0x1234, Name: "ns"},
		req(op, 7, cat(u32(0x1234), u16(2), u16(0), []byte("ns"))))
}

func TestEncodeListAuthTokens(t *testing.T) {
	checkEncode(t, &ListAuthTokensRequest{MajorOpcode: op, Name: "ns"},
		req(op, 8, cat(u16(2), u16(0), []byte("ns"))))
}

func TestEncodeGetClientNamespace(t *testing.T) {
	checkEncode(t, &GetClientNamespaceRequest{MajorOpcode: op, ClientResource: 0xabcd},
		req(op, 9, u32(0xabcd)))
}

// --- reply parsing ---

func TestParseQueryVersionReply(t *testing.T) {
	rr := makeReply(0, cat(u16(1), u16(0), make([]byte, 20)))
	var rep QueryVersionReply
	if err := rep.Parse(*rr); err != nil {
		t.Fatal(err)
	}
	if rep.Major != 1 || rep.Minor != 0 {
		t.Errorf("version = %d.%d, want 1.0", rep.Major, rep.Minor)
	}
}

func TestParseInfoList(t *testing.T) {
	// two records: "root" (immutable) and "web" (transient, 2 tokens)
	rec := func(caps, attrs, refcnt, ntok uint32, name string) []byte {
		return cat(u32(caps), u32(attrs), u32(refcnt), u32(ntok),
			u16(uint16(len(name))), u16(0), pad([]byte(name)))
	}
	tail := cat(u32(2), make([]byte, 20),
		rec(uint32(CapAll), uint32(AttrImmutable), 1, 0, "root"),
		rec(uint32(CapInput), uint32(AttrTransient), 3, 2, "web"))
	got := parseInfoList(makeReply(0, tail))
	if len(got) != 2 {
		t.Fatalf("count = %d, want 2", len(got))
	}
	if got[0].Name != "root" || got[0].Attributes != AttrImmutable || got[0].Capabilities != CapAll {
		t.Errorf("record 0 = %+v", got[0])
	}
	if got[1].Name != "web" || got[1].NumTokens != 2 || got[1].Refcnt != 3 {
		t.Errorf("record 1 = %+v", got[1])
	}
}

func TestParseTokenList(t *testing.T) {
	rec := func(handle uint32, proto string) []byte {
		return cat(u32(handle), u16(uint16(len(proto))), u16(0), pad([]byte(proto)))
	}
	tail := cat(u32(2), make([]byte, 20),
		rec(0x10, "MIT-MAGIC-COOKIE-1"),
		rec(0x11, "XDM-AUTHORIZATION-1"))
	got := parseTokenList(makeReply(0, tail))
	if len(got) != 2 {
		t.Fatalf("count = %d, want 2", len(got))
	}
	if got[0].Handle != 0x10 || got[0].Proto != "MIT-MAGIC-COOKIE-1" {
		t.Errorf("token 0 = %+v", got[0])
	}
	if got[1].Handle != 0x11 || got[1].Proto != "XDM-AUTHORIZATION-1" {
		t.Errorf("token 1 = %+v", got[1])
	}
}

func TestParseGetClientNamespace(t *testing.T) {
	// isServer flag in data0, nameLen + 22 pad + name tail
	tail := cat(u16(4), make([]byte, 22), []byte("root"))
	name, isServer := parseGetClientNamespace(makeReply(1, tail))
	if name != "root" || !isServer {
		t.Errorf("got (%q, %v), want (\"root\", true)", name, isServer)
	}
	name, isServer = parseGetClientNamespace(makeReply(0, cat(u16(3), make([]byte, 22), []byte("web"))))
	if name != "web" || isServer {
		t.Errorf("got (%q, %v), want (\"web\", false)", name, isServer)
	}
}
