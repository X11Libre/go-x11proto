Pure golang X11 protocol library
================================

A pure golang X11 protocol library (yet just client-side), which is
fully utilizing goroutines for parallel processing.

DISCLAIMER: still work-in-progress

Providing both (xcb-style) low-level access, was well as (Xlib-style)
high-level API. Developers can choose which one to use individually
(even both at the same time). Also supports both little and big endian mode.

In the future, also adding lots an X11 protocol test suite step by step.


To do
-----

* implement lots of missing X11 calls
* implement extensions

License
-------

ALGPLv3+
