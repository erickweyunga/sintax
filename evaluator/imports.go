package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

var importedUserEnvs = map[string]*Environment{}

func RegisterImports(imports []preprocessor.Import) error {
	importedUserEnvs = map[string]*Environment{}

	for _, imp := range imports {
		var filePath string
		modName := imp.Module
		if strings.HasPrefix(modName, "std/") {
			stdName := strings.TrimPrefix(modName, "std/")
			filePath = findStdlib(stdName)
			if filePath == "" {
				return fmt.Errorf("Error: unknown stdlib module '%s'", modName)
			}
			imp.Module = stdName
		} else if strings.HasSuffix(modName, ".sx") {
			filePath = modName
		} else {
			return fmt.Errorf("Error: unknown module '%s' (use 'std/name' for stdlib or 'file.sx' for user modules)", modName)
		}

		if err := loadModule(imp, filePath); err != nil {
			return err
		}
	}
	return nil
}

func findStdlib(name string) string {
	home := os.Getenv("SINTAX_HOME")
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".sintax")
	}
	p := filepath.Join(home, "stdlib", name+".sx")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	for _, rel := range []string{
		filepath.Join("stdlib", name+".sx"),
		filepath.Join("..", "stdlib", name+".sx"),
	} {
		if _, err := os.Stat(rel); err == nil {
			abs, _ := filepath.Abs(rel)
			return abs
		}
	}

	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		dir := filepath.Dir(exe)
		for _, rel := range []string{"stdlib", "../stdlib"} {
			p := filepath.Join(dir, rel, name+".sx")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	return ""
}

func loadModule(imp preprocessor.Import, filePath string) error {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("Error: cannot read module '%s'", filePath)
	}

	result := preprocessor.Process(string(source))

	p := parser.NewParser()
	program, err := p.ParseString(filePath, result.Source)
	if err != nil {
		return fmt.Errorf("Error: syntax error in module '%s': %v", filePath, err)
	}

	modEnv := NewEnvironment()
	evalStatements(program.Statements, modEnv)

	modName := strings.TrimSuffix(filepath.Base(filePath), ".sx")

	switch imp.Function {
	case "":
		importedUserEnvs[modName] = modEnv
	default:
		importedUserEnvs["__specific__"+imp.Function+"__"+modName] = modEnv
	}

	return nil
}

func callUserModule(name string, args []*parser.Expr, env *Environment) (object.Object, bool) {
	if !strings.Contains(name, "__") {
		return nil, false
	}
	parts := strings.SplitN(name, "__", 2)
	modEnv, ok := importedUserEnvs[parts[0]]
	if !ok {
		return nil, false
	}
	return callFromEnv(modEnv, parts[1], parts[0], args, env)
}

func callSpecificImport(name string, args []*parser.Expr, env *Environment) (object.Object, bool) {
	for key, modEnv := range importedUserEnvs {
		if strings.HasPrefix(key, "__specific__"+name+"__") {
			return callFromEnv(modEnv, name, "", args, env)
		}
	}
	return nil, false
}

func callFromEnv(modEnv *Environment, funcName, modName string, args []*parser.Expr, callerEnv *Environment) (object.Object, bool) {
	obj, ok := modEnv.Get(funcName)
	if !ok {
		return nil, false
	}
	fn, ok := obj.(*object.FuncObj)
	if !ok {
		if modName != "" {
			runtimeError("'%s' is not a function in module '%s'", funcName, modName)
		}
		return nil, false
	}
	if !fn.Pub {
		if modName != "" {
			runtimeError("'%s' is private in module '%s' (add 'pub' to export it)", funcName, modName)
		}
		return nil, false
	}
	if len(fn.Params) != len(args) {
		runtimeError("'%s' expects %d args, got %d", funcName, len(fn.Params), len(args))
	}
	fnEnv := NewEnclosed(fn.Env.(*Environment))
	for i, param := range fn.Params {
		fnEnv.Set(param.Name, evalExpr(args[i], callerEnv))
	}
	result := evalStatements(fn.Body.Statements, fnEnv)
	if ret, ok := result.(*object.ReturnObj); ok {
		return ret.Value, true
	}
	return result, true
}
