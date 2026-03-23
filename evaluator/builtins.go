package evaluator

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
)

// BuiltinFn is the signature for built-in functions.
type BuiltinFn func(args []*parser.Expr, env *Environment) object.Object

// builtins is initialized at runtime to avoid initialization cycles.
var builtins map[string]BuiltinFn

func init() {
	builtins = map[string]BuiltinFn{
		"print":  builtinPrint,
		"type":   builtinType,
		"len":    builtinLen,
		"push":   builtinPush,
		"pop":    builtinPop,
		"range":  builtinRange,
		"input":  builtinInput,
		"keys":   builtinKeys,
		"values": builtinValues,
		"has":    builtinHas,
		"num":    builtinNum,
		"str":    builtinStr,
		"bool":   builtinBool,
		"err":    builtinErr,
		"error":  builtinError,
	}
}

// print — print values
func builtinPrint(args []*parser.Expr, env *Environment) object.Object {
	vals := make([]string, len(args))
	for i, arg := range args {
		vals[i] = evalExpr(arg, env).Inspect()
	}
	fmt.Println(strings.Join(vals, " "))
	return object.Null
}

// type — return the type of a value as a string
func builtinType(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("type() requires 1 argument")
	}
	return &object.StringObj{Value: object.TypeName(evalExpr(args[0], env))}
}

// len — return the length of a list or string
func builtinLen(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("len() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	switch o := val.(type) {
	case *object.ListObj:
		return &object.NumberObj{Value: float64(len(o.Elements))}
	case *object.StringObj:
		return &object.NumberObj{Value: float64(len(o.Value))}
	case *object.DictObj:
		return &object.NumberObj{Value: float64(len(o.Pairs))}
	default:
		runtimeError("len() requires a list, str, or dict")
	}
	return object.Null
}

// push — append an item to a list
func builtinPush(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("push() requires 2 arguments (list, element)")
	}
	list := evalExpr(args[0], env)
	item := evalExpr(args[1], env)

	l, ok := list.(*object.ListObj)
	if !ok {
		runtimeError("push() first argument must be a list")
	}
	l.Elements = append(l.Elements, item)
	return l
}

// pop — remove an item from a list by index
func builtinPop(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("pop() requires 2 arguments (list, index)")
	}
	list := evalExpr(args[0], env)
	idx := evalExpr(args[1], env)

	l, ok := list.(*object.ListObj)
	if !ok {
		runtimeError("pop() first argument must be a list")
	}
	idxNum, ok := idx.(*object.NumberObj)
	if !ok {
		runtimeError("pop() second argument must be a num")
	}
	i := int(idxNum.Value)
	if i < 0 || i >= len(l.Elements) {
		runtimeError("Index %d out of range (length %d)", i, len(l.Elements))
	}
	removed := l.Elements[i]
	l.Elements = append(l.Elements[:i], l.Elements[i+1:]...)
	return removed
}

// range — generate a range of numbers as a list
func builtinRange(args []*parser.Expr, env *Environment) object.Object {
	if len(args) < 1 || len(args) > 2 {
		runtimeError("range() requires 1 or 2 arguments")
	}

	var start, end float64
	if len(args) == 1 {
		start = 0
		n, ok := evalExpr(args[0], env).(*object.NumberObj)
		if !ok {
			runtimeError("range() requires num arguments")
		}
		end = n.Value
	} else {
		sn, ok1 := evalExpr(args[0], env).(*object.NumberObj)
		en, ok2 := evalExpr(args[1], env).(*object.NumberObj)
		if !ok1 || !ok2 {
			runtimeError("range() requires num arguments")
		}
		start = sn.Value
		end = en.Value
	}

	var elements []object.Object
	for i := start; i < end; i++ {
		elements = append(elements, &object.NumberObj{Value: i})
	}
	return &object.ListObj{Elements: elements}
}

// keys — return keys of a dict as a list
func builtinKeys(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("keys() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	d, ok := val.(*object.DictObj)
	if !ok {
		runtimeError("keys() requires a dict")
	}
	elements := make([]object.Object, len(d.Keys))
	for i, k := range d.Keys {
		elements[i] = &object.StringObj{Value: k}
	}
	return &object.ListObj{Elements: elements}
}

// values — return values of a dict as a list
func builtinValues(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("values() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	d, ok := val.(*object.DictObj)
	if !ok {
		runtimeError("values() requires a dict")
	}
	elements := make([]object.Object, len(d.Keys))
	for i, k := range d.Keys {
		elements[i] = d.Pairs[k]
	}
	return &object.ListObj{Elements: elements}
}

// has — check if a dict has a key
func builtinHas(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("has() requires 2 arguments (dict, key)")
	}
	val := evalExpr(args[0], env)
	key := evalExpr(args[1], env)

	d, ok := val.(*object.DictObj)
	if !ok {
		runtimeError("has() first argument must be a dict")
	}
	keyStr, ok := key.(*object.StringObj)
	if !ok {
		runtimeError("has() second argument must be a str")
	}
	_, exists := d.Pairs[keyStr.Value]
	return &object.BoolObj{Value: exists}
}

var stdinReader = bufio.NewReader(os.Stdin)

// input — read user input, optionally with a prompt
func builtinInput(args []*parser.Expr, env *Environment) object.Object {
	if len(args) > 1 {
		runtimeError("input() requires 0 or 1 arguments")
	}

	// Print prompt if provided
	if len(args) == 1 {
		prompt := evalExpr(args[0], env)
		fmt.Print(prompt.Inspect())
	}

	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimRight(input, "\r\n")

	// Try to convert to number
	if num, err := strconv.ParseFloat(input, 64); err == nil {
		return &object.NumberObj{Value: num}
	}

	return &object.StringObj{Value: input}
}

// num — convert to number
func builtinNum(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("num() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	switch v := val.(type) {
	case *object.NumberObj:
		return v
	case *object.StringObj:
		num, err := strconv.ParseFloat(v.Value, 64)
		if err != nil {
			runtimeError("Cannot convert '%s' to num", v.Value)
		}
		return &object.NumberObj{Value: num}
	case *object.BoolObj:
		if v.Value {
			return &object.NumberObj{Value: 1}
		}
		return &object.NumberObj{Value: 0}
	default:
		runtimeError("Cannot convert %s to num", object.TypeName(val))
	}
	return object.Null
}

// str — convert to string
func builtinStr(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("str() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	return &object.StringObj{Value: val.Inspect()}
}

// bool — convert to boolean
func builtinBool(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("bool() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	return &object.BoolObj{Value: object.IsTruthy(val)}
}

// err — check if a value is an error
func builtinErr(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("err() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	_, isErr := val.(*object.ErrorObj)
	return &object.BoolObj{Value: isErr}
}

// error — create an error value
func builtinError(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("error() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	return &object.ErrorObj{Message: val.Inspect()}
}
