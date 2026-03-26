# Variables

## Type Inference

Variables infer their type from the first assignment. Once a variable has a type, it cannot be reassigned to a different type.

```
x = 42              -- x is num (inferred)
name = "Eric"       -- name is str (inferred)
items = [1, 2, 3]   -- items is list (inferred)

x = 100             -- ok, still num
x = "hello"         -- Error: Type mismatch: 'x' is num, cannot assign str
```

This means every variable is effectively typed from its first use, even without an explicit type annotation.

## Explicit Type Annotations

You can also declare a type explicitly. This works the same as inference but makes the intent clear.

```
num x = 42
str name = "Eric"
bool active = true
list items = [1, 2, 3]
dict data = {"key": "value"}
```

Reassignment is checked:

```
num x = 42
x = 100        -- ok, still num
x = "hello"    -- Error: Type mismatch: 'x' is num, cannot assign str
```

## When to Use Explicit Types

Use explicit type annotations when you want to document intent. The compiler enforces types either way.

```
num score = 0
score += 10        -- ok
score = "high"     -- caught by compiler
```

Since types are inferred automatically, explicit annotations are optional but recommended for clarity at the top of a function or module.

## Scope

Variables defined at the top level are visible everywhere. Variables defined inside a function are local to that function.

```
x = "global"

fn () void show:
    y = "local"
    >> x       -- can see global
    >> y       -- can see local

show()
>> x           -- global
-- >> y        -- Error: y not defined here
```

Variables defined inside `if`, `while`, `for`, and `match` blocks are visible in the enclosing scope -- these blocks do not create new scopes.

```
if true:
    result = 42

>> result      -- 42 (visible here)
```

## Closures

Functions capture variables from their enclosing scope. Changes to captured variables are shared.

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
>> counter()    -- 3
```
