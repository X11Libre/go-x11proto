package term

import (
	"testing"

	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

func TestEncodeKeyPrintable(t *testing.T) {
	got := EncodeKey(keyboard.Event{Rune: 'x'}, false)
	if string(got) != "x" {
		t.Errorf("got %q, want %q", got, "x")
	}
}

func TestEncodeKeyEnterIsCR(t *testing.T) {
	got := EncodeKey(keyboard.Event{Key: keyboard.KeyEnter}, false)
	if string(got) != "\r" {
		t.Errorf("got %q, want CR", got)
	}
}

func TestEncodeKeyBackspaceIsDEL(t *testing.T) {
	got := EncodeKey(keyboard.Event{Key: keyboard.KeyBackspace}, false)
	if len(got) != 1 || got[0] != 0x7f {
		t.Errorf("got %v, want [0x7f]", got)
	}
}

func TestEncodeKeyArrowsNormalVsAppCursor(t *testing.T) {
	cases := []struct {
		key     keyboard.Key
		normal  string
		appMode string
	}{
		{keyboard.KeyUp, "\x1b[A", "\x1bOA"},
		{keyboard.KeyDown, "\x1b[B", "\x1bOB"},
		{keyboard.KeyRight, "\x1b[C", "\x1bOC"},
		{keyboard.KeyLeft, "\x1b[D", "\x1bOD"},
	}
	for _, c := range cases {
		if got := string(EncodeKey(keyboard.Event{Key: c.key}, false)); got != c.normal {
			t.Errorf("%v normal = %q, want %q", c.key, got, c.normal)
		}
		if got := string(EncodeKey(keyboard.Event{Key: c.key}, true)); got != c.appMode {
			t.Errorf("%v app-cursor = %q, want %q", c.key, got, c.appMode)
		}
	}
}

func TestEncodeKeyCtrlLetters(t *testing.T) {
	cases := map[rune]byte{'a': 1, 'c': 3, 'z': 26, 'A': 1, 'C': 3}
	for r, want := range cases {
		got := EncodeKey(keyboard.Event{Rune: r, Ctrl: true}, false)
		if len(got) != 1 || got[0] != want {
			t.Errorf("Ctrl+%q = %v, want [0x%02x]", r, got, want)
		}
	}
}

func TestEncodeKeyCtrlBracket(t *testing.T) {
	got := EncodeKey(keyboard.Event{Rune: '[', Ctrl: true}, false)
	if len(got) != 1 || got[0] != 0x1b {
		t.Errorf("Ctrl+[ = %v, want ESC", got)
	}
}

func TestEncodeKeyTabAndShiftTab(t *testing.T) {
	if got := string(EncodeKey(keyboard.Event{Key: keyboard.KeyTab}, false)); got != "\t" {
		t.Errorf("Tab = %q, want tab", got)
	}
	if got := string(EncodeKey(keyboard.Event{Key: keyboard.KeyTab, Shift: true}, false)); got != "\x1b[Z" {
		t.Errorf("Shift+Tab = %q, want CSI Z", got)
	}
}

func TestEncodeKeyHomeEndPageUpDown(t *testing.T) {
	cases := []struct {
		key  keyboard.Key
		want string
	}{
		{keyboard.KeyHome, "\x1b[1~"},
		{keyboard.KeyEnd, "\x1b[4~"},
		{keyboard.KeyPageUp, "\x1b[5~"},
		{keyboard.KeyPageDown, "\x1b[6~"},
		{keyboard.KeyDelete, "\x1b[3~"},
	}
	for _, c := range cases {
		if got := string(EncodeKey(keyboard.Event{Key: c.key}, false)); got != c.want {
			t.Errorf("%v = %q, want %q", c.key, got, c.want)
		}
	}
}

func TestEncodeKeyUnrecognisedYieldsNil(t *testing.T) {
	// A pure modifier press or an event with neither a rune nor a logical key
	// (e.g. a bare Shift keydown) must produce no bytes.
	if got := EncodeKey(keyboard.Event{}, false); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestBracketPaste(t *testing.T) {
	if got := string(bracketPaste("hi", true)); got != "\x1b[200~hi\x1b[201~" {
		t.Errorf("got %q", got)
	}
	if got := string(bracketPaste("hi", false)); got != "hi" {
		t.Errorf("got %q, want unwrapped", got)
	}
}
