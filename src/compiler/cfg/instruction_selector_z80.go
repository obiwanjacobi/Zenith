package cfg

import (
	"fmt"
	"zenith/compiler/zir"
)

// Z80InstructionSelector implements InstructionSelector for the Z80
type Z80InstructionSelector struct {
	vrAlloc           *VirtualRegisterAllocator
	instructions      []MachineInstruction
	callingConvention CallingConvention
	labelCounter      int
}

// NewZ80InstructionSelector creates a new Z80 instruction selector
func NewZ80InstructionSelector(cc CallingConvention) *Z80InstructionSelector {
	return &Z80InstructionSelector{
		vrAlloc:           NewVirtualRegisterAllocator(),
		instructions:      make([]MachineInstruction, 0),
		callingConvention: cc,
		labelCounter:      0,
	}
}

// ============================================================================
// Arithmetic Operations
// ============================================================================

// SelectAdd generates instructions for addition (a + b)
func (z *Z80InstructionSelector) SelectAdd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		// 8-bit add: requires A register
		// LD A, left
		// ADD A, right
		// LD result, A
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.EmitInstruction(NewZ80Instruction(Z80_ADD_A_R, vrA, right))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit add: ADD HL, rr
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
		z.EmitInstruction(NewZ80Instruction(Z80_ADD_HL_RR, vrHL, right))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for ADD: %d", size)
	}

	return result, nil
}

// SelectSubtract generates instructions for subtraction (a - b)
func (z *Z80InstructionSelector) SelectSubtract(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
	switch size {
	case 8:
		// 8-bit subtract: SUB uses A register implicitly
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.EmitInstruction(NewZ80Instruction(Z80_SUB_R, vrA, right))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit subtract: SBC HL, rr
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
		// Clear carry flag first (OR A)
		z.EmitInstruction(NewZ80Instruction(Z80_OR_R, vrA, vrA))
		z.EmitInstruction(NewZ80Instruction(Z80_SBC_HL_RR, vrHL, right))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for SUB: %d", size)
	}

	return result, nil
}

// SelectMultiply generates instructions for multiplication (a * b)
// Z80 has no multiply instruction - call runtime helper
func (z *Z80InstructionSelector) SelectMultiply(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	// Prepare arguments according to calling convention
	// Typically: HL = left, DE = right, result in HL
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	// Call multiply runtime helper
	if size == 8 {
		z.EmitInstruction(NewZ80Call("__mul8"))
	} else {
		z.EmitInstruction(NewZ80Call("__mul16"))
	}

	// Result is in HL
	result := z.vrAlloc.Allocate(size)
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))

	return result, nil
}

// SelectDivide generates instructions for division (a / b)
// Z80 has no divide instruction - call runtime helper
func (z *Z80InstructionSelector) SelectDivide(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__div8"))
	} else {
		z.EmitInstruction(NewZ80Call("__div16"))
	}

	result := z.vrAlloc.Allocate(size)
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))

	return result, nil
}

// SelectNegate generates instructions for negation (-a)
func (z *Z80InstructionSelector) SelectNegate(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		// Two's complement: XOR 0xFF, INC
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, operand))
		z.EmitInstruction(NewZ80InstructionImm8(Z80_XOR_N, vrA, 0xFF))
		z.EmitInstruction(NewZ80Instruction(Z80_INC_R, vrA, nil))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("unsupported size for NEGATE: %d", size)
	}

	return result, nil
}

// ============================================================================
// Bitwise Operations
// ============================================================================

// SelectBitwiseAnd generates instructions for bitwise AND (a & b)
func (z *Z80InstructionSelector) SelectBitwiseAnd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.EmitInstruction(NewZ80Instruction(Z80_AND_R, vrA, right))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		// 16-bit AND: do byte-by-byte
		return nil, fmt.Errorf("16-bit AND not yet implemented")
	}

	return result, nil
}

