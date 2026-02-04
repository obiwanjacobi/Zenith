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
var Z80RegHL = []*Register{&RegHL}
var Z80RegDE = []*Register{&RegDE}

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
			opcode = Z80_ADD_A_R
		}

		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, reg))
		z.emit(newInstructionZ80(opcode, vrA, imm))

		// for reg-alloc flexibility, move result to wider VR
		result = z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	case 16:
		// TODO: refactor to handle immediate 16-bit addition
		// 16-bit add: ADD HL, rr
		result = z.vrAlloc.Allocate(Z80Registers16)
		vrHL := z.vrAlloc.Allocate(Z80RegHL)
		z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
		z.emit(newInstructionZ80(Z80_ADD_HL_RR, vrHL, right))
		z.emit(newInstructionZ80(Z80_LD_RR_NN, result, vrHL))
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
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_SUB_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
	case 16:
		// 16-bit subtract: SBC HL, rr
		result = z.vrAlloc.Allocate(Z80Registers16)
		vrHL := z.vrAlloc.Allocate(Z80RegHL)
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
func (z *instructionSelectorZ80) SelectMultiply(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
	// Prepare arguments according to calling convention
	// Typically: HL = left, DE = right, result in HL
	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	// Call multiply runtime helper
	if size == 8 {
		z.emit(newCallZ80("__mul8"))
	} else {
		z.emit(newCallZ80("__mul16"))
	}

	// Result is always in HL (16-bit) - even 8x8 multiply produces 16-bit result
	result := z.vrAlloc.Allocate(Z80Registers16)
	z.emit(newInstructionZ80(Z80_LD_RR_NN, result, vrHL))
	return result, nil
}

// SelectDivide generates instructions for division (a / b)
// Z80 has no divide instruction - call runtime helper
func (z *instructionSelectorZ80) SelectDivide(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, left))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, right))

	var result *VirtualRegister
	if size == 8 {
		result = z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newCallZ80("__div8"))
	} else {
		result = z.vrAlloc.Allocate(Z80Registers16)
		z.emit(newCallZ80("__div16"))
	}

	z.emit(newInstructionZ80(Z80_LD_RR_NN, result, vrHL))
	return result, nil
}

// SelectNegate generates instructions for negation (-a)
func (z *instructionSelectorZ80) SelectNegate(operand *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
	var result *VirtualRegister

	if size == 8 {
		// TODO: NEG instruction?
		// Two's complement: XOR 0xFF, INC
		result = z.vrAlloc.Allocate(Z80Registers8)
		vrA := z.vrAlloc.Allocate(Z80RegA)
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
func (z *instructionSelectorZ80) SelectBitwiseAnd(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
	var result *VirtualRegister

	if size == 8 {
		result = z.vrAlloc.Allocate(Z80Registers8)
		vrA := z.vrAlloc.Allocate(Z80RegA)
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
func (z *instructionSelectorZ80) SelectBitwiseOr(left, right *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
	var result *VirtualRegister

	if size == 8 {
		result = z.vrAlloc.Allocate(Z80Registers8)
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_OR_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
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
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, left))
		z.emit(newInstructionZ80(Z80_XOR_R, vrA, right))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
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
		z.emit(newInstructionZ80(Z80_LD_R_R, vrA, operand))
		z.emit(newInstructionZ80Imm8(Z80_XOR_N, vrA, 0xFF))
		z.emit(newInstructionZ80(Z80_LD_R_R, result, vrA))
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

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, value))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, amount))

	var result *VirtualRegister
	if size == 8 {
		result = z.vrAlloc.Allocate(Z80RegA)
		z.emit(newCallZ80("__shl8"))
	} else {
		result = z.vrAlloc.Allocate(Z80RegHL)
		z.emit(newCallZ80("__shl16"))
	}

	return result, nil
}

