# Getting Started with Sintax

Sintax is a strictly-typed, compiled programming language with clean, minimal syntax.

## Installation

```bash
git clone https://github.com/erickweyunga/sintax.git
cd sintax
make build
make install
```

## Running Programs

Sintax is a compiled language. The default command compiles and runs your program:

```bash
sintax hello.sx          # Compile and run
sintax build hello.sx     # Compile to native binary
sintax check hello.sx     # Analyze without running
sintax eval hello.sx      # Interpret (dev only)
sintax                    # Interactive REPL
```

When you run `sintax hello.sx`, the compiler compiles your code and executes it. Use `sintax build hello.sx` to produce a standalone binary you can distribute.

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

- [Data Types](data-types.md) -- Numbers, strings, booleans, lists, dicts, void, fn
- [Variables](variables.md) -- Type-inferred variables
- [Functions](functions.md) -- Typed functions with required return types
- [Control Flow](control-flow.md) -- if/else, match/case, loops, catch
- [Collections](collections.md) -- Lists and dicts in depth
- [Strings](strings.md) -- String functions and methods
- [Modules](modules.md) -- Importing stdlib and user modules
- [Error Handling](error-handling.md) -- error(), err(), catch, and match
