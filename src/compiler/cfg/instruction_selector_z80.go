package cfg

import (
	"fmt"
	"zenith/compiler/zsm"
)

// instructionSelectorZ80 implements InstructionSelector for the Z80
type instructionSelectorZ80 struct {
	vrAlloc      *VirtualRegisterAllocator
	currentBlock *BasicBlock // Current block for instruction emission
}

var Z80RegA = []*Register{&RegA}
var Z80RegB = []*Register{&RegB}
var Z80RegC = []*Register{&RegC}
var Z80RegD = []*Register{&RegD}
var Z80RegE = []*Register{&RegE}
var Z80RegH = []*Register{&RegH}
var Z80RegL = []*Register{&RegL}
var Z80RegHL = []*Register{&RegHL}
var Z80RegDE = []*Register{&RegDE}
var Z80RegBC = []*Register{&RegBC}
var Z80RegSP = []*Register{&RegSP}

// NewInstructionSelectorZ80 creates a new InstructionSelector for the Z80
func NewInstructionSelectorZ80(vrAlloc *VirtualRegisterAllocator) InstructionSelector {
	return &instructionSelectorZ80{
		vrAlloc: vrAlloc,
	}
}

// ============================================================================
// Arithmetic Operations
// ============================================================================

// SelectAdd generates instructions for addition (a + b)
func (z *instructionSelectorZ80) SelectAdd(left, right *VirtualRegister) (*VirtualRegister, error) {
	size := largestSize(left, right)
	var result *VirtualRegister

	// swap left.right if right is immediate and left is not
	imm, reg, isImm := orderImmediateFirst(left, right)

	switch size {
	case 8:
		var opcode Z80Opcode
		if isImm {
			opcode = Z80_ADD_A_N
		} else {
			reg, imm = orderToMatchRegisters(left, right, &RegA)
			opcode = Z80_ADD_A_R
		}

		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstruction(Z80_LD_R_R, vrA, reg))
		z.emit(newInstruction(opcode, vrA, imm))

		// for reg-alloc flexibility, move result to wider VR
		result = z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newInstruction(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit add: ADD HL, rr (HL is destination and first operand)
		vrHL := z.emitLoadIntoReg16(left, Z80RegHL)
		vrRight := z.emitLoadIntoReg16(right, Z80Registers16)
		z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrRight))
		// Result is in HL
		result = vrHL
	default:
		return nil, fmt.Errorf("unsupported size for ADD: %d", size)
	}

	return result, nil
}

