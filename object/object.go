package object

import (
	"fmt"
	"strings"

	"github.com/erickweyunga/sintax/parser"
)

type Object interface {
	Inspect() string
}

// Env allows FuncObj to reference its closure without circular imports.
type Env interface {
	Get(name string) (Object, bool)
	Set(name string, val Object)
}

type NumberObj struct{ Value float64 }

func (o *NumberObj) Inspect() string { return fmt.Sprintf("%g", o.Value) }

type StringObj struct{ Value string }

func (o *StringObj) Inspect() string { return o.Value }

type BoolObj struct{ Value bool }

func (o *BoolObj) Inspect() string {
	if o.Value {
		return "true"
	}
	return "false"
}

type NullObj struct{}

func (o *NullObj) Inspect() string { return "null" }

type ReturnObj struct{ Value Object }

func (o *ReturnObj) Inspect() string { return o.Value.Inspect() }

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

type DictObj struct {
	Pairs map[string]Object
	Keys  []string
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

type BreakObj struct{}

func (o *BreakObj) Inspect() string { return "0" }

type ContinueObj struct{}

func (o *ContinueObj) Inspect() string { return "1" }

type ErrorObj struct{ Message string }

func (o *ErrorObj) Inspect() string { return "error: " + o.Message }

type FuncParam struct {
	Name       string
	Type       string
	HasDefault bool
	Default    Object
}

type FuncObj struct {
	Name        string
	Params      []FuncParam
	ReturnTypes []string
	Body        *parser.Block
	Env         Env
	Pub         bool
}

func (o *FuncObj) Inspect() string { return fmt.Sprintf("<fn %s>", o.Name) }

func TypeName(obj Object) string {
	switch obj.(type) {
	case *NumberObj:
		return "num"
	case *StringObj:
		return "str"
	case *BoolObj:
		return "bool"
	case *ListObj:
		return "list"
	case *DictObj:
		return "dict"
	case *FuncObj:
		return "fn"
	case *ErrorObj:
		return "error"
	default:
		return "null"
	}
}

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

func NormalizeType(t string) string {
	return t
}

var (
	Null  = &NullObj{}
	True  = &BoolObj{Value: true}
	False = &BoolObj{Value: false}
)
