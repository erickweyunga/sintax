package analyzer

import (
	"fmt"
	"strings"

	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
	"github.com/erickweyunga/sintax/stdlib"
)

// Error represents an analysis error with source context.
type Error struct {
	Level   string // "error" or "warning"
	Message string
	File    string
	Line    int
	Source  string // the original source line
}

// Format returns a Rust-style formatted error string.
func (e Error) Format() string {
	var b strings.Builder
	if e.Level == "warning" {
		b.WriteString(fmt.Sprintf("\033[33mwarning\033[0m: %s\n", e.Message))
	} else {
		b.WriteString(fmt.Sprintf("\033[31merror\033[0m: %s\n", e.Message))
	}
	if e.File != "" && e.Line > 0 {
		b.WriteString(fmt.Sprintf("  \033[2m-->\033[0m %s:%d\n", e.File, e.Line))
	}
	if e.Source != "" {
		b.WriteString(fmt.Sprintf("   |\n%3d| %s\n   |\n", e.Line, e.Source))
	}
	return b.String()
}

// FuncInfo holds what we know about a function at analysis time.
type FuncInfo struct {
	Name       string
	Arity      int
	ParamNames []string
	Line       int  // definition line (0 = builtin)
	Used       bool // was this function ever called?
	IsBuiltin  bool
}

// VarInfo holds what we know about a variable at analysis time.
type VarInfo struct {
	Type    string // "" = dynamic
	Defined bool
	Used    bool // was this variable ever read?
	Line    int  // definition line
	Name    string
}

// ImportInfo tracks whether an import was used.
type ImportInfo struct {
	Module   string
	Function string
	Used     bool
	Line     int
}

// Analyzer validates a Sintax AST before execution or compilation.
type Analyzer struct {
	scopes    []map[string]*VarInfo
	functions map[string]*FuncInfo
	imports   []preprocessor.Import

	// imported function names (stdlib or user modules)
	importedFuncs map[string]*ImportInfo

	// source info for error messages
	file    string
	lines   []string
	lineMap []int

	errors []Error
	inLoop int  // nesting depth of loops
	inFunc bool // inside a function body?

	// Track string concat in loops: variable name → true if s += "..." seen in loop
	loopStringConcat map[string]bool
}

// New creates a new analyzer.
func New(file string, lines []string, lineMap []int) *Analyzer {
	return &Analyzer{
		scopes:           []map[string]*VarInfo{},
		functions:        make(map[string]*FuncInfo),
		importedFuncs:    make(map[string]*ImportInfo),
		loopStringConcat: make(map[string]bool),
		file:             file,
		lines:            lines,
		lineMap:          lineMap,
	}
}

// Analyze runs all checks on a program and returns any errors found.
func Analyze(
	program *parser.Program,
	imports []preprocessor.Import,
	file string,
	lines []string,
	lineMap []int,
) []Error {
	a := New(file, lines, lineMap)
	a.imports = imports
	a.registerImports()
	a.registerBuiltins()
	a.analyze(program)

	// Post-analysis checks
	a.checkUnusedVars()
	a.checkUnusedFuncs()
	a.checkUnusedImports()

	return a.errors
}

func (a *Analyzer) analyze(program *parser.Program) {
	a.pushScope()

	// Pass 1: register all top-level function definitions (enables forward calls)
	for _, stmt := range program.Statements {
		if stmt.FuncDef != nil {
			a.registerFunc(stmt.FuncDef)
		}
	}

	// Mark functions referenced in -- test: comments as used
	a.markTestReferences()

	// Pass 2: check all statements
	for i, stmt := range program.Statements {
		a.checkStatement(stmt)

		// Check unreachable code after return at top level
		if stmt.ReturnStmt != nil && i < len(program.Statements)-1 {
			next := program.Statements[i+1]
			a.addWarning(next.Pos.Line, "Unreachable code after return")
			break
		}
	}

	a.popScope()
}

// markTestReferences scans source lines for -- test: comments and marks
// any function names found in them as used.
func (a *Analyzer) markTestReferences() {
	for _, line := range a.lines {
		trimmed := strings.TrimSpace(line)
		if expr, ok := strings.CutPrefix(trimmed, "-- test:"); ok {
			expr = strings.TrimSpace(expr)
			// Extract function names from test expressions
			// Look for patterns like funcName( in the test expression
			a.markFuncNamesInString(expr)
		}
	}
}