// SelectSubtract generates instructions for subtraction (a - b)
func (z *instructionSelectorZ80) SelectSubtract(left, right *VirtualRegister) (*VirtualRegister, error) {
	size := largestSize(left, right)
	var result *VirtualRegister

	vrA := z.vrAlloc.Allocate(Z80RegA)
	switch size {
	case 8:
		result = z.vrAlloc.Allocate(Z80Registers8)
		// 8-bit subtract: SUB uses A register implicitly
		z.emit(newInstruction(Z80_LD_R_R, vrA, left))
		z.emit(newInstruction(Z80_SUB_R, vrA, right))
		z.emit(newInstruction(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit subtract: SBC HL, rr
		result = z.vrAlloc.Allocate(Z80Registers16)
		vrHL := z.emitLoadIntoReg16(left, Z80RegHL)
		// Clear carry flag first (OR A)
		z.emit(newInstruction(Z80_OR_R, vrA, vrA))
		z.emit(newInstruction(Z80_SBC_HL_RR, vrHL, right))
		// Store result back - decompose into component moves
		resultLo, resultHi := z.getOrAllocateComponents(result)
		vrL := z.vrAlloc.Allocate(Z80RegL)
		vrH := z.vrAlloc.Allocate(Z80RegH)
		z.emit(newInstruction(Z80_LD_R_R, resultLo, vrL))
		z.emit(newInstruction(Z80_LD_R_R, resultHi, vrH))
	default:
		return nil, fmt.Errorf("unsupported size for SUB: %d", size)
	}

	return result, nil
}

// SelectMultiply generates instructions for multiplication (a * b)
// Z80 has no multiply instruction - call runtime helper
// Intrinsic calling convention: __mul8(A, L) -> HL (16-bit), __mul16(HL, DE) -> HLDE (32-bit)
func (z *instructionSelectorZ80) SelectMultiply(left, right *VirtualRegister) (*VirtualRegister, error) {
	var result *VirtualRegister

	// Call multiply runtime helper based on operand size
	// 8-bit × 8-bit = 16-bit result in HL
	if left.Size == 8 && right.Size == 8 {
		left, right = orderToMatchRegisters(left, right, &RegA)
		// __mul8: params in A and L, result in HL (16-bit)
		z.emitLoadIntoReg8(left, Z80RegA)
		z.emitLoadIntoReg8(right, Z80RegL)
		callInstr := newCall("__mul8")
		result = z.vrAlloc.Allocate(Z80RegHL)
		callInstr.result = result
		z.emit(callInstr)
	} else {
		// __mul16: params in HL and DE, result in HLDE (32-bit)
		left, right = orderToMatchRegisters(left, right, &RegHL)
		z.emitLoadIntoReg16(left, Z80RegHL)
		z.emitLoadIntoReg16(right, Z80RegDE)
		callInstr := newCall("__mul16")
		result = z.vrAlloc.Allocate(Z80RegHL)
		// TODO: implement 32-bit registers.
		callInstr.result = result
		z.emit(callInstr)
	}

	return result, nil
}

// SelectDivide generates instructions for division (a / b)
// Z80 has no divide instruction - call runtime helper
// Intrinsic calling convention: __div8(HL, DE) -> A, __div16(HL, DE) -> HL
func (z *instructionSelectorZ80) SelectDivide(left, right *VirtualRegister) (*VirtualRegister, error) {
	size := largestSize(left, right)
	// call parameters
	z.emitLoadIntoReg16(left, Z80RegHL)
	z.emitLoadIntoReg16(right, Z80RegDE)

	var result *VirtualRegister
	var callInstr *machineInstructionZ80

	if size == 8 {
		// __div8: params in HL and DE, result in A
		callInstr = newCall("__div8")
		result = z.vrAlloc.Allocate(Z80RegA)
	} else {
		// __div16: params in HL and DE, result in HL
		callInstr = newCall("__div16")
		result = z.vrAlloc.Allocate(Z80RegHL)
	}

	callInstr.result = result
	z.emit(callInstr)
	return result, nil
}

// SelectNegate generates instructions for negation (-a)
func (z *instructionSelectorZ80) SelectNegate(operand *VirtualRegister) (*VirtualRegister, error) {
	size := operand.Size
	var result *VirtualRegister
	if size == 8 {
		result = z.emitLoadIntoReg8(operand, Z80RegA)
		z.emit(newInstruction(Z80_NEG, result, result))
	} else {
		return nil, fmt.Errorf("unsupported size for NEGATE: %d", size)
	}

	return result, nil
}

func (z *instructionSelectorZ80) SelectIncrement(operand *VirtualRegister) (*VirtualRegister, error) {
	size := operand.Size
	var result *VirtualRegister
	if size == 8 {
		result, _ = z.allocateRegistersFor(Z80_INC_R)
		// Load the operand value first
		z.emit(newInstruction(Z80_LD_R_R, result, operand))
		// Then increment it
		z.emit(newInstruction(Z80_INC_R, result, result))
	} else {
		return nil, fmt.Errorf("unsupported size for INCREMENT: %d", size)
	}

	return result, nil
}

func (z *instructionSelectorZ80) SelectDecrement(operand *VirtualRegister) (*VirtualRegister, error) {
	size := operand.Size
	var result *VirtualRegister
	if size == 8 {
		result, _ = z.allocateRegistersFor(Z80_DEC_R)
		// Load the operand value first
		z.emit(newInstruction(Z80_LD_R_R, result, operand))
		// Then decrement it
		z.emit(newInstruction(Z80_DEC_R, result, result))
	} else {
		return nil, fmt.Errorf("unsupported size for DECREMENT: %d", size)
	}

	return result, nil
}

// ============================================================================
// Bitwise Operations
// ============================================================================

// SelectBitwiseAnd generates instructions for bitwise AND (a & b)
func (z *instructionSelectorZ80) SelectBitwiseAnd(left, right *VirtualRegister) (*VirtualRegister, error) {
	size := largestSize(left, right)
	var result *VirtualRegister

	if size == 8 {
		result = z.vrAlloc.Allocate(Z80Registers8)
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstruction(Z80_LD_R_R, vrA, left))
		z.emit(newInstruction(Z80_AND_R, vrA, right))
		z.emit(newInstruction(Z80_LD_R_R, result, vrA))
	} else {
		// 16-bit AND: do byte-by-byte
		return nil, fmt.Errorf("16-bit AND not yet implemented")
	}

	return result, nil
}

// SelectBitwiseOr generates instructions for bitwise OR (a | b)
func (z *instructionSelectorZ80) SelectBitwiseOr(left, right *VirtualRegister) (*VirtualRegister, error) {
	size := largestSize(left, right)
	var result *VirtualRegister

	if size == 8 {
		result = z.vrAlloc.Allocate(Z80Registers8)
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstruction(Z80_LD_R_R, vrA, left))
		z.emit(newInstruction(Z80_OR_R, vrA, right))
		z.emit(newInstruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit OR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseXor generates instructions for bitwise XOR (a ^ b)
func (z *instructionSelectorZ80) SelectBitwiseXor(left, right *VirtualRegister) (*VirtualRegister, error) {
	size := largestSize(left, right)
	var result *VirtualRegister

	if size == 8 {
		result = z.vrAlloc.Allocate(Z80Registers8)
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstruction(Z80_LD_R_R, vrA, left))
		z.emit(newInstruction(Z80_XOR_R, vrA, right))
		z.emit(newInstruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit XOR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseNot generates instructions for bitwise NOT (~a)
func (z *instructionSelectorZ80) SelectBitwiseNot(operand *VirtualRegister) (*VirtualRegister, error) {
	size := operand.Size
	var result *VirtualRegister

	if size == 8 {
		// CPL instruction complements A
		result = z.vrAlloc.Allocate(Z80Registers8)
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstruction(Z80_LD_R_R, vrA, operand))
		vrFF := z.vrAlloc.AllocateImmediate(0xFF, 8)
		z.emit(newInstruction(Z80_XOR_N, vrA, vrFF))
		z.emit(newInstruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit NOT not yet implemented")
	}

	return result, nil
}

// SelectShiftLeft generates instructions for left shift (a << b)
func (z *instructionSelectorZ80) SelectShiftLeft(value, amount *VirtualRegister) (*VirtualRegister, error) {
	size := value.Size
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, value))
	z.emit(newInstruction(Z80_LD_RR_NN, vrDE, amount))

	var result *VirtualRegister
	if size == 8 {
		result = z.vrAlloc.Allocate(Z80RegA)
		z.emit(newCall("__shl8"))
	} else {
		result = z.vrAlloc.Allocate(Z80RegHL)
		z.emit(newCall("__shl16"))
	}

	return result, nil
}

// SelectShiftRight generates instructions for right shift (a >> b)
func (z *instructionSelectorZ80) SelectShiftRight(value *VirtualRegister, amount *VirtualRegister) (*VirtualRegister, error) {
	size := value.Size
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, value))
	z.emit(newInstruction(Z80_LD_RR_NN, vrDE, amount))

	var result *VirtualRegister
	if size == 8 {
		result = z.vrAlloc.Allocate(Z80RegA)
		z.emit(newCall("__shr8"))
	} else {
		result = z.vrAlloc.Allocate(Z80RegHL)
		z.emit(newCall("__shr16"))
	}

	return result, nil
}

// SelectLogicalAnd generates instructions for logical AND (a && b)
func (z *instructionSelectorZ80) SelectLogicalAnd(ctx *ExprContext, left, right zsm.SemExpression, evaluateExpr func(*ExprContext, zsm.SemExpression) (*VirtualRegister, error)) (*VirtualRegister, error) {
	// In BranchMode: implement short-circuit evaluation
	if ctx != nil && ctx.Mode == BranchMode {
		// Create a label/block for testing right operand if left is true
		// For now, evaluate left with inverted logic
		// If left is false, jump to false block (short-circuit)
		// Otherwise, fall through and evaluate right

		// Evaluate left: if false, jump to falseBlock
		leftCtx := NewExprContextBranch(nil, ctx.FalseBlock)
		_, err := evaluateExpr(leftCtx, left)
		if err != nil {
			return nil, err
		}

		// Left was true, now evaluate right with original context
		return evaluateExpr(ctx, right)
	}

	// ValueMode: for now, use runtime helper
	vrLeft, err := evaluateExpr(ctx, left)
	if err != nil {
		return nil, err
	}
	vrRight, err := evaluateExpr(ctx, right)
	if err != nil {
		return nil, err
	}

	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, vrLeft))
	z.emit(newInstruction(Z80_LD_RR_NN, vrDE, vrRight))
	z.emit(newCall("__logical_and"))

	result := z.vrAlloc.Allocate(Z80RegA)
	return result, nil
}

