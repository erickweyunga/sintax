package object

import (
	"fmt"
	"strings"

	"github.com/erickweyunga/sintax/parser"
)

// Object is the interface all runtime values implement.
type Object interface {
	Inspect() string
}

// Env is the interface for variable scope environments.
// This allows FuncObj to reference its closure without circular imports.
type Env interface {
	Get(name string) (Object, bool)
	Set(name string, val Object)
}

// NumberObj represents a numeric value.
type NumberObj struct{ Value float64 }

func (o *NumberObj) Inspect() string { return fmt.Sprintf("%g", o.Value) }

// StringObj represents a string value.
type StringObj struct{ Value string }

func (o *StringObj) Inspect() string { return o.Value }

// BoolObj represents a boolean value.
type BoolObj struct{ Value bool }

func (o *BoolObj) Inspect() string {
	if o.Value {
		return "kweli"
	}
	return "sikweli"
}

// NullObj represents a null/void value.
type NullObj struct{}

func (o *NullObj) Inspect() string { return "tupu" }

// ReturnObj wraps a return value to propagate it up the call stack.
type ReturnObj struct{ Value Object }

func (o *ReturnObj) Inspect() string { return o.Value.Inspect() }

// ListObj represents a list of values.
type ListObj struct{ Elements []Object }

func (o *ListObj) Inspect() string {
	items := make([]string, len(o.Elements))
	for i, el := range o.Elements {
		if s, ok := el.(*StringObj); ok {
			items[i] = fmt.Sprintf("\"%s\"", s.Value)
		} else {
			items[i] = el.Inspect()
		}
	}
	return "[" + strings.Join(items, ", ") + "]"
}

// DictObj represents a dictionary/map of key-value pairs.
type DictObj struct {
	Pairs map[string]Object
	Keys  []string // preserve insertion order
}

func (o *DictObj) Inspect() string {
	items := make([]string, len(o.Keys))
	for i, k := range o.Keys {
		val := o.Pairs[k]
		if s, ok := val.(*StringObj); ok {
			items[i] = fmt.Sprintf("\"%s\": \"%s\"", k, s.Value)
		} else {
			items[i] = fmt.Sprintf("\"%s\": %s", k, val.Inspect())
		}
	}
	return "{" + strings.Join(items, ", ") + "}"
}

// BreakObj signals a loop break (0).
type BreakObj struct{}

func (o *BreakObj) Inspect() string { return "0" }

// ContinueObj signals a loop continue (1).
type ContinueObj struct{}

func (o *ContinueObj) Inspect() string { return "1" }

// FuncParam holds a function parameter name and optional type.
type FuncParam struct {
	Name string
	Type string // empty = untyped
}

// FuncObj represents a user-defined function with its closure.
type FuncObj struct {
	Name       string
	Params     []FuncParam
	ReturnType string // empty = untyped
	Body       *parser.Block
	Env        Env
}

func (o *FuncObj) Inspect() string { return fmt.Sprintf("<unda %s>", o.Name) }

// CompiledFunction is a bytecode-compiled function.
type CompiledFunction struct {
	Instructions []byte
	NumLocals    int
	NumParams    int
	Name         string
}

func (o *CompiledFunction) Inspect() string {
	return fmt.Sprintf("<unda-compiled %s>", o.Name)
}

// Closure wraps a compiled function with its captured upvalues.
type Closure struct {
	Fn       *CompiledFunction
	Upvalues []*Upvalue
}

func (o *Closure) Inspect() string { return fmt.Sprintf("<closure %s>", o.Fn.Name) }

// Upvalue represents a captured variable from an enclosing scope.
type Upvalue struct {
	Value  Object
	Closed bool
}
