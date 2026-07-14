// Command xnamespace-test is a conformance test for the X-NAMESPACE extension,
// in the spirit of Xorg's XTS or rendercheck: it connects to an X server that
// offers the extension and exercises every request plus the documented error
// cases, emitting TAP (Test Anything Protocol) so it drops straight into a
// meson test() or a CI step.
//
// It must run as a privileged (superPower) client, since the whole extension is
// reachable only by such clients. The companion runner
// (contrib/xnamespace/run-xnamespace-test.sh) launches a server with a config
// that grants the connecting client superpower and points this tool at it.
//
//	xnamespace-test [-be] [-v] [-display :N]
//
//	-be       use a big-endian client connection (default little-endian)
//	-v        verbose: log each request/result as a TAP diagnostic
//	-display  X display to connect to (default $DISPLAY)
//
// Exit status is 0 when every test passes, 1 otherwise (or on a bail-out, e.g.
// the extension not being present).
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/errorcode"
	"github.com/X11Libre/go-x11proto/proto/ext/namespace"
)

// test namespaces this tool creates; cleaned up before and after the run.
const (
	nsA = "xnstest"
	nsB = "xnstest-transient"
)

func main() {
	be := flag.Bool("be", false, "use a big-endian client connection")
	verbose := flag.Bool("v", false, "verbose TAP diagnostics")
	display := flag.String("display", "", "X display (default $DISPLAY)")
	skipIfUnavailable := flag.Bool("skip-if-unavailable", false,
		"skip (instead of failing) when the X-NAMESPACE extension is not present on the server")
	flag.Parse()

	dial := proto.Dial
	if *be {
		dial = proto.DialBE
	}
	conn, err := dial(*display)
	if err != nil {
		bail("connect: %v", err)
	}
	defer conn.Close()

	// Probe availability early so we can skip cleanly (instead of failing) when
	// the server was built without the X-NAMESPACE extension (CONFIG_NAMESPACE).
	ext, err := conn.QueryExtension(namespace.ExtName)
	if err != nil {
		bail("query extension: %v", err)
	}
	if !ext.Present {
		if *skipIfUnavailable {
			fmt.Printf("1..0 # SKIP X-NAMESPACE extension not available on this server\n")
			os.Exit(0)
		}
		bail("%s extension not available", namespace.ExtName)
	}

	ns, err := namespace.Query(conn)
	if err != nil {
		bail("%v", err)
	}

	t := &tap{verbose: *verbose}
	cleanup(ns) // remove leftovers from a previous run
	runChecks(t, ns)
	cleanup(ns)
	t.done()
	if t.failed > 0 {
		os.Exit(1)
	}
}

