APP_NAME = sintax
GO = go

.PHONY: build run compile install clean test repl example help

## build: Build the sintax binary
build:
	$(GO) build -o $(APP_NAME) .

## run: Run a .sx file (usage: make run FILE=examples/tests.sx)
run: build
	./$(APP_NAME) $(FILE)

## compile: Compile a .sx file to native binary
compile: build
	./$(APP_NAME) build $(FILE)

## install: Install sintax globally via go install
install:
	$(GO) install .

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

## help: Show available commands
help:
	@echo "Sintax"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