// SelectBitwiseOr generates instructions for bitwise OR (a | b)
func (z *Z80InstructionSelector) SelectBitwiseOr(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.EmitInstruction(NewZ80Instruction(Z80_OR_R, vrA, right))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit OR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseXor generates instructions for bitwise XOR (a ^ b)
func (z *Z80InstructionSelector) SelectBitwiseXor(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.EmitInstruction(NewZ80Instruction(Z80_XOR_R, vrA, right))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit XOR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseNot generates instructions for bitwise NOT (~a)
func (z *Z80InstructionSelector) SelectBitwiseNot(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		// CPL instruction complements A
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, operand))
		z.EmitInstruction(NewZ80InstructionImm8(Z80_XOR_N, vrA, 0xFF))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit NOT not yet implemented")
	}

	return result, nil
}

// SelectShiftLeft generates instructions for left shift (a << b)
func (z *Z80InstructionSelector) SelectShiftLeft(value, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, value))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, amount))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__shl8"))
	} else {
		z.EmitInstruction(NewZ80Call("__shl16"))
	}

	result := z.vrAlloc.AllocateConstrained(size, []*Register{&RegHL}, RegisterClassIndex)

	return result, nil
}

// SelectShiftRight generates instructions for right shift (a >> b)
func (z *Z80InstructionSelector) SelectShiftRight(value *VirtualRegister, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, value))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, amount))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__shr8"))
	} else {
		z.EmitInstruction(NewZ80Call("__shr16"))
	}

	result := z.vrAlloc.AllocateConstrained(size, []*Register{&RegHL}, RegisterClassIndex)

	return result, nil
}

