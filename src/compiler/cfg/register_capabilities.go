package cfg

import "zenith/compiler/zir"

// RegisterCapabilities defines architecture-specific register capabilities
type RegisterCapabilities interface {
	// ScoreRegisterForUsage returns a score (higher = better) for how well
	// a register matches a variable's usage pattern and size requirements
	ScoreRegisterForUsage(reg *Register, usage zir.VariableUsage, variableSize int) int

	// IsRegisterPair returns true if the register is a multi-byte register pair
	IsRegisterPair(reg *Register) bool
}

// GenericRegisterCapabilities provides basic scoring without architecture-specific optimizations
type GenericRegisterCapabilities struct{}

func (c *GenericRegisterCapabilities) ScoreRegisterForUsage(reg *Register, usage zir.VariableUsage, variableSize int) int {
	score := 10 // Base score

	// Strong preference for matching register size to variable size
	if reg.Size == variableSize {
		score += 100 // Critical match
	} else if variableSize == 8 && reg.Size == 16 {
		score -= 50 // Can use pair for 8-bit but wasteful
	} else if variableSize == 16 && reg.Size == 8 {
		score -= 200 // Cannot use 8-bit register for 16-bit variable
	}

	return score
}

func (c *GenericRegisterCapabilities) IsRegisterPair(reg *Register) bool {
	return reg.Size == 16
}
