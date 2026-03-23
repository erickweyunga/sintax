package repl

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/erickweyunga/sintax/evaluator"
	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

const prompt = ">>> "
const blockPrompt = "... "

// Start launches the interactive Sintax REPL.
func Start() {
	fmt.Println("Sintax REPL v0.1.0")
	fmt.Println("'toka' kutoka.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	env := evaluator.NewEnvironment()
	p := parser.NewParser()

	for {
		fmt.Print(prompt)
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		if strings.TrimSpace(line) == "toka" {
			fmt.Println("Kwaheri!")
			break
		}

		// Accumulate block input when line ends with ":"
		input := line
		if strings.HasSuffix(strings.TrimSpace(line), ":") {
			input = readBlock(scanner, line)
		}

		if strings.TrimSpace(input) == "" {
			continue
		}

		processed := preprocessor.Process(input)

		program, err := p.ParseString("repl", processed.Source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Kosa la sintaksia: %v\n", err)
			continue
		}

		val, err := evaluator.EvalWithEnv(program, env)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Kosa: %v\n", err)
			continue
		}

		if val != nil {
			if _, ok := val.(*object.NullObj); !ok {
				fmt.Println(val.Inspect())
			}
		}
	}
}

// readBlock collects indented lines for a block until an empty line is entered.
func readBlock(scanner *bufio.Scanner, firstLine string) string {
	lines := []string{firstLine}
	for {
		fmt.Print(blockPrompt)
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			break
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
