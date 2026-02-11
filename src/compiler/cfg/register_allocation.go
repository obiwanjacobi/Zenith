package cfg

import (
	"fmt"
)

// InstructionFactory handles generating move/spill/reload instructions
// during register allocation resolution
type InstructionFactory interface {
	// SelectMove generates a move instruction from source VR to target VR
	// Returns the generated instruction(s) to be inserted before the current instruction
	CreateMove(target *VirtualRegister, source *VirtualRegister) ([]MachineInstruction, error)

	// SelectSpill generates instructions to spill a VR to stack
	// Returns the generated instruction(s) to be inserted
	CreateSpill(vr *VirtualRegister, stackOffset int8) ([]MachineInstruction, error)

	// SelectReload generates instructions to reload a VR from stack
	// Returns the generated instruction(s) to be inserted before the current instruction
	CreateReload(vr *VirtualRegister, stackOffset int8) ([]MachineInstruction, error)
}

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

// ResolveUnallocated resolves unallocated operand VRs by inserting register moves
// Takes the CFG, liveness info, interference graph, and InstructionManipulator
// Unallocated operands need to be moved into allocated registers before use
//
// Strategy:
// 1. For each unallocated operand, try to find a free register (not live at this point)
// 2. If found, swap the operand to use that register (insert move before instruction)
// 3. If no free register, spill to stack and reload before use
//
// Note: Current implementation handles single-use operands (values consumed by instruction).
// For operands that need to persist after the instruction (multi-use), additional logic
// would be needed to restore the original register or track the new location.
func ResolveUnallocated(cfg *CFG, li *LivenessInfo, ig *InterferenceGraph, factory InstructionFactory, availableRegs []*Register) error {
	for _, block := range cfg.Blocks {
		// Build new instruction list with inserted moves/spills
		newInstructions := make([]MachineInstruction, 0, len(block.MachineInstructions)*2)

		for _, instr := range block.MachineInstructions {
			operands := instr.GetOperands()
			needsReplacement := false
			newOperands := make([]*VirtualRegister, len(operands))

			// Check each operand
			for opIdx, operand := range operands {
				if operand == nil {
					newOperands[opIdx] = operand
					continue
				}

				// Skip already allocated operands and immediates
				if operand.Type == AllocatedRegister || operand.Type == ImmediateValue {
					newOperands[opIdx] = operand
					continue
				}

				// Unallocated candidate register - needs resolution
				if operand.Type == CandidateRegister {
					// Try register swap first
					freeReg := findFreeRegister(block, li, availableRegs, operand.Size)
					if freeReg != nil {
						// TODO: Call VR allocator?
						// Allocate new VR with the free register
						replacementVR := &VirtualRegister{
							ID:          operand.ID, // Keep same ID for tracking
							Size:        operand.Size,
							Type:        AllocatedRegister,
							PhysicalReg: freeReg,
							Name:        operand.Name,
						}

						// Generate move from original to replacement
						// These instructions will be inserted BEFORE the current instruction
						moveInstrs, err := factory.CreateMove(replacementVR, operand)
						if err != nil {
							return fmt.Errorf("failed to generate swap move: %w", err)
						}

						// Insert move instruction(s) before current instruction
						newInstructions = append(newInstructions, moveInstrs...)

						// Mark original operand as unused since we've swapped it
						operand.Unused()

						newOperands[opIdx] = replacementVR
						needsReplacement = true
					} else {
						// No free register - spill to stack
						stackOffset := cfg.StackOffset
						cfg.StackOffset += int8(operand.Size / 8) // Convert bits to bytes

						// First, generate spill instruction to store current value to stack
						spillInstrs, err := factory.CreateSpill(operand, stackOffset)
						if err != nil {
							return fmt.Errorf("failed to generate spill: %w", err)
						}

						// Insert spill instruction(s) before current instruction
						newInstructions = append(newInstructions, spillInstrs...)

						// Mark original as stack location
						operand.Type = StackLocation
						operand.Value = uint32(stackOffset)

						// Create a new VR for the reloaded value
						// The reload instructions will load from stack into this register
						reloadedVR := &VirtualRegister{
							ID:   operand.ID, // Keep same ID for tracking
							Size: operand.Size,
							Type: CandidateRegister, // Needs register allocation
							Name: operand.Name,
						}

						// Generate reload from stack into temporary register
						// These instructions load from stack and will be inserted BEFORE the current instruction
						reloadInstrs, err := factory.CreateReload(reloadedVR, stackOffset)
						if err != nil {
							return fmt.Errorf("failed to generate reload: %w", err)
						}

						// Insert reload instruction(s) before current instruction
						newInstructions = append(newInstructions, reloadInstrs...)

						newOperands[opIdx] = reloadedVR
						needsReplacement = true
					}
				}
			}

			if !needsReplacement {
				newInstructions = append(newInstructions, instr)
			}
		}

		// Replace block instructions with updated list
		block.MachineInstructions = newInstructions
	}

	return nil
}

