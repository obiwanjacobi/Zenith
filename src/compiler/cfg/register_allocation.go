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
}

// NewRegisterAllocator creates a new register allocator
func NewRegisterAllocator(registers []*Register) *RegisterAllocator {
	return &RegisterAllocator{
		availableRegisters: registers,
	}
}

// AllocationStrategy defines the heuristic for building the simplification stack
type AllocationStrategy int

const (
	ConstrainedFirst AllocationStrategy = iota // Prioritize VRs with AllowedSet
	ResultFirst                                // Prioritize result VRs
	OperandFirst                               // Prioritize operand VRs (original strategy)
)

// Allocate performs graph coloring register allocation on a CFG
// Assigns physical registers to VirtualRegisters based on interference graph
// Uses iterative allocation with different strategies if initial allocation fails
// Returns true if a second pass (ResolveUnallocated) is needed for remaining unallocated VRs
func (ra *RegisterAllocator) Allocate(cfg *CFG, ig *InterferenceGraph) bool {
	// Gather all VRs from CFG, separating candidates from others
	// Also identify which are results vs operands, and which are constrained
	candidateVRs := make(map[int]*VirtualRegister)
	resultVRs := make(map[int]bool)
	constrainedVRs := make(map[int]bool)
	seen := make(map[int]bool)

	for _, block := range cfg.Blocks {
		for _, instr := range block.MachineInstructions {
			// Check result
			if result := instr.GetResult(); result != nil && !seen[result.ID] {
				seen[result.ID] = true
				if result.Type == CandidateRegister {
					candidateVRs[result.ID] = result
					resultVRs[result.ID] = true
					if len(result.AllowedSet) > 0 {
						constrainedVRs[result.ID] = true
					}
				}
			}

			// Check operands
			for _, op := range instr.GetOperands() {
				if op != nil && !seen[op.ID] {
					seen[op.ID] = true
					if op.Type == CandidateRegister {
						candidateVRs[op.ID] = op
						// resultVRs[op.ID] remains false (default)
						if len(op.AllowedSet) > 0 {
							constrainedVRs[op.ID] = true
						}
					}
				}
			}
		}
	}

	if len(candidateVRs) == 0 {
		return false // Nothing to allocate
	}

	// Try allocation with different strategies until one succeeds
	strategies := []AllocationStrategy{
		ConstrainedFirst, // Best for Z80: allocate A, HL first
		ResultFirst,      // Fallback: prioritize results
		OperandFirst,     // Last resort: original strategy
	}

	for _, strategy := range strategies {
		// Reset all candidates to unallocated state for retry
		if strategy != ConstrainedFirst {
			for _, vr := range candidateVRs {
				if vr.Type == AllocatedRegister {
					vr.Type = CandidateRegister
					vr.PhysicalReg = nil
				}
			}
		}

		// Build simplification stack using current strategy
		stack := ra.buildSimplificationStack(candidateVRs, resultVRs, constrainedVRs, ig, strategy)

		// Phase 2: Selection - assign registers in reverse order
		for i := len(stack) - 1; i >= 0; i-- {
			vrID := stack[i]
			vr := candidateVRs[vrID]

			// Find an available register
			reg := ra.selectRegister(vr, ig, candidateVRs)
			if reg != nil {
				vr.Assign(reg)
			}
			// If no register available, VR remains as CandidateRegister (unallocated)
		}

		// Check if all result VRs have been allocated
		allResultsAllocated := true
		for vrID, isResult := range resultVRs {
			if isResult {
				vr := candidateVRs[vrID]
				if vr.Type != AllocatedRegister || vr.PhysicalReg == nil {
					allResultsAllocated = false
					break
				}
			}
		}

		// Also check constrained operands - they're just as critical
		// If an operandhas AllowedSet, the instruction REQUIRES that specific register
		allConstrainedAllocated := true
		for vrID := range constrainedVRs {
			vr := candidateVRs[vrID]
			if vr.Type != AllocatedRegister || vr.PhysicalReg == nil {
				allConstrainedAllocated = false
				break
			}
		}

		if allResultsAllocated && allConstrainedAllocated {
			// Success! All critical VRs allocated
			// Check if any unconstrained operands remain unallocated
			for _, vr := range candidateVRs {
				if vr.Type == CandidateRegister {
					return true // Second pass needed
				}
			}
			return false // All VRs allocated
		}
	}

	// All strategies failed to allocate critical VRs
	// This shouldn't happen with the current iterative strategy approach
	// Return true to attempt second pass (ResolveUnallocated will handle the issue)
	return true
}

