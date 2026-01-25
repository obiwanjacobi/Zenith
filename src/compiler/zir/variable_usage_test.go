package zir

import (
	"testing"

	"zenith/compiler/lexer"
	"zenith/compiler/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_VariableUsage_Arithmetic(t *testing.T) {
	code := `main: () {
		x: u8 = 5
		y: u8 = 10
		z: u8 = x + y
	}`

	tokens := lexer.OpenTokenStream(code)
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors))

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	analyzer := NewSemanticAnalyzer()
	irCU, irErrors := analyzer.Analyze(cu)
	requireNoErrors(t, irErrors)

	// Get the function declaration
	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
	require.True(t, ok)

	// Get symbols from function body statements
	xDecl, ok := funcDecl.Body.Statements[0].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "x", xDecl.Symbol.Name)
	assert.Equal(t, VariableUsageArithmetic, xDecl.Symbol.Usage, "x should be marked as arithmetic")

	yDecl, ok := funcDecl.Body.Statements[1].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "y", yDecl.Symbol.Name)
	assert.Equal(t, VariableUsageArithmetic, yDecl.Symbol.Usage, "y should be marked as arithmetic")

	zDecl, ok := funcDecl.Body.Statements[2].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "z", zDecl.Symbol.Name)
	// z is defined but not used in arithmetic, should be general
	assert.Equal(t, VariableUsageGeneral, zDecl.Symbol.Usage, "z should be general (not used)")
}

func Test_VariableUsage_Counter(t *testing.T) {
	code := `main: () {
		for i: u8 = 0; i < 10; i = i + 1 {
			x: u8 = 5
		}
	}`

	tokens := lexer.OpenTokenStream(code)
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors))

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	analyzer := NewSemanticAnalyzer()
	irCU, irErrors := analyzer.Analyze(cu)
	requireNoErrors(t, irErrors)

	// Get the function declaration
	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
	require.True(t, ok)

	// Get the for loop statement
	forStmt, ok := funcDecl.Body.Statements[0].(*IRFor)
	require.True(t, ok)

	// Get the loop variable from initializer
	varDecl, ok := forStmt.Initializer.(*IRVariableDecl)
	require.True(t, ok)

	// Check that i is marked as counter
	assert.Equal(t, VariableUsageCounter, varDecl.Symbol.Usage, "i should be marked as counter")
}

func Test_VariableUsage_Pointer(t *testing.T) {
	t.Skip("Pointers are not implemented in the grammar yet")

	code := `
	struct Point {
		x: u8,
		y: u8
	}
	main: () {
		p: Point = Point{x= 5, y= 10}
		val: Point* = &p
	}`

	tokens := lexer.OpenTokenStream(code)
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors), "Parse errors: %v", parseErrors)

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	analyzer := NewSemanticAnalyzer()
	irCU, irErrors := analyzer.Analyze(cu)
	require.Equal(t, 0, len(irErrors), "Expected no IR errors: %v", irErrors)

	// Get the function declaration (second declaration after Point type)
	require.Greater(t, len(irCU.Declarations), 1, "Expected at least 2 declarations")
	funcDecl, ok := irCU.Declarations[1].(*IRFunctionDecl)
	require.True(t, ok)

	// Find the p symbol in the function's scope
	require.Greater(t, len(funcDecl.Body.Statements), 0, "Expected at least 1 statement")
	varDecl, ok := funcDecl.Body.Statements[0].(*IRVariableDecl)
	require.True(t, ok)
	require.Equal(t, "p", varDecl.Symbol.Name)

	// Check that p is marked as pointer usage (used in member access)
	assert.Equal(t, VariableUsagePointer, varDecl.Symbol.Usage, "p should be marked as pointer (member access)")
}

func Test_VariableUsage_General(t *testing.T) {
	code := `main: () {
		x: u8 = 5
	}`

	tokens := lexer.OpenTokenStream(code)
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors))

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	analyzer := NewSemanticAnalyzer()
	irCU, irErrors := analyzer.Analyze(cu)
	requireNoErrors(t, irErrors)

	// Get the function declaration
	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
	require.True(t, ok)

	// Check that x has no specific usage (just declared, not used)
	xDecl, ok := funcDecl.Body.Statements[0].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "x", xDecl.Symbol.Name)
	assert.Equal(t, VariableUsageGeneral, xDecl.Symbol.Usage, "x should be general (declared but not used)")
}

func Test_VariableUsage_MultipleArithmetic(t *testing.T) {
	code := `main: () {
		a: u8 = 1
		b: u8 = 2
		c: u8 = 3
		result: u8 = a + b * c
	}`

	tokens := lexer.OpenTokenStream(code)
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors))

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	analyzer := NewSemanticAnalyzer()
	irCU, irErrors := analyzer.Analyze(cu)
	requireNoErrors(t, irErrors)

	// Get the function declaration
	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
	require.True(t, ok)

	// All three variables used in arithmetic should be marked
	aDecl, ok := funcDecl.Body.Statements[0].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "a", aDecl.Symbol.Name)
	assert.Equal(t, VariableUsageArithmetic, aDecl.Symbol.Usage)

	bDecl, ok := funcDecl.Body.Statements[1].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "b", bDecl.Symbol.Name)
	assert.Equal(t, VariableUsageArithmetic, bDecl.Symbol.Usage)

	cDecl, ok := funcDecl.Body.Statements[2].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "c", cDecl.Symbol.Name)
	assert.Equal(t, VariableUsageArithmetic, cDecl.Symbol.Usage)
}
