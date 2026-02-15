package zsm

import (
	"fmt"
	"testing"

	"zenith/compiler"
	"zenith/compiler/lexer"
	"zenith/compiler/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to parse code and run semantic analysis
func analyzeCode(t *testing.T, testName string, code string) (*SemCompilationUnit, []*compiler.Diagnostic) {
	// Tokenize
	tokens := lexer.OpenTokenStream(code)

	// Parse
	astNode, parseErrors := parser.Parse(testName, tokens)
	require.NotNil(t, astNode, "Parser should return a node")
	require.Equal(t, 0, len(parseErrors), fmt.Sprintf("Parser errors: %v", parseErrors))

	cu, ok := astNode.(parser.CompilationUnit)
	require.True(t, ok, "Root node should be CompilationUnit")

	// Analyze
	analyzer := NewSemanticAnalyzer()
	semCU, semErrors := analyzer.Analyze(cu)

	return semCU, semErrors
}

// Helper function to require no errors
func requireNoErrors(t *testing.T, errors []*compiler.Diagnostic) {
	if len(errors) > 0 {
		for _, err := range errors {
			t.Log(err.Error())
		}
	}
	require.Equal(t, 0, len(errors), "Expected no IR errors")
}

// ============================================================================
// Variable Declaration Tests
// ============================================================================

func Test_Analyze_VarDeclWithType(t *testing.T) {
	code := "count: u8"
	semCU, errors := analyzeCode(t, "Test_Analyze_VarDeclWithType", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(semCU.Declarations))

	varDecl, ok := semCU.Declarations[0].(*SemVariableDecl)
	require.True(t, ok, "Declaration should be SemVariableDecl")
	assert.Equal(t, "count", varDecl.Symbol.Name)
	assert.Equal(t, U8Type, varDecl.Symbol.Type)
	assert.Nil(t, varDecl.Initializer)
}

func Test_Analyze_VarDeclWithTypeAndInit(t *testing.T) {
	code := "count: u16 = 42"
	semCU, errors := analyzeCode(t, "Test_Analyze_VarDeclWithTypeAndInit", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(semCU.Declarations))

	varDecl, ok := semCU.Declarations[0].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "count", varDecl.Symbol.Name)
	assert.Equal(t, U16Type, varDecl.Symbol.Type)
	assert.NotNil(t, varDecl.Initializer)

	// Check initializer is a constant
	constant, ok := varDecl.Initializer.(*SemConstant)
	assert.True(t, ok, "Initializer should be SemConstant")
	assert.Equal(t, U8Type, constant.Type()) // TODO: Should parse as correct type
}

func Test_Analyze_VarDeclInferred(t *testing.T) {
	code := "value: = 100"
	semCU, errors := analyzeCode(t, "Test_Analyze_VarDeclInferred", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(semCU.Declarations))

	varDecl, ok := semCU.Declarations[0].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "value", varDecl.Symbol.Name)
	assert.NotNil(t, varDecl.Initializer)
	// Type should be inferred from initializer
	assert.Equal(t, varDecl.Initializer.Type(), varDecl.Symbol.Type)
}

func Test_Analyze_VarDeclInferredNoInit_Error(t *testing.T) {
	// Note: The parser already rejects "noInit:" (no type, no initializer)
	// So we test a different error case - this test is now for duplicate declaration
	code := "value: u8\nvalue: u16"
	_, errors := analyzeCode(t, "Test_Analyze_VarDeclInferredNoInit_Error", code)

	// Should have error for duplicate declaration
	require.Greater(t, len(errors), 0, "Expected at least one error")
	assert.Contains(t, errors[0].Error(), "already declared")
}

func Test_Analyze_VarDeclDuplicate_Error(t *testing.T) {
	code := "x: u8\nx: u16"
	_, errors := analyzeCode(t, "Test_Analyze_VarDeclDuplicate_Error", code)

	require.Greater(t, len(errors), 0, "Expected error for duplicate declaration")
	assert.Contains(t, errors[0].Error(), "already declared")
}

func Test_Analyze_VarDeclUndefinedType_Error(t *testing.T) {
	code := "value: UnknownType"
	_, errors := analyzeCode(t, "Test_Analyze_VarDeclUndefinedType_Error", code)

	require.Greater(t, len(errors), 0, "Expected error for undefined type")
	assert.Contains(t, errors[0].Error(), "undefined type")
}

