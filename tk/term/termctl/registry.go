package termctl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// registry maps a caller-chosen terminal name to the path of its control
// pipe. It is process-local state used by Open(); multiple starfleetctl
// processes share it through the on-disk index file below.
var (
	regMu      sync.Mutex
	regIndex   = defaultRegistryPath()
	regEntries = map[string]string{}
	regLoaded  bool
)

// defaultRegistryPath returns the index file location. The work temp dir is
// preferred (see skill "use-work-tempdir"); fall back to /tmp.
func defaultRegistryPath() string {
	if d := os.Getenv("MPBT_WORK_TMPDIR"); d != "" {
		return filepath.Join(d, "termctl-registry.txt")
	}
	return filepath.Join(os.TempDir(), "termctl-registry.txt")
}

// SetRegistryPath overrides the on-disk index location. Call before New/Open.
func SetRegistryPath(path string) {
	regMu.Lock()
	defer regMu.Unlock()
	regIndex = path
	regLoaded = false
}

func loadRegistry() {
	if regLoaded {
		return
	}
	regEntries = map[string]string{}
	if data, err := os.ReadFile(regIndex); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				regEntries[parts[0]] = parts[1]
			}
		}
	}
	regLoaded = true
}

func saveRegistry() {
	var b strings.Builder
	b.WriteString("# termctl registry: name=pipepath\n")
	for k, v := range regEntries {
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	_ = os.MkdirAll(filepath.Dir(regIndex), 0o755)
	_ = os.WriteFile(regIndex, []byte(b.String()), 0o644)
}

// register records name -> pipePath and persists it. It returns an error if
// the name is already taken by a different pipe.
func register(name, pipePath string) error {
	regMu.Lock()
	defer regMu.Unlock()
	loadRegistry()
	if old, ok := regEntries[name]; ok && old != pipePath {
		return fmt.Errorf("termctl: name %q already registered to %q", name, old)
	}
	regEntries[name] = pipePath
	saveRegistry()
	return nil
}

func unregister(name string) {
	regMu.Lock()
	defer regMu.Unlock()
	loadRegistry()
	if _, ok := regEntries[name]; ok {
		delete(regEntries, name)
		saveRegistry()
	}
}

// Open returns an external controller for a terminal previously created with
// WithName + WithControlPipe by another process. It does not own the
// terminal; commands are forwarded over the control pipe.
func Open(name string) (*Remote, error) {
	regMu.Lock()
	loadRegistry()
	pipe, ok := regEntries[name]
	regMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("termctl: no terminal named %q", name)
	}
	return &Remote{name: name, pipe: pipe}, nil
}

// Remote is a handle to a TermHandle owned by another process, driven via its
// control pipe.
type Remote struct {
	name string
	pipe string
}

// Attach shows the remote terminal on display.
func (r *Remote) Attach(display string) error {
	return r.send("attach " + display)
}

// Detach hides the remote terminal's window.
func (r *Remote) Detach() error {
	return r.send("detach")
}

// Stop terminates the remote terminal's shell.
func (r *Remote) Stop() error {
	return r.send("stop")
}

// Status queries attached/detached.
func (r *Remote) Status() (string, error) {
	if err := r.send("status"); err != nil {
		return "", err
	}
	return "", nil
}

// Name returns the terminal name.
func (r *Remote) Name() string { return r.name }

func (r *Remote) send(cmd string) error {
	f, err := os.OpenFile(r.pipe, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, cmd)
	return err
}
