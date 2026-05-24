package base

import (
	"fmt"
	"regexp"
	"strconv"
)

type DisplaySpec struct {
	Host    string
	Display int
	Screen  int
}

func (d DisplaySpec) DialInfo() (string, string) {
	if d.Host == "" || d.Host == "unix" {
		return "unix", "/tmp/.X11-unix/X" + strconv.Itoa(d.Display)
	}
	return "tcp", d.Host + ":" + strconv.Itoa(6000+d.Screen)
}

var x11DisplayRegex = regexp.MustCompile(`^([^:]*):(\d+)(?:\.(\d+))?$`)

func ParseDisplay(displayStr string) (DisplaySpec, error) {
	spec := DisplaySpec{}

	matches := x11DisplayRegex.FindStringSubmatch(displayStr)
	if matches == nil {
		return spec, fmt.Errorf("illegal format: %q", displayStr)
	}

	spec.Host = matches[1]

	var err error
	spec.Display, err = strconv.Atoi(matches[2])
	if err != nil {
		return spec, fmt.Errorf("illegal format in display number: %w", err)
	}

	if matches[3] != "" {
		spec.Screen, err = strconv.Atoi(matches[3])
		if err != nil {
			return spec, fmt.Errorf("illegal format in screen number: %w", err)
		}
	}

	return spec, nil
}
