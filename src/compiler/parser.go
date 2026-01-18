package compiler

import (
	"fmt"
)

type ParserNode interface {
	Children() []ParserNode
	Leafs() []Token
}
type ParserNodeData struct {
	children []ParserNode
	leafs    []Token
}

func (n *ParserNodeData) Children() []ParserNode {
	return n.children
}
func (n *ParserNodeData) Leafs() []Token {
	return n.leafs
}

type ParserError struct {
	source   string
	location Location
	message  string
}

func (err *ParserError) Error() string {
	return fmt.Sprintf("Parser Error (%s at %d, %d) %s",
		err.source, err.location.Line, err.location.Column, err.message)
}

//
// PublicLabel and PrivateLabel
//

type Label interface {
	ParserNode
	Label() string
	IsPublic() bool
}
type PublicLabel struct {
	ParserNodeData
}
type PrivateLabel struct {
	ParserNodeData
}

func (n *PublicLabel) Children() []ParserNode {
	return n.ParserNodeData.Children()
}
func (n *PublicLabel) Leafs() []Token {
	return n.ParserNodeData.Leafs()
}
func (n *PublicLabel) Label() string {
	return n.ParserNodeData.leafs[0].Text()
}
func (n *PublicLabel) IsPublic() bool {
	return true
}
func (n *PrivateLabel) Children() []ParserNode {
	return n.ParserNodeData.Children()
}
func (n *PrivateLabel) Leafs() []Token {
	return n.ParserNodeData.Leafs()
}
func (n *PrivateLabel) Label() string {
	return n.ParserNodeData.leafs[1].Text()
}
func (n *PrivateLabel) IsPublic() bool {
	return false
}

func newLabel(tokens []Token) (Label, error) {
	// not the correct number of tokens
	if len(tokens) != 2 {
		var location Location
		if len(tokens) > 0 {
			location = tokens[0].Location()
		}
		return nil, &ParserError{"", location,
			fmt.Sprintf("Expected 2 tokens to create a Label but got %d.", len(tokens))}
	}

	if tokens[0].Id() == TokenDot && tokens[1].Id() == TokenIdentifier {
		return &PrivateLabel{ParserNodeData: ParserNodeData{children: []ParserNode{}, leafs: tokens}}, nil
	}
	if tokens[0].Id() == TokenIdentifier && tokens[1].Id() == TokenColon {
		return &PublicLabel{ParserNodeData: ParserNodeData{[]ParserNode{}, tokens}}, nil
	}

	// not the correct token types
	return nil, &ParserError{"", tokens[0].Location(),
		fmt.Sprintf("Unexpected token types to create a Label: %d and %d.", tokens[0].Id(), tokens[1].Id())}
}

//
// Parse entry point
//

func Parse(tokens []Token) (ParserNode, error) {
	return newLabel(tokens)
}
