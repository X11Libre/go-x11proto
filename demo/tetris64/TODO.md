[x] FHD mono theme still looking blurred  --> regenerated mono frames as greyscale of the (sharp) colour frames; was posterized to ~3 grey levels, now full tonal range
[x] UHD4K mono theme still looking blurred     -- same fix (regenerated from colour)
[x] WQHD mono theme still looking blurred      -- same fix (regenerated from colour)   (note: mono *.xpm are now stale 3-level; game uses the PNGs)
[x] drop the old 320x200 theme - not needed anymore, use the next bigger one as fallback instead   -- removed 320 from resolutions/layouts + deleted assets; FHD colour frame is now the embedded fallback
[x] next stone preview is in the wrong place - not where it was in the original game   -- repositioned (nx/ny) from the original screenshots, per resolution
[x] highscore and line counter aren't positioned correctly   -- measured per-frame score/lines positions; 5-digit score, 48px-style advance, clear-rects fixed (no more bleed/overlap). verified 320/FHD/WQHD
[x] stones should look a bit more smoothed (like on the original game screenshots)   -- subtle per-colour horizontal scanlines on board/current/next stones
[x] score and line counter should be right aligned. move it a little bit more to the right   -- right-aligned to a per-res numRight edge, nudged right (tunable)
[x] add a help page - both within loader as well as game - should pause the game - use a border like the game area's border   -- F1 toggles in loader+game, pauses (gravity halts), hides board so it's not occluded, well-style thick border, updated keys
[x] upload assets as server-side images and use those, if not already done yet   -- already done: frame/loader/glyphs uploaded as pixmaps (Image.Upload), reused via SetWindowBackgroundPixmap + CopyArea (no per-frame pixel uploads)
[x] add little readme with help   -- demo/tetris64/README.md (run + controls + assets)
[x] crop the VIC black border out of the bg images; let the bg window's black background generate it; pixmaps keep the original C64 8:5 aspect (zoomed)   -- cropped all frame/loader PNGs (color+mono, FHD/WQHD/UHD4K) to the 8:5 framebuffer (FHD 1728x1080, WQHD 2304x1440, 4K 3456x2160); bgWin is now framebuffer-sized & centered in the black frame window; layout X coords shifted by -border (96/128/192). verified FHD+WQHD
[ ] add current resolution / theme to main/frame window title
