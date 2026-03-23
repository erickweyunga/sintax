package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func libCommand() {
	if len(os.Args) < 3 {
		// List all stdlib modules
		stdlibDir := findStdlibDir()
		if stdlibDir == "" {
			fmt.Println("No stdlib found. Run 'make install'.")
			return
		}

		fmt.Println("Sintax Libraries:")
		fmt.Println()

		entries, _ := os.ReadDir(stdlibDir)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".sx") {
				name := strings.TrimSuffix(e.Name(), ".sx")
				desc := readFirstComment(filepath.Join(stdlibDir, e.Name()))
				fmt.Printf("  %-12s %s\n", name, desc)
			}
		}
		fmt.Println()
		fmt.Println("Use: sintax lib <name> for details")
		return
	}

	// Show specific module
	name := os.Args[2]
	stdlibDir := findStdlibDir()
	if stdlibDir == "" {
		fmt.Fprintf(os.Stderr, "Error: stdlib not found\n")
		os.Exit(1)
	}

	path := filepath.Join(stdlibDir, name+".sx")
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: module '%s' not found\n", name)
		os.Exit(1)
	}

	// Print all fn signatures
	fmt.Printf("%s\n\n", name)
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "fn ") {
			fmt.Printf("  %s\n", trimmed)
		}
	}
}

func findStdlibDir() string {
	for _, p := range []string{
		filepath.Join(SintaxHome(), "stdlib"),
		"stdlib",
		"../stdlib",
	} {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

func readFirstComment(path string) string {
	source, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- ") {
			return strings.TrimPrefix(trimmed, "-- ")
		}
	}
	return ""
}