// markFuncNamesInString finds function call patterns (name followed by '(')
// in a string and marks those functions as used.
func (a *Analyzer) markFuncNamesInString(s string) {
	i := 0
	for i < len(s) {
		// Find start of identifier
		if isIdentStart(s[i]) {
			j := i + 1
			for j < len(s) && isIdentCont(s[j]) {
				j++
			}
			name := s[i:j]
			// Check if followed by '(' — it's a function call
			rest := strings.TrimSpace(s[j:])
			if len(rest) > 0 && rest[0] == '(' {
				if f, ok := a.functions[name]; ok {
					f.Used = true
				}
			}
			// Also mark as a variable reference
			a.markTestVar(name)
			i = j
		} else {
			i++
		}
	}
}

func (a *Analyzer) markTestVar(name string) {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if v, ok := a.scopes[i][name]; ok {
			v.Used = true
			return
		}
	}
}

// markInterpolationRefs finds {varName} patterns in string literals
// and marks those variables as used.
func (a *Analyzer) markInterpolationRefs(s string) {
	// Strip surrounding quotes
	if len(s) >= 2 {
		s = s[1 : len(s)-1]
	}
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			end := strings.IndexByte(s[i:], '}')
			if end == -1 {
				break
			}
			name := strings.TrimSpace(s[i+1 : i+end])
			if name != "" {
				a.markUsed(name)
			}
			i += end + 1
		} else {
			i++
		}
	}
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isIdentCont(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

// --- Error helpers ---

func (a *Analyzer) addError(line int, format string, args ...interface{}) {
	origLine := a.originalLine(line)
	a.errors = append(a.errors, Error{
		Level:   "error",
		Message: fmt.Sprintf(format, args...),
		File:    a.file,
		Line:    origLine,
		Source:  a.sourceLine(origLine),
	})
}

func (a *Analyzer) addWarning(line int, format string, args ...interface{}) {
	origLine := a.originalLine(line)
	a.errors = append(a.errors, Error{
		Level:   "warning",
		Message: fmt.Sprintf(format, args...),
		File:    a.file,
		Line:    origLine,
		Source:  a.sourceLine(origLine),
	})
}

func (a *Analyzer) originalLine(preprocessedLine int) int {
	if preprocessedLine > 0 && preprocessedLine <= len(a.lineMap) {
		return a.lineMap[preprocessedLine-1]
	}
	return preprocessedLine
}

func (a *Analyzer) sourceLine(origLine int) string {
	if origLine > 0 && origLine <= len(a.lines) {
		return strings.TrimSpace(a.lines[origLine-1])
	}
	return ""
}

// --- Scope management ---

func (a *Analyzer) pushScope() {
	a.scopes = append(a.scopes, make(map[string]*VarInfo))
}

func (a *Analyzer) popScope() {
	a.scopes = a.scopes[:len(a.scopes)-1]
}

func (a *Analyzer) define(name string, typ string, line int) {
	if len(a.scopes) > 0 {
		a.scopes[len(a.scopes)-1][name] = &VarInfo{
			Type:    typ,
			Defined: true,
			Name:    name,
			Line:    line,
		}
	}
}

func (a *Analyzer) isDefined(name string) bool {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if _, ok := a.scopes[i][name]; ok {
			return true
		}
	}
	if _, ok := a.functions[name]; ok {
		return true
	}
	if _, ok := a.importedFuncs[name]; ok {
		return true
	}
	return false
}

func (a *Analyzer) markUsed(name string) {
	// Mark variable as used
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if v, ok := a.scopes[i][name]; ok {
			v.Used = true
			return
		}
	}
	// Mark function as used
	if f, ok := a.functions[name]; ok {
		f.Used = true
		return
	}
	// Mark imported function as used
	if imp, ok := a.importedFuncs[name]; ok {
		imp.Used = true
	}
}

func (a *Analyzer) getVarType(name string) string {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if v, ok := a.scopes[i][name]; ok {
			return v.Type
		}
	}
	return ""
}

// --- Registration ---

func (a *Analyzer) registerFunc(fd *parser.FuncDef) {
	paramNames := make([]string, len(fd.Params))
	for i, p := range fd.Params {
		paramNames[i] = p.Name
	}
	a.functions[fd.Name] = &FuncInfo{
		Name:       fd.Name,
		Arity:      len(fd.Params),
		ParamNames: paramNames,
	}
}

func (a *Analyzer) registerBuiltins() {
	builtins := map[string]int{
		"print":  -1,
		"type":   1,
		"len":    1,
		"push":   2,
		"pop":    2,
		"range":  -1, // 1 or 2
		"input":  -1, // 0 or 1
		"keys":   1,
		"values": 1,
		"has":    2,
		"num":    1,
		"str":    1,
		"bool":   1,
	}
	for name, arity := range builtins {
		a.functions[name] = &FuncInfo{
			Name:      name,
			Arity:     arity,
			IsBuiltin: true,
			Used:      true, // don't warn about unused builtins
		}
	}
}

