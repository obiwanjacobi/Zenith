package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Z80CallingConvention_FirstParam16Bit(t *testing.T) {
	cc := NewCallingConvention_Z80()

	reg, offset, useStack := cc.GetParameterLocation(0, 16)

	assert.False(t, useStack, "First 16-bit param should be in register")
	assert.NotNil(t, reg)
	assert.Equal(t, "HL", reg.Name)
	assert.Equal(t, 0, offset)
}

func Test_Z80CallingConvention_FirstParam8Bit(t *testing.T) {
	cc := NewCallingConvention_Z80()

	reg, offset, useStack := cc.GetParameterLocation(0, 8)

	assert.False(t, useStack, "First 8-bit param should be in register")
	assert.NotNil(t, reg)
	assert.Equal(t, "L", reg.Name)
	assert.Equal(t, 0, offset)
}

func Test_Z80CallingConvention_SecondParam16Bit(t *testing.T) {
	cc := NewCallingConvention_Z80()

	reg, offset, useStack := cc.GetParameterLocation(1, 16)

	assert.False(t, useStack, "Second 16-bit param should be in register")
	assert.NotNil(t, reg)
	assert.Equal(t, "DE", reg.Name)
	assert.Equal(t, 0, offset)
}

func Test_Z80CallingConvention_FourthParamOnStack(t *testing.T) {
	cc := NewCallingConvention_Z80()

	reg, offset, useStack := cc.GetParameterLocation(3, 16)

	assert.True(t, useStack, "Fourth param should be on stack")
	assert.Nil(t, reg)
	assert.Equal(t, 2, offset, "Stack offset should account for return address")
}

func Test_Z80CallingConvention_ReturnValue8Bit(t *testing.T) {
	cc := NewCallingConvention_Z80()

	reg := cc.GetReturnValueRegister(8)

	assert.NotNil(t, reg)
	assert.Equal(t, "A", reg.Name)
}

func Test_Z80CallingConvention_ReturnValue16Bit(t *testing.T) {
	cc := NewCallingConvention_Z80()

	reg := cc.GetReturnValueRegister(16)

	assert.NotNil(t, reg)
	assert.Equal(t, "HL", reg.Name)
}

func Test_Z80CallingConvention_CallerSavedRegisters(t *testing.T) {
	cc := NewCallingConvention_Z80()

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
	cc := NewCallingConvention_Z80()
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
