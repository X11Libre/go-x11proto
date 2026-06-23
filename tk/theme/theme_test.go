package theme

import "testing"

func TestParseFontName(t *testing.T) {
	cases := []struct {
		in     string
		family string
		pt     float64
	}{
		{"Sans 10", "Sans", 10},
		{"DejaVu Sans Bold 12", "DejaVu Sans Bold", 12},
		{"Monospace 10", "Monospace", 10},
		{"NoSize", "NoSize", 10},
		{"", "Sans", 10},
	}
	for _, c := range cases {
		fam, pt := parseFontName(c.in)
		if fam != c.family || pt != c.pt {
			t.Errorf("parseFontName(%q) = (%q,%v); want (%q,%v)", c.in, fam, pt, c.family, c.pt)
		}
	}
}

func TestPointsToPixels(t *testing.T) {
	// 10pt @ 96dpi = 13.33 -> 13; 10pt @ 192dpi = 26.66 -> 27
	t96 := &Theme{DPI: 96, PointSize: 10}
	if px := t96.FontPixelSize(); px != 13 {
		t.Errorf("96dpi 10pt = %d px, want 13", px)
	}
	t192 := &Theme{DPI: 192, PointSize: 10}
	if px := t192.FontPixelSize(); px != 27 {
		t.Errorf("192dpi 10pt = %d px, want 27", px)
	}
}
