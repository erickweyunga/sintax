package vm

import (
	"fmt"
	"math"
	"strings"

	"github.com/erickweyunga/sintax/compiler"
	"github.com/erickweyunga/sintax/object"
	"github.com/erickweyunga/sintax/opcode"
)

const (
	StackSize = 2048
	MaxFrames = 256
	MaxGlobals = 65536
)

// Frame represents a call frame on the VM's frame stack.
type Frame struct {
	cl      *object.Closure
	ip      int
	basePtr int
}

func (f *Frame) Instructions() []byte { return f.cl.Fn.Instructions }

// VM is the Sintax virtual machine.
type VM struct {
	constants   []object.Object
	globals     [MaxGlobals]object.Object
	globalTypes [MaxGlobals]uint8

	stack   [StackSize]object.Object
	sp      int

	frames  [MaxFrames]Frame
	fp      int
}

// New creates a VM from compiled bytecode.
func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{
		Instructions: bytecode.Instructions,
	}
	mainClosure := &object.Closure{Fn: mainFn}

	vm := &VM{
		constants: bytecode.Constants,
		sp:        0,
		fp:        1,
	}
	vm.frames[0] = Frame{cl: mainClosure, ip: 0, basePtr: 0}

	return vm
}

// Run executes the bytecode.
func (vm *VM) Run() error {
	for vm.currentFrame().ip < len(vm.currentFrame().Instructions()) {
		frame := vm.currentFrame()
		ins := frame.Instructions()
		ip := frame.ip
		op := opcode.Opcode(ins[ip])
		frame.ip++

		switch op {
		case opcode.OpConstant:
			idx := int(opcode.ReadUint16(ins, ip+1))
			frame.ip += 2
			vm.push(vm.constants[idx])

		case opcode.OpNull:
			vm.push(&object.NullObj{})
		case opcode.OpTrue:
			vm.push(&object.BoolObj{Value: true})
		case opcode.OpFalse:
			vm.push(&object.BoolObj{Value: false})

		// Arithmetic
		case opcode.OpAdd:
			right := vm.pop()
			left := vm.pop()
			result, err := execAdd(left, right)
			if err != nil {
				return err
			}
			vm.push(result)

		case opcode.OpSub, opcode.OpMul, opcode.OpDiv, opcode.OpMod, opcode.OpPow:
			right := vm.pop()
			left := vm.pop()
			result, err := execArith(op, left, right)
			if err != nil {
				return err
			}
			vm.push(result)

		// Comparison
		case opcode.OpEqual:
			right := vm.pop()
			left := vm.pop()
			vm.push(&object.BoolObj{Value: objectsEqual(left, right)})

		case opcode.OpNotEqual:
			right := vm.pop()
			left := vm.pop()
			vm.push(&object.BoolObj{Value: !objectsEqual(left, right)})

		case opcode.OpGreaterThan, opcode.OpLessThan, opcode.OpGreaterEq, opcode.OpLessEq:
			right := vm.pop()
			left := vm.pop()
			result, err := execComparison(op, left, right)
			if err != nil {
				return err
			}
			vm.push(result)

		// Membership
		case opcode.OpIn:
			right := vm.pop() // haystack
			left := vm.pop()  // needle
			result, err := execMembership(left, right)
			if err != nil {
				return err
			}
			vm.push(result)

		// Logical
		case opcode.OpNot:
			val := vm.pop()
			vm.push(&object.BoolObj{Value: !isTruthy(val)})

		// Variables
		case opcode.OpSetGlobal:
			idx := int(opcode.ReadUint16(ins, ip+1))
			frame.ip += 2
			vm.globals[idx] = vm.pop()

		case opcode.OpGetGlobal:
			idx := int(opcode.ReadUint16(ins, ip+1))
			frame.ip += 2
			val := vm.globals[idx]
			if val == nil {
				val = &object.NullObj{}
			}
			vm.push(val)

		case opcode.OpSetLocal:
			idx := int(opcode.ReadUint8(ins, ip+1))
			frame.ip++
			vm.stack[frame.basePtr+idx] = vm.pop()

		case opcode.OpGetLocal:
			idx := int(opcode.ReadUint8(ins, ip+1))
			frame.ip++
			vm.push(vm.stack[frame.basePtr+idx])

		// Upvalues
		case opcode.OpGetUpvalue:
			idx := int(opcode.ReadUint8(ins, ip+1))
			frame.ip++
			vm.push(frame.cl.Upvalues[idx].Value)

		case opcode.OpSetUpvalue:
			idx := int(opcode.ReadUint8(ins, ip+1))
			frame.ip++
			frame.cl.Upvalues[idx].Value = vm.pop()

		// Control flow
		case opcode.OpJump:
			target := int(opcode.ReadUint16(ins, ip+1))
			frame.ip = target

		case opcode.OpJumpIfFalse:
			target := int(opcode.ReadUint16(ins, ip+1))
			frame.ip += 2
			val := vm.pop()
			if !isTruthy(val) {
				frame.ip = target
			}

		case opcode.OpLoop:
			offset := int(opcode.ReadUint16(ins, ip+1))
			frame.ip = frame.ip + 2 - offset

		// Functions
		case opcode.OpClosure:
			fnIdx := int(opcode.ReadUint16(ins, ip+1))
			numUpvalues := int(opcode.ReadUint8(ins, ip+3))
			frame.ip += 3

			fn := vm.constants[fnIdx].(*object.CompiledFunction)
			upvalues := make([]*object.Upvalue, numUpvalues)

			for i := 0; i < numUpvalues; i++ {
				isLocal := ins[frame.ip] == 1
				idx := int(ins[frame.ip+1])
				frame.ip += 2

				if isLocal {
					upvalues[i] = &object.Upvalue{Value: vm.stack[frame.basePtr+idx]}
				} else {
					upvalues[i] = frame.cl.Upvalues[idx]
				}
			}

			closure := &object.Closure{Fn: fn, Upvalues: upvalues}
			vm.push(closure)

		case opcode.OpCall:
			numArgs := int(opcode.ReadUint8(ins, ip+1))
			frame.ip++

			callee := vm.stack[vm.sp-numArgs-1]
			cl, ok := callee.(*object.Closure)
			if !ok {
				return fmt.Errorf("Kosa: si unda")
			}

			if numArgs != cl.Fn.NumParams {
				return fmt.Errorf("Kosa: Unda '%s' inahitaji hoja %d, imepata %d", cl.Fn.Name, cl.Fn.NumParams, numArgs)
			}

			newFrame := Frame{
				cl:      cl,
				ip:      0,
				basePtr: vm.sp - numArgs,
			}
			vm.frames[vm.fp] = newFrame
			vm.fp++
			// Allocate space for locals
			vm.sp = newFrame.basePtr + cl.Fn.NumLocals

		case opcode.OpReturn:
			var returnVal object.Object
			if vm.sp > vm.currentFrame().basePtr {
				returnVal = vm.pop()
			} else {
				returnVal = &object.NullObj{}
			}

			basePtr := vm.currentFrame().basePtr
			vm.fp--
			vm.sp = basePtr - 1 // remove function from stack

			vm.push(returnVal)

		// Print
		case opcode.OpPrint:
			val := vm.pop()
			fmt.Println(val.Inspect())

		// Stack
		case opcode.OpPop:
			vm.pop()

		// Collections
		case opcode.OpList:
			count := int(opcode.ReadUint16(ins, ip+1))
			frame.ip += 2
			elements := make([]object.Object, count)
			for i := count - 1; i >= 0; i-- {
				elements[i] = vm.pop()
			}
			vm.push(&object.ListObj{Elements: elements})

		case opcode.OpDict:
			count := int(opcode.ReadUint16(ins, ip+1))
			frame.ip += 2
			pairs := make(map[string]object.Object)
			keys := make([]string, count)
			for i := count - 1; i >= 0; i-- {
				val := vm.pop()
				key := vm.pop()
				keyStr := key.(*object.StringObj).Value
				pairs[keyStr] = val
				keys[i] = keyStr
			}
			vm.push(&object.DictObj{Pairs: pairs, Keys: keys})

		case opcode.OpIndex:
			index := vm.pop()
			collection := vm.pop()
			result, err := execIndex(collection, index)
			if err != nil {
				return err
			}
			vm.push(result)

		case opcode.OpIndexAssign:
			val := vm.pop()
			index := vm.pop()
			collection := vm.pop()
			if err := execIndexAssign(collection, index, val); err != nil {
				return err
			}

		// Builtins
		case opcode.OpCallBuiltin:
			builtinIdx := int(opcode.ReadUint8(ins, ip+1))
			numArgs := int(opcode.ReadUint8(ins, ip+2))
			frame.ip += 2

			args := make([]object.Object, numArgs)
			for i := numArgs - 1; i >= 0; i-- {
				args[i] = vm.pop()
			}

			fn := vmBuiltins[builtinIdx]
			result, err := fn(args)
			if err != nil {
				return err
			}
			vm.push(result)

		// Type checking
		case opcode.OpCheckType:
			typeTag := opcode.ReadUint8(ins, ip+1)
			frame.ip++
			val := vm.peek()
			if !checkType(val, typeTag) {
				return fmt.Errorf("Kosa: Aina si sahihi: ni %s, inahitaji %s", typeOfObj(val), typeTagName(typeTag))
			}

		// String interpolation
		case opcode.OpInterpolate:
			count := int(opcode.ReadUint8(ins, ip+1))
			frame.ip++
			parts := make([]string, count)
			for i := count - 1; i >= 0; i-- {
				parts[i] = vm.pop().Inspect()
			}
			vm.push(&object.StringObj{Value: strings.Join(parts, "")})

		default:
			return fmt.Errorf("Kosa: opcode isiyojulikana: %d", op)
		}
	}

	return nil
}

