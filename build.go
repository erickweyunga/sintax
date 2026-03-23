package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/codegen"
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
	candidates := []string{}

	// Relative to executable (handles symlinks)
	exe, err := os.Executable()
	if err == nil {
		real, err := filepath.EvalSymlinks(exe)
		if err == nil {
			exe = real
		}
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "runtime", "runtime.c"),
			filepath.Join(exeDir, "..", "runtime", "runtime.c"),
		)
	}

	// Relative to working directory (and parent)
	candidates = append(candidates,
		"runtime/runtime.c",
		"../runtime/runtime.c",
	)

	// GOPATH
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	candidates = append(candidates,
		filepath.Join(gopath, "src", "github.com", "erickweyunga", "sintax", "runtime", "runtime.c"),
		filepath.Join(gopath, "pkg", "mod", "github.com", "erickweyunga", "sintax@*", "runtime", "runtime.c"),
	)

	for _, p := range candidates {
		// Handle glob patterns
		if matches, _ := filepath.Glob(p); len(matches) > 0 {
			abs, _ := filepath.Abs(matches[0])
			return abs
		}
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}

	fmt.Fprintf(os.Stderr, "Error: Cannot find runtime.c\n")
	fmt.Fprintf(os.Stderr, "Make sure you're in the sintax project directory or runtime.c is installed.\n")
	os.Exit(1)
	return ""
}
