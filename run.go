package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/codegen"
)

const cacheDir = ".sintax"

func runCompiledCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: sintax run <file.sx>\n")
		os.Exit(1)
	}

	filename := os.Args[2]

	// Read source
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: File '%s' not found\n", filename)
		os.Exit(1)
	}

	// Ensure .sintax/ exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot create %s directory\n", cacheDir)
		os.Exit(1)
	}

	// Binary name: .sintax/<hash>
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	hash := fmt.Sprintf("%x", sha256.Sum256(source))[:12]
	binaryPath := filepath.Join(cacheDir, baseName+"_"+hash)

	// Check cache — if binary exists and is newer than source, reuse it
	if isCached(binaryPath, filename) {
		execBinary(binaryPath)
		return
	}

	// Compile
	program, _, _ := parseFile(filename)
	cg := codegen.New()
	llvmIR := cg.Generate(program)

	irFile := binaryPath + ".ll"
	if err := os.WriteFile(irFile, []byte(llvmIR), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot write IR file\n")
		os.Exit(1)
	}

	runtimePath := findRuntime()

	args := []string{"-O2", "-Wno-override-module", "-o", binaryPath, irFile, runtimePath, "-lm"}

	// Check for Boehm GC
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
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

	// Clean up IR
	os.Remove(irFile)

	// Run the binary
	execBinary(binaryPath)
}

func isCached(binaryPath, sourcePath string) bool {
	binInfo, err := os.Stat(binaryPath)
	if err != nil {
		return false
	}
	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		return false
	}
	return binInfo.ModTime().After(srcInfo.ModTime())
}

func execBinary(path string) {
	cmd := exec.Command(path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