// SelectLogicalAnd generates instructions for logical AND (a && b)
func (z *Z80InstructionSelector) SelectLogicalAnd(left, right *VirtualRegister) (*VirtualRegister, error) {
	// For logical AND, we need short-circuit evaluation which requires CFG support
	// For now, use runtime helper that evaluates both operands
	// TODO: Handle short-circuit evaluation at CFG level
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))
	z.EmitInstruction(NewZ80Call("__logical_and"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLogicalOr generates instructions for logical OR (a || b)
func (z *Z80InstructionSelector) SelectLogicalOr(left, right *VirtualRegister) (*VirtualRegister, error) {
	// For logical OR, we need short-circuit evaluation which requires CFG support
	// For now, use runtime helper that evaluates both operands
	// TODO: Handle short-circuit evaluation at CFG level
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))
	z.EmitInstruction(NewZ80Call("__logical_or"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLogicalNot generates instructions for logical NOT (!a)
func (z *Z80InstructionSelector) SelectLogicalNot(operand *VirtualRegister) (*VirtualRegister, error) {
	// Use runtime helper to convert operand != 0 to boolean, then invert
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, operand))
	z.EmitInstruction(NewZ80Call("__logical_not"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// ============================================================================
// Comparison Operations
// ============================================================================

// SelectEqual generates instructions for equality comparison (a == b)
func (z *Z80InstructionSelector) SelectEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(8) // Boolean result

	if size == 8 {
		// CP sets flags, then check Z flag
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.EmitInstruction(NewZ80Instruction(Z80_CP_R, vrA, right))
		// TODO: Convert flags to 0/1 value
	}

	return result, nil
}

// SelectNotEqual generates instructions for inequality comparison (a != b)
func (z *Z80InstructionSelector) SelectNotEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	// For comparison operations that return boolean, use runtime helper
	// Converting flags to 0/1 values requires control flow or special instructions
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__cmp_ne8"))
	} else {
		z.EmitInstruction(NewZ80Call("__cmp_ne16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLessThan generates instructions for less-than comparison (a < b)
func (z *Z80InstructionSelector) SelectLessThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__cmp_lt8"))
	} else {
		z.EmitInstruction(NewZ80Call("__cmp_lt16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLessEqual generates instructions for less-or-equal comparison (a <= b)
func (z *Z80InstructionSelector) SelectLessEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__cmp_le8"))
	} else {
		z.EmitInstruction(NewZ80Call("__cmp_le16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectGreaterThan generates instructions for greater-than comparison (a > b)
func (z *Z80InstructionSelector) SelectGreaterThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__cmp_gt8"))
	} else {
		z.EmitInstruction(NewZ80Call("__cmp_gt16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectGreaterEqual generates instructions for greater-or-equal comparison (a >= b)
func (z *Z80InstructionSelector) SelectGreaterEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.EmitInstruction(NewZ80Call("__cmp_ge8"))
	} else {
		z.EmitInstruction(NewZ80Call("__cmp_ge16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// ============================================================================
// Memory Operations
// ============================================================================

// SelectLoad generates instructions to load from memory
func (z *Z80InstructionSelector) SelectLoad(address *VirtualRegister, offset int, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		// LD A, (HL) - assumes address is in HL
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)

		// Load address into HL (with offset if needed)
		if offset != 0 {
			// Add offset to address
			vrOffset := z.vrAlloc.Allocate(16)
			z.EmitInstruction(NewZ80InstructionImm16(Z80_LD_RR_NN, vrOffset, uint16(offset)))
			z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
			z.EmitInstruction(NewZ80Instruction(Z80_ADD_HL_RR, vrHL, vrOffset))
		} else {
			z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
		}

		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_HL, vrA, vrHL))
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	case 16:
		// Load 16-bit value
		return nil, fmt.Errorf("16-bit load not yet implemented")
	}

	return result, nil
}

// SelectStore generates instructions to store to memory
func (z *Z80InstructionSelector) SelectStore(address *VirtualRegister, value *VirtualRegister, offset int, size int) error {
	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)

		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, vrA, value))

		if offset != 0 {
			vrOffset := z.vrAlloc.Allocate(16)
			z.EmitInstruction(NewZ80InstructionImm16(Z80_LD_RR_NN, vrOffset, uint16(offset)))
			z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
			z.EmitInstruction(NewZ80Instruction(Z80_ADD_HL_RR, vrHL, vrOffset))
		} else {
			z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
		}

		z.EmitInstruction(NewZ80Instruction(Z80_LD_HL_R, vrHL, vrA))
	}

	return nil
}

// SelectLoadConstant generates instructions to load an immediate value
func (z *Z80InstructionSelector) SelectLoadConstant(value interface{}, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		val := value.(int)
		z.EmitInstruction(NewZ80InstructionImm8(Z80_LD_R_N, result, uint8(val)))
	case 16:
		val := value.(int)
		z.EmitInstruction(NewZ80InstructionImm16(Z80_LD_RR_NN, result, uint16(val)))
	}

	return result, nil
}

// SelectLoadVariable generates instructions to load a variable's value
func (z *Z80InstructionSelector) SelectLoadVariable(symbol *zir.Symbol) (*VirtualRegister, error) {
	// TODO: Variable load not yet implemented
	// Decision needed: Use SP-relative addressing, HL indirection, or runtime helpers
	// IX/IY indexed addressing avoided due to instruction overhead
	return nil, fmt.Errorf("variable load not yet implemented for symbol '%s'", symbol.Name)
}

// SelectStoreVariable generates instructions to store to a variable
func (z *Z80InstructionSelector) SelectStoreVariable(symbol *zir.Symbol, value *VirtualRegister) error {
	// TODO: Variable store not yet implemented
	// Decision needed: Use SP-relative addressing, HL indirection, or runtime helpers
	// IX/IY indexed addressing avoided due to instruction overhead
	return fmt.Errorf("variable store not yet implemented for symbol '%s'", symbol.Name)
}

// SelectMove moves a value from source to target
func (z *Z80InstructionSelector) SelectMove(target *VirtualRegister, source *VirtualRegister, size int) error {
	switch size {
	case 8:
		z.EmitInstruction(NewZ80Instruction(Z80_LD_R_R, target, source))
	case 16:
		z.EmitInstruction(NewZ80Instruction(Z80_LD_RR_NN, target, source))
	}
	return nil
}

// ============================================================================
// Control Flow
// ============================================================================

// SelectBranch generates a conditional branch
func (z *Z80InstructionSelector) SelectBranch(condition *VirtualRegister, trueBlock, falseBlock *BasicBlock) error {
	// Test condition (should already set flags)
	// JP NZ, trueBlock
	// JP falseBlock
	z.EmitInstruction(NewZ80Branch(Z80_JP_CC_NN, condition, trueBlock, falseBlock))
	return nil
}

// SelectJump generates an unconditional jump
func (z *Z80InstructionSelector) SelectJump(target *BasicBlock) error {
	z.EmitInstruction(NewZ80Jump(Z80_JP_NN, target))
	return nil
}

// SelectCall generates a function call
func (z *Z80InstructionSelector) SelectCall(functionName string, args []*VirtualRegister, returnSize int) (*VirtualRegister, error) {
	// Set up arguments according to calling convention
	// For now, assume simple convention: pass in registers/stack

	z.EmitInstruction(NewZ80Call(functionName))

	// Get return value if non-void
	if returnSize > 0 {
		returnReg := z.callingConvention.GetReturnValueRegister(returnSize / 8)
		result := z.vrAlloc.AllocateConstrained(returnSize, []*Register{returnReg}, returnReg.Class)
		return result, nil
	}

	return nil, nil
}

// SelectReturn generates a return statement
func (z *Z80InstructionSelector) SelectReturn(value *VirtualRegister) error {
	// Value should already be in return register (set by caller)
	z.EmitInstruction(NewZ80Return())
	return nil
}

// ============================================================================
// Function Management
// ============================================================================

// SelectFunctionPrologue generates function entry code
func (z *Z80InstructionSelector) SelectFunctionPrologue(fn *zir.SemFunctionDecl) error {
	// Z80 function prologue typically:
	// - Save registers that need preserving
	// - Allocate stack frame if needed
	return nil
}

// SelectFunctionEpilogue generates function exit code
func (z *Z80InstructionSelector) SelectFunctionEpilogue(fn *zir.SemFunctionDecl) error {
	// Restore registers, deallocate stack frame
	return nil
}

// ============================================================================
// Utility
// ============================================================================

// AllocateVirtual creates a new virtual register
func (z *Z80InstructionSelector) AllocateVirtual(size int) *VirtualRegister {
	return z.vrAlloc.Allocate(size)
}

// AllocateVirtualConstrained creates a virtual register with constraints
func (z *Z80InstructionSelector) AllocateVirtualConstrained(size int, allowedSet []*Register, requiredClass RegisterClass) *VirtualRegister {
	return z.vrAlloc.AllocateConstrained(size, allowedSet, requiredClass)
}

// EmitInstruction adds an instruction to the sequence
func (z *Z80InstructionSelector) EmitInstruction(instr MachineInstruction) {
	z.instructions = append(z.instructions, instr)
}

// GetInstructions returns all emitted instructions
func (z *Z80InstructionSelector) GetInstructions() []MachineInstruction {
	return z.instructions
}

// ClearInstructions resets the instruction buffer
func (z *Z80InstructionSelector) ClearInstructions() {
	z.instructions = make([]MachineInstruction, 0)
}

// GetCallingConvention returns the calling convention
func (z *Z80InstructionSelector) GetCallingConvention() CallingConvention {
	return z.callingConvention
}

// GetTargetRegisters returns the set of physical registers available on Z80
func (z *Z80InstructionSelector) GetTargetRegisters() []*Register {
	return Z80Registers
}

// ============================================================================
// Z80-specific helper types
// ============================================================================

// Z80MachineInstruction represents a concrete Z80 instruction
type Z80MachineInstruction struct {
	opcode        Z80Opcode
	result        *VirtualRegister
	operands      []*VirtualRegister
	imm8          uint8
	imm16         uint16
	targetBlock   *BasicBlock
	branchTargets [2]*BasicBlock
	functionName  string
	comment       string
}

// NewZ80Instruction creates a new Z80 instruction
func NewZ80Instruction(opcode Z80Opcode, result, operand *VirtualRegister) *Z80MachineInstruction {
	operands := []*VirtualRegister{}
	if operand != nil {
		operands = append(operands, operand)
	}
	return &Z80MachineInstruction{
		opcode:   opcode,
		result:   result,
		operands: operands,
	}
}

// NewZ80InstructionImm8 creates an instruction with 8-bit immediate
func NewZ80InstructionImm8(opcode Z80Opcode, result *VirtualRegister, imm uint8) *Z80MachineInstruction {
	return &Z80MachineInstruction{
		opcode: opcode,
		result: result,
		imm8:   imm,
	}
}

// NewZ80InstructionImm16 creates an instruction with 16-bit immediate
func NewZ80InstructionImm16(opcode Z80Opcode, result *VirtualRegister, imm uint16) *Z80MachineInstruction {
	return &Z80MachineInstruction{
		opcode: opcode,
		result: result,
		imm16:  imm,
	}
}

// NewZ80Branch creates a conditional branch instruction
func NewZ80Branch(opcode Z80Opcode, condition *VirtualRegister, trueBlock, falseBlock *BasicBlock) *Z80MachineInstruction {
	return &Z80MachineInstruction{
		opcode:        opcode,
		operands:      []*VirtualRegister{condition},
		branchTargets: [2]*BasicBlock{trueBlock, falseBlock},
	}
}

// NewZ80Jump creates an unconditional jump
func NewZ80Jump(opcode Z80Opcode, target *BasicBlock) *Z80MachineInstruction {
	return &Z80MachineInstruction{
		opcode:      opcode,
		targetBlock: target,
	}
}

// NewZ80Call creates a function call
func NewZ80Call(functionName string) *Z80MachineInstruction {
	return &Z80MachineInstruction{
		opcode:       Z80_CALL_NN,
		functionName: functionName,
	}
}

// NewZ80Return creates a return instruction
func NewZ80Return() *Z80MachineInstruction {
	return &Z80MachineInstruction{
		opcode: Z80_RET,
	}
}

// Implement MachineInstruction interface

func (z *Z80MachineInstruction) GetResult() *VirtualRegister {
	return z.result
}

func (z *Z80MachineInstruction) GetOperands() []*VirtualRegister {
	return z.operands
}

func (z *Z80MachineInstruction) SetResult(vr *VirtualRegister) {
	z.result = vr
}

func (z *Z80MachineInstruction) SetOperand(index int, vr *VirtualRegister) {
	if index < len(z.operands) {
		z.operands[index] = vr
	}
}

func (z *Z80MachineInstruction) GetCategory() InstrCategory {
	// Lookup from descriptor table
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		return desc.Category
	}
	return CatOther
}

func (z *Z80MachineInstruction) GetAddressingMode() AddressingMode {
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		return desc.AddressingMode
	}
	return 0
}

func (z *Z80MachineInstruction) GetTargetBlock() *BasicBlock {
	return z.targetBlock
}

func (z *Z80MachineInstruction) GetBranchTargets() (trueBlock, falseBlock *BasicBlock) {
	if len(z.branchTargets) == 2 {
		return z.branchTargets[0], z.branchTargets[1]
	}
	return nil, nil
}

func (z *Z80MachineInstruction) GetComment() string {
	return z.comment
}

func (z *Z80MachineInstruction) String() string {
	if desc, ok := Z80InstrDescriptors[z.opcode]; ok {
		// TODO format with operands
		return fmt.Sprintf("%d", desc.Opcode)
	}

	return fmt.Sprintf("%d", z.opcode)
}

