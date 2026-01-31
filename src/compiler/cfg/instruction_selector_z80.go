package cfg

import (
	"fmt"
	"zenith/compiler/zsm"
)

// instructionSelectorZ80 implements InstructionSelector for the Z80
type instructionSelectorZ80 struct {
	vrAlloc           *VirtualRegisterAllocator
	currentBlock      *BasicBlock // Current block for instruction emission
	callingConvention CallingConvention
}

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
func (z *instructionSelectorZ80) SelectAdd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		// 8-bit add: requires A register
		// LD A, left
		// ADD A, right
		// LD result, A
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_ADD_A_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit add: ADD HL, rr
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
		z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
		z.emit(newInstructionZ80(Z80_ADD_HL_RR, vrHL, right))
		z.emit(newInstructionZ80(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for ADD: %d", size)
	}

	return result, nil
}

// SelectSubtract generates instructions for subtraction (a - b)
func (z *instructionSelectorZ80) SelectSubtract(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
	switch size {
	case 8:
		// 8-bit subtract: SUB uses A register implicitly
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_SUB_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit subtract: SBC HL, rr
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
		z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
		// Clear carry flag first (OR A)
		z.emit(newInstructionZ80(Z80_OR_R, vrA, vrA))
		z.emit(newInstructionZ80(Z80_SBC_HL_RR, vrHL, right))
		z.emit(newInstructionZ80(Z80_LD_RR_NN, result, vrHL))
	default:
		return nil, fmt.Errorf("unsupported size for SUB: %d", size)
	}

	return result, nil
}

// SelectMultiply generates instructions for multiplication (a * b)
// Z80 has no multiply instruction - call runtime helper
func (z *instructionSelectorZ80) SelectMultiply(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	// Prepare arguments according to calling convention
	// Typically: HL = left, DE = right, result in HL
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	// Call multiply runtime helper
	if size == 8 {
		z.emit(newCallZ80("__mul8"))
	} else {
		z.emit(newCallZ80("__mul16"))
	}

	// Result is in HL
	result := z.vrAlloc.Allocate(size)
	z.emit(newInstructionZ80(Z80_LD_RR_NN, result, vrHL))

	return result, nil
}

// SelectDivide generates instructions for division (a / b)
// Z80 has no divide instruction - call runtime helper
func (z *instructionSelectorZ80) SelectDivide(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(newCallZ80("__div8"))
	} else {
		z.emit(newCallZ80("__div16"))
	}

	result := z.vrAlloc.Allocate(size)
	z.emit(newInstructionZ80(Z80_LD_RR_NN, result, vrHL))

	return result, nil
}

// SelectNegate generates instructions for negation (-a)
func (z *instructionSelectorZ80) SelectNegate(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		// Two's complement: XOR 0xFF, INC
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, operand))
		z.emit(newInstructionZ80Imm8(Z80_XOR_N, vrA, 0xFF))
		z.emit(newInstructionZ80(Z80_INC_R, vrA, nil))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("unsupported size for NEGATE: %d", size)
	}

	return result, nil
}

// ============================================================================
// Bitwise Operations
// ============================================================================

// SelectBitwiseAnd generates instructions for bitwise AND (a & b)
func (z *instructionSelectorZ80) SelectBitwiseAnd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_AND_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	} else {
		// 16-bit AND: do byte-by-byte
		return nil, fmt.Errorf("16-bit AND not yet implemented")
	}

	return result, nil
}

// SelectBitwiseOr generates instructions for bitwise OR (a | b)
func (z *instructionSelectorZ80) SelectBitwiseOr(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_OR_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit OR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseXor generates instructions for bitwise XOR (a ^ b)
func (z *instructionSelectorZ80) SelectBitwiseXor(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_XOR_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit XOR not yet implemented")
	}

	return result, nil
}

// SelectBitwiseNot generates instructions for bitwise NOT (~a)
func (z *instructionSelectorZ80) SelectBitwiseNot(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	if size == 8 {
		// CPL instruction complements A
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, operand))
		z.emit(newInstructionZ80Imm8(Z80_XOR_N, vrA, 0xFF))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	} else {
		return nil, fmt.Errorf("16-bit NOT not yet implemented")
	}

	return result, nil
}

