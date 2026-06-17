[x] stone landing preview (in the game area) just as grid instead of full color
[x] pause via space key
[x] line and score counter yet missing       -- rendered now (drawNumber)
[x] switch between mono and color theme (eg. C key)
[ ] FHD mono theme still looking blurred      -- could NOT reproduce; renders sharp from disk assets. (the embedded-320 fallback tiles/looks bad only when the per-res asset is missing.) please recheck
[ ] UHD4K mono theme still looking blurred     -- likewise, please recheck
[ ] WQHD mono theme still looking blurred      -- likewise, please recheck
[x] theme: move .sid file under music subdir   -- assets/music/tetris.sid
[x] when resource embedding is enabled: also embed the .sid file and write it out to tempfile for sidplayfp  -- already done (sidData embed -> startMusic writes tmpfile)
[ ] drop the old 300px theme - not needed anymore   -- CLARIFY: is "300px" the 320x200 resolution? (it's currently the only embedded fallback + small-screen default)
[ ] next stone preview is in the wrong place - not where it was in the original game   -- needs the original positions (nx/ny per resolution)
[~] highscore and line counter aren't positioned correctly - and there's an useless extra line   -- removed the extra "level" line; exact score/lines positions still off (digits overlap the baked-in labels), needs per-resolution coords
[x] highscore and line counter don't actually count yet   -- they DO count (game.CheckLines updates Score/Lines); please confirm
