package codegen

import (
	"strings"

	"github.com/erickweyunga/sintax/preprocessor"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	llvmValue "github.com/llir/llvm/ir/value"

	"github.com/erickweyunga/sintax/parser"
)

func (cg *CodeGen) compileExpr(expr *parser.Expr) llvmValue.Value {
	left := cg.compileLogicalAnd(expr.Left)
	for _, op := range expr.Ops {
		leftTruthy := cg.callRT("sx_truthy", left)
		cond := cg.block.NewICmp(enum.IPredNE, leftTruthy, constant.NewInt(i32, 0))
		leftBlock := cg.block

		thenBlock := cg.newBlock("or.true")
		elseBlock := cg.newBlock("or.false")
		mergeBlock := cg.newBlock("or.merge")

		leftBlock.NewCondBr(cond, thenBlock, elseBlock)

		cg.block = thenBlock
		thenBlock.NewBr(mergeBlock)

		cg.block = elseBlock
		right := cg.compileLogicalAnd(op.Right)
		rightBlock := cg.block
		rightBlock.NewBr(mergeBlock)

		cg.block = mergeBlock
		phi := cg.block.NewPhi(ir.NewIncoming(left, thenBlock), ir.NewIncoming(right, rightBlock))
		left = phi
	}
	return left
}

func (cg *CodeGen) compileLogicalAnd(and *parser.LogicalAnd) llvmValue.Value {
	left := cg.compileComparison(and.Left)
	for _, op := range and.Ops {
		leftTruthy := cg.callRT("sx_truthy", left)
		cond := cg.block.NewICmp(enum.IPredNE, leftTruthy, constant.NewInt(i32, 0))
		leftBlock := cg.block

		thenBlock := cg.newBlock("and.true")
		falseBlock := cg.newBlock("and.false")
		mergeBlock := cg.newBlock("and.merge")

		leftBlock.NewCondBr(cond, thenBlock, falseBlock)

		cg.block = thenBlock
		right := cg.compileComparison(op.Right)
		rightBlock := cg.block
		rightBlock.NewBr(mergeBlock)

		cg.block = falseBlock
		falseBlock.NewBr(mergeBlock)

		cg.block = mergeBlock
		phi := cg.block.NewPhi(ir.NewIncoming(right, rightBlock), ir.NewIncoming(left, falseBlock))
		left = phi
	}
	return left
}

func (cg *CodeGen) compileComparison(cmp *parser.Comparison) llvmValue.Value {
	left := cg.compileAddition(cmp.Left)
	if cmp.Op != "" {
		right := cg.compileAddition(cmp.Right)
		switch cmp.Op {
		case "==":
			return cg.callRT("sx_eq", left, right)
		case "!=":
			return cg.callRT("sx_neq", left, right)
		case ">":
			return cg.callRT("sx_gt", left, right)
		case "<":
			return cg.callRT("sx_lt", left, right)
		case ">=":
			return cg.callRT("sx_gte", left, right)
		case "<=":
			return cg.callRT("sx_lte", left, right)
		case "in":
			return cg.callRT("sx_in", left, right)
		}
	}
	return left
}

func (cg *CodeGen) compileAddition(add *parser.Addition) llvmValue.Value {
	result := cg.compileMultiplication(add.Left)
	for _, op := range add.Ops {
		right := cg.compileMultiplication(op.Right)
		switch op.Op {
		case "+":
			result = cg.callRT("sx_add", result, right)
		case "-":
			result = cg.callRT("sx_sub", result, right)
		}
	}
	return result
}

func (cg *CodeGen) compileMultiplication(mul *parser.Multiplication) llvmValue.Value {
	result := cg.compileUnary(mul.Left)
	for _, op := range mul.Ops {
		right := cg.compileUnary(op.Right)
		switch op.Op {
		case "*":
			result = cg.callRT("sx_mul", result, right)
		case "/":
			result = cg.callRT("sx_div", result, right)
		case "%":
			result = cg.callRT("sx_mod", result, right)
		case "**":
			result = cg.callRT("sx_pow", result, right)
		}
	}
	return result
}

func (cg *CodeGen) compileUnary(u *parser.Unary) llvmValue.Value {
	if u.Not != nil {
		val := cg.compileUnary(u.Not)
		return cg.callRT("sx_not", val)
	}
	if u.Neg != nil {
		val := cg.compileUnary(u.Neg)
		zero := cg.callRT("sx_number", constant.NewFloat(types.Double, 0))
		return cg.callRT("sx_sub", zero, val)
	}
	if u.Pos != nil {
		return cg.compileUnary(u.Pos)
	}
	return cg.compilePrimary(u.Primary)
}

