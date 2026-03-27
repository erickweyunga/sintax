PREFIX = $(HOME)/.sintax

.PHONY: build install clean

build:
	go build -o sintax .

install: build
	@mkdir -p $(PREFIX)/runtime/gc/include/private
	@mkdir -p $(PREFIX)/runtime/gc/include/gc
	@mkdir -p $(PREFIX)/stdlib
	@echo "Installing Sintax to $(PREFIX)..."
	@cp runtime/runtime.c runtime/runtime.h runtime/native.c $(PREFIX)/runtime/
	@cp runtime/gc/gc_all.c $(PREFIX)/runtime/gc/
	@cp -r runtime/gc/include/* $(PREFIX)/runtime/gc/include/
	@cp runtime/gc/*.c runtime/gc/*.h $(PREFIX)/runtime/gc/ 2>/dev/null; true
	@cp stdlib/*.sx $(PREFIX)/stdlib/
	@install -m 755 sintax /usr/local/bin/sintax 2>/dev/null || install -m 755 sintax $(PREFIX)/sintax
	@echo "Done! Run: sintax --help"

clean:
	rm -f sintax
