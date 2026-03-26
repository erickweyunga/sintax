# Data Types

Sintax has eight core data types.

## num

Numbers are 64-bit floating point. Integers and decimals use the same type.

```
>> 42
>> 3.14
>> -10
>> 0
```

Arithmetic:

```
>> 10 + 5       -- 15
>> 10 - 3       -- 7
>> 4 * 7        -- 28
>> 20 / 4       -- 5
>> 17 % 5       -- 2
>> 2 ** 10      -- 1024
```

Compound assignment:

```
x = 10
x += 5     -- x is 15
x -= 3     -- x is 12
x *= 2     -- x is 24
x /= 4    -- x is 6
```

## str

Strings are sequences of characters. Sintax has two kinds of string literals.

**Double-quoted** -- supports escape sequences and interpolation:

```
>> "Hello, World!"
name = "Eric"
>> "Hello, {name}!"     -- Hello, Eric!
>> "line1\nline2"        -- newline
>> "tab\there"           -- tab
>> "say \"hi\""          -- escaped quote
```

**Single-quoted** -- raw, no escapes or interpolation:

```
>> 'Hello, World!'
>> '{"key": "value"}'    -- great for JSON
>> 'no {interpolation}'  -- literal braces
>> 'backslash\ stays'    -- literal backslash
```

String concatenation:

```
>> "Hello" + " " + "World"    -- Hello World
```

String indexing:

```
s = "hello"
>> s[0]     -- h
>> s[4]     -- o
>> len(s)   -- 5
```

## bool

Booleans are `true` or `false`.

```
>> true
>> false
>> 5 > 3         -- true
>> 5 == 5         -- true
>> not true       -- false
```

Logical operators:

```
>> true and true      -- true
>> true and false     -- false
>> false or true      -- true
>> not false          -- true
```

## list

Lists are ordered collections of any values.

```
numbers = [1, 2, 3]
mixed = [1, "hello", true, [4, 5]]
empty = []
```

Access and modify:

```
items = ["a", "b", "c"]
>> items[0]         -- a
>> items[2]         -- c
items[1] = "B"      -- replace
>> len(items)        -- 3
```

Add and remove:

```
items = [1, 2, 3]
push(items, 4)       -- [1, 2, 3, 4]
pop(items, 0)        -- removes first element
```

Iterate:

```
for item in [10, 20, 30]:
    >> item
```

List methods:

```
>> [3, 1, 2].len()           -- 3
>> [3, 1, 2].contains(2)     -- true
>> [3, 1, 2].reverse()       -- [2, 1, 3]
>> ["a", "b"].join("-")      -- a-b
```

Functional methods:

```
double = fn(x) -> x * 2
>> [1, 2, 3].map(double)               -- [2, 4, 6]
>> [1, 2, 3, 4].filter(fn(x) -> x > 2) -- [3, 4]
>> [1, 2, 3].reduce(fn(a, x) -> a + x, 0)  -- 6
```

## dict

Dicts are key-value pairs. Keys are always strings. Insertion order is preserved.

```
person = {"name": "Eric", "age": 25}
empty = {}
```

Access and modify:

```
person = {"name": "Eric", "age": 25}
>> person["name"]            -- Eric
person["age"] = 26           -- update
person["country"] = "TZ"     -- add new key
```

Dict functions:

```
>> keys(person)              -- ["name", "age", "country"]
>> values(person)            -- ["Eric", 26, "TZ"]
>> has(person, "name")       -- true
>> has(person, "email")      -- false
>> len(person)               -- 3
```

Iterate:

```
for key in person:
    print(key, "=", person[key])
```

Dict methods:

```
>> {"a": 1}.len()        -- 1
>> {"a": 1}.has("a")     -- true
>> {"a": 1}.keys()       -- ["a"]
>> {"a": 1}.values()     -- [1]
```

## void

`void` indicates that a function returns nothing. It is used only as a function return type, not as a value you can assign.

```
fn (str msg) void log:
    >> msg

log("hello")    -- prints: hello
```

## fn

`fn` is the type for function values. Use it when a function returns another function or accepts a function as a parameter.

```
fn () fn make_counter:
    count = 0
    fn () num inc:
        count = count + 1
        return count
    return inc

counter = make_counter()
>> counter()    -- 1
>> counter()    -- 2
```

## Union Types

Functions can return multiple possible types using `|`:

```
fn (str s) dict | list | str | num | bool parse:
    use "std/json"
    return json/parse(s)
```

Union types are used in function return type declarations when the return value may be one of several types.

## null

`null` represents the absence of a value.

```
>> null
>> type(null)    -- null
```

## Truthiness

Every value has a truthy or falsy interpretation:

| Value | Truthy? |
|-------|---------|
| `false` | No |
| `null` | No |
| `0` | No |
| `""` | No |
| `[]` | No |
| `{}` | No |
| Everything else | Yes |

```
if "hello":
    >> "truthy"     -- prints

if 0:
    >> "never"      -- doesn't print

if [1, 2]:
    >> "truthy"     -- prints
```

## Type Checking

Use `type()` to get the type name of any value:

```
>> type(42)        -- num
>> type("hello")   -- str
>> type(true)      -- bool
>> type([1, 2])    -- list
>> type({})        -- dict
>> type(null)      -- null
```

## Type Conversion

Convert between types:

```
>> num("42")       -- 42
>> num(true)       -- 1
>> num(false)      -- 0

>> str(42)         -- "42"
>> str(true)       -- "true"

>> bool(1)         -- true
>> bool(0)         -- false
>> bool("")        -- false
>> bool("hello")   -- true
```
