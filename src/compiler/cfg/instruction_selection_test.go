package cfg

import (
	"testing"
	"zenith/compiler/zsm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to get basic types
func u8Type() zsm.Type {
	return zsm.U8Type
}

func u16Type() zsm.Type {
	return zsm.U16Type
}

// Test helpers to create IR nodes with proper types
func newSemConstant(value interface{}, typ zsm.Type) *zsm.SemConstant {
	return &zsm.SemConstant{
		Value:    value,
		TypeInfo: typ,
	}
}

func newSemBinaryOp(op zsm.BinaryOperator, left, right zsm.SemExpression, typ zsm.Type) *zsm.SemBinaryOp {
	return &zsm.SemBinaryOp{
		Op:       op,
		Left:     left,
		Right:    right,
		TypeInfo: typ,
	}
}

func newSemUnaryOp(op zsm.UnaryOperator, operand zsm.SemExpression, typ zsm.Type) *zsm.SemUnaryOp {
	return &zsm.SemUnaryOp{
		Op:       op,
		Operand:  operand,
		TypeInfo: typ,
	}
}

// Test selectConstant
func Test_InstructionSelection_Constant(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	constant := &zsm.SemConstant{
		Value:    42,
		TypeInfo: u8Type(),
	}

	vr, err := ctx.selectConstant(constant)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, 8, vr.Size)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test selectBinaryOp with addition
func Test_InstructionSelection_BinaryOp_Add(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	left := &zsm.SemConstant{Value: 10, TypeInfo: u8Type()}
	right := &zsm.SemConstant{Value: 20, TypeInfo: u8Type()}

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpAdd,
		Left:     left,
		Right:    right,
		TypeInfo: u8Type(),
	}

	vr, err := ctx.selectBinaryOp(binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, 8, vr.Size)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
	// Should have at least 2 load constants and add instructions
	assert.GreaterOrEqual(t, len(instructions), 3)
}

// Test all binary operators
func Test_InstructionSelection_BinaryOp_AllOperators(t *testing.T) {
	tests := []struct {
		name string
		op   zsm.BinaryOperator
	}{
		{"Add", zsm.OpAdd},
		{"Subtract", zsm.OpSubtract},
		{"Multiply", zsm.OpMultiply},
		{"Divide", zsm.OpDivide},
		{"BitwiseAnd", zsm.OpBitwiseAnd},
		{"BitwiseOr", zsm.OpBitwiseOr},
		{"BitwiseXor", zsm.OpBitwiseXor},
		{"Equal", zsm.OpEqual},
		{"NotEqual", zsm.OpNotEqual},
		{"LessThan", zsm.OpLessThan},
		{"LessEqual", zsm.OpLessEqual},
		{"GreaterThan", zsm.OpGreaterThan},
		{"GreaterEqual", zsm.OpGreaterEqual},
		{"LogicalAnd", zsm.OpLogicalAnd},
		{"LogicalOr", zsm.OpLogicalOr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := NewZ80CallingConvention()
			selector := NewZ80InstructionSelector(cc)
			ctx := NewInstructionSelectionContext(selector, cc)

			left := newSemConstant(10, u8Type())
			right := newSemConstant(20, u8Type())
			binaryOp := newSemBinaryOp(tt.op, left, right, u8Type())

			vr, err := ctx.selectBinaryOp(binaryOp)

			require.NoError(t, err)
			assert.NotNil(t, vr)

			// Check that instructions were generated
			instructions := selector.GetInstructions()
			assert.NotEmpty(t, instructions)
		})
	}
}

