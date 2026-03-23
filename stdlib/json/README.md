# json

JSON parse and stringify for Sintax.

## Usage

```
use "json"
data = json/parse("{\"name\": \"Eric\"}")
>> data["name"]

>> json/stringify({"x": 1, "y": 2})
```

Or import everything:

```
use "json/*"
data = parse("{\"name\": \"Eric\", \"age\": 25}")
>> stringify(data)
```

## Functions

| Function | Description |
|----------|-------------|
| `parse(str)` | Parse JSON string to Sintax value |
| `stringify(value)` | Convert value to compact JSON string |
| `pretty(value)` | Convert value to indented JSON string |

## Type Mapping

| JSON | Sintax |
|------|--------|
| `number` | `num` |
| `string` | `str` |
| `boolean` | `bool` |
| `null` | `null` |
| `array` | `list` |
| `object` | `dict` |

## Examples

```
use "json/*"

-- Parse API response
response = parse("{\"users\": [{\"name\": \"Eric\"}, {\"name\": \"John\"}]}")
>> response["users"]

-- Build and stringify
data = {"name": "Eric", "scores": [95, 87, 92]}
>> stringify(data)
-- {"name":"Eric","scores":[95,87,92]}

>> pretty(data)
-- {
--   "name": "Eric",
--   "scores": [
--     95,
--     87,
--     92
--   ]
-- }

-- Round trip
original = {"key": "value"}
text = stringify(original)
restored = parse(text)
>> restored["key"]   -- value
```
