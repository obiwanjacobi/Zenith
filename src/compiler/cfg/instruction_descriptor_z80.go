package cfg

// Z80 Instruction Descriptor Database
// Defines properties of all Z80 instructions for instruction selection and scheduling

// Notes about Z80 instruction Encoding:
// See http://z80.info/decoding.htm

// | 7 | 6 | 5 | 4 | 3 | 2 | 1 | 0 |
// +---+---+---+---+---+---+---+---+
// |   x   |     y     |     z     |
//         +---+---+---+
//         |   p   | q |

// ============================================================================
// Z80 Opcodes
// ============================================================================

// Z80Opcode represents actual Z80 machine instructions
type Z80Opcode uint8

const (
	// 8-bit Load
	Z80_LD_R_R  Z80Opcode = 0x40 // LD r, r'  (register to register)
	Z80_LD_R_N  Z80Opcode = 0x06 // LD r, n   (immediate to register)
	Z80_LD_R_HL Z80Opcode = 0x46 // LD r, (HL) (memory at HL to register)
	Z80_LD_HL_R Z80Opcode = 0x70 // LD (HL), r (register to memory at HL)
	Z80_LD_HL_N Z80Opcode = 0x36 // LD (HL), n (immediate to memory at HL)

	// 8-bit access to memory via register pairs
	Z80_LD_A_PP Z80Opcode = 0x0A // LD A, (BC|DE)
	Z80_LD_A_NN Z80Opcode = 0x3A // LD A, (nn)
	Z80_LD_PP_A Z80Opcode = 0x02 // LD (BC|DE), A
	Z80_LD_NN_A Z80Opcode = 0x32 // LD (nn), A

	// 16-bit Load
	Z80_LD_RR_NN      Z80Opcode = 0x01 // LD rr, nn  (immediate to register pair)
	Z80_LD_HL_NN      Z80Opcode = 0x2A // LD HL, (nn) (memory to register pair)
	Z80_LD_RR_NN_ADDR Z80Opcode = 0x4B // LD rr, (nn) (memory to register pair)
	Z80_LD_NN_HL      Z80Opcode = 0x22 // LD (nn), HL (register pair to memory)
	Z80_LD_NN_RR_ADDR Z80Opcode = 0x43 // LD (nn), RR (register pair to memory)
	Z80_LD_SP_HL      Z80Opcode = 0xF9 // LD SP, HL

	// 8-bit Arithmetic
	Z80_ADD_A_R  Z80Opcode = 0x80 // ADD A, r
	Z80_ADD_A_N  Z80Opcode = 0xC6 // ADD A, n
	Z80_ADD_A_HL Z80Opcode = 0x86 // ADD A, (HL)
	Z80_ADC_A_R  Z80Opcode = 0x88 // ADC A, r (add with carry)
	Z80_ADC_A_N  Z80Opcode = 0xCE // ADC A, n
	Z80_ADC_A_HL Z80Opcode = 0x8E // ADC A, (HL)
	Z80_SUB_R    Z80Opcode = 0x90 // SUB r
	Z80_SUB_N    Z80Opcode = 0xD6 // SUB n
	Z80_SUB_HL   Z80Opcode = 0x96 // SUB (HL)
	Z80_SBC_A_R  Z80Opcode = 0x98 // SBC A, r (subtract with carry)
	Z80_SBC_A_N  Z80Opcode = 0xDE // SBC A, n
	Z80_SBC_A_HL Z80Opcode = 0x9E // SBC A, (HL)
	Z80_INC_R    Z80Opcode = 0x04 // INC r
	Z80_DEC_R    Z80Opcode = 0x05 // DEC r
	Z80_INC_HL   Z80Opcode = 0x34 // INC (HL)
	Z80_DEC_HL   Z80Opcode = 0x35 // DEC (HL)

	// 16-bit Arithmetic
	Z80_ADD_HL_RR Z80Opcode = 0x09 // ADD HL, rr
	Z80_ADC_HL_RR Z80Opcode = 0x4A // ADC HL, rr (ED prefix)
	Z80_SBC_HL_RR Z80Opcode = 0x42 // SBC HL, rr (ED prefix)
	Z80_INC_RR    Z80Opcode = 0x03 // INC rr
	Z80_DEC_RR    Z80Opcode = 0x0B // DEC rr

	// Logical
	Z80_AND_R  Z80Opcode = 0xA0 // AND r
	Z80_AND_N  Z80Opcode = 0xE6 // AND n
	Z80_AND_HL Z80Opcode = 0xA6 // AND (HL)
	Z80_OR_R   Z80Opcode = 0xB0 // OR r
	Z80_OR_N   Z80Opcode = 0xF6 // OR n
	Z80_OR_HL  Z80Opcode = 0xB6 // OR (HL)
	Z80_XOR_R  Z80Opcode = 0xA8 // XOR r
	Z80_XOR_N  Z80Opcode = 0xEE // XOR n
	Z80_XOR_HL Z80Opcode = 0xAE // XOR (HL)
	Z80_CP_R   Z80Opcode = 0xB8 // CP r (compare)
	Z80_CP_N   Z80Opcode = 0xFE // CP n
	Z80_CP_HL  Z80Opcode = 0xBE // CP (HL)

	// Bitwise (CB prefix instructions)
	Z80_BIT_I_R Z80Opcode = 0x40 // BIT b, r (test bit) - CB prefix
	Z80_SET_I_R Z80Opcode = 0xC0 // SET b, r (set bit) - CB prefix
	Z80_RES_I_R Z80Opcode = 0x80 // RES b, r (reset bit) - CB prefix

	// Rotate/Shift (CB prefix instructions)
	Z80_RLC_R Z80Opcode = 0x00 // RLC r (rotate left circular) - CB prefix
	Z80_RRC_R Z80Opcode = 0x08 // RRC r (rotate right circular) - CB prefix
	Z80_RL_R  Z80Opcode = 0x10 // RL r (rotate left through carry) - CB prefix
	Z80_RR_R  Z80Opcode = 0x18 // RR r (rotate right through carry) - CB prefix
	Z80_SLA_R Z80Opcode = 0x20 // SLA r (shift left arithmetic) - CB prefix
	Z80_SRA_R Z80Opcode = 0x28 // SRA r (shift right arithmetic) - CB prefix
	Z80_SRL_R Z80Opcode = 0x38 // SRL r (shift right logical) - CB prefix

	// Stack
	Z80_PUSH_QQ Z80Opcode = 0xC5 // PUSH qq
	Z80_POP_QQ  Z80Opcode = 0xC1 // POP qq

	// Jump/Branch
	Z80_JP_NN    Z80Opcode = 0xC3 // JP nn (unconditional jump)
	Z80_JP_HL    Z80Opcode = 0xE9 // JP (HL) (jump to address in HL)
	Z80_JP_CC_NN Z80Opcode = 0xC2 // JP cc, nn (conditional jump)
	Z80_JR_E     Z80Opcode = 0x18 // JR e (relative jump)
	Z80_JR_CC_E  Z80Opcode = 0x20 // JR cc, e (conditional relative jump)
	Z80_DJNZ_E   Z80Opcode = 0x10 // DJNZ e (decrement B and jump if not zero)

	// Call/Return
	Z80_CALL_NN    Z80Opcode = 0xCD // CALL nn
	Z80_CALL_CC_NN Z80Opcode = 0xC4 // CALL cc, nn (conditional call)
	Z80_RET        Z80Opcode = 0xC9 // RET
	Z80_RET_CC     Z80Opcode = 0xC0 // RET cc
	Z80_RETI       Z80Opcode = 0x4D // RETI (return from interrupt) - ED prefix
	Z80_RETN       Z80Opcode = 0x45 // RETN (return from NMI) - ED prefix
	Z80_RST_P      Z80Opcode = 0xC7 // RST p (restart to address p*8)

	// Special
	Z80_NOP  Z80Opcode = 0x00 // NOP
	Z80_HALT Z80Opcode = 0x76 // HALT
	Z80_DI   Z80Opcode = 0xF3 // DI (disable interrupts)
	Z80_EI   Z80Opcode = 0xFB // EI (enable interrupts)
	// others...
)

