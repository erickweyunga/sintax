package evaluator

import "github.com/erickweyunga/sintax/object"

// Environment holds variable bindings with lexical scoping.
type Environment struct {
	store  map[string]object.Object
	types  map[string]string // variable name → type constraint (empty = dynamic)
	consts map[string]bool   // variable name → true if const
	outer  *Environment
}

// NewEnvironment creates a new top-level environment.
func NewEnvironment() *Environment {
	return &Environment{
		store:  make(map[string]object.Object),
		types:  make(map[string]string),
		consts: make(map[string]bool),
	}
}

// NewEnclosed creates a child environment for lexical scoping.
func NewEnclosed(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get retrieves a variable, walking up the scope chain.
func (e *Environment) Get(name string) (object.Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
}

// GetType retrieves a variable's type constraint, walking up the scope chain.
func (e *Environment) GetType(name string) (string, bool) {
	t, ok := e.types[name]
	if !ok && e.outer != nil {
		return e.outer.GetType(name)
	}
	return t, ok
}

// IsConst checks if a variable is const, walking up the scope chain.
func (e *Environment) IsConst(name string) bool {
	if e.consts[name] {
		return true
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
	}
	return false
}

// Set updates a variable. If it exists in an outer scope, updates it there
// (enabling mutable closures). Otherwise creates a new binding in current scope.
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

// resolve checks if a variable exists anywhere in the scope chain.
func (e *Environment) resolve(name string) (*Environment, bool) {
	if _, ok := e.store[name]; ok {
		return e, true
	}
	if e.outer != nil {
		return e.outer.resolve(name)
	}
	return nil, false
}

// setInPlace updates a variable in the scope where it was defined.
func (e *Environment) setInPlace(name string, val object.Object) {
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return
	}
	if e.outer != nil {
		e.outer.setInPlace(name, val)
	}
}

// SetTyped binds a variable with a type constraint.
func (e *Environment) SetTyped(name string, typ string, val object.Object) {
	e.store[name] = val
	e.types[name] = typ
}

// SetConst binds a constant variable with a type constraint.
func (e *Environment) SetConst(name string, typ string, val object.Object) {
	e.store[name] = val
	e.types[name] = typ
	e.consts[name] = true
}
