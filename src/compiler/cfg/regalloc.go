package cfg

import (
	"fmt"
	"sort"
)

// ============================================================================
// Register Allocation — constraint propagation + linear scan (target-independent)
// ============================================================================

// AllocateRegisters allocates physical registers for every TempVR in the
// function's machine instructions using constraint propagation followed by a
// single-pass linear scan. On success every TempVR is replaced in-place with
// a PhysVR (or a StackVR for spilled values).
func AllocateRegisters(fnCFG *CFG) error {
	propagateConstraints(fnCFG)
	ranges := ComputeLiveRanges(fnCFG)

	assigned := make(map[int]*PhysVR)
	spilled := make(map[int]*StackVR)

	linearScan(fnCFG, ranges, assigned, spilled)
	substituteVRs(fnCFG, assigned, spilled)
	return nil
}

// ============================================================================
// Constraint propagation
// ============================================================================

// propagateConstraints walks every register-to-register copy and narrows the
// AllowedSet of an unconstrained destination to match a constrained source.
// This pre-colours result VRs through copy chains so the allocator treats them
// as pre-coloured without a separate analysis.
//
// Direction: constrained src → unconstrained dst only. Propagating the reverse
// (constrained dst → src) would force long-lived source VRs into a specific
// register even when other uses of that VR don't require it, causing spurious
// conflicts during linear scan.
func propagateConstraints(fnCFG *CFG) {
	for _, block := range fnCFG.Blocks {
		for _, mi := range block.MachineInstructions {
			dst, src, ok := mi.IsCopy()
			if !ok {
				continue
			}
			// Propagate only from a single-register source to an unconstrained dst,
			// and only when that register is actually in the dst's AllowedSet.
			// Without the membership check, a cross-class copy like LD {C,E}, L
			// would collapse {C,E} to {L}, turning the save into a self-move.
			if len(src.AllowedSet()) == 1 && len(dst.AllowedSet()) > 1 {
				srcReg := src.AllowedSet()[0]
				for _, r := range dst.AllowedSet() {
					if r == srcReg {
						dst.ConstrainTo(src.AllowedSet())
						break
					}
				}
			}
		}
	}
}

// ============================================================================
// Linear scan
// ============================================================================

type scanState struct {
	fnCFG    *CFG
	assigned map[int]*PhysVR
	spilled  map[int]*StackVR
	active   []LiveRange // sorted by End ascending
}

func linearScan(
	fnCFG *CFG,
	ranges []LiveRange,
	assigned map[int]*PhysVR,
	spilled map[int]*StackVR,
) {
	s := &scanState{fnCFG: fnCFG, assigned: assigned, spilled: spilled}

	for _, lr := range ranges {
		s.expireOld(lr.Start)

		if len(lr.VR.AllowedSet()) == 1 {
			// Pre-coloured: the instruction constrains this VR to exactly one register.
			reg := lr.VR.AllowedSet()[0]
			assigned[lr.VR.ID()] = &PhysVR{ID: lr.VR.ID(), Reg: reg}
			s.insertActive(lr)
			continue
		}

		reg := s.findFreeReg(lr.VR.AllowedSet())
		if reg != nil {
			assigned[lr.VR.ID()] = &PhysVR{ID: lr.VR.ID(), Reg: reg}
			s.insertActive(lr)
		} else {
			s.spillOne(lr)
		}
	}
}

// expireOld removes intervals from active whose End is strictly before pos.
func (s *scanState) expireOld(pos int) {
	keep := s.active[:0]
	for _, lr := range s.active {
		if lr.End >= pos {
			keep = append(keep, lr)
		}
	}
	s.active = keep
}

// insertActive adds lr to the active set, keeping it sorted by End.
func (s *scanState) insertActive(lr LiveRange) {
	s.active = append(s.active, lr)
	sort.Slice(s.active, func(i, j int) bool {
		return s.active[i].End < s.active[j].End
	})
}

