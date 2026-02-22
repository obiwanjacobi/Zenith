package cfg

// 8-bit single registers
var RegA = Register{Name: "A", Size: 8, RegisterId: 7}
var RegB = Register{Name: "B", Size: 8, RegisterId: 0}
var RegC = Register{Name: "C", Size: 8, RegisterId: 1}
var RegD = Register{Name: "D", Size: 8, RegisterId: 2}
var RegE = Register{Name: "E", Size: 8, RegisterId: 3}
var RegH = Register{Name: "H", Size: 8, RegisterId: 4}
var RegL = Register{Name: "L", Size: 8, RegisterId: 5}
var RegF = Register{Name: "F", Size: 8, RegisterId: 6}

// 16-bit register pairs
var RegBC = Register{Name: "BC", Size: 16,
	Composition: []*Register{&RegC, &RegB}, RegisterId: 0}
var RegDE = Register{Name: "DE", Size: 16,
	Composition: []*Register{&RegE, &RegD}, RegisterId: 1}
var RegHL = Register{Name: "HL", Size: 16,
	Composition: []*Register{&RegL, &RegH}, RegisterId: 2}
var RegAF = Register{Name: "AF", Size: 16,
	Composition: []*Register{&RegF, &RegA}, RegisterId: 3}
var RegSP = Register{Name: "SP", Size: 16, RegisterId: 3}

// Z80Registers defines the available registers for Z80 architecture
// Includes both single 8-bit registers and 16-bit register pairs
var Z80Registers = []*Register{
	&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL,
	&RegBC, &RegDE, &RegHL, &RegAF, &RegSP,
}

// the 8-bit registers (A|B|C|D|E|H|L) that can be used for general purposes
var Z80Registers8 = []*Register{
	&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL,
}

// the 16-bit registers (BC|DE|HL) that can be used for general purposes
var Z80Registers16 = []*Register{
	&RegBC, &RegDE, &RegHL,
}

// registers (BC|DE|HL|AF) that can be pushed on the stack
var Z80RegistersQQ = []*Register{
	&RegBC, &RegDE, &RegHL, &RegAF,
}

// registers (BC|DE|HL|SP) that can be used for load operations
var Z80RegistersRR = []*Register{
	&RegBC, &RegDE, &RegHL, &RegSP,
}

// indirect alternative registers (BC|DE) to HL
var Z80RegistersPP = []*Register{
	&RegBC, &RegDE,
}

// AsPairs splits a 16-bit register into its low and high byte registers
// if the register is not a pair, returns the register itself as low and nil as high
func (reg *Register) AsPairs() (lowReg *Register, highReg *Register) {
	if len(reg.Composition) == 2 {
		lowReg = reg.Composition[0]
		highReg = reg.Composition[1]
	} else {
		lowReg = reg
		highReg = nil
	}
	return lowReg, highReg
}

// ToPairs splits 16-bit registers into its low and high byte registers
// if the register is not a pair, returns the register itself as low and high
func ToPairs(regs []*Register) (lowRegs []*Register, highRegs []*Register) {
	lowRegs = make([]*Register, 0)
	highRegs = make([]*Register, 0)

	for _, reg := range regs {
		low, high := reg.AsPairs()
		lowRegs = append(lowRegs, low)
		if high != nil {
			highRegs = append(highRegs, high)
		}
	}
	return lowRegs, highRegs
}

// callingConventionZ80 implements a standard calling convention for Z80
type callingConventionZ80 struct {
	registers []*Register
}

func NewCallingConventionZ80() CallingConvention {
	return &callingConventionZ80{
		registers: Z80Registers,
	}
}

func (cc *callingConventionZ80) GetParameterLocation(paramIndex int, paramSize uint8) (register *Register, stackOffset uint8, useStack bool) {
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
			return nil, uint8(2 + (paramIndex-2)*1), true
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
			return nil, uint8(2 + (paramIndex-2)*2), true
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
