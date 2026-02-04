package cfg

import "fmt"

// Z80 Instruction Descriptor Database
// Defines properties of all Z80 instructions for instruction selection and scheduling

// ============================================================================
// Z80 Opcodes
// ============================================================================

// Z80Opcode represents actual Z80 machine instructions
// For prefixed instructions, the prefix is encoded in the high byte:
//   - 0x00xx: No prefix (standard instructions)
//   - 0xCBxx: CB prefix (bit operations)
//   - 0xEDxx: ED prefix (extended instructions)
//   - 0xDDxx: DD prefix (IX operations)
//   - 0xFDxx: FD prefix (IY operations)
type Z80Opcode uint16

const (
	// 8-bit Load
	Z80_LD_R_R  Z80Opcode = 0x0040 // LD r, r'  (register to register)
	Z80_LD_R_N  Z80Opcode = 0x0006 // LD r, n   (immediate to register)
	Z80_LD_R_HL Z80Opcode = 0x0046 // LD r, (HL) (memory at HL to register)
	Z80_LD_HL_R Z80Opcode = 0x0070 // LD (HL), r (register to memory at HL)
	Z80_LD_HL_N Z80Opcode = 0x0036 // LD (HL), n (immediate to memory at HL)

	// 8-bit access to memory via register pairs
	Z80_LD_A_PP Z80Opcode = 0x000A // LD A, (BC|DE)
	Z80_LD_A_NN Z80Opcode = 0x003A // LD A, (nn)
	Z80_LD_PP_A Z80Opcode = 0x0002 // LD (BC|DE), A
	Z80_LD_NN_A Z80Opcode = 0x0032 // LD (nn), A

	// 16-bit Load
	Z80_LD_RR_NN      Z80Opcode = 0x0001 // LD rr, nn  (immediate to register pair)
	Z80_LD_HL_NN      Z80Opcode = 0x002A // LD HL, (nn) (memory to register pair)
	Z80_LD_RR_NN_ADDR Z80Opcode = 0xED4B // LD rr, (nn) (memory to register pair) - ED prefix
	Z80_LD_NN_HL      Z80Opcode = 0x0022 // LD (nn), HL (register pair to memory)
	Z80_LD_NN_RR_ADDR Z80Opcode = 0xED43 // LD (nn), RR (register pair to memory) - ED prefix
	Z80_LD_SP_HL      Z80Opcode = 0x00F9 // LD SP, HL

	// 8-bit Arithmetic
	Z80_ADD_A_R  Z80Opcode = 0x0080 // ADD A, r
	Z80_ADD_A_N  Z80Opcode = 0x00C6 // ADD A, n
	Z80_ADD_A_HL Z80Opcode = 0x0086 // ADD A, (HL)
	Z80_ADC_A_R  Z80Opcode = 0x0088 // ADC A, r (add with carry)
	Z80_ADC_A_N  Z80Opcode = 0x00CE // ADC A, n
	Z80_ADC_A_HL Z80Opcode = 0x008E // ADC A, (HL)
	Z80_SUB_R    Z80Opcode = 0x0090 // SUB r
	Z80_SUB_N    Z80Opcode = 0x00D6 // SUB n
	Z80_SUB_HL   Z80Opcode = 0x0096 // SUB (HL)
	Z80_SBC_A_R  Z80Opcode = 0x0098 // SBC A, r (subtract with carry)
	Z80_SBC_A_N  Z80Opcode = 0x00DE // SBC A, n
	Z80_SBC_A_HL Z80Opcode = 0x009E // SBC A, (HL)
	Z80_INC_R    Z80Opcode = 0x0004 // INC r
	Z80_DEC_R    Z80Opcode = 0x0005 // DEC r
	Z80_INC_HL   Z80Opcode = 0x0034 // INC (HL)
	Z80_DEC_HL   Z80Opcode = 0x0035 // DEC (HL)

	// 16-bit Arithmetic
	Z80_ADD_HL_RR Z80Opcode = 0x0009 // ADD HL, rr
	Z80_ADC_HL_RR Z80Opcode = 0xED4A // ADC HL, rr - ED prefix
	Z80_SBC_HL_RR Z80Opcode = 0xED42 // SBC HL, rr - ED prefix
	Z80_INC_RR    Z80Opcode = 0x0003 // INC rr
	Z80_DEC_RR    Z80Opcode = 0x000B // DEC rr

	// Logical
	Z80_AND_R  Z80Opcode = 0x00A0 // AND r
	Z80_AND_N  Z80Opcode = 0x00E6 // AND n
	Z80_AND_HL Z80Opcode = 0x00A6 // AND (HL)
	Z80_OR_R   Z80Opcode = 0x00B0 // OR r
	Z80_OR_N   Z80Opcode = 0x00F6 // OR n
	Z80_OR_HL  Z80Opcode = 0x00B6 // OR (HL)
	Z80_XOR_R  Z80Opcode = 0x00A8 // XOR r
	Z80_XOR_N  Z80Opcode = 0x00EE // XOR n
	Z80_XOR_HL Z80Opcode = 0x00AE // XOR (HL)
	Z80_CP_R   Z80Opcode = 0x00B8 // CP r (compare)
	Z80_CP_N   Z80Opcode = 0x00FE // CP n
	Z80_CP_HL  Z80Opcode = 0x00BE // CP (HL)

	// Bitwise (CB prefix instructions)
	Z80_BIT_B_R Z80Opcode = 0xCB40 // BIT b, r (test bit) - CB prefix
	Z80_SET_B_R Z80Opcode = 0xCBC0 // SET b, r (set bit) - CB prefix
	Z80_RES_B_R Z80Opcode = 0xCB80 // RES b, r (reset bit) - CB prefix

	// Rotate/Shift (CB prefix instructions)
	Z80_RLC_R Z80Opcode = 0xCB00 // RLC r (rotate left circular) - CB prefix
	Z80_RRC_R Z80Opcode = 0xCB08 // RRC r (rotate right circular) - CB prefix
	Z80_RL_R  Z80Opcode = 0xCB10 // RL r (rotate left through carry) - CB prefix
	Z80_RR_R  Z80Opcode = 0xCB18 // RR r (rotate right through carry) - CB prefix
	Z80_SLA_R Z80Opcode = 0xCB20 // SLA r (shift left arithmetic) - CB prefix
	Z80_SRA_R Z80Opcode = 0xCB28 // SRA r (shift right arithmetic) - CB prefix
	Z80_SRL_R Z80Opcode = 0xCB38 // SRL r (shift right logical) - CB prefix

	// Stack
	Z80_PUSH_QQ Z80Opcode = 0x00C5 // PUSH qq
	Z80_POP_QQ  Z80Opcode = 0x00C1 // POP qq

	// Jump/Branch
	Z80_JP_NN    Z80Opcode = 0x00C3 // JP nn (unconditional jump)
	Z80_JP_HL    Z80Opcode = 0x00E9 // JP (HL) (jump to address in HL)
	Z80_JP_CC_NN Z80Opcode = 0x00C2 // JP cc, nn (conditional jump)
	Z80_JR_E     Z80Opcode = 0x0018 // JR e (relative jump)
	Z80_JR_CC_E  Z80Opcode = 0x0020 // JR cc, e (conditional relative jump)
	Z80_DJNZ_E   Z80Opcode = 0x0010 // DJNZ e (decrement B and jump if not zero)

	// Call/Return
	Z80_CALL_NN    Z80Opcode = 0x00CD // CALL nn
	Z80_CALL_CC_NN Z80Opcode = 0x00C4 // CALL cc, nn (conditional call)
	Z80_RET        Z80Opcode = 0x00C9 // RET
	Z80_RET_CC     Z80Opcode = 0x00C0 // RET cc
	Z80_RST_P      Z80Opcode = 0x00C7 // RST p (restart to address p*8)

	// interrupts
	Z80_RETI Z80Opcode = 0xED4D // RETI (return from interrupt) - ED prefix
	Z80_RETN Z80Opcode = 0xED45 // RETN (return from NMI) - ED prefix
	Z80_DI   Z80Opcode = 0x00F3 // DI (disable interrupts)
	Z80_EI   Z80Opcode = 0x00FB // EI (enable interrupts)
	// IM 0, IM 1, IM 2 (set interrupt mode) - ED prefix

	// Special
	Z80_NOP  Z80Opcode = 0x0000 // NOP
	Z80_HALT Z80Opcode = 0x0076 // HALT
	Z80_NEG  Z80Opcode = 0xED44 // NEG (two's complement negate A) - ED prefix
	Z80_CCF  Z80Opcode = 0x003F // CCF (complement carry flag)

	// others...
	// EX AF, AF' (exchange AF and AF')
	// EX DE, HL (exchange DE and HL)
	// EX (SP), HL (exchange HL with value at SP)
	// LDD, LDI, LDDR, LDIR (block transfer instructions) - ED prefix
	// CPI, CPD, CPIR, CPDR (block compare instructions) - ED prefix
	// INI, IND, INIR, INDR (block input instructions) - ED prefix
	// OUTI, OUTD, OTIR, OTDR (block output instructions) - ED prefix
)

