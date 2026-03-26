package parser

import "github.com/alecthomas/participle/v2/lexer"

// Program is the top-level AST node.
type Program struct {
	Statements []*Statement `@@*`
}

// Statement represents a single statement in the language.
type Statement struct {
	Pos         lexer.Position
	FuncDef     *FuncDef     `( @@`
	IfStmt      *IfStmt      `| @@`
	CatchStmt   *CatchStmt   `| @@`
	SwitchStmt  *SwitchStmt  `| @@`
	WhileStmt   *WhileStmt   `| @@`
	ForStmt     *ForStmt     `| @@`
	PrintStmt    *PrintStmt    `| @@`
	ReturnStmt   *ReturnStmt  `| @@`
	TypedAssign  *TypedAssign `| @@`
	IndexAssign    *IndexAssign    `| @@`
	CompoundAssign *CompoundAssign `| @@`
	Assignment     *Assignment     `| @@`
	ExprStmt    *ExprStmt    `| @@ )`
}

// FuncDef defines a function: [pub] fn (params) [returnTypes] name: body
// Return types can be a single type, void, or a union: str | num
type FuncDef struct {
	Pub        bool     `@"pub"?`
	Params     []*Param `"fn" "(" ( @@ ( "," @@ )* )? ")"`
	ReturnType *string  `( @( "num" | "str" | "bool" | "list" | "dict" | "fn" | "void" )`
	MoreTypes  []string `  ( "|" @( "num" | "str" | "bool" | "list" | "dict" | "fn" ) )* )?`
	Name       string   `@Ident`
	Body       *Block   `@@`
}

// Param is a function parameter, optionally typed.
type Param struct {
	Type *string `( @( "num" | "str" | "bool" | "list" | "dict" )`
	Name string  `  @Ident | @Ident )`
}

// ReturnTypes returns the full list of return types for a FuncDef.
func (fd *FuncDef) ReturnTypes() []string {
	if fd.ReturnType == nil {
		return nil
	}
	types := []string{*fd.ReturnType}
	types = append(types, fd.MoreTypes...)
	return types
}

// IsVoid returns true if the function is declared void.
func (fd *FuncDef) IsVoid() bool {
	return fd.ReturnType != nil && *fd.ReturnType == "void"
}

// Block is a brace-delimited group of statements.
type Block struct {
	Statements []*Statement `"{" @@* "}"`
}

// IfStmt: if condition: body else: else
type IfStmt struct {
	Condition *Expr  `"if" @@`
	Body      *Block `@@`
	Else      *Block `( "else" @@ )?`
}

// CatchStmt: catch name = expr: body
// Evaluates expr, assigns to name. If the value is an error, runs body.
type CatchStmt struct {
	Name  string `"catch" @Ident "="`
	Value *Expr  `@@`
	Body  *Block `@@`
}

// PrintStmt: >> expr;
type PrintStmt struct {
	Value *Expr `">>" @@ ";"`
}

// WhileStmt: while condition: body
type WhileStmt struct {
	Condition *Expr  `"while" @@`
	Body      *Block `@@`
}

// ForStmt: for var in iterable: body
type ForStmt struct {
	Var  string `"for" @Ident`
	Iter *Expr  `"in" @@`
	Body *Block `@@`
}

// SwitchStmt: match expr { case val: body ... _: body }
type SwitchStmt struct {
	Value   *Expr         `"match" @@`
	Cases   []*CaseClause `"{" @@*`
	Default *Block        `( "_" @@ )? "}"`
}

// CaseClause: case value: body
type CaseClause struct {
	Value *Expr  `"case" @@`
	Body  *Block `@@`
}

// ReturnStmt: return expr;
type ReturnStmt struct {
	Value *Expr `"return" @@ ";"`
}

// TypedAssign: type name = value; (e.g. num x = 5;)
type TypedAssign struct {
	Type  string `@( "num" | "str" | "bool" | "list" | "dict" )`
	Name  string `@Ident "="`
	Value *Expr  `@@ ";"`
}