func (a *Analyzer) registerImports() {
	for _, imp := range a.imports {
		if strings.HasSuffix(imp.Module, ".sx") {
			modName := strings.TrimSuffix(imp.Module, ".sx")
			if idx := strings.LastIndex(modName, "/"); idx != -1 {
				modName = modName[idx+1:]
			}
			switch imp.Function {
			case "":
				a.importedFuncs["__user_module__"+modName] = &ImportInfo{Module: imp.Module, Used: true}
			case "*":
				a.importedFuncs["__user_wildcard__"+modName] = &ImportInfo{Module: imp.Module, Used: true}
			default:
				a.importedFuncs[imp.Function] = &ImportInfo{Module: imp.Module, Function: imp.Function}
			}
			continue
		}

		mod, ok := stdlib.Registry[imp.Module]
		if !ok {
			a.errors = append(a.errors, Error{
				Level:   "error",
				Message: fmt.Sprintf("Unknown module '%s'", imp.Module),
				File:    a.file,
			})
			continue
		}

		switch imp.Function {
		case "":
			for name := range mod.Funcs {
				a.importedFuncs[imp.Module+"__"+name] = &ImportInfo{Module: imp.Module, Function: name}
			}
		case "*":
			for name := range mod.Funcs {
				a.importedFuncs[name] = &ImportInfo{Module: imp.Module, Function: name}
			}
		default:
			if _, ok := mod.Funcs[imp.Function]; !ok {
				a.errors = append(a.errors, Error{
					Level:   "error",
					Message: fmt.Sprintf("'%s' not found in module '%s'", imp.Function, imp.Module),
					File:    a.file,
				})
				continue
			}
			a.importedFuncs[imp.Function] = &ImportInfo{Module: imp.Module, Function: imp.Function}
		}
	}
}

// --- Post-analysis checks ---

func (a *Analyzer) checkUnusedVars() {
	if len(a.scopes) == 0 {
		return
	}
	// Check top-level scope
	for _, v := range a.scopes[0] {
		if !v.Used && v.Line > 0 && v.Type != "fn" {
			a.addWarning(v.Line, "Variable '%s' is defined but never used", v.Name)
		}
	}
}

func (a *Analyzer) checkUnusedFuncs() {
	for _, f := range a.functions {
		if !f.Used && !f.IsBuiltin && f.Line > 0 {
			a.addWarning(f.Line, "Function '%s' is defined but never called", f.Name)
		}
	}
}

func (a *Analyzer) checkUnusedImports() {
	// Group by module to detect entirely unused modules
	moduleUsed := make(map[string]bool)
	for _, imp := range a.importedFuncs {
		if imp.Used {
			moduleUsed[imp.Module] = true
		}
	}
	for _, imp := range a.imports {
		mod := imp.Module
		if strings.HasSuffix(mod, ".sx") {
			continue // skip user module imports
		}
		if _, ok := stdlib.Registry[mod]; !ok {
			continue // already reported as unknown module
		}
		if !moduleUsed[mod] {
			a.addWarning(0, "Imported module '%s' is never used", mod)
		}
	}
}

// --- Statement checks ---

func (a *Analyzer) checkStatement(stmt *parser.Statement) {
	line := stmt.Pos.Line

	switch {
	case stmt.FuncDef != nil:
		a.checkFuncDef(stmt.FuncDef, line)
	case stmt.IfStmt != nil:
		a.checkIfStmt(stmt.IfStmt, line)
	case stmt.SwitchStmt != nil:
		a.checkSwitchStmt(stmt.SwitchStmt, line)
	case stmt.WhileStmt != nil:
		a.checkWhileStmt(stmt.WhileStmt, line)
	case stmt.ForStmt != nil:
		a.checkForStmt(stmt.ForStmt, line)
	case stmt.PrintStmt != nil:
		a.checkExpr(stmt.PrintStmt.Value, line)
	case stmt.ReturnStmt != nil:
		a.checkReturnStmt(stmt.ReturnStmt, line)
	case stmt.TypedAssign != nil:
		a.checkTypedAssign(stmt.TypedAssign, line)
	case stmt.IndexAssign != nil:
		a.checkIndexAssign(stmt.IndexAssign, line)
	case stmt.CompoundAssign != nil:
		a.checkCompoundAssign(stmt.CompoundAssign, line)
	case stmt.Assignment != nil:
		a.checkAssignment(stmt.Assignment, line)
	case stmt.ExprStmt != nil:
		a.checkExprStmt(stmt.ExprStmt, line)
	}
}