// SelectLogicalOr generates instructions for logical OR (a || b)
func (z *instructionSelectorZ80) SelectLogicalOr(ctx *ExprContext, left, right zsm.SemExpression, evaluateExpr func(*ExprContext, zsm.SemExpression) (*VirtualRegister, error)) (*VirtualRegister, error) {
	// In BranchMode: implement short-circuit evaluation
	if ctx != nil && ctx.Mode == BranchMode {
		// Evaluate left: if true, jump to trueBlock (short-circuit)
		// Otherwise, fall through and evaluate right

		leftCtx := NewExprContextBranch(ctx.TrueBlock, nil)
		_, err := evaluateExpr(leftCtx, left)
		if err != nil {
			return nil, err
		}

		// Left was false, now evaluate right with original context
		return evaluateExpr(ctx, right)
	}

	// ValueMode: for now, use runtime helper
	vrLeft, err := evaluateExpr(ctx, left)
	if err != nil {
		return nil, err
	}
	vrRight, err := evaluateExpr(ctx, right)
	if err != nil {
		return nil, err
	}

	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, vrLeft))
	z.emit(newInstruction(Z80_LD_RR_NN, vrDE, vrRight))
	z.emit(newCall("__logical_or"))

	result := z.vrAlloc.Allocate(Z80RegA)
	return result, nil
}

// SelectLogicalNot generates instructions for logical NOT (!a)
func (z *instructionSelectorZ80) SelectLogicalNot(ctx *ExprContext, operand zsm.SemExpression, evaluateExpr func(*ExprContext, zsm.SemExpression) (*VirtualRegister, error)) (*VirtualRegister, error) {
	// In BranchMode: invert the target blocks
	if ctx != nil && ctx.Mode == BranchMode {
		// Swap true and false blocks
		invertedCtx := NewExprContextBranch(ctx.FalseBlock, ctx.TrueBlock)
		return evaluateExpr(invertedCtx, operand)
	}

	// ValueMode: use runtime helper
	vrOperand, err := evaluateExpr(ctx, operand)
	if err != nil {
		return nil, err
	}

	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, vrOperand))
	z.emit(newCall("__logical_not"))

	result := z.vrAlloc.Allocate(Z80RegA)
	return result, nil
}

// ============================================================================
// Comparison Operations
// ============================================================================

// SelectEqual generates instructions for equality comparison (a == b)
func (z *instructionSelectorZ80) SelectEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error) {
	result, err := z.emitCompare(left, right)
	if err != nil {
		return nil, err
	}

	// In BranchMode: emit conditional branch based on flags
	if ctx != nil && ctx.Mode == BranchMode {
		z.emit(newJumpWithCondition(Cond_Z, ctx.TrueBlock, ctx.FalseBlock))
		return result, nil // No value produced
	}

	return z.emitFlagToRegA(Cond_Z)
}

// SelectNotEqual generates instructions for inequality comparison (a != b)
func (z *instructionSelectorZ80) SelectNotEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error) {
	result, err := z.emitCompare(left, right)
	if err != nil {
		return nil, err
	}

	// In BranchMode: emit conditional branch (NZ for not-equal)
	if ctx != nil && ctx.Mode == BranchMode {
		z.emit(newJumpWithCondition(Cond_NZ, ctx.TrueBlock, ctx.FalseBlock))
		return result, nil
	}

	return z.emitFlagToRegA(Cond_NZ)
}

