package cfg

import (
	"testing"
	"zenith/compiler/zsm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func u8Type() zsm.Type {
	return zsm.U8Type
}

func u16Type() zsm.Type {
	return zsm.U16Type
}

func newTestBlock() *BasicBlock {
	return &BasicBlock{
		ID:                  0,
		Label:               LabelEntry,
		Instructions:        []zsm.SemStatement{},
		MachineInstructions: []MachineInstruction{},
		Successors:          []*BasicBlock{},
		Predecessors:        []*BasicBlock{},
	}
}

// ============================================================================
// Test Helpers
// ============================================================================

// testContext holds common test setup
type testContext struct {
	block    *BasicBlock
	vrAlloc  *VirtualRegisterAllocator
	selector InstructionSelector
	ctx      *InstructionSelectionContext
}

// newTestContext creates a new test context with all necessary components initialized
func newTestContext() *testContext {
	symbolContext := make(map[string]*VirtualRegisterType)
	block := newTestBlock()
	vrAlloc := NewVirtualRegisterAllocator()
	selector := NewInstructionSelectorZ80(vrAlloc, symbolContext)
	selector.SetCurrentBlock(block)

	ctx := NewInstructionSelectionContext(selector, vrAlloc)
	ctx.currentBlock = block
	ctx.currentCFG = &CFG{StackFrame: NewStackFrame()}

	return &testContext{
		block:    block,
		vrAlloc:  vrAlloc,
		selector: selector,
		ctx:      ctx,
	}
}

// getInstructions returns the generated machine instructions
func (tc *testContext) getInstructions() []MachineInstruction {
	return tc.block.MachineInstructions
}

// getZ80Instructions returns instructions cast to Z80 type for assertion
func (tc *testContext) getZ80Instructions() []*machineInstructionZ80 {
	instructions := tc.getInstructions()
	z80Instructions := make([]*machineInstructionZ80, 0, len(instructions))
	for _, instr := range instructions {
		if z80Instr, ok := instr.(*machineInstructionZ80); ok {
			z80Instructions = append(z80Instructions, z80Instr)
		}
	}
	return z80Instructions
}

// Instruction assertion helpers

// assertHasOpcode checks that at least one instruction has the given opcode
func assertHasOpcode(t *testing.T, instructions []*machineInstructionZ80, opcode Z80Opcode, msgAndArgs ...interface{}) bool {
	for _, instr := range instructions {
		if instr.opcode == opcode {
			return true
		}
	}
	args := append([]interface{}{"Opcode %v not found in instructions", opcode}, msgAndArgs...)
	return assert.Fail(t, "Expected opcode not found", args...)
}

// countOpcode returns the number of instructions with the given opcode
func countOpcode(instructions []*machineInstructionZ80, opcode Z80Opcode) int {
	count := 0
	for _, instr := range instructions {
		if instr.opcode == opcode {
			count++
		}
	}
	return count
}

// assertOpcodeSequence checks that opcodes appear in order (not necessarily consecutive)
func assertOpcodeSequence(t *testing.T, instructions []*machineInstructionZ80, opcodes []Z80Opcode, msgAndArgs ...interface{}) bool {
	if len(opcodes) == 0 {
		return true
	}

	opcodeIdx := 0
	for _, instr := range instructions {
		if instr.opcode == opcodes[opcodeIdx] {
			opcodeIdx++
			if opcodeIdx >= len(opcodes) {
				return true
			}
		}
	}

	args := append([]interface{}{"Expected sequence %v, found only %d matches in %d instructions", opcodes, opcodeIdx, len(instructions)}, msgAndArgs...)
	return assert.Fail(t, "Opcode sequence not found", args...)
}

// assertInstructionCount checks the exact number of instructions generated
func assertInstructionCount(t *testing.T, instructions []*machineInstructionZ80, expected int, msgAndArgs ...interface{}) bool {
	return assert.Equal(t, expected, len(instructions), msgAndArgs...)
}

// dumpInstructions prints all instructions (useful for debugging)
func dumpInstructions(t *testing.T, instructions []*machineInstructionZ80) {
	t.Helper()
	t.Log("Generated instructions:")
	for i, instr := range instructions {
		t.Logf("  [%d] Opcode=0x%04X (%v) %s", i, instr.opcode, instr.opcode, instr.String())
	}
}

