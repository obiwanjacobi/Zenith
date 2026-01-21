package parser

import (
	"fmt"
	"zenith/compiler/lexer"
)

type ParserError struct {
	source   string
	location lexer.Location
	message  string
}

func (err *ParserError) Error() string {
	return fmt.Sprintf("Parser Error (%s at %d, %d) %s",
		err.source, err.location.Line, err.location.Column, err.message)
}
