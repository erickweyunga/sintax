package evaluator

import (
	"encoding/json"
	"fmt"
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
		// JSON
		"__native_json_parse":     nativeJsonParse,
		"__native_json_stringify": nativeJsonStringify,
		"__native_json_pretty":    nativeJsonPretty,
	}

	for name, fn := range natives {
		builtins[name] = fn
	}
}

// --- Type check helpers ---

func expectNum(val object.Object, fn string) *object.NumberObj {
	n, ok := val.(*object.NumberObj)
	if !ok {
		runtimeError("%s() requires a num, got %s", fn, object.TypeName(val))
	}
	return n
}

func expectStr(val object.Object, fn string) *object.StringObj {
	s, ok := val.(*object.StringObj)
	if !ok {
		runtimeError("%s() requires a str, got %s", fn, object.TypeName(val))
	}
	return s
}

// --- Native implementations ---

func nativeMath1(fn func(float64) float64) BuiltinFn {
	return func(args []*parser.Expr, env *Environment) object.Object {
		val := expectNum(evalExpr(args[0], env), "math")
		return &object.NumberObj{Value: fn(val.Value)}
	}
}

func nativeMath2(fn func(float64, float64) float64) BuiltinFn {
	return func(args []*parser.Expr, env *Environment) object.Object {
		a := expectNum(evalExpr(args[0], env), "math")
		b := expectNum(evalExpr(args[1], env), "math")
		return &object.NumberObj{Value: fn(a.Value, b.Value)}
	}
}

func nativeRandom(args []*parser.Expr, env *Environment) object.Object {
	return &object.NumberObj{Value: rand.Float64()}
}

func nativeUpper(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "upper")
	return &object.StringObj{Value: strings.ToUpper(s.Value)}
}

func nativeLower(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "lower")
	return &object.StringObj{Value: strings.ToLower(s.Value)}
}

func nativeSplit(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "split")
	sep := expectStr(evalExpr(args[1], env), "split")
	parts := strings.Split(s.Value, sep.Value)
	elements := make([]object.Object, len(parts))
	for i, p := range parts {
		elements[i] = &object.StringObj{Value: p}
	}
	return &object.ListObj{Elements: elements}
}

func nativeReplace(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "replace")
	old := expectStr(evalExpr(args[1], env), "replace")
	new_ := expectStr(evalExpr(args[2], env), "replace")
	return &object.StringObj{Value: strings.ReplaceAll(s.Value, old.Value, new_.Value)}
}

func nativeReadFile(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "read")
	data, err := os.ReadFile(path.Value)
	if err != nil {
		return &object.ErrorObj{Message: "Cannot read file: " + path.Value}
	}
	return &object.StringObj{Value: string(data)}
}

func nativeWriteFile(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "write")
	data := evalExpr(args[1], env)
	if err := os.WriteFile(path.Value, []byte(data.Inspect()), 0644); err != nil {
		return &object.ErrorObj{Message: "Cannot write file: " + path.Value}
	}
	return object.Null
}

func nativeFileExists(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "exists")
	_, err := os.Stat(path.Value)
	return &object.BoolObj{Value: err == nil}
}

func nativeDeleteFile(args []*parser.Expr, env *Environment) object.Object {
	path := expectStr(evalExpr(args[0], env), "delete")
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
	name := expectStr(evalExpr(args[0], env), "getenv")
	val := os.Getenv(name.Value)
	if val == "" {
		return object.Null
	}
	return &object.StringObj{Value: val}
}

