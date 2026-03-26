package preprocessor

import (
	"path/filepath"
	"strings"
)

// Import represents a use directive.
type Import struct {
	Module   string // e.g. "math"
	Function string // e.g. "sqrt" or ""
}

// RewriteLine rewrites namespace calls in a line using the given imports.
// This is used by the test runner to rewrite test expressions.
func RewriteLine(line string, imports []Import) string {
	for _, imp := range imports {
		if imp.Function == "" {
			modName := imp.Module
			if strings.HasPrefix(modName, "std/") {
				modName = strings.TrimPrefix(modName, "std/")
			}
			if strings.HasSuffix(modName, ".sx") {
				modName = strings.TrimSuffix(filepath.Base(modName), ".sx")
			}
			line = rewriteNamespaceCalls(line, modName)
		}
	}
	return line
}

// Result holds the preprocessed source and a line mapping back to the original.
type Result struct {
	Source  string
	LineMap []int // preprocessed line number (1-based index) → original line number
	Imports []Import
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
	var imports []Import

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

		// Rewrite namespace calls: math/sqrt( → math__sqrt(
		for _, imp := range imports {
			if imp.Function == "" {
				modName := imp.Module
				// std/math → math (strip std/ prefix)
				if strings.HasPrefix(modName, "std/") {
					modName = strings.TrimPrefix(modName, "std/")
				}
				// myfile.sx → myfile (strip .sx extension)
				if strings.HasSuffix(modName, ".sx") {
					modName = strings.TrimSuffix(filepath.Base(modName), ".sx")
				}
				line = rewriteNamespaceCalls(line, modName)
			}
		}

		trimmed = strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Handle use imports
		if strings.HasPrefix(trimmed, "use ") {
			imp := parseUse(trimmed)
			if imp != nil {
				imports = append(imports, *imp)
			}
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
		Imports: imports,
	}
}

// parseUse parses a use directive.
// use "math"       → Module: "math", Function: ""
// use "math/sqrt"  → Module: "math", Function: "sqrt"
// rewriteNamespaceCalls replaces module/func( with module__func( in a line.
func rewriteNamespaceCalls(line, module string) string {
	prefix := module + "/"
	for {
		idx := strings.Index(line, prefix)
		if idx == -1 {
			break
		}
		// Replace the / with __
		line = line[:idx] + module + "__" + line[idx+len(prefix):]
	}
	return line
}

func parseUse(line string) *Import {
	// Extract the quoted string
	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start == -1 || end <= start {
		return nil
	}
	path := line[start+1 : end]

	// Handle std/ prefix: use "std/math" → Module: "std/math"
	// Handle std/math/sqrt → Module: "std/math", Function: "sqrt"
	if strings.HasPrefix(path, "std/") {
		rest := strings.TrimPrefix(path, "std/")
		if idx := strings.Index(rest, "/"); idx != -1 {
			return &Import{
				Module:   "std/" + rest[:idx],
				Function: rest[idx+1:],
			}
		}
		return &Import{Module: path}
	}

	// User modules: use "file.sx/func"
	if idx := strings.Index(path, "/"); idx != -1 {
		return &Import{
			Module:   path[:idx],
			Function: path[idx+1:],
		}
	}
	return &Import{Module: path}
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
