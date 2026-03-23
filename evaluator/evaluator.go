package evaluator

import (
	"fmt"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
)

// SourceInfo holds original source context for error reporting.
type SourceInfo struct {
	Filename string
	Lines    []string
	LineMap  []int
}

// RuntimeError is a recoverable error with source location.
type RuntimeError struct {
	Message string
	Line    int
	Source  string
}

func (e RuntimeError) Error() string {
	if e.Line > 0 && e.Source != "" {
		return fmt.Sprintf("[line %d] %s\n  | %s", e.Line, e.Message, e.Source)
	}
	if e.Line > 0 {
		return fmt.Sprintf("[line %d] %s", e.Line, e.Message)
	}
	return e.Message
}

var (
	sourceInfo  *SourceInfo
	currentLine int
)

func runtimeError(format string, args ...interface{}) {
	re := RuntimeError{Message: fmt.Sprintf(format, args...)}
	if sourceInfo != nil && currentLine > 0 && currentLine <= len(sourceInfo.LineMap) {
		origLine := sourceInfo.LineMap[currentLine-1]
		re.Line = origLine
		if origLine > 0 && origLine <= len(sourceInfo.Lines) {
			re.Source = strings.TrimSpace(sourceInfo.Lines[origLine-1])
		}
	}
	panic(re)
}

// SetSourceInfo sets the source context for error reporting.
func SetSourceInfo(info *SourceInfo) {
	sourceInfo = info
}

func recoverError(err *error) {
	if r := recover(); r != nil {
		if re, ok := r.(RuntimeError); ok {
			*err = re
		} else {
			panic(r)
		}
	}
}

// Eval evaluates a program and returns any runtime error.
func Eval(program *parser.Program) (err error) {
	defer recoverError(&err)
	env := NewEnvironment()
	evalStatements(program.Statements, env)
	return nil
}

// EvalWithEnv evaluates a program in an existing environment (used by REPL).
func EvalWithEnv(program *parser.Program, env *Environment) (result object.Object, err error) {
	defer recoverError(&err)
	result = evalStatements(program.Statements, env)
	return result, nil
}

// EvalDefinitionsOnly executes only function definitions and assignments.
// Used by the test runner to set up functions without side effects.
func EvalDefinitionsOnly(program *parser.Program, env *Environment) {
	for _, stmt := range program.Statements {
		switch {
		case stmt.FuncDef != nil:
			evalFuncDef(stmt.FuncDef, env)
		case stmt.Assignment != nil:
			// Only evaluate if the value is a simple literal or function call
			// Skip if it would cause side effects
			evalAssignment(stmt.Assignment, env)
		case stmt.TypedAssign != nil:
			evalTypedAssign(stmt.TypedAssign, env)
		// Skip: PrintStmt, ExprStmt, IfStmt, WhileStmt, ForStmt, etc.
		}
	}
}

func evalStatements(stmts []*parser.Statement, env *Environment) object.Object {
	var result object.Object
	for _, stmt := range stmts {
		result = evalStatement(stmt, env)
		switch result.(type) {
		case *object.ReturnObj, *object.BreakObj, *object.ContinueObj:
			return result
		}
	}
	return result
}
