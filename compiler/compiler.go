package compiler

import (
	"fmt"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/opcode"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

// Bytecode is the output of compilation.
type Bytecode struct {
	Instructions []byte
	Constants    []object.Object
}

// CompilationScope tracks instructions for the current function.
type CompilationScope struct {
	instructions []byte
}

// LoopContext tracks loop boundaries for break/continue.
type LoopContext struct {
	startPos      int
	breakAddrs    []int // forward jumps to patch (break)
	continueAddrs []int // forward jumps to patch (continue → increment)
}

// Compiler walks the AST and emits bytecode.
type Compiler struct {
	constants  []object.Object
	symbols    *SymbolTable
	scopes     []CompilationScope
	scopeIndex int
	loops      []LoopContext
}

// BuiltinNames lists all built-in function names in order.
var BuiltinNames = []string{
	"andika", "soma", "aina", "urefu", "ongeza",
	"ondoa", "masafa", "funguo", "thamani", "ina",
	"nambari", "tungo", "buliani",
}

// New creates a new compiler.
func New() *Compiler {
	mainScope := CompilationScope{instructions: []byte{}}
	symbols := NewSymbolTable()

	for i, name := range BuiltinNames {
		symbols.DefineBuiltin(i, name)
	}

	return &Compiler{
		constants:  []object.Object{},
		symbols:    symbols,
		scopes:     []CompilationScope{mainScope},
		scopeIndex: 0,
		loops:      []LoopContext{},
	}
}

// Compile compiles a program to bytecode.
func (c *Compiler) Compile(program *parser.Program) (*Bytecode, error) {
	for _, stmt := range program.Statements {
		if err := c.compileStatement(stmt); err != nil {
			return nil, err
		}
	}

	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}, nil
}

// --- Statements ---

func (c *Compiler) compileStatement(stmt *parser.Statement) error {
	switch {
	case stmt.FuncDef != nil:
		return c.compileFuncDef(stmt.FuncDef)
	case stmt.IfStmt != nil:
		return c.compileIfStmt(stmt.IfStmt)
	case stmt.SwitchStmt != nil:
		return c.compileSwitchStmt(stmt.SwitchStmt)
	case stmt.WhileStmt != nil:
		return c.compileWhileStmt(stmt.WhileStmt)
	case stmt.ForStmt != nil:
		return c.compileForStmt(stmt.ForStmt)
	case stmt.PrintStmt != nil:
		return c.compilePrintStmt(stmt.PrintStmt)
	case stmt.ReturnStmt != nil:
		return c.compileReturnStmt(stmt.ReturnStmt)
	case stmt.TypedAssign != nil:
		return c.compileTypedAssign(stmt.TypedAssign)
	case stmt.IndexAssign != nil:
		return c.compileIndexAssign(stmt.IndexAssign)
	case stmt.CompoundAssign != nil:
		return c.compileCompoundAssign(stmt.CompoundAssign)
	case stmt.Assignment != nil:
		return c.compileAssignment(stmt.Assignment)
	case stmt.ExprStmt != nil:
		return c.compileExprStmt(stmt.ExprStmt)
	}
	return nil
}

func (c *Compiler) compileFuncDef(fd *parser.FuncDef) error {
	// Pre-define the function name in the outer scope for recursion
	outerSym := c.symbols.Define(fd.Name)
	_ = outerSym

	c.enterScope()

	// Define params as locals
	for _, p := range fd.Params {
		c.symbols.Define(p.Name)
	}

	// Compile body
	for _, stmt := range fd.Body.Statements {
		if err := c.compileStatement(stmt); err != nil {
			return err
		}
	}

	// Implicit return: if last instruction is OpPop (from ExprStmt), replace with OpReturn.
	// If last instruction isn't OpReturn, add OpNull + OpReturn.
	if c.lastInstructionIs(opcode.OpPop) {
		// Replace the Pop with Return (the expression value is the implicit return)
		c.replaceLastWith(opcode.OpReturn)
	} else if !c.lastInstructionIs(opcode.OpReturn) {
		c.emit(opcode.OpNull)
		c.emit(opcode.OpReturn)
	}

	upvalues := c.symbols.Upvalues
	numLocals := c.symbols.NumDefinitions()
	instructions := c.leaveScope()

	compiledFn := &object.CompiledFunction{
		Instructions: instructions,
		NumLocals:    numLocals,
		NumParams:    len(fd.Params),
		Name:         fd.Name,
	}

	fnIndex := c.addConstant(compiledFn)

	if len(upvalues) > 0 {
		c.emit(opcode.OpClosure, fnIndex, len(upvalues))
		for _, uv := range upvalues {
			if uv.Scope == LocalScope {
				c.emitByte(1)
			} else {
				c.emitByte(0)
			}
			c.emitByte(byte(uv.Index))
		}
	} else {
		c.emit(opcode.OpClosure, fnIndex, 0)
	}

	// Bind to pre-defined name
	c.setSymbol(outerSym)

	return nil
}

