package main

import (
	"os"
	"path/filepath"
)

func SintaxHome() string {
	if home := os.Getenv("SINTAX_HOME"); home != "" {
		return home
	}
	userHome, _ := os.UserHomeDir()
	return filepath.Join(userHome, ".sintax")
}

func SintaxRuntime() string {
	return filepath.Join(SintaxHome(), "runtime", "runtime.c")
}
