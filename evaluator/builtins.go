package evaluator

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/parser"
)

// BuiltinFn is the signature for built-in functions.
type BuiltinFn func(args []*parser.Expr, env *Environment) object.Object

// builtins is initialized at runtime to avoid initialization cycles.
var builtins map[string]BuiltinFn

func init() {
	builtins = map[string]BuiltinFn{
		"andika": builtinAndika,
		"aina":   builtinAina,
		"urefu":  builtinUrefu,
		"ongeza": builtinOngeza,
		"ondoa":  builtinOndoa,
		"masafa": builtinMasafa,
		"soma":    builtinSoma,
		"funguo":  builtinFunguo,
		"thamani": builtinThamani,
		"ina":     builtinIna,
		"nambari": builtinNambari,
		"tungo":   builtinTungo,
		"buliani": builtinBuliani,
	}
}

// andika — print values
func builtinAndika(args []*parser.Expr, env *Environment) object.Object {
	vals := make([]string, len(args))
	for i, arg := range args {
		vals[i] = evalExpr(arg, env).Inspect()
	}
	fmt.Println(strings.Join(vals, " "))
	return &object.NullObj{}
}

// aina — return the type of a value as a string
func builtinAina(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("aina() inahitaji hoja 1")
	}
	val := evalExpr(args[0], env)
	switch val.(type) {
	case *object.NumberObj:
		return &object.StringObj{Value: "nambari"}
	case *object.StringObj:
		return &object.StringObj{Value: "tungo"}
	case *object.BoolObj:
		return &object.StringObj{Value: "buliani"}
	case *object.FuncObj:
		return &object.StringObj{Value: "unda"}
	case *object.ListObj:
		return &object.StringObj{Value: "safu"}
	case *object.DictObj:
		return &object.StringObj{Value: "kamusi"}
	default:
		return &object.StringObj{Value: "tupu"}
	}
}

// urefu — return the length of a list or string
func builtinUrefu(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("urefu() inahitaji hoja 1")
	}
	val := evalExpr(args[0], env)
	switch o := val.(type) {
	case *object.ListObj:
		return &object.NumberObj{Value: float64(len(o.Elements))}
	case *object.StringObj:
		return &object.NumberObj{Value: float64(len(o.Value))}
	case *object.DictObj:
		return &object.NumberObj{Value: float64(len(o.Pairs))}
	default:
		runtimeError("urefu() inahitaji safu, tungo, au kamusi")
	}
	return &object.NullObj{}
}

// ongeza — append an item to a list
func builtinOngeza(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("ongeza() inahitaji hoja 2 (safu, kipengele)")
	}
	list := evalExpr(args[0], env)
	item := evalExpr(args[1], env)

	l, ok := list.(*object.ListObj)
	if !ok {
		runtimeError("ongeza() hoja ya kwanza lazima iwe safu")
	}
	l.Elements = append(l.Elements, item)
	return l
}

// ondoa — remove an item from a list by index
func builtinOndoa(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("ondoa() inahitaji hoja 2 (safu, fahirisi)")
	}
	list := evalExpr(args[0], env)
	idx := evalExpr(args[1], env)

	l, ok := list.(*object.ListObj)
	if !ok {
		runtimeError("ondoa() hoja ya kwanza lazima iwe safu")
	}
	idxNum, ok := idx.(*object.NumberObj)
	if !ok {
		runtimeError("ondoa() hoja ya pili lazima iwe nambari")
	}
	i := int(idxNum.Value)
	if i < 0 || i >= len(l.Elements) {
		runtimeError("Fahirisi %d nje ya masafa (urefu %d)", i, len(l.Elements))
	}
	removed := l.Elements[i]
	l.Elements = append(l.Elements[:i], l.Elements[i+1:]...)
	return removed
}

// masafa — generate a range of numbers as a list
func builtinMasafa(args []*parser.Expr, env *Environment) object.Object {
	if len(args) < 1 || len(args) > 2 {
		runtimeError("masafa() inahitaji hoja 1 au 2")
	}

	var start, end float64
	if len(args) == 1 {
		start = 0
		n, ok := evalExpr(args[0], env).(*object.NumberObj)
		if !ok {
			runtimeError("masafa() inahitaji nambari")
		}
		end = n.Value
	} else {
		sn, ok1 := evalExpr(args[0], env).(*object.NumberObj)
		en, ok2 := evalExpr(args[1], env).(*object.NumberObj)
		if !ok1 || !ok2 {
			runtimeError("masafa() inahitaji nambari")
		}
		start = sn.Value
		end = en.Value
	}

	var elements []object.Object
	for i := start; i < end; i++ {
		elements = append(elements, &object.NumberObj{Value: i})
	}
	return &object.ListObj{Elements: elements}
}