// AccessType specifies how a dependency is accessed
type AccessType uint8

const (
	AccessNone      AccessType = 0
	AccessRead      AccessType = 1 << 0
	AccessWrite     AccessType = 1 << 1
	AccessReadWrite AccessType = AccessRead | AccessWrite
)

// OperandType specifies the kind of operand
type OperandType int

const (
	OpNone           OperandType = iota // No operand (used for implicit dependencies)
	OpRegister                          // r: Physical register (A, B, C, D, E, H, L)
	OpRegisterPairRR                    // rr: Register pair (BC, DE, HL, SP)
	OpRegisterPairQQ                    // qq: Register pair (BC, DE, HL, AF)
	OpRegisterPairPP                    // pp: Register pair (BC, DE)
	OpConstant8                         // n: Constant value (8 bit)
	OpConstant16                        // nn: Constant value (16 bit)
	OpRelExpression                     // e: Jump/branch target (-126 to +129)
	OpDisplacement                      // d: IX/IY displacement (-128 to +127)
	OpConditionCode                     // cc: Condition code (Z, NZ, C, NC)
	OpBitIndex                          // b: Bit index (0-7)
	OpRestartVector                     // p: Restart vector (0H, 8H, 10H, 18H, 20H, 28H, 30H, 38H)
)

