package cfg

import (
	"testing"

	"zenith/compiler"
	"zenith/compiler/lexer"
	"zenith/compiler/parser"
	"zenith/compiler/zsm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test helpers
// ============================================================================

// lowerTACFromCode builds a CFG from source, runs TAC lowering, and returns it.
// Uses Z80 register sets so AllowedSet is never nil.
func lowerTACFromCode(t *testing.T, code string) *CFG {
	t.Helper()
	fnCFG := buildCFGFromCode(t, code)
	alloc := &TempVRAllocator{}
	regsets := RegisterSets{Regs8: Z80Registers8, Regs16: Z80Registers16}
	require.NoError(t, LowerTAC(fnCFG, alloc, regsets))
	return fnCFG
}

// buildAllLoweredCFGs parses, analyzes, builds CFGs for all functions, and
// lowers TAC on each. Returns one lowered CFG per function in source order.
func buildAllLoweredCFGs(t *testing.T, code string) []*CFG {
	t.Helper()
	tokens := lexer.OpenTokenStream(code)
	ast, parseErrs := parser.Parse(&compiler.Source{Name: "test"}, tokens)
	require.Empty(t, parseErrs, "parse errors: %v", parseErrs)
	cu, ok := ast.(parser.CompilationUnit)
	require.True(t, ok)
	analyzer := zsm.NewSemanticAnalyzer()
	semCU, semErrs := analyzer.Analyze(cu)
	require.Empty(t, semErrs, "sem errors: %v", semErrs)

	cfgs := BuildCFGs(semCU)
	alloc := &TempVRAllocator{}
	regsets := RegisterSets{Regs8: Z80Registers8, Regs16: Z80Registers16}
	for _, fnCFG := range cfgs {
		require.NoError(t, LowerTAC(fnCFG, alloc, regsets))
	}
	return cfgs
}

// firstBodyTAC returns the TAC from the first LabelFunction block that has any,
// falling back to any non-entry/exit block with TAC.
func firstBodyTAC(t *testing.T, fnCFG *CFG) []TacInstruction {
	t.Helper()
	for _, block := range fnCFG.Blocks {
		if block.Label == LabelFunction && len(block.TAC) > 0 {
			return block.TAC
		}
	}
	for _, block := range fnCFG.Blocks {
		if block.Label != LabelEntry && block.Label != LabelExit && len(block.TAC) > 0 {
			return block.TAC
		}
	}
	return nil
}

// allTACByLabel collects TAC from all blocks with the given label.
func allTACByLabel(fnCFG *CFG, label BlockLabel) []TacInstruction {
	var result []TacInstruction
	for _, block := range fnCFG.Blocks {
		if block.Label == label {
			result = append(result, block.TAC...)
		}
	}
	return result
}

// findFirst returns the first TAC instruction of type T in the slice, or nil.
func findFirst[T TacInstruction](tac []TacInstruction) T {
	for _, instr := range tac {
		if v, ok := instr.(T); ok {
			return v
		}
	}
	var zero T
	return zero
}

// ============================================================================
// 1. Scalar var decl with constant initialiser → TacCopy(ImmVR)
// ============================================================================

func TestTACLowering_ScalarDeclConstant(t *testing.T) {
	code := `main: () {
		x: u8 = 5
	}`
	fnCFG := lowerTACFromCode(t, code)
	tac := firstBodyTAC(t, fnCFG)
	require.NotEmpty(t, tac)

	cp := findFirst[*TacCopy](tac)
	require.NotNil(t, cp, "expected TacCopy for scalar initialiser")
	assert.Equal(t, uint8(8), cp.Size)

	imm, ok := cp.Src.(*ImmVR)
	require.True(t, ok, "source of copy should be ImmVR, got %T", cp.Src)
	assert.Equal(t, int32(5), imm.Value)
}

// ============================================================================
// 2. Binary-op initialiser → TacBinOp then TacCopy sourcing its result
// ============================================================================

func TestTACLowering_ScalarDeclBinaryOp(t *testing.T) {
	code := `main: () {
		a: u8 = 3
		b: u8 = 4
		c: u8 = a + b
	}`
	fnCFG := lowerTACFromCode(t, code)
	tac := firstBodyTAC(t, fnCFG)
	require.NotEmpty(t, tac)

	addOp := findFirst[*TacBinOp](tac)
	require.NotNil(t, addOp, "expected TacBinOp for a+b")
	assert.Equal(t, TacAdd, addOp.Op)
	assert.Equal(t, uint8(8), addOp.Size)

	// There must be a TacCopy that sources the add result (the binding for c).
	var copyC *TacCopy
	for _, instr := range tac {
		if cp, ok := instr.(*TacCopy); ok && cp.Src == addOp.Dst {
			copyC = cp
			break
		}
	}
	require.NotNil(t, copyC, "expected TacCopy sourcing the add result")
}

// ============================================================================
// 3. If-condition with comparison → TacBranchCond (BranchMode, no TacCompare)
// ============================================================================

func TestTACLowering_IfBranchModeComparison(t *testing.T) {
	code := `main: () {
		x: u8 = 5
		if x < 10 {
			x = 0
		}
	}`
	fnCFG := lowerTACFromCode(t, code)

	// The SemIf sentinel lives in the function body block; look there.
	condTAC := allTACByLabel(fnCFG, LabelFunction)

	bc := findFirst[*TacBranchCond](condTAC)
	require.NotNil(t, bc, "expected TacBranchCond in condition block")
	assert.Equal(t, TacCmpLess, bc.Op)
	assert.Equal(t, uint8(8), bc.Size)

	// BranchMode must not materialise a boolean into a VR.
	for _, instr := range condTAC {
		_, isCmp := instr.(*TacCompare)
		assert.False(t, isCmp, "BranchMode must not emit TacCompare")
	}
}

// ============================================================================
// 4. Comparison stored in a variable → TacCompare (ValueMode)
// ============================================================================

func TestTACLowering_ComparisonValueMode(t *testing.T) {
	code := `main: () {
		x: u8 = 5
		result: bit = x < 10
	}`
	fnCFG := lowerTACFromCode(t, code)
	tac := firstBodyTAC(t, fnCFG)

	cmp := findFirst[*TacCompare](tac)
	require.NotNil(t, cmp, "expected TacCompare for stored comparison")
	assert.Equal(t, TacCmpLess, cmp.Op)
	assert.Equal(t, uint8(8), cmp.Size)       // u8 operands
	assert.Equal(t, uint8(8), cmp.Dst.Size()) // bit result is 1 byte
}

// ============================================================================
// 5. Array var decl with literal → TacStackAddr + TacInitSeq
// ============================================================================

func TestTACLowering_ArrayInitSeq(t *testing.T) {
	code := `main: () {
		arr: u8[3] = [1, 2, 3]
	}`
	fnCFG := lowerTACFromCode(t, code)
	tac := firstBodyTAC(t, fnCFG)
	require.NotEmpty(t, tac)

	// First instruction: TacStackAddr.
	stackAddr, ok := tac[0].(*TacStackAddr)
	require.True(t, ok, "first TAC must be TacStackAddr, got %T", tac[0])

	// Second instruction: TacInitSeq referencing the same base VR.
	require.True(t, len(tac) >= 2, "expected at least 2 TAC instructions")
	initSeq, ok := tac[1].(*TacInitSeq)
	require.True(t, ok, "second TAC must be TacInitSeq, got %T", tac[1])

	assert.Equal(t, stackAddr.Dst, initSeq.Base, "INIT_SEQ base must be STACK_ADDR result")
	assert.Equal(t, uint8(1), initSeq.ElemSize, "element size should be 1 (u8)")
	require.Equal(t, 3, len(initSeq.Values), "expected 3 elements")

	for i, v := range initSeq.Values {
		imm, ok := v.(*ImmVR)
		require.True(t, ok, "element %d should be ImmVR", i)
		assert.Equal(t, int32(i+1), imm.Value)
	}

	assert.False(t, fnCFG.StackFrame.IsEmpty(), "stack frame must have a slot for arr")
}

// ============================================================================
// 6. Indexed assignment → TacStoreIndexed
// ============================================================================

func TestTACLowering_IndexedAssignment(t *testing.T) {
	code := `main: () {
		arr: u8[3] = [0, 0, 0]
		arr[1] = 42
	}`
	fnCFG := lowerTACFromCode(t, code)
	tac := firstBodyTAC(t, fnCFG)

	si := findFirst[*TacStoreIndexed](tac)
	require.NotNil(t, si, "expected TacStoreIndexed")
	assert.Equal(t, uint8(1), si.ElemSize, "element size should be 1 (u8)")

	imm, ok := si.Value.(*ImmVR)
	require.True(t, ok, "stored value should be ImmVR, got %T", si.Value)
	assert.Equal(t, int32(42), imm.Value)
}

// ============================================================================
// 7. Return statement → TacReturn with non-nil Value
// ============================================================================

func TestTACLowering_ReturnValue(t *testing.T) {
	code := `calc: (x: u8) u8 {
		ret x
	}`
	fnCFG := lowerTACFromCode(t, code)

	var ret *TacReturn
	for _, block := range fnCFG.Blocks {
		for _, instr := range block.TAC {
			if r, ok := instr.(*TacReturn); ok {
				ret = r
				break
			}
		}
		if ret != nil {
			break
		}
	}
	require.NotNil(t, ret, "expected TacReturn")
	require.NotNil(t, ret.Value, "TacReturn must carry the return value")
}

// ============================================================================
// 8. Non-void function call → TacCall with non-nil Dst
// ============================================================================

func TestTACLowering_FunctionCallNonVoid(t *testing.T) {
	code := `
add: (a: u8, b: u8) u8 {
	ret a
}
main: () {
	r: u8 = add(3, 4)
}`
	cfgs := buildAllLoweredCFGs(t, code)
	require.Equal(t, 2, len(cfgs), "expected 2 functions")

	mainCFG := cfgs[1]
	var call *TacCall
	for _, block := range mainCFG.Blocks {
		if c := findFirst[*TacCall](block.TAC); c != nil {
			call = c
			break
		}
	}
	require.NotNil(t, call, "expected TacCall in main")
	assert.Equal(t, "add", call.Fn)
	assert.Equal(t, 2, len(call.Args))
	require.NotNil(t, call.Dst, "non-void call must have a Dst VR")
	assert.Equal(t, uint8(8), call.Dst.Size())
}

// ============================================================================
// 9. Void function call → TacCall with nil Dst
// ============================================================================

func TestTACLowering_VoidCallNilDst(t *testing.T) {
	code := `
drain: (v: u8) {
}
caller: () {
	drain(5)
}`
	cfgs := buildAllLoweredCFGs(t, code)
	require.Equal(t, 2, len(cfgs), "expected 2 functions")

	mainCFG := cfgs[1]
	var call *TacCall
	for _, block := range mainCFG.Blocks {
		if c := findFirst[*TacCall](block.TAC); c != nil {
			call = c
			break
		}
	}
	require.NotNil(t, call, "expected TacCall in main")
	assert.Equal(t, "drain", call.Fn)
	assert.Nil(t, call.Dst, "void call must have nil Dst")
}

// ============================================================================
// 10. Logical-not in branch → then/else swapped, no TacUnary emitted
// ============================================================================

// TODO: Disabled — blocked by two bugs tracked in src/todo.md:
//   1. Parser greedily parses `flag {` as an expressionTypeInitializer (struct
//      literal), consuming the if-body before the code block is recognised.
//   2. sem_analyzer.go does not handle parser.ExpressionPrecedence — `not (x < 10)`
//      falls through to the default case and returns an "unknown expression type" error.
// Fix both bugs, then rewrite this test to use `if not (x < 10) {}` directly.
//
// func TestTACLowering_LogicalNotBranchInversion(t *testing.T) {
// 	code := `main: () {
// 		x: u8 = 5
// 		if not (x < 10) {
// 			x = 1
// 		}
// 	}`
// 	fnCFG := lowerTACFromCode(t, code)
//
// 	// Must have a TacBranchCond — not (comparison) must be fused, not materialised.
// 	var bc *TacBranchCond
// 	for _, block := range fnCFG.Blocks {
// 		if v := findFirst[*TacBranchCond](block.TAC); v != nil {
// 			bc = v
// 			break
// 		}
// 	}
// 	require.NotNil(t, bc, "expected TacBranchCond even for negated condition")
//
// 	// Logical-not inversion is structural (then/else swap); no TacUnary must
// 	// appear as a NOT instruction.
// 	for _, block := range fnCFG.Blocks {
// 		for _, instr := range block.TAC {
// 			if u, ok := instr.(*TacUnary); ok {
// 				assert.NotEqual(t, TacBitwiseNot, u.Op, "logical-not in branch must not emit TacUnary(BNOT)")
// 				assert.NotEqual(t, TacNegate, u.Op, "logical-not in branch must not emit TacUnary(NEG)")
// 			}
// 		}
// 	}
// }

// ============================================================================
// 11. For-loop condition → TacBranchCond with correct then/else block labels
// ============================================================================

func TestTACLowering_ForLoopCondition(t *testing.T) {
	code := `main: () {
		i: u8 = 0
		for i < 10 {
			i = i + 1
		}
	}`
	fnCFG := lowerTACFromCode(t, code)

	condTAC := allTACByLabel(fnCFG, LabelForCond)
	bc := findFirst[*TacBranchCond](condTAC)
	require.NotNil(t, bc, "for-condition block must emit TacBranchCond")
	assert.Equal(t, TacCmpLess, bc.Op)

	require.NotNil(t, bc.Then)
	require.NotNil(t, bc.Else)
	assert.Equal(t, LabelForBody, bc.Then.Label, "Then branch must go to loop body")
	assert.Equal(t, LabelForExit, bc.Else.Label, "Else branch must go to loop exit")
}
