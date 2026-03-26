package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/erickweyunga/sintax/codegen"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

const (
	green = "\033[32m"
	red   = "\033[31m"
	dim   = "\033[2m"
	reset = "\033[0m"
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

// runTestFile generates a test program from the source + assertions,
// compiles it to a native binary, runs it, and parses PASS/FAIL output.
func runTestFile(filename string, tests []TestCase) FileResult {
	start := time.Now()
	r := FileResult{Name: filename, Tests: len(tests)}

	source, _ := os.ReadFile(filename)
	sourceStr := string(source)
	result := preprocessor.Process(sourceStr)

	// Build the test program: original source + compiled assertions
	testSource := buildTestProgram(sourceStr, tests, result.Imports)
	testResult := preprocessor.Process(testSource)

	p := parser.NewParser()
	program, err := p.ParseString(filename, testResult.Source)
	if err != nil {
		r.Failed = len(tests)
		r.Failures = append(r.Failures, fmt.Sprintf("Syntax error: %s", err.Error()))
		r.Duration = time.Since(start)
		return r
	}

	// Compile to binary
	cg := codegen.New()
	compileImports(cg, testResult.Imports)
	llvmIR := cg.Generate(program)

	tmpDir, _ := os.MkdirTemp("", "sx-test-*")
	defer os.RemoveAll(tmpDir)

	irFile := filepath.Join(tmpDir, "test.ll")
	binFile := filepath.Join(tmpDir, "test")
	os.WriteFile(irFile, []byte(llvmIR), 0644)

	if err := compileToNative(irFile, binFile); err != nil {
		r.Failed = len(tests)
		r.Failures = append(r.Failures, fmt.Sprintf("Compile error: %s", err.Error()))
		r.Duration = time.Since(start)
		return r
	}

	// Run and parse output
	runCmd := exec.Command(binFile)
	output, runErr := runCmd.CombinedOutput()
	outputStr := string(output)

	resultMap := make(map[int]bool)
	for _, line := range strings.Split(strings.TrimSpace(outputStr), "\n") {
		line = strings.TrimSpace(line)
		var lineNum int
		if strings.HasPrefix(line, "PASS ") {
			fmt.Sscanf(line, "PASS %d", &lineNum)
			resultMap[lineNum] = true
		} else if strings.HasPrefix(line, "FAIL ") {
			fmt.Sscanf(line, "FAIL %d", &lineNum)
			resultMap[lineNum] = false
		}
	}

	for _, tc := range tests {
		if passed, ok := resultMap[tc.Line]; ok && passed {
			r.Passed++
		} else {
			r.Failed++
			reason := fmt.Sprintf("line %d: %s", tc.Line, tc.Expr)
			if runErr != nil && len(resultMap) == 0 {
				reason += " (runtime crash)"
			}
			r.Failures = append(r.Failures, reason)
		}
	}

	r.Duration = time.Since(start)
	return r
}

// buildTestProgram appends test assertions to the source code.
// Each -- test: expr becomes: if expr: print("PASS N") else: print("FAIL N")
func buildTestProgram(source string, tests []TestCase, imports []preprocessor.Import) string {
	var b strings.Builder
	b.WriteString(source)
	b.WriteString("\n")

	for _, tc := range tests {
		// Rewrite namespace calls in test expressions
		expr := preprocessor.RewriteLine(tc.Expr, imports)
		b.WriteString(fmt.Sprintf("if %s:\n", expr))
		b.WriteString(fmt.Sprintf("    print(\"PASS %d\")\n", tc.Line))
		b.WriteString("else:\n")
		b.WriteString(fmt.Sprintf("    print(\"FAIL %d\")\n", tc.Line))
	}

	return b.String()
}

// compileToNative compiles an LLVM IR file to a native binary.
func compileToNative(irFile, binFile string) error {
	runtimePath := findRuntime()
	runtimeDir := filepath.Dir(runtimePath)

	cFiles := []string{runtimePath}
	if entries, _ := os.ReadDir(runtimeDir); entries != nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".c") && e.Name() != "runtime.c" {
				cFiles = append(cFiles, filepath.Join(runtimeDir, e.Name()))
			}
		}
	}

	args := []string{"-O2", "-Wno-override-module", "-o", binFile, irFile}
	args = append(args, cFiles...)
	args = append(args, "-lm")

	for _, gcLib := range []string{"/opt/homebrew/lib", "/usr/local/lib", "/usr/lib"} {
		gcInclude := strings.Replace(gcLib, "/lib", "/include", 1)
		for _, ext := range []string{"libgc.dylib", "libgc.a", "libgc.so"} {
			if _, err := os.Stat(filepath.Join(gcLib, ext)); err == nil {
				args = append(args, "-DSX_USE_GC", "-I"+gcInclude, "-L"+gcLib, "-lgc")
				goto foundGC
			}
		}
	}
foundGC:

	cmd := exec.Command("clang", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}
