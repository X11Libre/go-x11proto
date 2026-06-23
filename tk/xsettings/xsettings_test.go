package xsettings

import "testing"

func TestCodecRoundTrip(t *testing.T) {
	in := []Setting{
		{Name: "Xft/DPI", Type: TypeInteger, Int: 98304, LastChange: 1},
		{Name: "Gtk/FontName", Type: TypeString, Str: "Sans 11", LastChange: 2},
		{Name: "Net/ThemeName", Type: TypeString, Str: "Adwaita", LastChange: 3},
		{Name: "MyColor", Type: TypeColor, Color: Color{0x1122, 0x3344, 0x5566, 0xffff}, LastChange: 4},
	}
	for _, be := range []bool{false, true} {
		s, err := decode(encode(7, in, be))
		if err != nil {
			t.Fatalf("be=%v: %v", be, err)
		}
		if s.Serial != 7 {
			t.Errorf("be=%v: serial = %d, want 7", be, s.Serial)
		}
		if len(s.Items) != len(in) {
			t.Fatalf("be=%v: %d items, want %d", be, len(s.Items), len(in))
		}
		if dpi, ok := s.DPI(); !ok || dpi != 96 {
			t.Errorf("be=%v: DPI = %v,%v want 96", be, dpi, ok)
		}
		if fn, ok := s.FontName(); !ok || fn != "Sans 11" {
			t.Errorf("be=%v: FontName = %q,%v", be, fn, ok)
		}
		if c, ok := s.ColorOf("MyColor"); !ok || c != (Color{0x1122, 0x3344, 0x5566, 0xffff}) {
			t.Errorf("be=%v: Color = %+v,%v", be, c, ok)
		}
	}
}

func TestDecodeShort(t *testing.T) {
	if _, err := decode([]byte{0, 0, 0}); err == nil {
		t.Error("expected error on short data")
	}
}
