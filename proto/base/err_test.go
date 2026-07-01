package base

import (
	"errors"
	"testing"
)

// MakeX11ErrorF must format like fmt.Errorf: a %w verb renders the wrapped
// error's message (not "%!w(...)") and stays reachable via errors.Is/As.
func TestMakeX11ErrorFWraps(t *testing.T) {
	sentinel := errors.New("connect: no such file or directory")
	err := MakeX11ErrorF("failed to connect: %w", sentinel)

	if got, want := err.Error(), "failed to connect: connect: no such file or directory"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("errors.Is could not reach the wrapped error through %T", err)
	}
}

func TestMakeX11ErrorFNoWrap(t *testing.T) {
	err := MakeX11ErrorF("plain %d/%s", 42, "x")
	if got, want := err.Error(), "plain 42/x"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if errors.Unwrap(err) != nil {
		t.Fatalf("Unwrap() = %v, want nil (no %%w)", errors.Unwrap(err))
	}
}
