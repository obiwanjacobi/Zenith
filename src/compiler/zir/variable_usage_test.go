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
	semCU, semErrors := analyzer.Analyze(cu)
	requireNoErrors(t, semErrors)

	// Get the function declaration
	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok)

	// Get symbols from function body statements
	xDecl, ok := funcDecl.Body.Statements[0].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "x", xDecl.Symbol.Name)
	assert.True(t, xDecl.Symbol.Usage.HasFlag(VarUsedArithmetic), "x should be marked as used in arithmetic")

	yDecl, ok := funcDecl.Body.Statements[1].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "y", yDecl.Symbol.Name)
	assert.True(t, yDecl.Symbol.Usage.HasFlag(VarUsedArithmetic), "y should be marked as used in arithmetic")

	zDecl, ok := funcDecl.Body.Statements[2].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "z", zDecl.Symbol.Name)
	// z is defined but not used in arithmetic
	assert.False(t, zDecl.Symbol.Usage.HasFlag(VarUsedArithmetic), "z should not be marked as used in arithmetic")
	// z is initialized with arithmetic
	assert.True(t, zDecl.Symbol.Usage.HasFlag(VarInitArithmetic), "z should be marked as initialized with arithmetic")
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
	semCU, semErrors := analyzer.Analyze(cu)
	requireNoErrors(t, semErrors)

	// Get the function declaration
	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok)

	// Get the for loop statement
	forStmt, ok := funcDecl.Body.Statements[0].(*SemFor)
	require.True(t, ok)

	// Get the loop variable from initializer
	varDecl, ok := forStmt.Initializer.(*SemVariableDecl)
	require.True(t, ok)

	// Check that i is marked as counter
	assert.True(t, varDecl.Symbol.Usage.HasFlag(VarInitCounter), "i should be marked as initialized as counter")
	assert.True(t, varDecl.Symbol.Usage.HasFlag(VarUsedCounter), "i should be marked as used as counter")
}

func Test_VariableUsage_MemberAccessPointer(t *testing.T) {

	code := `
	struct Point {
		x: u8,
		y: u8
	}
	main: () {
		p: Point = Point{x= 5, y= 10}
		val: u8 = p.x
	}`

	tokens := lexer.OpenTokenStream(code)
	astNode, parseErrors := parser.Parse("test", tokens)
	require.NotNil(t, astNode)
	require.Equal(t, 0, len(parseErrors), "Parse errors: %v", parseErrors)

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok)

	analyzer := NewSemanticAnalyzer()
	semCU, semErrors := analyzer.Analyze(cu)
	require.Equal(t, 0, len(semErrors), "Expected no IR errors: %v", semErrors)

	// Get the function declaration (second declaration after Point type)
	require.Greater(t, len(semCU.Declarations), 1, "Expected at least 2 declarations")
	funcDecl, ok := semCU.Declarations[1].(*SemFunctionDecl)
	require.True(t, ok)

	// Find the p symbol in the function's scope
	require.Greater(t, len(funcDecl.Body.Statements), 0, "Expected at least 1 statement")
	varDecl, ok := funcDecl.Body.Statements[0].(*SemVariableDecl)
	require.True(t, ok)
	require.Equal(t, "p", varDecl.Symbol.Name)

	// Check that p is marked as pointer usage (used in member access)
	assert.True(t, varDecl.Symbol.Usage.HasFlag(VarUsedPointer), "p should be marked as used as pointer (member access)")
	// p is initialized with struct initializer
	assert.True(t, varDecl.Symbol.Usage.HasFlag(VarInitPointer), "p should be marked as initialized with struct")
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
	semCU, semErrors := analyzer.Analyze(cu)
	requireNoErrors(t, semErrors)

	// Get the function declaration
	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok)

	// Check that x has no specific usage (just declared, not used)
	xDecl, ok := funcDecl.Body.Statements[0].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "x", xDecl.Symbol.Name)
	assert.False(t, xDecl.Symbol.Usage.HasFlag(VarUsedArithmetic), "x should not be used in arithmetic")
	assert.True(t, xDecl.Symbol.Usage.HasFlag(VarInitConstant), "x should be initialized with constant")
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
	semCU, semErrors := analyzer.Analyze(cu)
	requireNoErrors(t, semErrors)

	// Get the function declaration
	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok)

	// All three variables used in arithmetic should be marked
	aDecl, ok := funcDecl.Body.Statements[0].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "a", aDecl.Symbol.Name)
	assert.True(t, aDecl.Symbol.Usage.HasFlag(VarUsedArithmetic), "a should be used in arithmetic")

	bDecl, ok := funcDecl.Body.Statements[1].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "b", bDecl.Symbol.Name)
	assert.True(t, bDecl.Symbol.Usage.HasFlag(VarUsedArithmetic), "b should be used in arithmetic")

	cDecl, ok := funcDecl.Body.Statements[2].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "c", cDecl.Symbol.Name)
	assert.True(t, cDecl.Symbol.Usage.HasFlag(VarUsedArithmetic), "c should be used in arithmetic")
}

