package cfg

import (
	"fmt"
	"sort"
	"strings"
)

// InterferenceGraph represents which VirtualRegisters cannot share the same physical register
// because they are live at the same time
type InterferenceGraph struct {
	// Adjacency list representation: VR ID -> set of interfering VR IDs
	edges map[int]map[int]bool
	// All VirtualRegister IDs in the graph
	nodes map[int]bool
	// Instruction-level liveness: map[blockID][instrIdx] -> set of live VR IDs
	// Computed during BuildInterferenceGraph and reused by ResolveUnallocated
	InstructionLiveness map[int][]map[int]bool
}

// NewInterferenceGraph creates a new empty interference graph
func NewInterferenceGraph() *InterferenceGraph {
	return &InterferenceGraph{
		edges:               make(map[int]map[int]bool),
		nodes:               make(map[int]bool),
		InstructionLiveness: make(map[int][]map[int]bool),
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

// AddEdgeWithVRs adds an interference edge between two VirtualRegisters
// Takes the actual VR objects to check for composition compatibility
// If one VR can use a component of the other's register, they don't truly interfere
func (ig *InterferenceGraph) AddEdgeWithVRs(vr1, vr2 *VirtualRegister) {
	// Don't add self-loops
	if vr1.ID == vr2.ID {
		return
	}

	// Check if these VRs are composition-compatible:
	// If vr1 needs a component of vr2's register (or vice versa), they don't conflict
	if ig.areCompositionCompatible(vr1, vr2) {
		return
	}

	// Ensure both nodes exist
	ig.AddNode(vr1.ID)
	ig.AddNode(vr2.ID)

	// Add undirected edge
	ig.edges[vr1.ID][vr2.ID] = true
	ig.edges[vr2.ID][vr1.ID] = true
}

// areCompositionCompatible checks if two VRs can coexist due to register composition
// Returns true if one VR's AllowedSet only contains components of the other's AllowedSet
func (ig *InterferenceGraph) areCompositionCompatible(vr1, vr2 *VirtualRegister) bool {
	// If either has no constraints, they can't be composition-compatible
	if len(vr1.AllowedSet) == 0 || len(vr2.AllowedSet) == 0 {
		return false
	}

	// Check if vr1's allowed registers are all components of vr2's allowed registers
	if ig.allComponentsOf(vr1.AllowedSet, vr2.AllowedSet) {
		return true
	}

	// Check the reverse
	if ig.allComponentsOf(vr2.AllowedSet, vr1.AllowedSet) {
		return true
	}

	return false
}

// allComponentsOf checks if all registers in 'components' are components of any register in 'composites'
func (ig *InterferenceGraph) allComponentsOf(components, composites []*Register) bool {
	for _, component := range components {
		foundAsComponent := false

		for _, composite := range composites {
			// Check if component is part of this composite's composition
			if len(composite.Composition) > 0 {
				for _, part := range composite.Composition {
					if part == component {
						foundAsComponent = true
						break
					}
				}
			}
			if foundAsComponent {
				break
			}
		}

		// If this component isn't part of any composite, they're not compatible
		if !foundAsComponent {
			return false
		}
	}

	return len(components) > 0 // Must have at least one component
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

// BuildInterferenceGraph constructs an interference graph from liveness information
// Two VirtualRegisters interfere if they are both live at the same point in the program
// Uses instruction-level liveness for precision and considers register composition
func BuildInterferenceGraph(cfg *CFG, liveness *LivenessInfo) *InterferenceGraph {
	ig := NewInterferenceGraph()

	// Build a map of VR ID to VR object for composition checking
	vrMap := make(map[int]*VirtualRegister)
	for _, block := range cfg.Blocks {
		for _, instr := range block.MachineInstructions {
			if result := instr.GetResult(); result != nil {
				vrMap[result.ID] = result
			}
			for _, operand := range instr.GetOperands() {
				if operand != nil {
					vrMap[operand.ID] = operand
				}
			}
		}
	}

	// For each block, compute precise per-instruction liveness
	for _, block := range cfg.Blocks {
		// Prepare storage for this block's instruction liveness
		blockLiveness := make([]map[int]bool, len(block.MachineInstructions))

		// Start with live-out for this block (VRs live at the end)
		currentlyLive := make(map[int]bool)
		for vrID := range liveness.LiveOut[block.ID] {
			currentlyLive[vrID] = true
			ig.AddNode(vrID)
		}

		// Process machine instructions in reverse order (backward through the block)
		for i := len(block.MachineInstructions) - 1; i >= 0; i-- {
			instr := block.MachineInstructions[i]

			// Save liveness BEFORE this instruction executes (what operands see)
			liveAtInstr := make(map[int]bool)
			for vrID := range currentlyLive {
				liveAtInstr[vrID] = true
			}
			blockLiveness[i] = liveAtInstr

			// Get result (defined VR) and operands (used VRs)
			result := instr.GetResult()
			operands := instr.GetOperands()

			// The defined VR interferes with all VRs that are live AFTER this instruction
			// (i.e., those in currentlyLive before we remove the result)
			if result != nil && shouldTrackForLiveness(result) {
				ig.AddNode(result.ID)
				for liveVRID := range currentlyLive {
					// Use composition-aware interference checking
					if liveVR, exists := vrMap[liveVRID]; exists {
						ig.AddEdgeWithVRs(result, liveVR)
					} else {
						// Fallback if VR not in map
						ig.AddEdge(result.ID, liveVRID)
					}
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

		// Save this block's instruction liveness
		ig.InstructionLiveness[block.ID] = blockLiveness

		// For VRs that are live at block entry (live-in from predecessors),
		// they are simultaneously live and must interfere with each other.
		// IMPORTANT: Only add pairwise interference for VRs that are ALSO in LiveIn.
		// This excludes VRs that are only used locally within this block.
		liveAtEntry := liveness.LiveIn[block.ID]
		liveVRs := []int{}
		for vrID := range currentlyLive {
			// Only include if this VR is truly live-in (comes from predecessor)
			if liveAtEntry[vrID] {
				liveVRs = append(liveVRs, vrID)
			}
		}
		for i := 0; i < len(liveVRs); i++ {
			for j := i + 1; j < len(liveVRs); j++ {
				// Use composition-aware interference checking
				vr1, exists1 := vrMap[liveVRs[i]]
				vr2, exists2 := vrMap[liveVRs[j]]
				if exists1 && exists2 {
					ig.AddEdgeWithVRs(vr1, vr2)
				} else {
					ig.AddEdge(liveVRs[i], liveVRs[j])
				}
			}
		}
	}

	return ig
}

// String returns a string representation of the interference graph
func (ig *InterferenceGraph) String() string {
	var result strings.Builder
	result.WriteString("Interference Graph:\n")
	nodes := ig.GetNodes()
	for _, node := range nodes {
		neighbors := ig.GetNeighbors(node)
		fmt.Fprintf(&result, "  VR%d -> %v\n", node, neighbors)
	}
	return result.String()
}

func DumpInterference(fnName string, interference *InterferenceGraph) {
	fmt.Printf("========== Interference: %s ==========\n", fnName)
	nodes := interference.GetNodes()
	edgeCount := 0
	for _, node := range nodes {
		edgeCount += interference.GetDegree(node)
	}
	edgeCount /= 2 // Each edge counted twice
	for _, vrID := range nodes {
		neighbors := interference.GetNeighbors(vrID)
		if len(neighbors) > 0 {
			fmt.Printf("  VR%d interferes with VRs: %v\n", vrID, neighbors)
		}
	}
	fmt.Println()
}
