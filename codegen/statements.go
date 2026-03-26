package codegen

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	llvmValue "github.com/llir/llvm/ir/value"

	"github.com/erickweyunga/sintax/parser"
)

func (cg *CodeGen) compileStatement(stmt *parser.Statement) {
	if cg.block.Term != nil {
		return
	}
	switch {
	case stmt.FuncDef != nil:
		cg.compileFuncDef(stmt.FuncDef)
	case stmt.IfStmt != nil:
		cg.compileIfStmt(stmt.IfStmt)
	case stmt.CatchStmt != nil:
		cg.compileCatchStmt(stmt.CatchStmt)
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
		if stmt.ExprStmt.Expr.IsBareLiteral(0) && len(cg.loopExitBlocks) > 0 {
			cg.block.NewBr(cg.loopExitBlocks[len(cg.loopExitBlocks)-1])
			cg.block = cg.newBlock("after.break")
			return
		}
		if stmt.ExprStmt.Expr.IsBareLiteral(1) && len(cg.loopContinueBlocks) > 0 {
			cg.block.NewBr(cg.loopContinueBlocks[len(cg.loopContinueBlocks)-1])
			cg.block = cg.newBlock("after.continue")
			return
		}
		cg.compileExpr(stmt.ExprStmt.Expr)
	}
}

func (cg *CodeGen) compileFuncDef(fd *parser.FuncDef) {
	isNested := cg.fn != nil && cg.fn.Name() != "main"

	if isNested {
		cg.compileNestedFuncDef(fd)
		return
	}

	prevFn := cg.fn
	prevBlock := cg.block
	prevVars := cg.vars

	fn, exists := cg.userFuncs[fd.Name]
	if !exists {
		cg.forwardDeclare(fd)
		fn = cg.userFuncs[fd.Name]
	}

	entry := fn.NewBlock("entry")
	cg.fn = fn
	cg.block = entry
	cg.vars = make(map[string]llvmValue.Value)
	prevScopes := cg.scopes
	cg.scopes = []map[string]llvmValue.Value{}
	cg.pushScope()

	for i, p := range fd.Params {
		alloca := entry.NewAlloca(sxValuePtr)
		entry.NewStore(fn.Params[i], alloca)
		cg.scopes[len(cg.scopes)-1][p.Name] = alloca
	}

	cg.compileFuncBody(fd.Body.Statements)

	cg.popScope()
	cg.fn = prevFn
	cg.block = prevBlock
	cg.vars = prevVars
	cg.scopes = prevScopes
}