func (a *Analyzer) checkFuncDef(fd *parser.FuncDef, line int) {
	// Check for duplicate parameter names
	seen := make(map[string]bool)
	for _, p := range fd.Params {
		if seen[p.Name] {
			a.addError(line, "Duplicate parameter '%s' in function '%s'", p.Name, fd.Name)
		}
		seen[p.Name] = true
	}

	// Update function info with line
	if f, ok := a.functions[fd.Name]; ok {
		f.Line = line
	}

	// Define the function name in current scope
	a.define(fd.Name, "fn", line)

	// Check body in a new scope
	prevInFunc := a.inFunc
	a.inFunc = true
	a.pushScope()

	// Define parameters in the function scope
	for _, p := range fd.Params {
		typ := ""
		if p.Type != nil {
			typ = *p.Type
		}
		a.define(p.Name, typ, line)
		// Mark params as used (they're provided by caller)
		a.scopes[len(a.scopes)-1][p.Name].Used = true
	}

	// Register nested function defs (forward declaration within function body)
	for _, stmt := range fd.Body.Statements {
		if stmt.FuncDef != nil {
			a.registerFunc(stmt.FuncDef)
		}
	}

	// Check for empty function body
	if len(fd.Body.Statements) == 0 {
		a.addWarning(line, "Function '%s' has an empty body", fd.Name)
	}

	a.checkStatementsWithReachability(fd.Body.Statements)

	// Check unused variables in function scope
	for _, v := range a.scopes[len(a.scopes)-1] {
		if !v.Used && v.Type != "fn" {
			a.addWarning(v.Line, "Variable '%s' is defined but never used in function '%s'", v.Name, fd.Name)
		}
	}

	a.popScope()
	a.inFunc = prevInFunc
}

// checkStatementsWithReachability checks statements and detects unreachable code.
func (a *Analyzer) checkStatementsWithReachability(stmts []*parser.Statement) {
	for i, stmt := range stmts {
		a.checkStatement(stmt)

		// Detect unreachable code after return/break/continue
		isTerminator := false
		switch {
		case stmt.ReturnStmt != nil:
			isTerminator = true
		case stmt.ExprStmt != nil && stmt.ExprStmt.Expr.IsBareLiteral(0):
			isTerminator = true // break
		case stmt.ExprStmt != nil && stmt.ExprStmt.Expr.IsBareLiteral(1):
			isTerminator = true // continue
		}

		if isTerminator && i < len(stmts)-1 {
			next := stmts[i+1]
			a.addWarning(next.Pos.Line, "Unreachable code after %s", terminatorName(stmt))
			break
		}
	}
}

func terminatorName(stmt *parser.Statement) string {
	switch {
	case stmt.ReturnStmt != nil:
		return "return"
	case stmt.ExprStmt != nil && stmt.ExprStmt.Expr.IsBareLiteral(0):
		return "break"
	case stmt.ExprStmt != nil && stmt.ExprStmt.Expr.IsBareLiteral(1):
		return "continue"
	}
	return "terminator"
}

func (a *Analyzer) checkIfStmt(ifStmt *parser.IfStmt, line int) {
	a.checkExpr(ifStmt.Condition, line)

	// Warn on constant condition
	if isConstTrue(ifStmt.Condition) {
		a.addWarning(line, "Condition is always true")
	} else if isConstFalse(ifStmt.Condition) {
		a.addWarning(line, "Condition is always false, body will never execute")
	}

	if len(ifStmt.Body.Statements) == 0 {
		a.addWarning(line, "Empty 'if' body")
	}

	a.checkBlock(ifStmt.Body)
	if ifStmt.Else != nil {
		a.checkBlock(ifStmt.Else)
	}
}

func (a *Analyzer) checkSwitchStmt(sw *parser.SwitchStmt, line int) {
	a.checkExpr(sw.Value, line)

	// Check for duplicate case values
	seenCases := make(map[string]bool)
	for _, c := range sw.Cases {
		a.checkExpr(c.Value, line)
		key := exprLiteralKey(c.Value)
		if key != "" {
			if seenCases[key] {
				a.addWarning(line, "Duplicate case value: %s", key)
			}
			seenCases[key] = true
		}
		if len(c.Body.Statements) == 0 {
			a.addWarning(line, "Empty case body")
		}
		a.checkBlock(c.Body)
	}

	if sw.Default != nil {
		a.checkBlock(sw.Default)
	}
}

func (a *Analyzer) checkWhileStmt(ws *parser.WhileStmt, line int) {
	a.checkExpr(ws.Condition, line)

	// Detect infinite loops: while true: with no break
	if isConstTrue(ws.Condition) && !blockHasBreak(ws.Body) {
		a.addWarning(line, "Infinite loop: 'while true' without break")
	}

	if len(ws.Body.Statements) == 0 {
		a.addWarning(line, "Empty 'while' body")
	}

	a.inLoop++
	a.checkBlockForLoopPatterns(ws.Body)
	a.inLoop--
}