// SelectShiftRight generates instructions for right shift (a >> b)
func (z *instructionSelectorZ80) SelectShiftRight(value *VirtualRegister, amount *VirtualRegister, size RegisterSize) (*VirtualRegister, error) {
	// For variable shifts, call runtime helper
	// Constant shifts could be optimized later
	vrHL := z.vrAlloc.Allocate(Z80RegHL)
	vrDE := z.vrAlloc.Allocate(Z80RegDE)

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, value))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, amount))

	var result *VirtualRegister
	if size == 8 {
		result = z.vrAlloc.Allocate(Z80RegA)
		z.emit(newCallZ80("__shr8"))
	} else {
		result = z.vrAlloc.Allocate(Z80RegHL)
		z.emit(newCallZ80("__shr16"))
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

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, leftVR))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, rightVR))
	z.emit(newCallZ80("__logical_and"))

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

	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, leftVR))
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrDE, rightVR))
	z.emit(newCallZ80("__logical_or"))

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
	z.emit(newInstructionZ80(Z80_LD_RR_NN, vrHL, operandVR))
	z.emit(newCallZ80("__logical_not"))

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
		z.emit(newBranchZ80WithCondition(Cond_Z, ctx.TrueBlock, ctx.FalseBlock))
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
		z.emit(newBranchZ80WithCondition(Cond_NZ, ctx.TrueBlock, ctx.FalseBlock))
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
		z.emit(newBranchZ80WithCondition(Cond_C, ctx.TrueBlock, ctx.FalseBlock))
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
		z.emit(newBranchZ80WithCondition(Cond_NC, ctx.TrueBlock, ctx.FalseBlock))
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
		z.emit(newBranchZ80WithCondition(Cond_Z, ctx.TrueBlock, nil))
		z.emit(newBranchZ80WithCondition(Cond_C, ctx.TrueBlock, ctx.FalseBlock))
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
		z.emit(newBranchZ80WithCondition(Cond_Z, ctx.TrueBlock, nil))
		z.emit(newBranchZ80WithCondition(Cond_NC, ctx.TrueBlock, ctx.FalseBlock))
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
		vrHL := z.emitLoadIntoReg16(address, &RegHL)
		z.emitAddOffsetToHL(vrHL, int32(offset))

		result = z.vrAlloc.Allocate(Z80Registers8)
		z.emit(newInstructionZ80(Z80_LD_R_HL, result, vrHL))
	case 16:
		// Load 16-bit value
		return nil, fmt.Errorf("16-bit load not yet implemented")
	}
	return result, nil
}

