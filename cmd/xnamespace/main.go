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
// For programmatic use pass -s/--short for terse, tab-separated, header-less
// output — easy to parse from a shell or C, e.g.
// HANDLE=$(xnamespace -s addtoken ns proto hex). Errors still go to stderr;
// check the exit status.
//
// Capabilities: mouse shape transparency input keyboard admin all
//
// Auth options:
//
//	-a/--authority <file>    load entries from this Xauthority file
//	--no-xauth               skip loading any Xauthority file
//	-A/--auth-proto + --auth-data   inline auth entry (repeatable);
//	                               combined with file entries unless --no-xauth
//
// Examples:
//
//	# create a namespace with two capabilities
//	xnamespace create sandbox mouse keyboard
//
//	# add a MIT-MAGIC-COOKIE-1 token and capture its handle in a script
//	h=$(xnamespace -s addtoken sandbox MIT-MAGIC-COOKIE-1 00112233445566778899aabbccddeeff)
//
//	xnamespace -s whoami          # -> "sandbox\tclient"
//	xnamespace -s list            # one ns per line: name<TAB>caps<TAB>attrs<TAB>clients<TAB>tokens
//	xnamespace -s query sandbox   # one "key\tvalue" line per field
//	xnamespace delete sandbox kill
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
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/ext/namespace"
)

// short, set by the global -s/--short flag, switches command output to a
// terse, tab-separated, header-less form — trivially parsed from a shell
// ($(...), read, cut) or C (fscanf/strtok), for programmatic callers (e.g. a
// desktop launching a client into its own namespace). Default is human text.
var short bool

// auth authority file path (global -a/--authority flag)
var authorityFile string

// --no-xauth: skip loading any xauth file
var noXauth bool

// inline auth entries (from repeatable -A/--auth-proto + --auth-data pairs)
type authEntry struct {
	proto   string
	dataHex string
}

var inlineAuthEntries []authEntry

// pendingAuthProto tracks a -A/--auth-proto value waiting for its --auth-data
// during flag parsing.
var pendingAuthProto string

