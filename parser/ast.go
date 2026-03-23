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

// FuncDef defines a function: unda (params) [returnType] name: body
// Supports both typed and untyped params:
//   unda (nambari a, neno b) nambari jumla:
//   unda (a, b) jumla:
type FuncDef struct {
	Params     []*Param `"unda" "(" ( @@ ( "," @@ )* )? ")"`
	ReturnType *string  `@( "nambari" | "tungo" | "buliani" | "safu" | "kamusi" )?`
	Name       string   `@Ident`
	Body       *Block   `@@`
}

// Param is a function parameter, optionally typed.
type Param struct {
	Type *string `( @( "nambari" | "tungo" | "buliani" | "safu" | "kamusi" )`
	Name string  `  @Ident | @Ident )`
}

// Block is a brace-delimited group of statements.
type Block struct {
	Statements []*Statement `"{" @@* "}"`
}

// IfStmt: kama condition: body sivyo: else
type IfStmt struct {
	Condition *Expr  `"kama" @@`
	Body      *Block `@@`
	Else      *Block `( "sivyo" @@ )?`
}

// PrintStmt: >> expr;
type PrintStmt struct {
	Value *Expr `">>" @@ ";"`
}

// WhileStmt: wkt condition: body
type WhileStmt struct {
	Condition *Expr  `"wkt" @@`
	Body      *Block `@@`
}

// ForStmt: kwa var katika iterable: body
type ForStmt struct {
	Var  string `"kwa" @Ident`
	Iter *Expr  `"ktk" @@`
	Body *Block `@@`
}

// SwitchStmt: chagua expr { ikiwa val: body ... _: body }
type SwitchStmt struct {
	Value   *Expr         `"chagua" @@`
	Cases   []*CaseClause `"{" @@*`
	Default *Block        `( "_" @@ )? "}"`
}

// CaseClause: ikiwa value: body
type CaseClause struct {
	Value *Expr  `"ikiwa" @@`
	Body  *Block `@@`
}

// ReturnStmt: rudisha expr;
type ReturnStmt struct {
	Value *Expr `"rudisha" @@ ";"`
}

// TypedAssign: type name = value; (e.g. nambari x = 5;)
type TypedAssign struct {
	Type  string `@( "nambari" | "tungo" | "buliani" | "safu" | "kamusi" )`
	Name  string `@Ident "="`
	Value *Expr  `@@ ";"`
}

// IndexAssign: name[index] = value;
type IndexAssign struct {
	Name  string `@Ident`
	Index *Expr  `"[" @@ "]" "="`
	Value *Expr  `@@ ";"`
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
// Precedence (low → high): au → na → comparison → addition → multiplication → unary(si) → primary

type Expr struct {
	Left *LogicalAnd `@@`
	Ops  []*OrOp     `@@*`
}

type OrOp struct {
	Right *LogicalAnd `"au" @@`
}

type LogicalAnd struct {
	Left *Comparison `@@`
	Ops  []*AndOp    `@@*`
}

type AndOp struct {
	Right *Comparison `"na" @@`
}

type Comparison struct {
	Left  *Addition `@@`
	Op    string    `( @( ">" | "<" | ">=" | "<=" | "==" | "!=" | "ktk" )`
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
	Not     *Unary   `( "si" @@`
	Primary *Primary `| @@ )`
}

type Primary struct {
	IndexAccess *IndexAccess `( @@`
	FuncCall    *FuncCall    `| @@`
	DictLit     *DictLit     `| @@`
	ListLit     *ListLit     `| @@`
	Number      *float64     `| @Number`
	String      *string      `| @String`
	Ident       *string      `| @Ident`
	SubExpr     *Expr        `| "(" @@ ")" )`
}

// IndexAccess: name[index]
type IndexAccess struct {
	Name  string `@Ident`
	Index *Expr  `"[" @@ "]"`
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
