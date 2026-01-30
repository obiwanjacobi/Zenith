package cfg

import (
	"fmt"

	"zenith/compiler/zsm"
)

// RegisterClass represents the class/category of a register
type RegisterClass int

const (
	RegisterClassGeneral RegisterClass = iota
	RegisterClassAccumulator
	RegisterClassIndex
	RegisterClassFlags
	RegisterClassStackPointer
)

func (rc RegisterClass) String() string {
	switch rc {
	case RegisterClassGeneral:
		return "general"
	case RegisterClassAccumulator:
		return "accumulator"
	case RegisterClassIndex:
		return "index"
	case RegisterClassStackPointer:
		return "stack pointer"
	default:
		return "unknown"
	}
}

// SymbolInfo provides information needed for register allocation
type SymbolInfo interface {
	// GetTypeSize returns the size of the symbol's type in bits (8 or 16)
	GetTypeSize(qualifiedName string) int

	// GetUsage returns the usage pattern of the symbol
	GetUsage(qualifiedName string) zsm.VariableUsage
}

// Register represents a physical register
type Register struct {
	Name        string
	Size        int // 8 or 16 bits
	Class       RegisterClass
	Composition []*Register // For multi-byte registers (typical Intel and Zilog)
	RegisterId  int         // the register id for encoding
}

// AllocationResult contains the register allocation mapping
type AllocationResult struct {
	// Variable name to assigned register
	Allocation map[string]string

	// Variables that need to be spilled to memory
	Spilled map[string]bool

	// Variable usage patterns (from symbol table, used for preference-based allocation)
	VariableUsages map[string]zsm.VariableUsage

	// Variable type sizes in bits (8 or 16) - needed to match register width
	VariableSizes map[string]int
}

// RegisterAllocator performs graph coloring register allocation
type RegisterAllocator struct {
	availableRegisters []*Register
	numColors          int
	callingConvention  CallingConvention
	capabilities       RegisterCapabilities
}

// NewRegisterAllocator creates a new register allocator
func NewRegisterAllocator(registers []*Register) *RegisterAllocator {
	return &RegisterAllocator{
		availableRegisters: registers,
		numColors:          len(registers),
		callingConvention:  nil, // Optional, set via SetCallingConvention
		capabilities:       nil, // Optional, set via SetCapabilities
	}
}

// SetCallingConvention sets the calling convention for this allocator
func (ra *RegisterAllocator) SetCallingConvention(cc CallingConvention) {
	ra.callingConvention = cc
}

// SetCapabilities sets the register capabilities for architecture-specific scoring
func (ra *RegisterAllocator) SetCapabilities(cap RegisterCapabilities) {
	ra.capabilities = cap
}

// Allocate performs graph coloring on the interference graph
// symbolInfo provides type sizes and usage patterns for variables
func (ra *RegisterAllocator) Allocate(ig *InterferenceGraph, symbolInfo SymbolInfo) *AllocationResult {
	return ra.AllocateWithPrecoloring(ig, symbolInfo, nil)
}

// AllocateWithPrecoloring performs register allocation with pre-colored variables
// precolored maps variable names to their required registers (e.g., function parameters)
func (ra *RegisterAllocator) AllocateWithPrecoloring(ig *InterferenceGraph, symbolInfo SymbolInfo, precolored map[string]string) *AllocationResult {
	result := &AllocationResult{
		Allocation:     make(map[string]string),
		Spilled:        make(map[string]bool),
		VariableUsages: make(map[string]zsm.VariableUsage),
		VariableSizes:  make(map[string]int),
	}

	// Get all nodes (variables)
	nodes := ig.GetNodes()
	if len(nodes) == 0 {
		return result
	}

	// Populate variable sizes and usage patterns from symbol info
	if symbolInfo != nil {
		for _, node := range nodes {
			result.VariableSizes[node] = symbolInfo.GetTypeSize(node)
			result.VariableUsages[node] = symbolInfo.GetUsage(node)
		}
	} else {
		// Fallback: assume all variables are 8-bit
		for _, node := range nodes {
			result.VariableSizes[node] = 8
		}
	}

	// Apply pre-coloring (e.g., function parameters)
	for varName, regName := range precolored {
		result.Allocation[varName] = regName
	}

	// Graph coloring algorithm with simplification
	stack := []string{}
	remaining := make(map[string]bool)
	for _, node := range nodes {
		// Skip pre-colored nodes - they're already allocated
		if precolored != nil && precolored[node] != "" {
			continue
		}
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

		// Select best register based on variable usage and size
		// Use preference-based selection if usage info is available
		var colorIdx int
		if result.VariableUsages[node] != 0 && result.VariableSizes[node] != 0 {
			// Use preference-based selection
			colorIdx = selectBestRegister(
				node,
				result.VariableUsages[node],
				result.VariableSizes[node],
				ra.availableRegisters,
				usedColors,
				ra.capabilities,
			)
		} else {
			// Fallback: pick first available
			colorIdx = -1
			for i := 0; i < ra.numColors; i++ {
				if !usedColors[i] {
					colorIdx = i
					break
				}
			}
		}

		// Assign the selected register
		if colorIdx >= 0 {
			result.Allocation[node] = ra.availableRegisters[colorIdx].Name
		} else {
			// No color available, mark for spilling
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

// BuildParameterPrecoloring creates a pre-coloring map for function parameters
// based on the calling convention. Returns map of qualified param name -> register name
func (ra *RegisterAllocator) BuildParameterPrecoloring(functionName string, paramNames []string, paramSizes []int) map[string]string {
	if ra.callingConvention == nil {
		return nil
	}

	precolored := make(map[string]string)
	for i, paramName := range paramNames {
		paramSize := 8 // default
		if i < len(paramSizes) {
			paramSize = paramSizes[i]
		}

		reg, _, useStack := ra.callingConvention.GetParameterLocation(i, paramSize)
		if !useStack && reg != nil {
			// Qualify the parameter name with function scope
			qualifiedName := functionName + "." + paramName
			precolored[qualifiedName] = reg.Name
		}
		// Stack parameters are not pre-colored (they're already in memory)
	}

	return precolored
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
