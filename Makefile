include make.conf

EXECUTABLE=go-x11proto
TEMPFILES=*.tmp .*.tmp .tmp
SUBDIRS := tests

test:
	@if $(GO) test -v $(PACKAGE)/... ; then echo "=== Test okay ===" ; else echo " ==== self-test failed === "; exit 1 ; fi

compile:
	$(MAKE) -C core
	for d in $(SUBDIRS) ; do $(MAKE) -C $$d compile ; done

fmt:
	$(GO) fmt $(PACKAGE)/...

clean:
	for d in $(SUBDIRS) ; do $(MAKE) -C $$d clean ; done
	rm -Rf $(EXECUTABLE) $(TEMPFILES) tetris64 simple

simple:
	$(GO) build -o simple ./demo/simple

test-simple: simple
	$(GO) build -o simple ./demo/simple
	./simple

tetris:
	$(GO) run ./demo/tetris64

tetris64:
	$(GO) build -o tetris64 ./demo/tetris64