type ConditionCode uint8

const (
	Cond_None ConditionCode = iota
	Cond_NZ                 // Not Zero
	Cond_Z                  // Zero
	Cond_NC                 // Not Carry
	Cond_C                  // Carry
	Cond_PO                 // Parity Odd
	Cond_PE                 // Parity Even
	Cond_P                  // Positive
	Cond_M                  // Minus
)

// GetFlagsForCondition returns which flags a condition code depends on
func GetFlagsForCondition(cc ConditionCode) InstrFlags {
	switch cc {
	case Cond_NZ, Cond_Z:
		return InstrFlagZ
	case Cond_NC, Cond_C:
		return InstrFlagC
	case Cond_PO, Cond_PE:
		return InstrFlagPV
	case Cond_P, Cond_M:
		return InstrFlagS
	default:
		return 0
	}
}

type InstrFlags uint16

const (
	InstrFlagNone InstrFlags = 0
	// Flag effects (specific Z80 flags, 8-bits)
	InstrFlagC  InstrFlags = 1 << 0 // Modifies Carry flag
	InstrFlagN  InstrFlags = 1 << 1 // Modifies Add/Subtract flag
	InstrFlagPV InstrFlags = 1 << 2 // Modifies Parity/Overflow flag
	InstrFlagH  InstrFlags = 1 << 4 // Modifies Half-carry flag
	InstrFlagZ  InstrFlags = 1 << 6 // Modifies Zero flag
	InstrFlagS  InstrFlags = 1 << 7 // Modifies Sign flag

	// Special flag dependency indicator (> 8-bits)
	InstrFlagDynamic InstrFlags = 1 << 8 // Flag dependency determined by condition code operand at runtime
)

