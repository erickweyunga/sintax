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

### trim(str) -> str

Remove leading and trailing whitespace.

```
>> string/trim("  hello  ")    -- hello
>> string/trim("\t hi \n")     -- hi
```

### slice(str, num, num) -> str

Extract a substring by start and end index.

```
>> string/slice("hello", 1, 3)    -- el
>> string/slice("hello", 0, 2)    -- he
```

### index_of(str, str) -> num

Find the index of a substring. Returns -1 if not found.

```
>> string/index_of("hello", "ll")     -- 2
>> string/index_of("hello", "xyz")    -- -1
```

### reverse(str) -> str

Reverse a string.

```
>> string/reverse("hello")    -- olleh
```

### repeat(str, num) -> str

Repeat a string a given number of times.

```
>> string/repeat("ha", 3)     -- hahaha
>> string/repeat("-", 10)     -- ----------
```

### char_code(str) -> num

Get the character code of the first character.

```
>> string/char_code("A")      -- 65
>> string/char_code("a")      -- 97
```

### from_char_code(num) -> str

Convert a character code to a string.

```
>> string/from_char_code(65)   -- A
>> string/from_char_code(97)   -- a
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
