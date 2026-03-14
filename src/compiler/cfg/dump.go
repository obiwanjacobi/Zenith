package cfg

import (
	"fmt"
	"io"
)

// DumpCFG prints the control flow graph structure for fnCFG to w.
func DumpCFG(w io.Writer, fnName string, fnCFG *CFG, dumpInstructions func(io.Writer, []MachineInstruction)) {
	fmt.Fprintf(w, "========== Control Flow Graph: %s (Stack Offset: %d) ==========\n", fnName, fnCFG.StackOffset)
	for _, block := range fnCFG.Blocks {
		fmt.Fprintf(w, "  Block %d [%s]: %d sem-instructions, %d tac, %d successors\n",
			block.ID, block.Label, len(block.SemInstructions), len(block.TAC), len(block.Successors))
		for _, succ := range block.Successors {
			fmt.Fprintf(w, "    -> Block %d [%s]\n", succ.ID, succ.Label)
		}
		if dumpInstructions != nil {
			dumpInstructions(w, block.MachineInstructions)
		}
	}
	fmt.Fprintln(w)
}

// DumpTAC prints the TAC instructions for every block in fnCFG to w.
func DumpTAC(w io.Writer, fnName string, fnCFG *CFG) {
	fmt.Fprintf(w, "========== TAC: %s ==========\n", fnName)
	for _, block := range fnCFG.Blocks {
		fmt.Fprintf(w, "  Block %d [%s]:\n", block.ID, block.Label)
		for _, instr := range block.TAC {
			fmt.Fprintf(w, "    %s\n", instr)
		}
	}
	fmt.Fprintln(w)
}
