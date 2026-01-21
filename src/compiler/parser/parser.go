package parser

import (
	"fmt"
	"zenith/compiler/lexer"
)

type parserContext struct {
	source  string
	tokens  lexer.TokenStream
	current lexer.Token
	errors  []ParserError
}

func (ctx *parserContext) error(msg string) {
	err := ParserError{ctx.source, ctx.current.Location(), msg}
	ctx.errors = append(ctx.errors, err)
}
func (ctx *parserContext) internal_error(err error) {
	msg := fmt.Sprintf("INTERNAL ERROR: %s", err.Error())
	ctx.error(msg)
}

func (ctx *parserContext) mark() lexer.TokenStreamMark {
	return ctx.tokens.Mark()
}
func (ctx *parserContext) gotoMark(mark lexer.TokenStreamMark) bool {
	if ctx.tokens.GotoMark(mark) {
		ctx.current = ctx.tokens.Peek()
		return true
	}
	return false
}
func (ctx *parserContext) fromMark(mark lexer.TokenStreamMark) []lexer.Token {
	return ctx.tokens.FromMark(mark)
}

const (
	skipEOL = true
	retEOL  = false
)

// returns the next token that is not whitespace (not eol) or comment
// eol tokens are emitted for the parser to handle statement endings
func (ctx *parserContext) next(skipEOL bool) lexer.Token {
	for {
		t, err := ctx.tokens.Read()
		if err != nil {
			ctx.internal_error(err)
			return t
		}
		if t == nil {
			// stream closed with no more tokens
			return nil
		}
		ctx.current = t
		id := t.Id()
		if skipEOL && id == lexer.TokenEOL {
			continue
		}
		if id != lexer.TokenWhitespace && id != lexer.TokenComment {
			return t
		}
	}
}

// end: eol | eof
// Checks if current token is EOL or EOF and consumes EOL if present
func (ctx *parserContext) end() bool {
	if ctx.is(lexer.TokenEOL) {
		ctx.next(retEOL) // consume EOL
		return true
	}
	return ctx.is(lexer.TokenEOF)
}

// checks if the current token matches the given token Id
func (ctx *parserContext) is(tokenId lexer.TokenId) bool {
	return tokenId == ctx.current.Id()
}

// checks if the current token matches any of the given token Ids
func (ctx *parserContext) isAny(tokenIds []lexer.TokenId) bool {
	for i := 0; i < len(tokenIds); i++ {
		if tokenIds[i] == ctx.current.Id() {
			return true
		}
	}
	return false
}

// calls each parse function in order until one returns a non-nil node
// the token stream is rewound between each attempt
func (ctx *parserContext) parseOr(parseFuncs []func() ParserNode) ParserNode {
	mark := ctx.mark()
	for i := 0; i < len(parseFuncs); i++ {
		node := parseFuncs[i]()
		if node != nil {
			return node
		}
		// no use to continue now
		if ctx.is(lexer.TokenEOF) {
			break
		}
		ctx.gotoMark(mark)
	}
	return nil
}

//
// Parse entry point
//

func Parse(source string, tokens lexer.TokenStream) (ParserNode, []ParserError) {
	ctx := parserContext{source, tokens, nil, make([]ParserError, 0, 10)}
	if ctx.next(skipEOL) != nil {
		node := ctx.compilationUnit()
		return node, ctx.errors
	}
	return nil, ctx.errors
}