func nativeExec(args []*parser.Expr, env *Environment) object.Object {
	cmdStr := expectStr(evalExpr(args[0], env), "exec")
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

// --- JSON ---

func nativeJsonParse(args []*parser.Expr, env *Environment) object.Object {
	s := expectStr(evalExpr(args[0], env), "json/parse")
	dec := json.NewDecoder(strings.NewReader(s.Value))
	dec.UseNumber()
	result, err := jsonDecodeValue(dec)
	if err != nil {
		return &object.ErrorObj{Message: "Invalid JSON: " + err.Error()}
	}
	return result
}

func jsonDecodeValue(dec *json.Decoder) (object.Object, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			return jsonDecodeObject(dec)
		case '[':
			return jsonDecodeArray(dec)
		}
	case json.Number:
		f, _ := v.Float64()
		return &object.NumberObj{Value: f}, nil
	case string:
		return &object.StringObj{Value: v}, nil
	case bool:
		return &object.BoolObj{Value: v}, nil
	case nil:
		return object.Null, nil
	}
	return object.Null, nil
}

func jsonDecodeObject(dec *json.Decoder) (object.Object, error) {
	pairs := make(map[string]object.Object)
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key := tok.(string)
		keys = append(keys, key)
		val, err := jsonDecodeValue(dec)
		if err != nil {
			return nil, err
		}
		pairs[key] = val
	}
	dec.Token() // consume closing }
	return &object.DictObj{Pairs: pairs, Keys: keys}, nil
}

func jsonDecodeArray(dec *json.Decoder) (object.Object, error) {
	var elements []object.Object
	for dec.More() {
		val, err := jsonDecodeValue(dec)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
	}
	dec.Token() // consume closing ]
	return &object.ListObj{Elements: elements}, nil
}

func nativeJsonStringify(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	return &object.StringObj{Value: jsonMarshal(val, "", "")}
}

func nativeJsonPretty(args []*parser.Expr, env *Environment) object.Object {
	val := evalExpr(args[0], env)
	return &object.StringObj{Value: jsonMarshal(val, "", "  ")}
}

func jsonMarshal(obj object.Object, prefix, indent string) string {
	var buf strings.Builder
	jsonWriteValue(&buf, obj, prefix, indent, 0)
	return buf.String()
}

func jsonWriteValue(buf *strings.Builder, obj object.Object, prefix, indent string, depth int) {
	switch v := obj.(type) {
	case *object.NumberObj:
		if v.Value == float64(int64(v.Value)) {
			fmt.Fprintf(buf, "%d", int64(v.Value))
		} else {
			fmt.Fprintf(buf, "%g", v.Value)
		}
	case *object.StringObj:
		data, _ := json.Marshal(v.Value)
		buf.Write(data)
	case *object.BoolObj:
		if v.Value {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case *object.NullObj:
		buf.WriteString("null")
	case *object.ListObj:
		buf.WriteByte('[')
		for i, el := range v.Elements {
			if i > 0 {
				buf.WriteByte(',')
			}
			if indent != "" {
				buf.WriteByte('\n')
				buf.WriteString(prefix)
				for j := 0; j <= depth; j++ {
					buf.WriteString(indent)
				}
			}
			jsonWriteValue(buf, el, prefix, indent, depth+1)
		}
		if indent != "" && len(v.Elements) > 0 {
			buf.WriteByte('\n')
			buf.WriteString(prefix)
			for j := 0; j < depth; j++ {
				buf.WriteString(indent)
			}
		}
		buf.WriteByte(']')
	case *object.DictObj:
		buf.WriteByte('{')
		for i, k := range v.Keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if indent != "" {
				buf.WriteByte('\n')
				buf.WriteString(prefix)
				for j := 0; j <= depth; j++ {
					buf.WriteString(indent)
				}
			}
			keyData, _ := json.Marshal(k)
			buf.Write(keyData)
			buf.WriteByte(':')
			if indent != "" {
				buf.WriteByte(' ')
			}
			jsonWriteValue(buf, v.Pairs[k], prefix, indent, depth+1)
		}
		if indent != "" && len(v.Keys) > 0 {
			buf.WriteByte('\n')
			buf.WriteString(prefix)
			for j := 0; j < depth; j++ {
				buf.WriteString(indent)
			}
		}
		buf.WriteByte('}')
	default:
		buf.WriteString("null")
	}
}
