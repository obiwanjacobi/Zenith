package cfg

import (
	"fmt"
	"zenith/compiler/zsm"
)

// InstructionSelectorZ80 implements InstructionSelector for the Z80
type InstructionSelectorZ80 struct {
	vrAlloc           *VirtualRegisterAllocator
	instructions      []MachineInstruction // Deprecated: kept for backward compatibility
	currentBlock      *BasicBlock          // Current block for instruction emission
	callingConvention CallingConvention
	labelCounter      int
}

// NewInstructionSelectorZ80 creates a new Z80 instruction selector
func NewInstructionSelectorZ80(cc CallingConvention) *InstructionSelectorZ80 {
	return &InstructionSelectorZ80{
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
func (z *InstructionSelectorZ80) SelectAdd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		// 8-bit add: requires A register
		// LD A, left
		// ADD A, right
		// LD result, A
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.emit(NewZ80Instruction(Z80_ADD_A_R, vrA, right))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit add: ADD HL, rr
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
		z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
		z.emit(NewZ80Instruction(Z80_ADD_HL_RR, vrHL, right))
		z.emit(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for ADD: %d", size)
	}

	return result, nil
}

// SelectSubtract generates instructions for subtraction (a - b)
func (z *InstructionSelectorZ80) SelectSubtract(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
	switch size {
	case 8:
		// 8-bit subtract: SUB uses A register implicitly
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.emit(NewZ80Instruction(Z80_SUB_R, vrA, right))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit subtract: SBC HL, rr
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
		z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
		// Clear carry flag first (OR A)
		z.emit(NewZ80Instruction(Z80_OR_R, vrA, vrA))
		z.emit(NewZ80Instruction(Z80_SBC_HL_RR, vrHL, right))
		z.emit(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for SUB: %d", size)
	}

	return result, nil
}

// SelectMultiply generates instructions for multiplication (a * b)
// Z80 has no multiply instruction - call runtime helper
func (z *InstructionSelectorZ80) SelectMultiply(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	// Prepare arguments according to calling convention
	// Typically: HL = left, DE = right, result in HL
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	// Call multiply runtime helper
	if size == 8 {
		z.emit(NewZ80Call("__mul8"))
	} else {
		z.emit(NewZ80Call("__mul16"))
	}

	// Result is in HL
	result := z.vrAlloc.Allocate(size)
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))

	return result, nil
}

// SelectDivide generates instructions for division (a / b)
// Z80 has no divide instruction - call runtime helper
func (z *InstructionSelectorZ80) SelectDivide(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(NewZ80Call("__div8"))
	} else {
		z.emit(NewZ80Call("__div16"))
	}

	result := z.vrAlloc.Allocate(size)
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, result, vrHL))

	return result, nil
}

// SelectNegate generates instructions for negation (-a)
func (z *InstructionSelectorZ80) SelectNegate(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		// Two's complement: XOR 0xFF, INC
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, operand))
		z.emit(NewZ80InstructionImm8(Z80_XOR_N, vrA, 0xFF))
		z.emit(NewZ80Instruction(Z80_INC_R, vrA, nil))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("unsupported size for NEGATE: %d", size)
	}

	return result, nil
}

// ============================================================================
// Bitwise Operations
// ============================================================================

// SelectBitwiseAnd generates instructions for bitwise AND (a & b)
func (z *InstructionSelectorZ80) SelectBitwiseAnd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.emit(NewZ80Instruction(Z80_AND_R, vrA, right))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		// 16-bit AND: do byte-by-byte
		return nil, fmt.Errorf("16-bit AND not yet implemented")
	}

	return result, nil
}

// SelectBitwiseOr generates instructions for bitwise OR (a | b)
func (z *InstructionSelectorZ80) SelectBitwiseOr(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.emit(NewZ80Instruction(Z80_OR_R, vrA, right))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit OR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseXor generates instructions for bitwise XOR (a ^ b)
func (z *InstructionSelectorZ80) SelectBitwiseXor(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.emit(NewZ80Instruction(Z80_XOR_R, vrA, right))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit XOR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseNot generates instructions for bitwise NOT (~a)
func (z *InstructionSelectorZ80) SelectBitwiseNot(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		// CPL instruction complements A
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, operand))
		z.emit(NewZ80InstructionImm8(Z80_XOR_N, vrA, 0xFF))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit NOT not yet implemented")
	}

	return result, nil
}

