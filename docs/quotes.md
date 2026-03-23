# Strings and Quotes

Sintax has two types of string literals: double-quoted and single-quoted.

## Double-Quoted Strings `"..."`

Support escape sequences and string interpolation.

```
-- Escape sequences
>> "hello\nworld"     -- newline between hello and world
>> "tab\there"        -- tab between tab and here
>> "say \"hi\""       -- say "hi"
>> "back\\slash"      -- back\slash

-- String interpolation
name = "Eric"
age = 25
>> "Hello {name}!"           -- Hello Eric!
>> "{name} is {age}"         -- Eric is 25
```

### Escape Sequences

| Escape | Character |
|--------|-----------|
| `\n` | Newline |
| `\t` | Tab |
| `\\` | Backslash |
| `\"` | Double quote |
| `\r` | Carriage return |
| `\0` | Null byte |

### Interpolation

Any `{identifier}` inside a double-quoted string is replaced with the variable's value.

```
x = 42
>> "x is {x}"    -- x is 42

items = [1, 2, 3]
>> "count: {items}"    -- count: [1, 2, 3]
```

Non-identifier content in braces is kept literal:

```
>> "2 + 2 = {result}"    -- interpolates 'result'
>> "{not valid}"          -- kept as-is (spaces/symbols)
```

## Single-Quoted Strings `'...'`

Raw strings. No escape sequences, no interpolation. What you type is what you get.

```
>> '{"name": "John"}'    -- {"name": "John"}
>> 'hello\nworld'        -- hello\nworld (literal backslash-n)
>> 'no {interpolation}'  -- no {interpolation}
```

The only escape in single-quoted strings is `\'` to include a literal single quote:

```
>> 'it\'s here'    -- it's here
```

## When to Use Which

| Use case | Recommended |
|----------|-------------|
| Regular text | `"hello"` |
| Text with variables | `"Hello {name}!"` |
| JSON strings | `'{"key": "value"}'` |
| Regex patterns | `'^\d+$'` |
| Strings with many `"` | `'He said "hello"'` |
| Strings with `\n`, `\t` | `"line1\nline2"` |

## Examples

```
use "std/json/*"

-- JSON is clean with single quotes
data = parse('{"users": [{"name": "Eric"}, {"name": "John"}]}')
>> data["users"]

-- String building with double quotes
for user in data["users"]:
    name = user["name"]
    >> "Hello, {name}!"

-- Mixed
config = parse('{"debug": true, "port": 8080}')
port = config["port"]
>> "Server running on port {port}"
```
