package compiler

import (
	"io"
)

type Location struct {
	Index  int // stream index
	Line   int // code line
	Column int // column on line
}

type CodeReader interface {
	io.RuneScanner
}
