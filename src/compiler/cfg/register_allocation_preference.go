package cfg

import "zenith/compiler/zir"

// RegisterPreference represents how well a register matches a variable's usage
type RegisterPreference struct {
	Register Register
	Score    int // Higher score = better match
}

// calculateRegisterPreference scores how well a register matches a variable's usage pattern
// Uses the provided RegisterCapabilities to perform architecture-specific scoring
// variableSize is the bit width of the variable (8 or 16)
func calculateRegisterPreference(reg *Register, usage zir.VariableUsage, variableSize int, capabilities RegisterCapabilities) int {
	if capabilities == nil {
		// Fallback to generic scoring if no capabilities provided
		capabilities = &GenericRegisterCapabilities{}
	}
	return capabilities.ScoreRegisterForUsage(reg, usage, variableSize)
}

// selectBestRegister chooses the best register for a variable based on usage pattern
// variableSize is the bit width of the variable (8 or 16)
// Returns -1 if no suitable register is available
func selectBestRegister(
	variable string,
	usage zir.VariableUsage,
	variableSize int,
	availableRegisters []*Register,
	usedColors map[int]bool,
	capabilities RegisterCapabilities,
) int {
	bestIdx := -1
	bestScore := -1

	// Score each available register
	for i, reg := range availableRegisters {
		if usedColors[i] {
			continue // This register is already used by a neighbor
		}

		score := calculateRegisterPreference(reg, usage, variableSize, capabilities)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	return bestIdx
}

// Example: Simple allocation with preference-based selection
// This shows how you'd integrate usage-aware allocation into the coloring phase
func examplePreferenceBasedColoring(
	variable string,
	usage zir.VariableUsage,
	variableSize int,
	availableRegisters []*Register,
	usedColors map[int]bool,
	capabilities RegisterCapabilities,
) (registerName string, assigned bool) {
	// Instead of just picking the first available register:
	//   for i := 0; i < numColors; i++ {
	//       if !usedColors[i] {
	//           return availableRegisters[i].Name, true
	//       }
	//   }
	//
	// We select based on preference score (considering variable size and usage):
	// The variableSize would come from AllocationResult.VariableSizes[variable]
	// which is populated from the symbol's Type (e.g., zir.U8Type -> 8 bits, zir.U16Type -> 16 bits)
	bestIdx := selectBestRegister(variable, usage, variableSize, availableRegisters, usedColors, capabilities)
	if bestIdx >= 0 {
		return availableRegisters[bestIdx].Name, true
	}

	return "", false
}
