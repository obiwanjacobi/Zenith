package cfg

import "zenith/compiler/zir"

// RegisterPreference represents how well a register matches a variable's usage
type RegisterPreference struct {
	Register Register
	Score    int // Higher score = better match
}

// calculateRegisterPreference scores how well a register matches a variable's usage pattern
// This is Z80-specific logic that leverages variable usage flags
// variableSize is the bit width of the variable (8 or 16)
func calculateRegisterPreference(reg Register, usage zir.VariableUsage, variableSize int) int {
	score := 0

	// Base score for any register
	score = 10

	// Strong preference for matching register size to variable size
	if reg.Size == variableSize {
		score += 100 // Critical match
	} else if variableSize == 8 && reg.Size == 16 {
		score -= 50 // Can use pair for 8-bit but wasteful
	} else if variableSize == 16 && reg.Size == 8 {
		score -= 200 // Cannot use 8-bit register for 16-bit variable
	}

	// Z80-specific preferences based on register capabilities
	if isZ80RegisterPair(reg) {
		// 16-bit register pairs
		switch reg.Name {
		case "HL": // Best for 16-bit pointers
			if usage.HasFlag(zir.VarUsedPointer) {
				score += 80 // HL is THE pointer register for 16-bit (LD A,(HL); LD (HL),A; INC HL)
			}
			if usage.HasFlag(zir.VarUsedCounter) {
				score += 30 // Can use INC HL, DEC HL
			}
			if usage.HasFlag(zir.VarUsedArithmetic) {
				score += 20 // ADD HL,rr available
			}

		case "BC": // Good for 16-bit counters
			if usage.HasFlag(zir.VarUsedCounter) {
				score += 60 // BC is commonly used for loop counters
			}
			if usage.HasFlag(zir.VarUsedPointer) {
				score += 20 // Can use for pointers but less efficient than HL
			}

		case "DE": // General purpose 16-bit
			if usage.HasFlag(zir.VarUsedPointer) {
				score += 40 // Can be used for pointers, better than BC, worse than HL
			}
			if usage.HasFlag(zir.VarUsedCounter) {
				score += 30
			}
		}
	} else {
		// 8-bit single registers
		switch reg.Name {
		case "A": // Accumulator - best for 8-bit arithmetic and I/O
			if usage.HasFlag(zir.VarUsedArithmetic) {
				score += 60 // A has special arithmetic instructions (ADD A,r; SUB A,r)
			}
			if usage.HasFlag(zir.VarUsedIO) {
				score += 50 // A is required for IN/OUT instructions
			}
			if usage.HasFlag(zir.VarUsedComparison) {
				score += 40 // A is used in CP (compare) instructions
			}

		case "H", "L": // Components of HL pair - avoid using separately if possible
			if usage.HasFlag(zir.VarUsedPointer) {
				score += 30 // Better to use HL pair for pointers
			}
			if usage.HasFlag(zir.VarUsedArithmetic) {
				score += 10
			}

		case "B", "C": // Components of BC pair
			if usage.HasFlag(zir.VarUsedCounter) {
				score += 40 // B is used in DJNZ instruction
			}
			if usage.HasFlag(zir.VarUsedArithmetic) {
				score += 15
			}

		case "D", "E": // Components of DE pair - general purpose
			if usage.HasFlag(zir.VarUsedArithmetic) {
				score += 15
			}
			if usage.HasFlag(zir.VarUsedCounter) {
				score += 20
			}
		}
	}

	// Penalize mismatches
	if reg.Size == 8 && reg.Name == "A" && usage.HasFlag(zir.VarUsedPointer) {
		score -= 40 // A (single 8-bit) cannot be used for indirect addressing
	}

	return score
}

// selectBestRegister chooses the best register for a variable based on usage pattern
// variableSize is the bit width of the variable (8 or 16)
// Returns -1 if no suitable register is available
func selectBestRegister(
	variable string,
	usage zir.VariableUsage,
	variableSize int,
	availableRegisters []Register,
	usedColors map[int]bool,
) int {
	bestIdx := -1
	bestScore := -1

	// Score each available register
	for i, reg := range availableRegisters {
		if usedColors[i] {
			continue // This register is already used by a neighbor
		}

		score := calculateRegisterPreference(reg, usage, variableSize)
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
	availableRegisters []Register,
	usedColors map[int]bool,
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
	bestIdx := selectBestRegister(variable, usage, variableSize, availableRegisters, usedColors)
	if bestIdx >= 0 {
		return availableRegisters[bestIdx].Name, true
	}

	return "", false
}
