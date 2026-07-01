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
// For programmatic use pass a global output flag (errors still go to stderr;
// check the exit code):
//
//	-s, --short   terse, tab-separated, header-less lines — easy to parse from a
//	              shell or C (e.g. HANDLE=$(xnamespace -s addtoken ns proto hex))
//	-json         structured JSON, for richer consumers
//
// Capabilities: mouse shape transparency input keyboard admin all
//
// The extension is privileged; against a server that does not offer it (or to
// an unprivileged client) Query fails with "extension not available".
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/ext/namespace"
)

// Output mode, set by global flags:
//   -json    machine-readable JSON (structured, for richer consumers)
//   -s       terse, tab-separated, header-less lines — trivially parsed from a
//            shell ($(...), read, cut) or C (fscanf/strtok), for programmatic
//            callers (e.g. a desktop launching a client into its own namespace)
// short takes precedence over jsonOut if both are given; default is human text.
var (
	jsonOut bool
	short   bool
)

func main() {
	args := parseGlobalFlags(os.Args[1:])
	if len(args) < 1 {
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

	cmd, args := args[0], args[1:]
	switch cmd {
	case "version":
		maj, min, err := ns.QueryVersion()
		check(err)
		switch {
		case short:
			fmt.Printf("%d.%d\n", maj, min)
		case jsonOut:
			emit(map[string]any{"major": maj, "minor": min})
		default:
			fmt.Printf("X-NAMESPACE %d.%d\n", maj, min)
		}

	case "list":
		list, err := ns.ListNamespaces()
		check(err)
		switch {
		case short:
			for _, it := range list {
				fmt.Printf("%s\t%s\t%s\t%d\t%d\n", it.Name,
					csv(capList(it.Capabilities)), csv(attrList(it.Attributes)),
					it.Refcnt, it.NumTokens)
			}
		case jsonOut:
			out := []jsonInfo{}
			for _, it := range list {
				out = append(out, infoJSON(it))
			}
			emit(out)
		default:
			printList(list)
		}

	case "create":
		if len(args) < 1 {
			fatal("create needs a name")
		}
		caps, attrs := parseCapsAndAttrs(args[1:])
		check(ns.CreateNamespace(args[0], caps, attrs))
		switch {
		case short:
			fmt.Println(args[0])
		case jsonOut:
			emit(map[string]any{"created": args[0]})
		default:
			fmt.Printf("created %q\n", args[0])
		}

	case "delete":
		if len(args) < 1 {
			fatal("delete needs a name")
		}
		onClients := namespace.DeleteFailIfBusy
		if contains(args[1:], "kill") {
			onClients = namespace.DeleteKillClients
		}
		check(ns.DeleteNamespace(args[0], onClients))
		switch {
		case short:
			fmt.Println(args[0])
		case jsonOut:
			emit(map[string]any{"deleted": args[0]})
		default:
			fmt.Printf("deleted %q\n", args[0])
		}

	case "query":
		if len(args) < 1 {
			fatal("query needs a name")
		}
		info, err := ns.QueryNamespace(args[0])
		check(err)
		switch {
		case short:
			fmt.Printf("name\t%s\n", info.Name)
			fmt.Printf("capabilities\t%s\n", csv(capList(info.Capabilities)))
			fmt.Printf("attributes\t%s\n", csv(attrList(info.Attributes)))
			fmt.Printf("clients\t%d\n", info.Refcnt)
			fmt.Printf("tokens\t%d\n", info.NumTokens)
		case jsonOut:
			emit(infoJSON(info))
		default:
			printInfo(info)
		}

	case "setflags":
		if len(args) < 2 {
			fatal("setflags needs a name and at least one +cap/-cap")
		}
		mask, vals := parseFlagDeltas(args[1:])
		caps, err := ns.SetNamespaceFlags(args[0], mask, vals)
		check(err)
		switch {
		case short:
			fmt.Println(csv(capList(caps)))
		case jsonOut:
			emit(map[string]any{"name": args[0], "capabilities": capList(caps)})
		default:
			fmt.Printf("capabilities now: %s\n", capString(caps))
		}

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
		switch {
		case short:
			fmt.Printf("%s\t%s\n", name, who)
		case jsonOut:
			emit(map[string]any{"namespace": name, "server": isServer})
		default:
			fmt.Printf("namespace=%q (%s)\n", name, who)
		}

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
		switch {
		case short:
			fmt.Printf("0x%x\n", handle)
		case jsonOut:
			emit(map[string]any{"handle": uint32(handle)})
		default:
			fmt.Printf("token handle: 0x%x\n", handle)
		}

	case "rmtoken":
		if len(args) < 2 {
			fatal("rmtoken needs <name> <handle>")
		}
		check(ns.RemoveAuthToken(args[0], base.CARD32(parseUint(args[1]))))
		switch {
		case short:
			fmt.Println(args[0])
		case jsonOut:
			emit(map[string]any{"removed": args[0]})
		default:
			fmt.Printf("removed token from %q\n", args[0])
		}

	case "tokens":
		if len(args) < 1 {
			fatal("tokens needs a name")
		}
		toks, err := ns.ListAuthTokens(args[0])
		check(err)
		switch {
		case short:
			for _, t := range toks {
				fmt.Printf("0x%x\t%s\n", t.Handle, t.Proto)
			}
		case jsonOut:
			out := []jsonToken{}
			for _, t := range toks {
				out = append(out, jsonToken{Handle: uint32(t.Handle), Proto: t.Proto})
			}
			emit(out)
		default:
			for _, t := range toks {
				fmt.Printf("  0x%-8x %s\n", t.Handle, t.Proto)
			}
			if len(toks) == 0 {
				fmt.Println("  (no tokens)")
			}
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

// capList returns the set capability bits as stable name strings (never nil, so
// it marshals to a JSON [] rather than null).
func capList(c base.CARD32) []string {
	out := []string{}
	for _, name := range []string{"mouse", "shape", "transparency", "input", "keyboard", "admin"} {
		if c&capByName[name] != 0 {
			out = append(out, name)
		}
	}
	return out
}

// attrList returns the set attribute bits as stable name strings.
func attrList(a base.CARD32) []string {
	out := []string{}
	if a&namespace.AttrImmutable != 0 {
		out = append(out, "immutable")
	}
	if a&namespace.AttrTransient != 0 {
		out = append(out, "transient")
	}
	return out
}

func capString(c base.CARD32) string {
	if l := capList(c); len(l) > 0 {
		return strings.Join(l, ",")
	}
	return "(none)"
}

func attrString(a base.CARD32) string {
	if l := attrList(a); len(l) > 0 {
		return strings.Join(l, ",")
	}
	return "(none)"
}

// csv joins names for short (-s) output: a comma-separated list, or "" when
// empty (no "(none)" decoration, so scripts get a real empty field).
func csv(l []string) string { return strings.Join(l, ",") }

// --- machine-readable (JSON) output ---

type jsonInfo struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Attributes   []string `json:"attributes"`
	Clients      uint32   `json:"clients"`
	Tokens       uint32   `json:"tokens"`
}

type jsonToken struct {
	Handle uint32 `json:"handle"`
	Proto  string `json:"proto"`
}

func infoJSON(it namespace.Info) jsonInfo {
	return jsonInfo{
		Name:         it.Name,
		Capabilities: capList(it.Capabilities),
		Attributes:   attrList(it.Attributes),
		Clients:      uint32(it.Refcnt),
		Tokens:       uint32(it.NumTokens),
	}
}

// emit writes v as indented JSON to stdout.
func emit(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal("json: %v", err)
	}
	fmt.Println(string(b))
}

// parseGlobalFlags strips the global -json/--json flag from the argument list,
// leaving the command and its positional arguments.
func parseGlobalFlags(in []string) []string {
	out := make([]string, 0, len(in))
	for _, a := range in {
		switch a {
		case "-json", "--json":
			jsonOut = true
		case "-s", "--short":
			short = true
		default:
			out = append(out, a)
		}
	}
	return out
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

options:
  -s, --short                      terse, tab-separated, header-less output for
                                   scripts / C (values only; empty on no result)
  -json                            emit machine-readable JSON on stdout

capabilities: mouse shape transparency input keyboard admin all

Short (-s) output per command (TAB-separated, no header, newline per record):
  version    MAJOR.MINOR
  list       name  caps(csv)  attrs(csv)  clients  tokens   (one line per ns)
  create     name
  delete     name
  query      one "key<TAB>value" line per field (name/capabilities/attributes/clients/tokens)
  setflags   caps(csv)
  whoami     name  client|server
  addtoken   0xHANDLE
  rmtoken    name
  tokens     0xHANDLE  proto                              (one line per token)
`)
}
