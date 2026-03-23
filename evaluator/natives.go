package evaluator

import (
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
)

// Native bridge functions — Go implementations of __native_* for the interpreter.
// These match the C implementations in runtime/native.c.

func init() {
	natives := map[string]BuiltinFn{
		// Math
		"__native_sqrt":   nativeMath1(math.Sqrt),
		"__native_sin":    nativeMath1(math.Sin),
		"__native_cos":    nativeMath1(math.Cos),
		"__native_tan":    nativeMath1(math.Tan),
		"__native_asin":   nativeMath1(math.Asin),
		"__native_acos":   nativeMath1(math.Acos),
		"__native_atan":   nativeMath1(math.Atan),
		"__native_log":    nativeMath1(math.Log),
		"__native_log2":   nativeMath1(math.Log2),
		"__native_log10":  nativeMath1(math.Log10),
		"__native_exp":    nativeMath1(math.Exp),
		"__native_floor":  nativeMath1(math.Floor),
		"__native_ceil":   nativeMath1(math.Ceil),
		"__native_round":  nativeMath1(math.Round),
		"__native_cbrt":   nativeMath1(math.Cbrt),
		"__native_pow":    nativeMath2(math.Pow),
		"__native_random": nativeRandom,
		// String
		"__native_upper":   nativeUpper,
		"__native_lower":   nativeLower,
		"__native_split":   nativeSplit,
		"__native_replace": nativeReplace,
		// OS
		"__native_read_file":   nativeReadFile,
		"__native_write_file":  nativeWriteFile,
		"__native_file_exists": nativeFileExists,
		"__native_delete_file": nativeDeleteFile,
		"__native_cwd":         nativeCwd,
		"__native_getenv":      nativeGetenv,
		"__native_exec":        nativeExec,
		"__native_time":        nativeTime,
	}

	for name, fn := range natives {
		builtins[name] = fn
	}
}

// Helper: wrap a math.Func(float64) float64 as a builtin
func nativeMath1(fn func(float64) float64) BuiltinFn {
	return func(args []*parser.Expr, env *Environment) object.Object {
		val := evalExpr(args[0], env).(*object.NumberObj)
		return &object.NumberObj{Value: fn(val.Value)}
	}
}

func nativeMath2(fn func(float64, float64) float64) BuiltinFn {
	return func(args []*parser.Expr, env *Environment) object.Object {
		a := evalExpr(args[0], env).(*object.NumberObj)
		b := evalExpr(args[1], env).(*object.NumberObj)
		return &object.NumberObj{Value: fn(a.Value, b.Value)}
	}
}

func nativeRandom(args []*parser.Expr, env *Environment) object.Object {
	return &object.NumberObj{Value: rand.Float64()}
}

func nativeUpper(args []*parser.Expr, env *Environment) object.Object {
	s := evalExpr(args[0], env).(*object.StringObj)
	return &object.StringObj{Value: strings.ToUpper(s.Value)}
}

func nativeLower(args []*parser.Expr, env *Environment) object.Object {
	s := evalExpr(args[0], env).(*object.StringObj)
	return &object.StringObj{Value: strings.ToLower(s.Value)}
}

func nativeSplit(args []*parser.Expr, env *Environment) object.Object {
	s := evalExpr(args[0], env).(*object.StringObj)
	sep := evalExpr(args[1], env).(*object.StringObj)
	parts := strings.Split(s.Value, sep.Value)
	elements := make([]object.Object, len(parts))
	for i, p := range parts {
		elements[i] = &object.StringObj{Value: p}
	}
	return &object.ListObj{Elements: elements}
}

func nativeReplace(args []*parser.Expr, env *Environment) object.Object {
	s := evalExpr(args[0], env).(*object.StringObj)
	old := evalExpr(args[1], env).(*object.StringObj)
	new_ := evalExpr(args[2], env).(*object.StringObj)
	return &object.StringObj{Value: strings.ReplaceAll(s.Value, old.Value, new_.Value)}
}

func nativeReadFile(args []*parser.Expr, env *Environment) object.Object {
	path := evalExpr(args[0], env).(*object.StringObj)
	data, err := os.ReadFile(path.Value)
	if err != nil {
		return &object.ErrorObj{Message: "Cannot read file: " + path.Value}
	}
	return &object.StringObj{Value: string(data)}
}

func nativeWriteFile(args []*parser.Expr, env *Environment) object.Object {
	path := evalExpr(args[0], env).(*object.StringObj)
	data := evalExpr(args[1], env)
	if err := os.WriteFile(path.Value, []byte(data.Inspect()), 0644); err != nil {
		return &object.ErrorObj{Message: "Cannot write file: " + path.Value}
	}
	return object.Null
}

func nativeFileExists(args []*parser.Expr, env *Environment) object.Object {
	path := evalExpr(args[0], env).(*object.StringObj)
	_, err := os.Stat(path.Value)
	return &object.BoolObj{Value: err == nil}
}

func nativeDeleteFile(args []*parser.Expr, env *Environment) object.Object {
	path := evalExpr(args[0], env).(*object.StringObj)
	if err := os.Remove(path.Value); err != nil {
		return &object.ErrorObj{Message: "Cannot delete file: " + path.Value}
	}
	return object.Null
}

func nativeCwd(args []*parser.Expr, env *Environment) object.Object {
	dir, _ := os.Getwd()
	return &object.StringObj{Value: dir}
}

func nativeGetenv(args []*parser.Expr, env *Environment) object.Object {
	name := evalExpr(args[0], env).(*object.StringObj)
	val := os.Getenv(name.Value)
	if val == "" {
		return object.Null
	}
	return &object.StringObj{Value: val}
}

func nativeExec(args []*parser.Expr, env *Environment) object.Object {
	cmdStr := evalExpr(args[0], env).(*object.StringObj)
	parts := strings.Fields(cmdStr.Value)
	if len(parts) == 0 {
		return &object.StringObj{Value: ""}
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	output, _ := cmd.CombinedOutput()
	return &object.StringObj{Value: strings.TrimRight(string(output), "\n")}
}

func nativeTime(args []*parser.Expr, env *Environment) object.Object {
	return &object.NumberObj{Value: float64(time.Now().Unix())}
}
