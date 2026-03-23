package evaluator

import (
	"math"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

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
	// Evaluate base value
	var result object.Object
	switch {
	case p.Lambda != nil:
		result = evalLambda(p.Lambda, env)
	case p.IndexAccess != nil:
		result = evalIndexAccess(p.IndexAccess, env)
	case p.FuncCall != nil:
		result = evalFuncCall(p.FuncCall, env)
	case p.DictLit != nil:
		result = evalDictLit(p.DictLit, env)
	case p.ListLit != nil:
		result = evalListLit(p.ListLit, env)
	case p.Number != nil:
		result = &object.NumberObj{Value: *p.Number}
	case p.String != nil:
		s := (*p.String)[1 : len(*p.String)-1]
		s = preprocessor.ProcessEscapes(s)
		s = interpolateString(s, env)
		result = &object.StringObj{Value: s}
	case p.Ident != nil:
		switch *p.Ident {
		case "true":
			result = &object.BoolObj{Value: true}
		case "false":
			result = &object.BoolObj{Value: false}
		case "null":
			result = object.Null
		default:
			obj, ok := env.Get(*p.Ident)
			if !ok {
				runtimeError("Undefined name: '%s'", *p.Ident)
			}
			result = obj
		}
	case p.SubExpr != nil:
		result = evalExpr(p.SubExpr, env)
	default:
		result = object.Null
	}

	// Apply method chain: value.method(args).method(args)...
	for _, mc := range p.Methods {
		args := make([]object.Object, len(mc.Args))
		for i, arg := range mc.Args {
			args[i] = evalExpr(arg, env)
		}
		result = evalMethod(result, mc.Name, args)
	}

	return result
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

// --- Lambda ---

func evalLambda(l *parser.Lambda, env *Environment) object.Object {
	params := make([]object.FuncParam, len(l.Params))
	for i, name := range l.Params {
		params[i] = object.FuncParam{Name: name}
	}
	// Wrap the expression as a return statement in a block
	body := &parser.Block{
		Statements: []*parser.Statement{
			{ReturnStmt: &parser.ReturnStmt{Value: l.Body}},
		},
	}
	return &object.FuncObj{
		Name:   "<lambda>",
		Params: params,
		Body:   body,
		Env:    env,
	}
}

// --- Methods ---

func evalMethod(obj object.Object, method string, args []object.Object) object.Object {
	switch o := obj.(type) {
	case *object.StringObj:
		return evalStringMethod(o, method, args)
	case *object.ListObj:
		return evalListMethod(o, method, args)
	case *object.DictObj:
		return evalDictMethod(o, method, args)
	case *object.NumberObj:
		return evalNumberMethod(o, method, args)
	}
	runtimeError("'%s' has no method '%s'", object.TypeName(obj), method)
	return object.Null
}

func evalStringMethod(s *object.StringObj, method string, args []object.Object) object.Object {
	switch method {
	case "len":
		return &object.NumberObj{Value: float64(len(s.Value))}
	case "upper":
		return &object.StringObj{Value: strings.ToUpper(s.Value)}
	case "lower":
		return &object.StringObj{Value: strings.ToLower(s.Value)}
	case "trim":
		return &object.StringObj{Value: strings.TrimSpace(s.Value)}
	case "split":
		if len(args) != 1 {
			runtimeError("str.split() requires 1 argument")
		}
		sep := args[0].(*object.StringObj).Value
		parts := strings.Split(s.Value, sep)
		elements := make([]object.Object, len(parts))
		for i, p := range parts {
			elements[i] = &object.StringObj{Value: p}
		}
		return &object.ListObj{Elements: elements}
	case "replace":
		if len(args) != 2 {
			runtimeError("str.replace() requires 2 arguments")
		}
		old := args[0].(*object.StringObj).Value
		new_ := args[1].(*object.StringObj).Value
		return &object.StringObj{Value: strings.ReplaceAll(s.Value, old, new_)}
	case "contains":
		if len(args) != 1 {
			runtimeError("str.contains() requires 1 argument")
		}
		return &object.BoolObj{Value: strings.Contains(s.Value, args[0].(*object.StringObj).Value)}
	case "starts_with":
		if len(args) != 1 {
			runtimeError("str.starts_with() requires 1 argument")
		}
		return &object.BoolObj{Value: strings.HasPrefix(s.Value, args[0].(*object.StringObj).Value)}
	case "ends_with":
		if len(args) != 1 {
			runtimeError("str.ends_with() requires 1 argument")
		}
		return &object.BoolObj{Value: strings.HasSuffix(s.Value, args[0].(*object.StringObj).Value)}
	case "type":
		return &object.StringObj{Value: "str"}
	}
	runtimeError("str has no method '%s'", method)
	return object.Null
}

func evalListMethod(l *object.ListObj, method string, args []object.Object) object.Object {
	switch method {
	case "len":
		return &object.NumberObj{Value: float64(len(l.Elements))}
	case "push":
		if len(args) != 1 {
			runtimeError("list.push() requires 1 argument")
		}
		l.Elements = append(l.Elements, args[0])
		return l
	case "pop":
		if len(args) != 1 {
			runtimeError("list.pop() requires 1 argument (index)")
		}
		idx := int(args[0].(*object.NumberObj).Value)
		if idx < 0 || idx >= len(l.Elements) {
			runtimeError("Index %d out of range (length %d)", idx, len(l.Elements))
		}
		removed := l.Elements[idx]
		l.Elements = append(l.Elements[:idx], l.Elements[idx+1:]...)
		return removed
	case "contains":
		if len(args) != 1 {
			runtimeError("list.contains() requires 1 argument")
		}
		for _, el := range l.Elements {
			if object.ObjectsEqual(args[0], el) {
				return &object.BoolObj{Value: true}
			}
		}
		return &object.BoolObj{Value: false}
	case "reverse":
		reversed := make([]object.Object, len(l.Elements))
		for i, el := range l.Elements {
			reversed[len(l.Elements)-1-i] = el
		}
		return &object.ListObj{Elements: reversed}
	case "join":
		if len(args) != 1 {
			runtimeError("list.join() requires 1 argument")
		}
		sep := args[0].(*object.StringObj).Value
		parts := make([]string, len(l.Elements))
		for i, el := range l.Elements {
			parts[i] = el.Inspect()
		}
		return &object.StringObj{Value: strings.Join(parts, sep)}
	case "map":
		if len(args) != 1 {
			runtimeError("list.map() requires 1 argument (function)")
		}
		fn, ok := args[0].(*object.FuncObj)
		if !ok {
			runtimeError("list.map() argument must be a function")
		}
		return evalListMap(l, fn)
	case "filter":
		if len(args) != 1 {
			runtimeError("list.filter() requires 1 argument (function)")
		}
		fn, ok := args[0].(*object.FuncObj)
		if !ok {
			runtimeError("list.filter() argument must be a function")
		}
		return evalListFilter(l, fn)
	case "reduce":
		if len(args) != 2 {
			runtimeError("list.reduce() requires 2 arguments (function, initial)")
		}
		fn, ok := args[0].(*object.FuncObj)
		if !ok {
			runtimeError("list.reduce() first argument must be a function")
		}
		return evalListReduce(l, fn, args[1])
	case "each":
		if len(args) != 1 {
			runtimeError("list.each() requires 1 argument (function)")
		}
		fn, ok := args[0].(*object.FuncObj)
		if !ok {
			runtimeError("list.each() argument must be a function")
		}
		return evalListEach(l, fn)
	case "type":
		return &object.StringObj{Value: "list"}
	}
	runtimeError("list has no method '%s'", method)
	return object.Null
}

func callFuncObj(fn *object.FuncObj, args []object.Object) object.Object {
	fnEnv := NewEnclosed(fn.Env.(*Environment))
	for i, param := range fn.Params {
		if i < len(args) {
			fnEnv.Set(param.Name, args[i])
		}
	}
	result := evalStatements(fn.Body.Statements, fnEnv)
	if ret, ok := result.(*object.ReturnObj); ok {
		return ret.Value
	}
	return result
}

func evalListMap(l *object.ListObj, fn *object.FuncObj) object.Object {
	results := make([]object.Object, len(l.Elements))
	for i, el := range l.Elements {
		results[i] = callFuncObj(fn, []object.Object{el})
	}
	return &object.ListObj{Elements: results}
}

func evalListFilter(l *object.ListObj, fn *object.FuncObj) object.Object {
	var results []object.Object
	for _, el := range l.Elements {
		result := callFuncObj(fn, []object.Object{el})
		if object.IsTruthy(result) {
			results = append(results, el)
		}
	}
	return &object.ListObj{Elements: results}
}

func evalListReduce(l *object.ListObj, fn *object.FuncObj, initial object.Object) object.Object {
	acc := initial
	for _, el := range l.Elements {
		acc = callFuncObj(fn, []object.Object{acc, el})
	}
	return acc
}

func evalListEach(l *object.ListObj, fn *object.FuncObj) object.Object {
	for _, el := range l.Elements {
		callFuncObj(fn, []object.Object{el})
	}
	return object.Null
}

func evalDictMethod(d *object.DictObj, method string, args []object.Object) object.Object {
	switch method {
	case "len":
		return &object.NumberObj{Value: float64(len(d.Pairs))}
	case "keys":
		elements := make([]object.Object, len(d.Keys))
		for i, k := range d.Keys {
			elements[i] = &object.StringObj{Value: k}
		}
		return &object.ListObj{Elements: elements}
	case "values":
		elements := make([]object.Object, len(d.Keys))
		for i, k := range d.Keys {
			elements[i] = d.Pairs[k]
		}
		return &object.ListObj{Elements: elements}
	case "has":
		if len(args) != 1 {
			runtimeError("dict.has() requires 1 argument")
		}
		key := args[0].(*object.StringObj).Value
		_, exists := d.Pairs[key]
		return &object.BoolObj{Value: exists}
	case "type":
		return &object.StringObj{Value: "dict"}
	}
	runtimeError("dict has no method '%s'", method)
	return object.Null
}

func evalNumberMethod(n *object.NumberObj, method string, args []object.Object) object.Object {
	switch method {
	case "type":
		return &object.StringObj{Value: "num"}
	}
	runtimeError("num has no method '%s'", method)
	return object.Null
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
