# Comments

## Single-line Comments

Use `--` for single-line comments:

```
-- This is a comment
x = 42    -- inline comment
```

## Multi-line Comments

Use `--{` to start and `}--` to end:

```
--{
This is a
multi-line comment.
}--
```

## Test Comments

Special comments used by the test runner:

```
fn (num a, num b) num add:
    a + b

-- test: add(2, 3) == 5
-- test: add(0, 0) == 0
-- test: add(-1, 1) == 0
```

Run tests:

```bash
sintax test           # test all .sx files in current directory
sintax test file.sx   # test a specific file
```