// ============================================================================
// Function Declaration Tests
// ============================================================================

func Test_Analyze_FunctionDeclaration(t *testing.T) {
	code := `main: () {
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_FunctionDeclaration", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(semCU.Declarations))

	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok, "Declaration should be SemFunctionDecl")
	assert.Equal(t, "main", funcDecl.Name)
	assert.Equal(t, 0, len(funcDecl.Parameters))
	assert.Nil(t, funcDecl.ReturnType)
	assert.NotNil(t, funcDecl.Body)
	assert.NotNil(t, funcDecl.Scope)
}

func Test_Analyze_FunctionWithParameters(t *testing.T) {
	code := `add: (a: u8, b: u8) {
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_FunctionWithParameters", code)
	requireNoErrors(t, errors)

	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok)
	assert.Equal(t, "add", funcDecl.Name)
	assert.Equal(t, 2, len(funcDecl.Parameters))

	assert.Equal(t, "a", funcDecl.Parameters[0].Name)
	assert.Equal(t, U8Type, funcDecl.Parameters[0].Type)

	assert.Equal(t, "b", funcDecl.Parameters[1].Name)
	assert.Equal(t, U8Type, funcDecl.Parameters[1].Type)
}

func Test_Analyze_FunctionWithReturnType(t *testing.T) {
	code := `getValue: () u16 {
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_FunctionWithReturnType", code)
	requireNoErrors(t, errors)

	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok)
	assert.Equal(t, "getValue", funcDecl.Name)
	assert.Equal(t, U16Type, funcDecl.ReturnType)
}

func Test_Analyze_FunctionWithLocalVar(t *testing.T) {
	// Note: Local variables with explicit types are not pre-registered in pass 1,
	// so they need to be inferred or we need to rework the two-pass approach
	code := `main: () {
		local: = 5
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_FunctionWithLocalVar", code)
	requireNoErrors(t, errors)

	funcDecl, ok := semCU.Declarations[0].(*SemFunctionDecl)
	require.True(t, ok)
	require.Equal(t, 1, len(funcDecl.Body.Statements))

	varDecl, ok := funcDecl.Body.Statements[0].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "local", varDecl.Symbol.Name)
	assert.Equal(t, U8Type, varDecl.Symbol.Type)
}

func Test_Analyze_FunctionDuplicate_Error(t *testing.T) {
	code := "myFunc: () {\n\t}\n\tmyFunc: () {\n\t}"
	_, errors := analyzeCode(t, "Test_Analyze_FunctionDuplicate_Error", code)

	require.Greater(t, len(errors), 0, "Expected error for duplicate function")
	assert.Contains(t, errors[0].Error(), "already declared")
}

// ============================================================================
// Type Declaration Tests
// ============================================================================