// compileNestedFuncDef compiles a function defined inside another function
// as a closure. Captured variables are stored in a heap-allocated environment.
func (cg *CodeGen) compileNestedFuncDef(fd *parser.FuncDef) {
	captures := cg.findCaptures(fd)
	envSize := len(captures)

	// 1. For each captured variable, allocate a heap cell (SxValue**)
	//    so both parent and closure read/write through the same pointer.
	var envSlots []llvmValue.Value
	for _, name := range captures {
		varPtr, _ := cg.resolveVar(name)

		// Check if this variable is already a heap cell (from a previous
		// closure in the same scope). If so, reuse it.
		// A heap cell is a bitcast result; a stack alloca is an InstAlloca.
		if _, isAlloca := varPtr.(*ir.InstAlloca); isAlloca {
			// First time capturing — promote from stack to heap cell
			cell := cg.block.NewCall(cg.rtFuncs["sx_alloc_env"], constant.NewInt(i64, 8))
			cellTyped := cg.block.NewBitCast(cell, types.NewPointer(sxValuePtr))

			currentVal := cg.getVar(name)
			cg.block.NewStore(currentVal, cellTyped)

			cg.setVarPtr(name, cellTyped)
			envSlots = append(envSlots, cg.block.NewBitCast(cellTyped, sxValuePtr))
		} else {
			// Already a heap cell — reuse it
			envSlots = append(envSlots, cg.block.NewBitCast(varPtr, sxValuePtr))
		}
	}

	// 2. Build env array on the stack: SxValue* env[n] = {cell0, cell1, ...}
	//    Then heap-copy it so it outlives this stack frame.
	var envArray llvmValue.Value
	if envSize > 0 {
		// Allocate heap env array
		heapEnv := cg.block.NewCall(cg.rtFuncs["sx_alloc_env"],
			constant.NewInt(i64, int64(envSize*8)))
		heapEnvTyped := cg.block.NewBitCast(heapEnv, types.NewPointer(sxValuePtr))
		for i, slot := range envSlots {
			ptr := cg.block.NewGetElementPtr(sxValuePtr, heapEnvTyped, constant.NewInt(i32, int64(i)))
			cg.block.NewStore(slot, ptr)
		}
		envArray = cg.block.NewBitCast(heapEnvTyped, sxValuePtr)
	} else {
		envArray = constant.NewNull(sxValuePtr)
	}

	// 3. Compile the closure function body
	cg.lambdaCounter++
	closureName := fmt.Sprintf("sx_closure_%s_%d", fd.Name, cg.lambdaCounter)

	prevFn := cg.fn
	prevBlock := cg.block
	prevVars := cg.vars
	prevScopes := cg.scopes

	argsPtrType := types.NewPointer(sxValuePtr)
	envPtrType := types.NewPointer(sxValuePtr)
	closureFn := cg.mod.NewFunc(closureName, sxValuePtr,
		ir.NewParam("args", argsPtrType),
		ir.NewParam("argc", i32),
		ir.NewParam("env", envPtrType),
	)

	entry := closureFn.NewBlock("entry")
	cg.fn = closureFn
	cg.block = entry
	cg.vars = make(map[string]llvmValue.Value)
	cg.scopes = []map[string]llvmValue.Value{}
	cg.pushScope()

	// Load params from args array
	for i, p := range fd.Params {
		argPtr := entry.NewGetElementPtr(sxValuePtr, closureFn.Params[0], constant.NewInt(i32, int64(i)))
		argVal := entry.NewLoad(sxValuePtr, argPtr)
		alloca := entry.NewAlloca(sxValuePtr)
		entry.NewStore(argVal, alloca)
		cg.scopes[len(cg.scopes)-1][p.Name] = alloca
	}

	// Load captured variables from env — each slot is a pointer to
	// a heap cell (SxValue**). We use the cell directly as the variable's
	// storage, so reads/writes are shared with the enclosing scope.
	for i, name := range captures {
		envSlotPtr := entry.NewGetElementPtr(sxValuePtr, closureFn.Params[2], constant.NewInt(i32, int64(i)))
		cellRaw := entry.NewLoad(sxValuePtr, envSlotPtr)
		cellTyped := entry.NewBitCast(cellRaw, types.NewPointer(sxValuePtr))
		cg.scopes[len(cg.scopes)-1][name] = cellTyped
	}

	cg.compileFuncBody(fd.Body.Statements)

	cg.popScope()
	cg.fn = prevFn
	cg.block = prevBlock
	cg.vars = prevVars
	cg.scopes = prevScopes

	// 4. Create the closure value
	fnPtr := cg.block.NewBitCast(closureFn, sxValuePtr)
	closureVal := cg.callRT("sx_closure", fnPtr, envArray, constant.NewInt(i32, int64(envSize)))
	cg.setVar(fd.Name, closureVal)
}

