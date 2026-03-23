package preprocessor

import (
	"strings"
)

// Result holds the preprocessed source and a line mapping back to the original.
type Result struct {
	Source  string
	LineMap []int // preprocessed line number (1-based index) → original line number
}

// Process converts indentation-based Sintax source into brace-delimited form.
// Returns the processed source with newlines preserved for line tracking.
func Process(source string) Result {
	lines := strings.Split(source, "\n")
	indentStack := []int{0}
	var resultLines []string
	var lineMap []int
	lastOrigLine := 0

	inBlockComment := false

	for origLine, line := range lines {
		origLineNum := origLine + 1 // 1-based

		// Handle multiline comments --{ ... }--
		trimmed := strings.TrimSpace(line)
		if !inBlockComment && strings.HasPrefix(trimmed, "--{") {
			inBlockComment = true
			continue
		}
		if inBlockComment {
			if strings.HasSuffix(trimmed, "}--") {
				inBlockComment = false
			}
			continue
		}

		line = stripComment(line)

		if strings.TrimSpace(line) == "" {
			continue
		}

		lastOrigLine = origLineNum
		indent := countIndent(line)
		stripped := strings.TrimSpace(line)

		// Close blocks on dedent
		for len(indentStack) > 1 && indent < indentStack[len(indentStack)-1] {
			indentStack = indentStack[:len(indentStack)-1]
			resultLines = append(resultLines, "}")
			lineMap = append(lineMap, origLineNum)
		}

		// Track new indent level
		if indent > indentStack[len(indentStack)-1] {
			indentStack = append(indentStack, indent)
		}

		// Handle block-opening lines (ending with ":")
		if strings.HasSuffix(stripped, ":") {
			resultLines = append(resultLines, stripped[:len(stripped)-1]+" {")
			lineMap = append(lineMap, origLineNum)
		} else {
			resultLines = append(resultLines, stripped+" ;")
			lineMap = append(lineMap, origLineNum)
		}
	}

	// Close remaining open blocks
	for len(indentStack) > 1 {
		indentStack = indentStack[:len(indentStack)-1]
		resultLines = append(resultLines, "}")
		lineMap = append(lineMap, lastOrigLine)
	}

	return Result{
		Source:  strings.Join(resultLines, "\n"),
		LineMap: lineMap,
	}
}

func stripComment(line string) string {
	inString := false
	for i := 0; i < len(line)-1; i++ {
		if line[i] == '"' {
			inString = !inString
		}
		if !inString && line[i:i+2] == "--" {
			return line[:i]
		}
	}
	return line
}

// ProcessEscapes handles escape sequences in strings: \n, \t, \\, \"
func ProcessEscapes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case 'r':
				b.WriteByte('\r')
			case '0':
				b.WriteByte(0)
			default:
				b.WriteByte(s[i])
				b.WriteByte(s[i+1])
			}
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func countIndent(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}
