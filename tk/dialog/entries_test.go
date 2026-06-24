package dialog

import (
	"reflect"
	"testing"
)

func names(es []entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.display()
	}
	return out
}

func TestArrangeOrdersDirsThenFiles(t *testing.T) {
	raw := []entry{
		{name: "zeta.txt"},
		{name: "src", isDir: true},
		{name: "alpha.txt"},
		{name: "bin", isDir: true},
		{name: ".", isDir: true},  // dropped
		{name: "..", isDir: true}, // synthesised
	}
	got := names(arrange(raw, "/home/u"))
	want := []string{"../", "bin/", "src/", "alpha.txt", "zeta.txt"}
	// ".." has no trailing slash by display() rule
	want[0] = ".."
	if !reflect.DeepEqual(got, want) {
		t.Errorf("arrange = %v, want %v", got, want)
	}
}

func TestArrangeRootHasNoParent(t *testing.T) {
	got := arrange([]entry{{name: "etc", isDir: true}}, "/")
	if len(got) != 1 || got[0].name != "etc" {
		t.Errorf("root listing = %v, want just etc", names(got))
	}
}

func TestTarget(t *testing.T) {
	cases := []struct{ dir, name, want string }{
		{"/home/u", "..", "/home"},
		{"/home/u", "docs", "/home/u/docs"},
		{"/", "..", "/"},
		{"/home/u/", "..", "/home"}, // trailing slash normalised
	}
	for _, c := range cases {
		if got := target(c.dir, c.name); got != c.want {
			t.Errorf("target(%q,%q) = %q, want %q", c.dir, c.name, got, c.want)
		}
	}
}

func TestClampSel(t *testing.T) {
	if clampSel(-3, 5) != 0 {
		t.Error("negative should clamp to 0")
	}
	if clampSel(9, 5) != 4 {
		t.Error("over-range should clamp to n-1")
	}
	if clampSel(2, 0) != 0 {
		t.Error("empty list should be 0")
	}
}

func TestScrollTop(t *testing.T) {
	// sel above the window scrolls up to sel.
	if got := scrollTop(5, 2, 4); got != 2 {
		t.Errorf("scroll up = %d, want 2", got)
	}
	// sel below the window scrolls so sel is the last visible row.
	if got := scrollTop(0, 10, 4); got != 7 {
		t.Errorf("scroll down = %d, want 7", got)
	}
	// sel already visible keeps top.
	if got := scrollTop(3, 4, 4); got != 3 {
		t.Errorf("in-view = %d, want 3", got)
	}
}

func TestReadDirLive(t *testing.T) {
	// readDir against a real temp tree (no server needed).
	dir := t.TempDir()
	mustWrite(t, dir+"/b.txt")
	mustWrite(t, dir+"/a.txt")
	mustMkdir(t, dir+"/sub")
	es, err := readDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := names(es)
	want := []string{"..", "sub/", "a.txt", "b.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readDir = %v, want %v", got, want)
	}
}
