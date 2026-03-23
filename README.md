# Sintax

A gradually-typed, compiled programming language with Swahili syntax.

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
-- hello.sx
unda (tungo jina) tungo salamu:
    rudisha "Habari, {jina}!"

>> salamu("Dunia")

matunda = ["Embe", "Ndizi", "Nanasi"]
kwa tunda ktk matunda:
    >> tunda

kama urefu(matunda) > 2:
    >> "Matunda mengi!"
sivyo:
    >> "Matunda machache"
```

## Features

### Types
```
nambari x = 42          -- number (typed)
tungo jina = "Erick"    -- string (typed)
buliani hai = kweli     -- boolean (typed)
safu items = [1, 2, 3]  -- list (typed)
kamusi data = {"a": 1}  -- dict (typed)
y = "dynamic"           -- untyped (dynamic)
```

### Functions
```
unda (nambari a, nambari b) nambari jumla:
    a + b                -- implicit return

unda (n) factorial:
    kama n <= 1:
        rudisha 1
    sivyo:
        rudisha n * factorial(n - 1)
```

### Control Flow
```
-- if/else
kama x > 10:
    >> "kubwa"
sivyo:
    >> "ndogo"

-- switch/case
chagua siku:
    ikiwa "Jumatatu":
        >> "Siku ya kwanza"
    _:
        >> "Siku nyingine"

-- while loop
wkt i < 10:
    >> i
    i += 1

-- for loop
kwa n ktk masafa(10):
    >> n

-- break/continue
kwa n ktk masafa(100):
    kama n == 50:
        0               -- break
    kama n % 2 == 0:
        1               -- continue
    >> n
```

### Operators
```
-- Arithmetic
+ - * / % **

-- Comparison
== != > < >= <=

-- Logical
na  au  si              -- and, or, not

-- Membership
"a" ktk ["a", "b"]     -- kweli

-- Compound assignment
x += 1  x -= 1  x *= 2  x /= 2
```

### Collections
```
-- Lists (safu)
items = [1, 2, 3]
>> items[0]
ongeza(items, 4)
ondoa(items, 0)

-- Dicts (kamusi)
mtu = {"jina": "Erick", "umri": 25}
>> mtu["jina"]
mtu["nchi"] = "Tanzania"
kwa k ktk mtu:
    >> k
```

### Built-in Functions
| Function | Purpose |
|----------|---------|
| `andika(x, y)` | Print with multiple args |
| `>> x` | Print shorthand |
| `soma("prompt")` | Read user input |
| `aina(x)` | Get type name |
| `urefu(x)` | Length of list/string/dict |
| `ongeza(list, item)` | Append to list |
| `ondoa(list, idx)` | Remove from list |
| `masafa(n)` / `masafa(a, b)` | Range of numbers |
| `funguo(dict)` | Dict keys |
| `thamani(dict)` | Dict values |
| `ina(dict, key)` | Check if key exists |
| `nambari(x)` | Convert to number |
| `tungo(x)` | Convert to string |
| `buliani(x)` | Convert to boolean |

### String Features
```
-- Interpolation
jina = "Erick"
>> "Habari {jina}!"

-- Escape sequences
>> "mstari\nmpya"
>> "tab\there"
>> "nukuu \"hizi\""

-- Concatenation
>> "Habari" + " " + "Dunia"
```

### Comments
```
-- single line comment

--{
multiline
comment
}--
```

## Architecture

```
sintax file.sx          Interpreter (tree-walking)
sintax build file.sx    Compiler (AST → LLVM IR → clang → native binary)
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
- bdw-gc (optional, for garbage collection in compiled binaries)

```bash
# macOS
brew install bdw-gc

# Ubuntu
sudo apt install clang libgc-dev
```

## License

MIT
