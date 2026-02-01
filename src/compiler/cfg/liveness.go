package cfg

// LivenessInfo contains liveness analysis results for a CFG
// Works with VirtualRegister IDs instead of variable names
type LivenessInfo struct {
	// Live-in sets: VirtualRegisters live at block entry (block ID -> set of VR IDs)
	LiveIn map[int]map[int]bool

	// Live-out sets: VirtualRegisters live at block exit (block ID -> set of VR IDs)
	LiveOut map[int]map[int]bool

	// Use sets: VirtualRegisters used before being defined in block
	Use map[int]map[int]bool

	// Def sets: VirtualRegisters defined in block
	Def map[int]map[int]bool
}

// NewLivenessInfo creates a new liveness analysis result
func NewLivenessInfo() *LivenessInfo {
	return &LivenessInfo{
		LiveIn:  make(map[int]map[int]bool),
		LiveOut: make(map[int]map[int]bool),
		Use:     make(map[int]map[int]bool),
		Def:     make(map[int]map[int]bool),
	}
}

// ComputeLiveness performs liveness analysis on a CFG with MachineInstructions
func ComputeLiveness(cfg *CFG) *LivenessInfo {
	info := NewLivenessInfo()

	// Step 1: Compute use and def sets for each block from machine instructions
	for _, block := range cfg.Blocks {
		info.Use[block.ID] = make(map[int]bool)
		info.Def[block.ID] = make(map[int]bool)
		info.LiveIn[block.ID] = make(map[int]bool)
		info.LiveOut[block.ID] = make(map[int]bool)

		computeUseDefSetsFromMachineInstructions(block, info.Use[block.ID], info.Def[block.ID])
	}

	// Step 2: Iterate until live-in/live-out sets converge
	changed := true
	for changed {
		changed = false

		// Process blocks in reverse order (better convergence)
		for i := len(cfg.Blocks) - 1; i >= 0; i-- {
			block := cfg.Blocks[i]

			// Compute live-out: union of live-in of all successors
			newLiveOut := make(map[int]bool)
			for _, succ := range block.Successors {
				for vrID := range info.LiveIn[succ.ID] {
					newLiveOut[vrID] = true
				}
			}

			// Compute live-in: use ∪ (live-out - def)
			newLiveIn := make(map[int]bool)
			for vrID := range info.Use[block.ID] {
				newLiveIn[vrID] = true
			}
			for vrID := range newLiveOut {
				if !info.Def[block.ID][vrID] {
					newLiveIn[vrID] = true
				}
			}

			// Check if sets changed
			if !setsEqualInt(info.LiveIn[block.ID], newLiveIn) ||
				!setsEqualInt(info.LiveOut[block.ID], newLiveOut) {
				changed = true
				info.LiveIn[block.ID] = newLiveIn
				info.LiveOut[block.ID] = newLiveOut
			}
		}
	}

	return info
}

// computeUseDefSets analyzes a basic block to find used and defined variables
// computeUseDefSetsFromMachineInstructions analyzes machine instructions to find used and defined VirtualRegisters
func computeUseDefSetsFromMachineInstructions(block *BasicBlock, use, def map[int]bool) {
	for _, instr := range block.MachineInstructions {
		// Get VirtualRegisters used by this instruction (operands)
		for _, operand := range instr.GetOperands() {
			if operand != nil && shouldTrackForLiveness(operand) {
				vrID := operand.ID
				// Only add to use if not already defined in this block
				if !def[vrID] {
					use[vrID] = true
				}
			}
		}

		// Get VirtualRegister defined by this instruction (result)
		result := instr.GetResult()
		if result != nil && shouldTrackForLiveness(result) {
			def[result.ID] = true
		}
	}
}

// shouldTrackForLiveness returns true if this VirtualRegister needs liveness tracking
// ImmediateValues and StackLocations don't need physical register allocation
func shouldTrackForLiveness(vr *VirtualRegister) bool {
	return vr.Type == CandidateRegister || vr.Type == AllocatedRegister
}

// setsEqualInt checks if two int sets are equal
func setsEqualInt(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for key := range a {
		if !b[key] {
			return false
		}
	}
	return true
}

// GetLiveRanges computes live ranges for each VirtualRegister
// Returns map of VR ID to list of block IDs where it's live
func (info *LivenessInfo) GetLiveRanges() map[int][]int {
	ranges := make(map[int][]int)

	// Collect all blocks where each VR is live
	for blockID, liveVRs := range info.LiveIn {
		for vrID := range liveVRs {
			ranges[vrID] = append(ranges[vrID], blockID)
		}
	}

	for blockID, liveVRs := range info.LiveOut {
		for vrID := range liveVRs {
			// Add if not already present
			found := false
			for _, id := range ranges[vrID] {
				if id == blockID {
					found = true
					break
				}
			}
			if !found {
				ranges[vrID] = append(ranges[vrID], blockID)
			}
		}
	}

	return ranges
}

// IsLiveAt checks if a VirtualRegister is live at the entry of a block
func (info *LivenessInfo) IsLiveAt(vrID int, blockID int) bool {
	return info.LiveIn[blockID][vrID]
}

// IsLiveOutOf checks if a VirtualRegister is live at the exit of a block
func (info *LivenessInfo) IsLiveOutOf(vrID int, blockID int) bool {
	return info.LiveOut[blockID][vrID]
}
