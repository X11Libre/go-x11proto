package errorcode

const (
	BadRequest        = 1
	BadValue          = 2
	BadWindow         = 3
	BadPixmap         = 4
	BadAtom           = 5
	BadCursor         = 6
	BadFont           = 7
	BadMatch          = 8
	BadDrawable       = 9
	BadAccess         = 10
	BadAlloc          = 11
	BadColor          = 12
	BadGC             = 13
	BadIDChoice       = 14
	BadName           = 15
	BadLength         = 16
	BadImplementation = 17
)

var names = map[byte]string{
	BadRequest:        "BadRequest",
	BadValue:          "BadValue",
	BadWindow:         "BadWindow",
	BadPixmap:         "BadPixmap",
	BadAtom:           "BadAtom",
	BadCursor:         "BadCursor",
	BadFont:           "BadFont",
	BadMatch:          "BadMatch",
	BadDrawable:       "BadDrawable",
	BadAccess:         "BadAccess",
	BadAlloc:          "BadAlloc",
	BadColor:          "BadColor",
	BadGC:             "BadGC",
	BadIDChoice:       "BadIDChoice",
	BadName:           "BadName",
	BadLength:         "BadLength",
	BadImplementation: "BadImplementation",
}

func Name(code byte) string {
	n, ok := names[code]
	if !ok {
		return "unknown"
	}
	return n
}
