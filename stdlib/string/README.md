# string

String manipulation functions for Sintax.

## Usage

```
use "string"
>> string/kubwa("hello")    -- HELLO
>> string/gawa("a,b,c", ",")  -- ["a", "b", "c"]
```

Or import everything:

```
use "string/*"
>> kubwa("hello")
>> gawa("a,b,c", ",")
```

## Functions

| Function | Description | Example |
|----------|-------------|---------|
| `gawa(s, sep)` | Split string | `gawa("a-b", "-")` → `["a", "b"]` |
| `unganisha(list, sep)` | Join list to string | `unganisha(["a","b"], ",")` → `"a,b"` |
| `badilisha(s, old, new)` | Replace all occurrences | `badilisha("hello", "l", "r")` → `"herro"` |
| `punguza(s)` | Trim whitespace | `punguza("  hi  ")` → `"hi"` |
| `kubwa(s)` | Uppercase | `kubwa("hello")` → `"HELLO"` |
| `ndogo(s)` | Lowercase | `ndogo("HELLO")` → `"hello"` |
| `ina_neno(s, sub)` | Contains substring | `ina_neno("hello", "ell")` → `true` |
| `anza_na(s, prefix)` | Starts with | `anza_na("hello", "hel")` → `true` |
| `isha_na(s, suffix)` | Ends with | `isha_na("hello", "llo")` → `true` |
