package parser

import (
	"reflect"
	"zenith/compiler/lexer"
)

//
// additions to generated_nodes
//

// parserNodeData methods

func (n *parserNodeData) tokensOf(tokenId lexer.TokenId) []lexer.Token {
	result := make([]lexer.Token, 0)
	for i := 0; i < len(n._tokens); i++ {
		if n._tokens[i].Id() == tokenId {
			result = append(result, n._tokens[i])
		}
	}
	return result
}
func (n *parserNodeData) childrenOf(t reflect.Type) []interface{} {
	result := make([]interface{}, 0)
	for i := 0; i < len(n._children); i++ {
		child := n._children[i]
		if reflect.TypeOf(child).Implements(t) {
			result = append(result, child)
		}
	}
	return result
}
