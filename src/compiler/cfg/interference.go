package cfg

import (
	"fmt"
	"sort"
)

// InterferenceGraph represents which VirtualRegisters cannot share the same physical register
// because they are live at the same time
type InterferenceGraph struct {
	// Adjacency list representation: VR ID -> set of interfering VR IDs
	edges map[int]map[int]bool
	// All VirtualRegister IDs in the graph
	nodes map[int]bool
}

// NewInterferenceGraph creates a new empty interference graph
func NewInterferenceGraph() *InterferenceGraph {
	return &InterferenceGraph{
		edges: make(map[int]map[int]bool),
		nodes: make(map[int]bool),
	}
}

// AddNode adds a VirtualRegister to the graph
func (ig *InterferenceGraph) AddNode(vrID int) {
	ig.nodes[vrID] = true
	if ig.edges[vrID] == nil {
		ig.edges[vrID] = make(map[int]bool)
	}
}

// AddEdge adds an interference edge between two VirtualRegisters
// This means they cannot share the same physical register
func (ig *InterferenceGraph) AddEdge(vr1, vr2 int) {
	// Don't add self-loops
	if vr1 == vr2 {
		return
	}

	// Ensure both nodes exist
	ig.AddNode(vr1)
	ig.AddNode(vr2)

	// Add undirected edge
	ig.edges[vr1][vr2] = true
	ig.edges[vr2][vr1] = true
}

// Interferes returns true if two VirtualRegisters interfere with each other
func (ig *InterferenceGraph) Interferes(vr1, vr2 int) bool {
	if neighbors, exists := ig.edges[vr1]; exists {
		return neighbors[vr2]
	}
	return false
}

// GetNeighbors returns all VirtualRegister IDs that interfere with the given VR
func (ig *InterferenceGraph) GetNeighbors(vrID int) []int {
	neighbors := []int{}
	if edges, exists := ig.edges[vrID]; exists {
		for neighbor := range edges {
			neighbors = append(neighbors, neighbor)
		}
	}
	sort.Ints(neighbors)
	return neighbors
}

// GetDegree returns the number of VirtualRegisters that interfere with the given VR
func (ig *InterferenceGraph) GetDegree(vrID int) int {
	if edges, exists := ig.edges[vrID]; exists {
		return len(edges)
	}
	return 0
}

// GetNodes returns all VirtualRegister IDs in the graph
func (ig *InterferenceGraph) GetNodes() []int {
	nodes := []int{}
	for node := range ig.nodes {
		nodes = append(nodes, node)
	}
	sort.Ints(nodes)
	return nodes
}

// String returns a string representation of the interference graph
func (ig *InterferenceGraph) String() string {
	result := "Interference Graph:\n"
	nodes := ig.GetNodes()
	for _, node := range nodes {
		neighbors := ig.GetNeighbors(node)
		result += fmt.Sprintf("  VR%d -> %v\n", node, neighbors)
	}
	return result
}

// BuildInterferenceGraph constructs an interference graph from liveness information
// Two VirtualRegisters interfere if they are both live at the same point in the program
func BuildInterferenceGraph(cfg *CFG, liveness *LivenessInfo) *InterferenceGraph {
	ig := NewInterferenceGraph()

	// For each block, track live VRs as we process machine instructions
	for _, block := range cfg.Blocks {
		// Start with live-out for this block (VRs live at the end)
		currentlyLive := make(map[int]bool)
		for vrID := range liveness.LiveOut[block.ID] {
			currentlyLive[vrID] = true
			ig.AddNode(vrID)
		}

		// Process machine instructions in reverse order (backward through the block)
		for i := len(block.MachineInstructions) - 1; i >= 0; i-- {
			instr := block.MachineInstructions[i]

			// Get result (defined VR) and operands (used VRs)
			result := instr.GetResult()
			operands := instr.GetOperands()

			// The defined VR interferes with all currently live VRs
			if result != nil && shouldTrackForLiveness(result) {
				ig.AddNode(result.ID)
				for liveVRID := range currentlyLive {
					ig.AddEdge(result.ID, liveVRID)
				}
				// Remove the defined VR from live set (no longer live before definition)
				delete(currentlyLive, result.ID)
			}

			// Add all used VRs to the currently live set
			for _, operand := range operands {
				if operand != nil && shouldTrackForLiveness(operand) {
					currentlyLive[operand.ID] = true
					ig.AddNode(operand.ID)
				}
			}
		}

		// At the start of the block, all live VRs interfere with each other
		liveVRs := []int{}
		for vrID := range currentlyLive {
			liveVRs = append(liveVRs, vrID)
		}
		for i := 0; i < len(liveVRs); i++ {
			for j := i + 1; j < len(liveVRs); j++ {
				ig.AddEdge(liveVRs[i], liveVRs[j])
			}
		}
	}

	return ig
}
