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

func runCommand() {
	filename := os.Args[1]
	program, sourceStr, result := parseFile(filename)

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

	program, _, _ := parseFile(filename)

	cg := codegen.New()
	llvmIR := cg.Generate(program)

	irFile := outputName + ".ll"
	if err := os.WriteFile(irFile, []byte(llvmIR), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Kosa: Haiwezi kuandika faili la IR: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(irFile)

	runtimePath := findRuntime()

	fmt.Printf("Inapakia Programu %s → %s...\n", filename, outputName)

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
		fmt.Fprintf(os.Stderr, "Kosa la mkusanyiko: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Imefanikiwa! Anzisha Programu: ./%s\n", outputName)
}

func printHelp() {
	fmt.Println("Sintax")
	fmt.Println()
	fmt.Println("Matumizi:")
	fmt.Println("  sintax                     REPL")
	fmt.Println("  sintax <faili.sx>          Anzisha Programu faili")
	fmt.Println("  sintax build <faili.sx>    Kusanya hadi binary")
	fmt.Println("  sintax build <f.sx> -o out Kusanya na jina maalum")
	fmt.Println("  sintax help                Onyesha msaada huu")
}

func parseFile(filename string) (*parser.Program, string, preprocessor.Result) {
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
		fmt.Fprintf(os.Stderr, "Kosa la sintaksia: %s\n", translateParseError(err.Error()))
		os.Exit(1)
	}

	return program, sourceStr, result
}

func translateParseError(msg string) string {
	r := strings.NewReplacer(
		"unexpected token", "tokeni isiyojulikana",
		"expected", "ilitarajiwa",
		"invalid input text", "maandishi batili",
		"lexer: ", "",
	)
	return r.Replace(msg)
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

	fmt.Fprintf(os.Stderr, "Kosa: Haiwezi kupata runtime.c\n")
	os.Exit(1)
	return ""
}