// SelectShiftLeft generates instructions for left shift (a << b)
func (z *InstructionSelectorZ80) SelectShiftLeft(value, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, value))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, amount))

	if size == 8 {
		z.emit(NewZ80Call("__shl8"))
	} else {
		z.emit(NewZ80Call("__shl16"))
	}

	result := z.vrAlloc.AllocateConstrained(size, []*Register{&RegHL}, RegisterClassIndex)

	return result, nil
}

// SelectShiftRight generates instructions for right shift (a >> b)
func (z *InstructionSelectorZ80) SelectShiftRight(value *VirtualRegister, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, value))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, amount))

	if size == 8 {
		z.emit(NewZ80Call("__shr8"))
	} else {
		z.emit(NewZ80Call("__shr16"))
	}

	result := z.vrAlloc.AllocateConstrained(size, []*Register{&RegHL}, RegisterClassIndex)

	return result, nil
}

// SelectLogicalAnd generates instructions for logical AND (a && b)
func (z *InstructionSelectorZ80) SelectLogicalAnd(left, right *VirtualRegister) (*VirtualRegister, error) {
	// For logical AND, we need short-circuit evaluation which requires CFG support
	// For now, use runtime helper that evaluates both operands
	// TODO: Handle short-circuit evaluation at CFG level
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))
	z.emit(NewZ80Call("__logical_and"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLogicalOr generates instructions for logical OR (a || b)
func (z *InstructionSelectorZ80) SelectLogicalOr(left, right *VirtualRegister) (*VirtualRegister, error) {
	// For logical OR, we need short-circuit evaluation which requires CFG support
	// For now, use runtime helper that evaluates both operands
	// TODO: Handle short-circuit evaluation at CFG level
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))
	z.emit(NewZ80Call("__logical_or"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLogicalNot generates instructions for logical NOT (!a)
func (z *InstructionSelectorZ80) SelectLogicalNot(operand *VirtualRegister) (*VirtualRegister, error) {
	// Use runtime helper to convert operand != 0 to boolean, then invert
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, operand))
	z.emit(NewZ80Call("__logical_not"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// ============================================================================
// Comparison Operations
// ============================================================================

// SelectEqual generates instructions for equality comparison (a == b)
func (z *InstructionSelectorZ80) SelectEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(8) // Boolean result

	if size == 8 {
		// CP sets flags, then check Z flag
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, left))
		z.emit(NewZ80Instruction(Z80_CP_R, vrA, right))
		// TODO: Convert flags to 0/1 value
	}

	return result, nil
}

// SelectNotEqual generates instructions for inequality comparison (a != b)
func (z *InstructionSelectorZ80) SelectNotEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	// For comparison operations that return boolean, use runtime helper
	// Converting flags to 0/1 values requires control flow or special instructions
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(NewZ80Call("__cmp_ne8"))
	} else {
		z.emit(NewZ80Call("__cmp_ne16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLessThan generates instructions for less-than comparison (a < b)
func (z *InstructionSelectorZ80) SelectLessThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(NewZ80Call("__cmp_lt8"))
	} else {
		z.emit(NewZ80Call("__cmp_lt16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectLessEqual generates instructions for less-or-equal comparison (a <= b)
func (z *InstructionSelectorZ80) SelectLessEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(NewZ80Call("__cmp_le8"))
	} else {
		z.emit(NewZ80Call("__cmp_le16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectGreaterThan generates instructions for greater-than comparison (a > b)
func (z *InstructionSelectorZ80) SelectGreaterThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(NewZ80Call("__cmp_gt8"))
	} else {
		z.emit(NewZ80Call("__cmp_gt16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// SelectGreaterEqual generates instructions for greater-or-equal comparison (a >= b)
func (z *InstructionSelectorZ80) SelectGreaterEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE}, RegisterClassIndex)

	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, left))
	z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(NewZ80Call("__cmp_ge8"))
	} else {
		z.emit(NewZ80Call("__cmp_ge16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)

	return result, nil
}

// ============================================================================
// Memory Operations
// ============================================================================

// SelectLoad generates instructions to load from memory
func (z *InstructionSelectorZ80) SelectLoad(address *VirtualRegister, offset int, size int) (*VirtualRegister, error) {
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
			z.emit(NewZ80InstructionImm16(Z80_LD_RR_NN, vrOffset, uint16(offset)))
			z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
			z.emit(NewZ80Instruction(Z80_ADD_HL_RR, vrHL, vrOffset))
		} else {
			z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
		}

		z.emit(NewZ80Instruction(Z80_LD_R_HL, vrA, vrHL))
		z.emit(NewZ80Instruction(Z80_LD_R_R, result, vrA))
	case 16:
		// Load 16-bit value
		return nil, fmt.Errorf("16-bit load not yet implemented")
	}

	return result, nil
}

// SelectStore generates instructions to store to memory
func (z *InstructionSelectorZ80) SelectStore(address *VirtualRegister, value *VirtualRegister, offset int, size int) error {
	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA}, RegisterClassAccumulator)
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL}, RegisterClassIndex)

		z.emit(NewZ80Instruction(Z80_LD_R_R, vrA, value))

		if offset != 0 {
			vrOffset := z.vrAlloc.Allocate(16)
			z.emit(NewZ80InstructionImm16(Z80_LD_RR_NN, vrOffset, uint16(offset)))
			z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
			z.emit(NewZ80Instruction(Z80_ADD_HL_RR, vrHL, vrOffset))
		} else {
			z.emit(NewZ80Instruction(Z80_LD_RR_NN, vrHL, address))
		}

		z.emit(NewZ80Instruction(Z80_LD_HL_R, vrHL, vrA))
	}

	return nil
}

