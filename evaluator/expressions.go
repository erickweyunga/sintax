package evaluator

import (
	"math"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

// evalTry evaluates an expression, catching any runtime errors.
func evalTry(expr *parser.Expr, env *Environment) (result object.Object) {
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(RuntimeError); ok {
				result = &object.ErrorObj{Message: re.Message}
			} else {
				panic(r)
			}
		}
	}()
	return evalExpr(expr, env)
}

func evalExpr(expr *parser.Expr, env *Environment) object.Object {
	result := evalLogicalAnd(expr.Left, env)
	for _, op := range expr.Ops {
		if object.IsTruthy(result) {
			return result
		}
		result = evalLogicalAnd(op.Right, env)
	}
	return result
}

func evalLogicalAnd(and *parser.LogicalAnd, env *Environment) object.Object {
	result := evalComparison(and.Left, env)
	for _, op := range and.Ops {
		if !object.IsTruthy(result) {
			return result
		}
		result = evalComparison(op.Right, env)
	}
	return result
}

func evalComparison(cmp *parser.Comparison, env *Environment) object.Object {
	left := evalAddition(cmp.Left, env)
	if cmp.Op != "" {
		right := evalAddition(cmp.Right, env)
		return evalComparisonOp(cmp.Op, left, right)
	}
	return left
}

func evalAddition(add *parser.Addition, env *Environment) object.Object {
	result := evalMultiplication(add.Left, env)
	for _, op := range add.Ops {
		right := evalMultiplication(op.Right, env)
		result = evalArithOp(op.Op, result, right)
	}
	return result
}

func evalMultiplication(mul *parser.Multiplication, env *Environment) object.Object {
	result := evalUnary(mul.Left, env)
	for _, op := range mul.Ops {
		right := evalUnary(op.Right, env)
		result = evalArithOp(op.Op, result, right)
	}
	return result
}

func evalUnary(u *parser.Unary, env *Environment) object.Object {
	if u.Not != nil {
		return &object.BoolObj{Value: !object.IsTruthy(evalUnary(u.Not, env))}
	}
	if u.Neg != nil {
		val := evalUnary(u.Neg, env)
		num, ok := val.(*object.NumberObj)
		if !ok {
			runtimeError("'-' requires a num")
		}
		return &object.NumberObj{Value: -num.Value}
	}
	if u.Pos != nil {
		return evalUnary(u.Pos, env)
	}
	return evalPrimary(u.Primary, env)
}

func evalPrimary(p *parser.Primary, env *Environment) object.Object {
	switch {
	case p.IndexAccess != nil:
		return evalIndexAccess(p.IndexAccess, env)
	case p.FuncCall != nil:
		return evalFuncCall(p.FuncCall, env)
	case p.DictLit != nil:
		return evalDictLit(p.DictLit, env)
	case p.ListLit != nil:
		return evalListLit(p.ListLit, env)
	case p.Number != nil:
		return &object.NumberObj{Value: *p.Number}
	case p.String != nil:
		s := (*p.String)[1 : len(*p.String)-1]
		s = preprocessor.ProcessEscapes(s)
		s = interpolateString(s, env)
		return &object.StringObj{Value: s}
	case p.Ident != nil:
		switch *p.Ident {
		case "true":
			return &object.BoolObj{Value: true}
		case "false":
			return &object.BoolObj{Value: false}
		case "null":
			return object.Null
		default:
			obj, ok := env.Get(*p.Ident)
			if !ok {
				runtimeError("Undefined name: '%s'", *p.Ident)
			}
			return obj
		}
	case p.SubExpr != nil:
		return evalExpr(p.SubExpr, env)
	}
	return object.Null
}

