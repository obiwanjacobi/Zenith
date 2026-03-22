package z80

import (
	"fmt"

	"zenith/compiler/cfg"
)

// ============================================================================
// Spill expansion — StackVR → IX-indexed instructions
// ============================================================================

// ExpandSpills implements cfg.InstructionSelector.ExpandSpills for the Z80 target.
func (s *instructionSelectorZ80) ExpandSpills(fnCFG *cfg.CFG) {
	for _, block := range fnCFG.Blocks {
		block.MachineInstructions = expandBlock(block.MachineInstructions)
	}
}

func expandBlock(instrs []cfg.MachineInstruction) []cfg.MachineInstruction {
	result := make([]cfg.MachineInstruction, 0, len(instrs))
	for _, mi := range instrs {
		z, ok := mi.(*MachineInstrZ80)
		if !ok {
			// Pass non-Z80 pseudo-instructions (e.g. MachineInstrComment) through unchanged.
			result = append(result, mi)
			continue
		}
		result = append(result, expandInstr(z)...)
	}
	return result
}

// expandInstr replaces a single instruction that has one or more StackVR
// operands with the equivalent IX-indexed instruction(s).
//
// Patterns handled:
//   - Src1 = StackVR  →  LD r, (IX+d)            (8-bit or 16-bit)
//   - Result = StackVR →  LD (IX+d), r / LD (IX+d), n   (8-bit or 16-bit)
//
// StackVR in Src2 is not yet implemented (would require a scratch-register
// load sequence); a panic is raised so the gap is immediately visible.
func expandInstr(mi *MachineInstrZ80) []cfg.MachineInstruction {
	resultSV, resultIsStack := mi.Result.(*cfg.StackVR)
	src1SV, src1IsStack := mi.Src1.(*cfg.StackVR)

	if mi.Src2 != nil {
		if _, ok := mi.Src2.(*cfg.StackVR); ok {
			panic("ExpandSpills: StackVR in Src2 position is not yet implemented")
		}
	}
	if !resultIsStack && !src1IsStack {
		return []cfg.MachineInstruction{mi}
	}

	// ── Load from stack: Src1 is StackVR ─────────────────────────────────────
	if src1IsStack && !resultIsStack {
		phys, ok := mi.Result.(*cfg.PhysVR)
		if !ok {
			panic(fmt.Sprintf("ExpandSpills: expected PhysVR result when Src1 is StackVR, got %T", mi.Result))
		}
		d := int8(src1SV.Offset)
		if src1SV.Size() == 8 {
			return []cfg.MachineInstruction{
				&MachineInstrZ80{Opcode: Z80_LD_R_IX, Result: phys, ImmOffset: d},
			}
		}
		// 16-bit: two byte loads — low byte then high byte.
		comp := phys.Reg.Composition
		if len(comp) < 2 {
			panic(fmt.Sprintf("ExpandSpills: 16-bit spill load into register with no composition: %s", phys.Reg.Name))
		}
		return []cfg.MachineInstruction{
			&MachineInstrZ80{Opcode: Z80_LD_R_IX, Result: &cfg.PhysVR{Reg: comp[0]}, ImmOffset: d},
			&MachineInstrZ80{Opcode: Z80_LD_R_IX, Result: &cfg.PhysVR{Reg: comp[1]}, ImmOffset: d + 1},
		}
	}

	// ── Store to stack: Result is StackVR ────────────────────────────────────
	if resultIsStack && !src1IsStack {
		d := int8(resultSV.Offset)
		if resultSV.Size() == 8 {
			switch src := mi.Src1.(type) {
			case *cfg.PhysVR:
				return []cfg.MachineInstruction{
					&MachineInstrZ80{Opcode: Z80_LD_IX_R, Src1: src, ImmOffset: d},
				}
			case *cfg.ImmVR:
				return []cfg.MachineInstruction{
					&MachineInstrZ80{Opcode: Z80_LD_IX_N, Src1: src, ImmOffset: d},
				}
			default:
				panic(fmt.Sprintf("ExpandSpills: unhandled Src1 type for 8-bit stack store: %T", mi.Src1))
			}
		}
		// 16-bit: two byte stores — low byte then high byte.
		switch src := mi.Src1.(type) {
		case *cfg.PhysVR:
			comp := src.Reg.Composition
			if len(comp) < 2 {
				panic(fmt.Sprintf("ExpandSpills: 16-bit spill store from register with no composition: %s", src.Reg.Name))
			}
			return []cfg.MachineInstruction{
				&MachineInstrZ80{Opcode: Z80_LD_IX_R, Src1: &cfg.PhysVR{Reg: comp[0]}, ImmOffset: d},
				&MachineInstrZ80{Opcode: Z80_LD_IX_R, Src1: &cfg.PhysVR{Reg: comp[1]}, ImmOffset: d + 1},
			}
		case *cfg.ImmVR:
			return []cfg.MachineInstruction{
				&MachineInstrZ80{Opcode: Z80_LD_IX_N, Src1: cfg.NewImmVR(src.Value&0xFF, 8), ImmOffset: d},
				&MachineInstrZ80{Opcode: Z80_LD_IX_N, Src1: cfg.NewImmVR((src.Value>>8)&0xFF, 8), ImmOffset: d + 1},
			}
		default:
			panic(fmt.Sprintf("ExpandSpills: unhandled Src1 type for 16-bit stack store: %T", mi.Src1))
		}
	}

	panic("ExpandSpills: both Result and Src1 are StackVRs — should not occur")
}
