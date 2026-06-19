package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

func TestGetAtomNameEncode(t *testing.T) {
	checkEncode(t, &GetAtomNameRequest{Atom: 0xAB}, req(17, 0, u32(0xAB)))
}

func TestGetAtomNameReply(t *testing.T) {
	tail := cat(u16(5), make([]byte, 22), []byte("hello"))
	got := &GetAtomNameReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetAtomNameReply{Name: "hello"})
}

func TestDeletePropertyEncode(t *testing.T) {
	checkEncode(t, &DeletePropertyRequest{Window: 0x10, Property: 0xAB},
		req(19, 0, cat(u32(0x10), u32(0xAB))))
}

func TestGetPropertyEncode(t *testing.T) {
	checkEncode(t, &GetPropertyRequest{Window: 0x10, Property: 0xAB, Type: 0xCD, LongOffset: 0, LongLength: 100},
		req(20, 0, cat(u32(0x10), u32(0xAB), u32(0xCD), u32(0), u32(100))))
}

func TestGetPropertyReply(t *testing.T) {
	tail := cat(u32(0xCD), u32(0), u32(3), make([]byte, 12), []byte{1, 2, 3})
	got := &GetPropertyReply{}
	if err := got.Parse(makeReply(8, 1, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetPropertyReply{Format: 8, Type: 0xCD, BytesAfter: 0, ValueLen: 3, Value: []byte{1, 2, 3}})
}

func TestListPropertiesEncode(t *testing.T) {
	checkEncode(t, &ListPropertiesRequest{Window: 0x10}, req(21, 0, u32(0x10)))
}

func TestListPropertiesReply(t *testing.T) {
	tail := cat(u16(2), make([]byte, 22), u32(0xA), u32(0xB))
	got := &ListPropertiesReply{}
	if err := got.Parse(makeReply(0, 2, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &ListPropertiesReply{Atoms: []base.ATOM{0xA, 0xB}})
}

func TestSetSelectionOwnerEncode(t *testing.T) {
	checkEncode(t, &SetSelectionOwnerRequest{Owner: 0x10, Selection: 0xAB, Time: 0x123},
		req(22, 0, cat(u32(0x10), u32(0xAB), u32(0x123))))
}

func TestGetSelectionOwnerEncode(t *testing.T) {
	checkEncode(t, &GetSelectionOwnerRequest{Selection: 0xAB}, req(23, 0, u32(0xAB)))
}

func TestGetSelectionOwnerReply(t *testing.T) {
	got := &GetSelectionOwnerReply{}
	if err := got.Parse(makeReply(0, 0, cat(u32(0x10), make([]byte, 20)))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetSelectionOwnerReply{Owner: 0x10})
}

func TestConvertSelectionEncode(t *testing.T) {
	checkEncode(t, &ConvertSelectionRequest{Requestor: 0x10, Selection: 0xA, Target: 0xB, Property: 0xC, Time: 0x123},
		req(24, 0, cat(u32(0x10), u32(0xA), u32(0xB), u32(0xC), u32(0x123))))
}
