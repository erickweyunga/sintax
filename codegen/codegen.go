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
	vars   map[string]*ir.InstAlloca
	scopes []map[string]*ir.InstAlloca

	rtFuncs      map[string]*ir.Func
	strConstants map[string]*ir.Global
	strCounter   int
	userFuncs    map[string]*ir.Func

	loopExitBlocks     []*ir.Block
	loopContinueBlocks []*ir.Block
	blockCounter       int
}

// New creates a new code generator.
func New() *CodeGen {
	cg := &CodeGen{
		mod:          ir.NewModule(),
		vars:         make(map[string]*ir.InstAlloca),
		scopes:       []map[string]*ir.InstAlloca{},
		rtFuncs:      make(map[string]*ir.Func),
		strConstants: make(map[string]*ir.Global),
		userFuncs:    make(map[string]*ir.Func),
	}
	cg.declareRuntime()
	// No target triple — let clang use the host default
	return cg
}

// Generate compiles a program to LLVM IR.
func (cg *CodeGen) Generate(program *parser.Program) string {
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
	cg.scopes = append(cg.scopes, make(map[string]*ir.InstAlloca))
}

func (cg *CodeGen) popScope() {
	cg.scopes = cg.scopes[:len(cg.scopes)-1]
}

func (cg *CodeGen) setVar(name string, val llvmValue.Value) {
	alloca, ok := cg.resolveVar(name)
	if ok {
		cg.block.NewStore(val, alloca)
		return
	}
	alloca = cg.createEntryAlloca(name)
	cg.block.NewStore(val, alloca)
	if len(cg.scopes) > 0 {
		cg.scopes[len(cg.scopes)-1][name] = alloca
	}
	cg.vars[name] = alloca
}

func (cg *CodeGen) createEntryAlloca(name string) *ir.InstAlloca {
	entryBlock := cg.fn.Blocks[0]
	alloca := entryBlock.NewAlloca(sxValuePtr)
	alloca.SetName(name + ".ptr")
	return alloca
}

func (cg *CodeGen) getVar(name string) llvmValue.Value {
	alloca, ok := cg.resolveVar(name)
	if !ok {
		return cg.callRT("sx_null")
	}
	return cg.block.NewLoad(sxValuePtr, alloca)
}

func (cg *CodeGen) resolveVar(name string) (*ir.InstAlloca, bool) {
	for i := len(cg.scopes) - 1; i >= 0; i-- {
		if alloca, ok := cg.scopes[i][name]; ok {
			return alloca, true
		}
	}
	if alloca, ok := cg.vars[name]; ok {
		return alloca, true
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

func typeNameToTag(name string) int {
	if tag, ok := typeTagMap[name]; ok {
		return tag
	}
	return 0
}
