// Command xnamespace is a small client for the X-NAMESPACE extension: it lists,
// creates, queries and deletes server-side namespaces and manages their auth
// tokens. It is the reference user of proto/ext/namespace.
//
// Usage:
//
//	xnamespace version
//	xnamespace list
//	xnamespace create <name> [cap...] [transient]
//	xnamespace delete <name> [kill]
//	xnamespace query  <name>
//	xnamespace setflags <name> <+cap|-cap>...
//	xnamespace whoami [resource-id]
//	xnamespace addtoken <name> <proto> <hexdata>
//	xnamespace rmtoken  <name> <handle>
//	xnamespace tokens   <name>
//
// Capabilities: mouse shape transparency input keyboard admin all
//
// The extension is privileged; against a server that does not offer it (or to
// an unprivileged client) Query fails with "extension not available".
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/ext/namespace"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	conn, err := proto.Dial("")
	if err != nil {
		fatal("connect: %v", err)
	}
	defer conn.Close()

	ns, err := namespace.Query(conn)
	if err != nil {
		fatal("%v", err)
	}

	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "version":
		maj, min, err := ns.QueryVersion()
		check(err)
		fmt.Printf("X-NAMESPACE %d.%d\n", maj, min)

	case "list":
		list, err := ns.ListNamespaces()
		check(err)
		printList(list)

	case "create":
		if len(args) < 1 {
			fatal("create needs a name")
		}
		caps, attrs := parseCapsAndAttrs(args[1:])
		check(ns.CreateNamespace(args[0], caps, attrs))
		fmt.Printf("created %q\n", args[0])

	case "delete":
		if len(args) < 1 {
			fatal("delete needs a name")
		}
		onClients := namespace.DeleteFailIfBusy
		if contains(args[1:], "kill") {
			onClients = namespace.DeleteKillClients
		}
		check(ns.DeleteNamespace(args[0], onClients))
		fmt.Printf("deleted %q\n", args[0])

	case "query":
		if len(args) < 1 {
			fatal("query needs a name")
		}
		info, err := ns.QueryNamespace(args[0])
		check(err)
		printInfo(info)

	case "setflags":
		if len(args) < 2 {
			fatal("setflags needs a name and at least one +cap/-cap")
		}
		mask, vals := parseFlagDeltas(args[1:])
		caps, err := ns.SetNamespaceFlags(args[0], mask, vals)
		check(err)
		fmt.Printf("capabilities now: %s\n", capString(caps))

	case "whoami":
		var resource base.CARD32
		if len(args) >= 1 {
			resource = base.CARD32(parseUint(args[0]))
		}
		name, isServer, err := ns.GetClientNamespace(resource)
		check(err)
		who := "client"
		if isServer {
			who = "server"
		}
		fmt.Printf("namespace=%q (%s)\n", name, who)

	case "addtoken":
		if len(args) < 3 {
			fatal("addtoken needs <name> <proto> <hexdata>")
		}
		data, err := hex.DecodeString(args[2])
		if err != nil {
			fatal("bad hex data: %v", err)
		}
		handle, err := ns.AddAuthToken(args[0], args[1], data)
		check(err)
		fmt.Printf("token handle: 0x%x\n", handle)

	case "rmtoken":
		if len(args) < 2 {
			fatal("rmtoken needs <name> <handle>")
		}
		check(ns.RemoveAuthToken(args[0], base.CARD32(parseUint(args[1]))))
		fmt.Printf("removed token from %q\n", args[0])

	case "tokens":
		if len(args) < 1 {
			fatal("tokens needs a name")
		}
		toks, err := ns.ListAuthTokens(args[0])
		check(err)
		for _, t := range toks {
			fmt.Printf("  0x%-8x %s\n", t.Handle, t.Proto)
		}
		if len(toks) == 0 {
			fmt.Println("  (no tokens)")
		}

	default:
		usage()
		os.Exit(2)
	}
}

// --- capability / attribute parsing ---