// SelectLessThan generates instructions for less-than comparison (a < b)
func (z *instructionSelectorZ80) SelectLessThan(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error) {
	result, err := z.emitCompare(left, right)
	if err != nil {
		return nil, err
	}

	// In BranchMode: emit conditional branch (C for less-than unsigned)
	if ctx != nil && ctx.Mode == BranchMode {
		z.emit(newJumpWithCondition(Cond_C, ctx.TrueBlock, ctx.FalseBlock))
		return result, nil
	}

	return z.emitFlagToRegA(Cond_C)
}

// SelectGreaterThan generates instructions for greater-than comparison (a > b)
func (z *instructionSelectorZ80) SelectGreaterThan(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error) {
	result, err := z.emitCompare(left, right)
	if err != nil {
		return nil, err
	}

	// In BranchMode: emit conditional branch (C for less-than unsigned)
	if ctx != nil && ctx.Mode == BranchMode {
		z.emit(newJumpWithCondition(Cond_NC, ctx.TrueBlock, ctx.FalseBlock))
		return result, nil
	}

	return z.emitFlagToRegA(Cond_NC)
}

// SelectLessEqual generates instructions for less-or-equal comparison (a <= b)
func (z *instructionSelectorZ80) SelectLessEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error) {
	result, err := z.emitCompare(left, right)
	if err != nil {
		return nil, err
	}

	// In BranchMode: emit conditional branch (C or Z for <= unsigned)
	if ctx != nil && ctx.Mode == BranchMode {
		z.emit(newJumpWithCondition(Cond_Z, ctx.TrueBlock, nil))
		z.emit(newJumpWithCondition(Cond_C, ctx.TrueBlock, ctx.FalseBlock))
		return result, nil
	}

	return nil, fmt.Errorf("Value Mode not implemented for less-equal.")
}

// SelectGreaterEqual generates instructions for greater-or-equal comparison (a >= b)
func (z *instructionSelectorZ80) SelectGreaterEqual(ctx *ExprContext, left, right *VirtualRegister) (*VirtualRegister, error) {
	result, err := z.emitCompare(left, right)
	if err != nil {
		return nil, err
	}

	// In BranchMode: emit conditional branch (C or Z for <= unsigned)
	if ctx != nil && ctx.Mode == BranchMode {
		z.emit(newJumpWithCondition(Cond_Z, ctx.TrueBlock, nil))
		z.emit(newJumpWithCondition(Cond_NC, ctx.TrueBlock, ctx.FalseBlock))
		return result, nil
	}

	return nil, fmt.Errorf("Value Mode not implemented for greater-equal.")
}

// ============================================================================
// Memory Operations
// ============================================================================

// 	Move
// 	| Source  | Target  |
// 	|---------|---------|
// 	| Reg8    | Reg8    |
// 	| Reg8    | Reg16   | H=0
// 	| Reg16   | Reg16   |

// 	Store
// 	| Source  | Target  |
// 	|---------|---------|
// 	| Reg8    | Stack8  |
// 	| Reg8    | Stack16 | H=0
// 	| Reg16   | Stack16 |

// 	Load
// 	| Source  | Target  |
// 	|---------|---------|
// 	| Stack8  | Reg8    |
// 	| Stack8  | Reg16   | H=0
// 	| Stack16 | Reg16   |

// SelectLoad generates instructions to load from memory
func (z *instructionSelectorZ80) SelectLoad(address *VirtualRegister, offset uint16, size uint8) (*VirtualRegister, error) {
	var result *VirtualRegister

	switch size {
	case 8:
		vrHL := z.emitLoadIntoReg16(address, Z80RegHL)
		z.emitAddOffsetToHL(vrHL, offset)

		result = z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newInstruction(Z80_LD_R_HL, result, vrHL))
	case 16:
		// Load 16-bit value
		return nil, fmt.Errorf("16-bit load not yet implemented")
	}
	return result, nil
}

// SelectLoadIndexed generates instructions to load from memory with a dynamic index
func (z *instructionSelectorZ80) SelectLoadIndexed(address *VirtualRegister, index *VirtualRegister, elementSize uint16, size uint8) (*VirtualRegister, error) {
	// TODO: incorporate index*elementSize offset into address calculation instead of adding after loading base address

	// Materialize stack addresses
	if address.Type == StackAddress {
		var err error
		address, err = z.SelectLoadStackAddress(uint16(address.Value))
		if err != nil {
			return nil, err
		}
	}

	vrHL := z.emitLoadIntoReg16(address, Z80RegHL)

	if index.Type == ImmediateValue {
		// If index is an immediate, we can calculate offset directly
		offset := uint16(index.Value) * elementSize
		z.emitAddOffsetToHL(vrHL, offset)
	} else {
		vrIndex := z.emitLoadIntoReg16(index, Z80RegistersPP)
		// TODO: are 16-bit shifts (custom code) faster than multiple 16-bit adds?
		// Calculate offset: HL = base + index * elementSize
		for ; elementSize > 0; elementSize-- {
			z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrIndex))
		}
	}

	switch size {
	case 8:
		// Load from (HL)
		vrResult := z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newInstruction(Z80_LD_R_HL, vrResult, vrHL))
		return vrResult, nil
	case 16:
		// For 16-bit loads from memory, we can only load into defined component registers
		// Return the flexible result that can be allocated to any 16-bit register
		vrResult := z.vrAlloc.Allocate(Z80Registers16)

		// Create linked component VRs for the actual loads
		vrResultLo, vrResultHi := z.vrAlloc.AllocateComponents(vrResult)

		// Load low byte at (HL), high byte at (HL+1)
		z.emit(newInstruction(Z80_LD_R_HL, vrResultLo, vrHL))
		z.emit(newInstructionResult(Z80_INC_RR, vrHL))
		z.emit(newInstruction(Z80_LD_R_HL, vrResultHi, vrHL))

		// Return the composite parent VR
		// Liveness will connect vrResultLo/vrResultHi usage to vrResult via Register.Composition
		return vrResult, nil
	}

	return nil, fmt.Errorf("unsupported size for indexed load: %d", size)
}

