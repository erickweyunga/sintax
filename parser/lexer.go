package parser

import "github.com/alecthomas/participle/v2/lexer"

// SintaxLexer defines the token rules for the Sintax language.
var SintaxLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Number", Pattern: `\d+(\.\d+)?`},
	{Name: "String", Pattern: `"(?:[^"\\]|\\.)*"`},
	{Name: "Print", Pattern: `>>`},
	{Name: "Op", Pattern: `\*\*|==|!=|>=|<=|\+=|\-=|\*=|/=|[+\-*/<>%]`},
	{Name: "Assign", Pattern: `=`},
	{Name: "Punct", Pattern: `[{}()\[\],;:]`},
	{Name: "Ident", Pattern: `[a-zA-Z_]\w*`},
	{Name: "Whitespace", Pattern: `\s+`},
})
