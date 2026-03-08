package cfg

import "fmt"

// ============================================================================
// InstructionSelector interface
// ============================================================================

// InstructionSelector translates TAC instructions for a single function into
// target-specific MachineInstructions. Each Select* method handles one TAC
// node family; the top-level entry point is SelectInstructions.
//
// Implementations live in cfg/<target>/ and receive a shared TempVRAllocator
// so that all VRs produced during selection share a single ID namespace.
//
// Design rules:
//   - Every Select* method appends MachineInstructions to block.MachineInstructions.
//   - VRs are allocated with AllowedSet constrained to exactly what the
//     underlying instruction requires — no more, no less.
//   - Implementations must never mutate a VROperand received as an operand;
//     VRs are immutable once allocated.
//   - Correctness first; redundant LD r,r moves are removed by the peephole pass.
type InstructionSelector interface {
	// ── Memory ──────────────────────────────────────────────────────────────

	// SelectLoad handles TacLoad: Dst = *(Base + Offset).
	SelectLoad(block *BasicBlock, instr *TacLoad)

	// SelectLoadIndexed handles TacLoadIndexed: Dst = *(Base + Index*ElemSize).
	SelectLoadIndexed(block *BasicBlock, instr *TacLoadIndexed)

	// SelectStore handles TacStore: *(Base + Offset) = Value.
	SelectStore(block *BasicBlock, instr *TacStore)

	// SelectStoreIndexed handles TacStoreIndexed: *(Base+Index*ElemSize) = Value.
	SelectStoreIndexed(block *BasicBlock, instr *TacStoreIndexed)

	// SelectInitSeq handles TacInitSeq: sequential literal store into contiguous memory.
	SelectInitSeq(block *BasicBlock, instr *TacInitSeq)

	// SelectStackAddr handles TacStackAddr: Dst = SP + Offset.
	SelectStackAddr(block *BasicBlock, instr *TacStackAddr)

	// ── Data movement ───────────────────────────────────────────────────────

	// SelectCopy handles TacCopy: Dst = Src.
	SelectCopy(block *BasicBlock, instr *TacCopy)

}

// SelectInstructions runs instruction selection over the entire function CFG.
// It iterates every block's TAC list, dispatches to the appropriate InstructionSelector
// method, and populates each block's MachineInstructions slice.
// This logic is target-independent and shared across all backends.
func SelectInstructions(sel InstructionSelector, fnCFG *CFG) error {
	for _, block := range fnCFG.Blocks {
		for _, tac := range block.TAC {
			if err := selectOne(sel, block, tac); err != nil {
				return err
			}
		}
	}
	return nil
}

func selectOne(sel InstructionSelector, block *BasicBlock, tac TacInstruction) error {
	switch t := tac.(type) {
	case *TacLoad:
		sel.SelectLoad(block, t)
	case *TacLoadIndexed:
		sel.SelectLoadIndexed(block, t)
	case *TacStore:
		sel.SelectStore(block, t)
	case *TacStoreIndexed:
		sel.SelectStoreIndexed(block, t)
	case *TacInitSeq:
		sel.SelectInitSeq(block, t)
	case *TacStackAddr:
		sel.SelectStackAddr(block, t)
	case *TacCopy:
		sel.SelectCopy(block, t)
	default:
		return fmt.Errorf("instruction selector: unhandled TAC %T", tac)
	}
	return nil
}
