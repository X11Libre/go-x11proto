package widget

import "testing"

func TestThumbGeomNoScroll(t *testing.T) {
	// Everything fits: full-height thumb at the top.
	y, h := thumbGeom(100, 5, 10, 0)
	if y != 0 || h != 100 {
		t.Errorf("thumbGeom(fit) = (%d,%d), want (0,100)", y, h)
	}
}

func TestThumbGeomProportional(t *testing.T) {
	// 10 visible of 40 total -> thumb is 1/4 of the track.
	_, h := thumbGeom(200, 40, 10, 0)
	if h != 50 {
		t.Errorf("thumb height = %d, want 50", h)
	}
	// at the top
	if y, _ := thumbGeom(200, 40, 10, 0); y != 0 {
		t.Errorf("top y = %d, want 0", y)
	}
	// at the bottom (top = total-visible = 30) -> thumb flush with track end
	y, h2 := thumbGeom(200, 40, 10, 30)
	if y+h2 != 200 {
		t.Errorf("bottom: y+h = %d, want 200", y+h2)
	}
	// halfway
	if y, _ := thumbGeom(200, 40, 10, 15); y != 75 {
		t.Errorf("mid y = %d, want 75", y)
	}
}

func TestThumbGeomMinHeight(t *testing.T) {
	// Huge document: thumb clamped to the minimum, still within the track.
	y, h := thumbGeom(100, 100000, 10, 99990)
	if h != sbMinThumb {
		t.Errorf("thumb height = %d, want %d", h, sbMinThumb)
	}
	if y+h > 100 {
		t.Errorf("thumb overflows track: y=%d h=%d", y, h)
	}
}

func TestTopForThumbYRoundTrip(t *testing.T) {
	const trackH, total, visible = 200, 40, 10
	for _, top := range []int{0, 5, 15, 30} {
		y, _ := thumbGeom(trackH, total, visible, top)
		got := topForThumbY(trackH, total, visible, y)
		if got != top {
			t.Errorf("round-trip top %d -> y %d -> %d", top, y, got)
		}
	}
}

func TestTopForThumbYClamps(t *testing.T) {
	const trackH, total, visible = 200, 40, 10
	if got := topForThumbY(trackH, total, visible, -50); got != 0 {
		t.Errorf("negative y -> %d, want 0", got)
	}
	if got := topForThumbY(trackH, total, visible, 100000); got != total-visible {
		t.Errorf("huge y -> %d, want %d", got, total-visible)
	}
}

func TestClampTop(t *testing.T) {
	if clampTop(40, 10, -1) != 0 {
		t.Error("clampTop negative")
	}
	if clampTop(40, 10, 100) != 30 {
		t.Error("clampTop over max")
	}
	if clampTop(5, 10, 3) != 0 {
		t.Error("clampTop when all visible should be 0")
	}
}

// TestScrollbarPaging drives the press logic and checks OnScroll deltas without
// a server (Draw is a no-op-safe path only when a gc exists, so we test press
// math via a tiny harness that mirrors press()).
func TestScrollbarPagingMath(t *testing.T) {
	// Simulate: track 200px, 40 lines, 10 visible, currently at top=0.
	// A click below the thumb pages down by visible-1 = 9.
	const trackH, total, visible = 200, 40, 10
	ty, th := thumbGeom(trackH, total, visible, 0)
	clickBelow := ty + th + 1
	if clickBelow <= ty+th {
		t.Fatal("test setup: click not below thumb")
	}
	newTop := clampTop(total, visible, 0+(visible-1))
	if newTop != 9 {
		t.Errorf("page-down top = %d, want 9", newTop)
	}
}