// compileFuncBody compiles the statements of a function body with implicit return.
// The last statement's value is returned implicitly (like the Go evaluator).
func (cg *CodeGen) compileFuncBody(stmts []*parser.Statement) {
	for i, stmt := range stmts {
		isLast := i == len(stmts)-1

		if isLast && cg.block.Term == nil {
			// Implicit return for the last statement
			switch {
			case stmt.ExprStmt != nil:
				val := cg.compileExpr(stmt.ExprStmt.Expr)
				cg.block.NewRet(val)
				continue
			case stmt.Assignment != nil:
				val := cg.compileExpr(stmt.Assignment.Value)
				cg.setVar(stmt.Assignment.Name, val)
				cg.block.NewRet(val)
				continue
			case stmt.TypedAssign != nil:
				val := cg.compileExpr(stmt.TypedAssign.Value)
				cg.setVar(stmt.TypedAssign.Name, val)
				cg.block.NewRet(val)
				continue
			case stmt.CompoundAssign != nil:
				cg.compileCompoundAssign(stmt.CompoundAssign)
				result := cg.getVar(stmt.CompoundAssign.Name)
				cg.block.NewRet(result)
				continue
			}
		}

		cg.compileStatement(stmt)
	}
	if cg.block.Term == nil {
		cg.block.NewRet(cg.callRT("sx_null"))
	}
}

// findCaptures returns the names of variables from the enclosing scope
// that are referenced inside a function body.
func (cg *CodeGen) findCaptures(fd *parser.FuncDef) []string {
	// Collect all variable names used in the function body
	used := map[string]bool{}
	cg.collectIdents(fd.Body.Statements, used)

	// Subtract parameters (they're local, not captured)
	for _, p := range fd.Params {
		delete(used, p.Name)
	}

	// Subtract builtins and constants
	for _, name := range []string{"true", "false", "null"} {
		delete(used, name)
	}

	// Only keep names that exist in the current scope
	var captures []string
	for name := range used {
		if _, ok := cg.resolveVar(name); ok {
			captures = append(captures, name)
		}
	}
	return captures
}

// collectIdents walks statements and collects all identifier references.
func (cg *CodeGen) collectIdents(stmts []*parser.Statement, names map[string]bool) {
	for _, stmt := range stmts {
		switch {
		case stmt.Assignment != nil:
			names[stmt.Assignment.Name] = true
			cg.collectExprIdents(stmt.Assignment.Value, names)
		case stmt.CompoundAssign != nil:
			names[stmt.CompoundAssign.Name] = true
			cg.collectExprIdents(stmt.CompoundAssign.Value, names)
		case stmt.ReturnStmt != nil:
			cg.collectExprIdents(stmt.ReturnStmt.Value, names)
		case stmt.PrintStmt != nil:
			cg.collectExprIdents(stmt.PrintStmt.Value, names)
		case stmt.ExprStmt != nil:
			cg.collectExprIdents(stmt.ExprStmt.Expr, names)
		case stmt.IfStmt != nil:
			cg.collectExprIdents(stmt.IfStmt.Condition, names)
			cg.collectIdents(stmt.IfStmt.Body.Statements, names)
			if stmt.IfStmt.Else != nil {
				cg.collectIdents(stmt.IfStmt.Else.Statements, names)
			}
		case stmt.WhileStmt != nil:
			cg.collectExprIdents(stmt.WhileStmt.Condition, names)
			cg.collectIdents(stmt.WhileStmt.Body.Statements, names)
		case stmt.ForStmt != nil:
			cg.collectExprIdents(stmt.ForStmt.Iter, names)
			cg.collectIdents(stmt.ForStmt.Body.Statements, names)
		case stmt.SwitchStmt != nil:
			cg.collectExprIdents(stmt.SwitchStmt.Value, names)
			for _, c := range stmt.SwitchStmt.Cases {
				cg.collectIdents(c.Body.Statements, names)
			}
			if stmt.SwitchStmt.Default != nil {
				cg.collectIdents(stmt.SwitchStmt.Default.Statements, names)
			}
		case stmt.CatchStmt != nil:
			cg.collectExprIdents(stmt.CatchStmt.Value, names)
			cg.collectIdents(stmt.CatchStmt.Body.Statements, names)
		case stmt.TypedAssign != nil:
			cg.collectExprIdents(stmt.TypedAssign.Value, names)
		case stmt.IndexAssign != nil:
			names[stmt.IndexAssign.Name] = true
		}
	}
}