func (a *Analyzer) checkForStmt(fs *parser.ForStmt, line int) {
	a.checkExpr(fs.Iter, line)

	if len(fs.Body.Statements) == 0 {
		a.addWarning(line, "Empty 'for' body")
	}

	a.inLoop++
	a.define(fs.Var, "", line)
	// Mark for variable as used (it's the iteration variable)
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if v, ok := a.scopes[i][fs.Var]; ok {
			v.Used = true
			break
		}
	}

	for _, stmt := range fs.Body.Statements {
		if stmt.FuncDef != nil {
			a.registerFunc(stmt.FuncDef)
		}
	}
	a.checkBlockForLoopPatterns(fs.Body)
	a.inLoop--
}

func (a *Analyzer) checkReturnStmt(ret *parser.ReturnStmt, line int) {
	if !a.inFunc {
		a.addError(line, "'return' outside of function")
	}
	a.checkExpr(ret.Value, line)
}

func (a *Analyzer) checkTypedAssign(ta *parser.TypedAssign, line int) {
	a.checkExpr(ta.Value, line)

	if valType := a.inferExprType(ta.Value); valType != "" && valType != ta.Type {
		a.addError(line, "Type mismatch: assigning %s to '%s' (declared as %s)", valType, ta.Name, ta.Type)
	}

	a.define(ta.Name, ta.Type, line)
}

func (a *Analyzer) checkIndexAssign(ia *parser.IndexAssign, line int) {
	if !a.isDefined(ia.Name) {
		a.addError(line, "Undefined name: '%s'", ia.Name)
	} else {
		a.markUsed(ia.Name)
	}
	a.checkExpr(ia.Index, line)
	a.checkExpr(ia.Value, line)
}

func (a *Analyzer) checkCompoundAssign(ca *parser.CompoundAssign, line int) {
	if !a.isDefined(ca.Name) {
		a.addError(line, "Undefined name: '%s'", ca.Name)
	} else {
		a.markUsed(ca.Name)
	}
	a.checkExpr(ca.Value, line)

	// Detect string concatenation in loops: s += "..."
	if a.inLoop > 0 && ca.Op == "+=" {
		varType := a.getVarType(ca.Name)
		valType := a.inferExprType(ca.Value)
		if varType == "str" || valType == "str" {
			if !a.loopStringConcat[ca.Name] {
				a.loopStringConcat[ca.Name] = true
				a.addWarning(line, "String concatenation '%s += ...' in loop is O(n²), consider using a list and join()", ca.Name)
			}
		}
	}
}

func (a *Analyzer) checkAssignment(assign *parser.Assignment, line int) {
	a.checkExpr(assign.Value, line)

	if existingType := a.getVarType(assign.Name); existingType != "" && existingType != "fn" {
		if valType := a.inferExprType(assign.Value); valType != "" && valType != existingType {
			a.addError(line, "Type mismatch: '%s' is %s, cannot assign %s", assign.Name, existingType, valType)
		}
	}

	if !a.isDefined(assign.Name) {
		a.define(assign.Name, "", line)
	}
}

func (a *Analyzer) checkExprStmt(es *parser.ExprStmt, line int) {
	if es.Expr.IsBareLiteral(0) {
		if a.inLoop == 0 {
			a.addError(line, "'0' (break) outside of loop")
		}
		return
	}
	if es.Expr.IsBareLiteral(1) {
		if a.inLoop == 0 {
			a.addError(line, "'1' (continue) outside of loop")
		}
		return
	}
	a.checkExpr(es.Expr, line)
}

func (a *Analyzer) checkBlock(block *parser.Block) {
	for _, stmt := range block.Statements {
		if stmt.FuncDef != nil {
			a.registerFunc(stmt.FuncDef)
		}
	}
	a.checkStatementsWithReachability(block.Statements)
}

// checkBlockForLoopPatterns checks a block that is a loop body,
// detecting loop-specific issues.
func (a *Analyzer) checkBlockForLoopPatterns(block *parser.Block) {
	for _, stmt := range block.Statements {
		if stmt.FuncDef != nil {
			a.registerFunc(stmt.FuncDef)
		}
	}
	a.checkStatementsWithReachability(block.Statements)
}

// --- Expression checks ---

func (a *Analyzer) checkExpr(expr *parser.Expr, line int) {
	if expr == nil {
		return
	}
	a.checkLogicalAnd(expr.Left, line)
	for _, op := range expr.Ops {
		a.checkLogicalAnd(op.Right, line)
	}
}

