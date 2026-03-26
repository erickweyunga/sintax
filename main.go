package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/erickweyunga/sintax/analyzer"
	"github.com/erickweyunga/sintax/evaluator"
	"github.com/erickweyunga/sintax/lsp"
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
	case "check":
		checkCommand()
	case "test":
		testCommand()
	case "eval":
		evalCommand()
	case "lib":
		libCommand()
	case "lsp":
		lsp.Start(findStdlibDir())
	case "help", "--help", "-h":
		printHelp()
	default:
		// Default: compile and run
		runCommand()
	}
}

// runCommand compiles and runs a .sx file (with caching).
func runCommand() {
	filename := os.Args[1]
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File '%s' not found\n", filename)
		os.Exit(1)
	}
	compileAndRun(filename)
}

// evalCommand runs a file through the Go interpreter (dev/debug only).
func evalCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: sintax eval <file.sx>\n")
		os.Exit(1)
	}

	filename := os.Args[2]
	program, sourceStr, result := parseFile(filename)
	lines := strings.Split(sourceStr, "\n")

	if errors := analyzeProgram(program, result, filename, lines); len(errors) > 0 {
		printErrors(errors)
		if hasErrors(errors) {
			os.Exit(1)
		}
	}

	evaluator.SetSourceInfo(&evaluator.SourceInfo{
		Filename: filename,
		Lines:    lines,
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

func checkCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: sintax check <file.sx>\n")
		os.Exit(1)
	}

	filename := os.Args[2]
	program, sourceStr, result := parseFile(filename)
	lines := strings.Split(sourceStr, "\n")

	errors := analyzeProgram(program, result, filename, lines)
	if len(errors) > 0 {
		printErrors(errors)
		if hasErrors(errors) {
			os.Exit(1)
		}
		return
	}

	fmt.Printf("\033[32mAll checks passed\033[0m  %s\n", filename)
}

func analyzeProgram(program *parser.Program, result preprocessor.Result, filename string, lines []string) []analyzer.Error {
	return analyzer.Analyze(program, result.Imports, filename, lines, result.LineMap, findStdlibDir())
}

func printErrors(errors []analyzer.Error) {
	for _, e := range errors {
		fmt.Fprint(os.Stderr, e.Format())
	}

	errCount := 0
	warnCount := 0
	for _, e := range errors {
		if e.Level == "warning" {
			warnCount++
		} else {
			errCount++
		}
	}

	summary := ""
	if errCount > 0 {
		summary += fmt.Sprintf("%d error(s)", errCount)
	}
	if warnCount > 0 {
		if summary != "" {
			summary += ", "
		}
		summary += fmt.Sprintf("%d warning(s)", warnCount)
	}
	if errCount > 0 {
		fmt.Fprintf(os.Stderr, "\n%s. Aborted.\n", summary)
	} else {
		fmt.Fprintf(os.Stderr, "\n%s.\n", summary)
	}
}

func hasErrors(errors []analyzer.Error) bool {
	for _, e := range errors {
		if e.Level == "error" {
			return true
		}
	}
	return false
}

func printHelp() {
	fmt.Println("Sintax — a compiled programming language")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sintax <file.sx>           Compile and run")
	fmt.Println("  sintax build <file.sx>     Compile to binary")
	fmt.Println("  sintax build <f.sx> -o out Compile with custom name")
	fmt.Println("  sintax check <file.sx>     Analyze only (no run)")
	fmt.Println("  sintax test                Test all .sx files")
	fmt.Println("  sintax test <file.sx>      Test specific file")
	fmt.Println("  sintax eval <file.sx>      Interpret (dev/debug)")
	fmt.Println("  sintax lib                 List libraries")
	fmt.Println("  sintax lib <name>          Library details")
	fmt.Println("  sintax lsp                 Start LSP server")
	fmt.Println("  sintax                     REPL")
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
