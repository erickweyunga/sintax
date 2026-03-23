# JSON Module

Parse and generate JSON strings.

## Import

```
use "std/json"          -- namespaced: json/parse(...)
use "std/json/*"        -- wildcard: parse(...)
use "std/json/parse"    -- specific: parse(...)
```

## Functions

### parse(str) -> value

Parse a JSON string into a Sintax value.

- JSON objects become dicts
- JSON arrays become lists
- JSON numbers become nums
- JSON strings become strs
- JSON booleans become bools
- JSON null becomes null

```
use "std/json/*"

data = parse('{"name": "Eric", "age": 25}')
>> data["name"]    -- Eric
>> data["age"]     -- 25

items = parse('[1, 2, 3]')
>> items           -- [1, 2, 3]

>> parse("42")     -- 42
>> parse("true")   -- true
>> parse("null")   -- null
```

### stringify(value) -> str

Convert a Sintax value to a compact JSON string.

```
use "std/json/*"

data = {"name": "Eric", "scores": [95, 87, 92]}
>> stringify(data)
-- {"name":"Eric","scores":[95,87,92]}

>> stringify([1, 2, 3])     -- [1,2,3]
>> stringify("hello")       -- "hello"
>> stringify(42)             -- 42
>> stringify(true)           -- true
>> stringify(null)           -- null
```

### pretty(value) -> str

Convert a Sintax value to an indented JSON string (2-space indent).

```
use "std/json/*"

data = {"name": "Eric", "languages": ["Sintax", "Go", "C"]}
>> pretty(data)
```

Output:

```json
{
  "name": "Eric",
  "languages": [
    "Sintax",
    "Go",
    "C"
  ]
}
```

## Round-trip Example

```
use "std/json/*"

original = {"name": "Sintax", "version": 1}
text = stringify(original)
parsed = parse(text)
>> parsed["name"]    -- Sintax
```

## Tips

Use single-quoted strings for JSON to avoid escaping double quotes:

```
-- Clean
data = parse('{"name": "John", "age": 30}')

-- Also works but harder to read
data = parse("{\"name\": \"John\", \"age\": 30}")
```
