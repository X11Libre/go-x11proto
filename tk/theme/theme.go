// Package theme reads desktop theming hints (font DPI and UI font) from
// XSETTINGS and turns them into concrete pixel sizes / fonts, so a toolkit can
// scale with the running desktop instead of using fixed pixel sizes.
package theme

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"github.com/X11Libre/go-x11proto/tk/xsettings"
)

// Defaults used when no XSETTINGS manager is running.
const (
	DefaultDPI       = 96.0
	DefaultFontName  = "Sans 10"
	DefaultFamily    = "Sans"
	DefaultPointSize = 10.0
)

// Theme is the resolved desktop theming relevant to text sizing.
type Theme struct {
	DPI          float64 // Xft/DPI (dots per inch)
	FontName     string  // raw Gtk/FontName, e.g. "Sans 10"
	Family       string  // parsed family, e.g. "Sans"
	PointSize    float64 // parsed point size, e.g. 10
	TearOffMenus bool    // Gtk/MenuTearoff: menus are detachable
}

// Load reads the theme from XSETTINGS for screen 0, falling back to the defaults
// (96 dpi, "Sans 10") when no manager is present or a value is missing.
func Load(conn *core.X11Conn) *Theme {
	t := &Theme{DPI: DefaultDPI, FontName: DefaultFontName, Family: DefaultFamily, PointSize: DefaultPointSize}
	cl, err := xsettings.NewClient(conn, 0)
	if err != nil {
		return t
	}
	s, err := cl.Get()
	if err != nil || s == nil {
		return t
	}
	if d, ok := s.DPI(); ok && d > 0 {
		t.DPI = d
	}
	if fn, ok := s.FontName(); ok && fn != "" {
		t.FontName = fn
		t.Family, t.PointSize = parseFontName(fn)
	}
	if v, ok := s.Int(xsettings.KeyMenuTearoff); ok {
		t.TearOffMenus = v != 0
	}
	return t
}

// PointsToPixels converts a point size to pixels at the theme's DPI.
func (t *Theme) PointsToPixels(pt float64) int {
	return int(pt*t.DPI/72.0 + 0.5)
}

// FontPixelSize is the UI font's height in pixels at the theme's DPI.
func (t *Theme) FontPixelSize() int {
	return t.PointsToPixels(t.PointSize)
}

// OpenFont opens a core X font at the theme's pixel size (a scalable XLFD at
// that size, any family), falling back to "fixed" if the server has no match.
// Core fonts have no notion of fontconfig families like "Sans", so this matches
// the size, not the exact face.
func (t *Theme) OpenFont(conn *core.X11Conn) (base.FONT, error) {
	px := t.FontPixelSize()
	if px > 0 {
		xlfd := fmt.Sprintf("-*-*-*-r-*--%d-*-*-*-*-*-*-*", px)
		if f, err := rpc.OpenFont(conn, xlfd); err == nil {
			return f, nil
		}
	}
	return rpc.OpenFont(conn, "fixed")
}

// String summarises the theme for logging.
func (t *Theme) String() string {
	return fmt.Sprintf("DPI=%.0f font=%q (%s %.0fpt -> %dpx) tearoff=%t",
		t.DPI, t.FontName, t.Family, t.PointSize, t.FontPixelSize(), t.TearOffMenus)
}

// parseFontName splits a Pango/Gtk font string ("Sans 10", "DejaVu Sans Bold
// 12") into a family and a point size; a missing trailing size defaults to 10.
func parseFontName(s string) (family string, points float64) {
	points = DefaultPointSize
	fields := strings.Fields(s)
	if n := len(fields); n > 0 {
		if v, err := strconv.ParseFloat(fields[n-1], 64); err == nil && v > 0 {
			points = v
			fields = fields[:n-1]
		}
	}
	family = strings.Join(fields, " ")
	if family == "" {
		family = DefaultFamily
	}
	return family, points
}