// SelectShiftLeft generates instructions for left shift (a << b)
func (z *instructionSelectorZ80) SelectShiftLeft(value, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, value))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, amount))

	if size == 8 {
		z.emit(newCallZ80("__shl8"))
	} else {
		z.emit(newCallZ80("__shl16"))
	}

	result := z.vrAlloc.AllocateConstrained(size, []*Register{&RegHL})

	return result, nil
}

// SelectShiftRight generates instructions for right shift (a >> b)
func (z *instructionSelectorZ80) SelectShiftRight(value *VirtualRegister, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, value))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, amount))

	if size == 8 {
		z.emit(newCallZ80("__shr8"))
	} else {
		z.emit(newCallZ80("__shr16"))
	}

	result := z.vrAlloc.AllocateConstrained(size, []*Register{&RegHL})

	return result, nil
}

// SelectLogicalAnd generates instructions for logical AND (a && b)
func (z *instructionSelectorZ80) SelectLogicalAnd(left, right *VirtualRegister) (*VirtualRegister, error) {
	// For logical AND, we need short-circuit evaluation which requires CFG support
	// For now, use runtime helper that evaluates both operands
	// TODO: Handle short-circuit evaluation at CFG level
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))
	z.emit(newCallZ80("__logical_and"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// SelectLogicalOr generates instructions for logical OR (a || b)
func (z *instructionSelectorZ80) SelectLogicalOr(left, right *VirtualRegister) (*VirtualRegister, error) {
	// For logical OR, we need short-circuit evaluation which requires CFG support
	// For now, use runtime helper that evaluates both operands
	// TODO: Handle short-circuit evaluation at CFG level
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))
	z.emit(newCallZ80("__logical_or"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// SelectLogicalNot generates instructions for logical NOT (!a)
func (z *instructionSelectorZ80) SelectLogicalNot(operand *VirtualRegister) (*VirtualRegister, error) {
	// Use runtime helper to convert operand != 0 to boolean, then invert
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, operand))
	z.emit(newCallZ80("__logical_not"))

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// ============================================================================
// Comparison Operations
// ============================================================================

// SelectEqual generates instructions for equality comparison (a == b)
func (z *instructionSelectorZ80) SelectEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(8) // Boolean result

	if size == 8 {
		// CP sets flags, then check Z flag
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_CP_R, vrA, right))
		// TODO: Convert flags to 0/1 value
	}

	return result, nil
}

// SelectNotEqual generates instructions for inequality comparison (a != b)
func (z *instructionSelectorZ80) SelectNotEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	// For comparison operations that return boolean, use runtime helper
	// Converting flags to 0/1 values requires control flow or special instructions
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(newCallZ80("__cmp_ne8"))
	} else {
		z.emit(newCallZ80("__cmp_ne16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// SelectLessThan generates instructions for less-than comparison (a < b)
func (z *instructionSelectorZ80) SelectLessThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(newCallZ80("__cmp_lt8"))
	} else {
		z.emit(newCallZ80("__cmp_lt16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// SelectLessEqual generates instructions for less-or-equal comparison (a <= b)
func (z *instructionSelectorZ80) SelectLessEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(newCallZ80("__cmp_le8"))
	} else {
		z.emit(newCallZ80("__cmp_le16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// SelectGreaterThan generates instructions for greater-than comparison (a > b)
func (z *instructionSelectorZ80) SelectGreaterThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(newCallZ80("__cmp_gt8"))
	} else {
		z.emit(newCallZ80("__cmp_gt16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// SelectGreaterEqual generates instructions for greater-or-equal comparison (a >= b)
func (z *instructionSelectorZ80) SelectGreaterEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})
	vrDE := z.vrAlloc.AllocateConstrained(16, []*Register{&RegDE})

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	if size == 8 {
		z.emit(newCallZ80("__cmp_ge8"))
	} else {
		z.emit(newCallZ80("__cmp_ge16"))
	}

	result := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})

	return result, nil
}

// ============================================================================
// Memory Operations
// ============================================================================

// SelectLoad generates instructions to load from memory
func (z *instructionSelectorZ80) SelectLoad(address *VirtualRegister, offset int, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		// LD A, (HL) - assumes address is in HL
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})

		// Load address into HL (with offset if needed)
		if offset != 0 {
			// Add offset to address
			vrOffset := z.vrAlloc.Allocate(16)
			z.emit(newInstructionZ80Imm16(Z80_LD_RR_NN, vrOffset, uint16(offset)))
			z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, address))
			z.emit(newInstructionZ80(Z80_ADD_HL_RR, vrHL, vrOffset))
		} else {
			z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, address))
		}

		z.emit(newInstructionZ80(Z80_LD_R_HL, vrA, vrHL))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	case 16:
		// Load 16-bit value
		return nil, fmt.Errorf("16-bit load not yet implemented")
	}

	return result, nil
}