// Stack operations

func (vm *VM) push(obj object.Object) {
	vm.stack[vm.sp] = obj
	vm.sp++
}

func (vm *VM) pop() object.Object {
	vm.sp--
	return vm.stack[vm.sp]
}

func (vm *VM) peek() object.Object {
	return vm.stack[vm.sp-1]
}

func (vm *VM) currentFrame() *Frame {
	return &vm.frames[vm.fp-1]
}

// Helper functions

func isTruthy(obj object.Object) bool {
	switch o := obj.(type) {
	case *object.BoolObj:
		return o.Value
	case *object.NullObj:
		return false
	case *object.NumberObj:
		return o.Value != 0
	case *object.StringObj:
		return o.Value != ""
	case *object.ListObj:
		return len(o.Elements) > 0
	case *object.DictObj:
		return len(o.Pairs) > 0
	default:
		return true
	}
}

func objectsEqual(a, b object.Object) bool {
	switch av := a.(type) {
	case *object.NumberObj:
		if bv, ok := b.(*object.NumberObj); ok {
			return av.Value == bv.Value
		}
	case *object.StringObj:
		if bv, ok := b.(*object.StringObj); ok {
			return av.Value == bv.Value
		}
	case *object.BoolObj:
		if bv, ok := b.(*object.BoolObj); ok {
			return av.Value == bv.Value
		}
	case *object.NullObj:
		_, ok := b.(*object.NullObj)
		return ok
	}
	return false
}

