package z80

import (
	"fmt"
	"io"
	"strings"

	"zenith/compiler/cfg"
)

// DumpMachineInstructions prints the machine instructions for one block to w.
func DumpMachineInstructions(w io.Writer, block *cfg.BasicBlock) {
	fmt.Fprintf(w, "  Block %d [%s]:\n", block.ID, block.Label)
	sb := &strings.Builder{}
	for _, instr := range block.MachineInstructions {
		sb.Reset()
		sb.WriteString("    ")
		sb.WriteString(instr.String())
		fmt.Fprintln(w, sb.String())
	}
}
