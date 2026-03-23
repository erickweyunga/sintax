package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/erickweyunga/sintax/evaluator"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
	"github.com/erickweyunga/sintax/repl"
)

func main() {
	if len(os.Args) < 2 {
		repl.Start()
		return
	}

	switch os.Args[1] {
	case "build":
		buildCommand()
	case "lib":
		libCommand()
	case "help", "--help", "-h":
		printHelp()
	default:
		runCommand()
	}
}

func runCommand() {
	filename := os.Args[1]
	program, sourceStr, result := parseFile(filename)

	evaluator.SetSourceInfo(&evaluator.SourceInfo{
		Filename: filename,
		Lines:    strings.Split(sourceStr, "\n"),
		LineMap:  result.LineMap,
	})

	if err := evaluator.RegisterImports(result.Imports); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := evaluator.Eval(program); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Sintax")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sintax                     REPL")
	fmt.Println("  sintax <file.sx>           Run a program file")
	fmt.Println("  sintax build <file.sx>     Compile to binary")
	fmt.Println("  sintax build <f.sx> -o out Compile with custom name")
	fmt.Println("  sintax lib                 List libraries")
	fmt.Println("  sintax lib <name>          Library details")
	fmt.Println("  sintax help                Show help")
}

func parseFile(filename string) (*parser.Program, string, preprocessor.Result) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File '%s' not found\n", filename)
		os.Exit(1)
	}

	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot read file '%s'\n", filename)
		os.Exit(1)
	}

	sourceStr := string(source)
	result := preprocessor.Process(sourceStr)

	p := parser.NewParser()
	program, err := p.ParseString(filename, result.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Syntax error: %s\n", err.Error())
		os.Exit(1)
	}

	return program, sourceStr, result
}