// ResolveUnallocated resolves unallocated operand VRs by direct allocation with move insertion
// This is the second-chance allocation pass that runs after the main Allocate pass
//
// Strategy:
// 1. For each instruction with unallocated operands:
//    a. Pick a register from the operand's AllowedSet (architectural constraint)
//    b. Allocate the operand VR directly to that register
//    c. Find where the value is currently located (as a result somewhere)
//    d. Insert move instructions before this instruction to get value into the required register
//
// This respects instruction constraints (AllowedSet) and doesn't modify instruction semantics.
// It "spills to another register" by inserting moves to satisfy architectural requirements.
func (ra *RegisterAllocator) ResolveUnallocated(cfg *CFG, ig *InterferenceGraph, selector InstructionSelector) error {
	// Use pre-computed instruction liveness from interference graph
	instrLiveness := ig.InstructionLiveness

	for blockID, block := range cfg.Blocks {
		newInstructions := make([]MachineInstruction, 0, len(block.MachineInstructions))

		for instrIdx, instr := range block.MachineInstructions {
			operands := instr.GetOperands()

			// Check each operand for unallocated VRs
			for _, operand := range operands {
				if operand == nil {
					continue
				}

				// Skip already allocated operands and immediates
				if operand.Type != CandidateRegister {
					continue
				}

				// Unallocated operand - needs resolution

				// Pick a register from AllowedSet - these are the ONLY valid registers for this operand
				// The instruction selector already determined these constraints
				liveAtInstr := instrLiveness[blockID][instrIdx]
				targetReg := ra.pickRegisterFromAllowedSetAtPoint(operand, liveAtInstr, cfg)
				if targetReg == nil {
					// No register from AllowedSet is available - must spill to stack
					operand.Type = StackLocation
					stackOffset := cfg.StackOffset
					cfg.StackOffset += uint16(operand.Size / 8)
					operand.Value = int32(stackOffset)

					// Insert reload from stack before instruction
					reloadInstrs, err := selector.CreateReload(operand, int8(stackOffset))
					if err != nil {
						return fmt.Errorf("failed to create reload for VR %d: %w", operand.ID, err)
					}
					newInstructions = append(newInstructions, reloadInstrs...)
					continue
				}

				// Allocate operand to the target register from AllowedSet
				operand.Assign(targetReg)

				// Now find where this value is currently located
				// If the operand VR was defined as a result elsewhere, find that definition
				sourceVR := ra.findValueSource(cfg, operand.ID)
				if sourceVR != nil && sourceVR.Type == AllocatedRegister {
					// Value is in another register - insert move
					moveInstrs, err := selector.CreateMove(operand, sourceVR)
					if err != nil {
						return fmt.Errorf("failed to create move for VR %d: %w", operand.ID, err)
					}
					newInstructions = append(newInstructions, moveInstrs...)
				} else if sourceVR != nil && sourceVR.Type == StackLocation {
					// Value is on stack - insert reload
					reloadInstrs, err := selector.CreateReload(operand, int8(sourceVR.Value))
					if err != nil {
						return fmt.Errorf("failed to create reload for VR %d: %w", operand.ID, err)
					}
					newInstructions = append(newInstructions, reloadInstrs...)
				}
				// If sourceVR is nil, the value might be an input parameter or undefined
				// The instruction selector's CreateMove should handle this case
			}

			// Append the original instruction (unchanged)
			newInstructions = append(newInstructions, instr)
		}

		block.MachineInstructions = newInstructions
	}

	return nil
}

// pickRegisterFromAllowedSetAtPoint selects a register from AllowedSet that's not used by live VRs
// Uses precise per-instruction liveness rather than block-level liveness
func (ra *RegisterAllocator) pickRegisterFromAllowedSetAtPoint(vr *VirtualRegister, liveVRs map[int]bool, cfg *CFG) *Register {
	if len(vr.AllowedSet) == 0 {
		// No constraints - fall back to any register of matching size
		for _, reg := range ra.availableRegisters {
			if reg.Size == int(vr.Size) {
				return reg
			}
		}
		return nil
	}

	// Build set of registers used by currently live VRs
	// Accounts for register composition (HL uses both H and L)
	usedRegs := make(map[*Register]bool)

	for _, block := range cfg.Blocks {
		for _, instr := range block.MachineInstructions {
			// Check result
			if result := instr.GetResult(); result != nil && liveVRs[result.ID] {
				if result.Type == AllocatedRegister && result.PhysicalReg != nil {
					markRegisterAsUsed(result.PhysicalReg, usedRegs, ra.availableRegisters)
				}
			}

			// Check operands
			for _, operand := range instr.GetOperands() {
				if operand != nil && liveVRs[operand.ID] {
					if operand.Type == AllocatedRegister && operand.PhysicalReg != nil {
						markRegisterAsUsed(operand.PhysicalReg, usedRegs, ra.availableRegisters)
					}
				}
			}
		}
	}

	// Pick first available register from AllowedSet
	for _, reg := range vr.AllowedSet {
		if reg.Size == int(vr.Size) && !usedRegs[reg] {
			return reg
		}
	}

	// All AllowedSet registers appear used - pick first one anyway and let moves handle it
	for _, reg := range vr.AllowedSet {
		if reg.Size == int(vr.Size) {
			return reg
		}
	}

	return nil
}

