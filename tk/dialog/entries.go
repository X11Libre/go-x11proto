// Package dialog provides reusable, self-contained dialog windows built on the
// tk toolkit. The first is FilePicker, a keyboard- and mouse-driven file open
// chooser.
package dialog

import (
	"os"
	"path/filepath"
	"sort"
)

// entry is one row of a directory listing.
type entry struct {
	name  string
	isDir bool
}

// display is how the entry is shown (directories get a trailing slash).
func (e entry) display() string {
	if e.isDir && e.name != ".." {
		return e.name + "/"
	}
	return e.name
}

// arrange orders a raw listing for display: ".." first (unless dir is the
// filesystem root), then directories, then files, each group sorted by name.
func arrange(raw []entry, dir string) []entry {
	var dirs, files []entry
	for _, e := range raw {
		if e.name == "." || e.name == ".." {
			continue // synthesised below
		}
		if e.isDir {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	byName := func(s []entry) func(i, j int) bool {
		return func(i, j int) bool { return s[i].name < s[j].name }
	}
	sort.Slice(dirs, byName(dirs))
	sort.Slice(files, byName(files))

	out := make([]entry, 0, len(dirs)+len(files)+1)
	if filepath.Clean(dir) != "/" {
		out = append(out, entry{name: "..", isDir: true})
	}
	out = append(out, dirs...)
	out = append(out, files...)
	return out
}

// readDir reads dir and returns its arranged listing.
func readDir(dir string) ([]entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	raw := make([]entry, len(des))
	for i, de := range des {
		raw[i] = entry{name: de.Name(), isDir: de.IsDir()}
	}
	return arrange(raw, dir), nil
}

// target resolves activating the named entry within dir: ".." goes to the
// parent, anything else joins onto dir.
func target(dir, name string) string {
	if name == ".." {
		return filepath.Dir(filepath.Clean(dir))
	}
	return filepath.Join(dir, name)
}

// clampSel keeps a selection index within [0, n) (or 0 when the list is empty).
func clampSel(i, n int) int {
	if n <= 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

// scrollTop adjusts the first visible row so that sel is on screen given rows
// visible rows.
func scrollTop(top, sel, rows int) int {
	if rows < 1 {
		rows = 1
	}
	if sel < top {
		return sel
	}
	if sel >= top+rows {
		return sel - rows + 1
	}
	return top
}
