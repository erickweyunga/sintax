package codegen

import (
	"fmt"
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
		return cg.callRT("sx_not", cg.compileUnary(u.Not))
	}
	if u.Neg != nil {
		return cg.callRT("sx_sub", cg.callRT("sx_number", constant.NewFloat(types.Double, 0)), cg.compileUnary(u.Neg))
	}
	if u.Pos != nil {
		return cg.compileUnary(u.Pos)
	}
	return cg.compilePrimary(u.Primary)
}

func (cg *CodeGen) compilePrimary(p *parser.Primary) llvmValue.Value {
	var result llvmValue.Value

	switch {
	case p.Lambda != nil:
		result = cg.compileLambda(p.Lambda)
	case p.FuncCall != nil:
		result = cg.compileFuncCall(p.FuncCall)
	case p.DictLit != nil:
		dict := cg.callRT("sx_dict_new")
		for _, entry := range p.DictLit.Entries {
			key := cg.compileExpr(entry.Key)
			val := cg.compileExpr(entry.Value)
			cg.callRTVoid("sx_index_set", dict, key, val)
		}
		result = dict
	case p.ListLit != nil:
		list := cg.callRT("sx_list_new")
		for _, el := range p.ListLit.Elements {
			val := cg.compileExpr(el)
			cg.callRTVoid("sx_list_append", list, val)
		}
		result = list
	case p.Number != nil:
		result = cg.callRT("sx_number", constant.NewFloat(types.Double, *p.Number))
	case p.String != nil:
		raw := *p.String
		s := raw[1 : len(raw)-1]
		if raw[0] == '\'' {
			// Single-quoted: raw string, no interpolation
			s = strings.ReplaceAll(s, "\\'", "'")
			result = cg.callRT("sx_string", cg.globalString(s))
		} else {
			// Double-quoted: escapes + interpolation
			s = preprocessor.ProcessEscapes(s)
			if hasInterpolation(s) {
				result = cg.compileInterpolatedString(s)
			} else {
				result = cg.callRT("sx_string", cg.globalString(s))
			}
		}
	case p.Ident != nil:
		switch *p.Ident {
		case "true":
			result = cg.callRT("sx_bool", constant.NewInt(i32, 1))
		case "false":
			result = cg.callRT("sx_bool", constant.NewInt(i32, 0))
		case "null":
			result = cg.callRT("sx_null")
		default:
			result = cg.getVar(*p.Ident)
		}
	case p.SubExpr != nil:
		result = cg.compileExpr(p.SubExpr)
	default:
		result = cg.callRT("sx_null")
	}

	// Suffix chain: [index] and .method(args) interleaved
	for _, s := range p.Suffix {
		if s.Index != nil {
			idx := cg.compileExpr(s.Index.Index)
			result = cg.callRT("sx_index", result, idx)
		} else if s.Method != nil {
			mc := s.Method
			nameStr := cg.globalString(mc.Name)
			argc := len(mc.Args)
			if argc == 0 {
				result = cg.callRT("sx_method", result, nameStr, constant.NewNull(sxValuePtr), constant.NewInt(i32, 0))
			} else {
				// Allocate array with enough slots for ALL arguments
				argArray := cg.block.NewAlloca(types.NewArray(uint64(argc), sxValuePtr))
				for i, arg := range mc.Args {
					val := cg.compileExpr(arg)
					ptr := cg.block.NewGetElementPtr(
						types.NewArray(uint64(argc), sxValuePtr),
						argArray,
						constant.NewInt(i64, 0),
						constant.NewInt(i64, int64(i)),
					)
					cg.block.NewStore(val, ptr)
				}
				argPtr := cg.block.NewBitCast(argArray, types.NewPointer(sxValuePtr))
				result = cg.callRT("sx_method", result, nameStr, argPtr, constant.NewInt(i32, int64(argc)))
			}
		}
	}

	return result
}

