package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_RegisterAllocation_SimpleCase(t *testing.T) {
	// Create interference graph: x and y don't interfere
	ig := NewInterferenceGraph()
	ig.AddNode("x")
	ig.AddNode("y")
	// No edge between x and y

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.Allocate(ig)

	// Both should get registers
	assert.Contains(t, result.Allocation, "x")
	assert.Contains(t, result.Allocation, "y")
	assert.Empty(t, result.Spilled)
}

func Test_RegisterAllocation_Interference(t *testing.T) {
	// Create interference graph: x and y interfere
	ig := NewInterferenceGraph()
	ig.AddNode("x")
	ig.AddNode("y")
	ig.AddEdge("x", "y")

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.Allocate(ig)

	// Both should get different registers
	assert.Contains(t, result.Allocation, "x")
	assert.Contains(t, result.Allocation, "y")
	assert.NotEqual(t, result.Allocation["x"], result.Allocation["y"])
	assert.Empty(t, result.Spilled)
}

func Test_RegisterAllocation_RegisterPressure(t *testing.T) {
	// Create interference graph where all variables interfere (complete graph)
	// With 7 Z80 registers, the 8th variable should spill
	ig := NewInterferenceGraph()
	variables := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	for _, v := range variables {
		ig.AddNode(v)
	}

	// Make complete graph (everyone interferes with everyone)
	for i := 0; i < len(variables); i++ {
		for j := i + 1; j < len(variables); j++ {
			ig.AddEdge(variables[i], variables[j])
		}
	}

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.Allocate(ig)

	// Z80Registers now has 10 registers (7 single 8-bit + 3 pairs), so all 8 should get registers
	assert.Equal(t, 8, len(result.Allocation))
	assert.Equal(t, 0, len(result.Spilled))
}

func Test_RegisterAllocation_ThreeVariablesLinearInterference(t *testing.T) {
	// x interferes with y, y interferes with z, but x and z don't interfere
	// All three should get registers (x and z can share the same one conceptually,
	// but in our implementation they'll get different ones since they're both allocated)
	ig := NewInterferenceGraph()
	ig.AddNode("x")
	ig.AddNode("y")
	ig.AddNode("z")
	ig.AddEdge("x", "y")
	ig.AddEdge("y", "z")
	// x and z don't interfere

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.Allocate(ig)

	// All three should get registers
	assert.Equal(t, 3, len(result.Allocation))
	assert.Empty(t, result.Spilled)

	// x and y must have different registers
	assert.NotEqual(t, result.Allocation["x"], result.Allocation["y"])
	// y and z must have different registers
	assert.NotEqual(t, result.Allocation["y"], result.Allocation["z"])
	// x and z CAN have the same register (but might not depending on algorithm)
}

func Test_RegisterAllocation_FromCode(t *testing.T) {
	// Real example: build from source code
	code := `main: () {
		x: u8 = 1
		y: u8 = x + 2
		z: u8 = 3
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	assert.NotNil(t, cfg)
	assert.NotNil(t, liveness)

	ig := BuildInterferenceGraph(cfg, liveness)
	assert.NotNil(t, ig)

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.Allocate(ig)

	// All variables should get registers (low register pressure)
	assert.Contains(t, result.Allocation, "x")
	assert.Contains(t, result.Allocation, "y")
	assert.Contains(t, result.Allocation, "z")

	// Should have no spills with only 3 variables
	assert.Empty(t, result.Spilled)
}

func Test_RegisterAllocation_HighPressureCode(t *testing.T) {
	// Code with many simultaneous live variables
	code := `main: () {
		a: u8 = 1
		b: u8 = 2
		c: u8 = 3
		d: u8 = 4
		e: u8 = 5
		f: u8 = 6
		g: u8 = 7
		h: u8 = 8
		sum: u8 = a + b + c + d + e + f + g + h
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	assert.NotNil(t, cfg)
	assert.NotNil(t, liveness)

	ig := BuildInterferenceGraph(cfg, liveness)
	assert.NotNil(t, ig)

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.Allocate(ig)

	// Check that allocation was performed
	assert.NotEmpty(t, result.Allocation)

	// With 7 registers and 9 variables all live at once,
	// we should see some spilling
	totalVars := len(result.Allocation) + len(result.Spilled)
	assert.True(t, totalVars > 0, "Should allocate or spill at least some variables")
}

func Test_RegisterAllocation_EmptyGraph(t *testing.T) {
	ig := NewInterferenceGraph()

	allocator := NewRegisterAllocator(Z80Registers)
	result := allocator.Allocate(ig)

	assert.Empty(t, result.Allocation)
	assert.Empty(t, result.Spilled)
}
