package cfg

// Z80 Instruction Descriptors
// Static instances of InstrDescriptor for all Z80 opcodes

// ============================================================================
// 8-bit Load Instructions
// ============================================================================

var InstrDesc_LD_R_R = InstrDescriptor{
	Opcode:   Z80_LD_R_R,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_R_N = InstrDescriptor{
	Opcode:   Z80_LD_R_N,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_R_HL = InstrDescriptor{
	Opcode:   Z80_LD_R_HL,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_HL_R = InstrDescriptor{
	Opcode:   Z80_LD_HL_R,
	Category: CatStore,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessWrite, Registers: []*Register{&RegHL}},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_HL_N = InstrDescriptor{
	Opcode:   Z80_LD_HL_N,
	Category: CatStore,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessWrite, Registers: []*Register{&RegHL}},
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         10,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// 8-bit access to memory via register pairs
// ============================================================================

var InstrDesc_LD_A_PP = InstrDescriptor{
	Opcode:   Z80_LD_A_PP,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessWrite, Registers: []*Register{&RegA}},
		{Type: OpRegisterPairPP, Access: AccessRead, Registers: []*Register{&RegBC, &RegDE}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 4,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_A_NN = InstrDescriptor{
	Opcode:   Z80_LD_A_NN,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessWrite, Registers: []*Register{&RegA}},
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         13,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_PP_A = InstrDescriptor{
	Opcode:   Z80_LD_PP_A,
	Category: CatStore,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairPP, Access: AccessWrite, Registers: []*Register{&RegBC, &RegDE}},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 4,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_NN_A = InstrDescriptor{
	Opcode:   Z80_LD_NN_A,
	Category: CatStore,
	Dependencies: []InstrDependency{
		{Type: OpConstant16, Access: AccessRead},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA}},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         13,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// 16-bit Load Instructions
// ============================================================================

var InstrDesc_LD_RR_NN = InstrDescriptor{
	Opcode:   Z80_LD_RR_NN,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessWrite, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         10,
	CyclesTaken:    0,
	EncodingReg1SL: 4,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_HL_NN = InstrDescriptor{
	Opcode:   Z80_LD_HL_NN,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessWrite, Registers: []*Register{&RegHL}},
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         16,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_RR_NN_ADDR = InstrDescriptor{
	Opcode:   Z80_LD_RR_NN_ADDR,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessWrite, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         20,
	CyclesTaken:    0,
	EncodingReg1SL: 4,
	EncodingReg2SL: 0,
	Prefix1:        0xED,
	Prefix2:        0,
}

var InstrDesc_LD_NN_HL = InstrDescriptor{
	Opcode:   Z80_LD_NN_HL,
	Category: CatStore,
	Dependencies: []InstrDependency{
		{Type: OpConstant16, Access: AccessRead},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         16,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_LD_NN_RR_ADDR = InstrDescriptor{
	Opcode:   Z80_LD_NN_RR_ADDR,
	Category: CatStore,
	Dependencies: []InstrDependency{
		{Type: OpConstant16, Access: AccessRead},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         20,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 4,
	Prefix1:        0xED,
	Prefix2:        0,
}

var InstrDesc_LD_SP_HL = InstrDescriptor{
	Opcode:   Z80_LD_SP_HL,
	Category: CatLoad,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessWrite, Registers: []*Register{&RegSP}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         6,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// 8-bit Arithmetic Instructions
// ============================================================================

var InstrDesc_ADD_A_R = InstrDescriptor{
	Opcode:   Z80_ADD_A_R,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_ADD_A_N = InstrDescriptor{
	Opcode:   Z80_ADD_A_N,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_ADD_A_HL = InstrDescriptor{
	Opcode:   Z80_ADD_A_HL,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_ADC_A_R = InstrDescriptor{
	Opcode:   Z80_ADC_A_R,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // ADC reads Carry flag
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_ADC_A_N = InstrDescriptor{
	Opcode:   Z80_ADC_A_N,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // ADC reads Carry flag
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_ADC_A_HL = InstrDescriptor{
	Opcode:   Z80_ADC_A_HL,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // ADC reads Carry flag
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_SUB_R = InstrDescriptor{
	Opcode:   Z80_SUB_R,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_SUB_N = InstrDescriptor{
	Opcode:   Z80_SUB_N,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_SUB_HL = InstrDescriptor{
	Opcode:   Z80_SUB_HL,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_SBC_A_R = InstrDescriptor{
	Opcode:   Z80_SBC_A_R,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // SBC reads Carry flag
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_SBC_A_N = InstrDescriptor{
	Opcode:   Z80_SBC_A_N,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // SBC reads Carry flag
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_SBC_A_HL = InstrDescriptor{
	Opcode:   Z80_SBC_A_HL,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // SBC reads Carry flag
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_INC_R = InstrDescriptor{
	Opcode:   Z80_INC_R,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_DEC_R = InstrDescriptor{
	Opcode:   Z80_DEC_R,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_INC_HL = InstrDescriptor{
	Opcode:   Z80_INC_HL,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessReadWrite, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN,
	DependentFlags: InstrFlagNone,
	Cycles:         11,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_DEC_HL = InstrDescriptor{
	Opcode:   Z80_DEC_HL,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessReadWrite, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN,
	DependentFlags: InstrFlagNone,
	Cycles:         11,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// 16-bit Arithmetic Instructions
// ============================================================================

var InstrDesc_ADD_HL_RR = InstrDescriptor{
	Opcode:   Z80_ADD_HL_RR,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessReadWrite, Registers: []*Register{&RegHL}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagH | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         11,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 4,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_ADC_HL_RR = InstrDescriptor{
	Opcode:   Z80_ADC_HL_RR,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessReadWrite, Registers: []*Register{&RegHL}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // ADC reads Carry flag
	Cycles:         15,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 4,
	Prefix1:        0xED,
	Prefix2:        0,
}

var InstrDesc_SBC_HL_RR = InstrDescriptor{
	Opcode:   Z80_SBC_HL_RR,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessReadWrite, Registers: []*Register{&RegHL}},
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC, // SBC reads Carry flag
	Cycles:         15,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 4,
	Prefix1:        0xED,
	Prefix2:        0,
}

var InstrDesc_INC_RR = InstrDescriptor{
	Opcode:   Z80_INC_RR,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessReadWrite, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         6,
	CyclesTaken:    0,
	EncodingReg1SL: 4,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_DEC_RR = InstrDescriptor{
	Opcode:   Z80_DEC_RR,
	Category: CatArithmetic,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessReadWrite, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegSP}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         6,
	CyclesTaken:    0,
	EncodingReg1SL: 4,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// Logical Instructions
// ============================================================================

var InstrDesc_AND_R = InstrDescriptor{
	Opcode:   Z80_AND_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_AND_N = InstrDescriptor{
	Opcode:   Z80_AND_N,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_AND_HL = InstrDescriptor{
	Opcode:   Z80_AND_HL,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_OR_R = InstrDescriptor{
	Opcode:   Z80_OR_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_OR_N = InstrDescriptor{
	Opcode:   Z80_OR_N,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_OR_HL = InstrDescriptor{
	Opcode:   Z80_OR_HL,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_XOR_R = InstrDescriptor{
	Opcode:   Z80_XOR_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_XOR_N = InstrDescriptor{
	Opcode:   Z80_XOR_N,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_XOR_HL = InstrDescriptor{
	Opcode:   Z80_XOR_HL,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_CP_R = InstrDescriptor{
	Opcode:   Z80_CP_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_CP_N = InstrDescriptor{
	Opcode:   Z80_CP_N,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_CP_HL = InstrDescriptor{
	Opcode:   Z80_CP_HL,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         7,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// Bitwise Instructions (CB prefix)
// ============================================================================

var InstrDesc_BIT_B_R = InstrDescriptor{
	Opcode:   Z80_BIT_B_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpBitIndex, Access: AccessRead},
		{Type: OpRegister, Access: AccessRead, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_SET_B_R = InstrDescriptor{
	Opcode:   Z80_SET_B_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpBitIndex, Access: AccessRead},
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_RES_B_R = InstrDescriptor{
	Opcode:   Z80_RES_B_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpBitIndex, Access: AccessRead},
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

// ============================================================================
// Rotate/Shift Instructions (CB prefix)
// ============================================================================

var InstrDesc_RLC_R = InstrDescriptor{
	Opcode:   Z80_RLC_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_RRC_R = InstrDescriptor{
	Opcode:   Z80_RRC_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_RL_R = InstrDescriptor{
	Opcode:   Z80_RL_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_RR_R = InstrDescriptor{
	Opcode:   Z80_RR_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagC,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_SLA_R = InstrDescriptor{
	Opcode:   Z80_SLA_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_SRA_R = InstrDescriptor{
	Opcode:   Z80_SRA_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

var InstrDesc_SRL_R = InstrDescriptor{
	Opcode:   Z80_SRL_R,
	Category: CatBitwise,
	Dependencies: []InstrDependency{
		{Type: OpRegister, Access: AccessReadWrite, Registers: []*Register{&RegA, &RegB, &RegC, &RegD, &RegE, &RegH, &RegL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagS | InstrFlagZ | InstrFlagH | InstrFlagPV | InstrFlagN | InstrFlagC,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xCB,
	Prefix2:        0,
}

// ============================================================================
// Stack Instructions
// ============================================================================

var InstrDesc_PUSH_QQ = InstrDescriptor{
	Opcode:   Z80_PUSH_QQ,
	Category: CatStack,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairQQ, Access: AccessRead, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegAF}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         11,
	CyclesTaken:    0,
	EncodingReg1SL: 4,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_POP_QQ = InstrDescriptor{
	Opcode:   Z80_POP_QQ,
	Category: CatStack,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairQQ, Access: AccessWrite, Registers: []*Register{&RegBC, &RegDE, &RegHL, &RegAF}},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         10,
	CyclesTaken:    0,
	EncodingReg1SL: 4,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// Jump/Branch Instructions
// ============================================================================

var InstrDesc_JP_NN = InstrDescriptor{
	Opcode:   Z80_JP_NN,
	Category: CatBranch,
	Dependencies: []InstrDependency{
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         10,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_JP_HL = InstrDescriptor{
	Opcode:   Z80_JP_HL,
	Category: CatBranch,
	Dependencies: []InstrDependency{
		{Type: OpRegisterPairRR, Access: AccessRead, Registers: []*Register{&RegHL}},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_JP_CC_NN = InstrDescriptor{
	Opcode:   Z80_JP_CC_NN,
	Category: CatBranch,
	Dependencies: []InstrDependency{
		{Type: OpConditionCode, Access: AccessRead},
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagDynamic, // Depends on condition code operand
	Cycles:         10,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_JR_E = InstrDescriptor{
	Opcode:   Z80_JR_E,
	Category: CatBranch,
	Dependencies: []InstrDependency{
		{Type: OpRelExpression, Access: AccessRead},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         12,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_JR_CC_E = InstrDescriptor{
	Opcode:   Z80_JR_CC_E,
	Category: CatBranch,
	Dependencies: []InstrDependency{
		{Type: OpConditionCode, Access: AccessRead},
		{Type: OpRelExpression, Access: AccessRead},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagDynamic, // Depends on condition code operand
	Cycles:         7,
	CyclesTaken:    5,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_DJNZ_E = InstrDescriptor{
	Opcode:   Z80_DJNZ_E,
	Category: CatBranch,
	Dependencies: []InstrDependency{
		{Type: OpRelExpression, Access: AccessRead},
	},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         8,
	CyclesTaken:    5,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// Call/Return Instructions
// ============================================================================

var InstrDesc_CALL_NN = InstrDescriptor{
	Opcode:   Z80_CALL_NN,
	Category: CatSubroutine,
	Dependencies: []InstrDependency{
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         17,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_CALL_CC_NN = InstrDescriptor{
	Opcode:   Z80_CALL_CC_NN,
	Category: CatSubroutine,
	Dependencies: []InstrDependency{
		{Type: OpConditionCode, Access: AccessRead},
		{Type: OpConstant16, Access: AccessRead},
	},
	Properties:     InstrImmediate | InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagDynamic, // Depends on condition code operand
	Cycles:         10,
	CyclesTaken:    7,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_RET = InstrDescriptor{
	Opcode:         Z80_RET,
	Category:       CatSubroutine,
	Dependencies:   []InstrDependency{},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         10,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_RET_CC = InstrDescriptor{
	Opcode:   Z80_RET_CC,
	Category: CatSubroutine,
	Dependencies: []InstrDependency{
		{Type: OpConditionCode, Access: AccessRead},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagDynamic, // Depends on condition code operand
	Cycles:         5,
	CyclesTaken:    6,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_RETN = InstrDescriptor{
	Opcode:         Z80_RETN,
	Category:       CatSubroutine,
	Dependencies:   []InstrDependency{},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         14,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xED,
	Prefix2:        0,
}

var InstrDesc_RST_P = InstrDescriptor{
	Opcode:   Z80_RST_P,
	Category: CatSubroutine,
	Dependencies: []InstrDependency{
		{Type: OpConstant8, Access: AccessRead},
	},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         11,
	CyclesTaken:    0,
	EncodingReg1SL: 3,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// Interrupt Instructions
// ============================================================================

var InstrDesc_RETI = InstrDescriptor{
	Opcode:         Z80_RETI,
	Category:       CatInterrupt,
	Dependencies:   []InstrDependency{},
	Properties:     InstrIndirect,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         14,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0xED,
	Prefix2:        0,
}

var InstrDesc_DI = InstrDescriptor{
	Opcode:         Z80_DI,
	Category:       CatInterrupt,
	Dependencies:   []InstrDependency{},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_EI = InstrDescriptor{
	Opcode:         Z80_EI,
	Category:       CatInterrupt,
	Dependencies:   []InstrDependency{},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// Special Instructions
// ============================================================================

var InstrDesc_NOP = InstrDescriptor{
	Opcode:         Z80_NOP,
	Category:       CatOther,
	Dependencies:   []InstrDependency{},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

var InstrDesc_HALT = InstrDescriptor{
	Opcode:         Z80_HALT,
	Category:       CatOther,
	Dependencies:   []InstrDependency{},
	Properties:     0,
	AffectedFlags:  InstrFlagNone,
	DependentFlags: InstrFlagNone,
	Cycles:         4,
	CyclesTaken:    0,
	EncodingReg1SL: 0,
	EncodingReg2SL: 0,
	Prefix1:        0,
	Prefix2:        0,
}

// ============================================================================
// Instruction Descriptor Lookup Table
// ============================================================================

// Z80InstrDescriptors maps opcodes to their instruction descriptors
var Z80InstrDescriptors = map[Z80Opcode]*InstrDescriptor{
	// 8-bit Load
	Z80_LD_R_R:  &InstrDesc_LD_R_R,
	Z80_LD_R_N:  &InstrDesc_LD_R_N,
	Z80_LD_R_HL: &InstrDesc_LD_R_HL,
	Z80_LD_HL_R: &InstrDesc_LD_HL_R,
	Z80_LD_HL_N: &InstrDesc_LD_HL_N,
	Z80_LD_A_PP: &InstrDesc_LD_A_PP,
	Z80_LD_A_NN: &InstrDesc_LD_A_NN,
	Z80_LD_PP_A: &InstrDesc_LD_PP_A,
	Z80_LD_NN_A: &InstrDesc_LD_NN_A,

	// 16-bit Load
	Z80_LD_RR_NN:      &InstrDesc_LD_RR_NN,
	Z80_LD_HL_NN:      &InstrDesc_LD_HL_NN,
	Z80_LD_RR_NN_ADDR: &InstrDesc_LD_RR_NN_ADDR,
	Z80_LD_NN_HL:      &InstrDesc_LD_NN_HL,
	Z80_LD_NN_RR_ADDR: &InstrDesc_LD_NN_RR_ADDR,
	Z80_LD_SP_HL:      &InstrDesc_LD_SP_HL,

	// 8-bit Arithmetic
	Z80_ADD_A_R:  &InstrDesc_ADD_A_R,
	Z80_ADD_A_N:  &InstrDesc_ADD_A_N,
	Z80_ADD_A_HL: &InstrDesc_ADD_A_HL,
	Z80_ADC_A_R:  &InstrDesc_ADC_A_R,
	Z80_ADC_A_N:  &InstrDesc_ADC_A_N,
	Z80_ADC_A_HL: &InstrDesc_ADC_A_HL,
	Z80_SUB_R:    &InstrDesc_SUB_R,
	Z80_SUB_N:    &InstrDesc_SUB_N,
	Z80_SUB_HL:   &InstrDesc_SUB_HL,
	Z80_SBC_A_R:  &InstrDesc_SBC_A_R,
	Z80_SBC_A_N:  &InstrDesc_SBC_A_N,
	Z80_SBC_A_HL: &InstrDesc_SBC_A_HL,
	Z80_INC_R:    &InstrDesc_INC_R,
	Z80_DEC_R:    &InstrDesc_DEC_R,
	Z80_INC_HL:   &InstrDesc_INC_HL,
	Z80_DEC_HL:   &InstrDesc_DEC_HL,

	// 16-bit Arithmetic
	Z80_ADD_HL_RR: &InstrDesc_ADD_HL_RR,
	Z80_ADC_HL_RR: &InstrDesc_ADC_HL_RR,
	Z80_SBC_HL_RR: &InstrDesc_SBC_HL_RR,
	Z80_INC_RR:    &InstrDesc_INC_RR,
	Z80_DEC_RR:    &InstrDesc_DEC_RR,

	// Logical
	Z80_AND_R:  &InstrDesc_AND_R,
	Z80_AND_N:  &InstrDesc_AND_N,
	Z80_AND_HL: &InstrDesc_AND_HL,
	Z80_OR_R:   &InstrDesc_OR_R,
	Z80_OR_N:   &InstrDesc_OR_N,
	Z80_OR_HL:  &InstrDesc_OR_HL,
	Z80_XOR_R:  &InstrDesc_XOR_R,
	Z80_XOR_N:  &InstrDesc_XOR_N,
	Z80_XOR_HL: &InstrDesc_XOR_HL,
	Z80_CP_R:   &InstrDesc_CP_R,
	Z80_CP_N:   &InstrDesc_CP_N,
	Z80_CP_HL:  &InstrDesc_CP_HL,

	// Bitwise (CB prefix)
	Z80_BIT_B_R: &InstrDesc_BIT_B_R,
	Z80_SET_B_R: &InstrDesc_SET_B_R,
	Z80_RES_B_R: &InstrDesc_RES_B_R,

	// Rotate/Shift (CB prefix)
	Z80_RLC_R: &InstrDesc_RLC_R,
	Z80_RRC_R: &InstrDesc_RRC_R,
	Z80_RL_R:  &InstrDesc_RL_R,
	Z80_RR_R:  &InstrDesc_RR_R,
	Z80_SLA_R: &InstrDesc_SLA_R,
	Z80_SRA_R: &InstrDesc_SRA_R,
	Z80_SRL_R: &InstrDesc_SRL_R,

	// Stack
	Z80_PUSH_QQ: &InstrDesc_PUSH_QQ,
	Z80_POP_QQ:  &InstrDesc_POP_QQ,

	// Jump/Branch
	Z80_JP_NN:    &InstrDesc_JP_NN,
	Z80_JP_HL:    &InstrDesc_JP_HL,
	Z80_JP_CC_NN: &InstrDesc_JP_CC_NN,
	Z80_JR_E:     &InstrDesc_JR_E,
	Z80_JR_CC_E:  &InstrDesc_JR_CC_E,
	Z80_DJNZ_E:   &InstrDesc_DJNZ_E,

	// Call/Return
	Z80_CALL_NN:    &InstrDesc_CALL_NN,
	Z80_CALL_CC_NN: &InstrDesc_CALL_CC_NN,
	Z80_RET:        &InstrDesc_RET,
	Z80_RET_CC:     &InstrDesc_RET_CC,
	Z80_RETN:       &InstrDesc_RETN,
	Z80_RST_P:      &InstrDesc_RST_P,

	// Interrupts
	Z80_RETI: &InstrDesc_RETI,
	Z80_DI:   &InstrDesc_DI,
	Z80_EI:   &InstrDesc_EI,

	// Special
	Z80_NOP:  &InstrDesc_NOP,
	Z80_HALT: &InstrDesc_HALT,
}