// OperandType specifies the kind of operand
type OperandType int

const (
	OpNone           OperandType = iota // No operand
	OpRegister                          // r: Physical register (A, B, C, D, E, H, L)
	OpRegisterPairRR                    // rr: Register pair (BC, DE, HL, SP)
	OpRegisterPairQQ                    // qq: Register pair (BC, DE, HL, AF)
	OpRegisterPairPP                    // pp: Register pair (BC, DE)
	OpImmediate                         // n: Constant value (8 bit)
	OpIndirect                          // nn: Memory reference (16 bit)
	OpRelExpression                     // e: Jump/branch target (-126 to +129)
	OpDisplacement                      // d: IX/IY displacement (-128 to +127)
	OpCondition                         // cc: Condition code (Z, NZ, C, NC)
	OpBitIndex                          // i: Bit index (0-7)
)

type ConditionCode int

const (
	Cond_NZ ConditionCode = iota // Not Zero
	Cond_Z                       // Zero
	Cond_NC                      // Not Carry
	Cond_C                       // Carry
	Cond_PO                      // Parity Odd
	Cond_PE                      // Parity Even
	Cond_P                       // Positive
	Cond_M                       // Minus
)

// ============================================================================
// Instruction Property Flags
// ============================================================================

type InstrProperties uint32

