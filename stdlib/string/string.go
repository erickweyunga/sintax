package string

import (
	"fmt"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/stdlib/types"
)

// Load returns the string module.
func Load() *types.Module {
	return &types.Module{
		Name: "string",
		Desc: "String functions",
		Funcs: map[string]types.StdFn{
			"split":       Split,
			"join":        Join,
			"replace":     Replace,
			"trim":        Trim,
			"upper":       Upper,
			"lower":       Lower,
			"contains":    Contains,
			"starts_with": StartsWith,
			"ends_with":   EndsWith,
		},
		Info: []types.FuncInfo{
			{Name: "split(s, sep)", Desc: "Split string"},
			{Name: "join(list, sep)", Desc: "Join list into string"},
			{Name: "replace(s, old, new)", Desc: "Replace substring"},
			{Name: "trim(s)", Desc: "Trim whitespace"},
			{Name: "upper(s)", Desc: "Uppercase"},
			{Name: "lower(s)", Desc: "Lowercase"},
			{Name: "contains(s, sub)", Desc: "Check if contains substring"},
			{Name: "starts_with(s, prefix)", Desc: "Check prefix"},
			{Name: "ends_with(s, suffix)", Desc: "Check suffix"},
		},
	}
}

func requireStr(args []object.Object, name string, idx int) (string, error) {
	if idx >= len(args) {
		return "", fmt.Errorf("Error: %s() not enough arguments", name)
	}
	s, ok := args[idx].(*object.StringObj)
	if !ok {
		return "", fmt.Errorf("Error: %s() requires a str", name)
	}
	return s.Value, nil
}

func Split(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: split() requires 2 arguments (str, separator)")
	}
	s, err := requireStr(args, "split", 0)
	if err != nil {
		return nil, err
	}
	sep, err := requireStr(args, "split", 1)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(s, sep)
	elements := make([]object.Object, len(parts))
	for i, p := range parts {
		elements[i] = &object.StringObj{Value: p}
	}
	return &object.ListObj{Elements: elements}, nil
}

func Join(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: join() requires 2 arguments (list, separator)")
	}
	list, ok := args[0].(*object.ListObj)
	if !ok {
		return nil, fmt.Errorf("Error: join() first argument must be a list")
	}
	sep, err := requireStr(args, "join", 1)
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(list.Elements))
	for i, el := range list.Elements {
		parts[i] = el.Inspect()
	}
	return &object.StringObj{Value: strings.Join(parts, sep)}, nil
}

func Replace(args []object.Object) (object.Object, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("Error: replace() requires 3 arguments (str, old, new)")
	}
	s, err := requireStr(args, "replace", 0)
	if err != nil {
		return nil, err
	}
	old, err := requireStr(args, "replace", 1)
	if err != nil {
		return nil, err
	}
	new_, err := requireStr(args, "replace", 2)
	if err != nil {
		return nil, err
	}
	return &object.StringObj{Value: strings.ReplaceAll(s, old, new_)}, nil
}

func Trim(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: trim() requires 1 argument")
	}
	s, err := requireStr(args, "trim", 0)
	if err != nil {
		return nil, err
	}
	return &object.StringObj{Value: strings.TrimSpace(s)}, nil
}

func Upper(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: upper() requires 1 argument")
	}
	s, err := requireStr(args, "upper", 0)
	if err != nil {
		return nil, err
	}
	return &object.StringObj{Value: strings.ToUpper(s)}, nil
}

func Lower(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: lower() requires 1 argument")
	}
	s, err := requireStr(args, "lower", 0)
	if err != nil {
		return nil, err
	}
	return &object.StringObj{Value: strings.ToLower(s)}, nil
}

func Contains(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: contains() requires 2 arguments")
	}
	s, err := requireStr(args, "contains", 0)
	if err != nil {
		return nil, err
	}
	sub, err := requireStr(args, "contains", 1)
	if err != nil {
		return nil, err
	}
	return &object.BoolObj{Value: strings.Contains(s, sub)}, nil
}

func StartsWith(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: starts_with() requires 2 arguments")
	}
	s, err := requireStr(args, "starts_with", 0)
	if err != nil {
		return nil, err
	}
	prefix, err := requireStr(args, "starts_with", 1)
	if err != nil {
		return nil, err
	}
	return &object.BoolObj{Value: strings.HasPrefix(s, prefix)}, nil
}

func EndsWith(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: ends_with() requires 2 arguments")
	}
	s, err := requireStr(args, "ends_with", 0)
	if err != nil {
		return nil, err
	}
	suffix, err := requireStr(args, "ends_with", 1)
	if err != nil {
		return nil, err
	}
	return &object.BoolObj{Value: strings.HasSuffix(s, suffix)}, nil
}