func Test_Analyze_TypeDeclaration(t *testing.T) {
	code := `struct Point {
		x: u8,
		y: u8
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_TypeDeclaration", code)

	// Debug: print errors
	for _, err := range errors {
		t.Logf("Error: %s", err.Error())
	}
	t.Logf("Number of declarations: %d", len(semCU.Declarations))

	requireNoErrors(t, errors)

	require.Equal(t, 1, len(semCU.Declarations))

	typeDecl, ok := semCU.Declarations[0].(*SemTypeDecl)
	require.True(t, ok, "Declaration should be SemTypeDecl")
	assert.Equal(t, "Point", typeDecl.TypeInfo.Name())

	fields := typeDecl.TypeInfo.Fields()
	require.Equal(t, 2, len(fields))
	assert.Equal(t, "x", fields[0].Name)
	assert.Equal(t, U8Type, fields[0].Type)
	assert.Equal(t, "y", fields[1].Name)
	assert.Equal(t, U8Type, fields[1].Type)
}

func Test_Analyze_TypeDeclarationUsage(t *testing.T) {
	code := `struct Point {
		x: u8,
		y: u8
	}
	origin: Point`
	semCU, errors := analyzeCode(t, "Test_Analyze_TypeDeclarationUsage", code)
	requireNoErrors(t, errors)

	require.Equal(t, 2, len(semCU.Declarations))

	varDecl, ok := semCU.Declarations[1].(*SemVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "origin", varDecl.Symbol.Name)

	structType, ok := varDecl.Symbol.Type.(*StructType)
	require.True(t, ok, "Variable type should be StructType")
	assert.Equal(t, "Point", structType.Name())
}

// ============================================================================
// Statement Tests
// ============================================================================

func Test_Analyze_Assignment(t *testing.T) {
	code := `main: () {
		x: = 10
		x = 20
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_Assignment", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 2, len(funcDecl.Body.Statements))

	assignment, ok := funcDecl.Body.Statements[1].(*SemAssignment)
	require.True(t, ok, "Second statement should be SemAssignment")
	assert.Equal(t, "x", assignment.Target.Name)
	assert.NotNil(t, assignment.Value)
}

func Test_Analyze_AssignmentUndefined_Error(t *testing.T) {
	code := `main: () {
		x = 20
	}`
	_, errors := analyzeCode(t, "Test_Analyze_AssignmentUndefined_Error", code)

	require.Greater(t, len(errors), 0, "Expected error for undefined variable")
	assert.Contains(t, errors[0].Error(), "undefined variable")
}

// ============================================================================
// If, Elfis and Else Tests
// ============================================================================

func Test_Analyze_IfStatement(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_IfStatement", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 1, len(funcDecl.Body.Statements))

	ifStmt, ok := funcDecl.Body.Statements[0].(*SemIf)
	require.True(t, ok, "Statement should be SemIf")
	assert.NotNil(t, ifStmt.Condition)
	assert.NotNil(t, ifStmt.ThenBlock)
	assert.Equal(t, 1, len(ifStmt.ThenBlock.Statements))
}

func Test_Analyze_IfElseStatement(t *testing.T) {
	code := `main: () {
		if false {
			x: = 1
		} else {
			y: = 2
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_IfElseStatement", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	ifStmt := funcDecl.Body.Statements[0].(*SemIf)

	assert.NotNil(t, ifStmt.ThenBlock)
	assert.NotNil(t, ifStmt.ElseBlock)
	assert.Equal(t, 1, len(ifStmt.ThenBlock.Statements))
	assert.Equal(t, 1, len(ifStmt.ElseBlock.Statements))
}

func Test_Analyze_IfElsifStatement(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		} elsif false {
			y: = 2
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_IfElsifStatement", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	ifStmt := funcDecl.Body.Statements[0].(*SemIf)

	assert.NotNil(t, ifStmt.ThenBlock)
	assert.Equal(t, 1, len(ifStmt.ThenBlock.Statements))
	assert.Equal(t, 1, len(ifStmt.ElsifBlocks))
	assert.Nil(t, ifStmt.ElseBlock)

	elsif := ifStmt.ElsifBlocks[0]
	assert.NotNil(t, elsif.Condition)
	assert.NotNil(t, elsif.ThenBlock)
	assert.Equal(t, 1, len(elsif.ThenBlock.Statements))
}

func Test_Analyze_IfElsifElseStatement(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		} elsif false {
			y: = 2
		} elsif true {
			z: = 3
		} else {
			w: = 4
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_IfElsifElseStatement", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	ifStmt := funcDecl.Body.Statements[0].(*SemIf)

	assert.NotNil(t, ifStmt.ThenBlock)
	assert.Equal(t, 1, len(ifStmt.ThenBlock.Statements))
	assert.Equal(t, 2, len(ifStmt.ElsifBlocks))
	assert.NotNil(t, ifStmt.ElseBlock)
	assert.Equal(t, 1, len(ifStmt.ElseBlock.Statements))

	// Check first elsif
	elsif1 := ifStmt.ElsifBlocks[0]
	assert.NotNil(t, elsif1.Condition)
	assert.NotNil(t, elsif1.ThenBlock)
	assert.Equal(t, 1, len(elsif1.ThenBlock.Statements))

	// Check second elsif
	elsif2 := ifStmt.ElsifBlocks[1]
	assert.NotNil(t, elsif2.Condition)
	assert.NotNil(t, elsif2.ThenBlock)
	assert.Equal(t, 1, len(elsif2.ThenBlock.Statements))
}

// ============================================================================
// For Loop Tests
// ============================================================================

func Test_Analyze_ForLoop_Full(t *testing.T) {
	code := `main: () {
		for i: = 0; i < 10; i + 1 {
			x: = i
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ForLoop_Full", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 1, len(funcDecl.Body.Statements))

	forStmt, ok := funcDecl.Body.Statements[0].(*SemFor)
	require.True(t, ok, "Statement should be SemFor")
	assert.NotNil(t, forStmt.Initializer)
	assert.NotNil(t, forStmt.Condition)
	assert.NotNil(t, forStmt.Increment)
	assert.NotNil(t, forStmt.Body)
	assert.Equal(t, 1, len(forStmt.Body.Statements))
}

func Test_Analyze_ForLoop_OnlyCondition(t *testing.T) {
	code := `main: () {
		for true {
			x: = 1
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ForLoop_OnlyCondition", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	forStmt := funcDecl.Body.Statements[0].(*SemFor)

	assert.Nil(t, forStmt.Initializer)
	assert.NotNil(t, forStmt.Condition)
	assert.Nil(t, forStmt.Increment)
	assert.NotNil(t, forStmt.Body)
}

func Test_Analyze_ForLoop_Scope(t *testing.T) {
	code := `main: () {
		for i: = 0; i < 10; i + 1 {
			j: = i
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ForLoop_Scope", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	forStmt := funcDecl.Body.Statements[0].(*SemFor)

	// Initializer should be a variable declaration
	varDecl, ok := forStmt.Initializer.(*SemVariableDecl)
	require.True(t, ok, "Initializer should be SemVariableDecl")
	assert.Equal(t, "i", varDecl.Symbol.Name)

	// Body should be able to reference i
	bodyVarDecl := forStmt.Body.Statements[0].(*SemVariableDecl)
	assert.NotNil(t, bodyVarDecl.Initializer, "Loop variable should be accessible in body")
}

// ============================================================================
// Select Tests
// ============================================================================

func Test_Analyze_SelectStatement_Simple(t *testing.T) {
	code := `main: () {
		x: = 5
		select x {
			case 1 {
				y: = 10
			}
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_SelectStatement_Simple", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 2, len(funcDecl.Body.Statements))

	selectStmt, ok := funcDecl.Body.Statements[1].(*SemSelect)
	require.True(t, ok, "Statement should be SemSelect")
	assert.Equal(t, 1, len(selectStmt.Cases))
}

func Test_Analyze_SelectStatement(t *testing.T) {
	code := `main: () {
		x: = 5
		select x {
			case 1 {
				a: = 10
			}
			case 2 {
				b: = 20
			}
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_SelectStatement", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 2, len(funcDecl.Body.Statements))

	selectStmt, ok := funcDecl.Body.Statements[1].(*SemSelect)
	require.True(t, ok, "Statement should be SemSelect")
	assert.NotNil(t, selectStmt.Expression)
	assert.Equal(t, 2, len(selectStmt.Cases))
	assert.Nil(t, selectStmt.Else)

	// Check first case
	assert.NotNil(t, selectStmt.Cases[0].Value)
	assert.NotNil(t, selectStmt.Cases[0].Body)
	assert.Equal(t, 1, len(selectStmt.Cases[0].Body.Statements))

	// Check second case
	assert.NotNil(t, selectStmt.Cases[1].Value)
	assert.NotNil(t, selectStmt.Cases[1].Body)
	assert.Equal(t, 1, len(selectStmt.Cases[1].Body.Statements))
}

func Test_Analyze_SelectStatementWithElse(t *testing.T) {
	code := `main: () {
		x: = 5
		select x {
			case 1 {
				a: = 10
			}
			else {
				b: = 20
			}
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_SelectStatementWithElse", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	selectStmt := funcDecl.Body.Statements[1].(*SemSelect)

	assert.Equal(t, 1, len(selectStmt.Cases))
	assert.NotNil(t, selectStmt.Else)
	assert.Equal(t, 1, len(selectStmt.Else.Statements))
}

func Test_Analyze_SelectStatementMultipleCases(t *testing.T) {
	code := `main: () {
		x: = 5
		select x {
			case 1 {
				a: = 10
			}
			case 2 {
				b: = 20
			}
			case 3 {
				c: = 30
			}
			else {
				d: = 40
			}
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_SelectStatementMultipleCases", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	selectStmt := funcDecl.Body.Statements[1].(*SemSelect)

	assert.Equal(t, 3, len(selectStmt.Cases))
	assert.NotNil(t, selectStmt.Else)

	// Verify all cases have bodies
	for i, c := range selectStmt.Cases {
		assert.NotNil(t, c.Value, "Case %d should have value", i)
		assert.NotNil(t, c.Body, "Case %d should have body", i)
		assert.Equal(t, 1, len(c.Body.Statements), "Case %d should have 1 statement", i)
	}
}

// ============================================================================
// Return Statement Tests
// ============================================================================

func Test_Analyze_ReturnStatement(t *testing.T) {
	code := `main: () {
		ret
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ReturnStatement", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 1, len(funcDecl.Body.Statements))

	retStmt, ok := funcDecl.Body.Statements[0].(*SemReturn)
	require.True(t, ok, "Statement should be SemReturn")
	assert.Nil(t, retStmt.Value, "Return without value should have nil Value")
}

func Test_Analyze_ReturnStatementWithValue(t *testing.T) {
	code := `main: () {
		ret 42
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ReturnStatementWithValue", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 1, len(funcDecl.Body.Statements))

	retStmt, ok := funcDecl.Body.Statements[0].(*SemReturn)
	require.True(t, ok, "Statement should be SemReturn")
	require.NotNil(t, retStmt.Value, "Return with value should have non-nil Value")

	// Verify the value is a constant
	constant, ok := retStmt.Value.(*SemConstant)
	require.True(t, ok, "Return value should be SemConstant")
	assert.Equal(t, 42, constant.Value)
}

func Test_Analyze_ReturnStatementWithExpression(t *testing.T) {
	code := `main: () {
		x: = 10
		ret x + 5
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ReturnStatementWithExpression", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	require.Equal(t, 2, len(funcDecl.Body.Statements))

	retStmt, ok := funcDecl.Body.Statements[1].(*SemReturn)
	require.True(t, ok, "Statement should be SemReturn")
	require.NotNil(t, retStmt.Value, "Return with expression should have non-nil Value")

	// Verify the value is a binary operation
	binOp, ok := retStmt.Value.(*SemBinaryOp)
	require.True(t, ok, "Return value should be SemBinaryOp")
	assert.NotNil(t, binOp.Left)
	assert.NotNil(t, binOp.Right)
}

// ============================================================================
// Expression Tests
// ============================================================================

func Test_Analyze_BinaryOperation(t *testing.T) {
	code := `main: () {
		result: = 5 + 3
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_BinaryOperation", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	varDecl := funcDecl.Body.Statements[0].(*SemVariableDecl)

	binOp, ok := varDecl.Initializer.(*SemBinaryOp)
	require.True(t, ok, "Initializer should be SemBinaryOp")
	assert.Equal(t, OpAdd, binOp.Op)
	assert.NotNil(t, binOp.Left)
	assert.NotNil(t, binOp.Right)
}

func Test_Analyze_BooleanLiteral(t *testing.T) {
	code := `flag: = true`
	semCU, errors := analyzeCode(t, "Test_Analyze_BooleanLiteral", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, true, constant.Value)
	assert.Equal(t, BoolType, constant.Type())
}

func Test_Analyze_NumberLiteral_U8(t *testing.T) {
	code := `num: = 42`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_U8", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, 42, constant.Value)
	assert.Equal(t, U8Type, constant.Type())
}

func Test_Analyze_NumberLiteral_U16(t *testing.T) {
	code := `num: = 300`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_U16", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, 300, constant.Value)
	assert.Equal(t, U16Type, constant.Type())
}

func Test_Analyze_NumberLiteral_I8(t *testing.T) {
	code := `num: = -50`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_I8", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, -50, constant.Value)
	assert.Equal(t, I8Type, constant.Type())
}

func Test_Analyze_NumberLiteral_I16(t *testing.T) {
	code := `num: = -1000`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_I16", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, -1000, constant.Value)
	assert.Equal(t, I16Type, constant.Type())
}

func Test_Analyze_NumberLiteral_Hex(t *testing.T) {
	code := `num: = 0xFF`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_Hex", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, 255, constant.Value)
	assert.Equal(t, U8Type, constant.Type())
}

func Test_Analyze_NumberLiteral_HexLarge(t *testing.T) {
	code := `num: = 0xAB00`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_HexLarge", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, 0xAB00, constant.Value)
	assert.Equal(t, U16Type, constant.Type())
}

func Test_Analyze_NumberLiteral_Binary(t *testing.T) {
	code := `num: = 0b00101010`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_Binary", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, 0b00101010, constant.Value)
	assert.Equal(t, U8Type, constant.Type())
}

func Test_Analyze_NumberLiteral_BinaryLarge(t *testing.T) {
	code := `num: = 0b100000000`
	semCU, errors := analyzeCode(t, "Test_Analyze_NumberLiteral_BinaryLarge", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, 0b100000000, constant.Value)
	assert.Equal(t, U16Type, constant.Type())
}

func Test_Analyze_StringLiteral(t *testing.T) {
	code := `msg: = "hello"`
	semCU, errors := analyzeCode(t, "Test_Analyze_StringLiteral", code)
	requireNoErrors(t, errors)

	varDecl := semCU.Declarations[0].(*SemVariableDecl)
	constant, ok := varDecl.Initializer.(*SemConstant)
	require.True(t, ok)
	assert.Equal(t, "\"hello\"", constant.Value)

	// String should be an array type
	arrayType, ok := constant.Type().(*ArrayType)
	require.True(t, ok, "String type should be array")
	assert.Equal(t, U8Type, arrayType.ElementType())
}

func Test_Analyze_FunctionCall(t *testing.T) {
	code := `doSomething: () {
	}
	main: () {
		doSomething()
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_FunctionCall", code)
	requireNoErrors(t, errors)

	mainFunc := semCU.Declarations[1].(*SemFunctionDecl)
	exprStmt := mainFunc.Body.Statements[0].(*SemExpressionStmt)

	funcCall, ok := exprStmt.Expression.(*SemFunctionCall)
	require.True(t, ok, "Expression should be SemFunctionCall")
	assert.Equal(t, "doSomething", funcCall.Function.Name)
	assert.Equal(t, 0, len(funcCall.Arguments))
}

func Test_Analyze_FunctionCallWithArgs(t *testing.T) {
	code := `add: (a: u8, b: u8) {
	}
	main: () {
		add(5, 10)
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_FunctionCallWithArgs", code)
	requireNoErrors(t, errors)

	mainFunc := semCU.Declarations[1].(*SemFunctionDecl)
	exprStmt := mainFunc.Body.Statements[0].(*SemExpressionStmt)

	funcCall := exprStmt.Expression.(*SemFunctionCall)
	assert.Equal(t, "add", funcCall.Function.Name)
	assert.Equal(t, 2, len(funcCall.Arguments))
}

func Test_Analyze_FunctionCallUndefined_Error(t *testing.T) {
	code := `main: () {
		unknown()
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionCallUndefined_Error", code)

	require.Greater(t, len(errors), 0, "Expected error for undefined function")
	assert.Contains(t, errors[0].Error(), "undefined function")
}

// ============================================================================
// Scope Tests
// ============================================================================

func Test_Analyze_ScopeParameterAccess(t *testing.T) {
	code := `myFunc: (param: u8) {
		x: = param
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ScopeParameterAccess", code)
	requireNoErrors(t, errors)

	funcDecl := semCU.Declarations[0].(*SemFunctionDecl)
	varDecl := funcDecl.Body.Statements[0].(*SemVariableDecl)

	// The initializer should reference the parameter
	assert.NotNil(t, varDecl.Initializer)
}

func Test_Analyze_ScopeGlobalAccess(t *testing.T) {
	code := `global: u8 = 42
	myFunc: () {
		local: = global
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_ScopeGlobalAccess", code)
	requireNoErrors(t, errors)

	require.Equal(t, 2, len(semCU.Declarations))
	funcDecl := semCU.Declarations[1].(*SemFunctionDecl)
	assert.Equal(t, 1, len(funcDecl.Body.Statements))
}

// ============================================================================
// Built-in Types Tests
// ============================================================================

func Test_Analyze_BuiltinTypes(t *testing.T) {
	builtinTypes := []string{"u8", "u16", "i8", "i16", "d8", "d16", "bool"}

	for _, typeName := range builtinTypes {
		t.Run(typeName, func(t *testing.T) {
			code := fmt.Sprintf("var: %s", typeName)
			semCU, errors := analyzeCode(t, "Test_Analyze_BuiltinType_"+typeName, code)
			requireNoErrors(t, errors)

			varDecl := semCU.Declarations[0].(*SemVariableDecl)
			assert.NotNil(t, varDecl.Symbol.Type)
		})
	}
}

// ============================================================================
// Call Graph Tests
// ============================================================================

func Test_Analyze_CallGraph_Simple(t *testing.T) {
	code := `helper: () {
	}
	main: () {
		helper()
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_Simple", code)
	requireNoErrors(t, errors)

	// Check call graph
	assert.NotNil(t, semCU.CallGraph)

	// main should call helper
	mainCallees := semCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "helper", mainCallees[0])

	// helper should have no callees
	helperCallees := semCU.CallGraph.GetCallees("helper")
	assert.Equal(t, 0, len(helperCallees))

	// Both functions should be registered
	allFuncs := semCU.CallGraph.GetAllFunctions()
	assert.Equal(t, 2, len(allFuncs))
}

func Test_Analyze_CallGraph_Chain(t *testing.T) {
	code := `worker: () {
	}
	helper: () {
		worker()
	}
	main: () {
		helper()
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_Chain", code)
	requireNoErrors(t, errors)

	// main -> helper
	mainCallees := semCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "helper", mainCallees[0])

	// helper -> worker
	helperCallees := semCU.CallGraph.GetCallees("helper")
	assert.Equal(t, 1, len(helperCallees))
	assert.Equal(t, "worker", helperCallees[0])

	// worker has no callees
	workerCallees := semCU.CallGraph.GetCallees("worker")
	assert.Equal(t, 0, len(workerCallees))
}

func Test_Analyze_CallGraph_MultipleCalls(t *testing.T) {
	code := `foo: () {
	}
	bar: () {
	}
	main: () {
		foo()
		bar()
		foo()
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_MultipleCalls", code)
	requireNoErrors(t, errors)

	// main should call both foo and bar (foo should only appear once despite 2 calls)
	mainCallees := semCU.CallGraph.GetCallees("main")
	assert.Equal(t, 2, len(mainCallees))
	assert.Contains(t, mainCallees, "foo")
	assert.Contains(t, mainCallees, "bar")
}

func Test_Analyze_CallGraph_NestedInIf(t *testing.T) {
	code := `helper: () {
	}
	main: () {
		if true {
			helper()
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedInIf", code)
	requireNoErrors(t, errors)

	// Call inside if block should still be recorded
	mainCallees := semCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "helper", mainCallees[0])
}

func Test_Analyze_CallGraph_NestedInIfElse(t *testing.T) {
	code := `foo: () {
	}
	bar: () {
	}
	main: () {
		if true {
			foo()
		} else {
			bar()
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedInIfElse", code)
	requireNoErrors(t, errors)

	// Both calls should be recorded
	mainCallees := semCU.CallGraph.GetCallees("main")
	assert.Equal(t, 2, len(mainCallees))
	assert.Contains(t, mainCallees, "foo")
	assert.Contains(t, mainCallees, "bar")
}

func Test_Analyze_CallGraph_NestedInSelect(t *testing.T) {
	code := `helper: () {
	}
	main: () {
		x: u8 = 5
		select x {
			case 5 { helper() }
		}
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedInSelect", code)
	requireNoErrors(t, errors)

	// Call inside select block should be recorded
	mainCallees := semCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "helper", mainCallees[0])
}

func Test_Analyze_CallGraph_NestedFunctionCalls(t *testing.T) {
	code := `bar: () {
	}
	foo: () {
		bar()
	}
	main: () {
		foo()
	}`
	semCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedFunctionCalls", code)
	requireNoErrors(t, errors)

	// main -> foo
	mainCallees := semCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "foo", mainCallees[0])

	// foo -> bar
	fooCallees := semCU.CallGraph.GetCallees("foo")
	assert.Equal(t, 1, len(fooCallees))
	assert.Equal(t, "bar", fooCallees[0])

	// bar has no callees
	barCallees := semCU.CallGraph.GetCallees("bar")
	assert.Equal(t, 0, len(barCallees))

	// All three functions should be in the graph
	allFuncs := semCU.CallGraph.GetAllFunctions()
	assert.Equal(t, 3, len(allFuncs))
}