// SelectStore generates instructions to store to memory
func (z *instructionSelectorZ80) SelectStore(address *VirtualRegister, value *VirtualRegister, offset int, size RegisterSize) error {
	switch size {
	case 8:
		vrHL := z.emitLoadIntoReg16(address, &RegHL)
		z.emitAddOffsetToHL(vrHL, int32(offset))

		if value.Type == ImmediateValue {
			z.emit(newInstructionZ80Imm8(Z80_LD_HL_N, vrHL, uint8(value.Value)))
		} else {
			z.emit(newInstructionZ80(Z80_LD_HL_R, vrHL, value))
		}
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
func (z *instructionSelectorZ80) SelectMove(target *VirtualRegister, source *VirtualRegister, size RegisterSize) error {
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

// SelectBranch evaluates a conditional expression and generates branch
// Handles comparison operations and logical operators with short-circuit evaluation
func (z *instructionSelectorZ80) SelectBranch(evaluateExpr func(zsm.SemExpression) (*VirtualRegister, error), expr zsm.SemExpression, trueBlock, falseBlock *BasicBlock) error {
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
			return z.SelectBranch(evaluateExpr, e.Right, trueBlock, falseBlock)

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
			return z.SelectBranch(evaluateExpr, e.Right, trueBlock, falseBlock)

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
			return z.SelectBranch(evaluateExpr, e.Operand, falseBlock, trueBlock)
		}
	}

	// Fallback: evaluate expression and test for non-zero
	vr, err := evaluateExpr(expr)
	if err != nil {
		return err
	}

	// Test if non-zero
	vrA := z.vrAlloc.Allocate(Z80RegA)
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
	vrA := z.vrAlloc.Allocate(Z80RegA)
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
func (z *instructionSelectorZ80) SelectCall(functionName string, args []*VirtualRegister, returnSize RegisterSize) (*VirtualRegister, error) {
	// Set up arguments according to calling convention
	// For now, assume simple convention: pass in registers/stack

	z.emit(newCallZ80(functionName))

	// Get return value if non-void
	if returnSize > 0 {
		returnReg := z.callingConvention.GetReturnValueRegister(returnSize)
		result := z.vrAlloc.Allocate([]*Register{returnReg})
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

// emitLoadIntoReg16 loads a 16-bit value (register or immediate) into the target register
func (z *instructionSelectorZ80) emitLoadIntoReg16(value *VirtualRegister, targetReg *Register) *VirtualRegister {
	if targetReg.Size != 16 {
		return nil // Target register must be 16-bit
	}

	var vrTarget *VirtualRegister
	if !value.IsRegister(targetReg) {
		vrTarget = z.vrAlloc.Allocate([]*Register{targetReg})
		if value.Type == ImmediateValue {
			// Load immediate value into targetReg
			z.emit(newInstructionZ80Imm16(Z80_LD_RR_NN, vrTarget, uint16(value.Value)))
		} else if len(value.AllowedSet) == 1 {
			// extract the low and hi value registers
			regValueLo, regValueHi := value.AllowedSet[0].AsPairs()
			regTargetLo, regTargetHi := targetReg.AsPairs()

			// LD targetReg[Lo], value[Lo]
			vrTargetLo := z.vrAlloc.Allocate([]*Register{regTargetLo})
			vrValueLo := z.vrAlloc.Allocate([]*Register{regValueLo})
			z.emit(newInstructionZ80(Z80_LD_R_R, vrTargetLo, vrValueLo))

			// LD targetReg[Hi], value[Hi]
			vrTargetHi := z.vrAlloc.Allocate([]*Register{regTargetHi})
			if regValueHi != nil {
				vrValueHi := z.vrAlloc.Allocate([]*Register{regValueHi})
				z.emit(newInstructionZ80(Z80_LD_R_R, vrTargetHi, vrValueHi))
			} else {
				// reset high byte to 0 - not used
				z.emit(newInstructionZ80Imm8(Z80_LD_R_N, vrTargetHi, 0))
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
		z.emit(newInstructionZ80(Z80_LD_RR_NN, vrOffsetReg, vrOffset))
		z.emit(newInstructionZ80(Z80_ADD_HL_RR, vrHL, vrOffsetReg))
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
		z.emit(newInstructionZ80(opcode, vrA, left))

		if right.Type == ImmediateValue {
			opcode = Z80_CP_N
		} else {
			opcode = Z80_CP_R
		}
		z.emit(newInstructionZ80(opcode, vrA, right))
		return vrA, nil
	case 16:
		// ld hl, reg
		vrHL := z.emitLoadIntoReg16(left, &RegHL)
		// ld de, imm
		vrDE := z.emitLoadIntoReg16(right, &RegDE)

		// or a(, a) - clears carry flag
		vrA := z.vrAlloc.Allocate(Z80RegA)
		z.emit(newInstructionZ80(Z80_OR_R, vrA, nil))
		// sbc hl, de
		z.emit(newInstructionZ80(Z80_SBC_HL_RR, vrHL, vrDE))
		// add hl, de
		z.emit(newInstructionZ80(Z80_ADD_HL_RR, vrHL, vrDE))
		// c and z flags set accordingly
		return vrHL, nil
	default:
		return nil, fmt.Errorf("unsupported size for COMPARE: %d", regSize)
	}
}

// emitFlagToRegA converts a CPU flag to a boolean in register A (0 or 1)
func (z *instructionSelectorZ80) emitFlagToRegA(conditionCode ConditionCode) (*VirtualRegister, error) {
	result := z.vrAlloc.Allocate(Z80RegA)

	// do not use 'xor a' here, as it clears flags
	switch conditionCode {
	case Cond_Z, Cond_NZ:
		z.emit(newInstructionZ80Imm8(Z80_LD_R_N, result, 0))
		z.emit(newBranchRelativeZ80(conditionCode, 1)) // 1: jump over next instruction
		z.emit(newInstructionZ80(Z80_INC_R, result, nil))
	case Cond_C:
		z.emit(newInstructionZ80Imm8(Z80_LD_R_N, result, 0))
		z.emit(newInstructionZ80Imm8(Z80_ADC_A_N, result, 0))
	case Cond_NC:
		z.emit(newInstructionZ80(Z80_SBC_A_R, result, nil))
		z.emit(newInstructionZ80(Z80_INC_R, result, nil))
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
	if (*right).Type == ImmediateValue && (left).Type != ImmediateValue {
		return right, left, true
	} else if (left).Type == ImmediateValue && (right).Type != ImmediateValue {
		return left, right, true
	} else if (left).Type == ImmediateValue && (right).Type == ImmediateValue {
		// error: should have been constant folded earlier
		return nil, nil, false
	}
	return left, right, false
}

// machineInstructionZ80 represents a concrete Z80 instruction
type machineInstructionZ80 struct {
	opcode         Z80Opcode
	result         *VirtualRegister
	operands       []*VirtualRegister
	conditionCode  ConditionCode
	immediateValue uint16
	displacement   int8
	branchTargets  []*BasicBlock
	functionName   string
	comment        string
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
		opcode:         opcode,
		result:         result,
		immediateValue: uint16(imm),
	}
}

// newInstructionZ80Imm16 creates an instruction with 16-bit immediate
func newInstructionZ80Imm16(opcode Z80Opcode, result *VirtualRegister, imm uint16) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:         opcode,
		result:         result,
		immediateValue: imm,
	}
}

// newBranchRelativeZ80 is used when no basic block is needed (e.g., JR)
// displacement is relative offset of machine instructions
func newBranchRelativeZ80(condition ConditionCode, displacement int8) *machineInstructionZ80 {
	return &machineInstructionZ80{
		opcode:        Z80_JR_CC_E,
		conditionCode: condition,
		displacement:  displacement,
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

func (z *machineInstructionZ80) String() string {
	opName := z.opcode.String()

	// Handle different instruction formats
	switch {
	case z.opcode == Z80_CALL_NN && z.functionName != "":
		return fmt.Sprintf("CALL %s", z.functionName)

	case z.opcode == Z80_RET:
		return "RET"

	case z.opcode == Z80_JP_CC_NN:
		condName := z.conditionCode.String()
		if len(z.branchTargets) > 0 {
			return fmt.Sprintf("JP %s, L%d", condName, z.branchTargets[0].ID)
		}
		return fmt.Sprintf("JP %s, ???", condName)

	case len(z.branchTargets) > 0:
		// Branch instruction
		if len(z.branchTargets) == 1 {
			return fmt.Sprintf("%s L%d", opName, z.branchTargets[0].ID)
		} else if len(z.branchTargets) == 2 {
			return fmt.Sprintf("%s L%d, L%d", opName, z.branchTargets[0].ID, z.branchTargets[1].ID)
		}

	case z.immediateValue != 0:
		// Immediate value instruction
		if z.result != nil {
			return fmt.Sprintf("%s %s, %d", opName, z.result.String(), z.immediateValue)
		}
		return fmt.Sprintf("%s %d", opName, z.immediateValue)

	case z.result != nil && len(z.operands) > 0:
		// Result and operands
		operandStrs := make([]string, len(z.operands))
		for i, op := range z.operands {
			operandStrs[i] = op.String()
		}
		return fmt.Sprintf("%s %s, %s", opName, z.result.String(), strings.Join(operandStrs, ", "))

	case z.result != nil:
		// Result only
		return fmt.Sprintf("%s %s", opName, z.result.String())
	case len(z.operands) > 0:
		// Operands only
		operandStrs := make([]string, len(z.operands))
		for i, op := range z.operands {
			operandStrs[i] = op.String()
		}
		return fmt.Sprintf("%s %s", opName, strings.Join(operandStrs, ", "))
	}

	return opName
}
