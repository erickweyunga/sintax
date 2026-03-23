package opcode

import "fmt"

// Opcode represents a single bytecode instruction.
type Opcode byte

const (
	// Constants & Literals
	OpConstant Opcode = iota + 1
	OpNull
	OpTrue
	OpFalse

	// Arithmetic
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpPow

	// Comparison
	OpEqual
	OpNotEqual
	OpGreaterThan
	OpLessThan
	OpGreaterEq
	OpLessEq

	// Membership
	OpIn // ktk

	// Logical
	OpNot // si

	// Variables
	OpSetGlobal
	OpGetGlobal
	OpSetLocal
	OpGetLocal

	// Control Flow
	OpJump
	OpJumpIfFalse
	OpLoop

	// Functions
	OpCall
	OpReturn
	OpClosure

	// Upvalues
	OpGetUpvalue
	OpSetUpvalue
	OpCloseUpvalue

	// Print
	OpPrint // >>

	// Stack
	OpPop

	// Collections
	OpList
	OpDict
	OpIndex
	OpIndexAssign

	// Builtins
	OpCallBuiltin

	// Type checking
	OpCheckType

	// String interpolation
	OpInterpolate
)

// Definition holds the name and operand widths for an opcode.
type Definition struct {
	Name          string
	OperandWidths []int // width in bytes per operand
}

// Definitions maps each opcode to its definition.
var Definitions = map[Opcode]*Definition{
	OpConstant:     {"OpConstant", []int{2}},
	OpNull:         {"OpNull", []int{}},
	OpTrue:         {"OpTrue", []int{}},
	OpFalse:        {"OpFalse", []int{}},
	OpAdd:          {"OpAdd", []int{}},
	OpSub:          {"OpSub", []int{}},
	OpMul:          {"OpMul", []int{}},
	OpDiv:          {"OpDiv", []int{}},
	OpMod:          {"OpMod", []int{}},
	OpPow:          {"OpPow", []int{}},
	OpEqual:        {"OpEqual", []int{}},
	OpNotEqual:     {"OpNotEqual", []int{}},
	OpGreaterThan:  {"OpGreaterThan", []int{}},
	OpLessThan:     {"OpLessThan", []int{}},
	OpGreaterEq:    {"OpGreaterEq", []int{}},
	OpLessEq:       {"OpLessEq", []int{}},
	OpIn:           {"OpIn", []int{}},
	OpNot:          {"OpNot", []int{}},
	OpSetGlobal:    {"OpSetGlobal", []int{2}},
	OpGetGlobal:    {"OpGetGlobal", []int{2}},
	OpSetLocal:     {"OpSetLocal", []int{1}},
	OpGetLocal:     {"OpGetLocal", []int{1}},
	OpJump:         {"OpJump", []int{2}},
	OpJumpIfFalse:  {"OpJumpIfFalse", []int{2}},
	OpLoop:         {"OpLoop", []int{2}},
	OpCall:         {"OpCall", []int{1}},
	OpReturn:       {"OpReturn", []int{}},
	OpClosure:      {"OpClosure", []int{2, 1}},
	OpGetUpvalue:   {"OpGetUpvalue", []int{1}},
	OpSetUpvalue:   {"OpSetUpvalue", []int{1}},
	OpCloseUpvalue: {"OpCloseUpvalue", []int{}},
	OpPrint:        {"OpPrint", []int{}},
	OpPop:          {"OpPop", []int{}},
	OpList:         {"OpList", []int{2}},
	OpDict:         {"OpDict", []int{2}},
	OpIndex:        {"OpIndex", []int{}},
	OpIndexAssign:  {"OpIndexAssign", []int{}},
	OpCallBuiltin:  {"OpCallBuiltin", []int{1, 1}},
	OpCheckType:    {"OpCheckType", []int{1}},
	OpInterpolate:  {"OpInterpolate", []int{1}},
}

// Make creates a bytecode instruction from an opcode and operands.
func Make(op Opcode, operands ...int) []byte {
	def, ok := Definitions[op]
	if !ok {
		return []byte{}
	}

	instructionLen := 1
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		if i >= len(def.OperandWidths) {
			break
		}
		width := def.OperandWidths[i]
		switch width {
		case 2:
			instruction[offset] = byte(o >> 8)
			instruction[offset+1] = byte(o)
		case 1:
			instruction[offset] = byte(o)
		}
		offset += width
	}

	return instruction
}

// ReadUint16 reads a big-endian uint16 from instructions at offset.
func ReadUint16(ins []byte, offset int) uint16 {
	return uint16(ins[offset])<<8 | uint16(ins[offset+1])
}

// ReadUint8 reads a uint8 from instructions at offset.
func ReadUint8(ins []byte, offset int) uint8 {
	return ins[offset]
}

func (op Opcode) String() string {
	if def, ok := Definitions[op]; ok {
		return def.Name
	}
	return fmt.Sprintf("UNKNOWN(%d)", op)
}
