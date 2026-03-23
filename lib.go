package main

import (
	"fmt"
	"os"

	"github.com/erickweyunga/sintax/stdlib"
)

func libCommand() {
	if len(os.Args) < 3 {
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

	desc, err := stdlib.Describe(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Print(desc)
}
