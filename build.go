package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/codegen"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

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

	program, sourceStr, result := parseFile(filename)

	// Analyze before compiling
	lines := strings.Split(sourceStr, "\n")
	if errors := analyzeProgram(program, result, filename, lines); len(errors) > 0 {
		printErrors(errors)
		if hasErrors(errors) {
			os.Exit(1)
		}
	}

	fmt.Printf("\033[2mCompiling %s → %s\033[0m\n", filename, outputName)

	cg := codegen.New()
	compileImports(cg, result.Imports)
	fmt.Printf("  \033[2mGenerating IR...\033[0m\n")
	llvmIR := cg.Generate(program)

	irFile := outputName + ".ll"
	if err := os.WriteFile(irFile, []byte(llvmIR), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot write IR file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(irFile)

	runtimePath := findRuntime()
	runtimeDir := filepath.Dir(runtimePath)

	// Find all .c files in runtime directory
	cFiles := []string{runtimePath}
	if entries, err := os.ReadDir(runtimeDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".c") && e.Name() != "runtime.c" {
				cFiles = append(cFiles, filepath.Join(runtimeDir, e.Name()))
			}
		}
	}

	fmt.Printf("  \033[2mLinking native binary...\033[0m\n")

	args := []string{"-O2", "-Wno-override-module", "-o", outputName, irFile}
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done! Run: ./%s\n", outputName)
}

func findRuntime() string {
	paths := []string{
		SintaxRuntime(), // ~/.sintax/runtime/runtime.c
		"runtime/runtime.c",
		"../runtime/runtime.c",
	}

	// Also check relative to executable
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		dir := filepath.Dir(exe)
		paths = append(paths,
			filepath.Join(dir, "runtime", "runtime.c"),
			filepath.Join(dir, "..", "runtime", "runtime.c"),
		)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}

	fmt.Fprintf(os.Stderr, "Error: Cannot find runtime.c\n")
	fmt.Fprintf(os.Stderr, "Run 'make install' to set up ~/.sintax/\n")
	os.Exit(1)
	return ""
}

// compileImports parses imported .sx modules and compiles their functions
// into the codegen module so they're available to the main program.
func compileImports(cg *codegen.CodeGen, imports []preprocessor.Import) {
	for _, imp := range imports {
		modName := imp.Module

		// Resolve stdlib path
		var filePath string
		if strings.HasPrefix(modName, "std/") {
			stdName := strings.TrimPrefix(modName, "std/")
			filePath = findStdlibFile(stdName)
			modName = stdName // strip std/ for the function prefix
		} else if strings.HasSuffix(modName, ".sx") {
			filePath = modName
			modName = strings.TrimSuffix(filepath.Base(modName), ".sx")
		} else {
			continue
		}

		if filePath == "" {
			continue
		}

		source, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		result := preprocessor.Process(string(source))
		p := parser.NewParser()
		program, err := p.ParseString(filePath, result.Source)
		if err != nil {
			continue
		}

		wildcard := imp.Function == "*"
		fmt.Printf("  \033[2mCompiling module %s\033[0m\n", modName)
		cg.CompileModule(program, modName, wildcard)
	}
}

func findStdlibFile(name string) string {
	stdlibDir := findStdlibDir()
	if stdlibDir == "" {
		return ""
	}
	p := filepath.Join(stdlibDir, name+".sx")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
