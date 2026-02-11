package cfg

import (
	"fmt"
	"strings"
	"zenith/compiler/zsm"
)

// instructionSelectorZ80 implements InstructionSelector for the Z80
type instructionSelectorZ80 struct {
	vrAlloc           *VirtualRegisterAllocator
	currentBlock      *BasicBlock // Current block for instruction emission
	callingConvention CallingConvention
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

// NewInstructionSelectorZ80 creates a new InstructionSelector for the Z80
func NewInstructionSelectorZ80(vrAlloc *VirtualRegisterAllocator) InstructionSelector {
	return &instructionSelectorZ80{
		vrAlloc:           vrAlloc,
		callingConvention: NewCallingConventionZ80(),
	}
}

// ============================================================================
// Arithmetic Operations
// ============================================================================

// SelectAdd generates instructions for addition (a + b)
func (z *instructionSelectorZ80) SelectAdd(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
		// TODO: refactor to handle immediate 16-bit addition
		// 16-bit add: ADD HL, rr
		result = z.vrAlloc.Allocate(Z80Registers16)
		vrHL := z.vrAlloc.Allocate(Z80RegHL)
		z.emit(newInstruction(Z80_LD_RR_NN, vrHL, left))
		z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, right))
		z.emit(newInstruction(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for ADD: %d", size)
	}

	return result, nil
}

