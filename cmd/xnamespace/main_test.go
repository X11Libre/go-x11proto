package main

import (
	"encoding/json"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/ext/namespace"
)

func TestInfoJSONShape(t *testing.T) {
	it := namespace.Info{
		Name:         "demo",
		Capabilities: namespace.CapMouseMotion | namespace.CapShape,
		Attributes:   namespace.AttrTransient,
		Refcnt:       3,
		NumTokens:    2,
	}
	b, err := json.Marshal(infoJSON(it))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"demo","capabilities":["mouse","shape"],"attributes":["transient"],"clients":3,"tokens":2}`
	if got := string(b); got != want {
		t.Fatalf("infoJSON:\n got %s\nwant %s", got, want)
	}
}

// An empty capability/attribute set must marshal to [] (not null), so consumers
// can iterate without a nil check.
func TestEmptyListsAreArrays(t *testing.T) {
	b, err := json.Marshal(infoJSON(namespace.Info{Name: "empty"}))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"empty","capabilities":[],"attributes":[],"clients":0,"tokens":0}`
	if got := string(b); got != want {
		t.Fatalf("empty info:\n got %s\nwant %s", got, want)
	}
}

func TestTokenJSONShape(t *testing.T) {
	b, err := json.Marshal(jsonToken{Handle: 0x2a, Proto: "MIT-MAGIC-COOKIE-1"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"handle":42,"proto":"MIT-MAGIC-COOKIE-1"}`
	if got := string(b); got != want {
		t.Fatalf("jsonToken:\n got %s\nwant %s", got, want)
	}
}

func TestParseGlobalFlags(t *testing.T) {
	jsonOut = false
	got := parseGlobalFlags([]string{"-json", "create", "foo", "mouse"})
	if !jsonOut {
		t.Fatal("-json did not set jsonOut")
	}
	if len(got) != 3 || got[0] != "create" || got[1] != "foo" || got[2] != "mouse" {
		t.Fatalf("flag not stripped cleanly: %v", got)
	}

	jsonOut = false
	if got := parseGlobalFlags([]string{"list"}); len(got) != 1 || got[0] != "list" || jsonOut {
		t.Fatalf("unexpected: args=%v jsonOut=%v", got, jsonOut)
	}
}
