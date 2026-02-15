package zsm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Return Type Validation Tests
// ============================================================================

func Test_Analyze_FunctionReturnPrimitiveType_Valid(t *testing.T) {
	code := `getValue: () u8 {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnPrimitiveType_Valid", code)
	requireNoErrors(t, errors)
}

func Test_Analyze_FunctionReturnU16_Valid(t *testing.T) {
	code := `getValue: () u16 {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnU16_Valid", code)
	requireNoErrors(t, errors)
}

func Test_Analyze_FunctionReturnUnsizedArray_Valid(t *testing.T) {
	code := `getValue: (arr: u8[]) u8[] {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnUnsizedArray_Valid", code)
	requireNoErrors(t, errors)
}

func Test_Analyze_FunctionReturnStruct_Invalid(t *testing.T) {
	code := `
	struct Point {
		x: u8,
		y: u8
	}
	
	getPoint: () Point {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnStruct_Invalid", code)
	assert.Equal(t, 1, len(errors))
	assert.Contains(t, errors[0].Error(), "cannot return struct type 'Point' by value")
}

func Test_Analyze_FunctionReturnVoid_Valid(t *testing.T) {
	code := `doSomething: () {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnVoid_Valid", code)
	requireNoErrors(t, errors)
}

func Test_Analyze_FunctionReturnBit_Valid(t *testing.T) {
	code := `isValid: () bit {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnBit_Valid", code)
	requireNoErrors(t, errors)
}

func Test_Analyze_FunctionReturnI16_Valid(t *testing.T) {
	code := `getValue: () i16 {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnI16_Valid", code)
	requireNoErrors(t, errors)
}

func Test_Analyze_FunctionReturnStructInComplexProgram_Invalid(t *testing.T) {
	code := `
	struct Vector {
		x: i16,
		y: i16,
		z: i16
	}
	
	add: (a: Vector, b: Vector) Vector {
	}
	
	main: () {
		v1: Vector
		v2: Vector
	}`
	_, errors := analyzeCode(t, "Test_Analyze_FunctionReturnStructInComplexProgram_Invalid", code)
	assert.Equal(t, 1, len(errors))
	assert.Contains(t, errors[0].Error(), "cannot return struct type 'Vector' by value")
}

func Test_Analyze_MultipleStructReturnErrors(t *testing.T) {
	code := `
	struct Point {
		x: u8,
		y: u8
	}
	
	struct Size {
		w: u8,
		h: u8
	}
	
	getPoint: () Point {
	}
	
	getSize: () Size {
	}`
	_, errors := analyzeCode(t, "Test_Analyze_MultipleStructReturnErrors", code)
	assert.Equal(t, 2, len(errors))
	assert.Contains(t, errors[0].Error(), "cannot return struct type")
	assert.Contains(t, errors[1].Error(), "cannot return struct type")
}
