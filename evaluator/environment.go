package evaluator

import "github.com/erickweyunga/sintax/object"

// Environment holds variable bindings with lexical scoping.
type Environment struct {
	store map[string]object.Object
	types map[string]string // variable name → type constraint (empty = dynamic)
	outer *Environment
}

// NewEnvironment creates a new top-level environment.
func NewEnvironment() *Environment {
	return &Environment{
		store: make(map[string]object.Object),
		types: make(map[string]string),
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

// Set binds a variable in the current scope.
// If the variable has a type constraint, it enforces it.
func (e *Environment) Set(name string, val object.Object) {
	e.store[name] = val
}

// SetTyped binds a variable with a type constraint.
func (e *Environment) SetTyped(name string, typ string, val object.Object) {
	e.store[name] = val
	e.types[name] = typ
}