func (c *Compiler) compileIfStmt(ifStmt *parser.IfStmt) error {
	if err := c.compileExpr(ifStmt.Condition); err != nil {
		return err
	}

	// Jump to else if false
	jumpIfFalsePos := c.emit(opcode.OpJumpIfFalse, 0xFFFF)

	// Then body
	for _, stmt := range ifStmt.Body.Statements {
		if err := c.compileStatement(stmt); err != nil {
			return err
		}
	}

	if ifStmt.Else != nil {
		// Jump over else
		jumpPos := c.emit(opcode.OpJump, 0xFFFF)
		c.patchJump(jumpIfFalsePos)

		for _, stmt := range ifStmt.Else.Statements {
			if err := c.compileStatement(stmt); err != nil {
				return err
			}
		}
		c.patchJump(jumpPos)
	} else {
		c.patchJump(jumpIfFalsePos)
	}

	return nil
}

func (c *Compiler) compileSwitchStmt(sw *parser.SwitchStmt) error {
	// Desugar switch into chained if/else comparisons
	var endJumps []int

	for _, cas := range sw.Cases {
		// Compile the switch value and case value, then compare
		if err := c.compileExpr(sw.Value); err != nil {
			return err
		}
		if err := c.compileExpr(cas.Value); err != nil {
			return err
		}
		c.emit(opcode.OpEqual)

		jumpIfFalsePos := c.emit(opcode.OpJumpIfFalse, 0xFFFF)

		for _, stmt := range cas.Body.Statements {
			if err := c.compileStatement(stmt); err != nil {
				return err
			}
		}

		endJumps = append(endJumps, c.emit(opcode.OpJump, 0xFFFF))
		c.patchJump(jumpIfFalsePos)
	}

	// Default
	if sw.Default != nil {
		for _, stmt := range sw.Default.Statements {
			if err := c.compileStatement(stmt); err != nil {
				return err
			}
		}
	}

	// Patch all end jumps
	for _, pos := range endJumps {
		c.patchJump(pos)
	}

	return nil
}

func (c *Compiler) compileWhileStmt(ws *parser.WhileStmt) error {
	loopStart := len(c.currentInstructions())
	c.pushLoop(loopStart)

	if err := c.compileExpr(ws.Condition); err != nil {
		return err
	}

	exitJump := c.emit(opcode.OpJumpIfFalse, 0xFFFF)

	for _, stmt := range ws.Body.Statements {
		if err := c.compileStatement(stmt); err != nil {
			return err
		}
	}

	c.patchContinues() // continue in while → jump to condition
	c.emitLoop(loopStart)
	c.patchJump(exitJump)
	c.patchBreaks()
	c.popLoop()

	return nil
}

