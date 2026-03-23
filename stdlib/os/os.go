package os

import (
	"fmt"
	goos "os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/stdlib/types"
)

// Load returns the os module.
func Load() *types.Module {
	return &types.Module{
		Name: "os",
		Desc: "OS, file, and process functions",
		Funcs: map[string]types.StdFn{
			// File operations
			"read":    Read,
			"write":   Write,
			"exists":  Exists,
			"append":  Append,
			"remove":  Remove,
			"listdir": Listdir,
			"mkdir":   Mkdir,
			// Environment
			"getenv": Getenv,
			"setenv": Setenv,
			// Process
			"exec": Exec,
			"exit": Exit,
			// System info
			"platform": Platform,
			"pid":      Pid,
			"cwd":      Cwd,
		},
		Info: []types.FuncInfo{
			{Name: "read(path)", Desc: "Read file"},
			{Name: "write(path, data)", Desc: "Write file"},
			{Name: "exists(path)", Desc: "Check if file/dir exists"},
			{Name: "append(path, data)", Desc: "Append to file"},
			{Name: "remove(path)", Desc: "Remove file"},
			{Name: "listdir(path)", Desc: "List directory contents"},
			{Name: "mkdir(path)", Desc: "Create directory"},
			{Name: "getenv(name)", Desc: "Get environment variable"},
			{Name: "setenv(name, value)", Desc: "Set environment variable"},
			{Name: "exec(cmd)", Desc: "Execute shell command"},
			{Name: "exit(code)", Desc: "Exit with code"},
			{Name: "platform()", Desc: "OS name (linux/darwin/windows)"},
			{Name: "pid()", Desc: "Process ID"},
			{Name: "cwd()", Desc: "Current working directory"},
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

// --- File operations ---

func Read(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: read() requires 1 argument")
	}
	path, err := requireStr(args, "read", 0)
	if err != nil {
		return nil, err
	}
	data, err := goos.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error: Cannot read '%s'", path)
	}
	return &object.StringObj{Value: string(data)}, nil
}

func Write(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: write() requires 2 arguments")
	}
	path, err := requireStr(args, "write", 0)
	if err != nil {
		return nil, err
	}
	if err := goos.WriteFile(path, []byte(args[1].Inspect()), 0644); err != nil {
		return nil, fmt.Errorf("Error: Cannot write '%s'", path)
	}
	return object.Null, nil
}

func Exists(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: exists() requires 1 argument")
	}
	path, err := requireStr(args, "exists", 0)
	if err != nil {
		return nil, err
	}
	_, err = goos.Stat(path)
	return &object.BoolObj{Value: err == nil}, nil
}

func Append(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: append() requires 2 arguments")
	}
	path, err := requireStr(args, "append", 0)
	if err != nil {
		return nil, err
	}
	f, err := goos.OpenFile(path, goos.O_APPEND|goos.O_CREATE|goos.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("Error: Cannot open '%s'", path)
	}
	defer f.Close()
	f.WriteString(args[1].Inspect())
	return object.Null, nil
}

func Remove(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: remove() requires 1 argument")
	}
	path, err := requireStr(args, "remove", 0)
	if err != nil {
		return nil, err
	}
	if err := goos.Remove(path); err != nil {
		return nil, fmt.Errorf("Error: Cannot remove '%s'", path)
	}
	return object.Null, nil
}

func Listdir(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: listdir() requires 1 argument")
	}
	path, err := requireStr(args, "listdir", 0)
	if err != nil {
		return nil, err
	}
	entries, err := goos.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("Error: Cannot read directory '%s'", path)
	}
	elements := make([]object.Object, len(entries))
	for i, e := range entries {
		elements[i] = &object.StringObj{Value: filepath.Join(path, e.Name())}
	}
	return &object.ListObj{Elements: elements}, nil
}

func Mkdir(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: mkdir() requires 1 argument")
	}
	path, err := requireStr(args, "mkdir", 0)
	if err != nil {
		return nil, err
	}
	if err := goos.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("Error: Cannot create directory '%s'", path)
	}
	return object.Null, nil
}

// --- Environment ---

func Getenv(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: getenv() requires 1 argument")
	}
	name, err := requireStr(args, "getenv", 0)
	if err != nil {
		return nil, err
	}
	val := goos.Getenv(name)
	if val == "" {
		return object.Null, nil
	}
	return &object.StringObj{Value: val}, nil
}

func Setenv(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Error: setenv() requires 2 arguments")
	}
	name, err := requireStr(args, "setenv", 0)
	if err != nil {
		return nil, err
	}
	val, err := requireStr(args, "setenv", 1)
	if err != nil {
		return nil, err
	}
	goos.Setenv(name, val)
	return object.Null, nil
}

// --- Process ---

func Exec(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Error: exec() requires 1 argument")
	}
	cmdStr, err := requireStr(args, "exec", 0)
	if err != nil {
		return nil, err
	}
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return nil, fmt.Errorf("Error: empty command")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &object.StringObj{Value: string(output)}, nil
	}
	return &object.StringObj{Value: strings.TrimRight(string(output), "\n")}, nil
}

func Exit(args []object.Object) (object.Object, error) {
	code := 0
	if len(args) == 1 {
		if n, ok := args[0].(*object.NumberObj); ok {
			code = int(n.Value)
		}
	}
	goos.Exit(code)
	return object.Null, nil
}

// --- System info ---

func Platform(args []object.Object) (object.Object, error) {
	return &object.StringObj{Value: runtime.GOOS}, nil
}

func Pid(args []object.Object) (object.Object, error) {
	return &object.NumberObj{Value: float64(goos.Getpid())}, nil
}

func Cwd(args []object.Object) (object.Object, error) {
	dir, err := goos.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Error: Cannot get current directory")
	}
	return &object.StringObj{Value: dir}, nil
}
