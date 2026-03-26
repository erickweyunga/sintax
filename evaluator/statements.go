package evaluator

import (
	"fmt"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
)

func evalStatement(stmt *parser.Statement, env *Environment) object.Object {
	if stmt.Pos.Line > 0 {
		currentLine = stmt.Pos.Line
	}
	switch {
	case stmt.FuncDef != nil:
		return evalFuncDef(stmt.FuncDef, env)
	case stmt.IfStmt != nil:
		return evalIfStmt(stmt.IfStmt, env)
	case stmt.CatchStmt != nil:
		return evalCatchStmt(stmt.CatchStmt, env)
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
		Pub:        fd.Pub,
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

func evalCatchStmt(cs *parser.CatchStmt, env *Environment) object.Object {
	val := evalExpr(cs.Value, env)
	env.Set(cs.Name, val)
	if _, isErr := val.(*object.ErrorObj); isErr {
		return evalStatements(cs.Body.Statements, env)
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
		runtimeError("'for' requires a list, dict, or str")
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
		runtimeError("Undefined name: '%s'", ia.Name)
	}

	val := evalExpr(ia.Value, env)

	// Navigate through all indices except the last
	target := obj
	for i := 0; i < len(ia.Indices)-1; i++ {
		idx := evalExpr(ia.Indices[i].Index, env)
		target = evalIndexOn(target, idx)
	}

	// Set at the last index
	lastIdx := evalExpr(ia.Indices[len(ia.Indices)-1].Index, env)
	switch o := target.(type) {
	case *object.DictObj:
		key, ok := lastIdx.(*object.StringObj)
		if !ok {
			runtimeError("Dict key must be a str")
		}
		if _, exists := o.Pairs[key.Value]; !exists {
			o.Keys = append(o.Keys, key.Value)
		}
		o.Pairs[key.Value] = val
	case *object.ListObj:
		idxNum, ok := lastIdx.(*object.NumberObj)
		if !ok {
			runtimeError("Index must be a num")
		}
		i := int(idxNum.Value)
		if i < 0 {
			i += len(o.Elements)
		}
		if i < 0 || i >= len(o.Elements) {
			runtimeError("Index %d out of range (length %d)", int(idxNum.Value), len(o.Elements))
		}
		o.Elements[i] = val
	default:
		runtimeError("Cannot assign by index into %s", object.TypeName(target))
	}
	return val
}

func evalCompoundAssign(ca *parser.CompoundAssign, env *Environment) object.Object {
	obj, ok := env.Get(ca.Name)
	if !ok {
		runtimeError("Undefined name: '%s'", ca.Name)
	}
	right := evalExpr(ca.Value, env)
	op := string(ca.Op[0])
	result := evalArithOp(op, obj, right)
	if typ, ok := env.GetType(ca.Name); ok && typ != "" {
		valType := object.TypeName(result)
		if valType != typ {
			runtimeError("Type mismatch: '%s' is %s, expected %s", ca.Name, valType, typ)
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
		runtimeError("Type mismatch: '%s' is %s, expected %s", ta.Name, valType, expectedType)
	}
	env.SetTyped(ta.Name, expectedType, val)
	return val
}

func evalAssignment(assign *parser.Assignment, env *Environment) object.Object {
	val := evalExpr(assign.Value, env)
	if typ, ok := env.GetType(assign.Name); ok && typ != "" {
		valType := object.TypeName(val)
		if valType != typ {
			runtimeError("Type mismatch: '%s' is %s, expected %s", assign.Name, valType, typ)
		}
	}
	env.Set(assign.Name, val)
	return val
}
