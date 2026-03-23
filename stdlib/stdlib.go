package stdlib

import (
	"fmt"
	"sort"

	sxmath "github.com/erickweyunga/sintax/stdlib/math"
	sxos "github.com/erickweyunga/sintax/stdlib/os"
	sxstring "github.com/erickweyunga/sintax/stdlib/string"
	"github.com/erickweyunga/sintax/stdlib/types"
)

// Re-export types for external use.
type StdFn = types.StdFn
type Module = types.Module
type FuncInfo = types.FuncInfo

// Registry holds all available stdlib modules.
var Registry = map[string]*Module{}

func init() {
	Register(sxmath.Load())
	Register(sxstring.Load())
	Register(sxos.Load())
}

// Register adds a module to the registry.
func Register(m *Module) {
	Registry[m.Name] = m
}

// ListModules returns all available module names sorted.
func ListModules() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Describe returns a human-readable description of a module.
func Describe(name string) (string, error) {
	mod, ok := Registry[name]
	if !ok {
		return "", fmt.Errorf("Error: unknown module '%s'", name)
	}

	result := fmt.Sprintf("%s - %s\n\n", mod.Name, mod.Desc)
	for _, info := range mod.Info {
		result += fmt.Sprintf("  %-25s %s\n", info.Name, info.Desc)
	}
	return result, nil
}
