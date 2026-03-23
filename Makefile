APP_NAME = sintax
GO = go
SINTAX_HOME ?= $(HOME)/.sintax

.PHONY: build run compile install uninstall clean test repl example help

## build: Build the sintax binary
build:
	$(GO) build -o $(APP_NAME) .

## run: Run a .sx file (usage: make run FILE=examples/tests.sx)
run: build
	./$(APP_NAME) $(FILE)

## compile: Compile a .sx file to native binary
compile: build
	./$(APP_NAME) build $(FILE)

## install: Install sintax + runtime to ~/.sintax/
install: build
	@mkdir -p $(SINTAX_HOME)/runtime
	@cp runtime/runtime.c $(SINTAX_HOME)/runtime/
	@cp runtime/runtime.h $(SINTAX_HOME)/runtime/
	@cp $(APP_NAME) $(SINTAX_HOME)/$(APP_NAME)
	@echo "Installed to $(SINTAX_HOME)/"
	@echo "Add to PATH: export PATH=\$$PATH:$(SINTAX_HOME)"

## uninstall: Remove ~/.sintax/
uninstall:
	@rm -rf $(SINTAX_HOME)
	@echo "Removed $(SINTAX_HOME)/"

## repl: Launch the interactive REPL
repl: build
	./$(APP_NAME)

## example: Run all example programs
example: build
	@echo "=== hello.sx ===" && ./$(APP_NAME) examples/hello.sx
	@echo ""
	@echo "=== dicts.sx ===" && ./$(APP_NAME) examples/dicts.sx
	@echo ""
	@echo "=== tests.sx ===" && ./$(APP_NAME) examples/tests.sx | tail -1

## test: Run Go tests
test:
	$(GO) test ./...

## clean: Remove build artifacts
clean:
	rm -f $(APP_NAME)
	rm -rf .sintax/

## help: Show available commands
help:
	@echo "Sintax"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