// findFreeRegister finds a register that's not live at the given instruction point
func findFreeRegister(block *BasicBlock, li *LivenessInfo, availableRegs []*Register, size RegisterSize) *Register {
	// Get liveness at this point
	liveVRs := li.LiveIn[block.ID]
	if liveVRs == nil {
		liveVRs = make(map[int]bool)
	}

	// Collect all physical registers currently in use by live VRs
	usedRegs := make(map[*Register]bool)

	// Scan all instructions in the block to find VRs and their physical register assignments
	for _, instr := range block.MachineInstructions {
		// Check result register
		if result := instr.GetResult(); result != nil && liveVRs[result.ID] {
			if result.Type == AllocatedRegister && result.PhysicalReg != nil {
				markRegisterAsUsed(result.PhysicalReg, usedRegs, availableRegs)
			}
		}

		// Check operand registers
		for _, operand := range instr.GetOperands() {
			if operand != nil && liveVRs[operand.ID] {
				if operand.Type == AllocatedRegister && operand.PhysicalReg != nil {
					markRegisterAsUsed(operand.PhysicalReg, usedRegs, availableRegs)
				}
			}
		}
	}

	// Find a register of matching size that's not used
	for _, reg := range availableRegs {
		if reg.Size != int(size) {
			continue
		}

		if !usedRegs[reg] {
			return reg
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

	// Phase 1: Simplification
	for len(remaining) > 0 {
		found := false

		// Try to remove an operand with low degree first
		for vrID := range remaining {
			if !resultVRs[vrID] {
				vr := candidateVRs[vrID]
				degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
				numColors := ra.countAvailableColors(vr)
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
				vr := candidateVRs[vrID]
				degree := ra.getDegreeInSubgraph(vrID, ig, remaining)
				numColors := ra.countAvailableColors(vr)
				if degree < numColors {
					stack = append(stack, vrID)
					delete(remaining, vrID)
					found = true
					break
				}
			}
		}

		// If still no low-degree node found, pick an arbitrary one (potential spill)
		// Prefer operands over results for spilling
		if !found {
			// Try to pick an operand first
			for vrID := range remaining {
				if !resultVRs[vrID] {
					stack = append(stack, vrID)
					delete(remaining, vrID)
					found = true
					break
				}
			}
		}

		// If only results remain, pick one
		if !found {
			for vrID := range remaining {
				stack = append(stack, vrID)
				delete(remaining, vrID)
				found = true
				break
			}
		}
	}

	return stack
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

// countAvailableColors returns the number of physical registers available for a VR
// considering its size constraints and AllowedSet
func (ra *RegisterAllocator) countAvailableColors(vr *VirtualRegister) int {
	var candidates []*Register
	if len(vr.AllowedSet) > 0 {
		candidates = vr.AllowedSet
	} else {
		candidates = ra.availableRegisters
	}

	count := 0
	for _, reg := range candidates {
		if reg.Size == int(vr.Size) {
			count++
		}
	}
	return count
}

// markRegisterAsUsed marks a register and all overlapping registers as used
// If reg is a composite (pair), marks all components as used
// If reg is a component, marks all composites that contain it as used
func markRegisterAsUsed(reg *Register, usedRegs map[*Register]bool, availableRegs []*Register) {
	// Mark the register itself
	usedRegs[reg] = true

	// If this is a composite register (e.g., BC), mark its components (B, C)
	if len(reg.Composition) > 0 {
		for _, component := range reg.Composition {
			usedRegs[component] = true
		}
	}

	// If this is a component register, mark all composites that contain it
	// Search through all available registers to find composites
	for _, availReg := range availableRegs {
		if len(availReg.Composition) > 0 {
			// Check if this composite contains our register
			for _, component := range availReg.Composition {
				if component == reg {
					usedRegs[availReg] = true
					break
				}
			}
		}
	}
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
				markRegisterAsUsed(neighborVR.PhysicalReg, usedRegs, ra.availableRegisters)
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

func MarkUnusedVirtualRegisters(allVRs []*VirtualRegister, instructions []MachineInstruction) {
	var usedVRs = make(map[int]bool)

	// build a list af actually used VRs
	for _, instr := range instructions {
		// Check result register
		if result := instr.GetResult(); result != nil {
			usedVRs[result.ID] = true
		}

		// Check operand registers
		for _, operand := range instr.GetOperands() {
			usedVRs[operand.ID] = true
		}
	}

	// check all VRs against used list and mark unused ones
	for _, vr := range allVRs {
		if !usedVRs[vr.ID] {
			vr.Unused()
		}
	}
}
