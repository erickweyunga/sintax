package evaluator

import (
	"fmt"
	"math"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

// SourceInfo holds original source context for error reporting.
type SourceInfo struct {
	Filename string
	Lines    []string // original source lines (0-indexed)
	LineMap  []int    // preprocessed line (1-based) → original line (1-based)
}

// RuntimeError is a recoverable error with source location.
type RuntimeError struct {
	Message string
	Line    int    // original source line number (1-based, 0 = unknown)
	Source  string // the original source line text
}

func (e RuntimeError) Error() string {
	if e.Line > 0 && e.Source != "" {
		return fmt.Sprintf("[mstari %d] %s\n  | %s", e.Line, e.Message, e.Source)
	}
	if e.Line > 0 {
		return fmt.Sprintf("[mstari %d] %s", e.Line, e.Message)
	}
	return e.Message
}

// Package-level state for error context.
var (
	sourceInfo *SourceInfo
	currentLine int // current preprocessed line being evaluated
)

func runtimeError(format string, args ...interface{}) {
	re := RuntimeError{Message: fmt.Sprintf(format, args...)}
	if sourceInfo != nil && currentLine > 0 && currentLine <= len(sourceInfo.LineMap) {
		origLine := sourceInfo.LineMap[currentLine-1]
		re.Line = origLine
		if origLine > 0 && origLine <= len(sourceInfo.Lines) {
			re.Source = strings.TrimSpace(sourceInfo.Lines[origLine-1])
		}
	}
	panic(re)
}

// SetSourceInfo sets the source context for error reporting.
func SetSourceInfo(info *SourceInfo) {
	sourceInfo = info
}

func recoverError(err *error) {
	if r := recover(); r != nil {
		if re, ok := r.(RuntimeError); ok {
			*err = re
		} else {
			panic(r)
		}
	}
}

// Eval evaluates a program and returns any runtime error.
func Eval(program *parser.Program) (err error) {
	defer recoverError(&err)
	env := NewEnvironment()
	evalStatements(program.Statements, env)
	return nil
}

// EvalWithEnv evaluates a program in an existing environment (used by REPL).
func EvalWithEnv(program *parser.Program, env *Environment) (result object.Object, err error) {
	defer recoverError(&err)
	result = evalStatements(program.Statements, env)
	return result, nil
}

func evalStatements(stmts []*parser.Statement, env *Environment) object.Object {
	var result object.Object
	for _, stmt := range stmts {
		result = evalStatement(stmt, env)
		switch result.(type) {
		case *object.ReturnObj, *object.BreakObj, *object.ContinueObj:
			return result
		}
	}
	return result
}

func evalStatement(stmt *parser.Statement, env *Environment) object.Object {
	// Track position for error reporting
	if stmt.Pos.Line > 0 {
		currentLine = stmt.Pos.Line
	}
	switch {
	case stmt.FuncDef != nil:
		return evalFuncDef(stmt.FuncDef, env)
	case stmt.IfStmt != nil:
		return evalIfStmt(stmt.IfStmt, env)
	case stmt.SwitchStmt != nil:
		return evalSwitchStmt(stmt.SwitchStmt, env)
	case stmt.WhileStmt != nil:
		return evalWhileStmt(stmt.WhileStmt, env)
	case stmt.ForStmt != nil:
		return evalForStmt(stmt.ForStmt, env)
	case stmt.PrintStmt != nil:
		return evalPrintStmt(stmt.PrintStmt, env)
	case stmt.ReturnStmt != nil:
		return evalReturnStmt(stmt.ReturnStmt, env)
	case stmt.TypedAssign != nil:
		return evalTypedAssign(stmt.TypedAssign, env)
	case stmt.IndexAssign != nil:
		return evalIndexAssign(stmt.IndexAssign, env)
	case stmt.CompoundAssign != nil:
		return evalCompoundAssign(stmt.CompoundAssign, env)
	case stmt.Assignment != nil:
		return evalAssignment(stmt.Assignment, env)
	case stmt.ExprStmt != nil:
		if stmt.ExprStmt.Expr.IsBareLiteral(0) {
			return &object.BreakObj{}
		}
		if stmt.ExprStmt.Expr.IsBareLiteral(1) {
			return &object.ContinueObj{}
		}
		return evalExpr(stmt.ExprStmt.Expr, env)
	}
	return object.Null
}

func evalFuncDef(fd *parser.FuncDef, env *Environment) object.Object {
	params := make([]object.FuncParam, len(fd.Params))
	for i, p := range fd.Params {
		typ := ""
		if p.Type != nil {
			typ = object.NormalizeType(*p.Type)
		}
		params[i] = object.FuncParam{Name: p.Name, Type: typ}
	}
	retType := ""
	if fd.ReturnType != nil {
		retType = object.NormalizeType(*fd.ReturnType)
	}
	fn := &object.FuncObj{
		Name:       fd.Name,
		Params:     params,
		ReturnType: retType,
		Body:       fd.Body,
		Env:        env,
	}
	env.Set(fd.Name, fn)
	return fn
}

func evalIfStmt(ifStmt *parser.IfStmt, env *Environment) object.Object {
	cond := evalExpr(ifStmt.Condition, env)
	if object.IsTruthy(cond) {
		return evalStatements(ifStmt.Body.Statements, env)
	} else if ifStmt.Else != nil {
		return evalStatements(ifStmt.Else.Statements, env)
	}
	return object.Null
}

func evalSwitchStmt(sw *parser.SwitchStmt, env *Environment) object.Object {
	val := evalExpr(sw.Value, env)

	for _, c := range sw.Cases {
		caseVal := evalExpr(c.Value, env)
		if object.ObjectsEqual(val, caseVal) {
			return evalStatements(c.Body.Statements, env)
		}
	}

	if sw.Default != nil {
		return evalStatements(sw.Default.Statements, env)
	}
	return object.Null
}


func evalWhileStmt(ws *parser.WhileStmt, env *Environment) object.Object {
	var result object.Object = object.Null
	for {
		cond := evalExpr(ws.Condition, env)
		if !object.IsTruthy(cond) {
			break
		}
		result = evalStatements(ws.Body.Statements, env)
		if _, ok := result.(*object.BreakObj); ok {
			break
		}
		if _, ok := result.(*object.ContinueObj); ok {
			continue
		}
		if _, ok := result.(*object.ReturnObj); ok {
			return result
		}
	}
	return object.Null
}

func evalForStmt(fs *parser.ForStmt, env *Environment) object.Object {
	iter := evalExpr(fs.Iter, env)

	var items []object.Object
	switch it := iter.(type) {
	case *object.ListObj:
		items = it.Elements
	case *object.DictObj:
		for _, k := range it.Keys {
			items = append(items, &object.StringObj{Value: k})
		}
	case *object.StringObj:
		for _, ch := range it.Value {
			items = append(items, &object.StringObj{Value: string(ch)})
		}
	default:
		runtimeError("'kwa' inahitaji safu, kamusi, au tungo")
	}

	var result object.Object = object.Null
	for _, item := range items {
		env.Set(fs.Var, item)
		result = evalStatements(fs.Body.Statements, env)
		if _, ok := result.(*object.BreakObj); ok {
			break
		}
		if _, ok := result.(*object.ContinueObj); ok {
			continue
		}
		if _, ok := result.(*object.ReturnObj); ok {
			return result
		}
	}
	return object.Null
}

func evalPrintStmt(ps *parser.PrintStmt, env *Environment) object.Object {
	val := evalExpr(ps.Value, env)
	fmt.Println(val.Inspect())
	return object.Null
}

func evalReturnStmt(ret *parser.ReturnStmt, env *Environment) object.Object {
	val := evalExpr(ret.Value, env)
	return &object.ReturnObj{Value: val}
}

func evalIndexAssign(ia *parser.IndexAssign, env *Environment) object.Object {
	obj, ok := env.Get(ia.Name)
	if !ok {
		runtimeError("Jina halijulikani: '%s'", ia.Name)
	}

	idx := evalExpr(ia.Index, env)
	val := evalExpr(ia.Value, env)

	switch o := obj.(type) {
	case *object.DictObj:
		key, ok := idx.(*object.StringObj)
		if !ok {
			runtimeError("Ufunguo wa kamusi lazima uwe tungo")
		}
		if _, exists := o.Pairs[key.Value]; !exists {
			o.Keys = append(o.Keys, key.Value)
		}
		o.Pairs[key.Value] = val
	case *object.ListObj:
		idxNum, ok := idx.(*object.NumberObj)
		if !ok {
			runtimeError("Fahirisi lazima iwe nambari")
		}
		i := int(idxNum.Value)
		if i < 0 || i >= len(o.Elements) {
			runtimeError("Fahirisi %d nje ya masafa (urefu %d)", i, len(o.Elements))
		}
		o.Elements[i] = val
	default:
		runtimeError("'%s' si safu wala kamusi", ia.Name)
	}
	return val
}

func evalCompoundAssign(ca *parser.CompoundAssign, env *Environment) object.Object {
	obj, ok := env.Get(ca.Name)
	if !ok {
		runtimeError("Jina halijulikani: '%s'", ca.Name)
	}
	right := evalExpr(ca.Value, env)
	// Map += to +, -= to -, etc.
	op := string(ca.Op[0])
	result := evalArithOp(op, obj, right)
	// Enforce type if typed
	if typ, ok := env.GetType(ca.Name); ok && typ != "" {
		valType := object.TypeName(result)
		if valType != typ {
			runtimeError("Aina si sahihi: '%s' ni %s, inahitaji %s", ca.Name, valType, typ)
		}
	}
	env.Set(ca.Name, result)
	return result
}

func evalTypedAssign(ta *parser.TypedAssign, env *Environment) object.Object {
	val := evalExpr(ta.Value, env)
	expectedType := object.NormalizeType(ta.Type)
	valType := object.TypeName(val)
	if valType != expectedType {
		runtimeError("Aina si sahihi: '%s' ni %s, inahitaji %s", ta.Name, valType, expectedType)
	}
	env.SetTyped(ta.Name, expectedType, val)
	return val
}

func evalAssignment(assign *parser.Assignment, env *Environment) object.Object {
	val := evalExpr(assign.Value, env)
	// Enforce type if variable was declared with a type
	if typ, ok := env.GetType(assign.Name); ok && typ != "" {
		valType := object.TypeName(val)
		if valType != typ {
			runtimeError("Aina si sahihi: '%s' ni %s, inahitaji %s", assign.Name, valType, typ)
		}
	}
	env.Set(assign.Name, val)
	return val
}

// Expression evaluation

func evalExpr(expr *parser.Expr, env *Environment) object.Object {
	result := evalLogicalAnd(expr.Left, env)
	for _, op := range expr.Ops {
		// Short-circuit: if left is truthy, skip right
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
		// Short-circuit: if left is falsy, skip right
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
		val := evalUnary(u.Not, env)
		return &object.BoolObj{Value: !object.IsTruthy(val)}
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
		s := (*p.String)[1 : len(*p.String)-1] // remove first/last quote only
		s = preprocessor.ProcessEscapes(s)
		s = interpolateString(s, env)
		return &object.StringObj{Value: s}
	case p.Ident != nil:
		name := *p.Ident
		if name == "kweli" {
			return &object.BoolObj{Value: true}
		}
		if name == "sikweli" {
			return &object.BoolObj{Value: false}
		}
		if name == "tupu" {
			return object.Null
		}
		obj, ok := env.Get(name)
		if !ok {
			runtimeError("Jina halijulikani: '%s'", name)
		}
		return obj
	case p.SubExpr != nil:
		return evalExpr(p.SubExpr, env)
	}
	return object.Null
}

func evalIndexAccess(ia *parser.IndexAccess, env *Environment) object.Object {
	obj, ok := env.Get(ia.Name)
	if !ok {
		runtimeError("Jina halijulikani: '%s'", ia.Name)
	}

	idx := evalExpr(ia.Index, env)

	switch o := obj.(type) {
	case *object.DictObj:
		key, ok := idx.(*object.StringObj)
		if !ok {
			runtimeError("Ufunguo wa kamusi lazima uwe tungo")
		}
		val, exists := o.Pairs[key.Value]
		if !exists {
			return object.Null
		}
		return val
	case *object.ListObj:
		idxNum, ok := idx.(*object.NumberObj)
		if !ok {
			runtimeError("Fahirisi lazima iwe nambari")
		}
		i := int(idxNum.Value)
		if i < 0 || i >= len(o.Elements) {
			runtimeError("Fahirisi %d nje ya masafa (urefu %d)", i, len(o.Elements))
		}
		return o.Elements[i]
	case *object.StringObj:
		idxNum, ok := idx.(*object.NumberObj)
		if !ok {
			runtimeError("Fahirisi lazima iwe nambari")
		}
		i := int(idxNum.Value)
		if i < 0 || i >= len(o.Value) {
			runtimeError("Fahirisi %d nje ya masafa (urefu %d)", i, len(o.Value))
		}
		return &object.StringObj{Value: string(o.Value[i])}
	default:
		runtimeError("'%s' haiwezi kufikia kwa fahirisi", ia.Name)
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
			runtimeError("Ufunguo wa kamusi lazima uwe tungo")
		}
		pairs[keyStr.Value] = evalExpr(entry.Value, env)
		keys = append(keys, keyStr.Value)
	}
	return &object.DictObj{Pairs: pairs, Keys: keys}
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

func evalListLit(ll *parser.ListLit, env *Environment) object.Object {
	elements := make([]object.Object, len(ll.Elements))
	for i, el := range ll.Elements {
		elements[i] = evalExpr(el, env)
	}
	return &object.ListObj{Elements: elements}
}

func evalFuncCall(fc *parser.FuncCall, env *Environment) object.Object {
	// Check built-in functions first
	if fn, ok := builtins[fc.Name]; ok {
		return fn(fc.Args, env)
	}

	// User-defined functions
	obj, ok := env.Get(fc.Name)
	if !ok {
		runtimeError("Unda haijulikani: '%s'", fc.Name)
	}

	fn, ok := obj.(*object.FuncObj)
	if !ok {
		runtimeError("'%s' si unda", fc.Name)
	}

	if len(fc.Args) != len(fn.Params) {
		runtimeError("Unda '%s' inahitaji hoja %d, imepata %d", fn.Name, len(fn.Params), len(fc.Args))
	}

	// Resolve the closure environment
	closureEnv, ok := fn.Env.(*Environment)
	if !ok {
		runtimeError("Mazingira ya unda si sahihi")
	}

	fnEnv := NewEnclosed(closureEnv)
	for i, param := range fn.Params {
		val := evalExpr(fc.Args[i], env)
		// Enforce parameter type
		if param.Type != "" {
			valType := object.TypeName(val)
			if valType != param.Type {
				runtimeError("Unda '%s' hoja '%s' ni %s, inahitaji %s", fn.Name, param.Name, valType, param.Type)
			}
		}
		fnEnv.Set(param.Name, val)
	}

	result := evalStatements(fn.Body.Statements, fnEnv)
	if ret, ok := result.(*object.ReturnObj); ok {
		// Enforce return type
		if fn.ReturnType != "" {
			retType := object.TypeName(ret.Value)
			if retType != fn.ReturnType {
				runtimeError("Unda '%s' rudisha %s, inahitaji %s", fn.Name, retType, fn.ReturnType)
			}
		}
		return ret.Value
	}
	// Enforce return type on implicit return
	if fn.ReturnType != "" && result != nil {
		retType := object.TypeName(result)
		if retType != fn.ReturnType {
			runtimeError("Unda '%s' rudisha %s, inahitaji %s", fn.Name, retType, fn.ReturnType)
		}
	}
	return result
}

// Operators

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

	// ktk membership operator
	if op == "ktk" {
		return evalMembership(left, right)
	}

	runtimeError("Operesheni '%s' haiwezekani kwa aina hizi", op)
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
			runtimeError("Ufunguo wa kamusi lazima uwe tungo")
		}
		_, exists := h.Pairs[key.Value]
		return &object.BoolObj{Value: exists}
	case *object.StringObj:
		sub, ok := needle.(*object.StringObj)
		if !ok {
			runtimeError("ktk kwa tungo inahitaji tungo")
		}
		return &object.BoolObj{Value: strings.Contains(h.Value, sub.Value)}
	default:
		runtimeError("ktk haiwezi kutumika kwa aina hii")
	}
	return &object.BoolObj{Value: false}
}

func evalArithOp(op string, left, right object.Object) object.Object {
	if op == "+" {
		ls, lsok := left.(*object.StringObj)
		rs, rsok := right.(*object.StringObj)
		if lsok && rsok {
			return &object.StringObj{Value: ls.Value + rs.Value}
		}
	}

	ln, lok := left.(*object.NumberObj)
	rn, rok := right.(*object.NumberObj)
	if !lok || !rok {
		runtimeError("Operesheni '%s' inahitaji nambari", op)
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
			runtimeError("Haiwezekani kugawanya na sifuri")
		}
		return &object.NumberObj{Value: ln.Value / rn.Value}
	case "%":
		if rn.Value == 0 {
			runtimeError("Haiwezekani kugawanya na sifuri")
		}
		return &object.NumberObj{Value: float64(int64(ln.Value) % int64(rn.Value))}
	case "**":
		return &object.NumberObj{Value: math.Pow(ln.Value, rn.Value)}
	}

	return object.Null
}

