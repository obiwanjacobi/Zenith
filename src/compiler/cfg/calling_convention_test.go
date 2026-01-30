package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Z80CallingConvention_FirstParam16Bit(t *testing.T) {
	cc := NewCallingConventionZ80()

	reg, offset, useStack := cc.GetParameterLocation(0, 16)

	assert.False(t, useStack, "First 16-bit param should be in register")
	assert.NotNil(t, reg)
	assert.Equal(t, "HL", reg.Name)
	assert.Equal(t, 0, offset)
}

func Test_Z80CallingConvention_FirstParam8Bit(t *testing.T) {
	cc := NewCallingConventionZ80()

	reg, offset, useStack := cc.GetParameterLocation(0, 8)

	assert.False(t, useStack, "First 8-bit param should be in register")
	assert.NotNil(t, reg)
	assert.Equal(t, "L", reg.Name)
	assert.Equal(t, 0, offset)
}

func Test_Z80CallingConvention_SecondParam16Bit(t *testing.T) {
	cc := NewCallingConventionZ80()

	reg, offset, useStack := cc.GetParameterLocation(1, 16)

	assert.False(t, useStack, "Second 16-bit param should be in register")
	assert.NotNil(t, reg)
	assert.Equal(t, "DE", reg.Name)
	assert.Equal(t, 0, offset)
}

func Test_Z80CallingConvention_FourthParamOnStack(t *testing.T) {
	cc := NewCallingConventionZ80()

	reg, offset, useStack := cc.GetParameterLocation(3, 16)

	assert.True(t, useStack, "Fourth param should be on stack")
	assert.Nil(t, reg)
	assert.Equal(t, 2, offset, "Stack offset should account for return address")
}

func Test_Z80CallingConvention_ReturnValue8Bit(t *testing.T) {
	cc := NewCallingConventionZ80()

	reg := cc.GetReturnValueRegister(8)

	assert.NotNil(t, reg)
	assert.Equal(t, "A", reg.Name)
}

func Test_Z80CallingConvention_ReturnValue16Bit(t *testing.T) {
	cc := NewCallingConventionZ80()

	reg := cc.GetReturnValueRegister(16)

	assert.NotNil(t, reg)
	assert.Equal(t, "HL", reg.Name)
}

func Test_Z80CallingConvention_CallerSavedRegisters(t *testing.T) {
	cc := NewCallingConventionZ80()

	callerSaved := cc.GetCallerSavedRegisters()

	// Should include all general-purpose registers
	assert.Greater(t, len(callerSaved), 0)

	// Check that key registers are included
	hasA, hasHL, hasDE, hasBC := false, false, false, false
	for _, reg := range callerSaved {
		switch reg.Name {
		case "A":
			hasA = true
		case "HL":
			hasHL = true
		case "DE":
			hasDE = true
		case "BC":
			hasBC = true
		}
	}

	assert.True(t, hasA, "A should be caller-saved")
	assert.True(t, hasHL, "HL should be caller-saved")
	assert.True(t, hasDE, "DE should be caller-saved")
	assert.True(t, hasBC, "BC should be caller-saved")
}

func Test_RegisterAllocator_WithCallingConvention(t *testing.T) {
	allocator := NewRegisterAllocator(Z80Registers)
	cc := NewCallingConventionZ80()
	allocator.SetCallingConvention(cc)

	// Build pre-coloring for function with 2 params: foo(x: u16, y: u8)
	precolored := allocator.BuildParameterPrecoloring("main", []string{"x", "y"}, []int{16, 8})

	assert.Equal(t, "HL", precolored["main.x"], "First 16-bit param should be in HL")
	assert.Equal(t, "E", precolored["main.y"], "Second 8-bit param should be in E")
}

func Test_RegisterAllocator_PrecoloredParameters(t *testing.T) {
	// Simple interference graph with pre-colored parameter
	ig := NewInterferenceGraph()
	ig.AddNode("main.x")           // parameter in HL
	ig.AddNode("main.y")           // local variable
	ig.AddEdge("main.x", "main.y") // they interfere

	// Pre-color x to HL (it's a parameter)
	precolored := map[string]string{
		"main.x": "HL",
	}

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.AllocateWithPrecoloring(ig, nil, precolored)

	// x should be in HL (pre-colored)
	assert.Equal(t, "HL", result.Allocation["main.x"])

	// y should get a different register (not HL, since it interferes with x)
	assert.Contains(t, result.Allocation, "main.y")
	assert.NotEqual(t, "HL", result.Allocation["main.y"])
}

