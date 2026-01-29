package zir

// SymbolKind represents the kind of symbol
type SymbolKind int

const (
	SymbolType     SymbolKind = iota // Type definition (struct, primitive)
	SymbolVariable                   // Variable or parameter
	SymbolFunction                   // Function
)

// VariableUsage represents how a variable is initialized and used in the program (CPU-agnostic)
// Uses bitflags to track multiple usage patterns
type VariableUsage uint16

const (
	// Initialization flags (how the variable was initialized)
	VarInitNone       VariableUsage = 0
	VarInitArithmetic VariableUsage = 1 << 0 // Initialized with arithmetic expression
	VarInitPointer    VariableUsage = 1 << 1 // Initialized with pointer/struct/member access
	VarInitCounter    VariableUsage = 1 << 2 // Initialized in loop context
	VarInitIO         VariableUsage = 1 << 3 // Initialized from I/O operation
	VarInitConstant   VariableUsage = 1 << 4 // Initialized with constant/literal

	// Usage flags (how the variable is referenced/used after initialization)
	VarUsedArithmetic VariableUsage = 1 << 8  // Used in arithmetic operations
	VarUsedPointer    VariableUsage = 1 << 9  // Used for indirect addressing/dereferencing
	VarUsedCounter    VariableUsage = 1 << 10 // Used as loop counter or iteration variable
	VarUsedIO         VariableUsage = 1 << 11 // Used in I/O operations
	VarUsedComparison VariableUsage = 1 << 12 // Used in comparison operations
)

// HasFlag checks if a variable has a specific flag
func (vu VariableUsage) HasFlag(flag VariableUsage) bool {
	return (vu & flag) != 0
}

// AddFlag adds a flag
func (vu *VariableUsage) AddFlag(flag VariableUsage) {
	*vu |= flag
}

// Symbol represents a declared entity (variable, parameter, function, type)
type Symbol struct {
	Name          string
	QualifiedName string // Fully qualified name (e.g., "main.x", "<global>.count")
	Kind          SymbolKind
	Type          Type          // For variables/functions: their type. For type symbols: the type itself
	Usage         VariableUsage // How the variable is used (for register allocation hints)
}

// SymbolTable maintains symbols in a particular scope
type SymbolTable struct {
	symbols   map[string]*Symbol
	parent    *SymbolTable
	ScopeName string
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable(parent *SymbolTable, scopeName string) *SymbolTable {
	return &SymbolTable{
		symbols:   make(map[string]*Symbol),
		parent:    parent,
		ScopeName: scopeName,
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

// GetQualifiedName returns the fully qualified name for a variable in this scope
// e.g., "main.x", "helper.y", "main.block1.i"
func (st *SymbolTable) GetQualifiedName(variableName string) string {
	if st.ScopeName == "" {
		return variableName // Fallback for unnamed scopes
	}
	return st.ScopeName + "." + variableName
}