func main() {
	args := parseGlobalFlags(os.Args[1:])
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}

	conn, err := connectAuth()
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
		if short {
			fmt.Printf("%d.%d\n", maj, min)
		} else {
			fmt.Printf("X-NAMESPACE %d.%d\n", maj, min)
		}

	case "list":
		list, err := ns.ListNamespaces()
		check(err)
		if short {
			for _, it := range list {
				fmt.Printf("%s\t%s\t%s\t%d\t%d\n", it.Name,
					csv(capList(it.Capabilities)), csv(attrList(it.Attributes)),
					it.Refcnt, it.NumTokens)
			}
		} else {
			printList(list)
		}

	case "create":
		if len(args) < 1 {
			fatal("create needs a name")
		}
		caps, attrs := parseCapsAndAttrs(args[1:])
		check(ns.CreateNamespace(args[0], caps, attrs))
		if short {
			fmt.Println(args[0])
		} else {
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
		if short {
			fmt.Println(args[0])
		} else {
			fmt.Printf("deleted %q\n", args[0])
		}

	case "query":
		if len(args) < 1 {
			fatal("query needs a name")
		}
		info, err := ns.QueryNamespace(args[0])
		check(err)
		if short {
			fmt.Printf("name\t%s\n", info.Name)
			fmt.Printf("capabilities\t%s\n", csv(capList(info.Capabilities)))
			fmt.Printf("attributes\t%s\n", csv(attrList(info.Attributes)))
			fmt.Printf("clients\t%d\n", info.Refcnt)
			fmt.Printf("tokens\t%d\n", info.NumTokens)
		} else {
			printInfo(info)
		}

	case "setflags":
		if len(args) < 2 {
			fatal("setflags needs a name and at least one +cap/-cap")
		}
		mask, vals := parseFlagDeltas(args[1:])
		caps, err := ns.SetNamespaceFlags(args[0], mask, vals)
		check(err)
		if short {
			fmt.Println(csv(capList(caps)))
		} else {
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
		if short {
			fmt.Printf("%s\t%s\n", name, who)
		} else {
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
		if short {
			fmt.Printf("0x%x\n", handle)
		} else {
			fmt.Printf("token handle: 0x%x\n", handle)
		}

	case "rmtoken":
		if len(args) < 2 {
			fatal("rmtoken needs <name> <handle>")
		}
		check(ns.RemoveAuthToken(args[0], base.CARD32(parseUint(args[1]))))
		if short {
			fmt.Println(args[0])
		} else {
			fmt.Printf("removed token from %q\n", args[0])
		}

	case "tokens":
		if len(args) < 1 {
			fatal("tokens needs a name")
		}
		toks, err := ns.ListAuthTokens(args[0])
		check(err)
		if short {
			for _, t := range toks {
				fmt.Printf("0x%x\t%s\n", t.Handle, t.Proto)
			}
		} else {
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

// --- auth resolution ---

// connectAuth establishes an X11 connection using the combined auth
// configuration from flags: authority file (unless --no-xauth) plus any
// inline -A/--auth-proto + --auth-data entries.
func connectAuth() (*core.X11Conn, error) {
	// Fast path: no inline entries and no --no-xauth → standard resolution.
	if len(inlineAuthEntries) == 0 && !noXauth {
		return proto.DialAuth("", authorityFile, "", nil)
	}

	// Build combined entry list from file + inline sources.
	var entries []base.XauthEntry

	if !noXauth {
		fileEntries, ferr := base.XauthFileOrEntries(authorityFile)
		if ferr != nil {
			return nil, ferr
		}
		entries = append(entries, fileEntries...)
	}

	for _, ae := range inlineAuthEntries {
		data, derr := hex.DecodeString(ae.dataHex)
		if derr != nil {
			return nil, fmt.Errorf("bad --auth-data hex: %v", derr)
		}
		entries = append(entries, base.XauthEntry{
			Family: base.XauthFamilyWild,
			Proto:  ae.proto,
			Data:   data,
		})
	}

	if len(entries) == 0 {
		return proto.DialAuth("", "", "", nil)
	}

	// Single entry from file only (no inline) → let DialAuth handle it.
	if len(inlineAuthEntries) == 0 && len(entries) > 0 {
		return proto.DialAuth("", authorityFile, "", nil)
	}

	// Multiple entries or mixed sources → resolve via display lookup.
	displayStr := os.Getenv("DISPLAY")
	if displayStr == "" {
		// No DISPLAY: use first inline entry as fallback.
		ae := inlineAuthEntries[0]
		data, _ := hex.DecodeString(ae.dataHex)
		return proto.DialAuth("", "", ae.proto, data)
	}

	display, derr := base.ParseDisplay(displayStr)
	if derr != nil {
		return nil, fmt.Errorf("bad DISPLAY: %v", derr)
	}

	entry := base.LookupXauth(display, entries)
	if entry != nil {
		return proto.DialAuth("", "", entry.Proto, entry.Data)
	}
	return proto.DialAuth("", "", "", nil)
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

// parseGlobalFlags strips global flags from the argument list, leaving the
// command and its positional arguments.  Supported global flags:
//
//	-s, --short              terse output for scripts
//	-a, --authority <file>   explicit Xauthority file path
//	    --no-xauth           skip loading any Xauthority file
//	-A, --auth-proto <name>  auth protocol name (repeatable, pairs with --auth-data)
//	    --auth-data <hex>    auth token as hex string (requires preceding --auth-proto)
func parseGlobalFlags(in []string) []string {
	out := make([]string, 0, len(in))
	for i := 0; i < len(in); i++ {
		switch in[i] {
		case "-s", "--short":
			short = true
		case "-a", "--authority":
			if i+1 < len(in) {
				i++
				authorityFile = in[i]
			}
		case "--no-xauth":
			noXauth = true
		case "-A", "--auth-proto":
			if i+1 < len(in) {
				i++
				pendingAuthProto = in[i]
			}
		case "--auth-data":
			if i+1 < len(in) {
				i++
				if pendingAuthProto == "" {
					fatal("--auth-data requires a preceding --auth-proto")
				}
				inlineAuthEntries = append(inlineAuthEntries, authEntry{pendingAuthProto, in[i]})
				pendingAuthProto = ""
			}
		default:
			out = append(out, in[i])
		}
	}
	if pendingAuthProto != "" {
		fatal("--auth-proto %q requires a following --auth-data", pendingAuthProto)
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

// fatal prints an error message to stderr and exits.  Variable so that tests
// can override it to observe the message instead of calling os.Exit.
var fatal = func(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "xnamespace: "+format+"\n", a...)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: xnamespace [global-flags] <command> [args]

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

global options:
  -s, --short                      terse, tab-separated, header-less output for
                                   scripts / C (values only; empty on no result)
  -a, --authority <file>           path to Xauthority file (default: $XAUTHORITY
                                   or ~/.Xauthority)
      --no-xauth                   skip loading any Xauthority file
  -A, --auth-proto <name>          auth protocol name (repeatable, e.g.
                                   MIT-MAGIC-COOKIE-1); each must be followed by
                                   --auth-data
      --auth-data <hex>            auth token data as hex (requires preceding
                                   --auth-proto; may be repeated)

By default (no -a/-A/--auth-data), xauthority is read from $XAUTHORITY or
~/.Xauthority and the matching entry for the current $DISPLAY is used.

Multiple -A/--auth-proto + --auth-data pairs may be specified; entries are
combined with any file-based entries and the best match for $DISPLAY is used.
With --no-xauth, only inline entries are considered (file is not loaded).

Examples:
  xnamespace -s list                                          # auto auth
  xnamespace -a /tmp/server.xauth -s list                     # explicit file
  xnamespace --no-xauth -s list                               # no auth file
  xnamespace -A MIT-MAGIC-COOKIE-1 --auth-data DEADBEEF... -s list  # inline
  xnamespace -a /tmp/server.xauth \
             -A MIT-MAGIC-COOKIE-1 --auth-data AABB... -s list      # combined

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
