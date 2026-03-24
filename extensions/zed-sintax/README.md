# Sintax for Zed

Syntax highlighting and language support for [Sintax](https://github.com/erickweyunga/sintax) in [Zed](https://zed.dev).

## Features

- Syntax highlighting for all Sintax keywords, types, operators
- Bracket matching
- Code outline (functions)
- Auto-indentation
- Single and double quoted strings
- Comment toggling (`--`)

## Installation

1. Open Zed
2. Go to Extensions (⇧⌘X)
3. Search for "sintax"
4. Install

Or install from source:

```bash
cd extensions/zed-sintax
# Link to Zed extensions directory
ln -s $(pwd) ~/.config/zed/extensions/installed/sintax
```

## Requirements

Requires the `tree-sitter-sintax` grammar. The extension will download it automatically.

## Supported Syntax

```
-- Comments
use "std/math"

fn (num a, num b) num add:
    return a + b

x = add(5, 3)
>> x

if x > 4:
    >> "Big"
else:
    >> "Small"

for n in range(10):
    >> n

items = [1, 2, 3]
>> items.map(fn(x) -> x * 2)

match type(x):
    case "num":
        >> "number"
    _:
        >> "other"
```
