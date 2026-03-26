package codegen

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	llvmValue "github.com/llir/llvm/ir/value"

	"github.com/erickweyunga/sintax/parser"
)

// SxValue* is an opaque pointer in LLVM IR — we represent it as i8*
var sxValuePtr = types.I8Ptr
var i32 = types.I32
var i64 = types.I64
var voidType = types.Void

// CodeGen generates LLVM IR from a Sintax AST.
type CodeGen struct {
	mod    *ir.Module
	block  *ir.Block
	fn     *ir.Func
	vars   map[string]llvmValue.Value // name → pointer to SxValue* (alloca or heap cell)
	scopes []map[string]llvmValue.Value

	rtFuncs      map[string]*ir.Func
	strConstants map[string]*ir.Global
	strCounter   int
	userFuncs    map[string]*ir.Func

	loopExitBlocks     []*ir.Block
	loopContinueBlocks []*ir.Block
	blockCounter       int
	lambdaCounter      int
}

// New creates a new code generator.
func New() *CodeGen {
	cg := &CodeGen{
		mod:          ir.NewModule(),
		vars:         make(map[string]llvmValue.Value),
		scopes:       []map[string]llvmValue.Value{},
		rtFuncs:      make(map[string]*ir.Func),
		strConstants: make(map[string]*ir.Global),
		userFuncs:    make(map[string]*ir.Func),
	}
	cg.declareRuntime()
	return cg
}

// Generate compiles a program to LLVM IR.
func (cg *CodeGen) Generate(program *parser.Program) string {
	// Pass 1: Forward-declare top-level functions so mutual recursion works
	for _, stmt := range program.Statements {
		if stmt.FuncDef != nil {
			cg.forwardDeclare(stmt.FuncDef)
		}
	}

	// Pass 2: Compile everything
	mainFn := cg.mod.NewFunc("main", i32)
	entry := mainFn.NewBlock("entry")
	cg.fn = mainFn
	cg.block = entry
	cg.pushScope()

	for _, stmt := range program.Statements {
		cg.compileStatement(stmt)
	}

	if cg.block.Term == nil {
		cg.block.NewRet(constant.NewInt(i32, 0))
	}

	return cg.mod.String()
}

// CompileModule compiles an imported module's function definitions.
// The prefix is prepended to function names (e.g. "math" → "math__sqrt").
// For specific function imports, functions are also registered without prefix.
func (cg *CodeGen) CompileModule(program *parser.Program, prefix string, specificFunc string) {
	for _, stmt := range program.Statements {
		if stmt.FuncDef != nil {
			prefixed := *stmt.FuncDef
			prefixed.Name = prefix + "__" + stmt.FuncDef.Name
			cg.forwardDeclare(&prefixed)
			if specificFunc != "" && stmt.FuncDef.Name == specificFunc {
				cg.forwardDeclare(stmt.FuncDef)
			}
		}
	}
	for _, stmt := range program.Statements {
		if stmt.FuncDef != nil {
			prefixed := *stmt.FuncDef
			prefixed.Name = prefix + "__" + stmt.FuncDef.Name
			cg.compileFuncDef(&prefixed)
			if specificFunc != "" && stmt.FuncDef.Name == specificFunc {
				cg.compileFuncDef(stmt.FuncDef)
			}
		}
	}
}

// --- Runtime declarations (table-driven) ---

type rtDecl struct {
	name   string
	ret    types.Type
	params []types.Type
}