// SelectLoadConstant generates instructions to load an immediate value
func (z *instructionSelectorZ80) SelectLoadConstant(value interface{}, size uint8) (*VirtualRegister, error) {
	val := value.(int)
	result := z.vrAlloc.AllocateImmediate(int32(val), size)
	return result, nil
}

// SelectLoadStackAddress generates instructions to compute the address of a stack location
// Returns a VR containing SP + stackOffset
func (z *instructionSelectorZ80) SelectLoadStackAddress(stackOffset uint16) (*VirtualRegister, error) {

	// Load offset into HL
	vrOffset := z.vrAlloc.AllocateImmediate(int32(stackOffset), 16)
	vrResult := z.vrAlloc.Allocate(Z80RegHL)
	z.emit(newInstruction(Z80_LD_RR_NN, vrResult, vrOffset))

	// Add SP to HL: HL = HL + SP
	// Note: Z80 ADD HL, RR adds a register pair to HL
	vrSP := z.vrAlloc.Allocate(Z80RegSP)
	z.emit(newInstruction(Z80_ADD_HL_RR, vrResult, vrSP))

	return vrResult, nil
}

// SelectStore generates instructions to store to memory
func (z *instructionSelectorZ80) SelectStore(address *VirtualRegister, value *VirtualRegister, offset uint16, size uint8) (*VirtualRegister, error) {
	// // Materialize stack addresses
	// if address.Type == StackAddress {
	// 	var err error
	// 	address, err = z.SelectLoadStackAddress(uint16(address.Value))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	vrHL := z.emitLoadIntoReg16(address, Z80RegHL)
	z.emitAddOffsetToHL(vrHL, offset)

	var opcode Z80Opcode
	if value.Type == ImmediateValue {
		opcode = Z80_LD_HL_N
	} else {
		opcode = Z80_LD_HL_R
	}

	switch size {
	case 8:
		z.emit(newInstruction(opcode, vrHL, value))
	case 16:
		var vrLo, vrHi *VirtualRegister
		if value.Type == ImmediateValue {
			vrLo, vrHi = z.splitImmediateValue16(value)
		} else {
			vrLo, vrHi = z.vrAlloc.AllocateComponents(value)
		}

		z.emitStore16AtHL(vrHL, vrLo, vrHi, opcode)
	}
	return vrHL, nil
}

// SelectMove moves a value from source to target
// Handles size conversions when necessary (e.g., 16-bit to 8-bit extracts low byte)
func (z *instructionSelectorZ80) SelectMove(target *VirtualRegister, source *VirtualRegister, size uint8) error {
	// Skip no-op moves where source and target are the same VR
	if target == source {
		return nil
	}

	switch target.Type {
	case CandidateRegister, AllocatedRegister:
		switch size {
		case 8:
			// For 8-bit, check if source is compatible with target constraints
			if source.MatchAnyRegisters(target.AllowedSet) {
				// Source is already compatible, just emit a move from source to target
				z.emit(newInstruction(Z80_LD_R_R, target, source))
			} else {
				// Need to load source into target's allowed register set first
				vrTemp := z.emitLoadIntoReg8(source, target.AllowedSet)
				if vrTemp != target {
					z.emit(newInstruction(Z80_LD_R_R, target, vrTemp))
				}
			}
		case 16:
			// For 16-bit moves, create component VRs and emit component moves
			// The register allocator will handle any AllowedSet constraints
			vrTargetLo, vrTargetHi := z.vrAlloc.AllocateComponents(target)
			var vrSourceLo, vrSourceHi *VirtualRegister

			if (source.Size == 8) && (source.Type == CandidateRegister) {
				vrSourceLo = source
				vrSourceHi = z.vrAlloc.AllocateImmediate(0, 8)
			} else {
				vrSourceLo, vrSourceHi = z.getOrAllocateComponents(source)
			}

			z.emit(newInstruction(Z80_LD_R_R, vrTargetLo, vrSourceLo))
			z.emit(newInstruction(Z80_LD_R_R, vrTargetHi, vrSourceHi))

		}
	// case StackLocation:
	// 	z.emitStoreOnStack(source, target)
	default:
		return fmt.Errorf("unsupported target register type for move: %v", target.Type)
	}
	return nil
}

// ============================================================================
// Control Flow
// ============================================================================

// SelectJump generates an unconditional jump
func (z *instructionSelectorZ80) SelectJump(target *BasicBlock) error {
	z.emit(newJump(Z80_JP_NN, target))
	return nil
}

// SelectCall generates a function call
func (z *instructionSelectorZ80) SelectCall(functionName string, args []*VirtualRegister, returnSize uint8) (*VirtualRegister, error) {
	// Set up arguments according to calling convention
	// For now, assume simple convention: pass in registers/stack

	callInstr := newCall(functionName)

	// Get return value if non-void
	if returnSize > 0 {
		callConv := z.GetCallingConvention(nil)
		returnReg := callConv.GetReturnValueRegister(returnSize)
		result := z.vrAlloc.Allocate([]*Register{returnReg})
		// Associate the result VR with the CALL instruction for proper liveness tracking
		callInstr.result = result
		z.emit(callInstr)
		return result, nil
	}

	z.emit(callInstr)
	return nil, nil
}

