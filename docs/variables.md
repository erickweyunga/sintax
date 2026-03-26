# Variables

## Dynamic Variables

Assign any value without declaring a type. The variable takes whatever type you give it.

```
x = 42
name = "Eric"
items = [1, 2, 3]
x = "now a string"    -- allowed, x changes type
```

## Typed Variables

Declare a type to enforce it. Once typed, the variable can only hold values of that type.

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

## When to Use Typed Variables

Use typed variables when you want safety — the analyzer catches type mismatches before your code runs.

```
num score = 0
score += 10        -- ok
score = "high"     -- caught by analyzer
```

Use dynamic variables when the type may change or you don't need enforcement.

```
result = input("Enter value: ")    -- could be num or str
```

## Scope

Variables defined at the top level are visible everywhere. Variables defined inside a function are local to that function.

```
x = "global"

fn () show:
    y = "local"
    >> x       -- can see global
    >> y       -- can see local

show()
>> x           -- global
-- >> y        -- Error: y not defined here
```

Variables defined inside `if`, `while`, `for`, and `match` blocks are visible in the enclosing scope — these blocks do not create new scopes.

```
if true:
    result = 42

>> result      -- 42 (visible here)
```

## Closures

Functions capture variables from their enclosing scope. Changes to captured variables are shared.

```
fn () make_counter:
    count = 0
    fn () inc:
        count = count + 1
        return count
    return inc

counter = make_counter()
>> counter()    -- 1
>> counter()    -- 2
>> counter()    -- 3
```