// Test selectUnaryOp
func Test_InstructionSelection_UnaryOp(t *testing.T) {
	tests := []struct {
		name string
		op   zsm.UnaryOperator
	}{
		{"Negate", zsm.OpNegate},
		{"Not", zsm.OpNot},
		{"BitwiseNot", zsm.OpBitwiseNot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := NewZ80CallingConvention()
			selector := NewZ80InstructionSelector(cc)
			ctx := NewInstructionSelectionContext(selector, cc)

			operand := newSemConstant(42, u8Type())
			unaryOp := newSemUnaryOp(tt.op, operand, u8Type())

			vr, err := ctx.selectUnaryOp(unaryOp)

			require.NoError(t, err)
			assert.NotNil(t, vr)

			// Check that instructions were generated
			instructions := selector.GetInstructions()
			assert.NotEmpty(t, instructions)
		})
	}
}

// Test selectVariableDecl
func Test_InstructionSelection_VariableDecl(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol := &zsm.Symbol{
		Name: "x",
		Type: u8Type(),
	}

	decl := &zsm.SemVariableDecl{
		Symbol:      symbol,
		Initializer: &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo:    u8Type(),
	}

	err := ctx.selectVariableDecl(decl)

	require.NoError(t, err)

	// Check that symbol is mapped to VR
	vr, ok := ctx.symbolToVReg[symbol]
	assert.True(t, ok)
	assert.NotNil(t, vr)
	assert.Equal(t, "x", vr.Name)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test selectAssignment
func Test_InstructionSelection_Assignment(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	// Create a variable first
	symbol := &zsm.Symbol{
		Name: "x",
		Type: u8Type(),
	}
	ctx.symbolToVReg[symbol] = ctx.vrAlloc.AllocateNamed("x", 8)

	assignment := &zsm.SemAssignment{
		Target: symbol,
		Value:  &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
	}

	err := ctx.selectAssignment(assignment)

	require.NoError(t, err)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test selectReturn with value
func Test_InstructionSelection_ReturnWithValue(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	returnStmt := &zsm.SemReturn{
		Value: &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
	}

	err := ctx.selectReturn(returnStmt)

	require.NoError(t, err)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test selectReturn void
func Test_InstructionSelection_ReturnVoid(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	returnStmt := &zsm.SemReturn{
		Value: nil,
	}

	err := ctx.selectReturn(returnStmt)

	require.NoError(t, err)

	// Check that instructions were generated (at least RET)
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test selectFunctionCall
func Test_InstructionSelection_FunctionCall(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

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

	vr, err := ctx.selectFunctionCall(call)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test expression caching
func Test_InstructionSelection_ExpressionCaching(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	constant := &zsm.SemConstant{Value: 42, TypeInfo: u8Type()}

	// First call - should generate instruction
	vr1, err := ctx.selectExpression(constant)
	require.NoError(t, err)
	assert.NotNil(t, vr1)

	count1 := len(selector.GetInstructions())

	// Second call - should reuse cached result
	vr2, err := ctx.selectExpression(constant)
	require.NoError(t, err)
	assert.NotNil(t, vr2)
	assert.Equal(t, vr1, vr2, "Should return same VirtualRegister")

	count2 := len(selector.GetInstructions())
	assert.Equal(t, count1, count2, "Should not generate additional instructions")
}

// Test selectSymbolRef
func Test_InstructionSelection_SymbolRef(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	// Create a variable
	symbol := &zsm.Symbol{
		Name: "x",
		Type: u8Type(),
	}
	expectedVR := ctx.vrAlloc.AllocateNamed("x", 8)
	ctx.symbolToVReg[symbol] = expectedVR

	symbolRef := &zsm.SemSymbolRef{
		Symbol: symbol,
	}

	vr, err := ctx.selectSymbolRef(symbolRef)

	require.NoError(t, err)
	assert.Equal(t, expectedVR, vr)
}

// Test selectSymbolRef with undefined variable
func Test_InstructionSelection_SymbolRef_Undefined(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol := &zsm.Symbol{
		Name: "undefined",
		Type: u8Type(),
	}

	symbolRef := &zsm.SemSymbolRef{
		Symbol: symbol,
	}

	_, err := ctx.selectSymbolRef(symbolRef)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined variable")
}

// Test selectFunction with parameters
func Test_InstructionSelection_Function_WithParameters(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	param1 := &zsm.Symbol{Name: "a", Type: u8Type()}
	param2 := &zsm.Symbol{Name: "b", Type: u8Type()}

	fn := &zsm.SemFunctionDecl{
		Name:       "add",
		Parameters: []*zsm.Symbol{param1, param2},
		ReturnType: u8Type(),
		Body: &zsm.SemBlock{
			Statements: []zsm.SemStatement{
				&zsm.SemReturn{
					Value: &zsm.SemBinaryOp{
						Op:       zsm.OpAdd,
						Left:     &zsm.SemSymbolRef{Symbol: param1},
						Right:    &zsm.SemSymbolRef{Symbol: param2},
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

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test SelectInstructions with full compilation unit
func Test_SelectInstructions_Simple(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)

	fn := &zsm.SemFunctionDecl{
		Name:       "test",
		Parameters: []*zsm.Symbol{},
		ReturnType: u8Type(),
		Body: &zsm.SemBlock{
			Statements: []zsm.SemStatement{
				&zsm.SemReturn{
					Value: &zsm.SemConstant{Value: 42, TypeInfo: u8Type()},
				},
			},
		},
	}

	compilationUnit := &zsm.SemCompilationUnit{
		Declarations: []zsm.SemDeclaration{fn},
	}

	err := SelectInstructions(compilationUnit, selector, cc)

	require.NoError(t, err)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test complex expression with nested operations
func Test_InstructionSelection_ComplexExpression(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

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

	vr, err := ctx.selectExpression(expr)

	require.NoError(t, err)
	assert.NotNil(t, vr)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
	// Should have multiple instructions for nested operations
	assert.GreaterOrEqual(t, len(instructions), 5)
}

// Test multiple variable declarations
func Test_InstructionSelection_MultipleVariables(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol1 := &zsm.Symbol{Name: "x", Type: u8Type()}
	symbol2 := &zsm.Symbol{Name: "y", Type: u8Type()}

	decl1 := &zsm.SemVariableDecl{
		Symbol:      symbol1,
		Initializer: &zsm.SemConstant{Value: 10, TypeInfo: u8Type()},
		TypeInfo:    u8Type(),
	}

	decl2 := &zsm.SemVariableDecl{
		Symbol:      symbol2,
		Initializer: &zsm.SemConstant{Value: 20, TypeInfo: u8Type()},
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

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}

// Test variable declaration without initializer
func Test_InstructionSelection_VariableDecl_NoInitializer(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	symbol := &zsm.Symbol{Name: "x", Type: u8Type()}

	decl := &zsm.SemVariableDecl{
		Symbol:      symbol,
		Initializer: nil,
		TypeInfo:    u8Type(),
	}

	err := ctx.selectVariableDecl(decl)

	require.NoError(t, err)

	// Variable should still be allocated
	vr, ok := ctx.symbolToVReg[symbol]
	assert.True(t, ok)
	assert.NotNil(t, vr)
}

// Test 16-bit operations
func Test_InstructionSelection_16BitOperations(t *testing.T) {
	cc := NewZ80CallingConvention()
	selector := NewZ80InstructionSelector(cc)
	ctx := NewInstructionSelectionContext(selector, cc)

	binaryOp := &zsm.SemBinaryOp{
		Op:       zsm.OpAdd,
		Left:     &zsm.SemConstant{Value: 1000, TypeInfo: u16Type()},
		Right:    &zsm.SemConstant{Value: 2000, TypeInfo: u16Type()},
		TypeInfo: u16Type(),
	}

	vr, err := ctx.selectBinaryOp(binaryOp)

	require.NoError(t, err)
	assert.NotNil(t, vr)
	assert.Equal(t, 16, vr.Size)

	// Check that instructions were generated
	instructions := selector.GetInstructions()
	assert.NotEmpty(t, instructions)
}
