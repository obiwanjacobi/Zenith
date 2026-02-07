package cfg

import "fmt"

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
}

// NewRegisterAllocator creates a new register allocator
func NewRegisterAllocator(registers []*Register) *RegisterAllocator {
	return &RegisterAllocator{
		availableRegisters: registers,
	}
}

// Allocate performs graph coloring register allocation on a CFG
// Assigns physical registers to VirtualRegisters based on interference graph
// Prioritizes result registers over operands for better allocation success
func (ra *RegisterAllocator) Allocate(cfg *CFG, ig *InterferenceGraph) error {
	// Gather all VRs from CFG, separating candidates from others
	// Also identify which are results vs operands in one pass
	candidateVRs := make(map[int]*VirtualRegister)
	resultVRs := make(map[int]bool)
	seen := make(map[int]bool)

	for _, block := range cfg.Blocks {
		for _, instr := range block.MachineInstructions {
			// Check result
			if result := instr.GetResult(); result != nil && !seen[result.ID] {
				seen[result.ID] = true
				if result.Type == CandidateRegister {
					candidateVRs[result.ID] = result
					resultVRs[result.ID] = true
				}
			}

			// Check operands
			for _, op := range instr.GetOperands() {
				if op != nil && !seen[op.ID] {
					seen[op.ID] = true
					if op.Type == CandidateRegister {
						candidateVRs[op.ID] = op
						// resultVRs[op.ID] remains false (default)
					}
				}
			}
		}
	}

	if len(candidateVRs) == 0 {
		return nil // Nothing to allocate
	}

	// Build simplification stack using graph coloring
	// Priority: process operands first (pushed early) so results get allocated first (popped late)
	stack := ra.buildSimplificationStack(candidateVRs, resultVRs, ig)

	// Phase 2: Selection - assign registers in reverse order
	for i := len(stack) - 1; i >= 0; i-- {
		vrID := stack[i]
		vr := candidateVRs[vrID]

		// Find an available register
		reg := ra.selectRegister(vr, ig, candidateVRs)
		if reg != nil {
			vr.PhysicalReg = reg
			vr.Type = AllocatedRegister
		}
		// If no register available, VR remains as CandidateRegister (unallocated)
	}

	// Final check: ensure all result VRs have been allocated
	for vrID, isResult := range resultVRs {
		if isResult {
			vr := candidateVRs[vrID]
			if vr.Type != AllocatedRegister || vr.PhysicalReg == nil {
				return fmt.Errorf("failed to allocate result VR %d (%s)", vr.ID, vr.Name)
			}
		}
	}

	return nil
}

// buildSimplificationStack creates the stack for graph coloring
// Returns a stack where operands are pushed before results (so results pop first)
func (ra *RegisterAllocator) buildSimplificationStack(
	candidateVRs map[int]*VirtualRegister,
	resultVRs map[int]bool,
	ig *InterferenceGraph,
) []int {
	stack := make([]int, 0, len(candidateVRs))
	remaining := make(map[int]bool, len(candidateVRs))
	for vrID := range candidateVRs {
		remaining[vrID] = true
	}

	numColors := len(ra.availableRegisters)

	// Phase 1: Simplification
	for len(remaining) > 0 {
		found := false

		// Try to remove an operand with low degree first
		for vrID := range remaining {
			if !resultVRs[vrID] {
				degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
				if degree < numColors {
					stack = append(stack, vrID)
					delete(remaining, vrID)
					found = true
					break
				}
			}
		}

		// If no low-degree operand, try low-degree result
		if !found {
			for vrID := range remaining {
				degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
				if degree < numColors {
					stack = append(stack, vrID)
					delete(remaining, vrID)
					found = true
					break
				}
			}
		}

		// If still nothing, pick a spill candidate (prefer operand over result)
		if !found {
			spillCandidate := ra.selectSpillCandidate(remaining, resultVRs, ig)
			stack = append(stack, spillCandidate)
			delete(remaining, spillCandidate)
		}
	}

	return stack
}

func findUnallocatedVRs(cfg *CFG) []*VirtualRegister {
	seen := make(map[*VirtualRegister]bool)
	unallocated := []*VirtualRegister{}

	for _, block := range cfg.Blocks {
		for _, instr := range block.MachineInstructions {
			// Results are not considered because they are prioritized for allocation and won't be unallocated if allocation succeeded
			// and cannot be easily changed for instructions.

			vrs := instr.GetOperands()

			for _, vr := range vrs {
				if vr == nil || seen[vr] {
					continue
				}
				seen[vr] = true

				if vr.PhysicalReg == nil && vr.Type != ImmediateValue {
					unallocated = append(unallocated, vr)
				}
			}
		}
	}

	return unallocated
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

// selectSpillCandidate chooses a VR to potentially spill
// Prefers operands over results, and higher degree nodes
func (ra *RegisterAllocator) selectSpillCandidate(remaining map[int]bool, resultVRs map[int]bool, ig *InterferenceGraph) int {
	maxDegree := -1
	var spillCandidate int
	isResultCandidate := true

	// First pass: try to find an operand with high degree
	for vrID := range remaining {
		if resultVRs[vrID] {
			continue // Skip results
		}
		degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
		if degree > maxDegree {
			maxDegree = degree
			spillCandidate = vrID
			isResultCandidate = false
		}
	}

	// If no operands found, fall back to results
	if isResultCandidate {
		maxDegree = -1
		for vrID := range remaining {
			degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
			if degree > maxDegree {
				maxDegree = degree
				spillCandidate = vrID
			}
		}
	}

	return spillCandidate
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

	return nil // No register available
}

// Spill marks all unallocated VRs (still CandidateRegister) as StackLocation
func (ra *RegisterAllocator) Spill(allVRs []*VirtualRegister) int {
	spillCount := 0
	for _, vr := range allVRs {
		if vr.Type == CandidateRegister {
			// VR couldn't be allocated - spill to stack
			vr.Type = StackLocation
			vr.Value = uint32(spillCount) // Stack offset
			spillCount++
		}
	}
	return spillCount
}
