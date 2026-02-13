package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseMemberAccessInExpression(t *testing.T) {
	code := `test: () {
		x := obj.field / 2
	}`
	cu := parseCode(t, "Test_ParseMemberAccessInExpression", code)
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseMemberAccessInForLoop(t *testing.T) {
	code := `test: () {
		for i := 0; i < arr.length; i++ {
		}
	}`
	cu := parseCode(t, "Test_ParseMemberAccessInForLoop", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseReverseSimplified(t *testing.T) {
	code := `reverse: (arr: u8[]) {
		l := arr.length / 2
	}`
	cu := parseCode(t, "Test_ParseReverseSimplified", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseReverseWithFor(t *testing.T) {
	code := `reverse: (arr: u8[]) {
		l := arr.length / 2
		for i := 0; i < l ; i++ {
		}
	}`
	cu := parseCode(t, "Test_ParseReverseWithFor", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}

func Test_ParseReverseFull(t *testing.T) {
	code := `reverse: (arr: u8[]) {
		l := arr.length / 2
		for i := 0; i < l ; i++ {
			tmp := arr[i]
			arr[i] = arr[l - 1 - i]
			arr[l - 1 - i] = tmp
		}
	}`
	cu := parseCode(t, "Test_ParseReverseFull", code)
	for _, err := range cu.Errors() {
		t.Logf("Error: %s", err.Error())
	}
	assert.Equal(t, 0, len(cu.Errors()))
}
