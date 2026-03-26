# String Module

String manipulation functions.

## Import

```
use "std/string"           -- namespaced: string/upper(...)
use "std/string/upper"     -- specific: upper(...)
```

## Functions

### upper(str) -> str

Convert to uppercase.

```
use "std/string"
>> string/upper("hello")    -- HELLO
```

### lower(str) -> str

Convert to lowercase.

```
>> string/lower("HELLO")    -- hello
```

### split(str, str) -> list

Split a string by a separator.

```
>> string/split("a,b,c", ",")    -- ["a", "b", "c"]
>> string/split("hello world", " ")  -- ["hello", "world"]
```

### replace(str, str, str) -> str

Replace all occurrences of a substring.

```
>> string/replace("hello world", "world", "Sintax")    -- hello Sintax
```

### length(str) -> num

Get the length of a string.

```
>> string/length("hello")    -- 5
```

### contains(str, str) -> bool

Check if a string contains a substring.

```
>> string/contains("hello world", "world")    -- true
>> string/contains("hello", "xyz")            -- false
```

### starts_with(str, str) -> bool

Check if a string starts with a prefix.

```
>> string/starts_with("hello", "hel")    -- true
>> string/starts_with("hello", "world")  -- false
```

### ends_with(str, str) -> bool

Check if a string ends with a suffix.

```
>> string/ends_with("hello", "llo")     -- true
>> string/ends_with("hello", "world")   -- false
```

### join(list, str) -> str

Join a list of values into a string with a separator.

```
>> string/join(["a", "b", "c"], "-")    -- a-b-c
>> string/join([1, 2, 3], ", ")          -- 1, 2, 3
```

## String Methods

Strings also have built-in methods that can be called directly:

```
>> "hello".upper()               -- HELLO
>> "HELLO".lower()               -- hello
>> "  hello  ".trim()            -- hello
>> "hello".len()                 -- 5
>> "hello".contains("ell")       -- true
>> "hello".starts_with("hel")    -- true
>> "hello".ends_with("llo")      -- true
>> "hello world".split(" ")      -- ["hello", "world"]
>> "hello world".replace("world", "x")  -- hello x

-- Method chaining
>> "  HELLO  ".trim().lower()    -- hello
>> "a-b-c".split("-").join(",")  -- a,b,c
```