// funguo — return keys of a dict as a list
func builtinFunguo(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("funguo() inahitaji hoja 1")
	}
	val := evalExpr(args[0], env)
	d, ok := val.(*object.DictObj)
	if !ok {
		runtimeError("funguo() inahitaji kamusi")
	}
	elements := make([]object.Object, len(d.Keys))
	for i, k := range d.Keys {
		elements[i] = &object.StringObj{Value: k}
	}
	return &object.ListObj{Elements: elements}
}

// thamani — return values of a dict as a list
func builtinThamani(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("thamani() inahitaji hoja 1")
	}
	val := evalExpr(args[0], env)
	d, ok := val.(*object.DictObj)
	if !ok {
		runtimeError("thamani() inahitaji kamusi")
	}
	elements := make([]object.Object, len(d.Keys))
	for i, k := range d.Keys {
		elements[i] = d.Pairs[k]
	}
	return &object.ListObj{Elements: elements}
}

// ina — check if a dict has a key
func builtinIna(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 2 {
		runtimeError("ina() inahitaji hoja 2 (kamusi, ufunguo)")
	}
	val := evalExpr(args[0], env)
	key := evalExpr(args[1], env)

	d, ok := val.(*object.DictObj)
	if !ok {
		runtimeError("ina() hoja ya kwanza lazima iwe kamusi")
	}
	keyStr, ok := key.(*object.StringObj)
	if !ok {
		runtimeError("ina() hoja ya pili lazima iwe neno")
	}
	_, exists := d.Pairs[keyStr.Value]
	return &object.BoolObj{Value: exists}
}

var stdinReader = bufio.NewReader(os.Stdin)

// soma — read user input, optionally with a prompt
func builtinSoma(args []*parser.Expr, env *Environment) object.Object {
	if len(args) > 1 {
		runtimeError("soma() inahitaji hoja 0 au 1")
	}

	// Print prompt if provided
	if len(args) == 1 {
		prompt := evalExpr(args[0], env)
		fmt.Print(prompt.Inspect())
	}

	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimRight(input, "\r\n")

	// Try to convert to number
	if num, err := strconv.ParseFloat(input, 64); err == nil {
		return &object.NumberObj{Value: num}
	}

	return &object.StringObj{Value: input}
}

// nambari — convert to number
func builtinNambari(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("nambari() inahitaji hoja 1")
	}
	val := evalExpr(args[0], env)
	switch v := val.(type) {
	case *object.NumberObj:
		return v
	case *object.StringObj:
		num, err := strconv.ParseFloat(v.Value, 64)
		if err != nil {
			runtimeError("Haiwezi kubadilisha '%s' kuwa nambari", v.Value)
		}
		return &object.NumberObj{Value: num}
	case *object.BoolObj:
		if v.Value {
			return &object.NumberObj{Value: 1}
		}
		return &object.NumberObj{Value: 0}
	default:
		runtimeError("Haiwezi kubadilisha %s kuwa nambari", ainaName(val))
	}
	return &object.NullObj{}
}

// tungo — convert to string
func builtinTungo(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("tungo() inahitaji hoja 1")
	}
	val := evalExpr(args[0], env)
	return &object.StringObj{Value: val.Inspect()}
}

// buliani — convert to boolean
func builtinBuliani(args []*parser.Expr, env *Environment) object.Object {
	if len(args) != 1 {
		runtimeError("buliani() inahitaji hoja 1")
	}
	val := evalExpr(args[0], env)
	return &object.BoolObj{Value: isTruthy(val)}
}

// helper to get type name in Swahili
func ainaName(obj object.Object) string {
	switch obj.(type) {
	case *object.NumberObj:
		return "nambari"
	case *object.StringObj:
		return "tungo"
	case *object.BoolObj:
		return "buliani"
	case *object.ListObj:
		return "safu"
	case *object.DictObj:
		return "kamusi"
	case *object.FuncObj:
		return "unda"
	default:
		return "tupu"
	}
}