// findValueSource finds where a VR's value is defined (as a result)
// Returns the VR at the definition point (which may have been allocated)
func (ra *RegisterAllocator) findValueSource(cfg *CFG, vrID int) *VirtualRegister {
	for _, block := range cfg.Blocks {
		for _, instr := range block.MachineInstructions {
			if result := instr.GetResult(); result != nil && result.ID == vrID {
				return result
			}
		}
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
// Returns a stack ordered according to the specified strategy
// Stack is processed in reverse (last pushed = first allocated)
func (ra *RegisterAllocator) buildSimplificationStack(
	candidateVRs map[int]*VirtualRegister,
	resultVRs map[int]bool,
	constrainedVRs map[int]bool,
	ig *InterferenceGraph,
	strategy AllocationStrategy,
) []int {
	stack := make([]int, 0, len(candidateVRs))
	remaining := make(map[int]bool, len(candidateVRs))
	for vrID := range candidateVRs {
		remaining[vrID] = true
	}

	// Phase 1: Simplification with strategy-based prioritization
	for len(remaining) > 0 {
		found := false

		// Try to find a low-degree node matching our priority order
		priorityGroups := ra.getPriorityGroups(strategy, resultVRs, constrainedVRs)

		for _, checkPriority := range priorityGroups {
			for vrID := range remaining {
				if !ra.matchesPriority(vrID, checkPriority, resultVRs, constrainedVRs) {
					continue
				}

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
			if found {
				break
			}
		}

		// If no low-degree node found, pick an arbitrary one (potential spill)
		// Use same priority order
		if !found {
			for _, checkPriority := range priorityGroups {
				for vrID := range remaining {
					if ra.matchesPriority(vrID, checkPriority, resultVRs, constrainedVRs) {
						stack = append(stack, vrID)
						delete(remaining, vrID)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}

	return stack
}

// vrPriority defines the type of VR for prioritization
type vrPriority int

const (
	constrainedResult vrPriority = iota
	constrainedOperand
	unconstrainedResult
	unconstrainedOperand
)

// getPriorityGroups returns the order in which to try allocating VRs based on strategy
// Lower index = pushed earlier = allocated later (due to stack reversal)
func (ra *RegisterAllocator) getPriorityGroups(strategy AllocationStrategy, resultVRs, constrainedVRs map[int]bool) []vrPriority {
	switch strategy {
	case ConstrainedFirst:
		// Allocate constrained first (results > operands), then unconstrained
		return []vrPriority{unconstrainedOperand, unconstrainedResult, constrainedOperand, constrainedResult}
	case ResultFirst:
		// Allocate results first (constrained > unconstrained), then operands
		return []vrPriority{unconstrainedOperand, constrainedOperand, unconstrainedResult, constrainedResult}
	case OperandFirst:
		// Original strategy: allocate operands first, results last
		return []vrPriority{constrainedResult, unconstrainedResult, constrainedOperand, unconstrainedOperand}
	default:
		return []vrPriority{unconstrainedOperand, unconstrainedResult, constrainedOperand, constrainedResult}
	}
}

// matchesPriority checks if a VR matches the given priority category
func (ra *RegisterAllocator) matchesPriority(vrID int, priority vrPriority, resultVRs, constrainedVRs map[int]bool) bool {
	isResult := resultVRs[vrID]
	isConstrained := constrainedVRs[vrID]

	switch priority {
	case constrainedResult:
		return isResult && isConstrained
	case constrainedOperand:
		return !isResult && isConstrained
	case unconstrainedResult:
		return isResult && !isConstrained
	case unconstrainedOperand:
		return !isResult && !isConstrained
	default:
		return false
	}
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
			vr.Value = int32(spillCount) // Stack offset
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