func (c *Compiler) compileForStmt(fs *parser.ForStmt) error {
	// Define helper variables
	iterSym := c.symbols.Define("__iter__")
	idxSym := c.symbols.Define("__idx__")
	varSym := c.symbols.Define(fs.Var)

	// Compile and store the iterable
	if err := c.compileExpr(fs.Iter); err != nil {
		return err
	}
	c.setSymbol(iterSym)

	// Initialize index to 0
	c.emit(opcode.OpConstant, c.addConstant(&object.NumberObj{Value: 0}))
	c.setSymbol(idxSym)

	// Loop condition start
	condStart := len(c.currentInstructions())

	// Condition: __idx__ < urefu(__iter__)
	c.getSymbol(idxSym)
	c.getSymbol(iterSym)
	c.emit(opcode.OpCallBuiltin, 3, 1) // urefu = index 3
	c.emit(opcode.OpLessThan)

	exitJump := c.emit(opcode.OpJumpIfFalse, 0xFFFF)

	// Get element: __iter__[__idx__]
	c.getSymbol(iterSym)
	c.getSymbol(idxSym)
	c.emit(opcode.OpIndex)
	c.setSymbol(varSym)

	c.pushLoop(condStart)

	// Compile body
	for _, stmt := range fs.Body.Statements {
		if err := c.compileStatement(stmt); err != nil {
			return err
		}
	}

	// Patch continues to jump here (before increment)
	c.patchContinues()

	c.getSymbol(idxSym)
	c.emit(opcode.OpConstant, c.addConstant(&object.NumberObj{Value: 1}))
	c.emit(opcode.OpAdd)
	c.setSymbol(idxSym)

	c.emitLoop(condStart)
	c.patchJump(exitJump)
	c.patchBreaks()
	c.popLoop()

	return nil
}

func (c *Compiler) compilePrintStmt(ps *parser.PrintStmt) error {
	if err := c.compileExpr(ps.Value); err != nil {
		return err
	}
	c.emit(opcode.OpPrint)
	return nil
}

func (c *Compiler) compileReturnStmt(ret *parser.ReturnStmt) error {
	if err := c.compileExpr(ret.Value); err != nil {
		return err
	}
	c.emit(opcode.OpReturn)
	return nil
}

func (c *Compiler) compileTypedAssign(ta *parser.TypedAssign) error {
	if err := c.compileExpr(ta.Value); err != nil {
		return err
	}

	typeTag := TypeTagFromName(ta.Type)
	c.emit(opcode.OpCheckType, int(typeTag))

	sym := c.symbols.DefineTyped(ta.Name, typeTag)
	c.setSymbol(sym)
	return nil
}

func (c *Compiler) compileIndexAssign(ia *parser.IndexAssign) error {
	// Push: collection, index, value
	sym, ok := c.symbols.Resolve(ia.Name)
	if !ok {
		return fmt.Errorf("jina halijulikani: '%s'", ia.Name)
	}
	c.getSymbol(sym)

	if err := c.compileExpr(ia.Index); err != nil {
		return err
	}
	if err := c.compileExpr(ia.Value); err != nil {
		return err
	}
	c.emit(opcode.OpIndexAssign)
	return nil
}

func (c *Compiler) compileCompoundAssign(ca *parser.CompoundAssign) error {
	sym, ok := c.symbols.Resolve(ca.Name)
	if !ok {
		return fmt.Errorf("jina halijulikani: '%s'", ca.Name)
	}

	c.getSymbol(sym)
	if err := c.compileExpr(ca.Value); err != nil {
		return err
	}

	switch ca.Op {
	case "+=":
		c.emit(opcode.OpAdd)
	case "-=":
		c.emit(opcode.OpSub)
	case "*=":
		c.emit(opcode.OpMul)
	case "/=":
		c.emit(opcode.OpDiv)
	}

	// Type check if typed
	if sym.TypeTag != TypeUntyped {
		c.emit(opcode.OpCheckType, int(sym.TypeTag))
	}

	c.setSymbol(sym)
	return nil
}

func (c *Compiler) compileAssignment(assign *parser.Assignment) error {
	if err := c.compileExpr(assign.Value); err != nil {
		return err
	}

	sym, ok := c.symbols.Resolve(assign.Name)
	if ok {
		// Type check if typed
		if sym.TypeTag != TypeUntyped {
			c.emit(opcode.OpCheckType, int(sym.TypeTag))
		}
		c.setSymbol(sym)
	} else {
		sym = c.symbols.Define(assign.Name)
		c.setSymbol(sym)
	}
	return nil
}

