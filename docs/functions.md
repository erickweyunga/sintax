# Functions

## Basic Functions

Functions are defined with `fn`, parameters in parentheses, a name, and a body.

```
fn (a, b) add:
    a + b

>> add(5, 3)    -- 8
```

The last expression in a function body is its return value (implicit return).

## Typed Functions

Add types to parameters and return values for safety.

```
fn (num x, num y) num multiply:
    x * y

>> multiply(4, 5)    -- 20
```

The analyzer checks types at call sites:

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
fn () greet:
    >> "Hello!"

greet()
```

## Functions as Values

Functions can be stored in variables and passed around.

```
fn (x) double:
    x * 2

apply = double
>> apply(5)    -- 10
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
