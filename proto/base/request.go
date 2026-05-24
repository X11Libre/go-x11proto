package base

// Request Interface
type Request interface {
	WriteInto(*RequestWriter) error
}
