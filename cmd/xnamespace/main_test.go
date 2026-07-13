package main

import (
	"fmt"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/ext/namespace"
)

func resetFlags() {
	short = false
	authorityFile = ""
	noXauth = false
	inlineAuthEntries = nil
	pendingAuthProto = ""
}

func TestParseGlobalFlags(t *testing.T) {
	for _, f := range []string{"-s", "--short"} {
		resetFlags()
		got := parseGlobalFlags([]string{f, "query", "ns"})
		if !short {
			t.Fatalf("%s did not set short", f)
		}
		if len(got) != 2 || got[0] != "query" || got[1] != "ns" {
			t.Fatalf("%s not stripped cleanly: %v", f, got)
		}
	}

	resetFlags()
	if got := parseGlobalFlags([]string{"list"}); len(got) != 1 || got[0] != "list" || short {
		t.Fatalf("unexpected: args=%v short=%v", got, short)
	}
}

func TestParseGlobalFlagsAuthority(t *testing.T) {
	resetFlags()
	got := parseGlobalFlags([]string{"-a", "/tmp/test.xauth", "list"})
	if authorityFile != "/tmp/test.xauth" {
		t.Fatalf("-a did not set authorityFile: %q", authorityFile)
	}
	if len(got) != 1 || got[0] != "list" {
		t.Fatalf("args not stripped: %v", got)
	}
}

func TestParseGlobalFlagsNoXauth(t *testing.T) {
	resetFlags()
	got := parseGlobalFlags([]string{"--no-xauth", "list"})
	if !noXauth {
		t.Fatal("--no-xauth did not set noXauth")
	}
	if len(got) != 1 || got[0] != "list" {
		t.Fatalf("args not stripped: %v", got)
	}
}

func TestParseGlobalFlagsAuthEntry(t *testing.T) {
	resetFlags()
	got := parseGlobalFlags([]string{
		"-A", "MIT-MAGIC-COOKIE-1", "--auth-data", "deadbeef",
		"list",
	})
	if len(inlineAuthEntries) != 1 {
		t.Fatalf("expected 1 inline entry, got %d", len(inlineAuthEntries))
	}
	if inlineAuthEntries[0].proto != "MIT-MAGIC-COOKIE-1" {
		t.Fatalf("wrong proto: %q", inlineAuthEntries[0].proto)
	}
	if inlineAuthEntries[0].dataHex != "deadbeef" {
		t.Fatalf("wrong dataHex: %q", inlineAuthEntries[0].dataHex)
	}
	if len(got) != 1 || got[0] != "list" {
		t.Fatalf("args not stripped: %v", got)
	}
}

func TestParseGlobalFlagsMultipleAuthEntries(t *testing.T) {
	resetFlags()
	got := parseGlobalFlags([]string{
		"-A", "MIT-MAGIC-COOKIE-1", "--auth-data", "deadbeef",
		"-A", "MIT-MAGIC-COOKIE-2", "--auth-data", "faceface",
		"list",
	})
	if len(inlineAuthEntries) != 2 {
		t.Fatalf("expected 2 inline entries, got %d", len(inlineAuthEntries))
	}
	if inlineAuthEntries[0].proto != "MIT-MAGIC-COOKIE-1" || inlineAuthEntries[1].proto != "MIT-MAGIC-COOKIE-2" {
		t.Fatalf("wrong protos: %q, %q", inlineAuthEntries[0].proto, inlineAuthEntries[1].proto)
	}
	if len(got) != 1 || got[0] != "list" {
		t.Fatalf("args not stripped: %v", got)
	}
}

func TestParseGlobalFlagsAuthDataWithoutProto(t *testing.T) {
	resetFlags()
	var fatalMsg string
	origFatal := fatal
	fatal = func(format string, a ...any) {
		fatalMsg = fmt.Sprintf(format, a...)
		panic("fatal")
	}
	defer func() { fatal = origFatal }()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected fatal for --auth-data without --auth-proto")
		}
		if fatalMsg != "--auth-data requires a preceding --auth-proto" {
			t.Fatalf("wrong fatal message: %q", fatalMsg)
		}
	}()
	parseGlobalFlags([]string{"--auth-data", "deadbeef", "list"})
}

func TestParseGlobalFlagsProtoWithoutData(t *testing.T) {
	resetFlags()
	var fatalMsg string
	origFatal := fatal
	fatal = func(format string, a ...any) {
		fatalMsg = fmt.Sprintf(format, a...)
		panic("fatal")
	}
	defer func() { fatal = origFatal }()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected fatal for --auth-proto without --auth-data")
		}
		if fatalMsg != `--auth-proto "MIT-MAGIC-COOKIE-1" requires a following --auth-data` {
			t.Fatalf("wrong fatal message: %q", fatalMsg)
		}
	}()
	parseGlobalFlags([]string{"-A", "MIT-MAGIC-COOKIE-1", "list"})
}

func TestCSV(t *testing.T) {
	if got := csv(capList(namespace.CapMouseMotion | namespace.CapShape)); got != "mouse,shape" {
		t.Fatalf("csv = %q, want %q", got, "mouse,shape")
	}
	if got := csv(capList(0)); got != "" {
		t.Fatalf("csv(empty) = %q, want empty string (no \"(none)\")", got)
	}
}
