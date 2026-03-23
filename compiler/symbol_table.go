package compiler

// SymbolScope represents the scope of a symbol.
type SymbolScope string

const (
	GlobalScope  SymbolScope = "GLOBAL"
	LocalScope   SymbolScope = "LOCAL"
	UpvalueScope SymbolScope = "UPVALUE"
	BuiltinScope SymbolScope = "BUILTIN"
)

// Type tags for type enforcement.
const (
	TypeUntyped uint8 = iota
	TypeNambari
	TypeTungo
	TypeBuliani
	TypeSafu
	TypeKamusi
)

// TypeTagFromName converts a type name string to a type tag.
func TypeTagFromName(name string) uint8 {
	switch name {
	case "nambari":
		return TypeNambari
	case "tungo":
		return TypeTungo
	case "buliani":
		return TypeBuliani
	case "safu":
		return TypeSafu
	case "kamusi":
		return TypeKamusi
	default:
		return TypeUntyped
	}
}

// Symbol represents a resolved variable.
type Symbol struct {
	Name    string
	Scope   SymbolScope
	Index   int
	TypeTag uint8
}

// SymbolTable tracks variable bindings and scope resolution.
type SymbolTable struct {
	store    map[string]Symbol
	numDefs  int
	Outer    *SymbolTable
	Upvalues []Symbol
}

// NewSymbolTable creates a new top-level symbol table.
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		store:    make(map[string]Symbol),
		Upvalues: []Symbol{},
	}
}

// NewEnclosedSymbolTable creates a child symbol table.
func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}

// Define creates a new binding in the current scope.
func (s *SymbolTable) Define(name string) Symbol {
	sym := Symbol{Name: name, Index: s.numDefs}
	if s.Outer == nil {
		sym.Scope = GlobalScope
	} else {
		sym.Scope = LocalScope
	}
	s.store[name] = sym
	s.numDefs++
	return sym
}

// DefineTyped creates a new typed binding.
func (s *SymbolTable) DefineTyped(name string, typeTag uint8) Symbol {
	sym := s.Define(name)
	sym.TypeTag = typeTag
	s.store[name] = sym
	return sym
}

// DefineBuiltin registers a built-in function.
func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	sym := Symbol{Name: name, Scope: BuiltinScope, Index: index}
	s.store[name] = sym
	return sym
}

// DefineFunctionName defines the function name in the current scope (for recursion).
func (s *SymbolTable) DefineFunctionName(name string) Symbol {
	sym := Symbol{Name: name, Scope: LocalScope, Index: 0}
	s.store[name] = sym
	return sym
}

// Resolve looks up a symbol by name, walking the scope chain.
func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	sym, ok := s.store[name]
	if ok {
		return sym, true
	}
	if s.Outer == nil {
		return sym, false
	}

	sym, ok = s.Outer.Resolve(name)
	if !ok {
		return sym, false
	}

	// If found in an outer local/upvalue scope, create an upvalue
	if sym.Scope == GlobalScope || sym.Scope == BuiltinScope {
		return sym, true
	}

	return s.defineUpvalue(sym), true
}

func (s *SymbolTable) defineUpvalue(original Symbol) Symbol {
	s.Upvalues = append(s.Upvalues, original)
	sym := Symbol{
		Name:  original.Name,
		Scope: UpvalueScope,
		Index: len(s.Upvalues) - 1,
	}
	s.store[original.Name] = sym
	return sym
}

// NumDefinitions returns the number of definitions in this scope.
func (s *SymbolTable) NumDefinitions() int {
	return s.numDefs
}
