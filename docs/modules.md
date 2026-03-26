# Modules

Sintax has a standard library of modules and supports importing user modules.

## Importing Stdlib Modules

Stdlib modules are prefixed with `std/`.

**Namespaced import** — call functions as `module/function()`:

```
use "std/math"

>> math/sqrt(16)     -- 4
>> math/floor(3.7)   -- 3
```

**Specific import** — import one function:

```
use "std/math/sqrt"

>> sqrt(16)      -- 4
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

-- Environment
>> os/getenv("HOME")

-- Process
output = os/exec("ls -la")
>> os/cwd()

-- Time
>> os/time()    -- unix timestamp
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
