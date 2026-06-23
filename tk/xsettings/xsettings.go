// Package xsettings implements the XSETTINGS protocol: a property-based,
// server-mediated channel for distributing desktop theming settings (font DPI,
// font name, theme name, colours, ...) to all clients.
//
// A settings Manager owns the _XSETTINGS_S<screen> selection and stores the
// settings in a _XSETTINGS_SETTINGS property on its window; a Client finds the
// manager via the selection and reads/decodes that property. This is the
// mechanism GTK and Qt use, so values published here are seen by them and
// vice-versa.
package xsettings

import (
	"encoding/binary"
	"fmt"
)

// Type is the kind of an XSETTINGS value.
type Type uint8

const (
	TypeInteger Type = 0
	TypeString  Type = 1
	TypeColor   Type = 2
)

// Color is an XSETTINGS RGBA colour (16-bit channels).
type Color struct{ Red, Green, Blue, Alpha uint16 }

// Setting is one XSETTINGS entry. Only the field matching Type is meaningful.
type Setting struct {
	Name       string
	Type       Type
	Int        int32
	Str        string
	Color      Color
	LastChange uint32 // last-change serial
}

// Well-known setting names.
const (
	KeyDPI       = "Xft/DPI"      // integer, dpi * 1024
	KeyFontName  = "Gtk/FontName" // string, e.g. "Sans 10"
	KeyThemeName = "Net/ThemeName"
	KeyIconTheme = "Net/IconThemeName"
	KeyAntialias = "Xft/Antialias" // integer 0/1
	KeyHinting   = "Xft/Hinting"   // integer 0/1
	KeyRGBA      = "Xft/RGBA"      // string
)

// Settings is a decoded XSETTINGS database.
type Settings struct {
	Serial uint32
	Items  []Setting
	byName map[string]*Setting
}

// Get returns the named setting.
func (s *Settings) Get(name string) (*Setting, bool) { v, ok := s.byName[name]; return v, ok }

// Int returns an integer setting.
func (s *Settings) Int(name string) (int32, bool) {
	if v, ok := s.byName[name]; ok && v.Type == TypeInteger {
		return v.Int, true
	}
	return 0, false
}

// String returns a string setting.
func (s *Settings) String(name string) (string, bool) {
	if v, ok := s.byName[name]; ok && v.Type == TypeString {
		return v.Str, true
	}
	return "", false
}

// ColorOf returns a colour setting.
func (s *Settings) ColorOf(name string) (Color, bool) {
	if v, ok := s.byName[name]; ok && v.Type == TypeColor {
		return v.Color, true
	}
	return Color{}, false
}

// DPI returns the font DPI (Xft/DPI is stored as dpi*1024).
func (s *Settings) DPI() (float64, bool) {
	if v, ok := s.Int(KeyDPI); ok {
		return float64(v) / 1024.0, true
	}
	return 0, false
}

// FontName returns the configured UI font (Gtk/FontName), e.g. "Sans 10".
func (s *Settings) FontName() (string, bool) { return s.String(KeyFontName) }

// ThemeName returns the configured theme (Net/ThemeName).
func (s *Settings) ThemeName() (string, bool) { return s.String(KeyThemeName) }

func order(be bool) binary.ByteOrder {
	if be {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

// encode serialises settings into the _XSETTINGS_SETTINGS wire form, in the
// given byte order (which is recorded in the leading byte-order field).
func encode(serial uint32, items []Setting, be bool) []byte {
	o := order(be)
	var b []byte
	put8 := func(v byte) { b = append(b, v) }
	put16 := func(v uint16) { t := make([]byte, 2); o.PutUint16(t, v); b = append(b, t...) }
	put32 := func(v uint32) { t := make([]byte, 4); o.PutUint32(t, v); b = append(b, t...) }
	pad := func() {
		for len(b)%4 != 0 {
			b = append(b, 0)
		}
	}

	if be {
		put8(1) // byte-order: 0 = LSBFirst, otherwise MSBFirst
	} else {
		put8(0)
	}
	put8(0)
	put8(0)
	put8(0) // pad[3]
	put32(serial)
	put32(uint32(len(items)))

	for _, it := range items {
		put8(byte(it.Type))
		put8(0) // pad
		put16(uint16(len(it.Name)))
		b = append(b, it.Name...)
		pad()
		put32(it.LastChange)
		switch it.Type {
		case TypeInteger:
			put32(uint32(it.Int))
		case TypeString:
			put32(uint32(len(it.Str)))
			b = append(b, it.Str...)
			pad()
		case TypeColor:
			put16(it.Color.Red)
			put16(it.Color.Green)
			put16(it.Color.Blue)
			put16(it.Color.Alpha)
		}
	}
	return b
}

var errTrunc = fmt.Errorf("xsettings: truncated data")

// decode parses a _XSETTINGS_SETTINGS property value.
func decode(data []byte) (*Settings, error) {
	if len(data) < 12 {
		return nil, errTrunc
	}
	o := order(data[0] != 0) // 0 = LSBFirst, otherwise MSBFirst
	p := 4                   // skip byte-order + pad[3]
	rd16 := func() uint16 { v := o.Uint16(data[p : p+2]); p += 2; return v }
	rd32 := func() uint32 { v := o.Uint32(data[p : p+4]); p += 4; return v }

	serial := rd32()
	n := rd32()
	s := &Settings{Serial: serial, byName: map[string]*Setting{}}

	for i := uint32(0); i < n; i++ {
		if p+4 > len(data) {
			return nil, errTrunc
		}
		typ := Type(data[p])
		p += 2 // type + pad
		nl := int(rd16())
		if p+nl > len(data) {
			return nil, errTrunc
		}
		name := string(data[p : p+nl])
		p += nl
		p = (p + 3) &^ 3 // pad to 4
		if p+4 > len(data) {
			return nil, errTrunc
		}
		it := Setting{Name: name, Type: typ, LastChange: rd32()}
		switch typ {
		case TypeInteger:
			if p+4 > len(data) {
				return nil, errTrunc
			}
			it.Int = int32(rd32())
		case TypeString:
			if p+4 > len(data) {
				return nil, errTrunc
			}
			vl := int(rd32())
			if p+vl > len(data) {
				return nil, errTrunc
			}
			it.Str = string(data[p : p+vl])
			p += vl
			p = (p + 3) &^ 3
		case TypeColor:
			if p+8 > len(data) {
				return nil, errTrunc
			}
			it.Color = Color{rd16(), rd16(), rd16(), rd16()}
		default:
			return nil, fmt.Errorf("xsettings: unknown setting type %d", typ)
		}
		s.Items = append(s.Items, it)
	}
	for i := range s.Items {
		s.byName[s.Items[i].Name] = &s.Items[i]
	}
	return s, nil
}