func Test_RegisterAllocator_FunctionWithParametersIntegration(t *testing.T) {
	// Test full pipeline: code → CFG → liveness → interference → allocation with parameters
	code := `add: (x: u16, y: u16) u16 {
		result: u16 = x + y
		ret result
	}`

	cfg, liveness, symbolLookup := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// Get parameter names and sizes for the "add" function
	// Parameters: x (16-bit), y (16-bit)
	paramNames := []string{"x", "y"}
	paramSizes := []int{16, 16}

	// Create allocator with calling convention
	allocator := NewRegisterAllocator(Z80Registers)
	cc := NewCallingConventionZ80()
	allocator.SetCallingConvention(cc)

	// Build pre-coloring for parameters
	precolored := allocator.BuildParameterPrecoloring("add", paramNames, paramSizes)

	// Verify pre-coloring map
	require.Equal(t, "HL", precolored["add.x"], "First param should be pre-colored to HL")
	require.Equal(t, "DE", precolored["add.y"], "Second param should be pre-colored to DE")

	// Run allocation with pre-coloring
	result := allocator.AllocateWithPrecoloring(ig, symbolLookup, precolored)

	// Verify parameters are in their ABI registers
	assert.Equal(t, "HL", result.Allocation["add.x"], "Parameter x should be in HL")
	assert.Equal(t, "DE", result.Allocation["add.y"], "Parameter y should be in DE")

	// Verify local variable (result) got allocated
	assert.Contains(t, result.Allocation, "add.result", "Local variable result should be allocated")

	// result should not be in HL or DE if it interferes with params
	if ig.Interferes("add.result", "add.x") {
		assert.NotEqual(t, "HL", result.Allocation["add.result"], "result should not be in HL if it interferes with x")
	}
	if ig.Interferes("add.result", "add.y") {
		assert.NotEqual(t, "DE", result.Allocation["add.result"], "result should not be in DE if it interferes with y")
	}

	t.Logf("Parameter allocation: x=%s, y=%s, result=%s",
		result.Allocation["add.x"],
		result.Allocation["add.y"],
		result.Allocation["add.result"])
}

func Test_RegisterAllocator_ThreeParametersWithStackSpill(t *testing.T) {
	// Test with 3 parameters - third should use BC, fourth would go to stack
	code := `calc: (a: u16, b: u16, c: u16) u16 {
		temp: u16 = a + b
		result: u16 = temp + c
		ret result
	}`

	cfg, liveness, symbolLookup := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	paramNames := []string{"a", "b", "c"}
	paramSizes := []int{16, 16, 16}

	allocator := NewRegisterAllocator(Z80Registers)
	cc := NewCallingConventionZ80()
	allocator.SetCallingConvention(cc)

	precolored := allocator.BuildParameterPrecoloring("calc", paramNames, paramSizes)

	// Verify all three parameters are pre-colored (HL, DE, BC)
	require.Equal(t, "HL", precolored["calc.a"], "First param in HL")
	require.Equal(t, "DE", precolored["calc.b"], "Second param in DE")
	require.Equal(t, "BC", precolored["calc.c"], "Third param in BC")

	result := allocator.AllocateWithPrecoloring(ig, symbolLookup, precolored)

	// Verify parameters are in their ABI registers
	assert.Equal(t, "HL", result.Allocation["calc.a"])
	assert.Equal(t, "DE", result.Allocation["calc.b"])
	assert.Equal(t, "BC", result.Allocation["calc.c"])

	// Locals should get remaining registers or spill
	assert.Contains(t, result.Allocation, "calc.temp")
	assert.Contains(t, result.Allocation, "calc.result")

	t.Logf("Three parameter allocation: a=%s, b=%s, c=%s, temp=%s, result=%s",
		result.Allocation["calc.a"],
		result.Allocation["calc.b"],
		result.Allocation["calc.c"],
		result.Allocation["calc.temp"],
		result.Allocation["calc.result"])

	// With high register pressure, some variables may spill
	if len(result.Spilled) > 0 {
		t.Logf("Spilled variables: %v", result.Spilled)
	}
}

func Test_RegisterAllocator_MixedParameterSizes(t *testing.T) {
	// Test with mixed 8-bit and 16-bit parameters
	code := `process: (flag: u8, ptr: u16) u8 {
		value: u8 = flag
		ret value
	}`

	cfg, liveness, symbolLookup := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	paramNames := []string{"flag", "ptr"}
	paramSizes := []int{8, 16}

	allocator := NewRegisterAllocator(Z80Registers)
	cc := NewCallingConventionZ80()
	allocator.SetCallingConvention(cc)

	precolored := allocator.BuildParameterPrecoloring("process", paramNames, paramSizes)

	// First 8-bit param should use L (low byte of HL)
	// Second 16-bit param should use DE
	require.Equal(t, "L", precolored["process.flag"], "First 8-bit param in L")
	require.Equal(t, "DE", precolored["process.ptr"], "Second 16-bit param in DE")

	result := allocator.AllocateWithPrecoloring(ig, symbolLookup, precolored)

	assert.Equal(t, "L", result.Allocation["process.flag"])
	assert.Equal(t, "DE", result.Allocation["process.ptr"])
	assert.Contains(t, result.Allocation, "process.value")

	t.Logf("Mixed size allocation: flag=%s, ptr=%s, value=%s",
		result.Allocation["process.flag"],
		result.Allocation["process.ptr"],
		result.Allocation["process.value"])
}
