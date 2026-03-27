package evaluator

import "github.com/erickweyunga/sintax/object"

type Environment struct {
	store  map[string]object.Object
	types  map[string]string
	consts map[string]bool
	outer  *Environment
}

func NewEnvironment() *Environment {
	return &Environment{
		store:  make(map[string]object.Object),
		types:  make(map[string]string),
		consts: make(map[string]bool),
	}
}

func NewEnclosed(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (object.Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) GetType(name string) (string, bool) {
	t, ok := e.types[name]
	if !ok && e.outer != nil {
		return e.outer.GetType(name)
	}
	return t, ok
}

func (e *Environment) IsConst(name string) bool {
	if e.consts[name] {
		return true
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
	}
	return false
}

// Set updates a variable in the scope where it was defined,
// enabling mutable closures. Creates a new binding if not found.
func (e *Environment) Set(name string, val object.Object) {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return
	}
	if e.outer != nil {
		if _, ok := e.outer.resolve(name); ok {
			e.outer.setInPlace(name, val)
			return
		}
	}
	e.store[name] = val
}

func (e *Environment) resolve(name string) (*Environment, bool) {
	if _, ok := e.store[name]; ok {
		return e, true
	}
	if e.outer != nil {
		return e.outer.resolve(name)
	}
	return nil, false
}

func (e *Environment) setInPlace(name string, val object.Object) {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return
	}
	if e.outer != nil {
		e.outer.setInPlace(name, val)
	}
}

func (e *Environment) SetTyped(name string, typ string, val object.Object) {
	e.store[name] = val
	e.types[name] = typ
}

func (e *Environment) SetConst(name string, typ string, val object.Object) {
	e.store[name] = val
	e.types[name] = typ
	e.consts[name] = true
}
