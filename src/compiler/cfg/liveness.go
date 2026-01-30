package cfg

import (
	"zenith/compiler/zsm"
)

// LivenessInfo contains liveness analysis results for a CFG
type LivenessInfo struct {
	// Live-in sets: variables live at block entry
	LiveIn map[int]map[string]bool

	// Live-out sets: variables live at block exit
	LiveOut map[int]map[string]bool

	// Use sets: variables used before being defined in block
	Use map[int]map[string]bool

	// Def sets: variables defined in block
	Def map[int]map[string]bool
}

// NewLivenessInfo creates a new liveness analysis result
func NewLivenessInfo() *LivenessInfo {
	return &LivenessInfo{
		LiveIn:  make(map[int]map[string]bool),
		LiveOut: make(map[int]map[string]bool),
		Use:     make(map[int]map[string]bool),
		Def:     make(map[int]map[string]bool),
	}
}

// ComputeLiveness performs liveness analysis on a CFG
func ComputeLiveness(cfg *CFG) *LivenessInfo {
	info := NewLivenessInfo()

	// Step 1: Compute use and def sets for each block
	for _, block := range cfg.Blocks {
		info.Use[block.ID] = make(map[string]bool)
		info.Def[block.ID] = make(map[string]bool)
		info.LiveIn[block.ID] = make(map[string]bool)
		info.LiveOut[block.ID] = make(map[string]bool)

		computeUseDefSets(block, cfg.FunctionName, info.Use[block.ID], info.Def[block.ID])
	}

	// Step 2: Iterate until live-in/live-out sets converge
	changed := true
	for changed {
		changed = false

		// Process blocks in reverse order (better convergence)
		for i := len(cfg.Blocks) - 1; i >= 0; i-- {
			block := cfg.Blocks[i]

			// Compute live-out: union of live-in of all successors
			newLiveOut := make(map[string]bool)
			for _, succ := range block.Successors {
				for varName := range info.LiveIn[succ.ID] {
					newLiveOut[varName] = true
				}
			}

			// Compute live-in: use ∪ (live-out - def)
			newLiveIn := make(map[string]bool)
			for varName := range info.Use[block.ID] {
				newLiveIn[varName] = true
			}
			for varName := range newLiveOut {
				if !info.Def[block.ID][varName] {
					newLiveIn[varName] = true
				}
			}

			// Check if sets changed
			if !setsEqual(info.LiveIn[block.ID], newLiveIn) ||
				!setsEqual(info.LiveOut[block.ID], newLiveOut) {
				changed = true
				info.LiveIn[block.ID] = newLiveIn
				info.LiveOut[block.ID] = newLiveOut
			}
		}
	}

	return info
}

// computeUseDefSets analyzes a basic block to find used and defined variables
func computeUseDefSets(block *BasicBlock, scopeName string, use, def map[string]bool) {
	for _, stmt := range block.Instructions {
		// Get variables used by this statement (before any definitions)
		used := getUsedVariables(stmt, scopeName)
		for _, varName := range used {
			if !def[varName] {
				use[varName] = true
			}
		}

		// Get variables defined by this statement
		defined := getDefinedVariables(stmt, scopeName)
		for _, varName := range defined {
			def[varName] = true
		}
	}
}

// getUsedVariables returns the names of variables read by a statement
func getUsedVariables(stmt zsm.SemStatement, scopeName string) []string {
	var used []string

	switch s := stmt.(type) {
	case *zsm.SemVariableDecl:
		// Initializer uses variables
		if s.Initializer != nil {
			used = append(used, getUsedInExpression(s.Initializer, scopeName)...)
		}

	case *zsm.SemAssignment:
		// Right side uses variables
		used = append(used, getUsedInExpression(s.Value, scopeName)...)

	case *zsm.SemExpressionStmt:
		// Expression may use variables
		used = append(used, getUsedInExpression(s.Expression, scopeName)...)

	case *zsm.SemReturn:
		// Return value uses variables
		if s.Value != nil {
			used = append(used, getUsedInExpression(s.Value, scopeName)...)
		}
	}

	return used
}

// getDefinedVariables returns the names of variables written by a statement
// Uses fully qualified names from Symbol.QualifiedName
func getDefinedVariables(stmt zsm.SemStatement, scopeName string) []string {
	var defined []string

	switch s := stmt.(type) {
	case *zsm.SemVariableDecl:
		// Variable declaration defines a variable
		defined = append(defined, s.Symbol.QualifiedName)

	case *zsm.SemAssignment:
		// Assignment defines the target variable
		defined = append(defined, s.Target.QualifiedName)
	}

	return defined
}

// getUsedInExpression recursively extracts variable names used in an expression
// Uses fully qualified names from Symbol.QualifiedName
func getUsedInExpression(expr zsm.SemExpression, scopeName string) []string {
	if expr == nil {
		return nil
	}

	var used []string

	switch e := expr.(type) {
	case *zsm.SemSymbolRef:
		// Symbol reference uses the variable (with its qualified name)
		used = append(used, e.Symbol.QualifiedName)

	case *zsm.SemBinaryOp:
		// Binary operation uses both operands
		used = append(used, getUsedInExpression(e.Left, scopeName)...)
		used = append(used, getUsedInExpression(e.Right, scopeName)...)

	case *zsm.SemUnaryOp:
		// Unary operation uses its operand
		used = append(used, getUsedInExpression(e.Operand, scopeName)...)

	case *zsm.SemFunctionCall:
		// Function call uses all arguments
		for _, arg := range e.Arguments {
			used = append(used, getUsedInExpression(arg, scopeName)...)
		}

	case *zsm.SemConstant:
		// Constants don't use variables

	case *zsm.SemMemberAccess:
		// Member access uses the base object
		if e.Object != nil {
			used = append(used, getUsedInExpression(*e.Object, scopeName)...)
		}

	case *zsm.SemTypeInitializer:
		// Type initializer uses field values
		for _, field := range e.Fields {
			used = append(used, getUsedInExpression(field.Value, scopeName)...)
		}
	}

	return used
}

// setsEqual checks if two string sets are equal
func setsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if !b[key] {
			return false
		}
	}
	return true
}

// GetLiveRanges computes live ranges for each variable
// Returns map of variable name to list of block IDs where it's live
func (info *LivenessInfo) GetLiveRanges() map[string][]int {
	ranges := make(map[string][]int)

	// Collect all blocks where each variable is live
	for blockID, liveVars := range info.LiveIn {
		for varName := range liveVars {
			ranges[varName] = append(ranges[varName], blockID)
		}
	}

	for blockID, liveVars := range info.LiveOut {
		for varName := range liveVars {
			// Add if not already present
			found := false
			for _, id := range ranges[varName] {
				if id == blockID {
					found = true
					break
				}
			}
			if !found {
				ranges[varName] = append(ranges[varName], blockID)
			}
		}
	}

	return ranges
}

// IsLiveAt checks if a variable is live at the entry of a block
func (info *LivenessInfo) IsLiveAt(varName string, blockID int) bool {
	return info.LiveIn[blockID][varName]
}

// IsLiveOutOf checks if a variable is live at the exit of a block
func (info *LivenessInfo) IsLiveOutOf(varName string, blockID int) bool {
	return info.LiveOut[blockID][varName]
}
