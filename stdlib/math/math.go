package math

import (
	"fmt"
	gomath "math"
	"math/rand"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/stdlib/types"
)

// Load returns the math module.
func Load() *types.Module {
	return &types.Module{
		Name: "math",
		Desc: "Math functions",
		Funcs: map[string]types.StdFn{
			// Basic
			"sqrt":  Sqrt,
			"abs":   Abs,
			"floor": Floor,
			"ceil":  Ceil,
			"round": Round,
			"min":   Min,
			"max":   Max,
			// Constants
			"pi": Pi,
			"e":  E,
			// Trigonometry
			"sin":  Sin,
			"cos":  Cos,
			"tan":  Tan,
			"asin": Asin,
			"acos": Acos,
			"atan": Atan,
			// Logarithm / Exponent
			"log":   Log,
			"log2":  Log2,
			"log10": Log10,
			"exp":   Exp,
			// Power / Root
			"pow":  Pow,
			"cbrt": Cbrt,
			// Conversion
			"radians": Radians,
			"degrees": Degrees,
			// Random
			"random":       Random,
			"random_range": RandomRange,
			// Other
			"sign":  Sign,
			"clamp": Clamp,
			// Statistics
			"sum":      Sum,
			"mean":     Mean,
			"variance": Variance,
			"stddev":   Stddev,
			"median":   Median,
			// Combinatorics
			"factorial":    Factorial,
			"combinations": Combinations,
			"permutations": Permutations,
			// Percentages
			"percent": Percent,
		},
		Info: []types.FuncInfo{
			{Name: "sqrt(n)", Desc: "Square root"},
			{Name: "abs(n)", Desc: "Absolute value"},
			{Name: "floor(n)", Desc: "Floor"},
			{Name: "ceil(n)", Desc: "Ceiling"},
			{Name: "round(n)", Desc: "Round"},
			{Name: "min(a, b)", Desc: "Minimum"},
			{Name: "max(a, b)", Desc: "Maximum"},
			{Name: "pi()", Desc: "Pi (3.14...)"},
			{Name: "e()", Desc: "Euler's number (2.71...)"},
			{Name: "sin(n)", Desc: "Sine"},
			{Name: "cos(n)", Desc: "Cosine"},
			{Name: "tan(n)", Desc: "Tangent"},
			{Name: "asin(n)", Desc: "Arc sine"},
			{Name: "acos(n)", Desc: "Arc cosine"},
			{Name: "atan(n)", Desc: "Arc tangent"},
			{Name: "log(n)", Desc: "Natural logarithm (ln)"},
			{Name: "log2(n)", Desc: "Base-2 logarithm"},
			{Name: "log10(n)", Desc: "Base-10 logarithm"},
			{Name: "exp(n)", Desc: "e^n"},
			{Name: "pow(a, b)", Desc: "a^b"},
			{Name: "cbrt(n)", Desc: "Cube root"},
			{Name: "radians(degrees)", Desc: "Degrees to radians"},
			{Name: "degrees(radians)", Desc: "Radians to degrees"},
			{Name: "random()", Desc: "Random number 0-1"},
			{Name: "random_range(a, b)", Desc: "Random number between a and b"},
			{Name: "sign(n)", Desc: "Sign: -1, 0, or 1"},
			{Name: "clamp(n, min, max)", Desc: "Clamp number between min and max"},
			{Name: "sum(list)", Desc: "Sum of list"},
			{Name: "mean(list)", Desc: "Mean/average of list"},
			{Name: "variance(list)", Desc: "Variance of list"},
			{Name: "stddev(list)", Desc: "Standard deviation of list"},
			{Name: "median(list)", Desc: "Median of list"},
			{Name: "factorial(n)", Desc: "n! (factorial)"},
			{Name: "combinations(n, r)", Desc: "nCr (combinations)"},
			{Name: "permutations(n, r)", Desc: "nPr (permutations)"},
			{Name: "percent(n, total)", Desc: "Percentage of n in total"},
		},
	}
}

func requireNum(args []object.Object, name string, count int) ([]float64, error) {
	if len(args) != count {
		return nil, fmt.Errorf("Error: %s() requires %d argument(s)", name, count)
	}
	nums := make([]float64, count)
	for i, a := range args {
		n, ok := a.(*object.NumberObj)
		if !ok {
			return nil, fmt.Errorf("Error: %s() requires num arguments", name)
		}
		nums[i] = n.Value
	}
	return nums, nil
}

func num(v float64) object.Object { return &object.NumberObj{Value: v} }

func Sqrt(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "sqrt", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Sqrt(n[0])), nil
}

func Abs(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "abs", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Abs(n[0])), nil
}

func Floor(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "floor", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Floor(n[0])), nil
}

func Ceil(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "ceil", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Ceil(n[0])), nil
}

func Round(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "round", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Round(n[0])), nil
}

func Min(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "min", 2)
	if err != nil {
		return nil, err
	}
	return num(gomath.Min(n[0], n[1])), nil
}

func Max(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "max", 2)
	if err != nil {
		return nil, err
	}
	return num(gomath.Max(n[0], n[1])), nil
}

func Pi(args []object.Object) (object.Object, error) {
	return num(gomath.Pi), nil
}

func Sin(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "sin", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Sin(n[0])), nil
}

func Cos(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "cos", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Cos(n[0])), nil
}

func Tan(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "tan", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Tan(n[0])), nil
}

func Log(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "log", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Log(n[0])), nil
}

