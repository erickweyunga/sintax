# math

Math functions for Sintax.

## Usage

```
use "math"
>> math/sqrt(16)    -- 4
>> math/abs(-5)     -- 5
>> math/pi()        -- 3.14159...
```

Or import everything:

```
use "math/*"
>> sqrt(16)
>> abs(-5)
```

## Functions

### Basic
| Function | Description |
|----------|-------------|
| `sqrt(n)` | Square root |
| `abs(n)` | Absolute value |
| `floor(n)` | Round down |
| `ceil(n)` | Round up |
| `round(n)` | Round to nearest |
| `min(a, b)` | Smaller of two |
| `max(a, b)` | Larger of two |

### Constants
| Function | Description |
|----------|-------------|
| `pi()` | 3.14159... |
| `e()` | 2.71828... |

### Trigonometry
| Function | Description |
|----------|-------------|
| `sin(n)` | Sine (radians) |
| `cos(n)` | Cosine (radians) |
| `tan(n)` | Tangent (radians) |
| `asin(n)` | Arc sine |
| `acos(n)` | Arc cosine |
| `atan(n)` | Arc tangent |

### Logarithm / Exponent
| Function | Description |
|----------|-------------|
| `log(n)` | Natural log (ln) |
| `log2(n)` | Log base 2 |
| `log10(n)` | Log base 10 |
| `exp(n)` | e^n |

### Power / Root
| Function | Description |
|----------|-------------|
| `pow(a, b)` | a^b |
| `cbrt(n)` | Cube root |

### Conversion
| Function | Description |
|----------|-------------|
| `radiani(degrees)` | Degrees to radians |
| `digrii(radians)` | Radians to degrees |

### Random
| Function | Description |
|----------|-------------|
| `nasibu()` | Random float 0-1 |
| `nasibu_kati(a, b)` | Random float between a and b |

### Statistics
| Function | Description |
|----------|-------------|
| `jumla(list)` | Sum |
| `wastani(list)` | Mean/average |
| `ubaguzi(list)` | Variance |
| `kupotoka(list)` | Standard deviation |
| `kati(list)` | Median |

### Combinatorics
| Function | Description |
|----------|-------------|
| `factorial(n)` | n! |
| `mchanganyiko(n, r)` | nCr (combinations) |
| `mpangilio(n, r)` | nPr (permutations) |

### Other
| Function | Description |
|----------|-------------|
| `ishara(n)` | Sign: -1, 0, or 1 |
| `clamp(n, min, max)` | Clamp value to range |
| `asilimia(n, total)` | Percentage |
