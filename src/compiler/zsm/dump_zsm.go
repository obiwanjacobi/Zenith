package zsm

import (
	"fmt"
	"io"
)

// DumpSemanticModel prints a summary of the semantic model to w.
func DumpSemanticModel(w io.Writer, semCU *SemCompilationUnit) {
	fmt.Fprintln(w, "========== Semantic Model ===========")
	fmt.Fprintf(w, "Semantic Compilation Unit with %d declarations\n", len(semCU.Declarations))
	for _, decl := range semCU.Declarations {
		switch d := decl.(type) {
		case *SemFunctionDecl:
			fmt.Fprintf(w, "  Function: %s (params=%d)\n", d.Name, len(d.Parameters))
		case *SemVariableDecl:
			fmt.Fprintf(w, "  Variable: %s\n", d.Symbol.Name)
		case *SemTypeDecl:
			fmt.Fprintf(w, "  Type: %s\n", d.TypeInfo.Name())
		default:
			fmt.Fprintf(w, "  Unknown: %T\n", decl)
		}
	}
	fmt.Fprintln(w)
}