// SelectReturn generates a return statement
func (z *instructionSelectorZ80) SelectReturn(value *VirtualRegister) error {
	// Value should already be in return register (set by caller)
	z.emit(newInstruction0(Z80_RET))
	return nil
}

// ============================================================================
// Function Management
// ============================================================================

// SelectFunctionPrologue generates function entry code
func (z *instructionSelectorZ80) SelectFunctionPrologue(fn *zsm.SemFunctionDecl, frameSize uint16) error {
	// Allocate stack frame size needed
	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrSP := z.vrAlloc.Allocate(Z80RegSP)
	// negative frameSize: stack grows downwards
	vrSize := z.vrAlloc.AllocateImmediate(-int32(frameSize), 16)
	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, vrSize))
	z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrSP))
	z.emit(newInstruction(Z80_LD_SP_HL, vrSP, vrHL))
	return nil
}

// SelectFunctionEpilogue generates function exit code
func (z *instructionSelectorZ80) SelectFunctionEpilogue(fn *zsm.SemFunctionDecl, frameSize uint16) error {
	// Deallocate stack frame size
	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrSP := z.vrAlloc.Allocate(Z80RegSP)
	vrSize := z.vrAlloc.AllocateImmediate(int32(frameSize), 16)
	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, vrSize))
	z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrSP))
	z.emit(newInstruction(Z80_LD_SP_HL, vrSP, vrHL))

	// return from function
	z.emit(newInstruction0(Z80_RET))
	return nil
}

// ============================================================================
// Register Management
// ============================================================================

func (z *instructionSelectorZ80) CreateMove(target *VirtualRegister, source *VirtualRegister) ([]MachineInstruction, error) {
	var instrs []MachineInstruction

	// No-op: same register
	if target.PhysicalReg == source.PhysicalReg {
		return instrs, nil // Empty list = no instructions needed
	}

	// Handle 8-bit register moves
	if target.Size == 8 && source.Size == 8 {
		instrs = append(instrs, newInstruction(Z80_LD_R_R, target, source))
		return instrs, nil
	}

	// Handle 16-bit register moves
	if target.Size == 16 && source.Size == 16 {
		// Special case: HL -> SP (only valid 16-bit move instruction on Z80)
		if target.PhysicalReg == &RegSP && source.PhysicalReg == &RegHL {
			instrs = append(instrs, newInstruction(Z80_LD_SP_HL, target, source))
			return instrs, nil
		}

		// For other 16-bit moves where both have composition, decompose into 8-bit component moves
		if source.PhysicalReg != nil && target.PhysicalReg != nil &&
			len(source.PhysicalReg.Composition) > 0 && len(target.PhysicalReg.Composition) > 0 {

			// Create VRs for components
			srcLow := z.allocateComponentForPhysicalReg(source.PhysicalReg, 0)
			srcHigh := z.allocateComponentForPhysicalReg(source.PhysicalReg, 1)

			dstLow := z.allocateComponentForPhysicalReg(target.PhysicalReg, 0)
			dstHigh := z.allocateComponentForPhysicalReg(target.PhysicalReg, 1)

			// Emit component moves
			instrs = append(instrs, newInstruction(Z80_LD_R_R, dstLow, srcLow))
			instrs = append(instrs, newInstruction(Z80_LD_R_R, dstHigh, srcHigh))
			return instrs, nil
		}

		return nil, fmt.Errorf("cannot create 16-bit move from %v to %v", source.PhysicalReg, target.PhysicalReg)
	}

	return nil, fmt.Errorf("cannot create move from size %d to size %d", source.Size, target.Size)
}

