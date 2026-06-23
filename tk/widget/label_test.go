package widget

import "testing"

func TestLabelAlignX(t *testing.T) {
	const winW, textW = 100, 40
	cases := []struct {
		align Align
		pad   int
		want  int
	}{
		{AlignCenter, 0, 30}, // (100-40)/2
		{AlignLeft, 0, 2},    // default pad
		{AlignLeft, 8, 8},
		{AlignRight, 0, 58}, // 100-40-2
		{AlignRight, 10, 50},
	}
	for _, c := range cases {
		if got := alignX(c.align, winW, textW, c.pad); got != c.want {
			t.Errorf("alignX(%v, pad=%d) = %d, want %d", c.align, c.pad, got, c.want)
		}
	}
}

func TestLabelAlignDefaultIsCenter(t *testing.T) {
	// The zero value of Align must keep the original centred behaviour.
	if alignX(Align(0), 100, 40, 0) != alignX(AlignCenter, 100, 40, 0) {
		t.Error("zero Align is not AlignCenter")
	}
}
