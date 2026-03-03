package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseAssignmentWithSubscript(t *testing.T) {
	code := `test: () {
		arr[0] = 5
	}`
	cu := parseCode(t, "Test_ParseAssignmentWithSubscript", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseAssignmentWithSubscriptVar(t *testing.T) {
	code := `test: () {
		arr[i] = 5
	}`
	cu := parseCode(t, "Test_ParseAssignmentWithSubscriptVar", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseAssignmentWithSubscriptExpression(t *testing.T) {
	code := `test: () {
		arr[l - 1 - i] = 5
	}`
	cu := parseCode(t, "Test_ParseAssignmentWithSubscriptExpression", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseAssignmentWithSubscriptOnRHS(t *testing.T) {
	code := `test: () {
		arr[i] = arr[l - 1 - i]
	}`
	cu := parseCode(t, "Test_ParseAssignmentWithSubscriptOnRHS", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseMultipleAssignmentsWithSubscripts(t *testing.T) {
	code := `test: () {
		tmp := arr[i]
		arr[i] = arr[l - 1 - i]
		arr[l - 1 - i] = tmp
	}`
	cu := parseCode(t, "Test_ParseMultipleAssignmentsWithSubscripts", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseForWithSubscriptAssignments(t *testing.T) {
	code := `test: () {
		for i := 0; i < 5; i++ {
			tmp := arr[i]
			arr[i] = arr[5 - 1 - i]
			arr[5 - 1 - i] = tmp
		}
	}`
	cu := parseCode(t, "Test_ParseForWithSubscriptAssignments", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseReverseWithL(t *testing.T) {
	code := `reverse: (arr: u8[]) {
		l := 5
		for i := 0; i < l ; i++ {
			tmp := arr[i]
			arr[i] = arr[l - 1 - i]
			arr[l - 1 - i] = tmp
		}
	}`
	cu := parseCode(t, "Test_ParseReverseWithL", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseComplexSubscript(t *testing.T) {
	code := `complex: () {
		call1().field1[i + 1] = call2().field2[j - 1]
	}`
	cu := parseCode(t, "Test_ParseComplexSubscript", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

// Test_ParseSubscriptAsLValue_Structure verifies that a subscript expression
// used as an l-value in an assignment produces the correct AST structure:
// the first child of VariableAssignment must be an ExpressionSubscript,
// not merely an ExpressionIdentifier (i.e. the index must not be dropped).
func Test_ParseSubscriptAsLValue_Structure(t *testing.T) {
	code := `test: () {
		arr[i] = 5
	}`
	cu := parseCode(t, "Test_ParseSubscriptAsLValue_Structure", code)
	assert.Equal(t, 0, len(cu.Errors()))

	funcDecl, ok := cu.Declarations()[0].(FunctionDeclaration)
	assert.True(t, ok, "expected a function declaration")

	stmts := funcDecl.Body().Statements()
	assert.Equal(t, 1, len(stmts), "expected one statement")

	assign, ok := stmts[0].(VariableAssignment)
	assert.True(t, ok, "expected VariableAssignment")

	// The l-value (first child) must be an ExpressionSubscript, not just an identifier.
	children := assign.Children()
	assert.GreaterOrEqual(t, len(children), 2, "expected at least two children (lvalue, rvalue)")
	assert.NotEqual(t, children[0], children[1])

	subscript, ok := children[0].(ExpressionSubscript)
	assert.True(t, ok, "l-value should be ExpressionSubscript, not %T", children[0])

	if ok {
		arrayExpr, ok := subscript.Array().(ExpressionIdentifier)
		assert.True(t, ok, "array part of subscript should be ExpressionIdentifier")
		if ok {
			assert.Equal(t, "arr", arrayExpr.Identifier().Text(), "array name should be 'arr'")
		}

		indexExpr, ok := subscript.Index().(ExpressionIdentifier)
		assert.True(t, ok, "index part of subscript should be ExpressionIdentifier")
		if ok {
			assert.Equal(t, "i", indexExpr.Identifier().Text(), "index name should be 'i'")
		}
	}

	// The r-value must be present.
	assert.NotNil(t, assign.Expression(), "r-value expression should not be nil")
}