func (cg *CodeGen) collectExprIdents(expr *parser.Expr, names map[string]bool) {
	if expr == nil {
		return
	}
	cg.collectAndIdents(expr.Left, names)
	for _, op := range expr.Ops {
		cg.collectAndIdents(op.Right, names)
	}
}

func (cg *CodeGen) collectAndIdents(and *parser.LogicalAnd, names map[string]bool) {
	if and == nil {
		return
	}
	cg.collectCmpIdents(and.Left, names)
	for _, op := range and.Ops {
		cg.collectCmpIdents(op.Right, names)
	}
}

func (cg *CodeGen) collectCmpIdents(cmp *parser.Comparison, names map[string]bool) {
	if cmp == nil {
		return
	}
	cg.collectAddIdents(cmp.Left, names)
	if cmp.Right != nil {
		cg.collectAddIdents(cmp.Right, names)
	}
}

func (cg *CodeGen) collectAddIdents(add *parser.Addition, names map[string]bool) {
	if add == nil {
		return
	}
	cg.collectMulIdents(add.Left, names)
	for _, op := range add.Ops {
		cg.collectMulIdents(op.Right, names)
	}
}

func (cg *CodeGen) collectMulIdents(mul *parser.Multiplication, names map[string]bool) {
	if mul == nil {
		return
	}
	cg.collectUnaryIdents(mul.Left, names)
	for _, op := range mul.Ops {
		cg.collectUnaryIdents(op.Right, names)
	}
}

func (cg *CodeGen) collectUnaryIdents(u *parser.Unary, names map[string]bool) {
	if u == nil {
		return
	}
	if u.Not != nil {
		cg.collectUnaryIdents(u.Not, names)
		return
	}
	if u.Neg != nil {
		cg.collectUnaryIdents(u.Neg, names)
		return
	}
	if u.Pos != nil {
		cg.collectUnaryIdents(u.Pos, names)
		return
	}
	if u.Primary != nil {
		cg.collectPrimaryIdents(u.Primary, names)
	}
}

func (cg *CodeGen) collectPrimaryIdents(p *parser.Primary, names map[string]bool) {
	if p == nil {
		return
	}
	if p.Ident != nil {
		names[*p.Ident] = true
	}
	if p.FuncCall != nil {
		for _, arg := range p.FuncCall.Args {
			cg.collectExprIdents(arg, names)
		}
	}
	if p.SubExpr != nil {
		cg.collectExprIdents(p.SubExpr, names)
	}
	if p.ListLit != nil {
		for _, el := range p.ListLit.Elements {
			cg.collectExprIdents(el, names)
		}
	}
	if p.DictLit != nil {
		for _, e := range p.DictLit.Entries {
			cg.collectExprIdents(e.Key, names)
			cg.collectExprIdents(e.Value, names)
		}
	}
	if p.Lambda != nil {
		cg.collectExprIdents(p.Lambda.Body, names)
	}
	for _, s := range p.Suffix {
		if s.Index != nil {
			cg.collectExprIdents(s.Index.Index, names)
		}
		if s.Method != nil {
			for _, arg := range s.Method.Args {
				cg.collectExprIdents(arg, names)
			}
		}
	}
}

