package cfg

import (
	"fmt"

	"zenith/compiler/zir"
)

// RegisterClass represents the class/category of a register
type RegisterClass int

const (
	RegisterClassGeneral RegisterClass = iota
	RegisterClassAccumulator
	RegisterClassIndex
)

func (rc RegisterClass) String() string {
	switch rc {
	case RegisterClassGeneral:
		return "general"
	case RegisterClassAccumulator:
		return "accumulator"
	case RegisterClassIndex:
		return "index"
	default:
		return "unknown"
	}
}

// Register represents a physical register
type Register struct {
	Name  string
	Size  int // 8 or 16 bits
	Class RegisterClass
}

// AllocationResult contains the register allocation mapping
type AllocationResult struct {
	// Variable name to assigned register
	Allocation map[string]string

	// Variables that need to be spilled to memory
	Spilled map[string]bool

	// Variable usage patterns (from symbol table, used for preference-based allocation)
	VariableUsages map[string]zir.VariableUsage

	// Variable type sizes in bits (8 or 16) - needed to match register width
	VariableSizes map[string]int
}

// RegisterAllocator performs graph coloring register allocation
type RegisterAllocator struct {
	availableRegisters []Register
	numColors          int
}

// NewRegisterAllocator creates a new register allocator
func NewRegisterAllocator(registers []Register) *RegisterAllocator {
	return &RegisterAllocator{
		availableRegisters: registers,
		numColors:          len(registers),
	}
}

// Allocate performs graph coloring on the interference graph
func (ra *RegisterAllocator) Allocate(ig *InterferenceGraph) *AllocationResult {
	result := &AllocationResult{
		Allocation:     make(map[string]string),
		Spilled:        make(map[string]bool),
		VariableUsages: make(map[string]zir.VariableUsage),
		VariableSizes:  make(map[string]int),
	}

	// Get all nodes (variables)
	nodes := ig.GetNodes()
	if len(nodes) == 0 {
		return result
	}

	// Variables now use fully qualified names from Symbol.GetQualifiedName()
	// e.g., "main.x", "helper.count" - no collision between different scopes

	// TODO: Populate result.VariableSizes and result.VariableUsages from symbol table
	// For now, assume all variables are 8-bit (will need to pass symbol table or type info)
	for _, node := range nodes {
		result.VariableSizes[node] = 8 // Default to 8-bit
	}

	// Graph coloring algorithm with simplification
	stack := []string{}
	remaining := make(map[string]bool)
	for _, node := range nodes {
		remaining[node] = true
	}

	// Phase 1: Simplification
	// Remove nodes with degree < K and push onto stack
	for len(remaining) > 0 {
		found := false

		// Try to find a node with degree < K
		for node := range remaining {
			degree := ra.getDegreeInRemaining(ig, node, remaining)
			if degree < ra.numColors {
				// This node can definitely be colored
				stack = append(stack, node)
				delete(remaining, node)
				found = true
				break
			}
		}

		// If no node found with degree < K, pick one to potentially spill
		if !found {
			// Pick node with highest degree (heuristic: spill least used)
			var maxDegreeNode string
			maxDegree := -1

			for node := range remaining {
				degree := ra.getDegreeInRemaining(ig, node, remaining)
				if degree > maxDegree {
					maxDegree = degree
					maxDegreeNode = node
				}
			}

			if maxDegreeNode != "" {
				stack = append(stack, maxDegreeNode)
				delete(remaining, maxDegreeNode)
			} else {
				break
			}
		}
	}

	// Phase 2: Coloring (pop from stack and assign colors)
	for len(stack) > 0 {
		// Pop node from stack
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Find available colors (registers)
		usedColors := make(map[int]bool)

		// Check what colors neighbors have
		neighbors := ig.GetNeighbors(node)
		for _, neighbor := range neighbors {
			if reg, ok := result.Allocation[neighbor]; ok {
				// Find the color index of this register
				for i, r := range ra.availableRegisters {
					if r.Name == reg {
						usedColors[i] = true
						break
					}
				}
			}
		}

		// Assign first available color
		colorAssigned := false
		for i := 0; i < ra.numColors; i++ {
			if !usedColors[i] {
				result.Allocation[node] = ra.availableRegisters[i].Name
				colorAssigned = true
				break
			}
		}

		// If no color available, mark for spilling
		if !colorAssigned {
			result.Spilled[node] = true
		}
	}

	return result
}

// getDegreeInRemaining counts how many neighbors of a node are still in the remaining set
func (ra *RegisterAllocator) getDegreeInRemaining(ig *InterferenceGraph, node string, remaining map[string]bool) int {
	degree := 0
	neighbors := ig.GetNeighbors(node)
	for _, neighbor := range neighbors {
		if remaining[neighbor] {
			degree++
		}
	}
	return degree
}

// String returns a string representation of the allocation result
func (ar *AllocationResult) String() string {
	result := "Register Allocation:\n"
	for variable, register := range ar.Allocation {
		result += fmt.Sprintf("  %s -> %s\n", variable, register)
	}
	if len(ar.Spilled) > 0 {
		result += "Spilled variables:\n"
		for variable := range ar.Spilled {
			result += fmt.Sprintf("  %s (needs memory)\n", variable)
		}
	}
	return result
}