func execAdd(left, right object.Object) (object.Object, error) {
	// String concatenation
	if ls, ok := left.(*object.StringObj); ok {
		if rs, ok := right.(*object.StringObj); ok {
			return &object.StringObj{Value: ls.Value + rs.Value}, nil
		}
	}
	ln, lok := left.(*object.NumberObj)
	rn, rok := right.(*object.NumberObj)
	if lok && rok {
		return &object.NumberObj{Value: ln.Value + rn.Value}, nil
	}
	return nil, fmt.Errorf("Kosa: Operesheni '+' haiwezekani kwa aina hizi")
}

func execArith(op opcode.Opcode, left, right object.Object) (object.Object, error) {
	ln, lok := left.(*object.NumberObj)
	rn, rok := right.(*object.NumberObj)
	if !lok || !rok {
		return nil, fmt.Errorf("Kosa: Operesheni inahitaji nambari")
	}
	switch op {
	case opcode.OpSub:
		return &object.NumberObj{Value: ln.Value - rn.Value}, nil
	case opcode.OpMul:
		return &object.NumberObj{Value: ln.Value * rn.Value}, nil
	case opcode.OpDiv:
		if rn.Value == 0 {
			return nil, fmt.Errorf("Kosa: Haiwezekani kugawanya na sifuri")
		}
		return &object.NumberObj{Value: ln.Value / rn.Value}, nil
	case opcode.OpMod:
		if rn.Value == 0 {
			return nil, fmt.Errorf("Kosa: Haiwezekani kugawanya na sifuri")
		}
		return &object.NumberObj{Value: float64(int64(ln.Value) % int64(rn.Value))}, nil
	case opcode.OpPow:
		return &object.NumberObj{Value: math.Pow(ln.Value, rn.Value)}, nil
	}
	return nil, fmt.Errorf("Kosa: operesheni isiyojulikana")
}

func execComparison(op opcode.Opcode, left, right object.Object) (object.Object, error) {
	ln, lok := left.(*object.NumberObj)
	rn, rok := right.(*object.NumberObj)
	if !lok || !rok {
		return nil, fmt.Errorf("Kosa: Ulinganisho unahitaji nambari")
	}
	switch op {
	case opcode.OpGreaterThan:
		return &object.BoolObj{Value: ln.Value > rn.Value}, nil
	case opcode.OpLessThan:
		return &object.BoolObj{Value: ln.Value < rn.Value}, nil
	case opcode.OpGreaterEq:
		return &object.BoolObj{Value: ln.Value >= rn.Value}, nil
	case opcode.OpLessEq:
		return &object.BoolObj{Value: ln.Value <= rn.Value}, nil
	}
	return nil, fmt.Errorf("Kosa: ulinganisho usiojulikana")
}