func (z *instructionSelectorZ80) CreateSpill(vr *VirtualRegister, stackOffset int8) ([]MachineInstruction, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (z *instructionSelectorZ80) CreateReload(vr *VirtualRegister, stackOffset int8) ([]MachineInstruction, error) {
	return nil, fmt.Errorf("Not implemented")
}

// ============================================================================
// Utility
// ============================================================================

// SetCurrentBlock sets the active block for instruction emission
func (z *instructionSelectorZ80) SetCurrentBlock(block *BasicBlock) {
	z.currentBlock = block
}

// emit is a helper that emits to the current block
func (z *instructionSelectorZ80) emit(instr MachineInstruction) {
	z.currentBlock.MachineInstructions = append(z.currentBlock.MachineInstructions, instr)
}

// GetCallingConvention returns the calling convention
func (z *instructionSelectorZ80) GetCallingConvention(funcDecl *zsm.SemFunctionDecl) CallingConvention {
	return NewCallingConventionZ80()
}

// GetTargetRegisters returns the set of physical registers available on Z80
func (z *instructionSelectorZ80) GetTargetRegisters() []*Register {
	return Z80Registers
}

// ============================================================================
// Z80-specific helper types
// ============================================================================

// allocateRegistersFor creates VRs with constraints from instruction opcode
// Returns (result, operand) - either can be nil if not applicable
func (z *instructionSelectorZ80) allocateRegistersFor(opcode Z80Opcode) (result *VirtualRegister, operand *VirtualRegister) {
	desc := Z80InstrDescriptors[opcode]

	for _, dep := range desc.Dependencies {
		// Only care about register dependencies
		if dep.Type != OpRegister && dep.Type != OpRegisterPairPP &&
			dep.Type != OpRegisterPairQQ && dep.Type != OpRegisterPairRR {
			continue
		}

		switch dep.Access {
		case AccessWrite, AccessReadWrite:
			// This is a result/destination - allocate new VR
			if len(dep.Registers) > 0 && result == nil {
				result = z.vrAlloc.Allocate(dep.Registers)
			}
		case AccessRead:
			// This is an operand - ensure it's constrained correctly
			if operand == nil {
				operand = z.vrAlloc.Allocate(dep.Registers)
			}
		}
	}

	return result, operand
}

// getOrAllocateComponents returns the component VRs for a 16-bit VR, reusing existing ones if available
func (z *instructionSelectorZ80) getOrAllocateComponents(vr *VirtualRegister) (loVR, hiVR *VirtualRegister) {
	if len(vr.ComponentVRs) == 2 {
		// Reuse existing components
		return vr.ComponentVRs[0], vr.ComponentVRs[1]
	}
	// Create new components
	return z.vrAlloc.AllocateComponents(vr)
}

// allocateComponentForPhysicalReg creates and assigns a VR for a specific physical register component
func (z *instructionSelectorZ80) allocateComponentForPhysicalReg(physReg *Register, componentIndex int) *VirtualRegister {
	componentReg := physReg.Composition[componentIndex]
	vr := z.vrAlloc.Allocate([]*Register{componentReg})
	vr.Assign(componentReg)
	return vr
}

// splitImmediateValue16 splits a 16-bit immediate value into low and high byte VRs
func (z *instructionSelectorZ80) splitImmediateValue16(value *VirtualRegister) (loVR, hiVR *VirtualRegister) {
	valueLo := value.Value & 0xFF
	valueHi := (value.Value >> 8) & 0xFF
	loVR = z.vrAlloc.AllocateImmediate(int32(valueLo), 8)
	hiVR = z.vrAlloc.AllocateImmediate(int32(valueHi), 8)
	return loVR, hiVR
}

// emitStore16AtHL emits instructions to store a 16-bit value at (HL) in little-endian format
// Uses the specified opcode (Z80_LD_HL_R for registers, Z80_LD_HL_N for immediates)
func (z *instructionSelectorZ80) emitStore16AtHL(vrHL *VirtualRegister, loVR, hiVR *VirtualRegister, opcode Z80Opcode) {
	z.emit(newInstruction(opcode, vrHL, loVR))
	z.emit(newInstructionResult(Z80_INC_RR, vrHL))
	z.emit(newInstruction(opcode, vrHL, hiVR))
}

func (z *instructionSelectorZ80) emitLoadIntoReg8(value *VirtualRegister, targetRegs []*Register) *VirtualRegister {
	if targetRegs[0].Size != 8 {
		return nil // Target register must be 8-bit
	}

	var vrTarget *VirtualRegister
	if !value.MatchRegisters(targetRegs) {
		vrTarget = z.vrAlloc.Allocate(targetRegs)
		if value.Type == ImmediateValue {
			// Load immediate value into targetReg
			z.emit(newInstruction(Z80_LD_R_N, vrTarget, value))
		} else if len(value.AllowedSet) > 0 {
			// Handle size mismatch: if source is 16-bit, extract low byte
			sourceVR := value
			if value.Size == 16 {
				lowRegs, _ := ToPairs(value.AllowedSet)
				sourceVR = z.vrAlloc.Allocate(lowRegs)
			}
			// LD targetReg, value
			z.emit(newInstruction(Z80_LD_R_R, vrTarget, sourceVR))
		}
		// else - cannot do it => nil
	} else {
		vrTarget = value
	}
	return vrTarget
}

// emitLoadIntoReg16 loads a 16-bit value (register or immediate) into the target register
func (z *instructionSelectorZ80) emitLoadIntoReg16(value *VirtualRegister, targetRegs []*Register) *VirtualRegister {
	if targetRegs[0].Size != 16 {
		return nil // Target register must be 16-bit
	}

	var vrTarget *VirtualRegister
	// Check if value is already constrained to exactly the target registers
	if !value.MatchRegisters(targetRegs) {
		vrTarget = z.vrAlloc.Allocate(targetRegs)
		if value.Type == ImmediateValue {
			// Load immediate value into targetReg
			// Create instruction with immediate as operand, target as result
			z.emit(newInstruction(Z80_LD_RR_NN, vrTarget, value))
		} else if len(value.AllowedSet) > 0 {
			// Move 16-bit register by decomposing into component 8-bit moves.
			// The liveness analysis will track that using component registers
			// marks the parent VR as used via Register.Composition relationships.
			// The parent VR (vrTarget) will be marked as used by MarkUnusedVirtualRegisters
			// because its components are used.

			// Create linked component VRs for target
			vrTargetLo, vrTargetHi := z.getOrAllocateComponents(vrTarget)

			// LD targetReg[Lo], value[Lo]
			if value.Size == 16 {
				// Source is 16-bit - create linked component VRs
				vrValueLo, vrValueHi := z.getOrAllocateComponents(value)
				z.emit(newInstruction(Z80_LD_R_R, vrTargetLo, vrValueLo))
				z.emit(newInstruction(Z80_LD_R_R, vrTargetHi, vrValueHi))
			} else {
				// Source is 8-bit - just use it directly for low byte, zero-extend high byte
				z.emit(newInstruction(Z80_LD_R_R, vrTargetLo, value))
				vrZero := z.vrAlloc.AllocateImmediate(0, 8)
				z.emit(newInstruction(Z80_LD_R_N, vrTargetHi, vrZero))
			}
		}
		// else - cannot do it => nil
	} else {
		vrTarget = value
	}
	return vrTarget
}

func (z *instructionSelectorZ80) emitStoreOnStack(value *VirtualRegister, stackTarget *VirtualRegister) *VirtualRegister {

	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrSP := z.vrAlloc.Allocate(Z80RegSP)
	z.emitAddOffsetToHL(vrHL, uint16(stackTarget.Value))
	z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrSP))

	switch value.Size {
	case 8:
		switch value.Type {
		case ImmediateValue:
			z.emit(newInstruction(Z80_LD_HL_N, vrHL, value))
		case CandidateRegister:
			z.emit(newInstruction(Z80_LD_HL_R, vrHL, value))
		default:
			return nil // unsupported value type
		}
	case 16:
		switch value.Type {
		case ImmediateValue:
			loVR, hiVR := z.splitImmediateValue16(value)
			z.emitStore16AtHL(vrHL, loVR, hiVR, Z80_LD_HL_N)
		case CandidateRegister:
			loVR, hiVR := z.vrAlloc.AllocateComponents(value)
			z.emitStore16AtHL(vrHL, loVR, hiVR, Z80_LD_HL_R)
		default:
			return nil // unsupported value type
		}
	default:
		return nil // unsupported size
	}

	return vrHL
}

