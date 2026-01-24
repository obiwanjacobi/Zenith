package zir

import (
	"fmt"
	"testing"

	"zenith/compiler/lexer"
	"zenith/compiler/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to parse code and run semantic analysis
func analyzeCode(t *testing.T, testName string, code string) (*IRCompilationUnit, []*IRError) {
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
	irCU, irErrors := analyzer.Analyze(cu)

	return irCU, irErrors
}

// Helper function to require no errors
func requireNoErrors(t *testing.T, errors []*IRError) {
	if len(errors) > 0 {
		for _, err := range errors {
			t.Logf("IR Error: %s", err.Error())
		}
	}
	require.Equal(t, 0, len(errors), "Expected no IR errors")
}

// ============================================================================
// Variable Declaration Tests
// ============================================================================

func Test_Analyze_VarDeclWithType(t *testing.T) {
	code := "count: u8"
	irCU, errors := analyzeCode(t, "Test_Analyze_VarDeclWithType", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(irCU.Declarations))

	varDecl, ok := irCU.Declarations[0].(*IRVariableDecl)
	require.True(t, ok, "Declaration should be IRVariableDecl")
	assert.Equal(t, "count", varDecl.Symbol.Name)
	assert.Equal(t, U8Type, varDecl.Symbol.Type)
	assert.Nil(t, varDecl.Initializer)
}

func Test_Analyze_VarDeclWithTypeAndInit(t *testing.T) {
	code := "count: u16 = 42"
	irCU, errors := analyzeCode(t, "Test_Analyze_VarDeclWithTypeAndInit", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(irCU.Declarations))

	varDecl, ok := irCU.Declarations[0].(*IRVariableDecl)
	require.True(t, ok)
	assert.Equal(t, "count", varDecl.Symbol.Name)
	assert.Equal(t, U16Type, varDecl.Symbol.Type)
	assert.NotNil(t, varDecl.Initializer)

	// Check initializer is a constant
	constant, ok := varDecl.Initializer.(*IRConstant)
	assert.True(t, ok, "Initializer should be IRConstant")
	assert.Equal(t, U8Type, constant.Type()) // TODO: Should parse as correct type
}

func Test_Analyze_VarDeclInferred(t *testing.T) {
	code := "value: = 100"
	irCU, errors := analyzeCode(t, "Test_Analyze_VarDeclInferred", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(irCU.Declarations))

	varDecl, ok := irCU.Declarations[0].(*IRVariableDecl)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_FunctionDeclaration", code)
	requireNoErrors(t, errors)

	require.Equal(t, 1, len(irCU.Declarations))

	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
	require.True(t, ok, "Declaration should be IRFunctionDecl")
	assert.Equal(t, "main", funcDecl.Name)
	assert.Equal(t, 0, len(funcDecl.Parameters))
	assert.Nil(t, funcDecl.ReturnType)
	assert.NotNil(t, funcDecl.Body)
	assert.NotNil(t, funcDecl.Scope)
}

func Test_Analyze_FunctionWithParameters(t *testing.T) {
	code := `add: (a: u8, b: u8) {
	}`
	irCU, errors := analyzeCode(t, "Test_Analyze_FunctionWithParameters", code)
	requireNoErrors(t, errors)

	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_FunctionWithReturnType", code)
	requireNoErrors(t, errors)

	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_FunctionWithLocalVar", code)
	requireNoErrors(t, errors)

	funcDecl, ok := irCU.Declarations[0].(*IRFunctionDecl)
	require.True(t, ok)
	require.Equal(t, 1, len(funcDecl.Body.Statements))

	varDecl, ok := funcDecl.Body.Statements[0].(*IRVariableDecl)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_TypeDeclaration", code)

	// Debug: print errors
	for _, err := range errors {
		t.Logf("Error: %s", err.Error())
	}
	t.Logf("Number of declarations: %d", len(irCU.Declarations))

	requireNoErrors(t, errors)

	require.Equal(t, 1, len(irCU.Declarations))

	typeDecl, ok := irCU.Declarations[0].(*IRTypeDecl)
	require.True(t, ok, "Declaration should be IRTypeDecl")
	assert.Equal(t, "Point", typeDecl.Type.Name())

	fields := typeDecl.Type.Fields()
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
	irCU, errors := analyzeCode(t, "Test_Analyze_TypeDeclarationUsage", code)
	requireNoErrors(t, errors)

	require.Equal(t, 2, len(irCU.Declarations))

	varDecl, ok := irCU.Declarations[1].(*IRVariableDecl)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_Assignment", code)
	requireNoErrors(t, errors)

	funcDecl := irCU.Declarations[0].(*IRFunctionDecl)
	require.Equal(t, 2, len(funcDecl.Body.Statements))

	assignment, ok := funcDecl.Body.Statements[1].(*IRAssignment)
	require.True(t, ok, "Second statement should be IRAssignment")
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

func Test_Analyze_IfStatement(t *testing.T) {
	code := `main: () {
		if true {
			x: = 1
		}
	}`
	irCU, errors := analyzeCode(t, "Test_Analyze_IfStatement", code)
	requireNoErrors(t, errors)

	funcDecl := irCU.Declarations[0].(*IRFunctionDecl)
	require.Equal(t, 1, len(funcDecl.Body.Statements))

	ifStmt, ok := funcDecl.Body.Statements[0].(*IRIf)
	require.True(t, ok, "Statement should be IRIf")
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
	irCU, errors := analyzeCode(t, "Test_Analyze_IfElseStatement", code)
	requireNoErrors(t, errors)

	funcDecl := irCU.Declarations[0].(*IRFunctionDecl)
	ifStmt := funcDecl.Body.Statements[0].(*IRIf)

	assert.NotNil(t, ifStmt.ThenBlock)
	assert.NotNil(t, ifStmt.ElseBlock)
	assert.Equal(t, 1, len(ifStmt.ThenBlock.Statements))
	assert.Equal(t, 1, len(ifStmt.ElseBlock.Statements))
}

// ============================================================================
// Expression Tests
// ============================================================================

func Test_Analyze_BinaryOperation(t *testing.T) {
	code := `main: () {
		result: = 5 + 3
	}`
	irCU, errors := analyzeCode(t, "Test_Analyze_BinaryOperation", code)
	requireNoErrors(t, errors)

	funcDecl := irCU.Declarations[0].(*IRFunctionDecl)
	varDecl := funcDecl.Body.Statements[0].(*IRVariableDecl)

	binOp, ok := varDecl.Initializer.(*IRBinaryOp)
	require.True(t, ok, "Initializer should be IRBinaryOp")
	assert.Equal(t, OpAdd, binOp.Op)
	assert.NotNil(t, binOp.Left)
	assert.NotNil(t, binOp.Right)
}

func Test_Analyze_BooleanLiteral(t *testing.T) {
	code := `flag: = true`
	irCU, errors := analyzeCode(t, "Test_Analyze_BooleanLiteral", code)
	requireNoErrors(t, errors)

	varDecl := irCU.Declarations[0].(*IRVariableDecl)
	constant, ok := varDecl.Initializer.(*IRConstant)
	require.True(t, ok)
	assert.Equal(t, true, constant.Value)
	assert.Equal(t, BoolType, constant.Type())
}

func Test_Analyze_StringLiteral(t *testing.T) {
	code := `msg: = "hello"`
	irCU, errors := analyzeCode(t, "Test_Analyze_StringLiteral", code)
	requireNoErrors(t, errors)

	varDecl := irCU.Declarations[0].(*IRVariableDecl)
	constant, ok := varDecl.Initializer.(*IRConstant)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_FunctionCall", code)
	requireNoErrors(t, errors)

	mainFunc := irCU.Declarations[1].(*IRFunctionDecl)
	exprStmt := mainFunc.Body.Statements[0].(*IRExpressionStmt)

	funcCall, ok := exprStmt.Expression.(*IRFunctionCall)
	require.True(t, ok, "Expression should be IRFunctionCall")
	assert.Equal(t, "doSomething", funcCall.Function.Name)
	assert.Equal(t, 0, len(funcCall.Arguments))
}

func Test_Analyze_FunctionCallWithArgs(t *testing.T) {
	code := `add: (a: u8, b: u8) {
	}
	main: () {
		add(5, 10)
	}`
	irCU, errors := analyzeCode(t, "Test_Analyze_FunctionCallWithArgs", code)
	requireNoErrors(t, errors)

	mainFunc := irCU.Declarations[1].(*IRFunctionDecl)
	exprStmt := mainFunc.Body.Statements[0].(*IRExpressionStmt)

	funcCall := exprStmt.Expression.(*IRFunctionCall)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_ScopeParameterAccess", code)
	requireNoErrors(t, errors)

	funcDecl := irCU.Declarations[0].(*IRFunctionDecl)
	varDecl := funcDecl.Body.Statements[0].(*IRVariableDecl)

	// The initializer should reference the parameter
	assert.NotNil(t, varDecl.Initializer)
}

func Test_Analyze_ScopeGlobalAccess(t *testing.T) {
	code := `global: u8 = 42
	myFunc: () {
		local: = global
	}`
	irCU, errors := analyzeCode(t, "Test_Analyze_ScopeGlobalAccess", code)
	requireNoErrors(t, errors)

	require.Equal(t, 2, len(irCU.Declarations))
	funcDecl := irCU.Declarations[1].(*IRFunctionDecl)
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
			irCU, errors := analyzeCode(t, "Test_Analyze_BuiltinType_"+typeName, code)
			requireNoErrors(t, errors)

			varDecl := irCU.Declarations[0].(*IRVariableDecl)
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
	irCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_Simple", code)
	requireNoErrors(t, errors)

	// Check call graph
	assert.NotNil(t, irCU.CallGraph)

	// main should call helper
	mainCallees := irCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "helper", mainCallees[0])

	// helper should have no callees
	helperCallees := irCU.CallGraph.GetCallees("helper")
	assert.Equal(t, 0, len(helperCallees))

	// Both functions should be registered
	allFuncs := irCU.CallGraph.GetAllFunctions()
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
	irCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_Chain", code)
	requireNoErrors(t, errors)

	// main -> helper
	mainCallees := irCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "helper", mainCallees[0])

	// helper -> worker
	helperCallees := irCU.CallGraph.GetCallees("helper")
	assert.Equal(t, 1, len(helperCallees))
	assert.Equal(t, "worker", helperCallees[0])

	// worker has no callees
	workerCallees := irCU.CallGraph.GetCallees("worker")
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
	irCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_MultipleCalls", code)
	requireNoErrors(t, errors)

	// main should call both foo and bar (foo should only appear once despite 2 calls)
	mainCallees := irCU.CallGraph.GetCallees("main")
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
	irCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedInIf", code)
	requireNoErrors(t, errors)

	// Call inside if block should still be recorded
	mainCallees := irCU.CallGraph.GetCallees("main")
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
	irCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedInIfElse", code)
	requireNoErrors(t, errors)

	// Both calls should be recorded
	mainCallees := irCU.CallGraph.GetCallees("main")
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
			5 { helper() }
		}
	}`
	irCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedInSelect", code)
	requireNoErrors(t, errors)

	// Call inside select block should be recorded
	mainCallees := irCU.CallGraph.GetCallees("main")
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
	irCU, errors := analyzeCode(t, "Test_Analyze_CallGraph_NestedFunctionCalls", code)
	requireNoErrors(t, errors)

	// main -> foo
	mainCallees := irCU.CallGraph.GetCallees("main")
	assert.Equal(t, 1, len(mainCallees))
	assert.Equal(t, "foo", mainCallees[0])

	// foo -> bar
	fooCallees := irCU.CallGraph.GetCallees("foo")
	assert.Equal(t, 1, len(fooCallees))
	assert.Equal(t, "bar", fooCallees[0])

	// bar has no callees
	barCallees := irCU.CallGraph.GetCallees("bar")
	assert.Equal(t, 0, len(barCallees))

	// All three functions should be in the graph
	allFuncs := irCU.CallGraph.GetAllFunctions()
	assert.Equal(t, 3, len(allFuncs))
}
