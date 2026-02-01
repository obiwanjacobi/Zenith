package cfg

import (
	"fmt"
)

// Register represents a physical register
type Register struct {
	Name        string
	Size        int         // 8 or 16 bits
	Composition []*Register // For multi-byte registers (typical Intel and Zilog)
	RegisterId  int         // the register id for encoding
}

// RegisterAllocator performs graph coloring register allocation on VirtualRegisters
type RegisterAllocator struct {
	availableRegisters []*Register
	numColors          int
}

// NewRegisterAllocator creates a new register allocator
func NewRegisterAllocator(registers []*Register) *RegisterAllocator {
	return &RegisterAllocator{
		availableRegisters: registers,
		numColors:          len(registers),
	}
}

// Allocate performs graph coloring register allocation on a CFG
// Assigns physical registers to VirtualRegisters based on interference graph
func (ra *RegisterAllocator) Allocate(cfg *CFG, ig *InterferenceGraph, allVRs []*VirtualRegister) error {
	// Filter to only CandidateRegisters that need allocation
	candidateVRs := make(map[int]*VirtualRegister)
	for _, vr := range allVRs {
		if vr.Type == CandidateRegister {
			candidateVRs[vr.ID] = vr
		}
	}

	// Graph coloring with simplification
	stack := []int{}
	remaining := make(map[int]bool)
	for vrID := range candidateVRs {
		remaining[vrID] = true
	}

	// Phase 1: Simplification - remove nodes with degree < k
	for len(remaining) > 0 {
		// Find a node with degree < numColors (in the remaining subgraph)
		found := false
		for vrID := range remaining {
			degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
			if degree < ra.numColors {
				stack = append(stack, vrID)
				delete(remaining, vrID)
				found = true
				break
			}
		}

		// If no such node found, pick the node with highest degree (potential spill)
		if !found {
			maxDegree := -1
			var spillCandidate int
			for vrID := range remaining {
				degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
				if degree > maxDegree {
					maxDegree = degree
					spillCandidate = vrID
				}
			}
			stack = append(stack, spillCandidate)
			delete(remaining, spillCandidate)
		}
	}

	// Phase 2: Selection - assign registers in reverse order
	for i := len(stack) - 1; i >= 0; i-- {
		vrID := stack[i]
		vr := candidateVRs[vrID]

		// Find an available register
		reg := ra.selectRegister(vr, ig, candidateVRs)
		if reg != nil {
			vr.PhysicalReg = reg
			vr.Type = AllocatedRegister
		} else {
			// Spill to stack - mark as StackLocation
			vr.Type = StackLocation
			// TODO: Assign stack offset
			return fmt.Errorf("register spilling not yet implemented for VR%d", vrID)
		}
	}

	return nil
}

// getDegreeInSubgraph counts how many neighbors of a VR are in the remaining subgraph
func (ra *RegisterAllocator) getDegreeInSubgraph(vrID int, ig *InterferenceGraph, remaining map[int]bool) int {
	degree := 0
	neighbors := ig.GetNeighbors(vrID)
	for _, neighborID := range neighbors {
		if remaining[neighborID] {
			degree++
		}
	}
	return degree
}

// selectRegister chooses the best physical register for a VirtualRegister
// considering size constraints, AllowedSet, and neighbor assignments
func (ra *RegisterAllocator) selectRegister(vr *VirtualRegister, ig *InterferenceGraph, allVRs map[int]*VirtualRegister) *Register {
	// Find which registers are already used by neighbors
	usedRegs := make(map[*Register]bool)
	neighbors := ig.GetNeighbors(vr.ID)
	for _, neighborID := range neighbors {
		if neighborVR, exists := allVRs[neighborID]; exists {
			if neighborVR.PhysicalReg != nil {
				usedRegs[neighborVR.PhysicalReg] = true
			}
		}
	}

	// Filter available registers by size and AllowedSet
	var candidates []*Register
	if len(vr.AllowedSet) > 0 {
		// VR has constraints - only consider AllowedSet
		candidates = vr.AllowedSet
	} else {
		// Consider all registers of matching size
		candidates = ra.availableRegisters
	}

	// Find first available register with matching size
	for _, reg := range candidates {
		if reg.Size == int(vr.Size) && !usedRegs[reg] {
			return reg
		}
	}

	return nil // No register available (needs spilling)
}

func DumpAllocation(vrAlloc *VirtualRegisterAllocator) {
	fmt.Println("========== REGISTER ALLOCATION ==========")

	// Collect VRs by type
	allocated := []*VirtualRegister{}
	spilled := []*VirtualRegister{}
	immediates := []*VirtualRegister{}
	candidates := []*VirtualRegister{}

	for _, vr := range vrAlloc.GetAll() {
		switch vr.Type {
		case AllocatedRegister:
			allocated = append(allocated, vr)
		case StackLocation:
			spilled = append(spilled, vr)
		case ImmediateValue:
			immediates = append(immediates, vr)
		case CandidateRegister:
			candidates = append(candidates, vr)
		}
	}

	if len(allocated) > 0 {
		fmt.Printf("Allocated (%d):\n", len(allocated))
		for _, vr := range allocated {
			fmt.Println(vr.String())
		}
	}

	if len(spilled) > 0 {
		fmt.Printf("\nSpilled to stack (%d):\n", len(spilled))
		for _, vr := range spilled {
			fmt.Println(vr.String())
		}
	}

	if len(immediates) > 0 {
		fmt.Printf("\nImmediates (%d):\n", len(immediates))
		for _, vr := range immediates {
			fmt.Println(vr.String())
		}
	}

	if len(candidates) > 0 {
		fmt.Printf("\nUnallocated candidates (%d):\n", len(candidates))
		for _, vr := range candidates {
			fmt.Println(vr.String())
		}
	}

	fmt.Printf("\nTotal: %d VRs (%d allocated, %d spilled, %d immediate, %d unallocated)\n\n",
		len(vrAlloc.GetAll()), len(allocated), len(spilled), len(immediates), len(candidates))
}
