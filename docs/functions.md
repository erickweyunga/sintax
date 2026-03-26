# Functions

## Basic Functions

Functions are defined with `fn`, typed parameters in parentheses, a return type, a name, and a body. All functions must have a return type and all parameters must have a type.

```
fn (num a, num b) num add:
    a + b

>> add(5, 3)    -- 8
```

The last expression in a function body is its return value (implicit return).

## Return Types

Every function must declare a return type. Available return types are: `num`, `str`, `bool`, `list`, `dict`, `fn`, `void`, or union types with `|`.

```
fn (num x, num y) num multiply:
    x * y

fn (str name) str greet:
    "Hello, " + name + "!"

fn (num n) bool is_positive:
    n > 0
```

The compiler checks that the returned value matches the declared type.

## Void Functions

Use `void` for functions that do not return a value:

```
fn (str msg) void log:
    >> msg

fn (num ms) void sleep:
    use "std/os"
    os/sleep(ms)
```

## Union Return Types

When a function can return different types, use `|` to declare a union return type:

```
fn (str s) dict | list | str | num | bool parse:
    use "std/json"
    return json/parse(s)
```

## Typed Parameters

All parameters must have a type annotation:

```
fn (num x, num y) num multiply:
    x * y

>> multiply(4, 5)    -- 20
```

The compiler checks types at call sites:

```
>> multiply("a", "b")    -- Error: 'multiply' arg 1 expects num, got str
```

## Explicit Return

Use `return` to return early or make intent clear.

```
fn (num n) num factorial:
    if n <= 1:
        return 1
    return n * factorial(n - 1)

>> factorial(5)    -- 120
```

## Functions Without Parameters

```
fn () void greet:
    >> "Hello!"

greet()
```

## Public and Private Functions

Functions are **private by default**. Use the `pub` keyword to export a function from a module:

```
-- Private: only accessible within this file
fn (num a, num b) num add:
    a + b

-- Public: accessible when this module is imported
pub fn (num a, num b) num subtract:
    a - b
```

When another file imports this module, only `pub` functions are available:

```
use "mymodule.sx"

>> mymodule/subtract(10, 3)    -- 7
-- mymodule/add(1, 2)          -- Error: add is not exported
```

## Functions as Values

Functions can be stored in variables and passed around.

```
fn (num x) num double:
    x * 2

apply = double
>> apply(5)    -- 10
```

## Returning Functions

Use `fn` as the return type when a function returns another function:

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

## Lambdas

Short anonymous functions with `fn(params) -> expression`.

```
double = fn(x) -> x * 2
>> double(5)    -- 10

add = fn(a, b) -> a + b
>> add(3, 4)    -- 7
```

Lambdas are great with list methods:

```
>> [1, 2, 3].map(fn(x) -> x * 2)           -- [2, 4, 6]
>> [1, 2, 3, 4].filter(fn(x) -> x > 2)     -- [3, 4]
```

## Recursion

Functions can call themselves.

```
fn (num n) num fib:
    if n <= 1:
        return n
    return fib(n - 1) + fib(n - 2)

>> fib(10)    -- 55
```

## Forward Declarations

Functions can be called before they are defined in the same scope.

```
>> is_even(4)    -- true

fn (num n) bool is_even:
    if n == 0:
        return true
    return is_odd(n - 1)

fn (num n) bool is_odd:
    if n == 0:
        return false
    return is_even(n - 1)
```

## Built-in Functions

| Function | Description |
|----------|-------------|
| `print(a, b, ...)` | Print multiple values with spaces |
| `>> value` | Print a single value |
| `type(value)` | Get type name as string |
| `len(value)` | Length of string, list, or dict |
| `push(list, item)` | Append to list |
| `pop(list, index)` | Remove from list by index |
| `range(n)` | List [0, 1, ..., n-1] |
| `range(a, b)` | List [a, a+1, ..., b-1] |
| `input()` | Read line from stdin |
| `input(prompt)` | Print prompt, then read |
| `keys(dict)` | List of dict keys |
| `values(dict)` | List of dict values |
| `has(dict, key)` | Check if dict has key |
| `num(value)` | Convert to number |
| `str(value)` | Convert to string |
| `bool(value)` | Convert to boolean |
| `error(message)` | Create an error value |
| `err(value)` | Check if value is an error |
| `slice(value, start, end)` | Slice a string or list |
| `index_of(value, item)` | Find index of item in string or list |
| `sleep(ms)` | Sleep for milliseconds |
| `exit(code)` | Exit the program |