type InstrDependency struct {
	Type      OperandType
	Access    AccessType
	Registers []*Register // allowed/affected registers (if applicable)
}

// ============================================================================
// Instruction Descriptor
// ============================================================================

// InstrDescriptor describes properties of a Z80 instruction
type InstrDescriptor struct {
	Opcode         Z80Opcode
	Category       InstrCategory
	Dependencies   []InstrDependency // Operands and implicit register/flag dependencies
	AddressingMode AddressingMode
	AffectedFlags  InstrFlags // Flags this instruction modifies
	DependentFlags InstrFlags // Flags this instruction reads/depends on

	// Timing (in T-states/cycles) (includes prefixes)
	Cycles      uint8 // Mandatory cycle count (for non-branching or branch-not-taken)
	CyclesTaken uint8 // Additional cycles if branch is taken (0 for non-branch instructions)
	Size        uint8 // Instruction size in bytes (including prefixes)

	// TODO: these might be constant over the complete range of instructions?
	EncodingReg1SL uint8 // Shift left of operand register-id #1 in opcode encoding
	EncodingReg2SL uint8 // Shift left of operand register-id #2 in opcode encoding
	Prefix1        uint8 // Instruction prefix #1 byte (0 if none)
	Prefix2        uint8 // Instruction prefix #2 byte (0 if none)
}

func HasDependency(deps []InstrDependency, operandType OperandType) bool {
	for _, dep := range deps {
		if dep.Type == operandType {
			return true
		}
	}
	return false
}

