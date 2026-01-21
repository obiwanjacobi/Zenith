package parser

import (
	"fmt"
	"testing"

	"zenith/compiler/lexer"

	"github.com/stretchr/testify/assert"
)

func Test_ParseVarDeclType(t *testing.T) {
	code := "var: u8"
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseVarDeclType", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	assert.Equal(t, 1, len(cu.Declarations()))

	varDeclType, ok := cu.Declarations()[0].(VariableDeclarationType)
	assert.True(t, ok)
	assert.Equal(t, "var", varDeclType.Label().Name())
	assert.Equal(t, "u8", varDeclType.TypeRef().TypeName().Text())
}

func Test_ParseVarDeclTypeWithInit(t *testing.T) {
	code := "count: u16 = 42"
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseVarDeclTypeWithInit", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	assert.Equal(t, 1, len(cu.Declarations()))

	varDecl, ok := cu.Declarations()[0].(VariableDeclarationType)
	assert.True(t, ok)
	assert.Equal(t, "count", varDecl.Label().Name())
	assert.Equal(t, "u16", varDecl.TypeRef().TypeName().Text())
	assert.NotNil(t, varDecl.Initializer())
}

func Test_ParseVarDeclInferred(t *testing.T) {
	code := "value: = 100"
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseVarDeclInferred", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	assert.Equal(t, 1, len(cu.Declarations()))

	varDecl, ok := cu.Declarations()[0].(VariableDeclarationInferred)
	assert.True(t, ok)
	assert.Equal(t, "value", varDecl.Label().Name())
	assert.NotNil(t, varDecl.Initializer())
}

func Test_ParseVarAssignment(t *testing.T) {
	code := `fn: () {
			x = 5
		}`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseVarAssignment", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
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
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseFunctionDeclaration", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
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
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseFunctionWithParams", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	assert.Equal(t, "add", funcDecl.Label().Name())
	assert.NotNil(t, funcDecl.Parameters())
}

func Test_ParseFunctionWithReturnType(t *testing.T) {
	code := `getValue: () u16 {
	}`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseFunctionWithReturnType", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	assert.NotNil(t, funcDecl.ReturnType())
	assert.Equal(t, "u16", funcDecl.ReturnType().TypeName().Text())
}

func Test_ParseStructDeclaration(t *testing.T) {
	code := `struct Point {
		x: u8,
		y: u8
	}`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseStructDeclaration", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	structDecl, ok := cu.Declarations()[0].(TypeDeclaration)
	assert.True(t, ok)
	assert.NotNil(t, structDecl.Fields())
}

func Test_ParseIfStatement(t *testing.T) {
	code := `main: () {
		if x > 5 {
		}
	}`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseIfStatement", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
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
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseIfElsifElse", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
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
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseForLoop", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
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
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseSelectStatement", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	funcDecl := cu.Declarations()[0].(FunctionDeclaration)
	body := funcDecl.Body()

	selectStmt, ok := body.Statements()[0].(StatementSelect)
	assert.True(t, ok)
	assert.NotNil(t, selectStmt)
}

func Test_ParseExpressionLiteral(t *testing.T) {
	code := `value: = 42`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionLiteral", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	expr := varDecl.Initializer()
	assert.NotNil(t, expr)
}

func Test_ParseExpressionBinaryArithmetic(t *testing.T) {
	code := `result: = 10 + 20`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionBinaryArithmetic", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	binOp, ok := varDecl.Initializer().(ExpressionOperatorBinArithmetic)
	assert.True(t, ok)
	assert.NotNil(t, binOp.Left())
	assert.NotNil(t, binOp.Right())
}

func Test_ParseExpressionComplex(t *testing.T) {
	code := `result: = (a + b) * c - d / 2`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionComplex", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)
	assert.NotNil(t, varDecl.Initializer())
}

func Test_ParseExpressionComparison(t *testing.T) {
	code := `check: = x > 5`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionComparison", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	cmpOp, ok := varDecl.Initializer().(ExpressionOperatorBinComparison)
	assert.True(t, ok)
	assert.NotNil(t, cmpOp.Left())
	assert.NotNil(t, cmpOp.Right())
}

func Test_ParseExpressionLogical(t *testing.T) {
	code := `check: = x > 5 and y < 10`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionLogical", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	logOp, ok := varDecl.Initializer().(ExpressionOperatorBinLogical)
	assert.True(t, ok)
	assert.NotNil(t, logOp.Left())
	assert.NotNil(t, logOp.Right())
}

func Test_ParseExpressionBitwise(t *testing.T) {
	code := `result: = flags & 0xFF`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionBitwise", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	bitOp, ok := varDecl.Initializer().(ExpressionOperatorBinBitwise)
	assert.True(t, ok)
	assert.NotNil(t, bitOp.Left())
	assert.NotNil(t, bitOp.Right())
}

func Test_ParseExpressionUnaryPrefix(t *testing.T) {
	code := `neg: = -value`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionUnaryPrefix", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	unaryOp, ok := varDecl.Initializer().(ExpressionOperatorUnipreArithmetic)
	assert.True(t, ok)
	assert.NotNil(t, unaryOp.Operand())
}

func Test_ParseExpressionMemberAccess(t *testing.T) {
	code := `value: = obj.field`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseExpressionMemberAccess", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	memberAccess, ok := varDecl.Initializer().(ExpressionMemberAccess)
	assert.True(t, ok)
	assert.NotNil(t, memberAccess.Object())
}

func Test_ParseFunctionCall(t *testing.T) {
	code := `result: = add(1, 2)`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseFunctionCall", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	funcCall, ok := varDecl.Initializer().(FunctionInvocation)
	assert.True(t, ok)
	assert.NotNil(t, funcCall)
}

func Test_ParseTypeInitializer(t *testing.T) {
	code := `point: = Point{x = 10, y = 20}`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseTypeInitializer", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	typeInit, ok := varDecl.Initializer().(ExpressionTypeInitializer)
	assert.True(t, ok)
	assert.NotNil(t, typeInit.TypeRef())
	assert.NotNil(t, typeInit.Initializer())
}

func Test_ParseArrayType(t *testing.T) {
	code := `buffer: [256]u8`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseArrayType", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationType)

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
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseMultipleDeclarations", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	assert.Equal(t, 4, len(cu.Declarations()))
}

func Test_ParseOperatorPrecedence(t *testing.T) {
	code := `result: = 2 + 3 * 4`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseOperatorPrecedence", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	// Should parse as: 2 + (3 * 4)
	addOp, ok := varDecl.Initializer().(ExpressionOperatorBinArithmetic)
	assert.True(t, ok)

	// Right side should be multiplication
	_, rightIsMul := addOp.Right().(ExpressionOperatorBinArithmetic)
	assert.True(t, rightIsMul)
}

func Test_ParseStringLiteral(t *testing.T) {
	code := `msg: = "Hello, World!"`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseStringLiteral", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	literal, ok := varDecl.Initializer().(ExpressionLiteral)
	assert.True(t, ok)
	assert.NotNil(t, literal)
}

func Test_ParseBooleanLiteral(t *testing.T) {
	code := `flag: = true`
	tokens := lexer.OpenTokenStream(code)
	node, err := Parse("Test_ParseBooleanLiteral", tokens)
	assert.NotNil(t, node)
	assert.Equal(t, 0, len(err), fmt.Sprintf("%v", err))

	cu := node.(CompilationUnit)
	varDecl := cu.Declarations()[0].(VariableDeclarationInferred)

	literal, ok := varDecl.Initializer().(ExpressionLiteral)
	assert.True(t, ok)
	assert.NotNil(t, literal)
}
