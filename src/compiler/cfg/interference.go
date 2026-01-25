package cfg

import (
	"fmt"
	"sort"
	"zenith/compiler/zir"
)

// InterferenceGraph represents which variables cannot share the same register
// because they are live at the same time
type InterferenceGraph struct {
	// Adjacency list representation: variable -> set of interfering variables
	edges map[string]map[string]bool
	// All variables in the graph
	nodes map[string]bool
}

// NewInterferenceGraph creates a new empty interference graph
func NewInterferenceGraph() *InterferenceGraph {
	return &InterferenceGraph{
		edges: make(map[string]map[string]bool),
		nodes: make(map[string]bool),
	}
}

// AddNode adds a variable to the graph
func (ig *InterferenceGraph) AddNode(variable string) {
	ig.nodes[variable] = true
	if ig.edges[variable] == nil {
		ig.edges[variable] = make(map[string]bool)
	}
}

// AddEdge adds an interference edge between two variables
// This means they cannot share the same register
func (ig *InterferenceGraph) AddEdge(var1, var2 string) {
	// Don't add self-loops
	if var1 == var2 {
		return
	}

	// Ensure both nodes exist
	ig.AddNode(var1)
	ig.AddNode(var2)

	// Add undirected edge
	ig.edges[var1][var2] = true
	ig.edges[var2][var1] = true
}

// Interferes returns true if two variables interfere with each other
func (ig *InterferenceGraph) Interferes(var1, var2 string) bool {
	if neighbors, exists := ig.edges[var1]; exists {
		return neighbors[var2]
	}
	return false
}

// GetNeighbors returns all variables that interfere with the given variable
func (ig *InterferenceGraph) GetNeighbors(variable string) []string {
	neighbors := []string{}
	if edges, exists := ig.edges[variable]; exists {
		for neighbor := range edges {
			neighbors = append(neighbors, neighbor)
		}
	}
	sort.Strings(neighbors)
	return neighbors
}

// GetDegree returns the number of variables that interfere with the given variable
func (ig *InterferenceGraph) GetDegree(variable string) int {
	if edges, exists := ig.edges[variable]; exists {
		return len(edges)
	}
	return 0
}

// GetNodes returns all variables in the graph
func (ig *InterferenceGraph) GetNodes() []string {
	nodes := []string{}
	for node := range ig.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}

// String returns a string representation of the interference graph
func (ig *InterferenceGraph) String() string {
	result := "Interference Graph:\n"
	nodes := ig.GetNodes()
	for _, node := range nodes {
		neighbors := ig.GetNeighbors(node)
		result += fmt.Sprintf("  %s -> %v\n", node, neighbors)
	}
	return result
}

// BuildInterferenceGraph constructs an interference graph from liveness information
// Two variables interfere if they are both live at the same point in the program
func BuildInterferenceGraph(cfg *CFG, liveness *LivenessInfo) *InterferenceGraph {
	ig := NewInterferenceGraph()

	// For each block, we need to track live variables as we process statements
	for _, block := range cfg.Blocks {
		// Start with live-out for this block (variables live at the end)
		currentlyLive := make(map[string]bool)
		for v := range liveness.LiveOut[block.ID] {
			currentlyLive[v] = true
			ig.AddNode(v)
		}

		// Process statements in reverse order (backward through the block)
		for i := len(block.Instructions) - 1; i >= 0; i-- {
			stmt := block.Instructions[i]

			// Get variables used and defined in this statement
			used := getUsedInStatement(stmt)
			defined := getDefinedInStatement(stmt)

			// Any variable defined at this point interferes with all currently live variables
			for defVar := range defined {
				ig.AddNode(defVar)
				for liveVar := range currentlyLive {
					ig.AddEdge(defVar, liveVar)
				}
			}

			// After the definition, remove the defined variable from live set
			// (it's no longer live before its definition)
			for defVar := range defined {
				delete(currentlyLive, defVar)
			}

			// Add all used variables to the currently live set
			for usedVar := range used {
				currentlyLive[usedVar] = true
				ig.AddNode(usedVar)
			}
		}

		// At the start of the block, all variables in currentlyLive interfere with each other
		liveVars := []string{}
		for v := range currentlyLive {
			liveVars = append(liveVars, v)
		}
		for i := 0; i < len(liveVars); i++ {
			for j := i + 1; j < len(liveVars); j++ {
				ig.AddEdge(liveVars[i], liveVars[j])
			}
		}
	}

	return ig
}

// Helper functions to get used/defined variables in a statement
func getUsedInStatement(stmt zir.IRStatement) map[string]bool {
	used := make(map[string]bool)
	switch s := stmt.(type) {
	case *zir.IRVariableDecl:
		if s.Initializer != nil {
			for _, v := range getUsedInExpression(s.Initializer) {
				used[v] = true
			}
		}
	case *zir.IRAssignment:
		// Right-hand side uses variables
		for _, v := range getUsedInExpression(s.Value) {
			used[v] = true
		}
	case *zir.IRExpressionStmt:
		for _, v := range getUsedInExpression(s.Expression) {
			used[v] = true
		}
	}
	return used
}

func getDefinedInStatement(stmt zir.IRStatement) map[string]bool {
	defined := make(map[string]bool)
	switch s := stmt.(type) {
	case *zir.IRVariableDecl:
		defined[s.Symbol.Name] = true
	case *zir.IRAssignment:
		// Left-hand side defines a variable
		defined[s.Target.Name] = true
	}
	return defined
}