func (cg *CodeGen) compileIfStmt(ifStmt *parser.IfStmt) {
	cond := cg.compileExpr(ifStmt.Condition)
	condBool := cg.callRT("sx_truthy", cond)
	condI1 := cg.block.NewICmp(enum.IPredNE, condBool, constant.NewInt(i32, 0))

	thenBlock := cg.newBlock("then")
	elseBlock := cg.newBlock("else")
	mergeBlock := cg.newBlock("merge")

	cg.block.NewCondBr(condI1, thenBlock, elseBlock)

	cg.block = thenBlock
	for _, stmt := range ifStmt.Body.Statements {
		cg.compileStatement(stmt)
	}
	if cg.block.Term == nil {
		cg.block.NewBr(mergeBlock)
	}

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

func (cg *CodeGen) compileCatchStmt(cs *parser.CatchStmt) {
	val := cg.compileExpr(cs.Value)
	cg.setVar(cs.Name, val)

	isErr := cg.callRT("sx_is_error", val)
	errBool := cg.callRT("sx_truthy", isErr)
	condI1 := cg.block.NewICmp(enum.IPredNE, errBool, constant.NewInt(i32, 0))

	catchBlock := cg.newBlock("catch.body")
	mergeBlock := cg.newBlock("catch.merge")

	cg.block.NewCondBr(condI1, catchBlock, mergeBlock)

	cg.block = catchBlock
	for _, stmt := range cs.Body.Statements {
		cg.compileStatement(stmt)
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

	cg.loopContinueBlocks = append(cg.loopContinueBlocks, condBlock)
	cg.loopExitBlocks = append(cg.loopExitBlocks, exitBlock)

	cg.block.NewBr(condBlock)

	cg.block = condBlock
	cond := cg.compileExpr(ws.Condition)
	condBool := cg.callRT("sx_truthy", cond)
	condI1 := cg.block.NewICmp(enum.IPredNE, condBool, constant.NewInt(i32, 0))
	cg.block.NewCondBr(condI1, bodyBlock, exitBlock)

	cg.block = bodyBlock
	for _, stmt := range ws.Body.Statements {
		cg.compileStatement(stmt)
	}
	if cg.block.Term == nil {
		cg.block.NewBr(condBlock)
	}

	cg.loopContinueBlocks = cg.loopContinueBlocks[:len(cg.loopContinueBlocks)-1]
	cg.loopExitBlocks = cg.loopExitBlocks[:len(cg.loopExitBlocks)-1]

	cg.block = exitBlock
}

func (cg *CodeGen) compileForStmt(fs *parser.ForStmt) {
	iter := cg.compileExpr(fs.Iter)
	iterLen := cg.callRT("sx_len", iter)

	idxAlloca := cg.block.NewAlloca(sxValuePtr)
	cg.block.NewStore(cg.callRT("sx_number", constant.NewFloat(types.Double, 0)), idxAlloca)

	condBlock := cg.newBlock("for.cond")
	bodyBlock := cg.newBlock("for.body")
	incrBlock := cg.newBlock("for.incr")
	exitBlock := cg.newBlock("for.exit")

	cg.loopContinueBlocks = append(cg.loopContinueBlocks, incrBlock)
	cg.loopExitBlocks = append(cg.loopExitBlocks, exitBlock)

	cg.block.NewBr(condBlock)

	cg.block = condBlock
	idx := cg.block.NewLoad(sxValuePtr, idxAlloca)
	cond := cg.callRT("sx_lt", idx, iterLen)
	condBool := cg.callRT("sx_truthy", cond)
	condI1 := cg.block.NewICmp(enum.IPredNE, condBool, constant.NewInt(i32, 0))
	cg.block.NewCondBr(condI1, bodyBlock, exitBlock)

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

	cg.block = incrBlock
	idx3 := cg.block.NewLoad(sxValuePtr, idxAlloca)
	one := cg.callRT("sx_number", constant.NewFloat(types.Double, 1))
	newIdx := cg.callRT("sx_add", idx3, one)
	cg.block.NewStore(newIdx, idxAlloca)
	cg.block.NewBr(condBlock)

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
	target := cg.getVar(ia.Name)
	// Navigate through all indices except the last
	for i := 0; i < len(ia.Indices)-1; i++ {
		idx := cg.compileExpr(ia.Indices[i].Index)
		target = cg.callRT("sx_index", target, idx)
	}
	lastIdx := cg.compileExpr(ia.Indices[len(ia.Indices)-1].Index)
	val := cg.compileExpr(ia.Value)
	cg.callRTVoid("sx_index_set", target, lastIdx, val)
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