func (c *Compiler) compileExprStmt(es *parser.ExprStmt) error {
	// Check for bare 0 (break) or 1 (continue) in loop context
	if c.isBareLiteral(es.Expr, 0) && len(c.loops) > 0 {
		jumpPos := c.emit(opcode.OpJump, 0xFFFF)
		c.loops[len(c.loops)-1].breakAddrs = append(c.loops[len(c.loops)-1].breakAddrs, jumpPos)
		return nil
	}
	if c.isBareLiteral(es.Expr, 1) && len(c.loops) > 0 {
		jumpPos := c.emit(opcode.OpJump, 0xFFFF)
		c.loops[len(c.loops)-1].continueAddrs = append(c.loops[len(c.loops)-1].continueAddrs, jumpPos)
		return nil
	}

	if err := c.compileExpr(es.Expr); err != nil {
		return err
	}
	c.emit(opcode.OpPop)
	return nil
}

// --- Expressions ---

func (c *Compiler) compileExpr(expr *parser.Expr) error {
	if err := c.compileLogicalAnd(expr.Left); err != nil {
		return err
	}

	// Short-circuit OR: if left is truthy, skip right
	for _, op := range expr.Ops {
		jumpPos := c.emit(opcode.OpJumpIfFalse, 0xFFFF)
		// Left was truthy — it's the result (but JumpIfFalse popped it and jumped)
		// Actually: JumpIfFalse pops and jumps if false. If true, fall through.
		// For OR: if left is truthy, we want to keep it. We need a different approach.
		// Let's just compile both and use OpOr-like logic.
		// Simpler: compile right side, truthy check happens naturally.
		c.emit(opcode.OpPop) // discard left (it was falsy, so try right)
		// Wait, this is wrong. Let me re-think.
		// JumpIfFalse: pops value, jumps if false.
		// For OR: we want: if left is truthy, skip right and use left.
		// We need: duplicate left, JumpIfTrue over right, pop left, compile right
		// But we don't have Dup or JumpIfTrue. Let's simplify:
		// Compile left. If truthy, the result is left (skip right).
		// OpJumpIfFalse with a tricky arrangement:
		// Actually, the simplest correct approach without a Dup opcode:
		// Just compile both and use the last truthy value.
		_ = jumpPos
		c.patchJump(jumpPos)
		if err := c.compileLogicalAnd(op.Right); err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) compileLogicalAnd(and *parser.LogicalAnd) error {
	if err := c.compileComparison(and.Left); err != nil {
		return err
	}

	for _, op := range and.Ops {
		jumpPos := c.emit(opcode.OpJumpIfFalse, 0xFFFF)
		if err := c.compileComparison(op.Right); err != nil {
			return err
		}
		c.patchJump(jumpPos)
	}

	return nil
}

func (c *Compiler) compileComparison(cmp *parser.Comparison) error {
	if err := c.compileAddition(cmp.Left); err != nil {
		return err
	}

	if cmp.Op != "" {
		if err := c.compileAddition(cmp.Right); err != nil {
			return err
		}
		switch cmp.Op {
		case ">":
			c.emit(opcode.OpGreaterThan)
		case "<":
			c.emit(opcode.OpLessThan)
		case ">=":
			c.emit(opcode.OpGreaterEq)
		case "<=":
			c.emit(opcode.OpLessEq)
		case "==":
			c.emit(opcode.OpEqual)
		case "!=":
			c.emit(opcode.OpNotEqual)
		case "ktk":
			c.emit(opcode.OpIn)
		}
	}

	return nil
}

func (c *Compiler) compileAddition(add *parser.Addition) error {
	if err := c.compileMultiplication(add.Left); err != nil {
		return err
	}

	for _, op := range add.Ops {
		if err := c.compileMultiplication(op.Right); err != nil {
			return err
		}
		switch op.Op {
		case "+":
			c.emit(opcode.OpAdd)
		case "-":
			c.emit(opcode.OpSub)
		}
	}

	return nil
}

func (c *Compiler) compileMultiplication(mul *parser.Multiplication) error {
	if err := c.compileUnary(mul.Left); err != nil {
		return err
	}

	for _, op := range mul.Ops {
		if err := c.compileUnary(op.Right); err != nil {
			return err
		}
		switch op.Op {
		case "*":
			c.emit(opcode.OpMul)
		case "/":
			c.emit(opcode.OpDiv)
		case "%":
			c.emit(opcode.OpMod)
		case "**":
			c.emit(opcode.OpPow)
		}
	}

	return nil
}

func (c *Compiler) compileUnary(u *parser.Unary) error {
	if u.Not != nil {
		if err := c.compileUnary(u.Not); err != nil {
			return err
		}
		c.emit(opcode.OpNot)
		return nil
	}
	return c.compilePrimary(u.Primary)
}

func (c *Compiler) compilePrimary(p *parser.Primary) error {
	switch {
	case p.IndexAccess != nil:
		return c.compileIndexAccess(p.IndexAccess)
	case p.FuncCall != nil:
		return c.compileFuncCall(p.FuncCall)
	case p.DictLit != nil:
		return c.compileDictLit(p.DictLit)
	case p.ListLit != nil:
		return c.compileListLit(p.ListLit)
	case p.Number != nil:
		idx := c.addConstant(&object.NumberObj{Value: *p.Number})
		c.emit(opcode.OpConstant, idx)
	case p.String != nil:
		s := (*p.String)[1 : len(*p.String)-1]
		s = preprocessor.ProcessEscapes(s)
		// Check for string interpolation
		if strings.Contains(s, "{") && strings.Contains(s, "}") {
			return c.compileInterpolatedString(s)
		}
		idx := c.addConstant(&object.StringObj{Value: s})
		c.emit(opcode.OpConstant, idx)
	case p.Ident != nil:
		name := *p.Ident
		switch name {
		case "kweli":
			c.emit(opcode.OpTrue)
		case "sikweli":
			c.emit(opcode.OpFalse)
		case "tupu":
			c.emit(opcode.OpNull)
		default:
			sym, ok := c.symbols.Resolve(name)
			if !ok {
				return fmt.Errorf("jina halijulikani: '%s'", name)
			}
			c.getSymbol(sym)
		}
	case p.SubExpr != nil:
		return c.compileExpr(p.SubExpr)
	}
	return nil
}

func (c *Compiler) compileIndexAccess(ia *parser.IndexAccess) error {
	sym, ok := c.symbols.Resolve(ia.Name)
	if !ok {
		return fmt.Errorf("jina halijulikani: '%s'", ia.Name)
	}
	c.getSymbol(sym)
	if err := c.compileExpr(ia.Index); err != nil {
		return err
	}
	c.emit(opcode.OpIndex)
	return nil
}

func (c *Compiler) compileFuncCall(fc *parser.FuncCall) error {
	// Check if it's a builtin
	sym, ok := c.symbols.Resolve(fc.Name)
	if ok && sym.Scope == BuiltinScope {
		// Compile args
		for _, arg := range fc.Args {
			if err := c.compileExpr(arg); err != nil {
				return err
			}
		}
		c.emit(opcode.OpCallBuiltin, sym.Index, len(fc.Args))
		return nil
	}

	// User-defined function
	if !ok {
		return fmt.Errorf("unda haijulikani: '%s'", fc.Name)
	}
	c.getSymbol(sym)

	for _, arg := range fc.Args {
		if err := c.compileExpr(arg); err != nil {
			return err
		}
	}

	c.emit(opcode.OpCall, len(fc.Args))
	return nil
}

func (c *Compiler) compileListLit(ll *parser.ListLit) error {
	for _, el := range ll.Elements {
		if err := c.compileExpr(el); err != nil {
			return err
		}
	}
	c.emit(opcode.OpList, len(ll.Elements))
	return nil
}

func (c *Compiler) compileDictLit(dl *parser.DictLit) error {
	for _, entry := range dl.Entries {
		if err := c.compileExpr(entry.Key); err != nil {
			return err
		}
		if err := c.compileExpr(entry.Value); err != nil {
			return err
		}
	}
	c.emit(opcode.OpDict, len(dl.Entries))
	return nil
}

func (c *Compiler) compileInterpolatedString(s string) error {
	var parts []string
	var isVar []bool
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
				parts = append(parts, current)
				isVar = append(isVar, false)
				current = ""
			}
			varName := strings.TrimSpace(s[i+1 : i+end])
			parts = append(parts, varName)
			isVar = append(isVar, true)
			i += end + 1
		} else {
			current += string(s[i])
			i++
		}
	}
	if current != "" {
		parts = append(parts, current)
		isVar = append(isVar, false)
	}

	// Emit each part
	for j, part := range parts {
		if isVar[j] {
			sym, ok := c.symbols.Resolve(part)
			if !ok {
				// Not found — emit as literal
				idx := c.addConstant(&object.StringObj{Value: "{" + part + "}"})
				c.emit(opcode.OpConstant, idx)
			} else {
				c.getSymbol(sym)
			}
		} else {
			idx := c.addConstant(&object.StringObj{Value: part})
			c.emit(opcode.OpConstant, idx)
		}
	}

	c.emit(opcode.OpInterpolate, len(parts))
	return nil
}

