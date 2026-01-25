package cfg

import (
	"fmt"
	
	"zenith/compiler/zir"
)

// getTypeSizeInBits returns the size of a type in bits
// This is used to match variables to appropriately-sized registers
func getTypeSizeInBits(typ zir.Type) int {
	if typ == nil {
		return 8 // Default to 8-bit
	}

	switch typ {
	case zir.U8Type, zir.I8Type, zir.BoolType:
		return 8
	case zir.U16Type, zir.I16Type:
		return 16
	default:
		// For struct types, arrays, etc., this would need more logic
		// For now, treat unknown types as 8-bit
		return 8
	}
}

// getQualifiedVariableName creates a fully qualified variable name
// to distinguish variables across different scopes/functions
// Format: "functionName.variableName"
func getQualifiedVariableName(functionName string, variableName string) string {
	return fmt.Sprintf("%s.%s", functionName, variableName)
}

// Example: How to populate VariableSizes from a symbol table
// This would be called when building the allocation result
//
// func populateVariableSizes(result *AllocationResult, symbolTable *zir.SymbolTable, variables []string) {
//     for _, varName := range variables {
//         symbol := symbolTable.Lookup(varName)
//         if symbol != nil && symbol.Kind == zir.SymbolVariable {
//             result.VariableSizes[varName] = getTypeSizeInBits(symbol.Type)
//             result.VariableUsages[varName] = symbol.Usage
//         }
//     }
// }