// String returns a human-readable name for the Z80 opcode
func (op Z80Opcode) String() string {
	switch op {
	// 8-bit Load
	case Z80_LD_R_R:
		return "LD"
	case Z80_LD_R_N:
		return "LD"
	case Z80_LD_R_HL:
		return "LD"
	case Z80_LD_HL_R:
		return "LD"
	case Z80_LD_HL_N:
		return "LD"
	case Z80_LD_A_PP:
		return "LD"
	case Z80_LD_A_NN:
		return "LD"
	case Z80_LD_PP_A:
		return "LD"
	case Z80_LD_NN_A:
		return "LD"

	// 16-bit Load
	case Z80_LD_RR_NN:
		return "LD"
	case Z80_LD_HL_NN:
		return "LD"
	case Z80_LD_RR_NN_ADDR:
		return "LD"
	case Z80_LD_NN_HL:
		return "LD"
	case Z80_LD_NN_RR_ADDR:
		return "LD"
	case Z80_LD_SP_HL:
		return "LD"

	// 8-bit Arithmetic
	case Z80_ADD_A_R:
		return "ADD"
	case Z80_ADD_A_N:
		return "ADD"
	case Z80_ADD_A_HL:
		return "ADD"
	case Z80_ADC_A_R:
		return "ADC"
	case Z80_ADC_A_N:
		return "ADC"
	case Z80_ADC_A_HL:
		return "ADC"
	case Z80_SUB_R:
		return "SUB"
	case Z80_SUB_N:
		return "SUB"
	case Z80_SUB_HL:
		return "SUB"
	case Z80_SBC_A_R:
		return "SBC"
	case Z80_SBC_A_N:
		return "SBC"
	case Z80_SBC_A_HL:
		return "SBC"
	case Z80_AND_R:
		return "AND"
	case Z80_AND_N:
		return "AND"
	case Z80_AND_HL:
		return "AND"
	case Z80_OR_R:
		return "OR"
	case Z80_OR_N:
		return "OR"
	case Z80_OR_HL:
		return "OR"
	case Z80_XOR_R:
		return "XOR"
	case Z80_XOR_N:
		return "XOR"
	case Z80_XOR_HL:
		return "XOR"
	case Z80_CP_R:
		return "CP"
	case Z80_CP_N:
		return "CP"
	case Z80_CP_HL:
		return "CP"
	case Z80_INC_R:
		return "INC"
	case Z80_INC_HL:
		return "INC"
	case Z80_DEC_R:
		return "DEC"
	case Z80_DEC_HL:
		return "DEC"

	// 16-bit Arithmetic
	case Z80_ADD_HL_RR:
		return "ADD"
	case Z80_ADC_HL_RR:
		return "ADC"
	case Z80_SBC_HL_RR:
		return "SBC"
	case Z80_INC_RR:
		return "INC"
	case Z80_DEC_RR:
		return "DEC"

	// Control Flow
	case Z80_JP_NN:
		return "JP"
	case Z80_JP_CC_NN:
		return "JP"
	case Z80_JP_HL:
		return "JP"
	case Z80_JR_E:
		return "JR"
	case Z80_JR_CC_E:
		return "JR"
	case Z80_CALL_NN:
		return "CALL"
	case Z80_CALL_CC_NN:
		return "CALL"
	case Z80_RET:
		return "RET"
	case Z80_RET_CC:
		return "RET"
	// case Z80_RST:
	// 	return "RST"

	// Stack
	case Z80_PUSH_QQ:
		return "PUSH"
	case Z80_POP_QQ:
		return "POP"

	// Bit Operations
	case Z80_BIT_B_R:
		return "BIT"
	// case Z80_BIT_B_HL:
	// 	return "BIT"
	case Z80_SET_B_R:
		return "SET"
	// case Z80_SET_B_HL:
	// 	return "SET"
	case Z80_RES_B_R:
		return "RES"
	// case Z80_RES_B_HL:
	// 	return "RES"

	// Rotate/Shift
	// case Z80_RLCA:
	// 	return "RLCA"
	// case Z80_RLA:
	// 	return "RLA"
	// case Z80_RRCA:
	// 	return "RRCA"
	// case Z80_RRA:
	// 	return "RRA"
	case Z80_RLC_R:
		return "RLC"
	// case Z80_RLC_HL:
	// 	return "RLC"
	case Z80_RL_R:
		return "RL"
	// case Z80_RL_HL:
	// 	return "RL"
	case Z80_RRC_R:
		return "RRC"
	// case Z80_RRC_HL:
	// 	return "RRC"
	case Z80_RR_R:
		return "RR"
	// case Z80_RR_HL:
	// 	return "RR"
	case Z80_SLA_R:
		return "SLA"
	// case Z80_SLA_HL:
	// 	return "SLA"
	case Z80_SRA_R:
		return "SRA"
	// case Z80_SRA_HL:
	// 	return "SRA"
	case Z80_SRL_R:
		return "SRL"
	// case Z80_SRL_HL:
	// 	return "SRL"

	// Misc
	case Z80_NOP:
		return "NOP"
	case Z80_HALT:
		return "HALT"
	case Z80_DI:
		return "DI"
	case Z80_EI:
		return "EI"
	// case Z80_EX_DE_HL:
	// 	return "EX"
	// case Z80_EX_AF_AF:
	// 	return "EX"
	// case Z80_EXX:
	// 	return "EXX"
	// case Z80_EX_SP_HL:
	// 	return "EX"

	default:
		return fmt.Sprintf("UNKNOWN_OP_%04X", uint16(op))
	}
}

// String returns a human-readable name for the condition code
func (cc ConditionCode) String() string {
	switch cc {
	case 0:
		return ""
	case Cond_NZ:
		return "NZ"
	case Cond_Z:
		return "Z"
	case Cond_NC:
		return "NC"
	case Cond_C:
		return "C"
	case Cond_PO:
		return "PO"
	case Cond_PE:
		return "PE"
	case Cond_P:
		return "P"
	case Cond_M:
		return "M"
	default:
		return fmt.Sprintf("UNKNOWN_COND_%d", uint8(cc))
	}
}
