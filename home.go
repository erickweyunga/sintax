package main

import (
	"os"
	"path/filepath"
)

// SintaxHome returns the Sintax home directory.
// Uses SINTAX_HOME env var, defaults to ~/.sintax/
func SintaxHome() string {
	if home := os.Getenv("SINTAX_HOME"); home != "" {
		return home
	}
	userHome, _ := os.UserHomeDir()
	return filepath.Join(userHome, ".sintax")
}

// SintaxRuntime returns the path to runtime.c
func SintaxRuntime() string {
	return filepath.Join(SintaxHome(), "runtime", "runtime.c")
}
