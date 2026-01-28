package cfg

import (
	"testing"
	"zenith/compiler/zir"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock InstructionSelector for testing
type mockInstructionSelector struct {
	instructions      []string // Record of operations called
	allocatedVRs      []*VirtualRegister
	vrAlloc           *VirtualRegisterAllocator
	callingConvention CallingConvention
}

func newMockInstructionSelector(cc CallingConvention) *mockInstructionSelector {
	return &mockInstructionSelector{
		instructions:      make([]string, 0),
		allocatedVRs:      make([]*VirtualRegister, 0),
		vrAlloc:           NewVirtualRegisterAllocator(),
		callingConvention: cc,
	}
}

func (m *mockInstructionSelector) recordOp(op string) {
	m.instructions = append(m.instructions, op)
}

// Arithmetic operations
func (m *mockInstructionSelector) SelectAdd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("add")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectSubtract(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("subtract")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectMultiply(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("multiply")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectDivide(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("divide")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectNegate(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("negate")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

// Bitwise operations
func (m *mockInstructionSelector) SelectBitwiseAnd(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("and")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectBitwiseOr(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("or")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectBitwiseXor(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("xor")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectBitwiseNot(operand *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("not")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectShiftLeft(value, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("shl")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectShiftRight(value, amount *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("shr")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectLogicalAnd(left, right *VirtualRegister) (*VirtualRegister, error) {
	m.recordOp("logical_and")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectLogicalOr(left, right *VirtualRegister) (*VirtualRegister, error) {
	m.recordOp("logical_or")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectLogicalNot(operand *VirtualRegister) (*VirtualRegister, error) {
	m.recordOp("logical_not")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

// Comparison operations
func (m *mockInstructionSelector) SelectEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("equal")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectNotEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("not_equal")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectLessThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("less_than")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectLessEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("less_equal")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectGreaterThan(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("greater_than")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectGreaterEqual(left, right *VirtualRegister, size int) (*VirtualRegister, error) {
	m.recordOp("greater_equal")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

// Memory operations
func (m *mockInstructionSelector) SelectLoad(address *VirtualRegister, offset int, size int) (*VirtualRegister, error) {
	m.recordOp("load")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectStore(address, value *VirtualRegister, offset int, size int) error {
	m.recordOp("store")
	return nil
}

func (m *mockInstructionSelector) SelectLoadConstant(value interface{}, size int) (*VirtualRegister, error) {
	m.recordOp("load_constant")
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectLoadVariable(symbol *zir.Symbol) (*VirtualRegister, error) {
	m.recordOp("load_variable")
	vr := m.vrAlloc.Allocate(8)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr, nil
}

func (m *mockInstructionSelector) SelectStoreVariable(symbol *zir.Symbol, value *VirtualRegister) error {
	m.recordOp("store_variable")
	return nil
}

func (m *mockInstructionSelector) SelectMove(target, source *VirtualRegister, size int) error {
	m.recordOp("move")
	return nil
}

// Control flow
func (m *mockInstructionSelector) SelectBranch(condition *VirtualRegister, trueBlock, falseBlock *BasicBlock) error {
	m.recordOp("branch")
	return nil
}

func (m *mockInstructionSelector) SelectJump(target *BasicBlock) error {
	m.recordOp("jump")
	return nil
}

func (m *mockInstructionSelector) SelectBlockLabel(block *BasicBlock) error {
	m.recordOp("label")
	return nil
}

func (m *mockInstructionSelector) SelectCall(functionName string, args []*VirtualRegister, returnSize int) (*VirtualRegister, error) {
	m.recordOp("call")
	if returnSize > 0 {
		vr := m.vrAlloc.Allocate(returnSize)
		m.allocatedVRs = append(m.allocatedVRs, vr)
		return vr, nil
	}
	return nil, nil
}

func (m *mockInstructionSelector) SelectReturn(value *VirtualRegister) error {
	m.recordOp("return")
	return nil
}

func (m *mockInstructionSelector) SelectFunctionPrologue(fn *zir.IRFunctionDecl) error {
	m.recordOp("prologue")
	return nil
}

func (m *mockInstructionSelector) SelectFunctionEpilogue(fn *zir.IRFunctionDecl) error {
	m.recordOp("epilogue")
	return nil
}

// Utility
func (m *mockInstructionSelector) AllocateVirtual(size int) *VirtualRegister {
	vr := m.vrAlloc.Allocate(size)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr
}

func (m *mockInstructionSelector) AllocateVirtualConstrained(size int, allowedSet []*Register, requiredClass RegisterClass) *VirtualRegister {
	vr := m.vrAlloc.AllocateConstrained(size, allowedSet, requiredClass)
	m.allocatedVRs = append(m.allocatedVRs, vr)
	return vr
}

func (m *mockInstructionSelector) EmitInstruction(instr MachineInstruction) {
	m.recordOp("emit")
}

func (m *mockInstructionSelector) GetInstructions() []MachineInstruction {
	return nil
}

func (m *mockInstructionSelector) ClearInstructions() {
	m.instructions = make([]string, 0)
}

func (m *mockInstructionSelector) NewLabel(prefix string) string {
	return "label"
}

func (m *mockInstructionSelector) GetCallingConvention() CallingConvention {
	return m.callingConvention
}

func (m *mockInstructionSelector) GetTargetRegisters() []*Register {
	return []*Register{}
}

// Mock CallingConvention
type mockCallingConvention struct{}

func (m *mockCallingConvention) GetParameterLocation(paramIndex int, paramSize int) (register *Register, stackOffset int, useStack bool) {
	// First param in register, rest on stack
	if paramIndex == 0 {
		return &RegA, 0, false
	}
	return nil, paramIndex * 2, true
}

func (m *mockCallingConvention) GetReturnValueRegister(returnSize int) *Register {
	return &RegA
}

func (m *mockCallingConvention) GetCallerSavedRegisters() []*Register {
	return []*Register{&RegA}
}

func (m *mockCallingConvention) GetCalleeSavedRegisters() []*Register {
	return []*Register{}
}

func (m *mockCallingConvention) GetStackAlignment() int {
	return 2
}

func (m *mockCallingConvention) GetStackGrowthDirection() bool {
	return true
}

// Helper to get basic types
func u8Type() zir.Type {
	return zir.U8Type
}

func u16Type() zir.Type {
	return zir.U16Type
}

// Test helpers to create IR nodes with proper types
func newIRConstant(value interface{}, typ zir.Type) *zir.IRConstant {
	return &zir.IRConstant{
		Value:    value,
		TypeInfo: typ,
	}
}

func newIRBinaryOp(op zir.BinaryOperator, left, right zir.IRExpression, typ zir.Type) *zir.IRBinaryOp {
	return &zir.IRBinaryOp{
		Op:       op,
		Left:     left,
		Right:    right,
		TypeInfo: typ,
	}
}

func newIRUnaryOp(op zir.UnaryOperator, operand zir.IRExpression, typ zir.Type) *zir.IRUnaryOp {
	return &zir.IRUnaryOp{
		Op:       op,
		Operand:  operand,
		TypeInfo: typ,
	}
}

// Test selectConstant
func Test_InstructionSelection_Constant(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	constant := &zir.IRConstant{
		Value:    42,
		TypeInfo: u8Type(),
	}

	vr, err := ctx.selectConstant(constant)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Contains(t, selector.instructions, "load_constant")
}

// Test selectBinaryOp with addition
func Test_InstructionSelection_BinaryOp_Add(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	left := &zir.IRConstant{Value: 10, TypeInfo: u8Type()}
	right := &zir.IRConstant{Value: 20, TypeInfo: u8Type()}

	binaryOp := &zir.IRBinaryOp{
		Op:       zir.OpAdd,
		Left:     left,
		Right:    right,
		TypeInfo: u8Type(),
	}

	vr, err := ctx.selectBinaryOp(binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Contains(t, selector.instructions, "load_constant")
	assert.Contains(t, selector.instructions, "add")
}

// Test all binary operators
func Test_InstructionSelection_BinaryOp_AllOperators(t *testing.T) {
	tests := []struct {
		name     string
		op       zir.BinaryOperator
		expected string
	}{
		{"Add", zir.OpAdd, "add"},
		{"Subtract", zir.OpSubtract, "subtract"},
		{"Multiply", zir.OpMultiply, "multiply"},
		{"Divide", zir.OpDivide, "divide"},
		{"BitwiseAnd", zir.OpBitwiseAnd, "and"},
		{"BitwiseOr", zir.OpBitwiseOr, "or"},
		{"BitwiseXor", zir.OpBitwiseXor, "xor"},
		{"Equal", zir.OpEqual, "equal"},
		{"NotEqual", zir.OpNotEqual, "not_equal"},
		{"LessThan", zir.OpLessThan, "less_than"},
		{"LessEqual", zir.OpLessEqual, "less_equal"},
		{"GreaterThan", zir.OpGreaterThan, "greater_than"},
		{"GreaterEqual", zir.OpGreaterEqual, "greater_equal"},
		{"LogicalAnd", zir.OpLogicalAnd, "logical_and"},
		{"LogicalOr", zir.OpLogicalOr, "logical_or"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &mockCallingConvention{}
			selector := newMockInstructionSelector(cc)
			ctx := NewInstructionSelectionContext(selector, cc)

			left := newIRConstant(10, u8Type())
			right := newIRConstant(20, u8Type())
			binaryOp := newIRBinaryOp(tt.op, left, right, u8Type())

			vr, err := ctx.selectBinaryOp(binaryOp)

			require.NoError(t, err)
			assert.NotNil(t, vr)
			assert.Contains(t, selector.instructions, tt.expected)
		})
	}
}

// Test selectUnaryOp
func Test_InstructionSelection_UnaryOp(t *testing.T) {
	tests := []struct {
		name     string
		op       zir.UnaryOperator
		expected string
	}{
		{"Negate", zir.OpNegate, "negate"},
		{"Not", zir.OpNot, "logical_not"},
		{"BitwiseNot", zir.OpBitwiseNot, "not"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &mockCallingConvention{}
			selector := newMockInstructionSelector(cc)
			ctx := NewInstructionSelectionContext(selector, cc)

			operand := newIRConstant(42, u8Type())
			unaryOp := newIRUnaryOp(tt.op, operand, u8Type())

			vr, err := ctx.selectUnaryOp(unaryOp)

			require.NoError(t, err)
			assert.NotNil(t, vr)
			assert.Contains(t, selector.instructions, tt.expected)
		})
	}
}

// Test selectVariableDecl
func Test_InstructionSelection_VariableDecl(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol := &zir.Symbol{
		Name: "x",
		Type: u8Type(),
	}

	decl := &zir.IRVariableDecl{
		Symbol:      symbol,
		Initializer: &zir.IRConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo:    u8Type(),
	}

	err := ctx.selectVariableDecl(decl)

	require.NoError(t, err)
	assert.Contains(t, selector.instructions, "load_constant")
	assert.Contains(t, selector.instructions, "move")

	// Check that symbol is mapped to VR
	vr, ok := ctx.symbolToVReg[symbol]
	assert.True(t, ok)
	assert.NotNil(t, vr)
	assert.Equal(t, "x", vr.Name)
}

// Test selectAssignment
func Test_InstructionSelection_Assignment(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	// Create a variable first
	symbol := &zir.Symbol{
		Name: "x",
		Type: u8Type(),
	}
	ctx.symbolToVReg[symbol] = ctx.vrAlloc.AllocateNamed("x", 8)

	assignment := &zir.IRAssignment{
		Target: symbol,
		Value:  &zir.IRConstant{Value: 42, TypeInfo: u8Type()},
	}

	err := ctx.selectAssignment(assignment)

	require.NoError(t, err)
	assert.Contains(t, selector.instructions, "load_constant")
	assert.Contains(t, selector.instructions, "move")
}

// Test selectReturn with value
func Test_InstructionSelection_ReturnWithValue(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	returnStmt := &zir.IRReturn{
		Value: &zir.IRConstant{Value: 42, TypeInfo: u8Type()},
	}

	err := ctx.selectReturn(returnStmt)

	require.NoError(t, err)
	assert.Contains(t, selector.instructions, "load_constant")
	assert.Contains(t, selector.instructions, "move")
	assert.Contains(t, selector.instructions, "return")
}

// Test selectReturn void
func Test_InstructionSelection_ReturnVoid(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	returnStmt := &zir.IRReturn{
		Value: nil,
	}

	err := ctx.selectReturn(returnStmt)

	require.NoError(t, err)
	assert.Contains(t, selector.instructions, "return")
	// Should not have load_constant or move
	assert.NotContains(t, selector.instructions, "load_constant")
}

// Test selectFunctionCall
func Test_InstructionSelection_FunctionCall(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	funcSymbol := &zir.Symbol{
		Name: "add",
		Type: zir.NewFunctionType([]zir.Type{u8Type(), u8Type()}, u8Type()),
	}

	call := &zir.IRFunctionCall{
		Function: funcSymbol,
		Arguments: []zir.IRExpression{
			&zir.IRConstant{Value: 10, TypeInfo: u8Type()},
			&zir.IRConstant{Value: 20, TypeInfo: u8Type()},
		},
		TypeInfo: u8Type(),
	}

	vr, err := ctx.selectFunctionCall(call)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Contains(t, selector.instructions, "call")
	assert.Contains(t, selector.instructions, "load_constant")
}

// Test expression caching
func Test_InstructionSelection_ExpressionCaching(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	constant := &zir.IRConstant{Value: 42, TypeInfo: u8Type()}

	// First call - should generate instruction
	vr1, err := ctx.selectExpression(constant)
	require.NoError(t, err)
	assert.NotNil(t, vr1)

	count1 := len(selector.instructions)

	// Second call - should reuse cached result
	vr2, err := ctx.selectExpression(constant)
	require.NoError(t, err)
	assert.NotNil(t, vr2)
	assert.Equal(t, vr1, vr2, "Should return same VirtualRegister")

	count2 := len(selector.instructions)
	assert.Equal(t, count1, count2, "Should not generate additional instructions")
}

// Test selectSymbolRef
func Test_InstructionSelection_SymbolRef(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	// Create a variable
	symbol := &zir.Symbol{
		Name: "x",
		Type: u8Type(),
	}
	expectedVR := ctx.vrAlloc.AllocateNamed("x", 8)
	ctx.symbolToVReg[symbol] = expectedVR

	symbolRef := &zir.IRSymbolRef{
		Symbol: symbol,
	}

	vr, err := ctx.selectSymbolRef(symbolRef)

	require.NoError(t, err)
	assert.Equal(t, expectedVR, vr)
}

// Test selectSymbolRef with undefined variable
func Test_InstructionSelection_SymbolRef_Undefined(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol := &zir.Symbol{
		Name: "undefined",
		Type: u8Type(),
	}

	symbolRef := &zir.IRSymbolRef{
		Symbol: symbol,
	}

	_, err := ctx.selectSymbolRef(symbolRef)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined variable")
}

// Test selectFunction with parameters
func Test_InstructionSelection_Function_WithParameters(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	param1 := &zir.Symbol{Name: "a", Type: u8Type()}
	param2 := &zir.Symbol{Name: "b", Type: u8Type()}

	fn := &zir.IRFunctionDecl{
		Name:       "add",
		Parameters: []*zir.Symbol{param1, param2},
		ReturnType: u8Type(),
		Body: &zir.IRBlock{
			Statements: []zir.IRStatement{
				&zir.IRReturn{
					Value: &zir.IRBinaryOp{
						Op:       zir.OpAdd,
						Left:     &zir.IRSymbolRef{Symbol: param1},
						Right:    &zir.IRSymbolRef{Symbol: param2},
						TypeInfo: u8Type(),
					},
				},
			},
		},
	}

	err := ctx.selectFunction(fn)

	require.NoError(t, err)

	// Check that parameters are allocated
	vr1, ok := ctx.symbolToVReg[param1]
	assert.True(t, ok)
	assert.NotNil(t, vr1)

	vr2, ok := ctx.symbolToVReg[param2]
	assert.True(t, ok)
	assert.NotNil(t, vr2)

	// First param should be in register
	assert.False(t, vr1.HasStackHome, "First parameter should be in register")

	// Second param should be on stack
	assert.True(t, vr2.HasStackHome, "Second parameter should be on stack")
}

// Test SelectInstructions with full compilation unit
func Test_SelectInstructions_Simple(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)

	fn := &zir.IRFunctionDecl{
		Name:       "test",
		Parameters: []*zir.Symbol{},
		ReturnType: u8Type(),
		Body: &zir.IRBlock{
			Statements: []zir.IRStatement{
				&zir.IRReturn{
					Value: &zir.IRConstant{Value: 42, TypeInfo: u8Type()},
				},
			},
		},
	}

	compilationUnit := &zir.IRCompilationUnit{
		Declarations: []zir.IRDeclaration{fn},
	}

	err := SelectInstructions(compilationUnit, selector, cc)

	require.NoError(t, err)
	assert.Contains(t, selector.instructions, "load_constant")
	assert.Contains(t, selector.instructions, "return")
}

// Test complex expression with nested operations
func Test_InstructionSelection_ComplexExpression(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	// (10 + 20) * 30
	expr := &zir.IRBinaryOp{
		Op: zir.OpMultiply,
		Left: &zir.IRBinaryOp{
			Op:       zir.OpAdd,
			Left:     &zir.IRConstant{Value: 10, TypeInfo: u8Type()},
			Right:    &zir.IRConstant{Value: 20, TypeInfo: u8Type()},
			TypeInfo: u8Type(),
		},
		Right:    &zir.IRConstant{Value: 30, TypeInfo: u8Type()},
		TypeInfo: u8Type(),
	}

	vr, err := ctx.selectExpression(expr)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Contains(t, selector.instructions, "add")
	assert.Contains(t, selector.instructions, "multiply")
	assert.Contains(t, selector.instructions, "load_constant")
}

// Test multiple variable declarations
func Test_InstructionSelection_MultipleVariables(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol1 := &zir.Symbol{Name: "x", Type: u8Type()}
	symbol2 := &zir.Symbol{Name: "y", Type: u8Type()}

	decl1 := &zir.IRVariableDecl{
		Symbol:      symbol1,
		Initializer: &zir.IRConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo:    u8Type(),
	}

	decl2 := &zir.IRVariableDecl{
		Symbol:      symbol2,
		Initializer: &zir.IRConstant{Value: 20, TypeInfo: u8Type()},
		TypeInfo:    u8Type(),
	}

	err := ctx.selectVariableDecl(decl1)
	require.NoError(t, err)

	err = ctx.selectVariableDecl(decl2)
	require.NoError(t, err)

	// Check both variables are mapped
	assert.Contains(t, ctx.symbolToVReg, symbol1)
	assert.Contains(t, ctx.symbolToVReg, symbol2)
	assert.NotEqual(t, ctx.symbolToVReg[symbol1], ctx.symbolToVReg[symbol2])
}

// Test variable declaration without initializer
func Test_InstructionSelection_VariableDecl_NoInitializer(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol := &zir.Symbol{Name: "x", Type: u8Type()}

	decl := &zir.IRVariableDecl{
		Symbol:      symbol,
		Initializer: nil,
		TypeInfo:    u8Type(),
	}

	err := ctx.selectVariableDecl(decl)

	require.NoError(t, err)
	// Should not contain move instruction
	assert.NotContains(t, selector.instructions, "move")

	// Variable should still be allocated
	vr, ok := ctx.symbolToVReg[symbol]
	assert.True(t, ok)
	assert.NotNil(t, vr)
}

// Test 16-bit operations
func Test_InstructionSelection_16BitOperations(t *testing.T) {
	cc := &mockCallingConvention{}
	selector := newMockInstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	binaryOp := &zir.IRBinaryOp{
		Op:       zir.OpAdd,
		Left:     &zir.IRConstant{Value: 1000, TypeInfo: u16Type()},
		Right:    &zir.IRConstant{Value: 2000, TypeInfo: u16Type()},
		TypeInfo: u16Type(),
	}

	vr, err := ctx.selectBinaryOp(binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, 16, vr.Size)
	assert.Contains(t, selector.instructions, "add")
}
