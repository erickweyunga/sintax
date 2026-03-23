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
	prevFn := cg.fn
	prevBlock := cg.block
	prevVars := cg.vars // save outer function's vars

	params := make([]*ir.Param, len(fd.Params))
	for i, p := range fd.Params {
		params[i] = ir.NewParam(p.Name, sxValuePtr)
	}
	fn := cg.mod.NewFunc("sx_user_"+fd.Name, sxValuePtr, params...)
	cg.userFuncs[fd.Name] = fn

	entry := fn.NewBlock("entry")
	cg.fn = fn
	cg.block = entry
	cg.vars = make(map[string]*ir.InstAlloca)
	prevScopes := cg.scopes
	cg.scopes = []map[string]*ir.InstAlloca{} // completely fresh scope stack
	cg.pushScope()

	for i, p := range fd.Params {
		alloca := entry.NewAlloca(sxValuePtr)
		entry.NewStore(fn.Params[i], alloca)
		cg.scopes[len(cg.scopes)-1][p.Name] = alloca
	}

	for _, stmt := range fd.Body.Statements {
		cg.compileStatement(stmt)
	}

	if cg.block.Term == nil {
		cg.block.NewRet(cg.callRT("sx_null"))
	}

	cg.popScope()
	cg.fn = prevFn
	cg.block = prevBlock
	cg.vars = prevVars
	cg.scopes = prevScopes
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