// SelectStore generates instructions to store to memory
func (z *instructionSelectorZ80) SelectStore(address *VirtualRegister, value *VirtualRegister, offset int, size int) error {
	if size == 8 {
		vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
		vrHL := z.vrAlloc.AllocateConstrained(16, []*Register{&RegHL})

		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, value))

		if offset != 0 {
			vrOffset := z.vrAlloc.Allocate(16)
			z.emit(newInstructionZ80Imm16(Z80_LD_RR_NN, vrOffset, uint16(offset)))
			z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, address))
			z.emit(newInstructionZ80(Z80_ADD_HL_RR, vrHL, vrOffset))
		} else {
			z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, address))
		}

		z.emit(newInstructionZ80(Z80_LD_HL_R, vrHL, vrA))
	}

	return nil
}

// SelectLoadConstant generates instructions to load an immediate value
func (z *instructionSelectorZ80) SelectLoadConstant(value interface{}, size int) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(size)

	switch size {
	case 8:
		val := value.(int)
		z.emit(newInstructionZ80Imm8(Z80_LD_R_N, result, uint8(val)))
	case 16:
		val := value.(int)
		z.emit(newInstructionZ80Imm16(Z80_LD_RR_NN, result, uint16(val)))
	}

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
func (z *instructionSelectorZ80) SelectMove(target *VirtualRegister, source *VirtualRegister, size int) error {
	switch size {
	case 8:
		z.emit(newInstructionZ80(Z80_LD_R_R, target, source))
	case 16:
		z.emit(newInstructionZ80(Z80_LD_RR_NN, target, source))
	}
	return nil
}

// ============================================================================
// Control Flow
// ============================================================================

// SelectConditionalBranch evaluates a conditional expression and generates branch
// Handles comparison operations and logical operators with short-circuit evaluation
func (z *instructionSelectorZ80) SelectConditionalBranch(evaluateExpr func(zsm.SemExpression) (*VirtualRegister, error), expr zsm.SemExpression, trueBlock, falseBlock *BasicBlock) error {
	switch e := expr.(type) {
	case *zsm.SemBinaryOp:
		switch e.Op {
		case zsm.OpLogicalAnd:
			// For: a && b
			// Generate: test a, if false jump to falseBlock (short-circuit), else test b
			// Use JR for short jumps within expression
			continueLabel := fmt.Sprintf(".and_continue_%p", e)

			// Evaluate left condition with inverted logic (jump on false)
			if err := z.selectConditionalBranchInverted(evaluateExpr, e.Left, continueLabel); err != nil {
				return err
			}
			// If we get here, left was false - jump to false block
			z.emit(newJumpZ80(Z80_JP_NN, falseBlock))

			// continueLabel: left was true, evaluate right
			z.emitLabel(continueLabel)
			return z.SelectConditionalBranch(evaluateExpr, e.Right, trueBlock, falseBlock)

		case zsm.OpLogicalOr:
			// For: a || b
			// Generate: test a, if true jump to trueBlock (short-circuit), else test b
			continueLabel := fmt.Sprintf(".or_continue_%p", e)

			// Evaluate left condition normally (jump on true)
			if err := z.selectConditionalBranchDirect(evaluateExpr, e.Left, trueBlock, continueLabel); err != nil {
				return err
			}

			// continueLabel: left was false, evaluate right
			z.emitLabel(continueLabel)
			return z.SelectConditionalBranch(evaluateExpr, e.Right, trueBlock, falseBlock)

		case zsm.OpEqual:
			// Generate: CP + JP Z for equality test
			return z.selectComparisonBranch(evaluateExpr, e.Left, e.Right, Cond_Z, trueBlock, falseBlock)

		case zsm.OpNotEqual:
			// Generate: CP + JP NZ for inequality test
			return z.selectComparisonBranch(evaluateExpr, e.Left, e.Right, Cond_NZ, trueBlock, falseBlock)

		case zsm.OpLessThan:
			// Generate: CP + JP C for unsigned less-than (or JP M for signed)
			return z.selectComparisonBranch(evaluateExpr, e.Left, e.Right, Cond_C, trueBlock, falseBlock)

		case zsm.OpGreaterThan:
			// For a > b, do CP a,b and test for NOT(C OR Z)
			// Invert: jump to false if C or Z set
			return z.selectComparisonBranch(evaluateExpr, e.Left, e.Right, Cond_NC, trueBlock, falseBlock)

		case zsm.OpLessEqual:
			// For a <= b, do CP a,b and test for C OR Z
			return z.selectComparisonBranch(evaluateExpr, e.Left, e.Right, Cond_C, trueBlock, falseBlock)

		case zsm.OpGreaterEqual:
			// For a >= b, do CP a,b and test for NC
			return z.selectComparisonBranch(evaluateExpr, e.Left, e.Right, Cond_NC, trueBlock, falseBlock)
		}

	case *zsm.SemUnaryOp:
		if e.Op == zsm.OpLogicalNot {
			// For: !a, swap true and false blocks
			return z.SelectConditionalBranch(evaluateExpr, e.Operand, falseBlock, trueBlock)
		}
	}

	// Fallback: evaluate expression and test for non-zero
	vr, err := evaluateExpr(expr)
	if err != nil {
		return err
	}

	// Test if non-zero
	vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
	z.emit(newInstructionZ80(Z80_LD_R_R, vrA, vr))
	z.emit(newInstructionZ80(Z80_OR_R, vrA, vrA)) // Sets Z flag based on value
	z.emit(newBranchZ80WithCondition(Cond_NZ, trueBlock, falseBlock))

	return nil
}