var capByName = map[string]base.CARD32{
	"mouse":        namespace.CapMouseMotion,
	"shape":        namespace.CapShape,
	"transparency": namespace.CapTransparency,
	"input":        namespace.CapInput,
	"keyboard":     namespace.CapKeyboard,
	"admin":        namespace.CapAdmin,
	"all":          namespace.CapAll,
}

// parseCapsAndAttrs reads capability names and the "transient" attribute from a
// create command's trailing words.
func parseCapsAndAttrs(words []string) (caps, attrs base.CARD32) {
	for _, w := range words {
		switch w {
		case "transient":
			attrs |= namespace.AttrTransient
		default:
			c, ok := capByName[w]
			if !ok {
				fatal("unknown capability %q (have: %s)", w, capNames())
			}
			caps |= c
		}
	}
	return caps, attrs
}

// parseFlagDeltas reads +cap / -cap words into a (mask, values) pair.
func parseFlagDeltas(words []string) (mask, values base.CARD32) {
	for _, w := range words {
		if len(w) < 2 || (w[0] != '+' && w[0] != '-') {
			fatal("setflags expects +cap or -cap, got %q", w)
		}
		c, ok := capByName[w[1:]]
		if !ok {
			fatal("unknown capability %q (have: %s)", w[1:], capNames())
		}
		mask |= c
		if w[0] == '+' {
			values |= c
		}
	}
	return mask, values
}

func capString(c base.CARD32) string {
	if c == 0 {
		return "(none)"
	}
	var parts []string
	for _, name := range []string{"mouse", "shape", "transparency", "input", "keyboard", "admin"} {
		if c&capByName[name] != 0 {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ",")
}

func attrString(a base.CARD32) string {
	var parts []string
	if a&namespace.AttrImmutable != 0 {
		parts = append(parts, "immutable")
	}
	if a&namespace.AttrTransient != 0 {
		parts = append(parts, "transient")
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, ",")
}

func printList(list []namespace.Info) {
	if len(list) == 0 {
		fmt.Println("(no namespaces)")
		return
	}
	fmt.Printf("%-20s %-24s %-20s %6s %6s\n", "NAME", "CAPABILITIES", "ATTRIBUTES", "CLIENTS", "TOKENS")
	for _, it := range list {
		fmt.Printf("%-20s %-24s %-20s %6d %6d\n",
			it.Name, capString(it.Capabilities), attrString(it.Attributes), it.Refcnt, it.NumTokens)
	}
}

func printInfo(it namespace.Info) {
	fmt.Printf("name:         %s\n", it.Name)
	fmt.Printf("capabilities: %s\n", capString(it.Capabilities))
	fmt.Printf("attributes:   %s\n", attrString(it.Attributes))
	fmt.Printf("clients:      %d\n", it.Refcnt)
	fmt.Printf("tokens:       %d\n", it.NumTokens)
}

// --- small helpers ---

func capNames() string {
	return "mouse shape transparency input keyboard admin all"
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func parseUint(s string) uint64 {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), hexOrDec(s), 32)
	if err != nil {
		fatal("bad number %q: %v", s, err)
	}
	return v
}

func hexOrDec(s string) int {
	if strings.HasPrefix(s, "0x") {
		return 16
	}
	return 10
}

func check(err error) {
	if err != nil {
		fatal("%v", err)
	}
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "xnamespace: "+format+"\n", a...)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: xnamespace <command> [args]

  version                          show extension version
  list                             list all namespaces
  create <name> [cap...] [transient]   create a namespace
  delete <name> [kill]             delete a namespace (kill = terminate clients)
  query  <name>                    show one namespace
  setflags <name> <+cap|-cap>...   change capability bits
  whoami [resource-id]             show a client's namespace
  addtoken <name> <proto> <hex>    add an auth token
  rmtoken  <name> <handle>         remove an auth token
  tokens   <name>                  list a namespace's auth tokens

capabilities: mouse shape transparency input keyboard admin all
`)
}
