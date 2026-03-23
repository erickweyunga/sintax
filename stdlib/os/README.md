# os

OS, file, and process functions for Sintax.

## Usage

```
use "os"
>> os/cwd()
>> os/mfumo_jina()

os/andika("/tmp/test.txt", "hello")
data = os/soma("/tmp/test.txt")
>> data
```

Or import everything:

```
use "os/*"
andika("/tmp/test.txt", "hello")
>> soma("/tmp/test.txt")
```

## Functions

### File Operations
| Function | Description |
|----------|-------------|
| `soma(path)` | Read file contents |
| `andika(path, data)` | Write file |
| `ipo(path)` | Check if file/dir exists |
| `ongeza(path, data)` | Append to file |
| `futa(path)` | Delete file |
| `orodha(path)` | List directory contents |
| `tengeneza_saraka(path)` | Create directory (recursive) |

### Environment
| Function | Description |
|----------|-------------|
| `mazingira(name)` | Get environment variable |
| `weka_mazingira(name, value)` | Set environment variable |

### Process
| Function | Description |
|----------|-------------|
| `tekeleza(command)` | Execute shell command, return output |
| `toka(code)` | Exit with status code |

### System Info
| Function | Description |
|----------|-------------|
| `mfumo_jina()` | OS name (darwin/linux/windows) |
| `saa()` | Current time |
| `cwd()` | Current working directory |

## Examples

```
use "os/*"

-- Write and read
andika("/tmp/greet.txt", "Hello Sintax!")
>> soma("/tmp/greet.txt")

-- Check existence
if ipo("/tmp/greet.txt"):
    >> "File exists"

-- List directory
for f in orodha("."):
    >> f

-- Run shell command
>> tekeleza("date")

-- Environment
>> mazingira("HOME")
```
