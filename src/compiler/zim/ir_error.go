package zim

import (
	"fmt"
	"zenith/compiler/parser"
)

type IRError struct {
	message string
	node    parser.ParserNode
}

func (err *IRError) Error() string {
	if err.node != nil {
		// Get location from the first token of the parser node
		tokens := err.node.Tokens()
		if len(tokens) > 0 {
			loc := tokens[0].Location()
			return fmt.Sprintf("IR Error (line %d, col %d): %s",
				loc.Line, loc.Column, err.message)
		}
		return fmt.Sprintf("IR Error: %s", err.message)
	}
	return fmt.Sprintf("IR Error: %s", err.message)
}

func NewIRError(message string, node parser.ParserNode) *IRError {
	return &IRError{
		message: message,
		node:    node,
	}
}
