package zir

// SymbolKind represents the kind of symbol
type SymbolKind int

const (
	SymbolType     SymbolKind = iota // Type definition (struct, primitive)
	SymbolVariable                   // Variable or parameter
	SymbolFunction                   // Function
)

// VariableUsage represents how a variable is used in the program (CPU-agnostic)
type VariableUsage int

const (
	VariableUsageGeneral    VariableUsage = iota
	VariableUsagePointer                  // Used for indirect addressing/dereferencing
	VariableUsageArithmetic               // Used in arithmetic operations
	VariableUsageCounter                  // Used as loop counter or iteration variable
	VariableUsageIO                       // Used in I/O operations
)

func (vu VariableUsage) String() string {
	switch vu {
	case VariableUsageGeneral:
		return "general"
	case VariableUsagePointer:
		return "pointer"
	case VariableUsageArithmetic:
		return "arithmetic"
	case VariableUsageCounter:
		return "counter"
	case VariableUsageIO:
		return "io"
	default:
		return "unknown"
	}
}

// Symbol represents a declared entity (variable, parameter, function, type)
type Symbol struct {
	Name   string
	Kind   SymbolKind
	Type   Type          // For variables/functions: their type. For type symbols: the type itself
	Offset int           // Stack offset or memory address (computed during layout)
	Usage  VariableUsage // How the variable is used (for register allocation hints)
}

// SymbolTable maintains symbols in a particular scope
type SymbolTable struct {
	symbols map[string]*Symbol
	parent  *SymbolTable
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		symbols: make(map[string]*Symbol),
		parent:  parent,
	}
}

// Add adds a symbol to this scope
func (st *SymbolTable) Add(symbol *Symbol) bool {
	if _, exists := st.symbols[symbol.Name]; exists {
		return false // Symbol already exists in this scope
	}
	st.symbols[symbol.Name] = symbol
	return true
}

// Lookup finds a symbol in this scope or parent scopes
func (st *SymbolTable) Lookup(name string) *Symbol {
	if symbol, ok := st.symbols[name]; ok {
		return symbol
	}
	if st.parent != nil {
		return st.parent.Lookup(name)
	}
	return nil
}

// LookupLocal finds a symbol only in this scope (not parents)
func (st *SymbolTable) LookupLocal(name string) *Symbol {
	return st.symbols[name]
}

// IsGlobal returns true if this is the global scope
func (st *SymbolTable) IsGlobal() bool {
	return st.parent == nil
}

// Parent returns the parent symbol table
func (st *SymbolTable) Parent() *SymbolTable {
	return st.parent
}

// Symbols returns all symbols in this scope
func (st *SymbolTable) Symbols() map[string]*Symbol {
	return st.symbols
}