// SelectSubtract generates instructions for subtraction (a - b)
func (z *instructionSelectorZ80) SelectSubtract(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
		vrHL := z.vrAlloc.Allocate(Z80RegHL)
		z.emit(newInstruction(Z80_LD_RR_NN, vrHL, left))
		// Clear carry flag first (OR A)
		z.emit(newInstruction(Z80_OR_R, vrA, vrA))
		z.emit(newInstruction(Z80_SBC_HL_RR, vrHL, right))
		z.emit(newInstruction(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for SUB: %d", size)
	}

	return result, nil
}

// SelectMultiply generates instructions for multiplication (a * b)
// Z80 has no multiply instruction - call runtime helper
// Intrinsic calling convention: __mul8(A, L) -> HL (16-bit), __mul16(HL, DE) -> HLDE (32-bit)
func (z *instructionSelectorZ80) SelectMultiply(left, right *VirtualRegister, resultSize RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectDivide(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectNegate(operand *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
	var result *VirtualRegister
	if size == 8 {
		result = z.emitLoadIntoReg8(operand, Z80RegA)
		z.emit(newInstruction(Z80_NEG, result, result))
	} else {
		return nil, fmt.Errorf("unsupported size for NEGATE: %d", size)
	}

	return result, nil
}

// ============================================================================
// Bitwise Operations
// ============================================================================

// SelectBitwiseAnd generates instructions for bitwise AND (a & b)
func (z *instructionSelectorZ80) SelectBitwiseAnd(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectBitwiseOr(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectBitwiseXor(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectBitwiseNot(operand *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectShiftLeft(value, amount *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectShiftRight(value *VirtualRegister, amount *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
	// TODO: Implement proper short-circuit with phi nodes
	leftVR, err := evaluateExpr(ctx, left)
	if err != nil {
		return nil, err
	}
	rightVR, err := evaluateExpr(ctx, right)
	if err != nil {
		return nil, err
	}

	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, leftVR))
	z.emit(newInstruction(Z80_LD_RR_NN, vrDE, rightVR))
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
	leftVR, err := evaluateExpr(ctx, left)
	if err != nil {
		return nil, err
	}
	rightVR, err := evaluateExpr(ctx, right)
	if err != nil {
		return nil, err
	}

	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, leftVR))
	z.emit(newInstruction(Z80_LD_RR_NN, vrDE, rightVR))
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
	operandVR, err := evaluateExpr(ctx, operand)
	if err != nil {
		return nil, err
	}

	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	z.emit(newInstruction(Z80_LD_RR_NN, vrHL, operandVR))
	z.emit(newCall("__logical_not"))

	result := z.vrAlloc.Allocate(Z80RegA)
	return result, nil
}

// ============================================================================
// Comparison Operations
// ============================================================================

// SelectEqual generates instructions for equality comparison (a == b)
func (z *instructionSelectorZ80) SelectEqual(ctx *ExprContext, left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectNotEqual(ctx *ExprContext, left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectLessThan(ctx *ExprContext, left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectGreaterThan(ctx *ExprContext, left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectLessEqual(ctx *ExprContext, left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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
func (z *instructionSelectorZ80) SelectGreaterEqual(ctx *ExprContext, left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
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

// SelectLoad generates instructions to load from memory
func (z *instructionSelectorZ80) SelectLoad(address *VirtualRegister, offset int, size RegisterSize) (*VirtualRegister, error) {
	var result *VirtualRegister

	switch size {
	case 8:
		vrHL := z.emitLoadIntoReg16(address, Z80RegHL)
		z.emitAddOffsetToHL(vrHL, int32(offset))

		result = z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newInstruction(Z80_LD_R_HL, result, vrHL))
	case 16:
		// Load 16-bit value
		return nil, fmt.Errorf("16-bit load not yet implemented")
	}
	return result, nil
}

// SelectLoadIndexed generates instructions to load from memory with a dynamic index
func (z *instructionSelectorZ80) SelectLoadIndexed(address *VirtualRegister, index *VirtualRegister, elementSize int, size RegisterSize) (*VirtualRegister, error) {
	// For now, implement simple 8-bit element access
	// TODO: Handle multi-byte elements (multiply index by elementSize)

	switch size {
	case 8:
		// Load base address into HL
		vrHL := z.emitLoadIntoReg16(address, Z80RegHL)

		// If element size > 1, need to scale the index
		scaledIndex := index
		if elementSize > 1 {
			// Multiply index by elementSize
			// For power-of-2 sizes, we could use shifts, but for now use simple add
			// TODO: Optimize with shifts for powers of 2
			scaledIndex = z.vrAlloc.Allocate(Z80Registers8)

			// Simple loop: scaledIndex = index * elementSize
			// For now, just use index directly and document limitation
			scaledIndex = index
		}

		// Add index to HL: HL = HL + index
		// We need index in a register that can be added to HL
		// Options: BC, DE, or expand via A
		vrIndexExpanded := z.emitLoadIntoReg16(scaledIndex, Z80RegDE)
		z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrIndexExpanded))

		// Load from (HL)
		result := z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newInstruction(Z80_LD_R_HL, result, vrHL))
		return result, nil

	case 16:
		return nil, fmt.Errorf("16-bit indexed load not yet implemented")
	}
	return nil, fmt.Errorf("unsupported size for indexed load: %d", size)
}

// SelectStore generates instructions to store to memory
func (z *instructionSelectorZ80) SelectStore(address *VirtualRegister, value *VirtualRegister, offset int, size RegisterSize) error {
	switch size {
	case 8:
		vrHL := z.emitLoadIntoReg16(address, Z80RegHL)
		z.emitAddOffsetToHL(vrHL, int32(offset))

		var opcode Z80Opcode
		if value.Type == ImmediateValue {
			opcode = Z80_LD_HL_N
		} else {
			opcode = Z80_LD_HL_R
		}

		z.emit(newInstruction(opcode, vrHL, value))
	case 16:
		return fmt.Errorf("16-bit store not yet implemented")
	}
	return nil // store has no result
}

// SelectLoadConstant generates instructions to load an immediate value
func (z *instructionSelectorZ80) SelectLoadConstant(value interface{}, size RegisterSize) (*VirtualRegister, error) {
	val := value.(int)
	result := z.vrAlloc.AllocateImmediate(int32(val), size)
	return result, nil
}

// SelectLoadVariable generates instructions to load a variable's value
func (z *instructionSelectorZ80) SelectLoadVariable(symbol *zsm.Symbol) (*VirtualRegister, error) {
	// TODO: Variable load not yet implemented
	// Decision needed: Use SP-relative addressing, HL indirection, or runtime helpers
	// IX/IY indexed addressing avoided due to instruction overhead
	return nil, fmt.Errorf("variable load not yet implemented for symbol '%s'", symbol.Name)
}

// SelectStoreVariable generates instructions to store to a variable
func (z *instructionSelectorZ80) SelectStoreVariable(symbol *zsm.Symbol, value *VirtualRegister) error {
	// TODO: Variable store not yet implemented
	// Decision needed: Use SP-relative addressing, HL indirection, or runtime helpers
	// IX/IY indexed addressing avoided due to instruction overhead
	return fmt.Errorf("variable store not yet implemented for symbol '%s'", symbol.Name)
}

// SelectMove moves a value from source to target
// Handles size conversions when necessary (e.g., 16-bit to 8-bit extracts low byte)
func (z *instructionSelectorZ80) SelectMove(target *VirtualRegister, source *VirtualRegister, size RegisterSize) error {
	switch size {
	case 8:
		z.emitLoadIntoReg8(source, target.AllowedSet)
	case 16:
		z.emitLoadIntoReg16(source, target.AllowedSet)
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
func (z *instructionSelectorZ80) SelectCall(functionName string, args []*VirtualRegister, returnSize RegisterSize) (*VirtualRegister, error) {
	// Set up arguments according to calling convention
	// For now, assume simple convention: pass in registers/stack

	callInstr := newCall(functionName)

	// Get return value if non-void
	if returnSize > 0 {
		returnReg := z.callingConvention.GetReturnValueRegister(returnSize)
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
func (z *instructionSelectorZ80) SelectFunctionPrologue(fn *zsm.SemFunctionDecl) error {
	// Z80 function prologue typically:
	// - Save registers that need preserving
	// - Allocate stack frame if needed
	return nil
}

// SelectFunctionEpilogue generates function exit code
func (z *instructionSelectorZ80) SelectFunctionEpilogue(fn *zsm.SemFunctionDecl) error {
	// Restore registers, deallocate stack frame
	return nil
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
func (z *instructionSelectorZ80) GetCallingConvention() CallingConvention {
	return z.callingConvention
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

func (z *instructionSelectorZ80) emitLoadIntoReg8(value *VirtualRegister, targetRegs []*Register) *VirtualRegister {
	if targetRegs[0].Size != 8 {
		return nil // Target register must be 8-bit
	}

	var vrTarget *VirtualRegister
	if !value.MatchAnyRegisters(targetRegs) {
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
	if !value.MatchAnyRegisters(targetRegs) {
		vrTarget = z.vrAlloc.Allocate(targetRegs)
		if value.Type == ImmediateValue {
			// Load immediate value into targetReg
			// Create instruction with immediate as operand, target as result
			z.emit(newInstruction(Z80_LD_RR_NN, vrTarget, value))
		} else if len(value.AllowedSet) > 0 {
			// extract the low and hi value registers
			loRegsValue, hiRegsValue := ToPairs(value.AllowedSet)
			loRegsTarget, hiRegsTarget := ToPairs(targetRegs)

			// LD targetReg[Lo], value[Lo]
			vrTargetLo := z.vrAlloc.Allocate(loRegsTarget)
			vrValueLo := z.vrAlloc.Allocate(loRegsValue)
			z.emit(newInstruction(Z80_LD_R_R, vrTargetLo, vrValueLo))

			// LD targetReg[Hi], value[Hi]
			vrTargetHi := z.vrAlloc.Allocate(hiRegsTarget)
			if len(hiRegsValue) != 0 {
				vrValueHi := z.vrAlloc.Allocate(hiRegsValue)
				z.emit(newInstruction(Z80_LD_R_R, vrTargetHi, vrValueHi))
			} else {
				// reset high byte to 0 - not used
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

// emitAddOffsetToHL adds an offset to the address in HL
func (z *instructionSelectorZ80) emitAddOffsetToHL(vrHL *VirtualRegister, offset int32) {
	if offset != 0 {
		// Add offset to address
		vrOffset := z.vrAlloc.AllocateImmediate(offset, Bits16)
		vrOffsetReg := z.vrAlloc.Allocate(Z80RegistersPP)
		z.emit(newInstruction(Z80_LD_RR_NN, vrOffsetReg, vrOffset))
		z.emit(newInstruction(Z80_ADD_HL_RR, vrHL, vrOffsetReg))
	}
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

// largestSize returns the larger of two RegisterSizes
func largestSize(a, b *VirtualRegister) RegisterSize {
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

// ============================================================================
// Z80-specific instruction representation
// ============================================================================

// machineInstructionZ80 represents a concrete Z80 instruction
type machineInstructionZ80 struct {
	opcode        Z80Opcode
	result        *VirtualRegister
	operands      []*VirtualRegister
	conditionCode ConditionCode
	branchTargets []*BasicBlock
	comment       string
}

// newInstruction creates a new Z80 instruction
func newInstruction(opcode Z80Opcode, result, operand *VirtualRegister) *machineInstructionZ80 {
	operands := []*VirtualRegister{}
	if operand != nil {
		operands = append(operands, operand)
	}
	return &machineInstructionZ80{
		opcode:   opcode,
		result:   result,
		operands: operands,
	}
}
func newInstructionResult(opcode Z80Opcode, result *VirtualRegister) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode: opcode,
		result: result,
	}
}
func newInstructionOperand(opcode Z80Opcode, operand *VirtualRegister) *machineInstructionZ80 {
	return newInstruction(opcode, nil, operand)
}
func newInstruction0(opcode Z80Opcode) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode: opcode,
	}
}

// newBranchInternal is used when no basic block is needed (e.g., JR)
// displacement is relative offset of machine instructions (not bytes)
func newBranchInternal(condition ConditionCode, displacement *VirtualRegister) *machineInstructionZ80 {
	machInstr := newInstruction(Z80_JR_CC_E, nil, displacement)
	machInstr.conditionCode = condition
	return machInstr
}

// newJumpWithCondition creates a conditional jump with explicit condition code
func newJumpWithCondition(condition ConditionCode, trueBlock, falseBlock *BasicBlock) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        Z80_JP_CC_NN,
		conditionCode: condition,
		branchTargets: []*BasicBlock{trueBlock, falseBlock},
	}
}

// newJump creates an unconditional jump
func newJump(opcode Z80Opcode, target *BasicBlock) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        opcode,
		branchTargets: []*BasicBlock{target},
	}
}

// TODO: target block? or do we resolve them seperately after instruction selection?
// newCall creates a function call
func newCall(functionName string) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:  Z80_CALL_NN,
		comment: functionName,
	}
}

// Implement MachineInstruction interface

func (z *machineInstructionZ80) GetResult() *VirtualRegister {
	return z.result
}

func (z *machineInstructionZ80) GetOperands() []*VirtualRegister {
	return z.operands
}

func (z *machineInstructionZ80) SetResult(vr *VirtualRegister) {
	z.result = vr
}

func (z *machineInstructionZ80) SetOperand(index int, vr *VirtualRegister) {
	if index < len(z.operands) {
		z.operands[index] = vr
	}
}

func (z *machineInstructionZ80) GetCategory() InstrCategory {
	// Lookup from descriptor table
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		return desc.Category
	}
	return CatOther
}

func (z *machineInstructionZ80) GetAddressingMode() AddressingMode {
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		return desc.AddressingMode
	}
	return 0
}

func (z *machineInstructionZ80) GetTargetBlocks() []*BasicBlock {
	if z.branchTargets == nil {
		return []*BasicBlock{}
	}
	return z.branchTargets
}

func (z *machineInstructionZ80) GetCost() InstructionCost {
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		return InstructionCost{
			Cycles: desc.Cycles,
			Size:   desc.Size,
		}
	}
	return InstructionCost{255, 255} // Unknown cost
}

func (z *machineInstructionZ80) String() string {

	var builder strings.Builder
	builder.WriteString(z.opcode.String())
	builder.WriteString(" ")
	if z.conditionCode != 0 {
		builder.WriteString(z.conditionCode.String())
		builder.WriteString(" ")
	}
	if z.comment != "" {
		builder.WriteString(z.comment)
		builder.WriteString(" ")
	}
	if len(z.branchTargets) > 0 {
		for _, target := range z.branchTargets {
			if target != nil {
				fmt.Fprintf(&builder, "Block %d ", target.ID)
			}
		}
	}

	operandStrs := make([]string, 0)
	if z.result != nil {
		operandStrs = append(operandStrs, z.result.String())
	}
	for _, op := range z.operands {
		operandStrs = append(operandStrs, op.String())
	}
	builder.WriteString(strings.Join(operandStrs, ", "))

	return builder.String()
}