func (a *Analyzer) checkLogicalAnd(and *parser.LogicalAnd, line int) {
	if and == nil {
		return
	}
	a.checkComparison(and.Left, line)
	for _, op := range and.Ops {
		a.checkComparison(op.Right, line)
	}
}

func (a *Analyzer) checkComparison(cmp *parser.Comparison, line int) {
	if cmp == nil {
		return
	}
	a.checkAddition(cmp.Left, line)
	if cmp.Right != nil {
		a.checkAddition(cmp.Right, line)
	}

	// Self-comparison: x == x is always true, x != x is always false
	if cmp.Op == "==" || cmp.Op == "!=" {
		if leftName := additionIdentName(cmp.Left); leftName != "" {
			if rightName := additionIdentName(cmp.Right); rightName == leftName {
				if cmp.Op == "==" {
					a.addWarning(line, "Comparing '%s' to itself is always true", leftName)
				} else {
					a.addWarning(line, "Comparing '%s' to itself is always false", leftName)
				}
			}
		}
	}
}

func (a *Analyzer) checkAddition(add *parser.Addition, line int) {
	if add == nil {
		return
	}
	a.checkMultiplication(add.Left, line)
	for _, op := range add.Ops {
		a.checkMultiplication(op.Right, line)
	}
}

func (a *Analyzer) checkMultiplication(mul *parser.Multiplication, line int) {
	if mul == nil {
		return
	}
	a.checkUnary(mul.Left, line)
	for _, op := range mul.Ops {
		a.checkUnary(op.Right, line)

		// Division by zero: x / 0 or x % 0
		if op.Op == "/" || op.Op == "%" {
			if isLiteralZero(op.Right) {
				a.addError(line, "Division by zero")
			}
		}
	}
}

func (a *Analyzer) checkUnary(u *parser.Unary, line int) {
	if u == nil {
		return
	}
	if u.Not != nil {
		a.checkUnary(u.Not, line)
		return
	}
	if u.Neg != nil {
		a.checkUnary(u.Neg, line)
		return
	}
	if u.Pos != nil {
		a.checkUnary(u.Pos, line)
		return
	}
	if u.Primary != nil {
		a.checkPrimary(u.Primary, line)
	}
}

func (a *Analyzer) checkPrimary(p *parser.Primary, line int) {
	if p == nil {
		return
	}

	switch {
	case p.Lambda != nil:
		a.checkLambda(p.Lambda, line)
	case p.IndexAccess != nil:
		a.checkIndexAccess(p.IndexAccess, line)
	case p.FuncCall != nil:
		a.checkFuncCall(p.FuncCall, line)
	case p.DictLit != nil:
		a.checkDictLit(p.DictLit, line)
	case p.ListLit != nil:
		for _, el := range p.ListLit.Elements {
			a.checkExpr(el, line)
		}
	case p.String != nil:
		// Mark variables used in string interpolation: "hello {name}"
		a.markInterpolationRefs(*p.String)
	case p.Ident != nil:
		a.checkIdent(*p.Ident, line)
	case p.SubExpr != nil:
		a.checkExpr(p.SubExpr, line)
	}

	for _, mc := range p.Methods {
		a.checkMethodCall(mc, line)
	}
}

func (a *Analyzer) checkIdent(name string, line int) {
	switch name {
	case "true", "false", "null":
		return
	}
	if !a.isDefined(name) {
		a.addError(line, "Undefined name: '%s'", name)
	} else {
		a.markUsed(name)
	}
}

func (a *Analyzer) checkIndexAccess(ia *parser.IndexAccess, line int) {
	if !a.isDefined(ia.Name) {
		a.addError(line, "Undefined name: '%s'", ia.Name)
	} else {
		a.markUsed(ia.Name)
	}
	a.checkExpr(ia.Index, line)
}

func (a *Analyzer) checkLambda(l *parser.Lambda, line int) {
	seen := make(map[string]bool)
	for _, name := range l.Params {
		if seen[name] {
			a.addError(line, "Duplicate parameter '%s' in lambda", name)
		}
		seen[name] = true
	}

	prevInFunc := a.inFunc
	a.inFunc = true
	a.pushScope()
	for _, name := range l.Params {
		a.define(name, "", line)
		a.scopes[len(a.scopes)-1][name].Used = true
	}
	a.checkExpr(l.Body, line)
	a.popScope()
	a.inFunc = prevInFunc
}

