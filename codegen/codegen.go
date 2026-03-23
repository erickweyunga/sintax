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

// SxValue* is an opaque pointer in LLVM IR — we represent it as i8*
var sxValuePtr = types.I8Ptr
var i32 = types.I32
var i64 = types.I64
var voidType = types.Void

// CodeGen generates LLVM IR from a Sintax AST.
type CodeGen struct {
	mod    *ir.Module
	block  *ir.Block // current block
	fn     *ir.Func  // current function
	vars   map[string]*ir.InstAlloca
	scopes []map[string]*ir.InstAlloca

	// Runtime function declarations
	rtFuncs map[string]*ir.Func

	// String constants cache
	strConstants map[string]*ir.Global
	strCounter   int

	// User-defined functions lookup
	userFuncs map[string]*ir.Func

	// Loop context for break/continue
	loopExitBlocks     []*ir.Block // break jumps here
	loopContinueBlocks []*ir.Block // continue jumps here

	// Block name counter for unique names
	blockCounter int
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

// Generate compiles a program to LLVM IR and returns it as a string.
func (cg *CodeGen) Generate(program *parser.Program) string {
	// Create main function
	mainFn := cg.mod.NewFunc("main", i32)
	entry := mainFn.NewBlock("entry")
	cg.fn = mainFn
	cg.block = entry
	cg.pushScope()

	for _, stmt := range program.Statements {
		cg.compileStatement(stmt)
	}

	// Return 0
	if cg.block.Term == nil {
		cg.block.NewRet(constant.NewInt(i32, 0))
	}

	return cg.mod.String()
}

// --- Runtime declarations ---

func (cg *CodeGen) declareRuntime() {
	// Constructors
	cg.declareFunc("sx_number", sxValuePtr, types.Double)
	cg.declareFunc("sx_string", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_bool", sxValuePtr, i32)
	cg.declareFunc("sx_null", sxValuePtr)
	cg.declareFunc("sx_list_new", sxValuePtr)
	cg.declareFunc("sx_dict_new", sxValuePtr)

	// Arithmetic
	cg.declareFunc("sx_add", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_sub", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_mul", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_div", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_mod", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_pow", sxValuePtr, sxValuePtr, sxValuePtr)

	// Comparison
	cg.declareFunc("sx_eq", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_neq", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_gt", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_lt", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_gte", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_lte", sxValuePtr, sxValuePtr, sxValuePtr)

	// Logical
	cg.declareFunc("sx_not", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_truthy", i32, sxValuePtr)

	// Print
	cg.declareFunc("sx_print", voidType, sxValuePtr)

	// Collections
	cg.declareFunc("sx_list_append", voidType, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_list_remove", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_index", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_index_set", voidType, sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_in", sxValuePtr, sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_dict_keys", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_dict_values", sxValuePtr, sxValuePtr)
	cg.declareFunc("sx_dict_has", sxValuePtr, sxValuePtr, sxValuePtr)

	// Utilities
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
	fn := cg.mod.NewFunc(name, retType, params...)
	cg.rtFuncs[name] = fn
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
	// New variable — alloca in ENTRY block for mem2reg optimization
	alloca = cg.createEntryAlloca(name)
	cg.block.NewStore(val, alloca)
	if len(cg.scopes) > 0 {
		cg.scopes[len(cg.scopes)-1][name] = alloca
	}
	cg.vars[name] = alloca
}

// createEntryAlloca places alloca in the function's entry block,
// enabling LLVM's mem2reg pass to promote them to SSA registers.
func (cg *CodeGen) createEntryAlloca(name string) *ir.InstAlloca {
	entryBlock := cg.fn.Blocks[0]
	alloca := entryBlock.NewAlloca(sxValuePtr)
	alloca.SetName(name + ".ptr")
	return alloca
}

func (cg *CodeGen) getVar(name string) llvmValue.Value {
	alloca, ok := cg.resolveVar(name)
	if !ok {
		// This shouldn't happen if compiler works correctly
		return cg.callRT("sx_null")
	}
	return cg.block.NewLoad(sxValuePtr, alloca)
}

func (cg *CodeGen) resolveVar(name string) (*ir.InstAlloca, bool) {
	// Search scopes from innermost to outermost
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

// --- Runtime call helpers ---

func (cg *CodeGen) callRT(name string, args ...llvmValue.Value) llvmValue.Value {
	fn := cg.rtFuncs[name]
	return cg.block.NewCall(fn, args...)
}

func (cg *CodeGen) callRTVoid(name string, args ...llvmValue.Value) {
	fn := cg.rtFuncs[name]
	cg.block.NewCall(fn, args...)
}

// --- Statements ---

func (cg *CodeGen) compileStatement(stmt *parser.Statement) {
	// Don't emit into already-terminated blocks (e.g., after return)
	if cg.block.Term != nil {
		return
	}
	switch {
	case stmt.FuncDef != nil:
		cg.compileFuncDef(stmt.FuncDef)
	case stmt.IfStmt != nil:
		cg.compileIfStmt(stmt.IfStmt)
	case stmt.SwitchStmt != nil:
		cg.compileSwitchStmt(stmt.SwitchStmt)
	case stmt.WhileStmt != nil:
		cg.compileWhileStmt(stmt.WhileStmt)
	case stmt.ForStmt != nil:
		cg.compileForStmt(stmt.ForStmt)
	case stmt.PrintStmt != nil:
		val := cg.compileExpr(stmt.PrintStmt.Value)
		cg.callRTVoid("sx_print", val)
	case stmt.ReturnStmt != nil:
		val := cg.compileExpr(stmt.ReturnStmt.Value)
		cg.block.NewRet(val)
	case stmt.TypedAssign != nil:
		cg.compileTypedAssign(stmt.TypedAssign)
	case stmt.IndexAssign != nil:
		cg.compileIndexAssign(stmt.IndexAssign)
	case stmt.CompoundAssign != nil:
		cg.compileCompoundAssign(stmt.CompoundAssign)
	case stmt.Assignment != nil:
		val := cg.compileExpr(stmt.Assignment.Value)
		cg.setVar(stmt.Assignment.Name, val)
	case stmt.ExprStmt != nil:
		// Check for break (0) and continue (1) in loop context
		if cg.isBareLiteral(stmt.ExprStmt.Expr, 0) && len(cg.loopExitBlocks) > 0 {
			cg.block.NewBr(cg.loopExitBlocks[len(cg.loopExitBlocks)-1])
			// Create a dead block for subsequent statements
			cg.block = cg.newBlock("after.break")
			return
		}
		if cg.isBareLiteral(stmt.ExprStmt.Expr, 1) && len(cg.loopContinueBlocks) > 0 {
			cg.block.NewBr(cg.loopContinueBlocks[len(cg.loopContinueBlocks)-1])
			cg.block = cg.newBlock("after.continue")
			return
		}
		cg.compileExpr(stmt.ExprStmt.Expr)
	}
}

func (cg *CodeGen) compileFuncDef(fd *parser.FuncDef) {
	// Save current state
	prevFn := cg.fn
	prevBlock := cg.block

	// Create function: all params and return are SxValue*
	params := make([]*ir.Param, len(fd.Params))
	for i, p := range fd.Params {
		params[i] = ir.NewParam(p.Name, sxValuePtr)
	}
	fn := cg.mod.NewFunc("sx_user_"+fd.Name, sxValuePtr, params...)
	cg.userFuncs[fd.Name] = fn

	entry := fn.NewBlock("entry")
	cg.fn = fn
	cg.block = entry
	cg.pushScope()

	// Allocate params as local variables
	for i, p := range fd.Params {
		alloca := entry.NewAlloca(sxValuePtr)
		entry.NewStore(fn.Params[i], alloca)
		cg.scopes[len(cg.scopes)-1][p.Name] = alloca
	}

	// Compile body
	for _, stmt := range fd.Body.Statements {
		cg.compileStatement(stmt)
	}

	// Implicit return null if no return
	if cg.block.Term == nil {
		cg.block.NewRet(cg.callRT("sx_null"))
	}

	cg.popScope()

	// Restore state
	cg.fn = prevFn
	cg.block = prevBlock

	// Store function pointer as a variable (for calling)
	cg.vars["__fn_"+fd.Name] = nil // marker
	// We'll handle function calls by name lookup in compileFuncCall
	cg.setVar(fd.Name, constant.NewNull(sxValuePtr)) // placeholder
	// Store the actual function reference
	cg.vars["__fnref_"+fd.Name] = nil
	_ = fn // keep reference
}

func (cg *CodeGen) compileIfStmt(ifStmt *parser.IfStmt) {
	cond := cg.compileExpr(ifStmt.Condition)
	condBool := cg.callRT("sx_truthy", cond)
	condI1 := cg.block.NewICmp(enum.IPredNE, condBool, constant.NewInt(i32, 0))

	thenBlock := cg.newBlock("then")
	elseBlock := cg.newBlock("else")
	mergeBlock := cg.newBlock("merge")

	cg.block.NewCondBr(condI1, thenBlock, elseBlock)

	// Then
	cg.block = thenBlock
	for _, stmt := range ifStmt.Body.Statements {
		cg.compileStatement(stmt)
	}
	if cg.block.Term == nil {
		cg.block.NewBr(mergeBlock)
	}

	// Else
	cg.block = elseBlock
	if ifStmt.Else != nil {
		for _, stmt := range ifStmt.Else.Statements {
			cg.compileStatement(stmt)
		}
	}
	if cg.block.Term == nil {
		cg.block.NewBr(mergeBlock)
	}

	cg.block = mergeBlock
}

func (cg *CodeGen) compileSwitchStmt(sw *parser.SwitchStmt) {
	mergeBlock := cg.newBlock("switch.merge")

	for i, cas := range sw.Cases {
		val := cg.compileExpr(sw.Value)
		caseVal := cg.compileExpr(cas.Value)
		eq := cg.callRT("sx_eq", val, caseVal)
		eqBool := cg.callRT("sx_truthy", eq)
		condI1 := cg.block.NewICmp(enum.IPredNE, eqBool, constant.NewInt(i32, 0))

		bodyBlock := cg.newBlock(fmt.Sprintf("case.%d", i))
		nextBlock := cg.newBlock(fmt.Sprintf("case.next.%d", i))

		cg.block.NewCondBr(condI1, bodyBlock, nextBlock)

		cg.block = bodyBlock
		for _, stmt := range cas.Body.Statements {
			cg.compileStatement(stmt)
		}
		if cg.block.Term == nil {
			cg.block.NewBr(mergeBlock)
		}

		cg.block = nextBlock
	}

	// Default
	if sw.Default != nil {
		for _, stmt := range sw.Default.Statements {
			cg.compileStatement(stmt)
		}
	}
	if cg.block.Term == nil {
		cg.block.NewBr(mergeBlock)
	}

	cg.block = mergeBlock
}

func (cg *CodeGen) compileWhileStmt(ws *parser.WhileStmt) {
	condBlock := cg.newBlock("while.cond")
	bodyBlock := cg.newBlock("while.body")
	exitBlock := cg.newBlock("while.exit")

	// Push loop context (continue → condBlock, break → exitBlock)
	cg.loopContinueBlocks = append(cg.loopContinueBlocks, condBlock)
	cg.loopExitBlocks = append(cg.loopExitBlocks, exitBlock)

	cg.block.NewBr(condBlock)

	// Condition
	cg.block = condBlock
	cond := cg.compileExpr(ws.Condition)
	condBool := cg.callRT("sx_truthy", cond)
	condI1 := cg.block.NewICmp(enum.IPredNE, condBool, constant.NewInt(i32, 0))
	cg.block.NewCondBr(condI1, bodyBlock, exitBlock)

	// Body
	cg.block = bodyBlock
	for _, stmt := range ws.Body.Statements {
		cg.compileStatement(stmt)
	}
	if cg.block.Term == nil {
		cg.block.NewBr(condBlock)
	}

	// Pop loop context
	cg.loopContinueBlocks = cg.loopContinueBlocks[:len(cg.loopContinueBlocks)-1]
	cg.loopExitBlocks = cg.loopExitBlocks[:len(cg.loopExitBlocks)-1]

	cg.block = exitBlock
}

func (cg *CodeGen) compileForStmt(fs *parser.ForStmt) {
	iter := cg.compileExpr(fs.Iter)
	iterLen := cg.callRT("sx_len", iter)

	// Index variable
	idxAlloca := cg.block.NewAlloca(sxValuePtr)
	cg.block.NewStore(cg.callRT("sx_number", constant.NewFloat(types.Double, 0)), idxAlloca)

	condBlock := cg.newBlock("for.cond")
	bodyBlock := cg.newBlock("for.body")
	incrBlock := cg.newBlock("for.incr")
	exitBlock := cg.newBlock("for.exit")

	// Push loop context (continue → incrBlock, break → exitBlock)
	cg.loopContinueBlocks = append(cg.loopContinueBlocks, incrBlock)
	cg.loopExitBlocks = append(cg.loopExitBlocks, exitBlock)

	cg.block.NewBr(condBlock)

	// Condition: idx < len
	cg.block = condBlock
	idx := cg.block.NewLoad(sxValuePtr, idxAlloca)
	cond := cg.callRT("sx_lt", idx, iterLen)
	condBool := cg.callRT("sx_truthy", cond)
	condI1 := cg.block.NewICmp(enum.IPredNE, condBool, constant.NewInt(i32, 0))
	cg.block.NewCondBr(condI1, bodyBlock, exitBlock)

	// Body
	cg.block = bodyBlock
	idx2 := cg.block.NewLoad(sxValuePtr, idxAlloca)
	elem := cg.callRT("sx_index", iter, idx2)
	cg.setVar(fs.Var, elem)

	for _, stmt := range fs.Body.Statements {
		cg.compileStatement(stmt)
	}
	if cg.block.Term == nil {
		cg.block.NewBr(incrBlock)
	}

	// Increment
	cg.block = incrBlock
	idx3 := cg.block.NewLoad(sxValuePtr, idxAlloca)
	one := cg.callRT("sx_number", constant.NewFloat(types.Double, 1))
	newIdx := cg.callRT("sx_add", idx3, one)
	cg.block.NewStore(newIdx, idxAlloca)
	cg.block.NewBr(condBlock)

	// Pop loop context
	cg.loopContinueBlocks = cg.loopContinueBlocks[:len(cg.loopContinueBlocks)-1]
	cg.loopExitBlocks = cg.loopExitBlocks[:len(cg.loopExitBlocks)-1]

	cg.block = exitBlock
}

func (cg *CodeGen) compileTypedAssign(ta *parser.TypedAssign) {
	val := cg.compileExpr(ta.Value)
	typeTag := typeNameToTag(ta.Type)
	nameStr := cg.globalString(ta.Name)
	cg.callRTVoid("sx_check_type", val, constant.NewInt(i32, int64(typeTag)), nameStr)
	cg.setVar(ta.Name, val)
}

func (cg *CodeGen) compileIndexAssign(ia *parser.IndexAssign) {
	collection := cg.getVar(ia.Name)
	idx := cg.compileExpr(ia.Index)
	val := cg.compileExpr(ia.Value)
	cg.callRTVoid("sx_index_set", collection, idx, val)
}

func (cg *CodeGen) compileCompoundAssign(ca *parser.CompoundAssign) {
	current := cg.getVar(ca.Name)
	right := cg.compileExpr(ca.Value)
	var result llvmValue.Value
	switch ca.Op {
	case "+=":
		result = cg.callRT("sx_add", current, right)
	case "-=":
		result = cg.callRT("sx_sub", current, right)
	case "*=":
		result = cg.callRT("sx_mul", current, right)
	case "/=":
		result = cg.callRT("sx_div", current, right)
	}
	cg.setVar(ca.Name, result)
}

// --- Expressions ---

func (cg *CodeGen) compileExpr(expr *parser.Expr) llvmValue.Value {
	left := cg.compileLogicalAnd(expr.Left)
	for _, op := range expr.Ops {
		// OR: short-circuit
		right := cg.compileLogicalAnd(op.Right)
		leftTruthy := cg.callRT("sx_truthy", left)
		cond := cg.block.NewICmp(enum.IPredNE, leftTruthy, constant.NewInt(i32, 0))

		thenBlock := cg.newBlock("or.left")
		elseBlock := cg.newBlock("or.right")
		mergeBlock := cg.newBlock("or.merge")

		cg.block.NewCondBr(cond, thenBlock, elseBlock)

		cg.block = thenBlock
		cg.block.NewBr(mergeBlock)

		cg.block = elseBlock
		cg.block.NewBr(mergeBlock)

		cg.block = mergeBlock
		phi := cg.block.NewPhi(ir.NewIncoming(left, thenBlock), ir.NewIncoming(right, elseBlock))
		left = phi
	}
	return left
}

func (cg *CodeGen) compileLogicalAnd(and *parser.LogicalAnd) llvmValue.Value {
	left := cg.compileComparison(and.Left)
	for _, op := range and.Ops {
		right := cg.compileComparison(op.Right)
		leftTruthy := cg.callRT("sx_truthy", left)
		cond := cg.block.NewICmp(enum.IPredNE, leftTruthy, constant.NewInt(i32, 0))

		thenBlock := cg.newBlock("and.right")
		mergeBlock := cg.newBlock("and.merge")

		falseBlock := cg.newBlock("and.false")

		cg.block.NewCondBr(cond, thenBlock, falseBlock)

		cg.block = falseBlock
		cg.block.NewBr(mergeBlock)

		cg.block = thenBlock
		cg.block.NewBr(mergeBlock)

		cg.block = mergeBlock
		phi := cg.block.NewPhi(ir.NewIncoming(left, falseBlock), ir.NewIncoming(right, thenBlock))
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
		case "ktk":
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
		// String interpolation
		if strings.Contains(s, "{") && strings.Contains(s, "}") {
			return cg.compileInterpolatedString(s)
		}
		str := cg.globalString(s)
		return cg.callRT("sx_string", str)

	case p.Ident != nil:
		name := *p.Ident
		switch name {
		case "kweli":
			return cg.callRT("sx_bool", constant.NewInt(i32, 1))
		case "sikweli":
			return cg.callRT("sx_bool", constant.NewInt(i32, 0))
		case "tupu":
			return cg.callRT("sx_null")
		default:
			return cg.getVar(name)
		}

	case p.SubExpr != nil:
		return cg.compileExpr(p.SubExpr)
	}

	return cg.callRT("sx_null")
}

func (cg *CodeGen) compileFuncCall(fc *parser.FuncCall) llvmValue.Value {
	// Built-in functions
	switch fc.Name {
	case "andika":
		if len(fc.Args) == 1 {
			val := cg.compileExpr(fc.Args[0])
			cg.callRTVoid("sx_print", val)
			return cg.callRT("sx_null")
		}
		// Multi-arg: concatenate with spaces then print
		parts := make([]llvmValue.Value, 0)
		for _, arg := range fc.Args {
			val := cg.compileExpr(arg)
			parts = append(parts, cg.callRT("sx_to_string", val))
		}
		result := parts[0]
		for _, part := range parts[1:] {
			spaceStr := cg.globalString(" ")
			space := cg.callRT("sx_string", spaceStr)
			result = cg.callRT("sx_add", result, space)
			result = cg.callRT("sx_add", result, part)
		}
		cg.callRTVoid("sx_print", result)
		return cg.callRT("sx_null")
	case "aina":
		val := cg.compileExpr(fc.Args[0])
		return cg.callRT("sx_type", val)
	case "urefu":
		val := cg.compileExpr(fc.Args[0])
		return cg.callRT("sx_len", val)
	case "ongeza":
		list := cg.compileExpr(fc.Args[0])
		item := cg.compileExpr(fc.Args[1])
		cg.callRTVoid("sx_list_append", list, item)
		return list
	case "ondoa":
		list := cg.compileExpr(fc.Args[0])
		idx := cg.compileExpr(fc.Args[1])
		return cg.callRT("sx_list_remove", list, idx)
	case "masafa":
		if len(fc.Args) == 1 {
			end := cg.compileExpr(fc.Args[0])
			start := cg.callRT("sx_number", constant.NewFloat(types.Double, 0))
			return cg.callRT("sx_range", start, end)
		}
		start := cg.compileExpr(fc.Args[0])
		end := cg.compileExpr(fc.Args[1])
		return cg.callRT("sx_range", start, end)
	case "funguo":
		val := cg.compileExpr(fc.Args[0])
		return cg.callRT("sx_dict_keys", val)
	case "thamani":
		val := cg.compileExpr(fc.Args[0])
		return cg.callRT("sx_dict_values", val)
	case "ina":
		dict := cg.compileExpr(fc.Args[0])
		key := cg.compileExpr(fc.Args[1])
		return cg.callRT("sx_dict_has", dict, key)
	case "nambari":
		val := cg.compileExpr(fc.Args[0])
		return cg.callRT("sx_to_number", val)
	case "tungo":
		val := cg.compileExpr(fc.Args[0])
		return cg.callRT("sx_to_string", val)
	case "buliani":
		val := cg.compileExpr(fc.Args[0])
		return cg.callRT("sx_to_bool", val)
	case "soma":
		if len(fc.Args) > 0 {
			prompt := cg.compileExpr(fc.Args[0])
			return cg.callRT("sx_input", prompt)
		}
		return cg.callRT("sx_input", cg.callRT("sx_null"))
	}

	// User-defined function call
	if fn, ok := cg.userFuncs[fc.Name]; ok {
		args := make([]llvmValue.Value, len(fc.Args))
		for i, arg := range fc.Args {
			args[i] = cg.compileExpr(arg)
		}
		return cg.block.NewCall(fn, args...)
	}

	return cg.callRT("sx_null")
}

func (cg *CodeGen) compileInterpolatedString(s string) llvmValue.Value {
	// Build the string by concatenating parts
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
				str := cg.globalString(current)
				parts = append(parts, cg.callRT("sx_string", str))
				current = ""
			}
			varName := strings.TrimSpace(s[i+1 : i+end])
			varVal := cg.getVar(varName)
			parts = append(parts, cg.callRT("sx_to_string", varVal))
			i += end + 1
		} else {
			current += string(s[i])
			i++
		}
	}
	if current != "" {
		str := cg.globalString(current)
		parts = append(parts, cg.callRT("sx_string", str))
	}

	// Concatenate all parts
	if len(parts) == 0 {
		str := cg.globalString("")
		return cg.callRT("sx_string", str)
	}
	result := parts[0]
	for _, part := range parts[1:] {
		result = cg.callRT("sx_add", result, part)
	}
	return result
}

func (cg *CodeGen) newBlock(prefix string) *ir.Block {
	cg.blockCounter++
	return cg.fn.NewBlock(fmt.Sprintf("%s.%d", prefix, cg.blockCounter))
}

func (cg *CodeGen) isBareLiteral(expr *parser.Expr, val float64) bool {
	if expr.Left == nil || len(expr.Ops) > 0 {
		return false
	}
	and := expr.Left
	if and.Left == nil || len(and.Ops) > 0 {
		return false
	}
	cmp := and.Left
	if cmp.Op != "" || cmp.Left == nil {
		return false
	}
	add := cmp.Left
	if len(add.Ops) > 0 || add.Left == nil {
		return false
	}
	mul := add.Left
	if len(mul.Ops) > 0 || mul.Left == nil {
		return false
	}
	u := mul.Left
	if u.Not != nil || u.Primary == nil {
		return false
	}
	p := u.Primary
	return p.Number != nil && *p.Number == val
}

func typeNameToTag(name string) int {
	switch name {
	case "nambari":
		return 1 // SX_NUMBER
	case "tungo":
		return 2 // SX_STRING
	case "buliani":
		return 3 // SX_BOOL
	case "safu":
		return 4 // SX_LIST
	case "kamusi":
		return 5 // SX_DICT
	default:
		return 0
	}
}
