# Getting Started with Sintax

Sintax is a gradually-typed, compiled programming language with clean, minimal syntax.

## Installation

```bash
git clone https://github.com/erickweyunga/sintax.git
cd sintax
make build
make install
```

## Running Programs

```bash
sintax hello.sx          # Interpret
sintax run hello.sx       # Compile and run (cached)
sintax build hello.sx     # Compile to native binary
sintax check hello.sx     # Analyze without running
sintax                    # Interactive REPL
```

## Your First Program

Create a file called `hello.sx`:

```
>> "Hello, World!"
```

Run it:

```bash
sintax hello.sx
```

`>>` is the print operator. It prints any value followed by a newline.

## What's Next

- [Data Types](data-types.md) — Numbers, strings, booleans, lists, dicts
- [Variables](variables.md) — Dynamic and typed variables
- [Functions](functions.md) — Defining and calling functions
- [Control Flow](control-flow.md) — if/else, match/case, loops
- [Collections](collections.md) — Lists and dicts in depth
- [Strings and Quotes](quotes.md) — Single vs double quotes, interpolation
- [Modules](modules.md) — Importing stdlib and user modules
- [Error Handling](error-handling.md) — error(), err(), and match
