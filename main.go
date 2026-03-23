package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/codegen"
	"github.com/erickweyunga/sintax/stdlib"
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

func buildCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: sintax build <file.sx> [-o name]\n")
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
		fmt.Fprintf(os.Stderr, "Error: No .sx file specified\n")
		os.Exit(1)
	}

	if outputName == "" {
		base := filepath.Base(filename)
		outputName = strings.TrimSuffix(base, filepath.Ext(base))
	}

	program, _, _ := parseFile(filename)

	cg := codegen.New()
	llvmIR := cg.Generate(program)

	irFile := outputName + ".ll"
	if err := os.WriteFile(irFile, []byte(llvmIR), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot write IR file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(irFile)

	runtimePath := findRuntime()

	fmt.Printf("Compiling %s → %s...\n", filename, outputName)

	args := []string{"-O2", "-Wno-override-module", "-o", outputName, irFile, runtimePath, "-lm"}

	// Check for Boehm GC in common locations
	for _, gcLib := range []string{"/opt/homebrew/lib", "/usr/local/lib", "/usr/lib"} {
		gcInclude := strings.Replace(gcLib, "/lib", "/include", 1)
		if _, err := os.Stat(filepath.Join(gcLib, "libgc.dylib")); err == nil {
			args = append(args, "-DSX_USE_GC", "-I"+gcInclude, "-L"+gcLib, "-lgc")
			break
		}
		if _, err := os.Stat(filepath.Join(gcLib, "libgc.a")); err == nil {
			args = append(args, "-DSX_USE_GC", "-I"+gcInclude, "-L"+gcLib, "-lgc")
			break
		}
		if _, err := os.Stat(filepath.Join(gcLib, "libgc.so")); err == nil {
			args = append(args, "-DSX_USE_GC", "-I"+gcInclude, "-L"+gcLib, "-lgc")
			break
		}
	}

	cmd := exec.Command("clang", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done! Run: ./%s\n", outputName)
}

func libCommand() {
	if len(os.Args) < 3 {
		// List all modules
		fmt.Println("Sintax Libraries:")
		fmt.Println()
		for _, name := range stdlib.ListModules() {
			mod := stdlib.Registry[name]
			fmt.Printf("  %-12s %s\n", name, mod.Desc)
		}
		fmt.Println()
		fmt.Println("Use: sintax lib <name> for details")
		return
	}

	// Describe specific module
	desc, err := stdlib.Describe(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Print(desc)
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

func findRuntime() string {
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		for _, rel := range []string{"runtime/runtime.c", "../runtime/runtime.c"} {
			p := filepath.Join(exeDir, rel)
			if _, err := os.Stat(p); err == nil {
				abs, _ := filepath.Abs(p)
				return abs
			}
		}
	}

	if _, err := os.Stat("runtime/runtime.c"); err == nil {
		abs, _ := filepath.Abs("runtime/runtime.c")
		return abs
	}

	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		p := filepath.Join(gopath, "src", "github.com", "erickweyunga", "sintax", "runtime", "runtime.c")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	fmt.Fprintf(os.Stderr, "Error: Cannot find runtime.c\n")
	os.Exit(1)
	return ""
}