// emitAddOffsetToHL adds an offset to the address in HL
func (z *instructionSelectorZ80) emitAddOffsetToHL(vrHL *VirtualRegister, offset uint16) {
	if offset == 0 {
		return // no offset to add
	}

	if offset < 4 {
		// For small offsets, use INC HL multiple times
		for range offset {
			z.emit(newInstructionResult(Z80_INC_RR, vrHL))
		}
		return
	}

	// Add offset to address
	vrOffset := z.vrAlloc.AllocateImmediate(int32(offset), 16)
	vrOffsetReg := z.vrAlloc.Allocate(Z80RegistersPP)
	z.emit(newInstruction(Z80_LD_RR_NN, vrOffsetReg, vrOffset))
	z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrOffsetReg))
}

// emitCompare emits instructions to compare two VirtualRegisters
// Returns a VirtualRegister containing the comparison result (if needed)
// Sets flags accordingly
func (z *instructionSelectorZ80) emitCompare(left, right *VirtualRegister) (*VirtualRegister, error) {
	regSize := largestSize(left, right)

	switch regSize {
	case 8:
		var opcode Z80Opcode
		if left.Type == ImmediateValue {
			// CP N, r
			opcode = Z80_LD_R_N
		} else {
			// CP r, r
			opcode = Z80_LD_R_R
		}
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstruction(opcode, vrA, left))

		if right.Type == ImmediateValue {
			opcode = Z80_CP_N
		} else {
			opcode = Z80_CP_R
		}
		z.emit(newInstruction(opcode, vrA, right))
		return vrA, nil
	case 16:
		// ld hl, reg
		vrHL := z.emitLoadIntoReg16(left, Z80RegHL)
		// ld bc|de, imm
		vrDE := z.emitLoadIntoReg16(right, Z80RegistersPP)

		// or a(, a) - clears carry flag
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstructionResult(Z80_OR_R, vrA))
		// sbc hl, bc|de
		z.emit(newInstruction(Z80_SBC_HL_RR, vrHL, vrDE))
		// add hl, bc|de
		z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrDE))
		// c and z flags set accordingly
		return vrHL, nil
	default:
		return nil, fmt.Errorf("unsupported size for COMPARE: %d", regSize)
	}
}

// emitFlagToRegA converts a CPU flag to a boolean in register A (0 or 1)
func (z *instructionSelectorZ80) emitFlagToRegA(conditionCode ConditionCode) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(Z80RegA)

	vrZero := z.vrAlloc.AllocateImmediate(0, 8)

	// do not use 'xor a' here, as it clears flags
	switch conditionCode {
	case Cond_Z, Cond_NZ:
		vrOne := z.vrAlloc.AllocateImmediate(1, 8)
		z.emit(newInstruction(Z80_LD_R_N, result, vrZero))
		z.emit(newBranchInternal(conditionCode, vrOne)) // 1: jump over next instruction
		z.emit(newInstructionResult(Z80_INC_R, result))
	case Cond_C:
		z.emit(newInstruction(Z80_LD_R_N, result, vrZero))
		z.emit(newInstruction(Z80_ADC_A_N, result, vrZero))
	case Cond_NC:
		z.emit(newInstructionResult(Z80_SBC_A_R, result))
		z.emit(newInstructionResult(Z80_INC_R, result))
	default:
		return nil, fmt.Errorf("unsupported flag for bool conversion: %v", conditionCode)
	}
	return result, nil
}

// largestSize returns the larger of two uint8s
func largestSize(a, b *VirtualRegister) uint8 {
	if a.Size >= b.Size {
		return a.Size
	}
	return b.Size
}

// orderImmediateFirst checks two VRs and returns them ordered with immediate first if applicable
func orderImmediateFirst(left, right *VirtualRegister) (immediate *VirtualRegister, other *VirtualRegister, isImmediate bool) {
	if right.Type == ImmediateValue && left.Type != ImmediateValue {
		return right, left, true
	} else if left.Type == ImmediateValue && right.Type != ImmediateValue {
		return left, right, true
	} else if left.Type == ImmediateValue && right.Type == ImmediateValue {
		// error: should have been constant folded earlier
		return nil, nil, false
	}
	return left, right, false
}

func orderToMatchRegisters(left, right *VirtualRegister, reg *Register) (first *VirtualRegister, second *VirtualRegister) {
	if left.HasRegister(reg) {
		return left, right
	}
	if right.HasRegister(reg) {
		return right, left
	}
	return left, right
}