// Native function mappings — __native_* calls map directly to C runtime
var nativeOneArg = map[string]string{
	"__native_sqrt": "__native_sqrt", "__native_sin": "__native_sin",
	"__native_cos": "__native_cos", "__native_tan": "__native_tan",
	"__native_asin": "__native_asin", "__native_acos": "__native_acos",
	"__native_atan": "__native_atan", "__native_log": "__native_log",
	"__native_log2": "__native_log2", "__native_log10": "__native_log10",
	"__native_exp": "__native_exp", "__native_floor": "__native_floor",
	"__native_ceil": "__native_ceil", "__native_round": "__native_round",
	"__native_cbrt": "__native_cbrt",
	"__native_upper": "__native_upper", "__native_lower": "__native_lower",
	"__native_trim": "__native_trim", "__native_char_code": "__native_char_code",
	"__native_from_char_code": "__native_from_char_code",
	"__native_str_reverse": "__native_str_reverse",
	"__native_list_reverse": "__native_list_reverse",
	"__native_read_file": "__native_read_file", "__native_file_exists": "__native_file_exists",
	"__native_delete_file": "__native_delete_file", "__native_getenv": "__native_getenv",
	"__native_exec": "__native_exec", "__native_sleep": "__native_sleep",
	"__native_exit": "__native_exit",
	"__native_json_parse": "__native_json_parse", "__native_json_stringify": "__native_json_stringify",
	"__native_json_pretty": "__native_json_pretty",
}

var nativeNoArg = map[string]string{
	"__native_random": "__native_random",
	"__native_cwd":    "__native_cwd",
	"__native_time":   "__native_time",
}

var nativeTwoArg = map[string]string{
	"__native_pow": "__native_pow", "__native_split": "__native_split",
	"__native_str_repeat": "__native_str_repeat", "__native_index_of": "__native_index_of",
	"__native_list_concat": "__native_list_concat", "__native_dict_delete": "__native_dict_delete",
	"__native_dict_merge": "__native_dict_merge", "__native_write_file": "__native_write_file",
	"__native_format_time": "__native_format_time", "__native_rename": "__native_rename",
	"__native_regex_match": "__native_regex_match", "__native_regex_find": "__native_regex_find",
}

var nativeThreeArg = map[string]string{
	"__native_replace":      "__native_replace",
	"__native_slice":        "__native_slice",
	"__native_list_insert":  "__native_list_insert",
	"__native_regex_replace": "__native_regex_replace",
}

var nativeFourArg = map[string]string{
	"__native_http_request": "__native_http_request",
}

// Builtin mappings
var oneArgBuiltins = map[string]string{
	"type":   "sx_type",
	"len":    "sx_len",
	"keys":   "sx_dict_keys",
	"values": "sx_dict_values",
	"num":    "sx_to_number",
	"str":    "sx_to_string",
	"bool":   "sx_to_bool",
	"err":    "sx_is_error",
	"error":  "sx_error_new",
	"sort":   "sx_sort",
}

var twoArgBuiltins = map[string]string{
	"pop": "sx_list_remove",
	"has": "sx_dict_has",
}

func (cg *CodeGen) compileFuncCall(fc *parser.FuncCall) llvmValue.Value {
	// Single-arg builtins
	if rtName, ok := oneArgBuiltins[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]))
	}

	// Two-arg builtins
	if rtName, ok := twoArgBuiltins[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]))
	}

	// Special builtins
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
		if len(fc.Args) == 3 {
			return cg.callRT("sx_range3", cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]), cg.compileExpr(fc.Args[2]))
		}
		return cg.callRT("sx_range", cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]))
	case "input":
		if len(fc.Args) > 0 {
			return cg.callRT("sx_input", cg.compileExpr(fc.Args[0]))
		}
		return cg.callRT("sx_input", cg.callRT("sx_null"))
	}

	// Native bridge: __native_* → C runtime
	if rtName, ok := nativeNoArg[fc.Name]; ok {
		return cg.callRT(rtName)
	}
	if rtName, ok := nativeOneArg[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]))
	}
	if rtName, ok := nativeTwoArg[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]))
	}
	if rtName, ok := nativeThreeArg[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]), cg.compileExpr(fc.Args[2]))
	}
	if rtName, ok := nativeFourArg[fc.Name]; ok {
		return cg.callRT(rtName, cg.compileExpr(fc.Args[0]), cg.compileExpr(fc.Args[1]), cg.compileExpr(fc.Args[2]), cg.compileExpr(fc.Args[3]))
	}

	// User-defined function
	if fn, ok := cg.userFuncs[fc.Name]; ok {
		args := make([]llvmValue.Value, len(fc.Args))
		for i, arg := range fc.Args {
			args[i] = cg.compileExpr(arg)
		}
		return cg.block.NewCall(fn, args...)
	}

	// Variable that might be a lambda/function value — call via sx_call
	val := cg.getVar(fc.Name)
	argc := len(fc.Args)
	if argc > 0 {
		argArray := cg.block.NewAlloca(types.NewArray(uint64(argc), sxValuePtr))
		for i, arg := range fc.Args {
			compiled := cg.compileExpr(arg)
			ptr := cg.block.NewGetElementPtr(
				types.NewArray(uint64(argc), sxValuePtr),
				argArray,
				constant.NewInt(i64, 0),
				constant.NewInt(i64, int64(i)),
			)
			cg.block.NewStore(compiled, ptr)
		}
		argPtr := cg.block.NewBitCast(argArray, types.NewPointer(sxValuePtr))
		return cg.callRT("sx_call", val, argPtr, constant.NewInt(i32, int64(argc)))
	}
	return cg.callRT("sx_call", val, constant.NewNull(sxValuePtr), constant.NewInt(i32, 0))
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

