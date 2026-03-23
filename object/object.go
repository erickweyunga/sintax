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

// TypeName returns the Swahili type name for an object.
func TypeName(obj Object) string {
	switch obj.(type) {
	case *NumberObj:
		return "nambari"
	case *StringObj:
		return "tungo"
	case *BoolObj:
		return "buliani"
	case *ListObj:
		return "safu"
	case *DictObj:
		return "kamusi"
	case *FuncObj:
		return "unda"
	default:
		return "tupu"
	}
}

// IsTruthy returns whether a value is truthy.
func IsTruthy(obj Object) bool {
	switch o := obj.(type) {
	case *BoolObj:
		return o.Value
	case *NullObj:
		return false
	case *NumberObj:
		return o.Value != 0
	case *StringObj:
		return o.Value != ""
	case *ListObj:
		return len(o.Elements) > 0
	case *DictObj:
		return len(o.Pairs) > 0
	default:
		return true
	}
}

// ObjectsEqual checks structural equality of two objects.
func ObjectsEqual(a, b Object) bool {
	switch av := a.(type) {
	case *NumberObj:
		if bv, ok := b.(*NumberObj); ok {
			return av.Value == bv.Value
		}
	case *StringObj:
		if bv, ok := b.(*StringObj); ok {
			return av.Value == bv.Value
		}
	case *BoolObj:
		if bv, ok := b.(*BoolObj); ok {
			return av.Value == bv.Value
		}
	case *NullObj:
		_, ok := b.(*NullObj)
		return ok
	}
	return false
}

// NormalizeType maps short aliases to canonical type names.
func NormalizeType(t string) string {
	switch t {
	case "namba":
		return "nambari"
	case "bul":
		return "buliani"
	default:
		return t
	}
}

// Null is a singleton null value to avoid repeated allocations.
var Null = &NullObj{}

// True and False are singleton boolean values.
var True = &BoolObj{Value: true}
var False = &BoolObj{Value: false}
