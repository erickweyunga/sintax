package parser

import "github.com/alecthomas/participle/v2/lexer"

// SintaxLexer defines the token rules for the Sintax language.
var SintaxLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"Number", `\d+(\.\d+)?`},
	{"String", `"(?:[^"\\]|\\.)*"`},
	{"Print", `>>`},
	{"Op", `\*\*|==|!=|>=|<=|\+=|\-=|\*=|/=|[+\-*/<>%]`},
	{"Assign", `=`},
	{"Punct", `[{}()\[\],;:]`},
	{"Ident", `[a-zA-Z_]\w*`},
	{"Whitespace", `\s+`},
})
