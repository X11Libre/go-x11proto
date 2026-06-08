# C64 Tetris demo

A recreation of the 1988 Mirrorsoft / Andromeda **Commodore 64 Tetris**,
rendered directly over the X11 wire protocol using `go-x11proto` (no Xlib).

## Run

```sh
go run ./demo/tetris64
```

All assets are compiled into the binary, so it runs from any directory.

## Controls

| Key            | Action                                   |
| -------------- | ---------------------------------------- |
| Arrows / `H` `L` | move left / right                      |
| Down / `J`     | soft drop                                |
| Up / `K`       | rotate                                   |
| `Enter`        | hard drop (also restarts after game over)|
| `Space`        | pause / resume                           |
| `F1` / `Shift`+`H` | help page (loader & game; pauses)    |
| `C`            | toggle colour / mono theme               |
| `G`            | toggle landing preview (ghost grid)      |
| `F`            | toggle fullscreen                        |
| `+` / `-`      | cycle resolution                         |
| `Q`            | quit                                     |

## Assets

- Backgrounds live under `assets/<theme>/` (`frame.png`, `loader.png`), for
  themes `color` and `mono`, at FHD resolution. Only this FHD art is embedded
  into the binary; the WQHD and UHD4K backgrounds are produced at load time by
  upscaling it
  (Catmull-Rom, via `golang.org/x/image/draw`) and uploaded as server-side
  pixmaps. This keeps the binary small; the higher-resolution backgrounds are
  marginally softer, but the live score/lines digits stay crisp (they are
  rendered from masters at native scale, not upscaled).
- Score/lines digits come from greyscale master glyphs in
  `assets/glyphs/`, scaled per resolution and tinted to match the
  background's score text.
- Music: `assets/music/tetris.sid`, played via `sidplayfp` if it is installed.
