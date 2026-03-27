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

type BuiltinFn func(args []*parser.Expr, env *Environment) object.Object

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
		"sort":   builtinSort,
		"map":    builtinMap,
		"filter": builtinFilter,
		"reduce": builtinReduce,
	}
}

func builtinPrint(args []*parser.Expr, env *Environment) object.Object {
	vals := make([]string, len(args))
	for i, arg := range args {
		vals[i] = evalExpr(arg, env).Inspect()
	}
	fmt.Println(strings.Join(vals, " "))
	return object.Null
}

func builtinType(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("type() requires 1 argument")
	}
	return &object.StringObj{Value: object.TypeName(evalExpr(args[0], env))}
}

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

func builtinRange(args []*parser.Expr, env *Environment) object.Object {
	if len(args) < 1 || len(args) > 3 {
		runtimeError("range() requires 1, 2, or 3 arguments")
	}

	var start, end, step float64
	switch len(args) {
	case 1:
		start = 0
		step = 1
		n, ok := evalExpr(args[0], env).(*object.NumberObj)
		if !ok {
			runtimeError("range() requires num arguments")
		}
		end = n.Value
	case 2:
		sn, ok1 := evalExpr(args[0], env).(*object.NumberObj)
		en, ok2 := evalExpr(args[1], env).(*object.NumberObj)
		if !ok1 || !ok2 {
			runtimeError("range() requires num arguments")
		}
		start = sn.Value
		end = en.Value
		step = 1
	case 3:
		sn, ok1 := evalExpr(args[0], env).(*object.NumberObj)
		en, ok2 := evalExpr(args[1], env).(*object.NumberObj)
		st, ok3 := evalExpr(args[2], env).(*object.NumberObj)
		if !ok1 || !ok2 || !ok3 {
			runtimeError("range() requires num arguments")
		}
		start = sn.Value
		end = en.Value
		step = st.Value
		if step == 0 {
			runtimeError("range() step cannot be 0")
		}
	}

	var elements []object.Object
	if step > 0 {
		for i := start; i < end; i += step {
			elements = append(elements, &object.NumberObj{Value: i})
		}
	} else {
		for i := start; i > end; i += step {
			elements = append(elements, &object.NumberObj{Value: i})
		}
	}
	return &object.ListObj{Elements: elements}
}

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

func builtinInput(args []*parser.Expr, env *Environment) object.Object {
	if len(args) > 1 {
		runtimeError("input() requires 0 or 1 arguments")
	}

	if len(args) == 1 {
		prompt := evalExpr(args[0], env)
		fmt.Print(prompt.Inspect())
	}

	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimRight(input, "\r\n")
	if num, err := strconv.ParseFloat(input, 64); err == nil {
		return &object.NumberObj{Value: num}
	}

	return &object.StringObj{Value: input}
}

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

func builtinStr(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("str() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	return &object.StringObj{Value: val.Inspect()}
}

func builtinBool(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("bool() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	return &object.BoolObj{Value: object.IsTruthy(val)}
}

func builtinErr(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("err() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	_, isErr := val.(*object.ErrorObj)
	return &object.BoolObj{Value: isErr}
}

func builtinError(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("error() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	return &object.ErrorObj{Message: val.Inspect()}
}

func builtinSort(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("sort() requires 1 argument")
	}
	val := evalExpr(args[0], env)
	l, ok := val.(*object.ListObj)
	if !ok {
		runtimeError("sort() requires a list")
	}
	sortList(l.Elements)
	return l
}

func sortList(items []object.Object) {
	n := len(items)
	for i := 1; i < n; i++ {
		key := items[i]
		j := i - 1
		for j >= 0 && compareObjects(items[j], key) > 0 {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = key
	}
}

func compareObjects(a, b object.Object) int {
	an, aok := a.(*object.NumberObj)
	bn, bok := b.(*object.NumberObj)
	if aok && bok {
		if an.Value < bn.Value {
			return -1
		}
		if an.Value > bn.Value {
			return 1
		}
		return 0
	}
	as, aok := a.(*object.StringObj)
	bs, bok := b.(*object.StringObj)
	if aok && bok {
		if as.Value < bs.Value {
			return -1
		}
		if as.Value > bs.Value {
			return 1
		}
		return 0
	}
	ta := object.TypeName(a)
	tb := object.TypeName(b)
	if ta < tb {
		return -1
	}
	if ta > tb {
		return 1
	}
	return 0
}

func builtinMap(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("map() requires 2 arguments (list, fn)")
	}
	list := evalExpr(args[0], env)
	fn := evalExpr(args[1], env)

	l, ok := list.(*object.ListObj)
	if !ok {
		runtimeError("map() first argument must be a list")
	}
	fnObj, ok := fn.(*object.FuncObj)
	if !ok {
		runtimeError("map() second argument must be a function")
	}

	results := make([]object.Object, len(l.Elements))
	for i, elem := range l.Elements {
		closureEnv := NewEnclosed(fnObj.Env.(*Environment))
		if len(fnObj.Params) > 0 {
			closureEnv.Set(fnObj.Params[0].Name, elem)
		}
		result := evalStatements(fnObj.Body.Statements, closureEnv)
		if ret, ok := result.(*object.ReturnObj); ok {
			results[i] = ret.Value
		} else {
			results[i] = result
		}
	}
	return &object.ListObj{Elements: results}
}

func builtinFilter(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("filter() requires 2 arguments (list, fn)")
	}
	list := evalExpr(args[0], env)
	fn := evalExpr(args[1], env)

	l, ok := list.(*object.ListObj)
	if !ok {
		runtimeError("filter() first argument must be a list")
	}
	fnObj, ok := fn.(*object.FuncObj)
	if !ok {
		runtimeError("filter() second argument must be a function")
	}

	var results []object.Object
	for _, elem := range l.Elements {
		closureEnv := NewEnclosed(fnObj.Env.(*Environment))
		if len(fnObj.Params) > 0 {
			closureEnv.Set(fnObj.Params[0].Name, elem)
		}
		result := evalStatements(fnObj.Body.Statements, closureEnv)
		if ret, ok := result.(*object.ReturnObj); ok {
			result = ret.Value
		}
		if object.IsTruthy(result) {
			results = append(results, elem)
		}
	}
	if results == nil {
		results = []object.Object{}
	}
	return &object.ListObj{Elements: results}
}

func builtinReduce(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 3 {
		runtimeError("reduce() requires 3 arguments (list, fn, initial)")
	}
	list := evalExpr(args[0], env)
	fn := evalExpr(args[1], env)
	acc := evalExpr(args[2], env)

	l, ok := list.(*object.ListObj)
	if !ok {
		runtimeError("reduce() first argument must be a list")
	}
	fnObj, ok := fn.(*object.FuncObj)
	if !ok {
		runtimeError("reduce() second argument must be a function")
	}

	for _, elem := range l.Elements {
		closureEnv := NewEnclosed(fnObj.Env.(*Environment))
		if len(fnObj.Params) > 0 {
			closureEnv.Set(fnObj.Params[0].Name, acc)
		}
		if len(fnObj.Params) > 1 {
			closureEnv.Set(fnObj.Params[1].Name, elem)
		}
		result := evalStatements(fnObj.Body.Statements, closureEnv)
		if ret, ok := result.(*object.ReturnObj); ok {
			acc = ret.Value
		} else {
			acc = result
		}
	}
	return acc
}