func runChecks(t *tap, ns *namespace.Namespace) {
	t.check("QueryVersion reports >= 1.0", func() error {
		maj, min, err := ns.QueryVersion()
		if err != nil {
			return err
		}
		if maj < 1 {
			return fmt.Errorf("version %d.%d < 1.0", maj, min)
		}
		t.diag("version %d.%d", maj, min)
		return nil
	})

	t.check("GetClientNamespace returns this client's namespace", func() error {
		name, isServer, err := ns.GetClientNamespace(0)
		if err != nil {
			return err
		}
		if name == "" {
			return errors.New("empty namespace name")
		}
		if isServer {
			return errors.New("a client connection reported isServer=true")
		}
		t.diag("client namespace = %q", name)
		return nil
	})

	t.check("ListNamespaces includes the builtin root and anon", func() error {
		list, err := ns.ListNamespaces()
		if err != nil {
			return err
		}
		if !hasNS(list, "root") || !hasNS(list, "anon") {
			return fmt.Errorf("missing builtin namespace(s): %v", names(list))
		}
		return nil
	})

	t.check("root namespace is immutable", func() error {
		info, err := ns.QueryNamespace("root")
		if err != nil {
			return err
		}
		if info.Attributes&namespace.AttrImmutable == 0 {
			return fmt.Errorf("root attributes = %#x, want IMMUTABLE set", info.Attributes)
		}
		return nil
	})

	t.check("CreateNamespace creates a namespace", func() error {
		return ns.CreateNamespace(nsA, namespace.CapInput|namespace.CapKeyboard, 0)
	})

	t.check("QueryNamespace returns the created capabilities", func() error {
		info, err := ns.QueryNamespace(nsA)
		if err != nil {
			return err
		}
		if want := namespace.CapInput | namespace.CapKeyboard; info.Capabilities != want {
			return fmt.Errorf("capabilities = %#x, want %#x", info.Capabilities, want)
		}
		return nil
	})

	t.check("duplicate CreateNamespace is rejected with BadName", func() error {
		return wantErr(errorcode.BadName, ns.CreateNamespace(nsA, 0, 0))
	})

	t.check("CreateNamespace with an empty name is rejected with BadName", func() error {
		return wantErr(errorcode.BadName, ns.CreateNamespace("", 0, 0))
	})

	t.check("CreateNamespace with an illegal name is rejected with BadName", func() error {
		return wantErr(errorcode.BadName, ns.CreateNamespace("bad/name", 0, 0))
	})

	t.check("CreateNamespace with a reserved capability bit is rejected with BadValue", func() error {
		return wantErr(errorcode.BadValue, ns.CreateNamespace("xnstest-badcap", base.CARD32(1<<20), 0))
	})

	t.check("CreateNamespace with a reserved attribute bit is rejected with BadValue", func() error {
		return wantErr(errorcode.BadValue, ns.CreateNamespace("xnstest-badattr", 0, base.CARD32(1<<5)))
	})

	t.check("SetNamespaceFlags changes the capability bits", func() error {
		// drop KEYBOARD, add ADMIN; INPUT stays.
		mask := namespace.CapKeyboard | namespace.CapAdmin
		caps, err := ns.SetNamespaceFlags(nsA, mask, namespace.CapAdmin)
		if err != nil {
			return err
		}
		if want := namespace.CapInput | namespace.CapAdmin; caps != want {
			return fmt.Errorf("resulting capabilities = %#x, want %#x", caps, want)
		}
		info, err := ns.QueryNamespace(nsA)
		if err != nil {
			return err
		}
		if info.Capabilities != caps {
			return fmt.Errorf("query capabilities = %#x, want %#x", info.Capabilities, caps)
		}
		return nil
	})

	t.check("SetNamespaceFlags on a builtin namespace is rejected with BadAccess", func() error {
		return wantErr(errorcode.BadAccess, errOf2(ns.SetNamespaceFlags("root", namespace.CapAdmin, 0)))
	})

	t.check("DeleteNamespace on a builtin namespace is rejected with BadAccess", func() error {
		return wantErr(errorcode.BadAccess, ns.DeleteNamespace("root", namespace.DeleteFailIfBusy))
	})

	var token base.CARD32
	t.check("AddAuthToken returns a token handle", func() error {
		h, err := ns.AddAuthToken(nsA, "MIT-MAGIC-COOKIE-1", []byte{0xde, 0xad, 0xbe, 0xef})
		if err != nil {
			return err
		}
		token = h
		t.diag("token handle = %#x", h)
		return nil
	})

	t.check("ListAuthTokens lists the added token (no key material)", func() error {
		toks, err := ns.ListAuthTokens(nsA)
		if err != nil {
			return err
		}
		for _, tk := range toks {
			if tk.Handle == token && tk.Proto == "MIT-MAGIC-COOKIE-1" {
				return nil
			}
		}
		return fmt.Errorf("token %#x not found in %+v", token, toks)
	})

	t.check("RemoveAuthToken removes the token", func() error {
		if err := ns.RemoveAuthToken(nsA, token); err != nil {
			return err
		}
		toks, err := ns.ListAuthTokens(nsA)
		if err != nil {
			return err
		}
		if len(toks) != 0 {
			return fmt.Errorf("expected no tokens, got %d", len(toks))
		}
		return nil
	})

	t.check("transient attribute is honored", func() error {
		if err := ns.CreateNamespace(nsB, 0, namespace.AttrTransient); err != nil {
			return err
		}
		info, err := ns.QueryNamespace(nsB)
		if err != nil {
			return err
		}
		if info.Attributes&namespace.AttrTransient == 0 {
			return fmt.Errorf("attributes = %#x, want TRANSIENT set", info.Attributes)
		}
		return nil
	})

	t.check("DeleteNamespace removes a namespace", func() error {
		if err := ns.DeleteNamespace(nsA, namespace.DeleteKillClients); err != nil {
			return err
		}
		list, err := ns.ListNamespaces()
		if err != nil {
			return err
		}
		if hasNS(list, nsA) {
			return fmt.Errorf("%q still present after delete", nsA)
		}
		return nil
	})

	t.check("DeleteNamespace of a missing namespace is rejected with BadName", func() error {
		return wantErr(errorcode.BadName, ns.DeleteNamespace("no-such-namespace", namespace.DeleteFailIfBusy))
	})
}

// cleanup removes the namespaces this tool may have created, ignoring errors.
func cleanup(ns *namespace.Namespace) {
	for _, name := range []string{nsA, nsB} {
		_ = ns.DeleteNamespace(name, namespace.DeleteKillClients)
	}
}

// --- helpers ---

func hasNS(list []namespace.Info, name string) bool {
	for _, it := range list {
		if it.Name == name {
			return true
		}
	}
	return false
}

func names(list []namespace.Info) []string {
	out := make([]string, len(list))
	for i, it := range list {
		out[i] = it.Name
	}
	return out
}

// wantErr returns nil iff err is a protocol error with the given code.
func wantErr(code byte, err error) error {
	if err == nil {
		return fmt.Errorf("expected %s, got success", errorcode.Name(code))
	}
	var re *core.RequestError
	if !errors.As(err, &re) {
		return fmt.Errorf("expected %s, got non-protocol error: %v", errorcode.Name(code), err)
	}
	if byte(re.Code) != code {
		return fmt.Errorf("expected %s, got %s", errorcode.Name(code), errorcode.Name(byte(re.Code)))
	}
	return nil
}

// errOf2 discards the first result of a (T, error) call so its error can be fed
// to wantErr.
func errOf2[T any](_ T, err error) error { return err }

func bail(format string, a ...any) {
	fmt.Printf("1..0 # SKIP "+format+"\n", a...)
	fmt.Printf("Bail out! "+format+"\n", a...)
	os.Exit(1)
}

// --- minimal TAP emitter ---

type tap struct {
	n       int
	failed  int
	verbose bool
}

func (t *tap) check(name string, fn func() error) {
	t.n++
	if err := fn(); err != nil {
		t.failed++
		fmt.Printf("not ok %d - %s\n", t.n, name)
		fmt.Printf("# %v\n", err)
		return
	}
	fmt.Printf("ok %d - %s\n", t.n, name)
}

func (t *tap) diag(format string, a ...any) {
	if t.verbose {
		fmt.Printf("# "+format+"\n", a...)
	}
}

func (t *tap) done() {
	fmt.Printf("1..%d\n", t.n)
	if t.failed > 0 {
		fmt.Printf("# %d of %d tests failed\n", t.failed, t.n)
	} else {
		fmt.Printf("# all %d tests passed\n", t.n)
	}
}
