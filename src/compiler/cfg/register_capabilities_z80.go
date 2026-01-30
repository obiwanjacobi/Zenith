package cfg

import "zenith/compiler/zsm"

// Z80RegisterCapabilities provides Z80-specific register capability scoring
type Z80RegisterCapabilities struct {
	GenericRegisterCapabilities
}

func (c *Z80RegisterCapabilities) ScoreRegisterForUsage(reg *Register, usage zsm.VariableUsage, variableSize int) int {
	// Start with generic scoring (handles size matching)
	score := c.GenericRegisterCapabilities.ScoreRegisterForUsage(reg, usage, variableSize)

	// Add Z80-specific preferences based on register capabilities
	if c.IsRegisterPair(reg) {
		// 16-bit register pairs
		switch reg.Name {
		case "HL": // Best for 16-bit pointers
			if usage.HasFlag(zsm.VarUsedPointer) {
				score += 80 // HL is THE pointer register for 16-bit (LD A,(HL); LD (HL),A; INC HL)
			}
			if usage.HasFlag(zsm.VarUsedCounter) {
				score += 30 // Can use INC HL, DEC HL
			}
			if usage.HasFlag(zsm.VarUsedArithmetic) {
				score += 20 // ADD HL,rr available
			}

		case "BC": // Good for 16-bit counters
			if usage.HasFlag(zsm.VarUsedCounter) {
				score += 60 // BC is commonly used for loop counters
			}
			if usage.HasFlag(zsm.VarUsedPointer) {
				score += 20 // Can use for pointers but less efficient than HL
			}

		case "DE": // General purpose 16-bit
			if usage.HasFlag(zsm.VarUsedPointer) {
				score += 40 // Can be used for pointers, better than BC, worse than HL
			}
			if usage.HasFlag(zsm.VarUsedCounter) {
				score += 30
			}
		}
	} else {
		// 8-bit single registers
		switch reg.Name {
		case "A": // Accumulator - best for 8-bit arithmetic and I/O
			if usage.HasFlag(zsm.VarUsedArithmetic) {
				score += 60 // A has special arithmetic instructions (ADD A,r; SUB A,r)
			}
			if usage.HasFlag(zsm.VarUsedIO) {
				score += 50 // A is required for IN/OUT instructions
			}
			if usage.HasFlag(zsm.VarUsedComparison) {
				score += 40 // A is used in CP (compare) instructions
			}

		case "H", "L": // Components of HL pair - avoid using separately if possible
			if usage.HasFlag(zsm.VarUsedPointer) {
				score += 30 // Better to use HL pair for pointers
			}
			if usage.HasFlag(zsm.VarUsedArithmetic) {
				score += 10
			}

		case "B", "C": // Components of BC pair
			if usage.HasFlag(zsm.VarUsedCounter) {
				score += 40 // B is used in DJNZ instruction
			}
			if usage.HasFlag(zsm.VarUsedArithmetic) {
				score += 15
			}

		case "D", "E": // Components of DE pair - general purpose
			if usage.HasFlag(zsm.VarUsedArithmetic) {
				score += 15
			}
			if usage.HasFlag(zsm.VarUsedCounter) {
				score += 20
			}
		}
	}

	// Penalize mismatches
	if reg.Size == 8 && reg.Name == "A" && usage.HasFlag(zsm.VarUsedPointer) {
		score -= 40 // A (single 8-bit) cannot be used for indirect addressing
	}

	return score
}

func (c *Z80RegisterCapabilities) IsRegisterPair(reg *Register) bool {
	return reg.Size == 16 && (reg.Name == "BC" || reg.Name == "DE" || reg.Name == "HL")
}