func (cg *CodeGen) compilePrimary(p *parser.Primary) llvmValue.Value {
	switch {
	case p.IndexAccess != nil:
		collection := cg.getVar(p.IndexAccess.Name)
		idx := cg.compileExpr(p.IndexAccess.Index)
		return cg.callRT("sx_index", collection, idx)
	case p.FuncCall != nil:
		return cg.compileFuncCall(p.FuncCall)
	case p.DictLit != nil:
		dict := cg.callRT("sx_dict_new")
		for _, entry := range p.DictLit.Entries {
			key := cg.compileExpr(entry.Key)
			val := cg.compileExpr(entry.Value)
			cg.callRTVoid("sx_index_set", dict, key, val)
		}
		return dict
	case p.ListLit != nil:
		list := cg.callRT("sx_list_new")
		for _, el := range p.ListLit.Elements {
			val := cg.compileExpr(el)
			cg.callRTVoid("sx_list_append", list, val)
		}
		return list
	case p.Number != nil:
		return cg.callRT("sx_number", constant.NewFloat(types.Double, *p.Number))
	case p.String != nil:
		s := (*p.String)[1 : len(*p.String)-1]
		s = preprocessor.ProcessEscapes(s)
		if strings.Contains(s, "{") && strings.Contains(s, "}") {
			return cg.compileInterpolatedString(s)
		}
		str := cg.globalString(s)
		return cg.callRT("sx_string", str)
	case p.Ident != nil:
		switch *p.Ident {
		case "true":
			return cg.callRT("sx_bool", constant.NewInt(i32, 1))
		case "false":
			return cg.callRT("sx_bool", constant.NewInt(i32, 0))
		case "null":
			return cg.callRT("sx_null")
		default:
			return cg.getVar(*p.Ident)
		}
	case p.SubExpr != nil:
		return cg.compileExpr(p.SubExpr)
	}
	return cg.callRT("sx_null")
}

// Builtin mappings: Sintax name → C runtime name
var oneArgBuiltins = map[string]string{
	"type":   "sx_type",
	"len":    "sx_len",
	"keys":   "sx_dict_keys",
	"values": "sx_dict_values",
	"num":    "sx_to_number",
	"str":    "sx_to_string",
	"bool":   "sx_to_bool",
}

var twoArgBuiltins = map[string]string{
	"pop": "sx_list_remove",
	"has": "sx_dict_has",
}

func (cg *CodeGen) compileFuncCall(fc *parser.FuncCall) llvmValue.Value {
	// Single-arg builtins (name → runtime)
	if rtName, ok := oneArgBuiltins[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]))
	}

	// Two-arg builtins (name → runtime)
	if rtName, ok := twoArgBuiltins[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]))
	}

	// Special builtins (variable args or custom logic)
	switch fc.Name {
	case "print":
		return cg.compilePrint(fc)
	case "push":
		list := cg.compileExpr(fc.Args[0])
		cg.callRTVoid("sx_list_append", list, cg.compileExpr(fc.Args[1]))
		return list
	case "range":
		if len(fc.Args) == 1 {
			return cg.callRT("sx_range", cg.callRT("sx_number", constant.NewFloat(types.Double, 0)), cg.compileExpr(fc.Args[0]))
		}
		return cg.callRT("sx_range", cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]))
	case "input":
		if len(fc.Args) > 0 {
			return cg.callRT("sx_input", cg.compileExpr(fc.Args[0]))
		}
		return cg.callRT("sx_input", cg.callRT("sx_null"))
	}

	// User-defined function
	if fn, ok := cg.userFuncs[fc.Name]; ok {
		args := make([]llvmValue.Value, len(fc.Args))
		for i, arg := range fc.Args {
			args[i] = cg.compileExpr(arg)
		}
		return cg.block.NewCall(fn, args...)
	}

	return cg.callRT("sx_null")
}

func (cg *CodeGen) compilePrint(fc *parser.FuncCall) llvmValue.Value {
	if len(fc.Args) == 1 {
		cg.callRTVoid("sx_print", cg.compileExpr(fc.Args[0]))
		return cg.callRT("sx_null")
	}
	parts := make([]llvmValue.Value, len(fc.Args))
	for i, arg := range fc.Args {
		parts[i] = cg.callRT("sx_to_string", cg.compileExpr(arg))
	}
	result := parts[0]
	space := cg.callRT("sx_string", cg.globalString(" "))
	for _, part := range parts[1:] {
		result = cg.callRT("sx_add", result, space)
		result = cg.callRT("sx_add", result, part)
	}
	cg.callRTVoid("sx_print", result)
	return cg.callRT("sx_null")
}

func (cg *CodeGen) compileInterpolatedString(s string) llvmValue.Value {
	var parts []llvmValue.Value
	i := 0
	current := ""

	for i < len(s) {
		if s[i] == '{' {
			end := strings.IndexByte(s[i:], '}')
			if end == -1 {
				current += string(s[i])
				i++
				continue
			}
			if current != "" {
				parts = append(parts, cg.callRT("sx_string", cg.globalString(current)))
				current = ""
			}
			varName := strings.TrimSpace(s[i+1 : i+end])
			parts = append(parts, cg.callRT("sx_to_string", cg.getVar(varName)))
			i += end + 1
		} else {
			current += string(s[i])
			i++
		}
	}
	if current != "" {
		parts = append(parts, cg.callRT("sx_string", cg.globalString(current)))
	}

	if len(parts) == 0 {
		return cg.callRT("sx_string", cg.globalString(""))
	}
	result := parts[0]
	for _, part := range parts[1:] {
		result = cg.callRT("sx_add", result, part)
	}
	return result
}
