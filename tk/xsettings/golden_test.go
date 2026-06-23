package xsettings

import (
	"os"
	"testing"
)

// TestDecodeRealWorld decodes a _XSETTINGS_SETTINGS property captured from a
// live xfsettingsd (XFCE's settings daemon). The round-trip test exercises our
// own encode+decode, so a bug shared by both would slip through; this fixture
// pins our decoder against a real third-party producer instead.
func TestDecodeRealWorld(t *testing.T) {
	data, err := os.ReadFile("testdata/xfsettingsd.bin")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	s, err := decode(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(s.Items) != 30 {
		t.Errorf("items = %d, want 30", len(s.Items))
	}

	// integer setting (Xft/DPI is dpi*1024)
	if dpi, ok := s.DPI(); !ok || dpi != 96 {
		t.Errorf("DPI = %v,%v, want 96", dpi, ok)
	}
	if v, ok := s.Int("Gdk/WindowScalingFactor"); !ok || v != 1 {
		t.Errorf("Gdk/WindowScalingFactor = %v,%v, want 1", v, ok)
	}
	// string settings
	if fn, ok := s.FontName(); !ok || fn != "Sans 10" {
		t.Errorf("FontName = %q,%v, want \"Sans 10\"", fn, ok)
	}
	if tn, ok := s.ThemeName(); !ok || tn != "Xfce-kolors" {
		t.Errorf("ThemeName = %q,%v, want \"Xfce-kolors\"", tn, ok)
	}
	if mono, ok := s.String("Gtk/MonospaceFontName"); !ok || mono != "Monospace 10" {
		t.Errorf("Gtk/MonospaceFontName = %q,%v", mono, ok)
	}

	// every item must have a name and round-trip re-encode cleanly
	for _, it := range s.Items {
		if it.Name == "" {
			t.Errorf("decoded an item with empty name")
		}
	}
	if _, err := decode(encode(s.Serial, s.Items, false)); err != nil {
		t.Errorf("re-encode/decode of real-world settings failed: %v", err)
	}
}
