package xts

import (
	"testing"
	"time"

	"github.com/X11Libre/go-x11proto/tk/xsettings"
)

// TestXSettings round-trips settings through the server: a Manager acquires the
// _XSETTINGS_S0 selection and publishes, a Client finds it and reads them back.
func TestXSettings(t *testing.T) {
	c := connect(t)
	defer c.Close()

	mgr, err := xsettings.NewManager(c, 0)
	must(t, err, "NewManager")
	defer mgr.Close()

	must(t, mgr.Set([]xsettings.Setting{
		{Name: xsettings.KeyDPI, Type: xsettings.TypeInteger, Int: 96 * 1024},
		{Name: xsettings.KeyFontName, Type: xsettings.TypeString, Str: "Sans 11"},
		{Name: xsettings.KeyThemeName, Type: xsettings.TypeString, Str: "Adwaita"},
		{Name: "MyColor", Type: xsettings.TypeColor,
			Color: xsettings.Color{Red: 0x1122, Green: 0x3344, Blue: 0x5566, Alpha: 0xffff}},
	}), "Manager.Set")

	cl, err := xsettings.NewClient(c, 0)
	must(t, err, "NewClient")

	if w, err := cl.ManagerWindow(); err != nil || w != mgr.Window() {
		t.Fatalf("ManagerWindow = %d,%v; want %d", w, err, mgr.Window())
	}

	s, err := cl.Get()
	must(t, err, "Client.Get")
	if s == nil {
		t.Fatal("Get returned no settings")
	}
	if dpi, ok := s.DPI(); !ok || dpi != 96 {
		t.Errorf("DPI = %v,%v; want 96", dpi, ok)
	}
	if fn, ok := s.FontName(); !ok || fn != "Sans 11" {
		t.Errorf("FontName = %q,%v; want \"Sans 11\"", fn, ok)
	}
	if tn, ok := s.ThemeName(); !ok || tn != "Adwaita" {
		t.Errorf("ThemeName = %q,%v; want \"Adwaita\"", tn, ok)
	}
	if col, ok := s.ColorOf("MyColor"); !ok || col.Red != 0x1122 || col.Alpha != 0xffff {
		t.Errorf("Color = %+v,%v", col, ok)
	}
}

// TestXSettingsWatch verifies the client's live watch: changing a setting on
// the manager fires the client's callback with the new value.
func TestXSettingsWatch(t *testing.T) {
	c := connect(t)
	defer c.Close()

	mgr, err := xsettings.NewManager(c, 0)
	must(t, err, "NewManager")
	defer mgr.Close()
	must(t, mgr.Set([]xsettings.Setting{
		{Name: xsettings.KeyDPI, Type: xsettings.TypeInteger, Int: 96 * 1024},
	}), "Set initial")

	cl, err := xsettings.NewClient(c, 0)
	must(t, err, "NewClient")
	got := make(chan float64, 8)
	must(t, cl.Watch(func(s *xsettings.Settings) {
		if d, ok := s.DPI(); ok {
			got <- d
		}
	}), "Watch")

	// Watch fires once immediately with the current value
	select {
	case d := <-got:
		if d != 96 {
			t.Errorf("initial DPI = %v, want 96", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no initial Watch callback")
	}

	// change it; pump events (acting as the event loop) until the callback reports 120
	must(t, mgr.Set([]xsettings.Setting{
		{Name: xsettings.KeyDPI, Type: xsettings.TypeInteger, Int: 120 * 1024},
	}), "Set updated")

	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-c.Events():
			c.DeliverWindowEvent(ev)
		case d := <-got:
			if d == 120 {
				return // watch fired with the updated value
			}
		case <-deadline:
			t.Fatal("watch callback not fired for the DPI change")
		}
	}
}