// selectComparisonBranch generates a comparison and conditional branch
func (z *instructionSelectorZ80) selectComparisonBranch(evaluateExpr func(zsm.SemExpression) (*VirtualRegister, error), left, right zsm.SemExpression, condition ConditionCode, trueBlock, falseBlock *BasicBlock) error {
	leftVR, err := evaluateExpr(left)
	if err != nil {
		return err
	}

	rightVR, err := evaluateExpr(right)
	if err != nil {
		return err
	}

	// Generate CP instruction (sets flags)
	vrA := z.vrAlloc.AllocateConstrained(8, []*Register{&RegA})
	z.emit(newInstructionZ80(Z80_LD_R_R, vrA, leftVR))
	z.emit(newInstructionZ80(Z80_CP_R, vrA, rightVR))

	// Generate conditional branch based on flags
	z.emit(newBranchZ80WithCondition(condition, trueBlock, falseBlock))

	return nil
}

// Helper methods for short-circuit evaluation
func (z *instructionSelectorZ80) selectConditionalBranchInverted(evaluateExpr func(zsm.SemExpression) (*VirtualRegister, error), expr zsm.SemExpression, continueLabel string) error {
	// For now, simplified: evaluate and branch without proper label support
	// TODO: Implement proper label support for JR (relative jumps)
	// This should evaluate expr and jump to continueLabel if true, fall through if false
	// For now we'll just skip this optimization
	return nil
}

func (z *instructionSelectorZ80) selectConditionalBranchDirect(evaluateExpr func(zsm.SemExpression) (*VirtualRegister, error), expr zsm.SemExpression, trueBlock *BasicBlock, continueLabel string) error {
	// For now, simplified: evaluate and branch without proper label support
	// TODO: Implement proper label support for JR (relative jumps)
	// This should evaluate expr and jump to trueBlock if true, continueLabel if false
	// For now we'll just skip this optimization
	return nil
}

func (z *instructionSelectorZ80) emitLabel(label string) {
	// TODO: Emit a label for relative jumps
	// This requires support in the MachineInstruction representation
	// For now, this is a placeholder for future label support
}

// SelectJump generates an unconditional jump
func (z *instructionSelectorZ80) SelectJump(target *BasicBlock) error {
	z.emit(newJumpZ80(Z80_JP_NN, target))
	return nil
}

