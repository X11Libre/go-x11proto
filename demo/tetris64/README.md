# C64 Tetris demo

A recreation of the 1988 Mirrorsoft / Andromeda **Commodore 64 Tetris**,
rendered directly over the X11 wire protocol using `go-x11proto` (no Xlib).

## Run

```sh
go run ./demo/tetris64
```

Run it from the repository root so the on-disk assets under
`demo/tetris64/assets/` are picked up. If they are not found, the program falls
back to the embedded FHD colour frame.

## Controls

| Key            | Action                                   |
| -------------- | ---------------------------------------- |
| Arrows / `H` `L` | move left / right                      |
| Down / `J`     | soft drop                                |
| Up / `K`       | rotate                                   |
| `Enter`        | hard drop (also restarts after game over)|
| `Space`        | pause / resume                           |
| `F1`           | help page (loader & game; pauses)        |
| `C`            | toggle colour / mono theme               |
| `G`            | toggle landing preview (ghost grid)      |
| `F`            | toggle fullscreen                        |
| `+` / `-`      | cycle resolution                         |
| `Q`            | quit                                     |

## Assets

- Backgrounds live under `assets/<theme>/<res>/` (`frame.png`, `loader.png`),
  for themes `color` and `mono` and resolutions `FHD`, `WQHD`, `UHD4K`.
  They are uploaded once as server-side pixmaps.
- Score/lines digits come from greyscale master glyphs in
  `assets/glyph-masters/`, scaled per resolution and tinted to match the
  background's score text.
- Music: `assets/music/tetris.sid`, played via `sidplayfp` if it is installed.
