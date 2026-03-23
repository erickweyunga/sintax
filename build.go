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