// SelectCall generates a function call
func (z *instructionSelectorZ80) SelectCall(functionName string, args []*VirtualRegister, returnSize int) (*VirtualRegister, error) {
	// Set up arguments according to calling convention
	// For now, assume simple convention: pass in registers/stack

	z.emit(newCallZ80(functionName))

	// Get return value if non-void
	if returnSize > 0 {
		returnReg := z.callingConvention.GetReturnValueRegister(returnSize / 8)
		result := z.vrAlloc.AllocateConstrained(returnSize, []*Register{returnReg})
		return result, nil
	}

	return nil, nil
}

// SelectReturn generates a return statement
func (z *instructionSelectorZ80) SelectReturn(value *VirtualRegister) error {
	// Value should already be in return register (set by caller)
	z.emit(newReturnZ80())
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

// AllocateVirtual creates a new virtual register
func (z *instructionSelectorZ80) AllocateVirtual(size int) *VirtualRegister {
	return z.vrAlloc.Allocate(size)
}

// AllocateVirtualConstrained creates a virtual register with constraints
func (z *instructionSelectorZ80) AllocateVirtualConstrained(size int, allowedSet []*Register) *VirtualRegister {
	return z.vrAlloc.AllocateConstrained(size, allowedSet)
}

// SetCurrentBlock sets the active block for instruction emission
func (z *instructionSelectorZ80) SetCurrentBlock(block *BasicBlock) {
	z.currentBlock = block
}

// EmitInstruction adds an instruction to the specified block
func (z *instructionSelectorZ80) EmitInstruction(block *BasicBlock, instr MachineInstruction) {
	block.MachineInstructions = append(block.MachineInstructions, instr)

}

// emit is a helper that emits to the current block
func (z *instructionSelectorZ80) emit(instr MachineInstruction) {
	z.EmitInstruction(z.currentBlock, instr)
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

// machineInstructionZ80 represents a concrete Z80 instruction
type machineInstructionZ80 struct {
	opcode        Z80Opcode
	result        *VirtualRegister
	operands      []*VirtualRegister
	conditionCode ConditionCode
	imm8          uint8
	imm16         uint16
	branchTargets []*BasicBlock
	functionName  string
	comment       string
}

// newInstructionZ80 creates a new Z80 instruction
func newInstructionZ80(opcode Z80Opcode, result, operand *VirtualRegister) *machineInstructionZ80 {
	operands := []*VirtualRegister{}
	if operand != nil {
		operands = append(operands, operand)
	}
	return &machineInstructionZ80{
		opcode:        opcode,
		result:        result,
		operands:      operands,
		branchTargets: make([]*BasicBlock, 0),
	}
}

// newInstructionZ80Imm8 creates an instruction with 8-bit immediate
func newInstructionZ80Imm8(opcode Z80Opcode, result *VirtualRegister, imm uint8) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode: opcode,
		result: result,
		imm8:   imm,
	}
}

// newInstructionZ80Imm16 creates an instruction with 16-bit immediate
func newInstructionZ80Imm16(opcode Z80Opcode, result *VirtualRegister, imm uint16) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode: opcode,
		result: result,
		imm16:  imm,
	}
}

// newBranchZ80 creates a conditional branch instruction
func newBranchZ80(opcode Z80Opcode, condition *VirtualRegister, trueBlock, falseBlock *BasicBlock) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        opcode,
		operands:      []*VirtualRegister{condition},
		branchTargets: []*BasicBlock{trueBlock, falseBlock},
	}
}

// newBranchZ80WithCondition creates a conditional branch with explicit condition code
func newBranchZ80WithCondition(condition ConditionCode, trueBlock, falseBlock *BasicBlock) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        Z80_JP_CC_NN,
		conditionCode: condition,
		branchTargets: []*BasicBlock{trueBlock, falseBlock},
	}
}

// newJumpZ80 creates an unconditional jump
func newJumpZ80(opcode Z80Opcode, target *BasicBlock) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        opcode,
		branchTargets: []*BasicBlock{target},
	}
}

// newCallZ80 creates a function call
func newCallZ80(functionName string) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:       Z80_CALL_NN,
		functionName: functionName,
	}
}

// newReturnZ80 creates a return instruction
func newReturnZ80() *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode: Z80_RET,
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

// nil for non-branch instructions
// [1] for jump target
// branch [1] true target, [2] false target
func (z *machineInstructionZ80) GetTargetBlocks() []*BasicBlock {
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
