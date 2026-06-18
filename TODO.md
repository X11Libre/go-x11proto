[~] tetris: do pause window via extra tk.Window ?   -- answered: no. The pause overlay is a trivial centred "PAUSE" label on the board; a dedicated window/lifecycle would be more code for no real gain (unlike the help page). Kept as a drawGame overlay.
[x] add what's needed to integrate our XTS into Xserver build-time tests (at least via github CI, but even better meson tests)
    -- done (entrypoint + docs): contrib/xts/run-xts.sh points the suite at any
       -displayfd-capable server binary via XTS_XSERVER; docs/xts-ci.md covers
       the GitHub Actions step to drop into the xserver build job and a meson
       test() snippet; ci.yml has an xts-against-server job as the live template.
       Remaining work is on the xserver repo side: add that step/test there.
[x] extend XTS to test the operations we're currently using for the demos
    -- done: xts/tk_test.go exercises the tk Window / Drawable / GC / Pixmap /
       InternAtom / SetBackgroundPixmap operations and the Label widget, i.e. the
       tk layer the demos are built on. Coverage matrix in docs/tk-coverage.md.
[x] add requests / rpc and command line tool for Xnamespace protocol extension: https://github.com/X11Libre/xserver/pull/3103
    -- done: proto/ext/namespace implements all ten X-NAMESPACE requests
       (QueryVersion, List/Create/Delete/Query/SetNamespaceFlags,
       Add/Remove/ListAuthTokens, GetClientNamespace) with the exact wire
       layout from the server's namespaceproto.h; cmd/xnamespace is the CLI
       client. Byte-exact offline tests; verified end-to-end against the
       xlibre Xvfb built with the extension.