// findFreeReg returns the first register in allowed that is not currently
// occupied by any active live range. Respects compound-register conflicts
// (e.g. HL occupies both H and L).
func (s *scanState) findFreeReg(allowed []*Register) *Register {
	occupied := s.occupiedRegisters()
	for _, candidate := range allowed {
		free := true
		for _, occ := range occupied {
			if registersConflict(candidate, occ) {
				free = false
				break
			}
		}
		if free {
			return candidate
		}
	}
	return nil
}

func (s *scanState) occupiedRegisters() []*Register {
	occ := make([]*Register, 0, len(s.active))
	for _, lr := range s.active {
		if phys, ok := s.assigned[lr.VR.ID()]; ok {
			occ = append(occ, phys.Reg)
		}
	}
	return occ
}

// spillOne spills either the longest-lived unconstrained active interval (if
// its End is further than lr's End, freeing its register for lr), or spills
// lr itself.
func (s *scanState) spillOne(lr LiveRange) {
	// Find the longest-lived active interval that is NOT pre-coloured.
	bestIdx := -1
	for i, a := range s.active {
		if len(a.VR.AllowedSet()) <= 1 {
			continue // pre-coloured; cannot evict
		}
		if bestIdx < 0 || s.active[i].End > s.active[bestIdx].End {
			bestIdx = i
		}
	}

	if bestIdx >= 0 && s.active[bestIdx].End > lr.End {
		// Evict the longer-lived interval and give its register to lr.
		victim := s.active[bestIdx]
		reg := s.assigned[victim.VR.ID()].Reg
		delete(s.assigned, victim.VR.ID())
		s.spillVR(victim.VR)
		s.active = append(s.active[:bestIdx], s.active[bestIdx+1:]...)
		s.assigned[lr.VR.ID()] = &PhysVR{ID: lr.VR.ID(), Reg: reg}
		s.insertActive(lr)
	} else {
		// Spill the incoming interval itself.
		s.spillVR(lr.VR)
	}
}

func (s *scanState) spillVR(vr *TempVR) {
	byteSize := uint16(vr.Size() / 8)
	offset := s.fnCFG.StackFrame.AddSpillSlot(byteSize)
	// A StackVR in a machine instruction operand signals to the emitter that
	// it must load or store via [IX+offset]. The emitter handles the actual
	// load sequence; the allocator's job is only to assign the stack slot.
	s.spilled[vr.ID()] = NewStackVR(vr.String(), offset, vr.Size())
}

// ============================================================================
// Register conflict detection
// ============================================================================

// registersConflict reports whether assigning register a would conflict with
// already-assigned register b. Conflicts arise from:
//   - Same register (a == b)
//   - a is a 16-bit pair whose composition contains b (e.g. HL conflicts with H or L)
//   - b is a 16-bit pair whose composition contains a
func registersConflict(a, b *Register) bool {
	if a == b {
		return true
	}
	for _, comp := range a.Composition {
		if comp == b {
			return true
		}
	}
	for _, comp := range b.Composition {
		if comp == a {
			return true
		}
	}
	return false
}

// ============================================================================
// VR substitution
// ============================================================================

// substituteVRs calls SubstituteVRs on every machine instruction in the
// function, delegating the target-specific field mutations to each instruction.
func substituteVRs(fnCFG *CFG, assigned map[int]*PhysVR, spilled map[int]*StackVR) {
	for _, block := range fnCFG.Blocks {
		for _, mi := range block.MachineInstructions {
			mi.SubstituteVRs(assigned, spilled)
		}
	}
}

// SubstituteOne replaces a single VROperand with its allocated PhysVR or
// StackVR. Exported so target-specific SubstituteVRs implementations can
// call it for each operand field.
func SubstituteOne(op VROperand, assigned map[int]*PhysVR, spilled map[int]*StackVR) VROperand {
	if op == nil {
		return nil
	}
	tvr, ok := op.(*TempVR)
	if !ok {
		return op // ImmVR, StackVR, PhysVR — pass through unchanged
	}
	if phys, found := assigned[tvr.ID()]; found {
		return phys
	}
	if sv, found := spilled[tvr.ID()]; found {
		return sv
	}
	panic(fmt.Sprintf("regalloc: TempVR t%d (%s) was neither assigned nor spilled", tvr.ID(), tvr.String()))
}
