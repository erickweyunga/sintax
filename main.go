package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/codegen"
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
	case "help", "--help", "-h":
		printHelp()
	default:
		runCommand()
	}
}

// sintax file.sx — run with interpreter
func runCommand() {
	filename := os.Args[1]

	program, sourceStr := parseFile(filename)
	result := preprocessor.Process(sourceStr)

	evaluator.SetSourceInfo(&evaluator.SourceInfo{
		Filename: filename,
		Lines:    strings.Split(sourceStr, "\n"),
		LineMap:  result.LineMap,
	})

	if err := evaluator.Eval(program); err != nil {
		fmt.Fprintf(os.Stderr, "Kosa: %v\n", err)
		os.Exit(1)
	}
}

// sintax build file.sx [-o name] — compile to native binary
func buildCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Matumizi: sintax build <faili.sx> [-o jina]\n")
		os.Exit(1)
	}

	filename := ""
	outputName := ""
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "-o" && i+1 < len(os.Args) {
			outputName = os.Args[i+1]
			i++
		} else {
			filename = os.Args[i]
		}
	}

	if filename == "" {
		fmt.Fprintf(os.Stderr, "Kosa: Hakuna faili la .sx\n")
		os.Exit(1)
	}

	if outputName == "" {
		base := filepath.Base(filename)
		outputName = strings.TrimSuffix(base, filepath.Ext(base))
	}

	program, _ := parseFile(filename)

	// Generate LLVM IR
	cg := codegen.New()
	llvmIR := cg.Generate(program)

	// Write IR to temp file
	irFile := outputName + ".ll"
	if err := os.WriteFile(irFile, []byte(llvmIR), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Kosa: Haiwezi kuandika faili la IR: %v\n", err)
		os.Exit(1)
	}

	// Find runtime.c
	runtimePath := findRuntime()

	// Compile with clang
	fmt.Printf("Kukusanya %s → %s...\n", filename, outputName)

	gcInclude := "/opt/homebrew/include"
	gcLib := "/opt/homebrew/lib"
	args := []string{"-O2", "-Wno-override-module", "-o", outputName, irFile, runtimePath, "-lm"}

	if _, err := os.Stat(filepath.Join(gcLib, "libgc.dylib")); err == nil {
		args = append(args, "-DSX_USE_GC", "-I"+gcInclude, "-L"+gcLib, "-lgc")
	} else if _, err := os.Stat(filepath.Join(gcLib, "libgc.a")); err == nil {
		args = append(args, "-DSX_USE_GC", "-I"+gcInclude, "-L"+gcLib, "-lgc")
	}

	cmd := exec.Command("clang", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Kosa la mkusanyiko: %v\n", err)
		os.Exit(1)
	}

	os.Remove(irFile)
	fmt.Printf("Imefanikiwa! Endesha: ./%s\n", outputName)
}

func printHelp() {
	fmt.Println("Sintax")
	fmt.Println()
	fmt.Println("Matumizi:")
	fmt.Println("  sintax                     REPL")
	fmt.Println("  sintax <faili.sx>          Endesha faili")
	fmt.Println("  sintax build <faili.sx>    Kusanya hadi binary")
	fmt.Println("  sintax build <f.sx> -o out Kusanya na jina maalum")
	fmt.Println("  sintax help                Onyesha msaada huu")
}

func parseFile(filename string) (*parser.Program, string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Kosa: Faili '%s' haipo\n", filename)
		os.Exit(1)
	}

	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Kosa: Haiwezi kusoma faili '%s'\n", filename)
		os.Exit(1)
	}

	sourceStr := string(source)
	result := preprocessor.Process(sourceStr)

	p := parser.NewParser()
	program, err := p.ParseString(filename, result.Source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Kosa la sintaksia: %v\n", err)
		os.Exit(1)
	}

	return program, sourceStr
}

func findRuntime() string {
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	paths := []string{
		filepath.Join(exeDir, "runtime", "runtime.c"),
		filepath.Join(exeDir, "..", "runtime", "runtime.c"),
		"runtime/runtime.c",
		filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "erickweyunga", "sintax", "runtime", "runtime.c"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}

	fmt.Fprintf(os.Stderr, "Kosa: Haiwezi kupata runtime.c\n")
	os.Exit(1)
	return ""
}
