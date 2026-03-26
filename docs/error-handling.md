# Error Handling

Sintax handles errors with `error()`, `err()`, `catch`, and `match/case`.

## Creating Errors

Use `error()` to create an error value:

```
e = error("something went wrong")
>> e          -- error: something went wrong
>> type(e)    -- error
```

## Checking for Errors

Use `err()` to check if a value is an error:

```
e = error("bad")
>> err(e)       -- true
>> err(42)      -- false
>> err("hello") -- false
```

## catch Statement

`catch` is a shorthand for calling a function and handling errors inline. If the result is an error, the body executes. Otherwise, the variable holds the successful value.

```
use "std/os"

catch content = os/read("config.txt"):
    >> "Failed to read config: " + str(content)

>> content
```

`catch` reduces boilerplate compared to the `if err()` pattern:

```
use "std/json"

catch data = json/parse(raw_input):
    >> "Invalid JSON"
    return error("parse failed")

>> data["name"]
```

## Error Handling with match

The pattern: call a function, check if the result is an error, branch accordingly.

```
use "std/os"

result = os/read("config.txt")
match err(result):
    case true:
        >> "Failed to read config"
        >> result
    case false:
        >> "Config loaded:"
        >> result
```

## Functions That Return Errors

Functions can return error values instead of crashing:

```
fn (num a, num b) num divide:
    if b == 0:
        return error("division by zero")
    return a / b

result = divide(10, 0)
match err(result):
    case true:
        >> result    -- error: division by zero
    case false:
        >> result
```

Or using `catch`:

```
catch result = divide(10, 0):
    >> "Cannot divide: " + str(result)
```

## Error Truthiness

Errors are **falsy**:

```
e = error("bad")
if e:
    >> "truthy"
else:
    >> "falsy"     -- this prints
```

This means you can also use `if` for simple checks:

```
result = divide(10, 0)
if err(result):
    >> "Error: " + str(result)
else:
    >> result
```

## Stdlib Functions That Return Errors

Some stdlib functions return errors on failure:

```
use "std/os"

-- os/read() returns an error if file doesn't exist
content = os/read("missing.txt")
if err(content):
    >> "File not found"
```

Using `catch` for cleaner handling:

```
use "std/os"

catch content = os/read("missing.txt"):
    >> "File not found"
```

## Pattern: Safe Operations

```
fn (str path) str safe_read:
    use "std/os"
    if not os/exists(path):
        return error("file not found: " + path)
    return os/read(path)

catch result = safe_read("data.json"):
    >> result

>> "Got " + str(len(result)) + " bytes"
```

## When to Use What

- **`catch`** -- concise error handling for a single call, handle and continue
- **`match err()`** -- when you need distinct branches for success vs. failure
- **`if err()`** -- simple inline checks
- Return `error()` from functions instead of crashing
- Don't use errors for control flow -- errors are for exceptional conditions
