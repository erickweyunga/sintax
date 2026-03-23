package types

import "github.com/erickweyunga/sintax/object"

// StdFn is a stdlib function signature.
type StdFn func(args []object.Object) (object.Object, error)

// FuncInfo describes a stdlib function for discoverability.
type FuncInfo struct {
	Name string
	Desc string
}

// Module is a collection of named functions.
type Module struct {
	Name  string
	Desc  string
	Funcs map[string]StdFn
	Info  []FuncInfo
}