// ============================================================================
// Expression Tests
// ============================================================================

func TestZ80_Constant_U8(t *testing.T) {
	tc := newTestContext()

	constant := &zsm.SemConstant{
		Value:    42,
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectConstant(constant)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, uint8(8), vr.Size)
	assert.Equal(t, int32(42), vr.Value)

	// Constants should not generate instructions (just stored in VR)
	instructions := tc.getZ80Instructions()
	assert.Empty(t, instructions)
}

func TestZ80_Constant_U16(t *testing.T) {
	tc := newTestContext()

	constant := &zsm.SemConstant{
		Value:    1234,
		TypeInfo: u16Type(),
	}

	vr, err := tc.ctx.selectConstant(constant)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, uint8(16), vr.Size)
	assert.Equal(t, int32(1234), vr.Value)

	instructions := tc.getZ80Instructions()
	assert.Empty(t, instructions)
}

// ============================================================================
// Binary Operations - Arithmetic
// ============================================================================

func TestZ80_BinaryOp_Add_U8(t *testing.T) {
	tc := newTestContext()

	left := &zsm.SemConstant{Value: 10, TypeInfo: u8Type()}
	right := &zsm.SemConstant{Value: 20, TypeInfo: u8Type()}

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpAdd,
		Left:     left,
		Right:    right,
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, uint8(8), vr.Size)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// With constant operands: load first constant, then ADD with immediate
	// We just verify that an ADD instruction exists
	hasAdd := countOpcode(instructions, Z80_ADD_A_R) > 0 || countOpcode(instructions, Z80_ADD_A_N) > 0
	assert.True(t, hasAdd, "Should have ADD instruction (either ADD_A_R or ADD_A_N)")
}

func TestZ80_BinaryOp_Add_U16(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpAdd,
		Left:     &zsm.SemConstant{Value: 1000, TypeInfo: u16Type()},
		Right:    &zsm.SemConstant{Value: 2000, TypeInfo: u16Type()},
		TypeInfo: u16Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, uint8(16), vr.Size)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// 16-bit addition uses ADD HL, rr
	// TODO: Verify exact instruction sequence for 16-bit ADD
	assertHasOpcode(t, instructions, Z80_ADD_HL_RR, "Should have ADD_HL_RR instruction")
}

func TestZ80_BinaryOp_Subtract_U8(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpSubtract,
		Left:     &zsm.SemConstant{Value: 30, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate SUB instruction
	assertHasOpcode(t, instructions, Z80_SUB_R, "Should have SUB_R instruction")
}

func TestZ80_BinaryOp_Subtract_U16(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpSubtract,
		Left:     &zsm.SemConstant{Value: 3000, TypeInfo: u16Type()},
		Right:    &zsm.SemConstant{Value: 1000, TypeInfo: u16Type()},
		TypeInfo: u16Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, uint8(16), vr.Size)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// 16-bit subtraction uses SBC HL, rr
	// TODO: Verify exact instruction sequence for 16-bit SUB
	assertHasOpcode(t, instructions, Z80_SBC_HL_RR, "Should have SBC_HL_RR instruction")
}

