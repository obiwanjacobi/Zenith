package z80

import "zenith/compiler/cfg"

// ============================================================================
// Peephole optimisation
// ============================================================================

// RunPeephole removes locally redundant moves from all blocks in the function.
// Must run after register allocation so operands are PhysVRs.
//
// Patterns removed (per block, one forward pass):
//   - LD r, r        — self-move: result and source resolved to the same physical register
//   - LD r1,r2 ; LD r2,r1 — back-and-forth pair that cancel each other out
func RunPeephole(fnCFG *cfg.CFG) {
	for _, block := range fnCFG.Blocks {
		block.MachineInstructions = peepholeBlock(block.MachineInstructions)
	}
}

func peepholeBlock(instrs []cfg.MachineInstruction) []cfg.MachineInstruction {
	out := make([]cfg.MachineInstruction, 0, len(instrs))
	for i := 0; i < len(instrs); i++ {
		m, ok := instrs[i].(*MachineInstrZ80)
		if !ok || m.Opcode != Z80_LD_R_R {
			out = append(out, instrs[i])
			continue
		}
		dst := physReg(m.Result)
		src := physReg(m.Src1)
		if dst == nil || src == nil {
			out = append(out, instrs[i])
			continue
		}

		// Pattern: LD r, r (self-move — result of two VRs allocated to the same register)
		if dst == src {
			continue
		}

		// Pattern: LD r1, r2 ; LD r2, r1 (pair that cancels out)
		if i+1 < len(instrs) {
			next, ok2 := instrs[i+1].(*MachineInstrZ80)
			if ok2 && next.Opcode == Z80_LD_R_R {
				nextDst := physReg(next.Result)
				nextSrc := physReg(next.Src1)
				if nextDst == src && nextSrc == dst {
					i++ // consume the second instruction too
					continue
				}
			}
		}
		out = append(out, instrs[i])
	}
	return out
}

// physReg extracts the *Register from a PhysVR operand, returning nil for any
// other operand kind (ImmVR, StackVR, TempVR, nil).
func physReg(op cfg.VROperand) *cfg.Register {
	if op == nil {
		return nil
	}
	phys, ok := op.(*cfg.PhysVR)
	if !ok {
		return nil
	}
	return phys.Reg
}
