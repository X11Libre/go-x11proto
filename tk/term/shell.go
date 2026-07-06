package term

import "os"

// shellFromEnv picks the shell to spawn when Term.Shell is left empty:
// $SHELL if set, else the universally-present /bin/sh.
func shellFromEnv() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}
