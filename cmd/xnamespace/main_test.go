package main

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/ext/namespace"
)

func TestParseGlobalFlags(t *testing.T) {
	for _, f := range []string{"-s", "--short"} {
		short = false
		got := parseGlobalFlags([]string{f, "query", "ns"})
		if !short {
			t.Fatalf("%s did not set short", f)
		}
		if len(got) != 2 || got[0] != "query" || got[1] != "ns" {
			t.Fatalf("%s not stripped cleanly: %v", f, got)
		}
	}

	short = false
	if got := parseGlobalFlags([]string{"list"}); len(got) != 1 || got[0] != "list" || short {
		t.Fatalf("unexpected: args=%v short=%v", got, short)
	}
}

func TestCSV(t *testing.T) {
	if got := csv(capList(namespace.CapMouseMotion | namespace.CapShape)); got != "mouse,shape" {
		t.Fatalf("csv = %q, want %q", got, "mouse,shape")
	}
	if got := csv(capList(0)); got != "" {
		t.Fatalf("csv(empty) = %q, want empty string (no \"(none)\")", got)
	}
}
