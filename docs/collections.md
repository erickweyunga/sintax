# Collections

## Lists

Lists are ordered, mutable collections of any value type.

### Creating Lists

```
numbers = [1, 2, 3, 4, 5]
names = ["Alice", "Bob", "Charlie"]
mixed = [1, "hello", true, [4, 5]]
empty = []
```

### Access

```
items = ["a", "b", "c", "d"]
>> items[0]      -- a (first)
>> items[3]      -- d (last)
>> len(items)    -- 4
```

### Modify

```
items = ["a", "b", "c"]
items[1] = "B"           -- replace by index
push(items, "d")         -- append to end
pop(items, 0)            -- remove by index, returns removed value
```

### Iteration

```
for item in [10, 20, 30]:
    >> item

-- With index using range
items = ["a", "b", "c"]
for i in range(len(items)):
    print(i, "=", items[i])
```

### Methods

```
items = [3, 1, 2]

>> items.len()           -- 3
>> items.contains(2)     -- true
>> items.contains(9)     -- false
>> items.reverse()       -- [2, 1, 3]
>> ["a", "b"].join("-")  -- a-b
```

### Functional Methods

Transform lists without writing loops.

**map** — apply a function to each element:

```
>> [1, 2, 3].map(fn(x) -> x * 2)        -- [2, 4, 6]
>> ["hi", "hey"].map(fn(s) -> s.upper()) -- ["HI", "HEY"]
```

**filter** — keep elements that pass a test:

```
>> [1, 2, 3, 4, 5].filter(fn(x) -> x > 3)    -- [4, 5]
>> ["cat", "dog", "cow"].filter(fn(s) -> s.starts_with("c"))  -- ["cat", "cow"]
```

**reduce** — combine all elements into one value:

```
>> [1, 2, 3, 4].reduce(fn(acc, x) -> acc + x, 0)     -- 10
>> [1, 2, 3, 4].reduce(fn(acc, x) -> acc * x, 1)      -- 24
```

**each** — run a function for side effects:

```
[1, 2, 3].each(fn(x) -> print("item:", x))
```

### Membership

```
>> 3 in [1, 2, 3]       -- true
>> "x" in [1, 2, 3]     -- false
```

## Dicts

Dicts are ordered key-value maps. Keys are always strings.

### Creating Dicts

```
person = {"name": "Eric", "age": 25, "active": true}
empty = {}
```

### Access

```
person = {"name": "Eric", "age": 25}
>> person["name"]        -- Eric
>> person["age"]         -- 25
>> person["missing"]     -- null
```

### Modify

```
person = {"name": "Eric"}
person["name"] = "John"          -- update
person["email"] = "j@test.com"   -- add new key
```

### Introspection

```
person = {"name": "Eric", "age": 25}
>> keys(person)          -- ["name", "age"]
>> values(person)        -- ["Eric", 25]
>> has(person, "name")   -- true
>> has(person, "email")  -- false
>> len(person)           -- 2
```

### Iteration

Iterating over a dict yields its keys in insertion order:

```
config = {"host": "localhost", "port": 8080, "debug": true}
for key in config:
    print(key, "=", config[key])
```

Output:

```
host = localhost
port = 8080
debug = true
```

### Methods

```
d = {"a": 1, "b": 2}
>> d.len()           -- 2
>> d.keys()          -- ["a", "b"]
>> d.values()        -- [1, 2]
>> d.has("a")        -- true
```

### Membership

```
>> "name" in {"name": "Eric"}    -- true
>> "email" in {"name": "Eric"}   -- false
```

### Nested Collections

Lists and dicts can be nested.

```
users = [
    {"name": "Alice", "scores": [95, 87]},
    {"name": "Bob", "scores": [72, 91]}
]

for user in users:
    name = user["name"]
    scores = user["scores"]
    >> "{name}: {scores}"
```

### JSON Round-trip

Use the json module with dicts:

```
use "std/json"

-- Parse JSON into a dict
data = json/parse('{"users": [{"name": "Eric"}, {"name": "John"}]}')

-- Access nested data
for user in data["users"]:
    >> user["name"]

-- Convert back to JSON
>> json/stringify(data)
>> json/pretty(data)
```
