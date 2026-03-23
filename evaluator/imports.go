package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
	"github.com/erickweyunga/sintax/stdlib"
)

// importedFuncs holds stdlib functions registered via use.
var importedFuncs = map[string]stdlib.StdFn{}

// importedUserEnvs holds environments from imported .sx files.
var importedUserEnvs = map[string]*Environment{}

// RegisterImports processes use directives and loads stdlib/user modules.
func RegisterImports(imports []preprocessor.Import) error {
	importedFuncs = map[string]stdlib.StdFn{}
	importedUserEnvs = map[string]*Environment{}

	for _, imp := range imports {
		if strings.HasSuffix(imp.Module, ".sx") {
			if err := loadUserModule(imp); err != nil {
				return err
			}
			continue
		}

		mod, ok := stdlib.Registry[imp.Module]
		if !ok {
			return fmt.Errorf("Error: unknown module '%s'", imp.Module)
		}

		switch imp.Function {
		case "":
			for name, fn := range mod.Funcs {
				importedFuncs[imp.Module+"__"+name] = fn
			}
		case "*":
			for name, fn := range mod.Funcs {
				importedFuncs[name] = fn
			}
		default:
			fn, ok := mod.Funcs[imp.Function]
			if !ok {
				return fmt.Errorf("Error: '%s' not found in module '%s'", imp.Function, imp.Module)
			}
			importedFuncs[imp.Function] = fn
		}
	}
	return nil
}

func loadUserModule(imp preprocessor.Import) error {
	filename := imp.Module

	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Error: cannot read module '%s'", filename)
	}

	result := preprocessor.Process(string(source))

	p := parser.NewParser()
	program, err := p.ParseString(filename, result.Source)
	if err != nil {
		return fmt.Errorf("Error: syntax error in module '%s': %v", filename, err)
	}

	modEnv := NewEnvironment()
	evalStatements(program.Statements, modEnv)

	modName := strings.TrimSuffix(filepath.Base(filename), ".sx")

	switch imp.Function {
	case "":
		importedUserEnvs[modName] = modEnv
	case "*":
		importedUserEnvs["__wildcard__"+modName] = modEnv
	default:
		importedUserEnvs["__specific__"+imp.Function+"__"+modName] = modEnv
	}

	return nil
}

// callImportedStdlib looks up and calls an imported stdlib function.
func callImportedStdlib(name string, args []*parser.Expr, env *Environment) (object.Object, bool) {
	fn, ok := importedFuncs[name]
	if !ok {
		return nil, false
	}
	evaledArgs := make([]object.Object, len(args))
	for i, arg := range args {
		evaledArgs[i] = evalExpr(arg, env)
	}
	result, err := fn(evaledArgs)
	if err != nil {
		runtimeError("%s", err.Error())
	}
	return result, true
}

// callUserModule looks up and calls a namespaced user module function (module__func).
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

// callWildcardModule looks up a function in wildcard/specific user imports.
func callWildcardModule(name string, args []*parser.Expr, env *Environment) (object.Object, bool) {
	for key, modEnv := range importedUserEnvs {
		if strings.HasPrefix(key, "__wildcard__") || strings.HasPrefix(key, "__specific__"+name+"__") {
			if result, ok := callFromEnv(modEnv, name, "", args, env); ok {
				return result, true
			}
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
