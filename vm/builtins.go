package vm

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/erickweyunga/sintax/object"
)

// VMBuiltinFn is the signature for VM-compatible builtins.
type VMBuiltinFn func(args []object.Object) (object.Object, error)

// vmBuiltins maps builtin index to implementation.
// Order must match compiler.BuiltinNames.
var vmBuiltins = []VMBuiltinFn{
	vmAndika,   // 0
	vmSoma,     // 1
	vmAina,     // 2
	vmUrefu,    // 3
	vmOngeza,   // 4
	vmOndoa,    // 5
	vmMasafa,   // 6
	vmFunguo,   // 7
	vmThamani,  // 8
	vmIna,      // 9
	vmNambari,  // 10
	vmTungo,    // 11
	vmBuliani,  // 12
}

var stdinReader = bufio.NewReader(os.Stdin)

func vmAndika(args []object.Object) (object.Object, error) {
	vals := make([]string, len(args))
	for i, a := range args {
		vals[i] = a.Inspect()
	}
	fmt.Println(strings.Join(vals, " "))
	return &object.NullObj{}, nil
}

func vmSoma(args []object.Object) (object.Object, error) {
	if len(args) > 0 {
		fmt.Print(args[0].Inspect())
	}
	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimRight(input, "\r\n")
	if num, err := strconv.ParseFloat(input, 64); err == nil {
		return &object.NumberObj{Value: num}, nil
	}
	return &object.StringObj{Value: input}, nil
}

func vmAina(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Kosa: aina() inahitaji hoja 1")
	}
	return &object.StringObj{Value: typeOfObj(args[0])}, nil
}

func vmUrefu(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Kosa: urefu() inahitaji hoja 1")
	}
	switch o := args[0].(type) {
	case *object.ListObj:
		return &object.NumberObj{Value: float64(len(o.Elements))}, nil
	case *object.StringObj:
		return &object.NumberObj{Value: float64(len(o.Value))}, nil
	case *object.DictObj:
		return &object.NumberObj{Value: float64(len(o.Pairs))}, nil
	}
	return nil, fmt.Errorf("Kosa: urefu() inahitaji safu, tungo, au kamusi")
}

func vmOngeza(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Kosa: ongeza() inahitaji hoja 2")
	}
	l, ok := args[0].(*object.ListObj)
	if !ok {
		return nil, fmt.Errorf("Kosa: ongeza() hoja ya kwanza lazima iwe safu")
	}
	l.Elements = append(l.Elements, args[1])
	return l, nil
}

func vmOndoa(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Kosa: ondoa() inahitaji hoja 2")
	}
	l, ok := args[0].(*object.ListObj)
	if !ok {
		return nil, fmt.Errorf("Kosa: ondoa() hoja ya kwanza lazima iwe safu")
	}
	idx, ok := args[1].(*object.NumberObj)
	if !ok {
		return nil, fmt.Errorf("Kosa: ondoa() hoja ya pili lazima iwe nambari")
	}
	i := int(idx.Value)
	if i < 0 || i >= len(l.Elements) {
		return nil, fmt.Errorf("Kosa: Fahirisi %d nje ya masafa", i)
	}
	removed := l.Elements[i]
	l.Elements = append(l.Elements[:i], l.Elements[i+1:]...)
	return removed, nil
}

func vmMasafa(args []object.Object) (object.Object, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("Kosa: masafa() inahitaji hoja 1 au 2")
	}
	var start, end float64
	if len(args) == 1 {
		n, ok := args[0].(*object.NumberObj)
		if !ok {
			return nil, fmt.Errorf("Kosa: masafa() inahitaji nambari")
		}
		end = n.Value
	} else {
		sn, ok1 := args[0].(*object.NumberObj)
		en, ok2 := args[1].(*object.NumberObj)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("Kosa: masafa() inahitaji nambari")
		}
		start = sn.Value
		end = en.Value
	}
	var elements []object.Object
	for i := start; i < end; i++ {
		elements = append(elements, &object.NumberObj{Value: i})
	}
	return &object.ListObj{Elements: elements}, nil
}

func vmFunguo(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Kosa: funguo() inahitaji hoja 1")
	}
	d, ok := args[0].(*object.DictObj)
	if !ok {
		return nil, fmt.Errorf("Kosa: funguo() inahitaji kamusi")
	}
	elements := make([]object.Object, len(d.Keys))
	for i, k := range d.Keys {
		elements[i] = &object.StringObj{Value: k}
	}
	return &object.ListObj{Elements: elements}, nil
}

func vmThamani(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Kosa: thamani() inahitaji hoja 1")
	}
	d, ok := args[0].(*object.DictObj)
	if !ok {
		return nil, fmt.Errorf("Kosa: thamani() inahitaji kamusi")
	}
	elements := make([]object.Object, len(d.Keys))
	for i, k := range d.Keys {
		elements[i] = d.Pairs[k]
	}
	return &object.ListObj{Elements: elements}, nil
}

func vmIna(args []object.Object) (object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Kosa: ina() inahitaji hoja 2")
	}
	d, ok := args[0].(*object.DictObj)
	if !ok {
		return nil, fmt.Errorf("Kosa: ina() hoja ya kwanza lazima iwe kamusi")
	}
	key, ok := args[1].(*object.StringObj)
	if !ok {
		return nil, fmt.Errorf("Kosa: ina() hoja ya pili lazima iwe tungo")
	}
	_, exists := d.Pairs[key.Value]
	return &object.BoolObj{Value: exists}, nil
}

func vmNambari(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Kosa: nambari() inahitaji hoja 1")
	}
	switch v := args[0].(type) {
	case *object.NumberObj:
		return v, nil
	case *object.StringObj:
		num, err := strconv.ParseFloat(v.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("Kosa: Haiwezi kubadilisha '%s' kuwa nambari", v.Value)
		}
		return &object.NumberObj{Value: num}, nil
	case *object.BoolObj:
		if v.Value {
			return &object.NumberObj{Value: 1}, nil
		}
		return &object.NumberObj{Value: 0}, nil
	}
	return nil, fmt.Errorf("Kosa: Haiwezi kubadilisha kuwa nambari")
}

func vmTungo(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Kosa: tungo() inahitaji hoja 1")
	}
	return &object.StringObj{Value: args[0].Inspect()}, nil
}

func vmBuliani(args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Kosa: buliani() inahitaji hoja 1")
	}
	return &object.BoolObj{Value: isTruthy(args[0])}, nil
}
