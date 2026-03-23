APP_NAME = sintax
GO = go

.PHONY: build run install clean test repl example

## build: Build the sintax binary
build:
	$(GO) build -o $(APP_NAME) .

## run: Run a .sx file (usage: make run FILE=examples/habari.sx)
run: build
	./$(APP_NAME) $(FILE)

## compile: Compile a .sx file to native binary (usage: make compile FILE=examples/habari.sx)
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
	@echo "=== habari.sx ===" && ./$(APP_NAME) examples/habari.sx
	@echo ""
	@echo "=== kamusi.sx ===" && ./$(APP_NAME) examples/kamusi.sx
	@echo ""
	@echo "=== kikokotoo.sx ===" && echo "10\n+\n5" | ./$(APP_NAME) examples/kikokotoo.sx

## test: Run Go tests
test:
	$(GO) test ./...

## clean: Remove build artifacts
clean:
	rm -f $(APP_NAME)

## help: Show available commands
help:
	@echo "Sintax - Lugha ya programu kwa Kiswahili"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