// V = SxValue*, D = double, I = i32, void = void
var runtimeDecls = []rtDecl{
	// Constructors
	{"sx_number", sxValuePtr, []types.Type{types.Double}},
	{"sx_string", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_bool", sxValuePtr, []types.Type{i32}},
	{"sx_null", sxValuePtr, nil},
	{"sx_list_new", sxValuePtr, nil},
	{"sx_dict_new", sxValuePtr, nil},
	// Arithmetic
	{"sx_add", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_sub", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_mul", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_div", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_mod", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_pow", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	// Comparison
	{"sx_eq", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_neq", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_gt", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_lt", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_gte", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_lte", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	// Logical
	{"sx_not", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_truthy", i32, []types.Type{sxValuePtr}},
	// I/O
	{"sx_print", voidType, []types.Type{sxValuePtr}},
	{"sx_input", sxValuePtr, []types.Type{sxValuePtr}},
	// Collections
	{"sx_list_append", voidType, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_list_remove", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_index", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_index_set", voidType, []types.Type{sxValuePtr, sxValuePtr, sxValuePtr}},
	{"sx_in", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_dict_keys", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_dict_values", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_dict_has", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	// Utilities
	{"sx_len", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_type", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_range", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"sx_to_number", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_to_string", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_to_bool", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_check_type", voidType, []types.Type{sxValuePtr, i32, sxValuePtr}},
	{"sx_error", voidType, []types.Type{sxValuePtr}},
	// Native bridge functions (__native_*)
	{"__native_sqrt", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_sin", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_cos", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_tan", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_asin", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_acos", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_atan", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_log", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_log2", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_log10", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_exp", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_floor", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_ceil", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_round", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_cbrt", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_pow", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_random", sxValuePtr, nil},
	{"__native_upper", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_lower", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_split", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_replace", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr, sxValuePtr}},
	{"__native_read_file", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_write_file", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_file_exists", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_delete_file", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_cwd", sxValuePtr, nil},
	{"__native_getenv", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_exec", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_time", sxValuePtr, nil},
	// String building blocks
	{"__native_trim", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_char_code", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_from_char_code", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_str_reverse", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_str_repeat", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_index_of", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_slice", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr, sxValuePtr}},
	// List building blocks
	{"__native_list_concat", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_list_insert", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr, sxValuePtr}},
	{"__native_list_reverse", sxValuePtr, []types.Type{sxValuePtr}},
	// Dict building blocks
	{"__native_dict_delete", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_dict_merge", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	// System building blocks
	{"__native_sleep", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_exit", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_format_time", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	{"__native_rename", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr}},
	// JSON
	{"__native_json_parse", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_json_stringify", sxValuePtr, []types.Type{sxValuePtr}},
	{"__native_json_pretty", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_function", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_closure", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr, i32}},
	{"sx_alloc_env", sxValuePtr, []types.Type{i64}},
	{"sx_error_new", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_call", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr, i32}},
	{"sx_is_error", sxValuePtr, []types.Type{sxValuePtr}},
	{"sx_method", sxValuePtr, []types.Type{sxValuePtr, sxValuePtr, sxValuePtr, i32}},
}

func (cg *CodeGen) declareRuntime() {
	for _, d := range runtimeDecls {
		params := make([]*ir.Param, len(d.params))
		for i, t := range d.params {
			params[i] = ir.NewParam(fmt.Sprintf("p%d", i), t)
		}
		cg.rtFuncs[d.name] = cg.mod.NewFunc(d.name, d.ret, params...)
	}
}

// --- Scope ---

func (cg *CodeGen) pushScope() {
	cg.scopes = append(cg.scopes, make(map[string]llvmValue.Value))
}

func (cg *CodeGen) popScope() {
	cg.scopes = cg.scopes[:len(cg.scopes)-1]
}

func (cg *CodeGen) setVar(name string, val llvmValue.Value) {
	ptr, ok := cg.resolveVar(name)
	if ok {
		cg.block.NewStore(val, ptr)
		return
	}
	// New variable — allocate on the stack
	alloca := cg.block.NewAlloca(sxValuePtr)
	alloca.SetName(name + ".ptr")
	cg.block.NewStore(val, alloca)
	if len(cg.scopes) > 0 {
		cg.scopes[len(cg.scopes)-1][name] = alloca
	}
	cg.vars[name] = alloca
}

// setVarPtr overrides a variable's storage location to point at an
// existing pointer (alloca or heap cell). Used for closure-captured variables.
func (cg *CodeGen) setVarPtr(name string, ptr llvmValue.Value) {
	if len(cg.scopes) > 0 {
		cg.scopes[len(cg.scopes)-1][name] = ptr
	}
	cg.vars[name] = ptr
}

func (cg *CodeGen) getVar(name string) llvmValue.Value {
	ptr, ok := cg.resolveVar(name)
	if !ok {
		return cg.callRT("sx_null")
	}
	return cg.block.NewLoad(sxValuePtr, ptr)
}

func (cg *CodeGen) resolveVar(name string) (llvmValue.Value, bool) {
	for i := len(cg.scopes) - 1; i >= 0; i-- {
		if ptr, ok := cg.scopes[i][name]; ok {
			return ptr, true
		}
	}
	if ptr, ok := cg.vars[name]; ok {
		return ptr, true
	}
	return nil, false
}

// --- String constants ---

func (cg *CodeGen) globalString(s string) llvmValue.Value {
	if g, ok := cg.strConstants[s]; ok {
		zero := constant.NewInt(i64, 0)
		return cg.block.NewGetElementPtr(
			types.NewArray(uint64(len(s)+1), types.I8),
			g, zero, zero,
		)
	}

	name := fmt.Sprintf(".str.%d", cg.strCounter)
	cg.strCounter++

	g := cg.mod.NewGlobalDef(name, constant.NewCharArrayFromString(s+"\x00"))
	g.Immutable = true
	cg.strConstants[s] = g

	zero := constant.NewInt(i64, 0)
	return cg.block.NewGetElementPtr(
		types.NewArray(uint64(len(s)+1), types.I8),
		g, zero, zero,
	)
}

// --- Helpers ---

func (cg *CodeGen) callRT(name string, args ...llvmValue.Value) llvmValue.Value {
	return cg.block.NewCall(cg.rtFuncs[name], args...)
}

func (cg *CodeGen) callRTVoid(name string, args ...llvmValue.Value) {
	cg.block.NewCall(cg.rtFuncs[name], args...)
}

func (cg *CodeGen) newBlock(prefix string) *ir.Block {
	cg.blockCounter++
	return cg.fn.NewBlock(fmt.Sprintf("%s.%d", prefix, cg.blockCounter))
}

var typeTagMap = map[string]int{
	"num":  1, // SX_NUMBER
	"str":  2, // SX_STRING
	"bool": 3, // SX_BOOL
	"list": 4, // SX_LIST
	"dict": 5, // SX_DICT
}

func (cg *CodeGen) forwardDeclare(fd *parser.FuncDef) {
	if _, exists := cg.userFuncs[fd.Name]; exists {
		return
	}
	params := make([]*ir.Param, len(fd.Params))
	for i, p := range fd.Params {
		params[i] = ir.NewParam(p.Name, sxValuePtr)
	}
	fn := cg.mod.NewFunc("sx_user_"+fd.Name, sxValuePtr, params...)
	cg.userFuncs[fd.Name] = fn
}

func (cg *CodeGen) emitError(msg string) {
	str := cg.globalString(msg)
	cg.callRTVoid("sx_error", str)
}

func typeNameToTag(name string) int {
	if tag, ok := typeTagMap[name]; ok {
		return tag
	}
	return 0
}
