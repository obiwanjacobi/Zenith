package z80

import (
	"fmt"
	"strings"

	"zenith/compiler/cfg"
)

// ============================================================================
// MachineInstrZ80
// ============================================================================

// MachineInstrZ80 is a single Z80 machine instruction produced by the
// instruction selector.  It carries VROperand slots rather than physical
// registers; the register allocator later replaces TempVRs with PhysVRs.
//
// Field layout:
//
//	Opcode   — identifies the Z80 instruction (look up descriptor via
//	            Z80InstrDescriptors[Opcode] when needed)
//	CondCode — condition code for conditional jumps/calls (Cond_None otherwise)
//	Result   — the single VROperand written by this instruction (nil if none)
//	Src1     — first source operand (nil if unused)
//	Src2     — second source operand (nil if unused)
//	Target   — branch/call destination block (nil for non-control-flow instrs)
//	           Block layout decides JP vs JR at emit time.
type MachineInstrZ80 struct {
	Opcode   Z80Opcode
	CondCode ConditionCode
	Result   cfg.VROperand   // written operand (nil if none)
	Src1     cfg.VROperand   // first read operand (nil if unused)
	Src2     cfg.VROperand   // second read operand (nil if unused)
	Target   *cfg.BasicBlock // branch/call destination (nil if unused)
}

// GetResult returns the VROperand written by this instruction, or nil.
func (m *MachineInstrZ80) GetResult() cfg.VROperand {
	return m.Result
}

// GetOperands returns every VROperand read by this instruction.
// The order is Src1 then Src2; nil slots are omitted.
func (m *MachineInstrZ80) GetOperands() []cfg.VROperand {
	var ops []cfg.VROperand
	if m.Src1 != nil {
		ops = append(ops, m.Src1)
	}
	if m.Src2 != nil {
		ops = append(ops, m.Src2)
	}
	return ops
}

// String formats the instruction as a human-readable assembly line.
func (m *MachineInstrZ80) String() string {
	var parts []string
	if m.Src1 != nil {
		parts = append(parts, m.Src1.String())
	}
	if m.Src2 != nil {
		parts = append(parts, m.Src2.String())
	}
	if m.Target != nil {
		parts = append(parts, fmt.Sprintf("Block%d", m.Target.ID))
	}
	if m.Result != nil {
		// Result appears first for load/move instructions in assembly notation.
		// Prepend it to the operand list.
		parts = append([]string{m.Result.String()}, parts...)
	}
	return FormatInstruction(m.Opcode, m.CondCode, parts...)
}

// ============================================================================
// Emit helpers used by the instruction selector
// ============================================================================

// emitInstr appends a MachineInstrZ80 to a block's MachineInstructions slice.
func emitInstr(block *cfg.BasicBlock, opcode Z80Opcode, result cfg.VROperand, src1 cfg.VROperand, src2 cfg.VROperand) *MachineInstrZ80 {
	mi := &MachineInstrZ80{
		Opcode: opcode,
		Result: result,
		Src1:   src1,
		Src2:   src2,
	}
	block.MachineInstructions = append(block.MachineInstructions, mi)
	return mi
}

// emitBranch appends a conditional or unconditional branch instruction.
func emitBranch(block *cfg.BasicBlock, opcode Z80Opcode, cc ConditionCode, target *cfg.BasicBlock) *MachineInstrZ80 {
	mi := &MachineInstrZ80{
		Opcode:   opcode,
		CondCode: cc,
		Target:   target,
	}
	block.MachineInstructions = append(block.MachineInstructions, mi)
	return mi
}

// emitCall appends a CALL instruction that reads Src1 (callee label encoded as
// a synthetic ImmVR) and writes Result.
// The caller passes a labelVR holding the numeric address / label index as a
// placeholder; the emitter resolves it to a symbol during assembly output.
func emitCall(block *cfg.BasicBlock, result cfg.VROperand, labelVR cfg.VROperand) *MachineInstrZ80 {
	mi := &MachineInstrZ80{
		Opcode: Z80_CALL_NN,
		Result: result,
		Src1:   labelVR,
	}
	block.MachineInstructions = append(block.MachineInstructions, mi)
	return mi
}

// ============================================================================
// DumpMachineInstructions — debug helper
// ============================================================================

// DumpMachineInstructions prints the machine instructions for one block.
// Satisfies the cfg.DumpCFG dumpInstructions callback signature.
func DumpMachineInstructions(instrs []cfg.MachineInstruction) {
	if len(instrs) == 0 {
		fmt.Println("    (no machine instructions)")
		return
	}
	sb := &strings.Builder{}
	for _, instr := range instrs {
		sb.Reset()
		sb.WriteString("    ")
		sb.WriteString(instr.String())
		fmt.Println(sb.String())
	}
}
