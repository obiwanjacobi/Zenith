package parser

import (
	"fmt"
	"testing"

	"zenith/compiler/lexer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseCode is a helper function that parses code and returns the CompilationUnit
func parseCode(t *testing.T, testName string, code string) CompilationUnit {
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse(testName, tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))
	return node.(CompilationUnit)
}

func Test_ParseVarDeclType(t *testing.T) {
	code := "var: u8"
	cu := parseCode(t, "Test_ParseVarDeclType", code)
	assert.Equal(t, 1, len(cu.Declarations()))

	varDecl, ok := cu.Declarations()[0].(VariableDeclaration)
	assert.True(t, ok)
	assert.Equal(t, "var", varDecl.Label().Name())
	assert.NotNil(t, varDecl.TypeRef())
	assert.Equal(t, "u8", varDecl.TypeRef().TypeName().Text())
	assert.Nil(t, varDecl.Initializer())
}

func Test_ParseVarDeclTypeWithInit(t *testing.T) {
	code := "count: u16 = 42"
	cu := parseCode(t, "Test_ParseVarDeclTypeWithInit", code)
	assert.Equal(t, 1, len(cu.Declarations()))

	varDecl, ok := cu.Declarations()[0].(VariableDeclaration)
	assert.True(t, ok)
	assert.Equal(t, "count", varDecl.Label().Name())
	assert.NotNil(t, varDecl.TypeRef())
	assert.Equal(t, "u16", varDecl.TypeRef().TypeName().Text())
	assert.NotNil(t, varDecl.Initializer())
}

func Test_ParseVarDeclInferred(t *testing.T) {
	code := "value: = 100"
	cu := parseCode(t, "Test_ParseVarDeclInferred", code)
	assert.Equal(t, 1, len(cu.Declarations()))

	varDecl, ok := cu.Declarations()[0].(VariableDeclaration)
	assert.True(t, ok)
	assert.Equal(t, "value", varDecl.Label().Name())
	assert.Nil(t, varDecl.TypeRef())
	assert.NotNil(t, varDecl.Initializer())
}

func Test_ParseVarAssignment(t *testing.T) {
	code := `fn: () {
			x = 5
		}`
	cu := parseCode(t, "Test_ParseVarAssignment", code)
	assert.Equal(t, 1, len(cu.Declarations()))

	funcDecl, ok := cu.Declarations()[0].(FunctionDeclaration)
	assert.True(t, ok)
	body := funcDecl.Body()
	assert.Equal(t, 1, len(body.Statements()))

	varAssign, ok := body.Statements()[0].(VariableAssignment)
	assert.True(t, ok)
	assert.NotNil(t, varAssign.Expression())
}

func Test_ParseFunctionDeclaration(t *testing.T) {
	code := `func: () {
	}`
	cu := parseCode(t, "Test_ParseFunctionDeclaration", code)
	assert.Equal(t, 1, len(cu.Declarations()))

	funcDecl, ok := cu.Declarations()[0].(FunctionDeclaration)
	assert.True(t, ok)
	assert.Equal(t, "func", funcDecl.Label().Name())
	assert.Nil(t, funcDecl.Parameters())
	assert.Nil(t, funcDecl.ReturnType())
	assert.NotNil(t, funcDecl.Body())
}

