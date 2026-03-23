package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/evaluator"
	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

// TestCase represents a single -- test: directive.
type TestCase struct {
	Expr string
	Line int
}

func testCommand() {
	files := []string{}

	if len(os.Args) >= 3 {
		files = append(files, os.Args[2:]...)
	} else {
		// Find all .sx files in current directory
		matches, _ := filepath.Glob("*.sx")
		files = append(files, matches...)
		// Also check examples/
		exMatches, _ := filepath.Glob("examples/*.sx")
		files = append(files, exMatches...)
	}

	if len(files) == 0 {
		fmt.Println("No .sx files found")
		return
	}

	totalTests := 0
	totalPassed := 0
	totalFailed := 0

	for _, file := range files {
		tests := extractTests(file)
		if len(tests) == 0 {
			continue
		}

		fmt.Printf("Testing %s...\n", file)

		passed, failed := runTests(file, tests)
		totalTests += len(tests)
		totalPassed += passed
		totalFailed += failed
		fmt.Println()
	}

	if totalTests == 0 {
		fmt.Println("No tests found (add -- test: comments to your .sx files)")
		return
	}

	fmt.Printf("%d tests, %d passed, %d failed\n", totalTests, totalPassed, totalFailed)
	if totalFailed > 0 {
		os.Exit(1)
	}
}

// extractTests reads a file and pulls out all -- test: directives.
func extractTests(filename string) []TestCase {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	var tests []TestCase
	for i, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if expr, ok := strings.CutPrefix(trimmed, "-- test:"); ok {
			expr = strings.TrimSpace(expr)
			if expr != "" {
				tests = append(tests, TestCase{Expr: expr, Line: i + 1})
			}
		}
	}
	return tests
}

// runTests executes the file (to define functions), then runs each test.
func runTests(filename string, tests []TestCase) (passed, failed int) {
	source, _ := os.ReadFile(filename)
	sourceStr := string(source)
	result := preprocessor.Process(sourceStr)

	_ = evaluator.RegisterImports(result.Imports)

	p := parser.NewParser()
	program, err := p.ParseString(filename, result.Source)
	if err != nil {
		fmt.Printf("  Syntax error: %s\n", err.Error())
		return 0, len(tests)
	}

	// Execute the file to define all functions/variables
	env := evaluator.NewEnvironment()
	evaluator.EvalWithEnv(program, env)

	// Run each test
	for _, tc := range tests {
		ok := runSingleTest(tc, env)
		if ok {
			passed++
		} else {
			failed++
		}
	}
	return
}

func runSingleTest(tc TestCase, env *evaluator.Environment) bool {
	// Wrap test expression as a statement
	testSource := tc.Expr + "\n"
	result := preprocessor.Process(testSource)

	p := parser.NewParser()
	program, err := p.ParseString("test", result.Source)
	if err != nil {
		fmt.Printf("  \033[31m✗\033[0m line %d: %s  →  parse error\n", tc.Line, tc.Expr)
		return false
	}

	val, err := evaluator.EvalWithEnv(program, env)
	if err != nil {
		fmt.Printf("  \033[31m✗\033[0m line %d: %s  →  %s\n", tc.Line, tc.Expr, err.Error())
		return false
	}

	// Check if result is truthy (for assertions like x == 5)
	if val == nil {
		fmt.Printf("  \033[31m✗\033[0m line %d: %s  →  null\n", tc.Line, tc.Expr)
		return false
	}

	if !object.IsTruthy(val) {
		fmt.Printf("  \033[31m✗\033[0m line %d: %s  →  %s\n", tc.Line, tc.Expr, val.Inspect())
		return false
	}

	fmt.Printf("  \033[32m✓\033[0m %s\n", tc.Expr)
	return true
}
