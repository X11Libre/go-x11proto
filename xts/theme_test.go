package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/tk/theme"
	"github.com/X11Libre/go-x11proto/tk/xsettings"
)

// TestTheme verifies theme.Load picks up XSETTINGS and scales: a manager
// publishes Xft/DPI=192 and a 12pt font, and Load reflects it.
func TestTheme(t *testing.T) {
	c := connect(t)
	defer c.Close()

	mgr, err := xsettings.NewManager(c, 0)
	must(t, err, "NewManager")
	defer mgr.Close()
	must(t, mgr.Set([]xsettings.Setting{
		{Name: xsettings.KeyDPI, Type: xsettings.TypeInteger, Int: 192 * 1024},
		{Name: xsettings.KeyFontName, Type: xsettings.TypeString, Str: "DejaVu Sans 12"},
	}), "Set")

	th := theme.Load(c)
	if th.DPI != 192 {
		t.Errorf("DPI = %v, want 192", th.DPI)
	}
	if th.Family != "DejaVu Sans" || th.PointSize != 12 {
		t.Errorf("font = %q / %vpt, want \"DejaVu Sans\" / 12", th.Family, th.PointSize)
	}
	if px := th.FontPixelSize(); px != 32 { // 12 * 192/72 = 32
		t.Errorf("FontPixelSize = %d, want 32", px)
	}
	if _, err := th.OpenFont(c); err != nil { // themed or "fixed" fallback
		t.Errorf("OpenFont: %v", err)
	}
}
