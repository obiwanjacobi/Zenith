package z80

// callingConventionZ80 implements a standard calling convention for Z80
type callingConventionZ80 struct {
	registers []*Register
}

func NewCallingConventionZ80() CallingConvention {
	return &callingConventionZ80{
		registers: Z80Registers,
	}
}

func (cc *callingConventionZ80) GetParameterLocation(paramIndex int, paramSize uint8) (register *Register, stackOffset uint16, useStack bool) {
	switch paramSize {
	case 8:
		// 8-bit parameters
		switch paramIndex {
		case 0:
			return &RegE, 0, false
		case 1:
			return &RegC, 0, false
		default:
			// Stack parameters start after return address (2 bytes)
			// Stack grows downward, params accessed as [SP + offset]
			return nil, uint16(2 + (paramIndex-2)*1), true
		}
	case 16:
		// 16-bit parameters
		switch paramIndex {
		case 0:
			return &RegDE, 0, false
		case 1:
			return &RegBC, 0, false
		default:
			// Stack parameters start after return address (2 bytes)
			// Stack grows downward, params accessed as [SP + offset]
			return nil, uint16(2 + (paramIndex-2)*2), true
		}
	}

	// error
	return nil, 0, false
}

func (cc *callingConventionZ80) GetReturnValueRegister(returnSize uint8) *Register {
	switch returnSize {
	case 8:
		return &RegA
	case 16:
		return &RegDE
	}

	return nil
}
