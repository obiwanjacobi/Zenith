package cfg

// 8-bit single registers
var RegA = Register{Name: "A", Size: 8, Class: RegisterClassAccumulator, RegisterId: 7}
var RegB = Register{Name: "B", Size: 8, Class: RegisterClassGeneral, RegisterId: 0}
var RegC = Register{Name: "C", Size: 8, Class: RegisterClassGeneral, RegisterId: 1}
var RegD = Register{Name: "D", Size: 8, Class: RegisterClassGeneral, RegisterId: 2}
var RegE = Register{Name: "E", Size: 8, Class: RegisterClassGeneral, RegisterId: 3}
var RegH = Register{Name: "H", Size: 8, Class: RegisterClassGeneral, RegisterId: 4}
var RegL = Register{Name: "L", Size: 8, Class: RegisterClassGeneral, RegisterId: 5}
var RegF = Register{Name: "F", Size: 8, Class: RegisterClassFlags, RegisterId: 6}

// 16-bit register pairs
var RegBC = Register{Name: "BC", Size: 16, Class: RegisterClassGeneral,
	Composition: []*Register{&RegB, &RegC}, RegisterId: 0}
var RegDE = Register{Name: "DE", Size: 16, Class: RegisterClassGeneral,
	Composition: []*Register{&RegD, &RegE}, RegisterId: 1}
var RegHL = Register{Name: "HL", Size: 16, Class: RegisterClassIndex,
	Composition: []*Register{&RegH, &RegL}, RegisterId: 2}
var RegAF = Register{Name: "AF", Size: 16, Class: RegisterClassAccumulator,
	Composition: []*Register{&RegA, &RegF}, RegisterId: 3}
var RegSP = Register{Name: "SP", Size: 16, Class: RegisterClassStackPointer, RegisterId: 3}

// Z80Registers defines the available registers for Z80 architecture
// Includes both single 8-bit registers and 16-bit register pairs
var Z80Registers = []*Register{
	&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL, &RegF,
	&RegBC, &RegDE, &RegHL, &RegAF, &RegSP,
}

// callingConventionZ80 implements a standard calling convention for Z80
type callingConventionZ80 struct {
	registers []*Register
}

// NewZ80CallingConvention creates a Z80 calling convention
// Parameters:
//   - 1st 16-bit or two 8-bit: HL (H=high byte, L=low byte)
//   - 2nd 16-bit or two 8-bit: DE (D=high byte, E=low byte)
//   - 3rd 16-bit or two 8-bit: BC (B=high byte, C=low byte)
//   - Additional params: Stack (growing downward)
//
// Return values:
//   - 8-bit: A
//   - 16-bit: HL
//
// Caller-saved (volatile): AF, BC, DE, HL
// Callee-saved (non-volatile): IX, IY (if available)
func NewCallingConventionZ80() CallingConvention {
	return &callingConventionZ80{
		registers: Z80Registers,
	}
}

func (cc *callingConventionZ80) GetParameterLocation(paramIndex int, paramSize int) (register *Register, stackOffset int, useStack bool) {
	// Map parameter indices to register pairs
	// For 8-bit params, use the low byte of the pair
	var regName string

	if paramSize == 16 {
		// 16-bit parameters
		switch paramIndex {
		case 0:
			regName = "HL"
		case 1:
			regName = "DE"
		case 2:
			regName = "BC"
		default:
			// Stack parameters start after return address (2 bytes)
			// Stack grows downward, params accessed as [SP + offset]
			return nil, 2 + (paramIndex-3)*2, true
		}
	} else {
		// 8-bit parameters use low byte of register pairs
		switch paramIndex {
		case 0:
			regName = "L"
		case 1:
			regName = "E"
		case 2:
			regName = "C"
		default:
			// Stack parameters
			return nil, 2 + (paramIndex-3)*1, true
		}
	}

	// Find register by name
	for _, reg := range cc.registers {
		if reg.Name == regName {
			return reg, 0, false
		}
	}

	// Fallback to stack if register not found
	return nil, 2 + paramIndex*2, true
}

func (cc *callingConventionZ80) GetReturnValueRegister(returnSize int) *Register {
	var regName string
	if returnSize == 8 {
		regName = "A"
	} else {
		regName = "HL"
	}

	for _, reg := range cc.registers {
		if reg.Name == regName {
			return reg
		}
	}
	return nil
}

func (cc *callingConventionZ80) GetCallerSavedRegisters() []*Register {
	// Caller must save: AF, BC, DE, HL (all general-purpose registers)
	callerSaved := make([]*Register, 0)
	volatileNames := map[string]bool{
		"A": true, "F": true,
		"B": true, "C": true, "BC": true,
		"D": true, "E": true, "DE": true,
		"H": true, "L": true, "HL": true,
	}

	for _, reg := range cc.registers {
		if volatileNames[reg.Name] {
			callerSaved = append(callerSaved, reg)
		}
	}
	return callerSaved
}

func (cc *callingConventionZ80) GetCalleeSavedRegisters() []*Register {
	// Callee must preserve: IX, IY (if we use them)
	// For now, return empty since we're not using IX/IY
	return []*Register{}
}

func (cc *callingConventionZ80) GetStackAlignment() int {
	// Z80 doesn't have strict alignment requirements
	return 1
}

func (cc *callingConventionZ80) GetStackGrowthDirection() bool {
	// Z80 stack grows downward (toward lower addresses)
	return true
}
