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
	// ── Prologue / Epilogue ──────────────────────────────────────────────────

	// BindParameters emits a copy from each calling-convention register into
	// its parameter TempVR, constraining the TempVR's AllowedSet to the ABI
	// register. Called PRE-regalloc during SelectInstructions so that constraint
	// propagation and the linear scan see the incoming register binding.
	BindParameters(entryBlock *BasicBlock, fnCFG *CFG)

	// SelectPrologue emits the stack frame setup into the entry block:
	// LD HL,-n ; ADD HL,SP ; LD SP,HL. Called POST-regalloc so that
	// StackFrame.Size() reflects all spill slots added during allocation.
	// Uses PhysVR operands directly; does not go through register allocation.
	SelectPrologue(entryBlock *BasicBlock, fnCFG *CFG)

	// SelectEpilogue emits the stack frame teardown into the exit block,
	// prepended before the return-value move and RET: LD HL,n ; ADD HL,SP ;
	// LD SP,HL. Called POST-regalloc for the same reason as SelectPrologue.
	// Uses PhysVR operands directly; does not go through register allocation.
	SelectEpilogue(exitBlock *BasicBlock, fnCFG *CFG)

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

	// ── Arithmetic / logical ─────────────────────────────────────────────────────

	// SelectBinOp handles TacBinOp: Dst = Left Op Right.
	SelectBinOp(block *BasicBlock, instr *TacBinOp)

	// ── Unary operations ─────────────────────────────────────────────────────

	// SelectUnary handles TacUnary: Dst = Op Operand.
	SelectUnary(block *BasicBlock, instr *TacUnary)

	// ── Comparisons and control flow ─────────────────────────────────────

	// SelectBranchCond handles TacBranchCond: fused compare + conditional branch.
	SelectBranchCond(block *BasicBlock, instr *TacBranchCond)

	// SelectCompare handles TacCompare: Dst = (Left Op Right), materialising a bit.
	SelectCompare(block *BasicBlock, instr *TacCompare)

	// SelectJump handles TacJump: unconditional branch.
	SelectJump(block *BasicBlock, instr *TacJump)

	// SelectBranchIf handles TacBranchIf: branch on a pre-computed boolean operand.
	SelectBranchIf(block *BasicBlock, instr *TacBranchIf)

	// ── Calls and returns ──────────────────────────────────────────────────

	// SelectCall handles TacCall: Dst = Fn(Args...).
	SelectCall(block *BasicBlock, instr *TacCall)

	// SelectReturn handles TacReturn: return [Value].
	SelectReturn(block *BasicBlock, exitBlock *BasicBlock, instr *TacReturn)
}

// SelectInstructions runs instruction selection over the entire function CFG.
// It iterates every block's TAC list, dispatches to the appropriate InstructionSelector
// method, and populates each block's MachineInstructions slice.
// This logic is target-independent and shared across all backends.
func SelectInstructions(sel InstructionSelector, fnCFG *CFG) error {
	// Bind incoming parameters to their ABI registers (pre-regalloc: sets
	// AllowedSet constraints so the linear scan honours the calling convention).
	sel.BindParameters(fnCFG.Entry, fnCFG)

	for _, block := range fnCFG.Blocks {
		for _, tac := range block.TAC {
			if err := selectOne(sel, block, fnCFG.Exit, tac); err != nil {
				return err
			}
		}
	}
	return nil
}

func selectOne(sel InstructionSelector, block *BasicBlock, exitBlock *BasicBlock, tac TacInstruction) error {
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
	case *TacBinOp:
		sel.SelectBinOp(block, t)
	case *TacUnary:
		sel.SelectUnary(block, t)
	case *TacBranchCond:
		sel.SelectBranchCond(block, t)
	case *TacCompare:
		sel.SelectCompare(block, t)
	case *TacJump:
		sel.SelectJump(block, t)
	case *TacBranchIf:
		sel.SelectBranchIf(block, t)
	case *TacCall:
		sel.SelectCall(block, t)
	case *TacReturn:
		sel.SelectReturn(block, exitBlock, t)
	default:
		return fmt.Errorf("instruction selector: unhandled TAC %T", tac)
	}
	return nil
}