func evalIndexAccess(ia *parser.IndexAccess, env *Environment) object.Object {
	obj, ok := env.Get(ia.Name)
	if !ok {
		runtimeError("Undefined name: '%s'", ia.Name)
	}
	idx := evalExpr(ia.Index, env)

	switch o := obj.(type) {
	case *object.DictObj:
		key, ok := idx.(*object.StringObj)
		if !ok {
			runtimeError("Dict key must be a str")
		}
		val, exists := o.Pairs[key.Value]
		if !exists {
			return object.Null
		}
		return val
	case *object.ListObj:
		idxNum, ok := idx.(*object.NumberObj)
		if !ok {
			runtimeError("Index must be a num")
		}
		i := int(idxNum.Value)
		if i < 0 || i >= len(o.Elements) {
			runtimeError("Index %d out of range (length %d)", i, len(o.Elements))
		}
		return o.Elements[i]
	case *object.StringObj:
		idxNum, ok := idx.(*object.NumberObj)
		if !ok {
			runtimeError("Index must be a num")
		}
		i := int(idxNum.Value)
		if i < 0 || i >= len(o.Value) {
			runtimeError("Index %d out of range (length %d)", i, len(o.Value))
		}
		return &object.StringObj{Value: string(o.Value[i])}
	default:
		runtimeError("'%s' is not indexable", ia.Name)
	}
	return object.Null
}

func evalDictLit(dl *parser.DictLit, env *Environment) object.Object {
	pairs := make(map[string]object.Object)
	keys := make([]string, 0, len(dl.Entries))
	for _, entry := range dl.Entries {
		key := evalExpr(entry.Key, env)
		keyStr, ok := key.(*object.StringObj)
		if !ok {
			runtimeError("Dict key must be a str")
		}
		pairs[keyStr.Value] = evalExpr(entry.Value, env)
		keys = append(keys, keyStr.Value)
	}
	return &object.DictObj{Pairs: pairs, Keys: keys}
}

func evalListLit(ll *parser.ListLit, env *Environment) object.Object {
	elements := make([]object.Object, len(ll.Elements))
	for i, el := range ll.Elements {
		elements[i] = evalExpr(el, env)
	}
	return &object.ListObj{Elements: elements}
}

func interpolateString(s string, env *Environment) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			end := strings.IndexByte(s[i:], '}')
			if end == -1 {
				result.WriteByte(s[i])
				i++
				continue
			}
			name := strings.TrimSpace(s[i+1 : i+end])
			if name == "" {
				result.WriteString(s[i : i+end+1])
				i += end + 1
				continue
			}
			obj, ok := env.Get(name)
			if ok {
				result.WriteString(obj.Inspect())
			} else {
				result.WriteString("{" + name + "}")
			}
			i += end + 1
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

func evalFuncCall(fc *parser.FuncCall, env *Environment) object.Object {
	if fc.Name == "try" {
		if len(fc.Args) != 1 {
			runtimeError("try() requires 1 argument")
		}
		return evalTry(fc.Args[0], env)
	}

	if fn, ok := builtins[fc.Name]; ok {
		return fn(fc.Args, env)
	}

	if result, ok := callImportedStdlib(fc.Name, fc.Args, env); ok {
		return result
	}
	if result, ok := callUserModule(fc.Name, fc.Args, env); ok {
		return result
	}
	if result, ok := callWildcardModule(fc.Name, fc.Args, env); ok {
		return result
	}

	obj, ok := env.Get(fc.Name)
	if !ok {
		runtimeError("Undefined function: '%s'", fc.Name)
	}

	fn, ok := obj.(*object.FuncObj)
	if !ok {
		runtimeError("'%s' is not a function", fc.Name)
	}

	if len(fc.Args) != len(fn.Params) {
		runtimeError("Function '%s' expects %d args, got %d", fn.Name, len(fn.Params), len(fc.Args))
	}

	closureEnv, ok := fn.Env.(*Environment)
	if !ok {
		runtimeError("Invalid function environment")
	}

	fnEnv := NewEnclosed(closureEnv)
	for i, param := range fn.Params {
		val := evalExpr(fc.Args[i], env)
		if param.Type != "" {
			valType := object.TypeName(val)
			if valType != param.Type {
				runtimeError("Function '%s' arg '%s' is %s, expected %s", fn.Name, param.Name, valType, param.Type)
			}
		}
		fnEnv.Set(param.Name, val)
	}

	result := evalStatements(fn.Body.Statements, fnEnv)
	if ret, ok := result.(*object.ReturnObj); ok {
		if fn.ReturnType != "" {
			retType := object.TypeName(ret.Value)
			if retType != fn.ReturnType {
				runtimeError("Function '%s' returns %s, expected %s", fn.Name, retType, fn.ReturnType)
			}
		}
		return ret.Value
	}
	if fn.ReturnType != "" && result != nil {
		retType := object.TypeName(result)
		if retType != fn.ReturnType {
			runtimeError("Function '%s' returns %s, expected %s", fn.Name, retType, fn.ReturnType)
		}
	}
	return result
}

