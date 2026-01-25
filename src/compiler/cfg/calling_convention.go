package cfg

// CallingConvention defines how functions pass parameters and return values
type CallingConvention interface {
	// GetParameterLocation returns the register or stack location for a parameter
	// Returns (register, stackOffset, useStack)
	// If useStack is true, parameter is at [SP + stackOffset]
	// If useStack is false, parameter is in the returned register
	GetParameterLocation(paramIndex int, paramSize int) (register *Register, stackOffset int, useStack bool)

	// GetReturnValueRegister returns the register used for return values
	// For multi-value returns or large types, may need extension
	GetReturnValueRegister(returnSize int) *Register

	// GetCallerSavedRegisters returns registers that caller must save before calls
	// These registers may be clobbered by the callee
	GetCallerSavedRegisters() []*Register

	// GetCalleeSavedRegisters returns registers that callee must preserve
	// If callee uses these, it must save/restore them in prologue/epilogue
	GetCalleeSavedRegisters() []*Register

	// GetStackAlignment returns the required stack alignment in bytes
	GetStackAlignment() int

	// GetStackGrowthDirection returns true if stack grows downward (toward lower addresses)
	GetStackGrowthDirection() bool
}
