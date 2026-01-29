package zir

import (
	"fmt"
	"zenith/compiler/parser"
)

type SemError struct {
	message string
	node    parser.ParserNode
}

func (err *SemError) Error() string {
	if err.node != nil {
		// Get location from the first token of the parser node
		tokens := err.node.Tokens()
		if len(tokens) > 0 {
			loc := tokens[0].Location()
			return fmt.Sprintf("Semantic Error (line %d, col %d): %s",
				loc.Line, loc.Column, err.message)
		}
		return fmt.Sprintf("Semantic Error: %s", err.message)
	}
	return fmt.Sprintf("Semantic Error: %s", err.message)
}

func NewSemError(message string, node parser.ParserNode) *SemError {
	return &SemError{
		message: message,
		node:    node,
	}
}