func execMembership(needle, haystack object.Object) (object.Object, error) {
	switch h := haystack.(type) {
	case *object.ListObj:
		for _, el := range h.Elements {
			if objectsEqual(needle, el) {
				return &object.BoolObj{Value: true}, nil
			}
		}
		return &object.BoolObj{Value: false}, nil
	case *object.DictObj:
		key, ok := needle.(*object.StringObj)
		if !ok {
			return nil, fmt.Errorf("Kosa: Ufunguo wa kamusi lazima uwe tungo")
		}
		_, exists := h.Pairs[key.Value]
		return &object.BoolObj{Value: exists}, nil
	case *object.StringObj:
		sub, ok := needle.(*object.StringObj)
		if !ok {
			return nil, fmt.Errorf("Kosa: ktk kwa tungo inahitaji tungo")
		}
		return &object.BoolObj{Value: strings.Contains(h.Value, sub.Value)}, nil
	}
	return nil, fmt.Errorf("Kosa: ktk haiwezi kutumika kwa aina hii")
}

func execIndex(collection, index object.Object) (object.Object, error) {
	switch c := collection.(type) {
	case *object.ListObj:
		idx, ok := index.(*object.NumberObj)
		if !ok {
			return nil, fmt.Errorf("Kosa: Fahirisi lazima iwe nambari")
		}
		i := int(idx.Value)
		if i < 0 || i >= len(c.Elements) {
			return nil, fmt.Errorf("Kosa: Fahirisi %d nje ya masafa", i)
		}
		return c.Elements[i], nil
	case *object.DictObj:
		// Support string key access
		if key, ok := index.(*object.StringObj); ok {
			val, exists := c.Pairs[key.Value]
			if !exists {
				return &object.NullObj{}, nil
			}
			return val, nil
		}
		// Support numeric index (by key order) for iteration
		if idx, ok := index.(*object.NumberObj); ok {
			i := int(idx.Value)
			if i < 0 || i >= len(c.Keys) {
				return nil, fmt.Errorf("Kosa: Fahirisi %d nje ya masafa", i)
			}
			return &object.StringObj{Value: c.Keys[i]}, nil
		}
		return nil, fmt.Errorf("Kosa: Ufunguo wa kamusi lazima uwe tungo au nambari")
	case *object.StringObj:
		idx, ok := index.(*object.NumberObj)
		if !ok {
			return nil, fmt.Errorf("Kosa: Fahirisi lazima iwe nambari")
		}
		i := int(idx.Value)
		if i < 0 || i >= len(c.Value) {
			return nil, fmt.Errorf("Kosa: Fahirisi %d nje ya masafa", i)
		}
		return &object.StringObj{Value: string(c.Value[i])}, nil
	}
	return nil, fmt.Errorf("Kosa: haiwezi kufikia kwa fahirisi")
}

func execIndexAssign(collection, index, val object.Object) error {
	switch c := collection.(type) {
	case *object.ListObj:
		idx, ok := index.(*object.NumberObj)
		if !ok {
			return fmt.Errorf("Kosa: Fahirisi lazima iwe nambari")
		}
		i := int(idx.Value)
		if i < 0 || i >= len(c.Elements) {
			return fmt.Errorf("Kosa: Fahirisi %d nje ya masafa", i)
		}
		c.Elements[i] = val
	case *object.DictObj:
		key, ok := index.(*object.StringObj)
		if !ok {
			return fmt.Errorf("Kosa: Ufunguo wa kamusi lazima uwe tungo")
		}
		if _, exists := c.Pairs[key.Value]; !exists {
			c.Keys = append(c.Keys, key.Value)
		}
		c.Pairs[key.Value] = val
	default:
		return fmt.Errorf("Kosa: haiwezi kuweka kwa fahirisi")
	}
	return nil
}

func checkType(obj object.Object, tag uint8) bool {
	switch tag {
	case compiler.TypeNambari:
		_, ok := obj.(*object.NumberObj)
		return ok
	case compiler.TypeTungo:
		_, ok := obj.(*object.StringObj)
		return ok
	case compiler.TypeBuliani:
		_, ok := obj.(*object.BoolObj)
		return ok
	case compiler.TypeSafu:
		_, ok := obj.(*object.ListObj)
		return ok
	case compiler.TypeKamusi:
		_, ok := obj.(*object.DictObj)
		return ok
	}
	return true
}

func typeOfObj(obj object.Object) string {
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
	default:
		return "tupu"
	}
}

func typeTagName(tag uint8) string {
	switch tag {
	case compiler.TypeNambari:
		return "nambari"
	case compiler.TypeTungo:
		return "tungo"
	case compiler.TypeBuliani:
		return "buliani"
	case compiler.TypeSafu:
		return "safu"
	case compiler.TypeKamusi:
		return "kamusi"
	default:
		return "tupu"
	}
}