func E(args []object.Object) (object.Object, error) {
	return num(gomath.E), nil
}

func Asin(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "asin", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Asin(n[0])), nil
}

func Acos(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "acos", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Acos(n[0])), nil
}

func Atan(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "atan", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Atan(n[0])), nil
}

func Log2(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "log2", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Log2(n[0])), nil
}

func Log10(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "log10", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Log10(n[0])), nil
}

func Exp(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "exp", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Exp(n[0])), nil
}

func Pow(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "pow", 2)
	if err != nil {
		return nil, err
	}
	return num(gomath.Pow(n[0], n[1])), nil
}

func Cbrt(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "cbrt", 1)
	if err != nil {
		return nil, err
	}
	return num(gomath.Cbrt(n[0])), nil
}

func Radians(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "radians", 1)
	if err != nil {
		return nil, err
	}
	return num(n[0] * gomath.Pi / 180), nil
}

func Degrees(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "degrees", 1)
	if err != nil {
		return nil, err
	}
	return num(n[0] * 180 / gomath.Pi), nil
}

func Random(args []object.Object) (object.Object, error) {
	return num(rand.Float64()), nil
}

func RandomRange(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "random_range", 2)
	if err != nil {
		return nil, err
	}
	return num(n[0] + rand.Float64()*(n[1]-n[0])), nil
}

func Sign(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "sign", 1)
	if err != nil {
		return nil, err
	}
	if n[0] > 0 {
		return num(1), nil
	} else if n[0] < 0 {
		return num(-1), nil
	}
	return num(0), nil
}

func Clamp(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "clamp", 3)
	if err != nil {
		return nil, err
	}
	return num(gomath.Max(n[1], gomath.Min(n[0], n[2]))), nil
}

// --- Statistics ---

func requireList(args []object.Object, name string) ([]float64, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: %s() requires 1 argument (list)", name)
	}
	list, ok := args[0].(*object.ListObj)
	if !ok {
		return nil, fmt.Errorf("Error: %s() requires a list", name)
	}
	if len(list.Elements) == 0 {
		return nil, fmt.Errorf("Error: %s() empty list", name)
	}
	nums := make([]float64, len(list.Elements))
	for i, el := range list.Elements {
		n, ok := el.(*object.NumberObj)
		if !ok {
			return nil, fmt.Errorf("Error: %s() list must contain only num values", name)
		}
		nums[i] = n.Value
	}
	return nums, nil
}

func Sum(args []object.Object) (object.Object, error) {
	nums, err := requireList(args, "sum")
	if err != nil {
		return nil, err
	}
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	return num(sum), nil
}

func Mean(args []object.Object) (object.Object, error) {
	nums, err := requireList(args, "mean")
	if err != nil {
		return nil, err
	}
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	return num(sum / float64(len(nums))), nil
}

func Variance(args []object.Object) (object.Object, error) {
	nums, err := requireList(args, "variance")
	if err != nil {
		return nil, err
	}
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	mean := sum / float64(len(nums))
	variance := 0.0
	for _, n := range nums {
		d := n - mean
		variance += d * d
	}
	return num(variance / float64(len(nums))), nil
}

func Stddev(args []object.Object) (object.Object, error) {
	v, err := Variance(args)
	if err != nil {
		return nil, err
	}
	return num(gomath.Sqrt(v.(*object.NumberObj).Value)), nil
}

func Median(args []object.Object) (object.Object, error) {
	nums, err := requireList(args, "median")
	if err != nil {
		return nil, err
	}
	// Sort
	sorted := make([]float64, len(nums))
	copy(sorted, nums)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	n := len(sorted)
	if n%2 == 0 {
		return num((sorted[n/2-1] + sorted[n/2]) / 2), nil
	}
	return num(sorted[n/2]), nil
}

// --- Combinatorics ---

func factorialVal(n int) float64 {
	if n <= 1 {
		return 1
	}
	result := 1.0
	for i := 2; i <= n; i++ {
		result *= float64(i)
	}
	return result
}

func Factorial(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "factorial", 1)
	if err != nil {
		return nil, err
	}
	if n[0] < 0 || n[0] != gomath.Floor(n[0]) {
		return nil, fmt.Errorf("Error: factorial() requires a non-negative integer")
	}
	return num(factorialVal(int(n[0]))), nil
}

func Combinations(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "combinations", 2)
	if err != nil {
		return nil, err
	}
	nv, rv := int(n[0]), int(n[1])
	if rv > nv || rv < 0 {
		return nil, fmt.Errorf("Error: combinations() r must be 0 <= r <= n")
	}
	// nCr = n! / (r! * (n-r)!)
	return num(factorialVal(nv) / (factorialVal(rv) * factorialVal(nv-rv))), nil
}

func Permutations(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "permutations", 2)
	if err != nil {
		return nil, err
	}
	nv, rv := int(n[0]), int(n[1])
	if rv > nv || rv < 0 {
		return nil, fmt.Errorf("Error: permutations() r must be 0 <= r <= n")
	}
	// nPr = n! / (n-r)!
	return num(factorialVal(nv) / factorialVal(nv-rv)), nil
}

// --- Percent ---

func Percent(args []object.Object) (object.Object, error) {
	n, err := requireNum(args, "percent", 2)
	if err != nil {
		return nil, err
	}
	if n[1] == 0 {
		return nil, fmt.Errorf("Error: percent() cannot divide by zero")
	}
	return num((n[0] / n[1]) * 100), nil
}
