package widget

import "testing"

func TestBorderLayoutFull(t *testing.T) {
	// 200x100 frame: top 20, bottom 10, right 16, no left.
	top, bottom, left, right, center := computeBorderLayout(200, 100, 20, 10, 0, 16)

	if top != (rect{0, 0, 200, 20}) {
		t.Errorf("top = %+v", top)
	}
	if bottom != (rect{0, 90, 200, 10}) {
		t.Errorf("bottom = %+v", bottom)
	}
	if left != (rect{0, 20, 0, 70}) {
		t.Errorf("left = %+v", left)
	}
	if right != (rect{184, 20, 16, 70}) {
		t.Errorf("right = %+v", right)
	}
	// center fills the middle, left of the scrollbar
	if center != (rect{0, 20, 184, 70}) {
		t.Errorf("center = %+v", center)
	}
}

func TestBorderLayoutCenterOnly(t *testing.T) {
	_, _, _, _, center := computeBorderLayout(80, 60, 0, 0, 0, 0)
	if center != (rect{0, 0, 80, 60}) {
		t.Errorf("center = %+v, want full frame", center)
	}
}

func TestBorderLayoutCoversWidth(t *testing.T) {
	// left + center + right must tile the width exactly with no gap/overlap.
	_, _, left, right, center := computeBorderLayout(300, 200, 0, 0, 24, 16)
	if left.W+center.W+right.W != 300 {
		t.Errorf("widths %d+%d+%d != 300", left.W, center.W, right.W)
	}
	if left.X != 0 || center.X != left.W || right.X != left.W+center.W {
		t.Errorf("x positions don't tile: left=%d center=%d right=%d", left.X, center.X, right.X)
	}
}

func TestBorderLayoutVerticalTiles(t *testing.T) {
	top, bottom, _, _, center := computeBorderLayout(100, 100, 15, 25, 0, 0)
	if top.H+center.H+bottom.H != 100 {
		t.Errorf("heights %d+%d+%d != 100", top.H, center.H, bottom.H)
	}
	if center.Y != top.H || bottom.Y != top.H+center.H {
		t.Errorf("y positions don't tile: center=%d bottom=%d", center.Y, bottom.Y)
	}
}

func TestBorderLayoutOversizedBars(t *testing.T) {
	// Bars larger than the frame must clamp, never producing negative sizes.
	top, bottom, _, _, center := computeBorderLayout(50, 30, 100, 100, 0, 0)
	if top.H < 0 || bottom.H < 0 || center.H < 0 {
		t.Errorf("negative height: top=%d bottom=%d center=%d", top.H, bottom.H, center.H)
	}
	if top.H+center.H+bottom.H != 30 {
		t.Errorf("clamped heights must still sum to 30, got %d", top.H+center.H+bottom.H)
	}
}