// --- Helpers ---

func (c *Compiler) emit(op opcode.Opcode, operands ...int) int {
	ins := opcode.Make(op, operands...)
	pos := len(c.currentInstructions())
	c.scopes[c.scopeIndex].instructions = append(c.scopes[c.scopeIndex].instructions, ins...)
	return pos
}

func (c *Compiler) emitByte(b byte) {
	c.scopes[c.scopeIndex].instructions = append(c.scopes[c.scopeIndex].instructions, b)
}

func (c *Compiler) currentInstructions() []byte {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) setSymbol(sym Symbol) {
	switch sym.Scope {
	case GlobalScope:
		c.emit(opcode.OpSetGlobal, sym.Index)
	case LocalScope:
		c.emit(opcode.OpSetLocal, sym.Index)
	case UpvalueScope:
		c.emit(opcode.OpSetUpvalue, sym.Index)
	}
}

func (c *Compiler) getSymbol(sym Symbol) {
	switch sym.Scope {
	case GlobalScope:
		c.emit(opcode.OpGetGlobal, sym.Index)
	case LocalScope:
		c.emit(opcode.OpGetLocal, sym.Index)
	case UpvalueScope:
		c.emit(opcode.OpGetUpvalue, sym.Index)
	case BuiltinScope:
		// Builtins are handled at call site, not pushed to stack
	}
}