func TestZ80_BinaryOp_Multiply_U8(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpMultiply,
		Left:     &zsm.SemConstant{Value: 6, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 7, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	// 8x8 multiply returns 16-bit result
	assert.Equal(t, uint8(16), vr.Size)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Z80 doesn't have native multiply, so this will be a library call or loop
	// TODO: Verify multiplication implementation strategy
	// assertHasOpcode(t, instructions, Z80_CALL_NN, "Multiply likely implemented as function call")
}

func TestZ80_BinaryOp_Divide_U8(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpDivide,
		Left:     &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 6, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Z80 doesn't have native divide, so this will be a library call or loop
	assertHasOpcode(t, instructions, Z80_CALL_NN, "Divide likely implemented as function call")
}

// ============================================================================
// Binary Operations - Bitwise
// ============================================================================

func TestZ80_BinaryOp_BitwiseAnd(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpBitwiseAnd,
		Left:     &zsm.SemConstant{Value: 0xFF, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 0x0F, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate AND instruction
	assertHasOpcode(t, instructions, Z80_AND_R, "Should have AND_R instruction")
}

func TestZ80_BinaryOp_BitwiseOr(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpBitwiseOr,
		Left:     &zsm.SemConstant{Value: 0xF0, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 0x0F, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate OR instruction
	assertHasOpcode(t, instructions, Z80_OR_R, "Should have OR_R instruction")
}

func TestZ80_BinaryOp_BitwiseXor(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpBitwiseXor,
		Left:     &zsm.SemConstant{Value: 0xFF, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 0xAA, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectBinaryOp(nil, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate XOR instruction
	assertHasOpcode(t, instructions, Z80_XOR_R, "Should have XOR_R instruction")
}

// ============================================================================
// Binary Operations - Comparison
// ============================================================================

func TestZ80_BinaryOp_Equal(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpEqual,
		Left:     &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	// Comparison operators need branch context
	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate CP (compare) followed by conditional jump
	assertHasOpcode(t, instructions, Z80_CP_N, "Should have CP_N instruction")
	assertHasOpcode(t, instructions, Z80_JP_CC_NN, "Should have conditional jump instruction")
}

func TestZ80_BinaryOp_NotEqual(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpNotEqual,
		Left:     &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate CP with immediate and conditional jump
	assertHasOpcode(t, instructions, Z80_CP_N, "Should have CP_N instruction")
	assertHasOpcode(t, instructions, Z80_JP_CC_NN, "Should have conditional jump instruction")
}

func TestZ80_BinaryOp_LessThan(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpLessThan,
		Left:     &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate CP with immediate and conditional jump (JP C for carry flag)
	assertHasOpcode(t, instructions, Z80_CP_N, "Should have CP_N instruction")
	assertHasOpcode(t, instructions, Z80_JP_CC_NN, "Should have conditional jump instruction")
}

func TestZ80_BinaryOp_LessEqual(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpLessEqual,
		Left:     &zsm.SemConstant{Value: 15, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate CP with immediate and conditional jump
	assertHasOpcode(t, instructions, Z80_CP_N, "Should have CP_N instruction")
	assertHasOpcode(t, instructions, Z80_JP_CC_NN, "Should have conditional jump instruction")
}

func TestZ80_BinaryOp_GreaterThan(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpGreaterThan,
		Left:     &zsm.SemConstant{Value: 30, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate CP with immediate and conditional jump
	assertHasOpcode(t, instructions, Z80_CP_N, "Should have CP_N instruction")
	assertHasOpcode(t, instructions, Z80_JP_CC_NN, "Should have conditional jump instruction")
}

func TestZ80_BinaryOp_GreaterEqual(t *testing.T) {
	tc := newTestContext()

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpGreaterEqual,
		Left:     &zsm.SemConstant{Value: 25, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate CP with immediate and conditional jump
	assertHasOpcode(t, instructions, Z80_CP_N, "Should have CP_N instruction")
	assertHasOpcode(t, instructions, Z80_JP_CC_NN, "Should have conditional jump instruction")
}

// ============================================================================
// Binary Operations - Logical
// ============================================================================

func TestZ80_BinaryOp_LogicalAnd(t *testing.T) {
	tc := newTestContext()

	// Create comparison expressions (5 < 10) && (20 > 5)
	left := &zsm.SemBinaryOp{
		Op:       zsm.OpLessThan,
		Left:     &zsm.SemConstant{Value: 5, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}
	right := &zsm.SemBinaryOp{
		Op:       zsm.OpGreaterThan,
		Left:     &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 5, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpLogicalAnd,
		Left:     left,
		Right:    right,
		TypeInfo: u8Type(),
	}

	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Logical AND uses short-circuit evaluation with conditional jumps
	// TODO: Verify short-circuit AND implementation
	// Should have conditional jumps to handle short-circuit
	hasBranch := false
	for _, instr := range instructions {
		if len(instr.GetTargetBlocks()) > 0 {
			hasBranch = true
			break
		}
	}
	assert.True(t, hasBranch, "Should have branch instructions for logical AND")
}

func TestZ80_BinaryOp_LogicalOr(t *testing.T) {
	tc := newTestContext()

	// Create comparison expressions (5 < 10) || (20 > 5)
	left := &zsm.SemBinaryOp{
		Op:       zsm.OpLessThan,
		Left:     &zsm.SemConstant{Value: 5, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}
	right := &zsm.SemBinaryOp{
		Op:       zsm.OpGreaterThan,
		Left:     &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		Right:    &zsm.SemConstant{Value: 5, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpLogicalOr,
		Left:     left,
		Right:    right,
		TypeInfo: u8Type(),
	}

	trueBlock := newTestBlock()
	falseBlock := newTestBlock()
	exprCtx := NewExprContextBranch(trueBlock, falseBlock)

	vr, err := tc.ctx.selectBinaryOp(exprCtx, binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Logical OR uses short-circuit evaluation with conditional jumps
	// TODO: Verify short-circuit OR implementation
	hasBranch := false
	for _, instr := range instructions {
		if len(instr.GetTargetBlocks()) > 0 {
			hasBranch = true
			break
		}
	}
	assert.True(t, hasBranch, "Should have branch instructions for logical OR")
}

// ============================================================================
// Unary Operations
// ============================================================================

func TestZ80_UnaryOp_Negate(t *testing.T) {
	tc := newTestContext()

	unaryOp := &zsm.SemUnaryOp{
		Op:       zsm.OpNegate,
		Operand:  &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectUnaryOp(nil, unaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Negate uses NEG instruction (two's complement)
	assertHasOpcode(t, instructions, Z80_NEG, "Should have NEG instruction")
}

func TestZ80_UnaryOp_LogicalNot(t *testing.T) {
	tc := newTestContext()

	unaryOp := &zsm.SemUnaryOp{
		Op:       zsm.OpLogicalNot,
		Operand:  &zsm.SemConstant{Value: 1, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectUnaryOp(nil, unaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Logical NOT typically implemented as comparison with zero
	// TODO: Verify logical NOT implementation strategy
}

func TestZ80_UnaryOp_BitwiseNot(t *testing.T) {
	tc := newTestContext()

	unaryOp := &zsm.SemUnaryOp{
		Op:       zsm.OpBitwiseNot,
		Operand:  &zsm.SemConstant{Value: 0xAA, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectUnaryOp(nil, unaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Bitwise NOT typically uses XOR with 0xFF or CPL instruction
	// TODO: Verify bitwise NOT implementation (CPL instruction)
}

// ============================================================================
// Statement Tests
// ============================================================================

func TestZ80_VariableDecl_WithInitializer(t *testing.T) {
	tc := newTestContext()

	symbol := &zsm.Symbol{
		Name: "x",
		Type: u8Type(),
	}

	decl := &zsm.SemVariableDecl{
		Symbol:      symbol,
		Initializer: &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo:    u8Type(),
	}

	err := tc.ctx.selectVariableDecl(decl)

	require.NoError(t, err)

	// Check that symbol is mapped to VR
	vr, ok := tc.ctx.symbolToVReg[symbol]
	assert.True(t, ok)
	assert.NotNil(t, vr)
	assert.Equal(t, "x", vr.Name)
	assert.Equal(t, uint8(8), vr.Size)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate load instruction for initializer
	// TODO: Verify exact pattern for variable initialization
}

func TestZ80_VariableDecl_NoInitializer(t *testing.T) {
	tc := newTestContext()

	symbol := &zsm.Symbol{
		Name: "y",
		Type: u8Type(),
	}

	decl := &zsm.SemVariableDecl{
		Symbol:      symbol,
		Initializer: nil,
		TypeInfo:    u8Type(),
	}

	err := tc.ctx.selectVariableDecl(decl)

	require.NoError(t, err)

	// Variable should still be allocated
	vr, ok := tc.ctx.symbolToVReg[symbol]
	assert.True(t, ok)
	assert.NotNil(t, vr)

	// No initializer means no instructions (or minimal setup)
	// instructions := tc.getZ80Instructions()
	// assert.Empty(t, instructions) // May or may not generate instructions
}

func TestZ80_Assignment(t *testing.T) {
	tc := newTestContext()

	// Create a variable first
	symbol := &zsm.Symbol{
		Name: "x",
		Type: u8Type(),
	}
	tc.ctx.symbolToVReg[symbol] = tc.ctx.vrAlloc.AllocateNamed("x", Z80Registers8)

	assignment := &zsm.SemAssignment{
		Target: symbol,
		Value:  &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
	}

	err := tc.ctx.selectAssignment(assignment)

	require.NoError(t, err)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate load/move instructions
	// TODO: Verify assignment instruction pattern
}

func TestZ80_Return_WithValue(t *testing.T) {
	tc := newTestContext()

	returnStmt := &zsm.SemReturn{
		Value: &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
	}

	err := tc.ctx.selectReturn(returnStmt)

	require.NoError(t, err)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate instructions to load return value and RET
	assertHasOpcode(t, instructions, Z80_RET, "Should have RET instruction")
}

func TestZ80_Return_Void(t *testing.T) {
	tc := newTestContext()

	returnStmt := &zsm.SemReturn{
		Value: nil,
	}

	err := tc.ctx.selectReturn(returnStmt)

	require.NoError(t, err)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate RET instruction
	assertHasOpcode(t, instructions, Z80_RET, "Should have RET instruction")
	// For void return, should only have RET (no value loading)
	// assertInstructionCount(t, instructions, 1, "Void return should only generate RET")
}

func TestZ80_FunctionCall(t *testing.T) {
	tc := newTestContext()

	funcSymbol := &zsm.Symbol{
		Name: "add",
		Type: zsm.NewFunctionType([]zsm.Type{u8Type(), u8Type()}, u8Type()),
	}

	call := &zsm.SemFunctionCall{
		Function: funcSymbol,
		Arguments: []zsm.SemExpression{
			&zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
			&zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
		},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectFunctionCall(nil, call)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should generate CALL instruction
	assertHasOpcode(t, instructions, Z80_CALL_NN, "Should have CALL_NN instruction")
	// TODO: Verify argument passing convention
}

func TestZ80_SymbolRef(t *testing.T) {
	tc := newTestContext()

	// Create a variable
	symbol := &zsm.Symbol{
		Name: "x",
		Type: u8Type(),
	}
	expectedVR := tc.ctx.vrAlloc.AllocateNamed("x", Z80Registers8)
	tc.ctx.symbolToVReg[symbol] = expectedVR

	symbolRef := &zsm.SemSymbolRef{
		Symbol: symbol,
	}

	vr, err := tc.ctx.selectSymbolRef(symbolRef)

	require.NoError(t, err)
	assert.Equal(t, expectedVR, vr, "Should return the same VR")

	// Symbol references don't generate instructions by themselves
	instructions := tc.getZ80Instructions()
	assert.Empty(t, instructions)
}

// ============================================================================
// Complex Expression Tests
// ============================================================================

func TestZ80_ComplexExpression_Nested(t *testing.T) {
	tc := newTestContext()

	// (10 + 20) * 30
	expr := &zsm.SemBinaryOp{
		Op: zsm.OpMultiply,
		Left: &zsm.SemBinaryOp{
			Op:       zsm.OpAdd,
			Left:     &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
			Right:    &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
			TypeInfo: u8Type(),
		},
		Right:    &zsm.SemConstant{Value: 30, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectExpression(expr)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should have ADD for inner expression
	assertHasOpcode(t, instructions, Z80_ADD_A_R, "Should have ADD for inner expression")
	// TODO: Verify multiply instructions follow the ADD
}

func TestZ80_ComplexExpression_MultipleVariables(t *testing.T) {
	tc := newTestContext()

	// Create variables
	symbolX := &zsm.Symbol{Name: "x", Type: u8Type()}
	symbolY := &zsm.Symbol{Name: "y", Type: u8Type()}

	tc.ctx.symbolToVReg[symbolX] = tc.ctx.vrAlloc.AllocateNamed("x", Z80Registers8)
	tc.ctx.symbolToVReg[symbolY] = tc.ctx.vrAlloc.AllocateNamed("y", Z80Registers8)

	// x + y + 42
	expr := &zsm.SemBinaryOp{
		Op: zsm.OpAdd,
		Left: &zsm.SemBinaryOp{
			Op:       zsm.OpAdd,
			Left:     &zsm.SemSymbolRef{Symbol: symbolX},
			Right:    &zsm.SemSymbolRef{Symbol: symbolY},
			TypeInfo: u8Type(),
		},
		Right:    &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectExpression(expr)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Should have multiple ADD instructions (x+y uses ADD_A_R, result+42 uses ADD_A_N)
	addCount := countOpcode(instructions, Z80_ADD_A_R) + countOpcode(instructions, Z80_ADD_A_N)
	assert.True(t, addCount >= 2, "Should have at least 2 ADD instructions (any combination of ADD_A_R and ADD_A_N)")
}

// ============================================================================
// Array/Subscript Tests
// ============================================================================

func TestZ80_Subscript_U8Array(t *testing.T) {
	tc := newTestContext()

	// Create array variable
	arrayType := zsm.NewArrayType(u8Type(), 10) // Array of 10 elements
	arraySymbol := &zsm.Symbol{
		Name: "arr",
		Type: arrayType,
	}
	tc.ctx.symbolToVReg[arraySymbol] = tc.ctx.vrAlloc.AllocateNamed("arr", Z80Registers16)

	subscript := &zsm.SemSubscript{
		Array:    &zsm.SemSymbolRef{Symbol: arraySymbol},
		Index:    &zsm.SemConstant{Value: 2, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectExpression(subscript)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Array access requires address calculation and load
	// TODO: Verify array indexing instruction pattern
	// Likely involves: address calc, then LD_R_HL or similar
}

func TestZ80_Subscript_U16Array(t *testing.T) {
	tc := newTestContext()

	// Create array variable
	arrayType := zsm.NewArrayType(u16Type(), 5) // Array of 5 elements
	arraySymbol := &zsm.Symbol{
		Name: "arr16",
		Type: arrayType,
	}
	tc.ctx.symbolToVReg[arraySymbol] = tc.ctx.vrAlloc.AllocateNamed("arr16", Z80Registers16)

	subscript := &zsm.SemSubscript{
		Array:    &zsm.SemSymbolRef{Symbol: arraySymbol},
		Index:    &zsm.SemConstant{Value: 1, TypeInfo: u8Type()},
		TypeInfo: u16Type(),
	}

	vr, err := tc.ctx.selectExpression(subscript)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, uint8(16), vr.Size)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// 16-bit array access
	// TODO: Verify 16-bit array indexing instruction pattern
}

// ============================================================================
// Member Access Tests
// ============================================================================

func TestZ80_MemberAccess(t *testing.T) {
	tc := newTestContext()

	// Create struct type
	xField := &zsm.StructField{Name: "x", Type: u8Type()}
	yField := &zsm.StructField{Name: "y", Type: u8Type()}
	structType := zsm.NewStructType("Point", []*zsm.StructField{xField, yField})

	// Create struct variable
	structSymbol := &zsm.Symbol{
		Name: "pt",
		Type: zsm.NewPointerType(structType),
	}
	tc.ctx.symbolToVReg[structSymbol] = tc.ctx.vrAlloc.AllocateNamed("pt", Z80Registers16)

	structRef := zsm.SemExpression(&zsm.SemSymbolRef{Symbol: structSymbol})
	memberAccess := &zsm.SemMemberAccess{
		Object:   &structRef,
		Field:    xField,
		TypeInfo: u8Type(),
	}

	vr, err := tc.ctx.selectExpression(memberAccess)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	instructions := tc.getZ80Instructions()
	require.NotEmpty(t, instructions)

	// Member access requires offset calculation and load
	// TODO: Verify struct member access instruction pattern
}

// ============================================================================
// Expression Caching Test
// ============================================================================

func TestZ80_ExpressionCaching(t *testing.T) {
	tc := newTestContext()

	constant := &zsm.SemConstant{Value: 42, TypeInfo: u8Type()}

	// First call
	vr1, err := tc.ctx.selectExpression(constant)
	require.NoError(t, err)
	assert.NotNil(t, vr1)

	count1 := len(tc.getInstructions())

	// Second call - should reuse cached result
	vr2, err := tc.ctx.selectExpression(constant)
	require.NoError(t, err)
	assert.NotNil(t, vr2)
	assert.Equal(t, vr1, vr2, "Should return same VirtualRegister")

	count2 := len(tc.getInstructions())
	assert.Equal(t, count1, count2, "Should not generate additional instructions")
}
