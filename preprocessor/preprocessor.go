package preprocessor

import (
	"path/filepath"
	"strings"
)

type Import struct {
	Module   string // e.g. "math"
	Function string // e.g. "sqrt" or ""
}

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

type Result struct {
	Source  string
	LineMap []int
	Imports []Import
}

// Process converts indentation-based Sintax source into brace-delimited form.
func Process(source string) Result {
	source = collapseBacktickStrings(source)
	lines := strings.Split(source, "\n")
	indentStack := []int{0}
	var resultLines []string
	var lineMap []int
	lastOrigLine := 0

	inBlockComment := false
	var imports []Import

	for origLine, line := range lines {
		origLineNum := origLine + 1
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

		trimmed = strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

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

		for len(indentStack) > 1 && indent < indentStack[len(indentStack)-1] {
			indentStack = indentStack[:len(indentStack)-1]
			resultLines = append(resultLines, "}")
			lineMap = append(lineMap, origLineNum)
		}

		if indent > indentStack[len(indentStack)-1] {
			indentStack = append(indentStack, indent)
		}

		if strings.HasSuffix(stripped, ":") {
			resultLines = append(resultLines, stripped[:len(stripped)-1]+" {")
			lineMap = append(lineMap, origLineNum)
		} else {
			resultLines = append(resultLines, stripped+" ;")
			lineMap = append(lineMap, origLineNum)
		}
	}

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

func rewriteNamespaceCalls(line, module string) string {
	prefix := module + "/"
	for {
		idx := strings.Index(line, prefix)
		if idx == -1 {
			break
		}
		line = line[:idx] + module + "__" + line[idx+len(prefix):]
	}
	return line
}

func parseUse(line string) *Import {
	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start == -1 || end <= start {
		return nil
	}
	path := line[start+1 : end]
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

	if idx := strings.Index(path, "/"); idx != -1 {
		return &Import{
			Module:   path[:idx],
			Function: path[idx+1:],
		}
	}
	return &Import{Module: path}
}

func collapseBacktickStrings(source string) string {
	var result strings.Builder
	i := 0
	for i < len(source) {
		if source[i] == '`' {
			result.WriteByte('`')
			i++
			for i < len(source) && source[i] != '`' {
				if source[i] == '\n' {
					result.WriteString(`\n`)
				} else {
					result.WriteByte(source[i])
				}
				i++
			}
			if i < len(source) {
				result.WriteByte('`')
				i++
			}
		} else {
			result.WriteByte(source[i])
			i++
		}
	}
	return result.String()
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