const (
	// Operand access patterns
	InstrReadsOp0  InstrProperties = 1 << 0 // Reads first operand
	InstrReadsOp1  InstrProperties = 1 << 1 // Reads second operand
	InstrWritesOp0 InstrProperties = 1 << 2 // Writes first operand
	InstrWritesOp1 InstrProperties = 1 << 3 // Writes second operand

	// Special properties
	InstrImmediate      InstrProperties = 1 << 4 // literal/immediate operand
	InstrIndirect       InstrProperties = 1 << 5 // Accesses memory
	InstrIsBranch       InstrProperties = 1 << 6 // Control flow instruction
	InstrIsCall         InstrProperties = 1 << 7 // Function call
	InstrHasAlternative InstrProperties = 1 << 8 // Has alternative encoding/instruction
)

type AffectedFlags uint32

const (
	// Flag effects (specific Z80 flags)
	InstrAffectsZ  AffectedFlags = 1 << 0 // Modifies Zero flag
	InstrAffectsN  AffectedFlags = 1 << 1 // Modifies Add/Subtract flag
	InstrAffectsH  AffectedFlags = 1 << 2 // Modifies Half-carry flag
	InstrAffectsC  AffectedFlags = 1 << 3 // Modifies Carry flag
	InstrAffectsS  AffectedFlags = 1 << 4 // Modifies Sign flag (bit 7)
	InstrAffectsPV AffectedFlags = 1 << 5 // Modifies Parity/Overflow flag
)

type InstrOperand struct {
	Type      OperandType
	Registers []*Register // allowed registers for this operand (if applicable)
}

// ============================================================================
// Instruction Descriptor
// ============================================================================

// InstrDescriptor describes properties of a Z80 instruction
type InstrDescriptor struct {
	Opcode        Z80Opcode
	Operands      []InstrOperand // Expected operands
	Properties    InstrProperties
	AffectedFlags AffectedFlags

	// Timing (in T-states/cycles)
	Cycles      int // Mandatory cycle count (for non-branching or branch-not-taken)
	CyclesTaken int // Additional cycles if branch is taken (0 for non-branch instructions)

	// TODO: these might be constant over the complete range of instructions?
	EncodingReg1SL int // Shift left of register-id in opcode encoding
	EncodingReg2SL int // Shift left of register-id in opcode encoding
}
