package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/erickweyunga/sintax/evaluator"
	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

const (
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	dim    = "\033[2m"
	bold   = "\033[1m"
	reset  = "\033[0m"
)

type TestCase struct {
	Expr string
	Line int
}

type FileResult struct {
	Name     string
	Tests    int
	Passed   int
	Failed   int
	Duration time.Duration
	Failures []string
}

func testCommand() {
	files := []string{}

	if len(os.Args) >= 3 {
		files = append(files, os.Args[2:]...)
	} else {
		matches, _ := filepath.Glob("*.sx")
		files = append(files, matches...)
		exMatches, _ := filepath.Glob("examples/*.sx")
		files = append(files, exMatches...)
	}

	if len(files) == 0 {
		fmt.Println("No .sx files found")
		return
	}

	totalStart := time.Now()
	var results []FileResult

	for _, file := range files {
		tests := extractTests(file)
		if len(tests) == 0 {
			continue
		}
		r := runTestFile(file, tests)
		results = append(results, r)
	}

	if len(results) == 0 {
		fmt.Println("No tests found (add -- test: comments to your .sx files)")
		return
	}

	totalDuration := time.Since(totalStart)

	// Print results
	fmt.Println()
	totalTests := 0
	totalPassed := 0
	totalFailed := 0

	for _, r := range results {
		icon := green + "PASS" + reset
		if r.Failed > 0 {
			icon = red + "FAIL" + reset
		}
		fmt.Printf("  %s  %s %s(%d tests, %s)%s\n",
			icon, r.Name, dim, r.Tests, r.Duration.Round(time.Millisecond), reset)

		for _, f := range r.Failures {
			fmt.Printf("         %s%s%s\n", red, f, reset)
		}

		totalTests += r.Tests
		totalPassed += r.Passed
		totalFailed += r.Failed
	}

	fmt.Println()
	fmt.Printf("  %sFiles:%s   %d\n", dim, reset, len(results))
	fmt.Printf("  %sTests:%s   %d passed", dim, reset, totalPassed)
	if totalFailed > 0 {
		fmt.Printf(", %s%d failed%s", red, totalFailed, reset)
	}
	fmt.Println()
	fmt.Printf("  %sTime:%s    %s\n", dim, reset, totalDuration.Round(time.Millisecond))
	fmt.Println()

	if totalFailed > 0 {
		os.Exit(1)
	}
}

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

func runTestFile(filename string, tests []TestCase) FileResult {
	start := time.Now()
	r := FileResult{Name: filename, Tests: len(tests)}

	source, _ := os.ReadFile(filename)
	sourceStr := string(source)
	result := preprocessor.Process(sourceStr)

	_ = evaluator.RegisterImports(result.Imports)

	p := parser.NewParser()
	program, err := p.ParseString(filename, result.Source)
	if err != nil {
		r.Failed = len(tests)
		r.Failures = append(r.Failures, fmt.Sprintf("Syntax error: %s", err.Error()))
		r.Duration = time.Since(start)
		return r
	}

	env := evaluator.NewEnvironment()
	evaluator.EvalDefinitionsOnly(program, env)

	for _, tc := range tests {
		if runSingleTest(tc, env) {
			r.Passed++
		} else {
			r.Failed++
			r.Failures = append(r.Failures, fmt.Sprintf("line %d: %s", tc.Line, tc.Expr))
		}
	}

	r.Duration = time.Since(start)
	return r
}

func runSingleTest(tc TestCase, env *evaluator.Environment) bool {
	testSource := tc.Expr + "\n"
	result := preprocessor.Process(testSource)

	p := parser.NewParser()
	program, err := p.ParseString("test", result.Source)
	if err != nil {
		return false
	}

	val, err := evaluator.EvalWithEnv(program, env)
	if err != nil {
		return false
	}

	return val != nil && object.IsTruthy(val)
}