// SelectLoadConstant generates instructions to load an immediate value
func (z *InstructionSelectorZ80) SelectLoadConstant(value interface{}, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		val := value.(int)
		z.emit(NewZ80InstructionImm8(Z80_LD_R_N, result, uint8(val)))
	case 16:
		val := value.(int)
		z.emit(NewZ80InstructionImm16(Z80_LD_RR_NN, result, uint16(val)))
	}

	return result, nil
}

// SelectLoadVariable generates instructions to load a variable's value
func (z *InstructionSelectorZ80) SelectLoadVariable(symbol *zsm.Symbol) (*VirtualRegister, error) {
	// TODO: Variable load not yet implemented
	// Decision needed: Use SP-relative addressing, HL indirection, or runtime helpers
	// IX/IY indexed addressing avoided due to instruction overhead
	return nil, fmt.Errorf("variable load not yet implemented for symbol '%s'", symbol.Name)
}

// SelectStoreVariable generates instructions to store to a variable
func (z *InstructionSelectorZ80) SelectStoreVariable(symbol *zsm.Symbol, value *VirtualRegister) error {
	// TODO: Variable store not yet implemented
	// Decision needed: Use SP-relative addressing, HL indirection, or runtime helpers
	// IX/IY indexed addressing avoided due to instruction overhead
	return fmt.Errorf("variable store not yet implemented for symbol '%s'", symbol.Name)
}

// SelectMove moves a value from source to target
func (z *InstructionSelectorZ80) SelectMove(target *VirtualRegister, source *VirtualRegister, size int) error {
	switch size {
	case 8:
		z.emit(NewZ80Instruction(Z80_LD_R_R, target, source))
	case 16:
		z.emit(NewZ80Instruction(Z80_LD_RR_NN, target, source))
	}
	return nil
}

// ============================================================================
// Control Flow
// ============================================================================

// SelectBranch generates a conditional branch
func (z *InstructionSelectorZ80) SelectBranch(condition *VirtualRegister, trueBlock, falseBlock *BasicBlock) error {
	// Test condition (should already set flags)
	// JP NZ, trueBlock
	// JP falseBlock
	z.emit(NewZ80Branch(Z80_JP_CC_NN, condition, trueBlock, falseBlock))
	return nil
}

