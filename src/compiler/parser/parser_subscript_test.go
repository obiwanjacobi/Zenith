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
