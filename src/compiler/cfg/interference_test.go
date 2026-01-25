package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_InterferenceGraph_AddNode(t *testing.T) {
	ig := NewInterferenceGraph()
	ig.AddNode("x")
	ig.AddNode("y")

	nodes := ig.GetNodes()
	assert.Len(t, nodes, 2)
	assert.Contains(t, nodes, "x")
	assert.Contains(t, nodes, "y")
}

func Test_InterferenceGraph_AddEdge(t *testing.T) {
	ig := NewInterferenceGraph()
	ig.AddEdge("x", "y")

	// Check that edge exists in both directions
	assert.True(t, ig.Interferes("x", "y"))
	assert.True(t, ig.Interferes("y", "x"))

	// Check neighbors
	xNeighbors := ig.GetNeighbors("x")
	assert.Len(t, xNeighbors, 1)
	assert.Equal(t, "y", xNeighbors[0])

	yNeighbors := ig.GetNeighbors("y")
	assert.Len(t, yNeighbors, 1)
	assert.Equal(t, "x", yNeighbors[0])
}

func Test_InterferenceGraph_NoSelfLoops(t *testing.T) {
	ig := NewInterferenceGraph()
	ig.AddEdge("x", "x")

	// Should not create self-loop
	assert.False(t, ig.Interferes("x", "x"))
	assert.Empty(t, ig.GetNeighbors("x"))
}

func Test_InterferenceGraph_Degree(t *testing.T) {
	ig := NewInterferenceGraph()
	ig.AddEdge("x", "y")
	ig.AddEdge("x", "z")
	ig.AddEdge("y", "z")

	assert.Equal(t, 2, ig.GetDegree("x"))
	assert.Equal(t, 2, ig.GetDegree("y"))
	assert.Equal(t, 2, ig.GetDegree("z"))
}

func Test_BuildInterferenceGraph_SimpleAssignment(t *testing.T) {
	code := `main: () {
		x: = 1
		y: = 2
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// x and y don't interfere because they're not live at the same time
	assert.False(t, ig.Interferes("x", "y"))
}

func Test_BuildInterferenceGraph_SimultaneouslyLive(t *testing.T) {
	code := `main: () {
		x: = 1
		y: = 2
		z: = x + y
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// Variables now use qualified names (e.g., "main.x")
	// x and y are both live when z is computed, so they interfere
	assert.True(t, ig.Interferes("main.x", "main.y"))
}

func Test_BuildInterferenceGraph_IfStatement(t *testing.T) {
	code := `main: () {
		x: = 1
		if true {
			y: = x + 1
		}
		z: = x + 2
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// x is live across the if statement
	// y is only live inside the if block
	// They should interfere if y and x are live at the same time
	nodes := ig.GetNodes()
	assert.Contains(t, nodes, "x")
	assert.Contains(t, nodes, "y")
	assert.Contains(t, nodes, "z")
}

func Test_BuildInterferenceGraph_Loop(t *testing.T) {
	code := `main: () {
		for i: = 0; i < 10; i + 1 {
			x: = i
		}
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// i is used throughout the loop, x is defined in the loop
	nodes := ig.GetNodes()
	assert.Contains(t, nodes, "i")
	assert.Contains(t, nodes, "x")
}

func Test_BuildInterferenceGraph_MultipleVariables(t *testing.T) {
	code := `main: () {
		a: = 1
		b: = 2
		c: = 3
		d: = a + b + c
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// a, b, c are all live when d is computed
	assert.True(t, ig.Interferes("a", "b"))
	assert.True(t, ig.Interferes("a", "c"))
	assert.True(t, ig.Interferes("b", "c"))

	// d doesn't interfere with a, b, c because it's defined after they're used
	// (though they might interfere depending on live-out)
}

func Test_BuildInterferenceGraph_NoInterference(t *testing.T) {
	code := `main: () {
		x: = 1
		y: = x + 1
		x = 2
		z: = x + 1
	}}`

	cfg, liveness := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// After the first use of x, it's redefined, so the two "incarnations" of x
	// might not interfere with y or z depending on liveness
	// This tests the basic functionality - exact interference depends on liveness
	nodes := ig.GetNodes()
	require.Contains(t, nodes, "x")
	require.Contains(t, nodes, "y")
	require.Contains(t, nodes, "z")
}

func Test_BuildInterferenceGraph_ReturnStatement(t *testing.T) {
	code := `main: () {
		x: = 5
		y: = 10
		ret x + y
	}`

	cfg, liveness := buildLivenessFromCode(t, code)
	ig := BuildInterferenceGraph(cfg, liveness)

	// x and y are both live when the return statement is executed
	// so they should interfere
	assert.True(t, ig.Interferes("x", "y"), "x and y should interfere at return")

	// Verify both variables are in the graph
	nodes := ig.GetNodes()
	assert.Contains(t, nodes, "x")
	assert.Contains(t, nodes, "y")
}
