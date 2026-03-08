package main

import (
"fmt"
"zenith/compiler"
"zenith/compiler/lexer"
"zenith/compiler/parser"
)

func main() {
code := "main: () {\n\tx: u8 = 5\n\tflag: bit = x < 10\n\tif not flag {\n\t\tx = 1\n\t}\n}"
tokens := lexer.OpenTokenStream(code)
_, errs := parser.Parse(&compiler.Source{Name: "test"}, tokens)
fmt.Printf("parse errors (%d): %v\n", len(errs), errs)
}