// SelectJump generates an unconditional jump
func (z *InstructionSelectorZ80) SelectJump(target *BasicBlock) error {
	z.emit(NewZ80Jump(Z80_JP_NN, target))
	return nil
}

// SelectCall generates a function call
func (z *InstructionSelectorZ80) SelectCall(functionName string, args []*VirtualRegister, returnSize int) (*VirtualRegister, error) {
	// Set up arguments according to calling convention
	// For now, assume simple convention: pass in registers/stack

	z.emit(NewZ80Call(functionName))

	// Get return value if non-void
	if returnSize > 0 {
		returnReg := z.callingConvention.GetReturnValueRegister(returnSize / 8)
		result := z.vrAlloc.AllocateConstrained(returnSize, []*Register{returnReg}, returnReg.Class)
		return result, nil
	}

	return nil, nil
}

// SelectReturn generates a return statement
func (z *InstructionSelectorZ80) SelectReturn(value *VirtualRegister) error {
	// Value should already be in return register (set by caller)
	z.emit(NewZ80Return())
	return nil
}

// ============================================================================
// Function Management
// ============================================================================

// SelectFunctionPrologue generates function entry code
func (z *InstructionSelectorZ80) SelectFunctionPrologue(fn *zsm.SemFunctionDecl) error {
	// Z80 function prologue typically:
	// - Save registers that need preserving
	// - Allocate stack frame if needed
	return nil
}

// SelectFunctionEpilogue generates function exit code
func (z *InstructionSelectorZ80) SelectFunctionEpilogue(fn *zsm.SemFunctionDecl) error {
	// Restore registers, deallocate stack frame
	return nil
}

// ============================================================================
// Utility
// ============================================================================

// AllocateVirtual creates a new virtual register
func (z *InstructionSelectorZ80) AllocateVirtual(size int) *VirtualRegister {
	return z.vrAlloc.Allocate(size)
}

// AllocateVirtualConstrained creates a virtual register with constraints
func (z *InstructionSelectorZ80) AllocateVirtualConstrained(size int, allowedSet []*Register, requiredClass RegisterClass) *VirtualRegister {
	return z.vrAlloc.AllocateConstrained(size, allowedSet, requiredClass)
}

// SetCurrentBlock sets the active block for instruction emission
func (z *InstructionSelectorZ80) SetCurrentBlock(block *BasicBlock) {
	z.currentBlock = block
}

// EmitInstruction adds an instruction to the specified block
func (z *InstructionSelectorZ80) EmitInstruction(block *BasicBlock, instr MachineInstruction) {
	if block != nil {
		block.MachineInstructions = append(block.MachineInstructions, instr)
	}
	// Also add to flat list for backward compatibility
	z.instructions = append(z.instructions, instr)
}

// emit is a helper that emits to the current block
func (z *InstructionSelectorZ80) emit(instr MachineInstruction) {
	z.EmitInstruction(z.currentBlock, instr)
}

// GetInstructions returns all emitted instructions (backward compatibility)
func (z *InstructionSelectorZ80) GetInstructions() []MachineInstruction {
	return z.instructions
}

// ClearInstructions resets the instruction buffer
func (z *InstructionSelectorZ80) ClearInstructions() {
	z.instructions = make([]MachineInstruction, 0)
}

// GetCallingConvention returns the calling convention
func (z *InstructionSelectorZ80) GetCallingConvention() CallingConvention {
	return z.callingConvention
}

// GetTargetRegisters returns the set of physical registers available on Z80
func (z *InstructionSelectorZ80) GetTargetRegisters() []*Register {
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

func (z *Z80MachineInstruction) GetTargetBlocks() []*BasicBlock {
	// Unconditional jump/call
	if z.targetBlock != nil {
		return []*BasicBlock{z.targetBlock}
	}

	// Conditional branch (true/false)
	if z.branchTargets[0] != nil || z.branchTargets[1] != nil {
		return z.branchTargets[:]
	}

	// No control flow
	return nil
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