// --- Operators ---

func evalComparisonOp(op string, left, right object.Object) object.Object {
	ln, lok := left.(*object.NumberObj)
	rn, rok := right.(*object.NumberObj)
	if lok && rok {
		switch op {
		case ">":
			return &object.BoolObj{Value: ln.Value > rn.Value}
		case "<":
			return &object.BoolObj{Value: ln.Value < rn.Value}
		case ">=":
			return &object.BoolObj{Value: ln.Value >= rn.Value}
		case "<=":
			return &object.BoolObj{Value: ln.Value <= rn.Value}
		case "==":
			return &object.BoolObj{Value: ln.Value == rn.Value}
		case "!=":
			return &object.BoolObj{Value: ln.Value != rn.Value}
		}
	}

	ls, lsok := left.(*object.StringObj)
	rs, rsok := right.(*object.StringObj)
	if lsok && rsok {
		switch op {
		case "==":
			return &object.BoolObj{Value: ls.Value == rs.Value}
		case "!=":
			return &object.BoolObj{Value: ls.Value != rs.Value}
		}
	}

	// Bool equality
	lb, lbok := left.(*object.BoolObj)
	rb, rbok := right.(*object.BoolObj)
	if lbok && rbok {
		switch op {
		case "==":
			return &object.BoolObj{Value: lb.Value == rb.Value}
		case "!=":
			return &object.BoolObj{Value: lb.Value != rb.Value}
		}
	}

	// Null equality
	if _, lok := left.(*object.NullObj); lok {
		if _, rok := right.(*object.NullObj); rok {
			if op == "==" {
				return &object.BoolObj{Value: true}
			}
			if op == "!=" {
				return &object.BoolObj{Value: false}
			}
		}
	}

	if op == "in" {
		return evalMembership(left, right)
	}

	runtimeError("Operation '%s' not supported for these types", op)
	return object.Null
}

func evalMembership(needle, haystack object.Object) object.Object {
	switch h := haystack.(type) {
	case *object.ListObj:
		for _, el := range h.Elements {
			if object.ObjectsEqual(needle, el) {
				return &object.BoolObj{Value: true}
			}
		}
		return &object.BoolObj{Value: false}
	case *object.DictObj:
		key, ok := needle.(*object.StringObj)
		if !ok {
			runtimeError("Dict key must be a str")
		}
		_, exists := h.Pairs[key.Value]
		return &object.BoolObj{Value: exists}
	case *object.StringObj:
		sub, ok := needle.(*object.StringObj)
		if !ok {
			runtimeError("'in' for str requires a str")
		}
		return &object.BoolObj{Value: strings.Contains(h.Value, sub.Value)}
	default:
		runtimeError("'in' not supported for this type")
	}
	return &object.BoolObj{Value: false}
}

func evalArithOp(op string, left, right object.Object) object.Object {
	if op == "+" {
		if ls, ok := left.(*object.StringObj); ok {
			if rs, ok := right.(*object.StringObj); ok {
				return &object.StringObj{Value: ls.Value + rs.Value}
			}
		}
	}

	ln, lok := left.(*object.NumberObj)
	rn, rok := right.(*object.NumberObj)
	if !lok || !rok {
		runtimeError("Operation '%s' requires num values", op)
	}

	switch op {
	case "+":
		return &object.NumberObj{Value: ln.Value + rn.Value}
	case "-":
		return &object.NumberObj{Value: ln.Value - rn.Value}
	case "*":
		return &object.NumberObj{Value: ln.Value * rn.Value}
	case "/":
		if rn.Value == 0 {
			runtimeError("Division by zero")
		}
		return &object.NumberObj{Value: ln.Value / rn.Value}
	case "%":
		if rn.Value == 0 {
			runtimeError("Division by zero")
		}
		return &object.NumberObj{Value: float64(int64(ln.Value) % int64(rn.Value))}
	case "**":
		return &object.NumberObj{Value: math.Pow(ln.Value, rn.Value)}
	}

	return object.Null
}