// IndexAssign: name[index] = value; or name[i][j] = value;
type IndexAssign struct {
	Name    string      `@Ident`
	Indices []*IndexOp  `@@+ "="`
	Value   *Expr       `@@ ";"`
}

// CompoundAssign: name += value; name -= value; etc.
type CompoundAssign struct {
	Name string `@Ident`
	Op   string `@( "+=" | "-=" | "*=" | "/=" )`
	Value *Expr `@@ ";"`
}

// Assignment: name = value;
type Assignment struct {
	Name  string `@Ident "="`
	Value *Expr  `@@ ";"`
}

// ExprStmt is an expression used as a statement.
type ExprStmt struct {
	Expr *Expr `@@ ";"`
}

// Expression hierarchy for operator precedence.
// Precedence (low → high): or → and → comparison → addition → multiplication → unary(not) → primary

type Expr struct {
	Left *LogicalAnd `@@`
	Ops  []*OrOp     `@@*`
}

type OrOp struct {
	Right *LogicalAnd `"or" @@`
}

type LogicalAnd struct {
	Left *Comparison `@@`
	Ops  []*AndOp    `@@*`
}

type AndOp struct {
	Right *Comparison `"and" @@`
}

type Comparison struct {
	Left  *Addition `@@`
	Op    string    `( @( ">" | "<" | ">=" | "<=" | "==" | "!=" | "in" )`
	Right *Addition `  @@ )?`
}

type Addition struct {
	Left *Multiplication `@@`
	Ops  []*AddOp        `@@*`
}

type AddOp struct {
	Op    string          `@( "+" | "-" )`
	Right *Multiplication `@@`
}

type Multiplication struct {
	Left *Unary   `@@`
	Ops  []*MulOp `@@*`
}

type MulOp struct {
	Op    string `@( "**" | "*" | "/" | "%" )`
	Right *Unary `@@`
}

type Unary struct {
	Not     *Unary   `( "not" @@`
	Neg     *Unary   `| "-" @@`
	Pos     *Unary   `| "+" @@`
	Primary *Primary `| @@ )`
}

// Primary is the base expression. After parsing the base value,
// a chain of suffix operations (indexing and method calls) is applied.
type Primary struct {
	Lambda   *Lambda    `( @@`
	FuncCall *FuncCall  `| @@`
	DictLit  *DictLit   `| @@`
	ListLit  *ListLit   `| @@`
	Number   *float64   `| @Number`
	String   *string    `| @String`
	Ident    *string    `| @Ident`
	SubExpr  *Expr      `| "(" @@ ")" )`
	Suffix   []*Suffix  `@@*`
}

// Suffix is a postfix operation: either [index] or .method(args).
type Suffix struct {
	Index  *IndexOp    `( @@`
	Method *MethodCall `| @@ )`
}

// IndexOp: [expr]
type IndexOp struct {
	Index *Expr `"[" @@ "]"`
}

// Lambda: fn(params) -> expr
type Lambda struct {
	Params []string `"fn" "(" ( @Ident ( "," @Ident )* )? ")" "->"`
	Body   *Expr    `@@`
}

// MethodCall: .name(args)
type MethodCall struct {
	Name string  `"." @Ident "("`
	Args []*Expr `( @@ ( "," @@ )* )? ")"`
}

// FuncCall: name(args)
type FuncCall struct {
	Name string  `@Ident "("`
	Args []*Expr `( @@ ( "," @@ )* )? ")"`
}

// DictLit: {"key": value, ...}
type DictLit struct {
	Entries []*DictEntry `"{" ( @@ ( "," @@ )* )? "}"`
}

// DictEntry: key: value
type DictEntry struct {
	Key   *Expr `@@ ":"`
	Value *Expr `@@`
}

// ListLit: [elem, elem, ...]
type ListLit struct {
	Elements []*Expr `"[" ( @@ ( "," @@ )* )? "]"`
}

// IsBareLiteral checks if an expression is just a bare number literal (e.g. 0 or 1).
func (e *Expr) IsBareLiteral(val float64) bool {
	if e.Left == nil || len(e.Ops) > 0 {
		return false
	}
	and := e.Left
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
