package base

import "testing"

func TestSetExtOpcode(t *testing.T) {
	w := MakeRequestWriter(false)
	w.SetExtOpcode(0x80, 5) // major 128, minor 5
	w.WriteCARD32(0xAABBCCDD)
	got := w.ToBytes()

	if got[0] != 0x80 {
		t.Errorf("major opcode = %d, want 128", got[0])
	}
	if got[1] != 5 {
		t.Errorf("minor opcode (param0) = %d, want 5", got[1])
	}
	// length field (units) and payload
	units := uint16(got[2]) | uint16(got[3])<<8
	if int(units)*4 != len(got) {
		t.Errorf("length field %d units != %d bytes", units, len(got))
	}
	want := []byte{0x80, 5, 2, 0, 0xDD, 0xCC, 0xBB, 0xAA}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestSetMinorOpcode(t *testing.T) {
	w := MakeRequestWriter(false)
	w.SetOpcode(130)
	w.SetMinorOpcode(7)
	got := w.ToBytes()
	if got[0] != 130 || got[1] != 7 {
		t.Errorf("got opcode=%d minor=%d, want 130/7", got[0], got[1])
	}
}
