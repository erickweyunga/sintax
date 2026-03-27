package parser

import "github.com/alecthomas/participle/v2/lexer"

type Program struct {
	Statements []*Statement `@@*`
}

type Statement struct {
	Pos         lexer.Position
	FuncDef     *FuncDef     `( @@`
	IfStmt      *IfStmt      `| @@`
	CatchStmt   *CatchStmt   `| @@`
	SwitchStmt  *SwitchStmt  `| @@`
	WhileStmt   *WhileStmt   `| @@`
	ForStmt     *ForStmt     `| @@`
	PrintStmt      *PrintStmt      `| @@`
	ReturnStmt     *ReturnStmt     `| @@`
	TypedAssign    *TypedAssign    `| @@`
	IndexAssign    *IndexAssign    `| @@`
	CompoundAssign *CompoundAssign `| @@`
	Assignment     *Assignment     `| @@`
	ExprStmt    *ExprStmt    `| @@ )`
}

type FuncDef struct {
	Pub        bool     `@"pub"?`
	Params     []*Param `"fn" "(" ( @@ ( "," @@ )* )? ")"`
	ReturnType *string  `( @( "num" | "str" | "bool" | "list" | "dict" | "fn" | "void" )`
	MoreTypes  []string `  ( "|" @( "num" | "str" | "bool" | "list" | "dict" | "fn" ) )* )?`
	Name       string   `@Ident`
	Body       *Block   `@@`
}

type Param struct {
	Type       *string  `( @( "num" | "str" | "bool" | "list" | "dict" )`
	TypedName  *string  `  @Ident`
	DefaultNum *float64 `  ( "=" ( @Number`
	DefaultStr *string  `         | @String`
	DefaultBool *string `         | @( "true" | "false" | "null" ) ) )?`
	Name       string   `| @Ident )`
}

func (p *Param) GetName() string {
	if p.TypedName != nil {
		return *p.TypedName
	}
	return p.Name
}

func (p *Param) HasDefault() bool {
	return p.DefaultNum != nil || p.DefaultStr != nil || p.DefaultBool != nil
}

func (fd *FuncDef) ReturnTypes() []string {
	if fd.ReturnType == nil {
		return nil
	}
	types := []string{*fd.ReturnType}
	types = append(types, fd.MoreTypes...)
	return types
}

func (fd *FuncDef) IsVoid() bool {
	return fd.ReturnType != nil && *fd.ReturnType == "void"
}

type Block struct {
	Statements []*Statement `"{" @@* "}"`
}

type IfStmt struct {
	Condition *Expr  `"if" @@`
	Body      *Block `@@`
	Else      *Block `( "else" @@ )?`
}

type CatchStmt struct {
	Name  string `"catch" @Ident "="`
	Value *Expr  `@@`
	Body  *Block `@@`
}

type PrintStmt struct {
	Value *Expr `">>" @@ ";"`
}

type WhileStmt struct {
	Condition *Expr  `"while" @@`
	Body      *Block `@@`
}

type ForStmt struct {
	Var      string  `"for" @Ident`
	ValueVar *string `( "," @Ident )?`
	Iter     *Expr   `"in" @@`
	Body     *Block  `@@`
}

type SwitchStmt struct {
	Value   *Expr         `"match" @@`
	Cases   []*CaseClause `"{" @@*`
	Default *Block        `( "_" @@ )? "}"`
}

type CaseClause struct {
	Value *Expr  `"case" @@`
	Body  *Block `@@`
}

type ReturnStmt struct {
	Value *Expr `"return" @@ ";"`
}

type TypedAssign struct {
	Const bool   `@"const"?`
	Type  string `@( "num" | "str" | "bool" | "list" | "dict" )`
	Name  string `@Ident "="`
	Value *Expr  `@@ ";"`
}

type IndexAssign struct {
	Name    string      `@Ident`
	Indices []*IndexOp  `@@+ "="`
	Value   *Expr       `@@ ";"`
}

type CompoundAssign struct {
	Name string `@Ident`
	Op   string `@( "+=" | "-=" | "*=" | "/=" )`
	Value *Expr `@@ ";"`
}

type Assignment struct {
	Name  string `@Ident "="`
	Value *Expr  `@@ ";"`
}

type ExprStmt struct {
	Expr *Expr `@@ ";"`
}

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

type Suffix struct {
	Index  *IndexOp    `( @@`
	Method *MethodCall `| @@ )`
}

type IndexOp struct {
	Index *Expr `"[" @@ "]"`
}

type Lambda struct {
	Params []string `"fn" "(" ( @Ident ( "," @Ident )* )? ")" "->"`
	Body   *Expr    `@@`
}

type MethodCall struct {
	Name string  `"." @Ident "("`
	Args []*Expr `( @@ ( "," @@ )* )? ")"`
}

type FuncCall struct {
	Name string  `@Ident "("`
	Args []*Expr `( @@ ( "," @@ )* )? ")"`
}

type DictLit struct {
	Entries []*DictEntry `"{" ( @@ ( "," @@ )* )? "}"`
}

type DictEntry struct {
	Key   *Expr `@@ ":"`
	Value *Expr `@@`
}

type ListLit struct {
	Elements []*Expr `"[" ( @@ ( "," @@ )* )? "]"`
}

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
