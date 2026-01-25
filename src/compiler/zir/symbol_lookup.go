package zir

// SymbolLookup provides symbol information for register allocation
// Maps qualified symbol names to their type information
type SymbolLookup struct {
	symbols map[string]*Symbol
}

// NewSymbolLookup creates a new symbol lookup from a compilation unit
func NewSymbolLookup(cu *IRCompilationUnit) *SymbolLookup {
	lookup := &SymbolLookup{
		symbols: make(map[string]*Symbol),
	}

	// Collect all symbols from global scope
	lookup.collectSymbols(cu.GlobalScope)

	// Collect symbols from function scopes
	for _, decl := range cu.Declarations {
		if funcDecl, ok := decl.(*IRFunctionDecl); ok {
			lookup.collectSymbols(funcDecl.Scope)
		}
	}

	return lookup
}

// collectSymbols recursively collects symbols from a scope and its children
func (sl *SymbolLookup) collectSymbols(scope *SymbolTable) {
	if scope == nil {
		return
	}

	// Add all symbols from this scope
	for _, symbol := range scope.Symbols() {
		sl.symbols[symbol.QualifiedName] = symbol
	}

	// Note: We don't recursively walk children because function scopes
	// are collected separately in NewSymbolLookup
}

// GetTypeSize returns the size of a symbol's type in bits (8 or 16)
func (sl *SymbolLookup) GetTypeSize(qualifiedName string) int {
	symbol, ok := sl.symbols[qualifiedName]
	if !ok {
		return 8 // Default to 8-bit if not found
	}

	return getTypeSizeInBits(symbol.Type)
}

// GetUsage returns the usage pattern of a symbol
func (sl *SymbolLookup) GetUsage(qualifiedName string) VariableUsage {
	symbol, ok := sl.symbols[qualifiedName]
	if !ok {
		return VarInitNone // Default if not found
	}

	return symbol.Usage
}

// getTypeSizeInBits returns the size of a type in bits
func getTypeSizeInBits(typ Type) int {
	if typ == nil {
		return 8
	}

	switch typ {
	case U8Type, I8Type, BoolType:
		return 8
	case U16Type, I16Type:
		return 16
	default:
		// For struct types, arrays, etc., would need more logic
		// For now, treat as 8-bit
		return 8
	}
}
