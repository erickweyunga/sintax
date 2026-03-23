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

// --- Runtime declarations ---

func (cg *CodeGen) declareRuntime() {
	cg.declareFunc("sx_number", sxValuePtr, types.Double)
	cg.declareFunc("sx_string", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_bool", sxValuePtr, i32)
	cg.declareFunc("sx_null", sxValuePtr)
	cg.declareFunc("sx_list_new", sxValuePtr)
	cg.declareFunc("sx_dict_new", sxValuePtr)
	cg.declareFunc("sx_add", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_sub", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_mul", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_div", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_mod", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_pow", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_eq", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_neq", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_gt", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_lt", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_gte", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_lte", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_not", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_truthy", i32, sxValuePtr)
	cg.declareFunc("sx_print", voidType, sxValuePtr)
	cg.declareFunc("sx_list_append", voidType, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_list_remove", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_index", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_index_set", voidType, sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_in", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_dict_keys", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_dict_values", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_dict_has", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_len", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_type", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_range", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_to_number", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_to_string", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_to_bool", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_check_type", voidType, sxValuePtr, i32, sxValuePtr)
	cg.declareFunc("sx_input", sxValuePtr, sxValuePtr)
}

func (cg *CodeGen) declareFunc(name string, retType types.Type, paramTypes ...types.Type) {
	params := make([]*ir.Param, len(paramTypes))
	for i, t := range paramTypes {
		params[i] = ir.NewParam(fmt.Sprintf("p%d", i), t)
	}
	cg.rtFuncs[name] = cg.mod.NewFunc(name, retType, params...)
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

func typeNameToTag(name string) int {
	switch name {
	case "num":
		return 1
	case "str":
		return 2
	case "bool":
		return 3
	case "list":
		return 4
	case "dict":
		return 5
	default:
		return 0
	}
}
