package zir

// Symbol represents a declared entity (variable, parameter, function, type)
type Symbol struct {
	Name   string
	Type   Type
	Offset int // Stack offset or memory address (computed during layout)
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
