package base

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// XauthFamily constants (network byte order in file, but stored as uint16).
const (
	XauthFamilyLocal    uint16 = 256  // FamilyLocal (0x100 big-endian)
	XauthFamilyWild     uint16 = 65535
	XauthFamilyNetname  uint16 = 257
	XauthFamilyKernel   uint16 = 258
	XauthFamilyIPv4     uint16 = 260
	XauthFamilyIPv6     uint16 = 261
	XauthFamilyServerSI uint16 = 65534
)

// XauthEntry holds one parsed entry from an Xauthority file.
type XauthEntry struct {
	Family   uint16
	Address  []byte
	Number   string // display number as string, e.g. "0"
	Proto    string // auth protocol name, e.g. "MIT-MAGIC-COOKIE-1"
	Data     []byte // auth data (the cookie)
}

// ReadXauthFile reads and parses an Xauthority file, returning all entries.
// The file uses a fixed big-endian (network byte order) binary format.
func ReadXauthFile(path string) ([]XauthEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []XauthEntry
	for {
		e, err := readOneXauthEntry(f)
		if err == io.EOF {
			break
		}
		if err != nil {
			return entries, fmt.Errorf("xauth parse: %w", err)
		}
		entries = append(entries, *e)
	}
	return entries, nil
}

func readOneXauthEntry(r io.Reader) (*XauthEntry, error) {
	var e XauthEntry
	var u16 uint16

	// Family (2 bytes, big-endian)
	if err := binary.Read(r, binary.BigEndian, &u16); err != nil {
		return nil, err
	}
	e.Family = u16

	// Address (length-prefixed)
	if err := binary.Read(r, binary.BigEndian, &u16); err != nil {
		return nil, err
	}
	e.Address = make([]byte, u16)
	if _, err := io.ReadFull(r, e.Address); err != nil {
		return nil, err
	}

	// Number (length-prefixed string)
	if err := binary.Read(r, binary.BigEndian, &u16); err != nil {
		return nil, err
	}
	num := make([]byte, u16)
	if _, err := io.ReadFull(r, num); err != nil {
		return nil, err
	}
	e.Number = string(num)

	// Protocol name (length-prefixed string)
	if err := binary.Read(r, binary.BigEndian, &u16); err != nil {
		return nil, err
	}
	proto := make([]byte, u16)
	if _, err := io.ReadFull(r, proto); err != nil {
		return nil, err
	}
	e.Proto = string(proto)

	// Auth data (length-prefixed)
	if err := binary.Read(r, binary.BigEndian, &u16); err != nil {
		return nil, err
	}
	e.Data = make([]byte, u16)
	if _, err := io.ReadFull(r, e.Data); err != nil {
		return nil, err
	}

	return &e, nil
}

// XauthorityPath returns the path to the Xauthority file. If the XAUTHORITY
// environment variable is set, it is used; otherwise ~/.Xauthority.
func XauthorityPath() string {
	if p := os.Getenv("XAUTHORITY"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".Xauthority")
}

// LookupXauth finds the best matching auth entry for the given display.
// It reads the Xauthority file (from XAUTHORITY env or ~/.Xauthority) and
// returns the first entry whose family+address+number match the display.
//
// Matching rules (in priority order):
//   - FamilyWild matches any display (lowest priority)
//   - FamilyLocal matches when the address matches the host or is empty
//     and the number matches the display number
func LookupXauth(display DisplaySpec, entries []XauthEntry) *XauthEntry {
	numStr := fmt.Sprintf("%d", display.Display)
	host := display.Host

	// Two passes: first look for exact (non-Wild) matches, then Wild.
	var wild *XauthEntry
	for i := range entries {
		e := &entries[i]
		if e.Family == XauthFamilyWild && e.Number == numStr {
			if wild == nil {
				wild = e
			}
			continue
		}
		if e.Number != numStr {
			continue
		}
		switch e.Family {
		case XauthFamilyLocal:
			// FamilyLocal: address must be empty or match host
			if host == "" || host == "unix" || host == string(e.Address) {
				return e
			}
		}
	}
	return wild
}

// XauthFileOrEntries resolves an authority source from either an explicit file
// path or by reading the default XAUTHORITY. Returns nil entries and nil error
// when no file exists (no-auth is valid).
func XauthFileOrEntries(path string) ([]XauthEntry, error) {
	if path == "" {
		path = XauthorityPath()
	}
	if path == "" {
		return nil, nil
	}
	entries, err := ReadXauthFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no file = no auth, not an error
		}
		return nil, err
	}
	return entries, nil
}

// FormatAuthName returns a padded auth name suitable for the X11 handshake.
// The name is padded to a 4-byte boundary with NUL bytes.
func FormatAuthName(name string) []byte {
	b := make([]byte, pad4(len(name)))
	copy(b, name)
	return b
}

// FormatAuthData returns padded auth data suitable for the X11 handshake.
// The data is padded to a 4-byte boundary with NUL bytes.
func FormatAuthData(data []byte) []byte {
	b := make([]byte, pad4(len(data)))
	copy(b, data)
	return b
}

func pad4(n int) int {
	return (n + 3) &^ 3
}

// FormatXauthDisplay returns the display string suitable for xauth lookup
// from a DisplaySpec, e.g. ":0" or "hostname:0".
func FormatXauthDisplay(d DisplaySpec) string {
	if d.Host == "" || d.Host == "unix" {
		return fmt.Sprintf(":%d", d.Display)
	}
	return fmt.Sprintf("%s:%d", d.Host, d.Display)
}

// NormalizeAuthProto normalizes the protocol name for comparison.
func NormalizeAuthProto(s string) string {
	return strings.ToUpper(s)
}
