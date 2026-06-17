[ ] FHD mono theme still looking blurred      -- could NOT reproduce; renders sharp from disk assets. (the embedded-320 fallback tiles/looks bad only when the per-res asset is missing.) please recheck
[ ] UHD4K mono theme still looking blurred     -- likewise, please recheck
[ ] WQHD mono theme still looking blurred      -- likewise, please recheck
[ ] drop the old 300px theme - not needed anymore   -- CLARIFY: is "300px" the 320x200 resolution? (it's currently the only embedded fallback + small-screen default)
[x] next stone preview is in the wrong place - not where it was in the original game   -- repositioned (nx/ny) from the original screenshots, per resolution
[x] highscore and line counter aren't positioned correctly   -- measured per-frame score/lines positions; 5-digit score, 48px-style advance, clear-rects fixed (no more bleed/overlap). verified 320/FHD/WQHD
[ ] stones should look a bit more smoothed (like on the original game screenshots)
[ ] score and line counter should be right aligned. move it a little bit more to the right
[ ] add a help page - both within loader as well as game - should pause the game - use a border like the game area's border
