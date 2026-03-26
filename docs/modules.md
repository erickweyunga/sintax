# Modules

Sintax has a standard library of modules and supports importing user modules.

## Importing Stdlib Modules

Stdlib modules are prefixed with `std/`. There are exactly two import styles -- no wildcard imports.

**Namespaced import** -- call functions as `module/function()`:

```
use "std/math"

>> math/sqrt(16)     -- 4
>> math/floor(3.7)   -- 3
```

**Specific import** -- import one function:

```
use "std/math/sqrt"

>> sqrt(16)      -- 4
```

These are the only two import styles. There are no wildcard or bulk imports.

## Public and Private Functions

Functions in modules are **private by default**. Only functions marked with `pub` are accessible when the module is imported.

```
-- In mylib.sx

-- Private: only usable inside mylib.sx
fn (num x) num helper:
    x * 2

-- Public: exported when someone imports mylib
pub fn (num x) num double:
    helper(x)
```

```
-- In main.sx
use "mylib.sx"

>> mylib/double(5)     -- 10
-- mylib/helper(5)     -- Error: helper is not exported
```

## Standard Library

### std/math

Mathematical functions.

```
use "std/math"

-- Basic
>> math/sqrt(16)        -- 4
>> math/floor(3.7)      -- 3
>> math/ceil(3.2)       -- 4
>> math/round(3.5)      -- 4

-- Trigonometry
>> math/sin(0)          -- 0
>> math/cos(0)          -- 1

-- Logarithms
>> math/log(1)          -- 0
>> math/log10(100)      -- 2

-- Power
>> math/pow(2, 10)      -- 1024
>> math/cbrt(27)        -- 3

-- Random
>> math/random()        -- 0.0 to 1.0
```

### std/string

String manipulation.

```
use "std/string"

>> string/upper("hello")                    -- HELLO
>> string/lower("HELLO")                    -- hello
>> string/split("a,b,c", ",")              -- ["a", "b", "c"]
>> string/replace("hello world", "world", "Sintax")  -- hello Sintax
>> string/join(["a", "b", "c"], "-")       -- a-b-c
>> string/trim("  hello  ")                -- hello
>> string/slice("hello", 1, 3)             -- el
>> string/index_of("hello", "ll")          -- 2
>> string/reverse("hello")                 -- olleh
>> string/repeat("ha", 3)                  -- hahaha
>> string/char_code("A")                   -- 65
>> string/from_char_code(65)               -- A
```

### std/json

JSON parsing and generation.

```
use "std/json"

data = json/parse('{"name": "Eric", "age": 25}')
>> data["name"]        -- Eric

>> json/stringify(data)     -- {"name":"Eric","age":25}
>> json/pretty(data)        -- formatted with indentation
```

### std/os

File system and process operations.

```
use "std/os"

-- Files
content = os/read("file.txt")
os/write("out.txt", "hello")
>> os/exists("file.txt")     -- true
os/rename("old.txt", "new.txt")

-- Environment
>> os/getenv("HOME")

-- Process
output = os/exec("ls -la")
>> os/cwd()

-- Time
>> os/time()           -- unix timestamp
>> os/format_time(os/time())

-- Control
os/sleep(1000)         -- sleep 1 second
os/exit(0)             -- exit program
```

### std/list

List manipulation functions.

```
use "std/list"

>> list/concat([1, 2], [3, 4])      -- [1, 2, 3, 4]
>> list/insert([1, 3], 1, 2)        -- [1, 2, 3]
>> list/reverse([1, 2, 3])          -- [3, 2, 1]
>> list/index_of([10, 20, 30], 20)  -- 1
>> list/slice([1, 2, 3, 4], 1, 3)   -- [2, 3]
```

### std/dict

Dict manipulation functions.

```
use "std/dict"

>> dict/delete({"a": 1, "b": 2}, "a")              -- {"b": 2}
>> dict/merge({"a": 1}, {"b": 2})                   -- {"a": 1, "b": 2}
```

### std/regex

Regular expression functions.

```
use "std/regex"

>> regex/match("hello123", "[0-9]+")                         -- true
>> regex/find("hello123world456", "[0-9]+")                  -- ["123", "456"]
>> regex/replace("hello world", "world", "Sintax")           -- hello Sintax
```

### std/http

HTTP client functions.

```
use "std/http"

response = http/request("GET", "https://api.example.com/data", "", {})
>> response
```

## User Modules

Import other `.sx` files:

```
use "helpers.sx"

>> helpers/my_function(42)
```

Import a specific function from a user module:

```
use "helpers.sx/my_function"

>> my_function(42)
```

## How Imports Work

When you write `use "std/math"` and call `math/sqrt(16)`:

1. The preprocessor rewrites `math/sqrt(16)` to `math__sqrt(16)`
2. The stdlib `math.sx` file is loaded
3. Its `sqrt` function is registered as `math__sqrt`
4. The call resolves to the stdlib implementation