func (a *Analyzer) checkFuncCall(fc *parser.FuncCall, line int) {
	for _, arg := range fc.Args {
		a.checkExpr(arg, line)
	}

	// Mark function as used
	if f, ok := a.functions[fc.Name]; ok {
		f.Used = true
		a.checkArity(fc.Name, len(fc.Args), f.Arity, line)
		return
	}

	if imp, ok := a.importedFuncs[fc.Name]; ok {
		imp.Used = true
		return
	}

	// Namespaced user module call
	if strings.Contains(fc.Name, "__") {
		parts := strings.SplitN(fc.Name, "__", 2)
		if imp, ok := a.importedFuncs["__user_module__"+parts[0]]; ok {
			imp.Used = true
			return
		}
		if imp, ok := a.importedFuncs[fc.Name]; ok {
			imp.Used = true
			return
		}
	}

	// Wildcard user module
	for key, imp := range a.importedFuncs {
		if strings.HasPrefix(key, "__user_wildcard__") {
			imp.Used = true
			return
		}
	}

	// Variable holding a function
	if a.isDefined(fc.Name) {
		a.markUsed(fc.Name)
		return
	}

	a.addError(line, "Undefined function: '%s'", fc.Name)
}

func (a *Analyzer) checkArity(name string, got, expected int, line int) {
	if expected < 0 {
		switch name {
		case "range":
			if got < 1 || got > 2 {
				a.addError(line, "'%s' expects 1 or 2 args, got %d", name, got)
			}
		case "input":
			if got > 1 {
				a.addError(line, "'%s' expects 0 or 1 args, got %d", name, got)
			}
		}
		return
	}
	if got != expected {
		a.addError(line, "'%s' expects %d args, got %d", name, expected, got)
	}
}

func (a *Analyzer) checkDictLit(dl *parser.DictLit, line int) {
	// Check for duplicate keys
	seenKeys := make(map[string]bool)
	for _, entry := range dl.Entries {
		a.checkExpr(entry.Key, line)
		a.checkExpr(entry.Value, line)
		key := exprLiteralKey(entry.Key)
		if key != "" {
			if seenKeys[key] {
				a.addWarning(line, "Duplicate dict key: %s", key)
			}
			seenKeys[key] = true
		}
	}
}

// Valid methods per type
var validMethods = map[string]map[string]int{
	"str": {
		"len": 0, "upper": 0, "lower": 0, "trim": 0,
		"split": 1, "replace": 2, "contains": 1,
		"starts_with": 1, "ends_with": 1, "type": 0,
	},
	"list": {
		"len": 0, "push": 1, "pop": 1, "contains": 1,
		"reverse": 0, "join": 1, "map": 1, "filter": 1,
		"reduce": 2, "each": 1, "type": 0,
	},
	"dict": {
		"len": 0, "keys": 0, "values": 0, "has": 1, "type": 0,
	},
	"num": {
		"type": 0,
	},
}

var allMethods map[string]bool

func init() {
	allMethods = make(map[string]bool)
	for _, methods := range validMethods {
		for name := range methods {
			allMethods[name] = true
		}
	}
}

func (a *Analyzer) checkMethodCall(mc *parser.MethodCall, line int) {
	for _, arg := range mc.Args {
		a.checkExpr(arg, line)
	}

	if !allMethods[mc.Name] {
		a.addError(line, "Unknown method '%s'", mc.Name)
		return
	}

	for _, methods := range validMethods {
		if arity, ok := methods[mc.Name]; ok {
			if len(mc.Args) != arity {
				a.addError(line, "Method '%s' expects %d args, got %d", mc.Name, arity, len(mc.Args))
			}
			return
		}
	}
}

// --- Helper functions for pattern detection ---