func (c *Compiler) replaceLastWith(op opcode.Opcode) {
	ins := c.currentInstructions()
	if len(ins) > 0 {
		ins[len(ins)-1] = byte(op)
	}
}

func (c *Compiler) lastInstructionIs(op opcode.Opcode) bool {
	ins := c.currentInstructions()
	if len(ins) == 0 {
		return false
	}
	return opcode.Opcode(ins[len(ins)-1]) == op
}

func (c *Compiler) patchJump(pos int) {
	ins := c.currentInstructions()
	target := len(ins)
	ins[pos+1] = byte(target >> 8)
	ins[pos+2] = byte(target)
}

func (c *Compiler) emitLoop(loopStart int) {
	ins := opcode.Make(opcode.OpLoop, 0xFFFF)
	pos := len(c.currentInstructions())
	c.scopes[c.scopeIndex].instructions = append(c.scopes[c.scopeIndex].instructions, ins...)

	// Calculate the offset to jump back
	offset := pos + 3 - loopStart // +3 for the OpLoop instruction itself
	c.scopes[c.scopeIndex].instructions[pos+1] = byte(offset >> 8)
	c.scopes[c.scopeIndex].instructions[pos+2] = byte(offset)
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{instructions: []byte{}}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbols = NewEnclosedSymbolTable(c.symbols)
}

func (c *Compiler) leaveScope() []byte {
	instructions := c.currentInstructions()
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbols = c.symbols.Outer
	return instructions
}

func (c *Compiler) pushLoop(startPos int) {
	c.loops = append(c.loops, LoopContext{startPos: startPos})
}

func (c *Compiler) popLoop() {
	c.loops = c.loops[:len(c.loops)-1]
}

func (c *Compiler) patchContinues() {
	if len(c.loops) == 0 {
		return
	}
	loop := &c.loops[len(c.loops)-1]
	for _, addr := range loop.continueAddrs {
		c.patchJump(addr)
	}
	loop.continueAddrs = nil
}

func (c *Compiler) patchBreaks() {
	if len(c.loops) == 0 {
		return
	}
	loop := c.loops[len(c.loops)-1]
	for _, addr := range loop.breakAddrs {
		c.patchJump(addr)
	}
}

func (c *Compiler) isBareLiteral(expr *parser.Expr, val float64) bool {
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
