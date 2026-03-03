package cfg

// CallingConvention defines how functions pass parameters and return values
type CallingConvention interface {
	// GetParameterLocation returns the register or stack location for a parameter
	// Returns (register, stackOffset, useStack)
	// If useStack is true, parameter is at [SP + stackOffset]
	// If useStack is false, parameter is in the returned register
	GetParameterLocation(paramIndex int, paramSize uint8) (register *Register, stackOffset uint16, useStack bool)

	// GetReturnValueRegister returns the register used for return values
	// For multi-value returns or large types, may need extension
	GetReturnValueRegister(returnSize uint8) *Register
}
