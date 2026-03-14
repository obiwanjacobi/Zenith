package parser

import (
	"fmt"
	"io"
)

// DumpAST prints a summary of the AST to w.
func DumpAST(w io.Writer, ast CompilationUnit) {
	fmt.Fprintln(w, "========== AST ==========")
	fmt.Fprintf(w, "Compilation Unit with %d declarations\n", len(ast.Declarations()))
	for i, decl := range ast.Declarations() {
		fmt.Fprintf(w, "  [%d] %T\n", i, decl)
	}
	fmt.Fprintln(w)
}