// isConstTrue checks if an expression is the literal `true`.
func isConstTrue(expr *parser.Expr) bool {
	if expr == nil || expr.Left == nil || len(expr.Ops) > 0 {
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
	if u.Primary == nil {
		return false
	}
	return u.Primary.Ident != nil && *u.Primary.Ident == "true"
}

// isConstFalse checks if an expression is the literal `false`.
func isConstFalse(expr *parser.Expr) bool {
	if expr == nil || expr.Left == nil || len(expr.Ops) > 0 {
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
	if u.Primary == nil {
		return false
	}
	return u.Primary.Ident != nil && *u.Primary.Ident == "false"
}

// isLiteralZero checks if a Unary node is just the number 0.
func isLiteralZero(u *parser.Unary) bool {
	if u == nil || u.Primary == nil {
		return false
	}
	return u.Primary.Number != nil && *u.Primary.Number == 0
}

// blockHasBreak checks if a block contains a break statement (0).
func blockHasBreak(block *parser.Block) bool {
	for _, stmt := range block.Statements {
		if stmt.ExprStmt != nil && stmt.ExprStmt.Expr.IsBareLiteral(0) {
			return true
		}
		// Check nested if/else blocks for break
		if stmt.IfStmt != nil {
			if blockHasBreak(stmt.IfStmt.Body) {
				return true
			}
			if stmt.IfStmt.Else != nil && blockHasBreak(stmt.IfStmt.Else) {
				return true
			}
		}
		if stmt.SwitchStmt != nil {
			for _, c := range stmt.SwitchStmt.Cases {
				if blockHasBreak(c.Body) {
					return true
				}
			}
			if stmt.SwitchStmt.Default != nil && blockHasBreak(stmt.SwitchStmt.Default) {
				return true
			}
		}
	}
	return false
}

// additionIdentName extracts a simple identifier name from an Addition node.
func additionIdentName(add *parser.Addition) string {
	if add == nil || add.Left == nil || len(add.Ops) > 0 {
		return ""
	}
	mul := add.Left
	if mul.Left == nil || len(mul.Ops) > 0 {
		return ""
	}
	u := mul.Left
	if u.Primary == nil || u.Not != nil || u.Neg != nil || u.Pos != nil {
		return ""
	}
	if u.Primary.Ident != nil {
		return *u.Primary.Ident
	}
	return ""
}

// exprLiteralKey returns a string representation of a literal expression for duplicate detection.
func exprLiteralKey(expr *parser.Expr) string {
	if expr == nil || expr.Left == nil || len(expr.Ops) > 0 {
		return ""
	}
	and := expr.Left
	if and.Left == nil || len(and.Ops) > 0 {
		return ""
	}
	cmp := and.Left
	if cmp.Op != "" || cmp.Left == nil {
		return ""
	}
	add := cmp.Left
	if len(add.Ops) > 0 || add.Left == nil {
		return ""
	}
	mul := add.Left
	if len(mul.Ops) > 0 || mul.Left == nil {
		return ""
	}
	u := mul.Left
	if u.Primary == nil {
		return ""
	}
	p := u.Primary
	if p.Number != nil {
		return fmt.Sprintf("%g", *p.Number)
	}
	if p.String != nil {
		return *p.String
	}
	if p.Ident != nil {
		return *p.Ident
	}
	return ""
}

// --- Type inference (best-effort) ---

func (a *Analyzer) inferExprType(expr *parser.Expr) string {
	if expr == nil || expr.Left == nil || len(expr.Ops) > 0 {
		return ""
	}
	return a.inferLogicalAndType(expr.Left)
}

func (a *Analyzer) inferLogicalAndType(and *parser.LogicalAnd) string {
	if and == nil || and.Left == nil {
		return ""
	}
	if len(and.Ops) > 0 {
		return ""
	}
	return a.inferComparisonType(and.Left)
}

func (a *Analyzer) inferComparisonType(cmp *parser.Comparison) string {
	if cmp == nil || cmp.Left == nil {
		return ""
	}
	if cmp.Op != "" {
		return "bool"
	}
	return a.inferAdditionType(cmp.Left)
}

func (a *Analyzer) inferAdditionType(add *parser.Addition) string {
	if add == nil || add.Left == nil {
		return ""
	}
	if len(add.Ops) > 0 {
		return ""
	}
	return a.inferMultiplicationType(add.Left)
}

func (a *Analyzer) inferMultiplicationType(mul *parser.Multiplication) string {
	if mul == nil || mul.Left == nil {
		return ""
	}
	if len(mul.Ops) > 0 {
		return "num"
	}
	return a.inferUnaryType(mul.Left)
}

func (a *Analyzer) inferUnaryType(u *parser.Unary) string {
	if u == nil {
		return ""
	}
	if u.Not != nil {
		return "bool"
	}
	if u.Neg != nil || u.Pos != nil {
		return "num"
	}
	if u.Primary != nil {
		return a.inferPrimaryType(u.Primary)
	}
	return ""
}

func (a *Analyzer) inferPrimaryType(p *parser.Primary) string {
	if p == nil {
		return ""
	}
	switch {
	case p.Number != nil:
		return "num"
	case p.String != nil:
		return "str"
	case p.ListLit != nil:
		return "list"
	case p.DictLit != nil:
		return "dict"
	case p.Lambda != nil:
		return "fn"
	case p.Ident != nil:
		switch *p.Ident {
		case "true", "false":
			return "bool"
		case "null":
			return ""
		default:
			return a.getVarType(*p.Ident)
		}
	case p.FuncCall != nil:
		switch p.FuncCall.Name {
		case "len", "num":
			return "num"
		case "type", "str":
			return "str"
		case "bool", "has":
			return "bool"
		case "keys", "values", "range":
			return "list"
		}
	}
	return ""
}
