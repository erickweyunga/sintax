# Sintax

A gradually-typed, compiled programming language with clean, minimal syntax.

Sintax compiles to **native binaries** via LLVM, or runs interpreted for quick development.

## Quick Start

```bash
# Install
go install github.com/erickweyunga/sintax@latest

# Run a program
sintax hello.sx

# Compile to native binary
sintax build hello.sx
./hello

# Interactive REPL
sintax
```

## Example

```
fn (str name) str greet:
    return "Hello, {name}!"

>> greet("World")

fruits = ["Mango", "Banana", "Pineapple"]
for fruit in fruits:
    >> fruit

if len(fruits) > 2:
    >> "Many fruits!"
else:
    >> "Few fruits"
```

## Features

### Types

```
num x = 42              -- number (typed)
str name = "Eric"       -- string (typed)
bool active = true      -- boolean (typed)
list items = [1, 2, 3]  -- list (typed)
dict data = {"a": 1}    -- dict (typed)
y = "dynamic"           -- untyped (dynamic)
```

### Functions

```
-- Typed params and return
fn (num a, num b) num add:
    a + b

-- Untyped (dynamic)
fn (a, b) multiply:
    a * b

-- Recursive
fn (num n) num factorial:
    if n <= 1:
        return 1
    else:
        return n * factorial(n - 1)
```

### Control Flow

```
-- if/else
if x > 10:
    >> "big"
else:
    >> "small"

-- match/case
match day:
    case "Monday":
        >> "First day"
    _:
        >> "Other day"

-- while
while i < 10:
    >> i
    i += 1

-- for..in
for n in range(10):
    >> n

-- break (0) / continue (1)
for n in range(100):
    if n == 50:
        0
    if n % 2 == 0:
        1
    >> n
```

### Operators

```
-- Arithmetic
+ - * / % **

-- Comparison
== != > < >= <=

-- Logical
and  or  not

-- Membership
"a" in ["a", "b"]      -- true

-- Unary
-5  +5

-- Compound assignment
x += 1  x -= 1  x *= 2  x /= 2
```

### Collections

```
-- Lists
items = [1, 2, 3]
>> items[0]
push(items, 4)
pop(items, 0)
>> len(items)

-- Dicts
person = {"name": "Eric", "age": 25}
>> person["name"]
person["country"] = "Tanzania"
>> keys(person)
>> has(person, "name")
```

### Strings

```
-- Interpolation
name = "Eric"
>> "Hello {name}!"

-- Escape sequences
>> "line1\nline2"
>> "tab\there"
>> "quote \"here\""

-- Concatenation
>> "Hello" + " " + "World"
```

### Comments

```
-- single line comment

--{
multiline
comment
}--
```

### Imports

```
-- Namespaced
use "math"
>> math/sqrt(16)

-- Wildcard (all functions directly available)
use "math/*"
>> sqrt(16)

-- Specific function
use "math/sqrt"
>> sqrt(16)
```

## Built-in Functions

| Function | Purpose |
|----------|---------|
| `print(x, y)` | Print with multiple args |
| `>> x` | Print shorthand |
| `input("prompt")` | Read user input |
| `type(x)` | Get type name |
| `len(x)` | Length of list/str/dict |
| `push(list, item)` | Append to list |
| `pop(list, idx)` | Remove from list |
| `range(n)` / `range(a, b)` | Range of numbers |
| `keys(dict)` | Dict keys |
| `values(dict)` | Dict values |
| `has(dict, key)` | Check key exists |
| `num(x)` | Convert to number |
| `str(x)` | Convert to string |
| `bool(x)` | Convert to boolean |

## Standard Library

```bash
sintax lib              # list all libraries
sintax lib math         # show math functions
```

### math

sqrt, abs, floor, ceil, round, min, max, pi, e, sin, cos, tan, asin, acos, atan, log, log2, log10, exp, pow, cbrt, radiani, digrii, nasibu, nasibu_kati, ishara, clamp, jumla, wastani, ubaguzi, kupotoka, kati, factorial, mchanganyiko, mpangilio, asilimia

### string

gawa (split), unganisha (join), badilisha (replace), punguza (trim), kubwa (upper), ndogo (lower), ina_neno (contains), anza_na (starts_with), isha_na (ends_with)

### os

soma (read), andika (write), ipo (exists), ongeza (append), futa (delete), orodha (list_dir), tengeneza_saraka (mkdir), mazingira (getenv), weka_mazingira (setenv), tekeleza (exec), toka (exit), mfumo_jina (os_name), saa (time), cwd (cwd)

## Architecture

```
sintax file.sx          Interpreter (tree-walking)
sintax build file.sx    Compiler (AST → LLVM IR → clang → native binary)
```

### Pipeline

```
Source (.sx)
    ↓
Preprocessor (indentation → braces, use directives)
    ↓
Parser (participle → AST)
    ↓
    ├── sintax file.sx     → Tree-walking evaluator
    └── sintax build       → LLVM IR → clang -O2 → native binary
```

## Building from Source

```bash
git clone https://github.com/erickweyunga/sintax.git
cd sintax
make build
```

### Requirements

- Go 1.21+
- clang (for `sintax build`)
- bdw-gc (optional, garbage collection for compiled binaries)

```bash
# macOS
brew install bdw-gc

# Ubuntu
sudo apt install clang libgc-dev
```

## Makefile

```bash
make build      # Build the binary
make run FILE=f # Run a .sx file
make compile FILE=f # Compile to native
make test       # Run Go tests
make clean      # Remove binary
make help       # Show commands
```

## License

MIT — Eric Kweyunga
