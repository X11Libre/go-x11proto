package xpm

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Image struct {
	Width  int
	Height int
	Data   []byte // RGBA pixel data, 4 bytes per pixel
}

func Decode(r io.Reader) (*Image, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("xpm: read: %w", err)
	}

	vals := extractStrings(string(buf))
	if len(vals) < 2 {
		return nil, fmt.Errorf("xpm: no data found")
	}

	var w, h, nc, cpp int
	_, err = fmt.Sscanf(vals[0], "%d %d %d %d", &w, &h, &nc, &cpp)
	if err != nil {
		return nil, fmt.Errorf("xpm: bad header: %w", err)
	}

	if len(vals) < 1+nc+h {
		return nil, fmt.Errorf("xpm: truncated: need %d strings, got %d", 1+nc+h, len(vals))
	}

	cmap := make(map[string][4]byte)
	for i := 0; i < nc; i++ {
		key, color, err := parseColor(vals[1+i], cpp)
		if err != nil {
			return nil, fmt.Errorf("xpm: color %d: %w", i, err)
		}
		cmap[key] = color
	}

	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		row := vals[1+nc+y]
		for x := 0; x < w; x++ {
			idx := x * cpp
			if idx+cpp > len(row) {
				return nil, fmt.Errorf("xpm: row %d too short", y)
			}
			key := row[idx : idx+cpp]
			c, ok := cmap[key]
			if !ok {
				return nil, fmt.Errorf("xpm: undefined color key %q at %d,%d", key, x, y)
			}
			pix[(y*w+x)*4+0] = c[0]
			pix[(y*w+x)*4+1] = c[1]
			pix[(y*w+x)*4+2] = c[2]
			pix[(y*w+x)*4+3] = c[3]
		}
	}

	return &Image{Width: w, Height: h, Data: pix}, nil
}

func extractStrings(src string) []string {
	start := strings.IndexByte(src, '{')
	if start < 0 {
		return nil
	}
	src = src[start:]

	end := strings.IndexByte(src, '}')
	if end < 0 {
		return nil
	}
	src = src[:end+1]

	var vals []string
	for {
		q := strings.IndexByte(src, '"')
		if q < 0 {
			break
		}
		src = src[q+1:]
		q = strings.IndexByte(src, '"')
		if q < 0 {
			break
		}
		vals = append(vals, src[:q])
		src = src[q+1:]
	}
	return vals
}

func parseColor(s string, cpp int) (string, [4]byte, error) {
	key := s
	if len(s) >= cpp {
		key = s[:cpp]
		s = s[cpp:]
	}

	s = strings.TrimSpace(s)

	var color [4]byte
	color[3] = 0xFF

	if strings.Contains(s, "None") {
		color[3] = 0
		return key, color, nil
	}

	for {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			break
		}
		tag := s[0]
		s = s[1:]
		s = strings.TrimSpace(s)

		sp := strings.IndexByte(s, ' ')
		var val string
		if sp < 0 {
			val = s
			s = ""
		} else {
			val = s[:sp]
			s = s[sp+1:]
		}

		if tag == 'c' {
			if len(val) == 7 && val[0] == '#' {
				r, _ := strconv.ParseUint(val[1:3], 16, 8)
				g, _ := strconv.ParseUint(val[3:5], 16, 8)
				b, _ := strconv.ParseUint(val[5:7], 16, 8)
				color[0] = byte(r)
				color[1] = byte(g)
				color[2] = byte(b)
			} else if len(val) == 9 && val[0] == '#' {
				r, _ := strconv.ParseUint(val[1:3], 16, 8)
				g, _ := strconv.ParseUint(val[3:5], 16, 8)
				b, _ := strconv.ParseUint(val[5:7], 16, 8)
				a, _ := strconv.ParseUint(val[7:9], 16, 8)
				color[0] = byte(r)
				color[1] = byte(g)
				color[2] = byte(b)
				color[3] = byte(a)
			}
		}
	}

	return key, color, nil
}
