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

func compileAndRun(filename string) {
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: File '%s' not found\n", filename)
		os.Exit(1)
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot create %s directory\n", cacheDir)
		os.Exit(1)
	}

	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	hash := fmt.Sprintf("%x", sha256.Sum256(source))[:12]
	binaryPath := filepath.Join(cacheDir, baseName+"_"+hash)

	if isCached(binaryPath, filename) {
		execBinary(binaryPath)
		return
	}

	program, sourceStr, result := parseFile(filename)

	srcLines := strings.Split(sourceStr, "\n")
	if errors := analyzeProgram(program, result, filename, srcLines); len(errors) > 0 {
		printErrors(errors)
		if hasErrors(errors) {
			os.Exit(1)
		}
	}

	cg := codegen.New()
	compileImports(cg, result.Imports, false)
	llvmIR := cg.Generate(program)

	irFile := binaryPath + ".ll"
	if err := os.WriteFile(irFile, []byte(llvmIR), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Cannot write IR file\n")
		os.Exit(1)
	}
	defer os.Remove(irFile)

	if err := clangCompile(irFile, binaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation error: %v\n", err)
		os.Exit(1)
	}

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