// hasInterpolation checks if a string contains {identifier} patterns.
// Returns false for strings like {"key": "value"} where the content isn't a valid identifier.
func hasInterpolation(s string) bool {
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			end := strings.IndexByte(s[i:], '}')
			if end == -1 {
				return false
			}
			name := strings.TrimSpace(s[i+1 : i+end])
			if name != "" && isValidIdent(name) {
				return true
			}
			i += end + 1
		} else {
			i++
		}
	}
	return false
}

func isValidIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] != '_' && !(s[0] >= 'a' && s[0] <= 'z') && !(s[0] >= 'A' && s[0] <= 'Z') {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') {
			return false
		}
	}
	return true
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
			varName := strings.TrimSpace(s[i+1 : i+end])
			if varName == "" || !isValidIdent(varName) {
				// Not an interpolation — keep literal
				current += s[i : i+end+1]
				i += end + 1
				continue
			}
			if current != "" {
				parts = append(parts, cg.callRT("sx_string", cg.globalString(current)))
				current = ""
			}
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

func (cg *CodeGen) compileLambda(l *parser.Lambda) llvmValue.Value {
	cg.lambdaCounter++
	fnName := fmt.Sprintf("sx_lambda_%d", cg.lambdaCounter)

	prevFn := cg.fn
	prevBlock := cg.block
	prevVars := cg.vars
	prevScopes := cg.scopes

	// Lambda uses SxFnPtr signature: SxValue* fn(SxValue** args, int argc, SxValue** env)
	argsPtrType := types.NewPointer(sxValuePtr)
	envPtrType := types.NewPointer(sxValuePtr)
	fn := cg.mod.NewFunc(fnName, sxValuePtr,
		ir.NewParam("args", argsPtrType),
		ir.NewParam("argc", i32),
		ir.NewParam("env", envPtrType),
	)

	entry := fn.NewBlock("entry")
	cg.fn = fn
	cg.block = entry
	cg.vars = make(map[string]llvmValue.Value)
	cg.scopes = []map[string]llvmValue.Value{}
	cg.pushScope()

	// Extract params from args array
	for i, name := range l.Params {
		argPtr := entry.NewGetElementPtr(sxValuePtr, fn.Params[0], constant.NewInt(i32, int64(i)))
		argVal := entry.NewLoad(sxValuePtr, argPtr)
		alloca := entry.NewAlloca(sxValuePtr)
		entry.NewStore(argVal, alloca)
		cg.scopes[len(cg.scopes)-1][name] = alloca
	}

	bodyResult := cg.compileExpr(l.Body)
	cg.block.NewRet(bodyResult)

	cg.popScope()
	cg.fn = prevFn
	cg.block = prevBlock
	cg.vars = prevVars
	cg.scopes = prevScopes

	// Cast to i8* and wrap as SxValue function
	fnPtr := cg.block.NewBitCast(fn, sxValuePtr)
	return cg.callRT("sx_function", fnPtr)
}