func Test_ParseFunctionWithParams(t *testing.T) {
	code := `add: (a: u8, b: u8) {
	}`
	cu := parseCode(t, "Test_ParseFunctionWithParams", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	assert.Equal(t, "add", funcDecl.Label().Name())
	assert.NotNil(t, funcDecl.Parameters())
}

func Test_ParseFunctionWithReturnType(t *testing.T) {
	code := `getValue: () u16 {
	}`
	cu := parseCode(t, "Test_ParseFunctionWithReturnType", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	assert.NotNil(t, funcDecl.ReturnType())
	assert.Equal(t, "u16", funcDecl.ReturnType().TypeName().Text())
}

func Test_ParseStructDeclaration(t *testing.T) {
	code := `struct Point {
		x: u8,
		y: u8
	}`
	cu := parseCode(t, "Test_ParseStructDeclaration", code)
	structDecl, ok := cu.Declarations()[0].(TypeDeclaration)
	assert.True(t, ok)
	assert.NotNil(t, structDecl.Fields())
}

func Test_ParseIfStatement(t *testing.T) {
	code := `main: () {
		if x > 5 {
		}
	}`
	cu := parseCode(t, "Test_ParseIfStatement", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	body := funcDecl.Body()
	assert.Equal(t, 1, len(body.Statements()))

	ifStmt, ok := body.Statements()[0].(StatementIf)
	assert.True(t, ok)
	assert.NotNil(t, ifStmt.Condition())
	assert.NotNil(t, ifStmt.ThenBlock())
}

func Test_ParseIfElsifElse(t *testing.T) {
	code := `main: () {
		if x > 5 {
		} elsif x > 0 {
		} else {
		}
	}`
	cu := parseCode(t, "Test_ParseIfElsifElse", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	body := funcDecl.Body()
	ifStmt := body.Statements()[0].(StatementIf)

	// Should have 4 children: condition, then block, elsif, else block
	assert.True(t, len(ifStmt.Children()) >= 4)
}

func Test_ParseForLoop(t *testing.T) {
	code := `main: () {
		for i: = 0; i < 10; i++ {
		}
	}`
	cu := parseCode(t, "Test_ParseForLoop", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	body := funcDecl.Body()

	forStmt, ok := body.Statements()[0].(StatementFor)
	assert.True(t, ok)
	assert.NotNil(t, forStmt)
}

func Test_ParseSelectStatement(t *testing.T) {
	code := `main: () {
		select value {
			case 1 {
			}
			case 2 {
			}
			else {
			}
		}
	}`
	cu := parseCode(t, "Test_ParseSelectStatement", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	body := funcDecl.Body()

	selectStmt, ok := body.Statements()[0].(StatementSelect)
	assert.True(t, ok)
	assert.NotNil(t, selectStmt)
}

func Test_ParseReturnStatement(t *testing.T) {
	code := `main: () {
		ret
	}`
	cu := parseCode(t, "Test_ParseReturnStatement", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	body := funcDecl.Body()
	assert.Equal(t, 1, len(body.Statements()))

	retStmt, ok := body.Statements()[0].(StatementReturn)
	assert.True(t, ok)
	assert.Nil(t, retStmt.Value(), "Return without expression should have nil value")
}

func Test_ParseReturnStatementWithExpression(t *testing.T) {
	code := `main: () {
		ret 42
	}`
	cu := parseCode(t, "Test_ParseReturnStatementWithExpression", code)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	body := funcDecl.Body()
	assert.Equal(t, 1, len(body.Statements()))

	retStmt, ok := body.Statements()[0].(StatementReturn)
	assert.True(t, ok)
	assert.NotNil(t, retStmt.Value(), "Return with expression should have non-nil value")

	// Check that the expression is a number literal
	_, isLiteral := retStmt.Value().(ExpressionLiteral)
	assert.True(t, isLiteral, "Return value should be a literal expression")
}

func Test_ParseExpressionLiteral(t *testing.T) {
	code := `value: = 42`
	cu := parseCode(t, "Test_ParseExpressionLiteral", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	expr := varDecl.Initializer()
	assert.NotNil(t, expr)
}

func Test_ParseExpressionBinaryArithmetic(t *testing.T) {
	code := `result: = 10 + 20`
	cu := parseCode(t, "Test_ParseExpressionBinaryArithmetic", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	binOp, ok := varDecl.Initializer().(ExpressionOperatorBinArithmetic)
	assert.True(t, ok)
	assert.NotNil(t, binOp.Left())
	assert.NotNil(t, binOp.Right())
}

func Test_ParseExpressionComplex(t *testing.T) {
	code := `result: = (a + b) * c - d / 2`
	cu := parseCode(t, "Test_ParseExpressionComplex", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)
	assert.NotNil(t, varDecl.Initializer())
}

func Test_ParseExpressionComparison(t *testing.T) {
	code := `check: = x > 5`
	cu := parseCode(t, "Test_ParseExpressionComparison", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	cmpOp, ok := varDecl.Initializer().(ExpressionOperatorBinComparison)
	assert.True(t, ok)
	assert.NotNil(t, cmpOp.Left())
	assert.NotNil(t, cmpOp.Right())
}

func Test_ParseExpressionLogical(t *testing.T) {
	code := `check: = x > 5 and y < 10`
	cu := parseCode(t, "Test_ParseExpressionLogical", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	logOp, ok := varDecl.Initializer().(ExpressionOperatorBinLogical)
	assert.True(t, ok)
	assert.NotNil(t, logOp.Left())
	assert.NotNil(t, logOp.Right())
}

func Test_ParseExpressionBitwise(t *testing.T) {
	code := `result: = flags & 0xFF`
	cu := parseCode(t, "Test_ParseExpressionBitwise", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	bitOp, ok := varDecl.Initializer().(ExpressionOperatorBinBitwise)
	assert.True(t, ok)
	assert.NotNil(t, bitOp.Left())
	assert.NotNil(t, bitOp.Right())
}

func Test_ParseExpressionUnaryPrefix(t *testing.T) {
	code := `neg: = -value`
	cu := parseCode(t, "Test_ParseExpressionUnaryPrefix", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	unaryOp, ok := varDecl.Initializer().(ExpressionOperatorUnipreArithmetic)
	assert.True(t, ok)
	assert.NotNil(t, unaryOp.Operand())
}

func Test_ParseExpressionIdentifier(t *testing.T) {
	code := `result: = myVar`
	cu := parseCode(t, "Test_ParseExpressionIdentifier", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	identifier, ok := varDecl.Initializer().(ExpressionIdentifier)
	assert.True(t, ok, "Initializer should be ExpressionIdentifier")
	assert.NotNil(t, identifier.Identifier(), "Identifier token should not be nil")
	assert.Equal(t, "myVar", identifier.Identifier().Text(), "Identifier name should be 'myVar'")
}

func Test_ParseExpressionMemberAccess(t *testing.T) {
	code := `value: = obj.field`
	cu := parseCode(t, "Test_ParseExpressionMemberAccess", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	memberAccess, ok := varDecl.Initializer().(ExpressionMemberAccess)
	assert.True(t, ok)
	assert.NotNil(t, memberAccess.Object())
}

func Test_ParseFunctionCall(t *testing.T) {
	code := `result: = add(1, 2)`
	cu := parseCode(t, "Test_ParseFunctionCall", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	funcCall, ok := varDecl.Initializer().(ExpressionFunctionInvocation)
	assert.True(t, ok)
	assert.NotNil(t, funcCall)
}

func Test_ParseTypeInitializer(t *testing.T) {
	code := `point: = Point{x = 10, y = 20}`
	cu := parseCode(t, "Test_ParseTypeInitializer", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	typeInit, ok := varDecl.Initializer().(ExpressionTypeInitializer)
	assert.True(t, ok)
	assert.NotNil(t, typeInit.TypeRef())
	assert.NotNil(t, typeInit.Initializer())
}

func Test_ParseArrayType(t *testing.T) {
	code := `buffer: u8[256]`
	cu := parseCode(t, "Test_ParseArrayType", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	typeRef := varDecl.TypeRef()
	assert.NotNil(t, typeRef)
	// Array syntax should be captured in tokens
	assert.True(t, len(typeRef.Tokens()) > 0)
}

func Test_ParseMultipleDeclarations(t *testing.T) {
	code := `
		x: u8
		y: u16 = 100
		func: () {
		}
		struct Data {
			value: u8
		}
	`
	cu := parseCode(t, "Test_ParseMultipleDeclarations", code)
	assert.Equal(t, 4, len(cu.Declarations()))
}

func Test_ParseOperatorPrecedence(t *testing.T) {
	code := `result: = 2 + 3 * 4`
	cu := parseCode(t, "Test_ParseOperatorPrecedence", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	// Should parse as: (2 + 3) * 4 (left-to-right, no operator precedence)
	mulOp, ok := varDecl.Initializer().(ExpressionOperatorBinArithmetic)
	assert.True(t, ok)

	// Left side should be addition
	_, leftIsAdd := mulOp.Left().(ExpressionOperatorBinArithmetic)
	assert.True(t, leftIsAdd)
}

func Test_ParseStringLiteral(t *testing.T) {
	code := `msg: = "Hello, World!"`
	cu := parseCode(t, "Test_ParseStringLiteral", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	literal, ok := varDecl.Initializer().(ExpressionLiteral)
	assert.True(t, ok)
	assert.NotNil(t, literal)
}

func Test_ParseBooleanLiteral(t *testing.T) {
	code := `flag: = true`
	cu := parseCode(t, "Test_ParseBooleanLiteral", code)
	varDecl := cu.Declarations()[0].(VariableDeclaration)

	literal, ok := varDecl.Initializer().(ExpressionLiteral)
	assert.True(t, ok)
	assert.NotNil(t, literal)
}

func Test_ParseStructDeclarationTopLevel(t *testing.T) {
	code := `struct Point {
		x: u8,
		y: u8
	}`
	cu := parseCode(t, "Test_ParseStructDeclarationTopLevel", code)
	require.Equal(t, 1, len(cu.Declarations()))

	structDecl, ok := cu.Declarations()[0].(TypeDeclaration)
	assert.True(t, ok)
	assert.NotNil(t, structDecl.Fields())
}

func Test_ParseInitStructWithFields(t *testing.T) {
	code := `
	struct Point {
		x: u8,
		y: u8
	}
	main: () {
		p: Point = Point{x = 5, y = 10}
	}`
	cu := parseCode(t, "Test_ParseInitStructWithFields", code)
	require.Equal(t, 2, len(cu.Declarations()))

	// First should be struct
	structDecl, ok := cu.Declarations()[0].(TypeDeclaration)
	assert.True(t, ok)
	assert.Equal(t, "Point", structDecl.Name().Text())

	// Second should be function
	funcDecl, ok := cu.Declarations()[1].(FunctionDeclaration)
	assert.True(t, ok)
	assert.Equal(t, "main", funcDecl.Label().Name())
}

func Test_ParseStructUsageInFunction(t *testing.T) {
	code := `
	struct Point {
		x: u8,
		y: u8
	}
	main: () {
		p: Point = Point{x= 5, y= 10}
		val: u8 = p.x
	}`
	cu := parseCode(t, "Test_ParseStructUsageInFunction", code)
	require.Equal(t, 2, len(cu.Declarations()))

	funcDecl, ok := cu.Declarations()[1].(FunctionDeclaration)
	assert.True(t, ok)

	// Check that function body parses correctly
	body := funcDecl.Body()
	assert.NotNil(t, body)
	assert.Greater(t, len(body.Statements()), 0)
}

func Test_ParseStructDeclarationMissingComma(t *testing.T) {
	code := `struct Point {
		x: u8
		y: u8
	}`
	tokens := lexer.OpenTokenStream(code)
	_, errors := Parse("Test_ParseStructDeclarationMissingComma", tokens)

	require.NotEqual(t, 0, len(errors), "Parser should report error for missing comma")
}

func Test_ParseSelectInvalidCaseOrElse(t *testing.T) {
	code := `main: () {
		select value {
			5: {
			}
		}
	}`
	tokens := lexer.OpenTokenStream(code)
	_, errors := Parse("Test_ParseSelectInvalidCaseOrElse", tokens)

	require.NotEqual(t, 0, len(errors), "Parser should report error for missing case or else clause")
}

func Test_ParseFuncParamArray(t *testing.T) {
	code := `max: (arr: u8[]) u8 {
		if arr[0] > arr[1] {
			ret arr[0]
		} else {
			ret arr[1]
		}
	}`
	tokens := lexer.OpenTokenStream(code)
	_, errors := Parse("Test_ParseFuncParamArray", tokens)

	assert.Empty(t, errors, fmt.Sprintf("Parser should not report error for array parameter: %v", errors))
}
