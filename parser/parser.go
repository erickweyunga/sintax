package parser

import "github.com/alecthomas/participle/v2"

// NewParser creates a new Sintax parser.
func NewParser() *participle.Parser[Program] {
	return participle.MustBuild[Program](
		participle.Lexer(SintaxLexer),
		participle.Elide("Whitespace"),
	)
}
