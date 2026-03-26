# Control Flow

## if / else

```
x = 10
if x > 5:
    >> "big"
else:
    >> "small"
```

Conditions can use `and`, `or`, `not`:

```
if x > 0 and x < 100:
    >> "in range"

if not active:
    >> "inactive"
```

There is no `else if`. Use `match/case` for multi-branch logic.

## match / case

`match` is the universal branching tool in Sintax. Use it instead of `else if` chains.

```
day = "Monday"
match day:
    case "Monday":
        >> "Start of week"
    case "Friday":
        >> "Almost weekend"
    case "Saturday":
        >> "Weekend!"
    case "Sunday":
        >> "Weekend!"
    _:
        >> "Midweek"
```

`_` is the default case — it runs when nothing else matches.

Match works with any type:

```
x = 42
match x:
    case 0:
        >> "zero"
    case 1:
        >> "one"
    _:
        >> "other"
```

### Match for Error Handling

```
result = some_operation()
match err(result):
    case true:
        >> "Error occurred"
    case false:
        >> "Success"
```

## while

```
i = 0
while i < 5:
    >> i
    i += 1
```

## for .. in

Iterate over lists, ranges, strings, and dicts.

```
-- Over a list
for fruit in ["Mango", "Banana", "Orange"]:
    >> fruit

-- Over a range
for i in range(5):
    >> i      -- 0, 1, 2, 3, 4

for i in range(2, 7):
    >> i      -- 2, 3, 4, 5, 6

-- Over a string
for ch in "hello":
    >> ch     -- h, e, l, l, o

-- Over a dict (iterates keys)
person = {"name": "Eric", "age": 25}
for key in person:
    print(key, "=", person[key])
```

## Break and Continue

Sintax uses `0` for break and `1` for continue.

```
-- Break: stop at 3
for n in range(10):
    if n == 3:
        0
    >> n
-- prints 0, 1, 2

-- Continue: skip even numbers
for n in range(6):
    if n % 2 == 0:
        1
    >> n
-- prints 1, 3, 5
```

`0` and `1` work in `while` loops too:

```
i = 0
while true:
    if i == 5:
        0       -- break
    >> i
    i += 1
```

## Membership

Use `in` to check if a value exists in a collection:

```
>> 3 in [1, 2, 3]           -- true
>> "x" in [1, 2, 3]         -- false
>> "name" in {"name": "E"}  -- true
>> "ell" in "hello"          -- true
```
