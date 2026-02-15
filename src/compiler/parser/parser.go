package parser

import (
	"fmt"
	"zenith/compiler"
	"zenith/compiler/lexer"
)

type parserContext struct {
	source  string
	tokens  lexer.TokenStream
	current lexer.Token
	errors  []*compiler.Diagnostic
}

func (ctx *parserContext) appendError(errors *[]*compiler.Diagnostic, msg string) {
	err := compiler.NewDiagnostic(ctx.source, msg, ctx.current.Location(), compiler.PipelineParser, compiler.SeverityError)
	*errors = append(*errors, err)
}

func (ctx *parserContext) error(msg string) {
	ctx.appendError(&ctx.errors, msg)
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
	takeEOL = false
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
		if id == lexer.TokenUnknown {
			ctx.error("unknown token: " + t.Text())
		}
		if id == lexer.TokenInvalid {
			ctx.error("invalid token: " + t.Text())
		}
		if skipEOL && id == lexer.TokenEOL {
			continue
		}
		if id != lexer.TokenWhitespace && id != lexer.TokenComment {
			return t
		}
	}
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
// Prefers nodes without errors; if all have errors, returns the one with fewest errors
func (ctx *parserContext) parseOr(parseFuncs []func() ParserNode) ParserNode {
	mark := ctx.mark()
	var bestNode ParserNode
	var bestMark lexer.TokenStreamMark
	bestErrorCount := -1

	for i := 0; i < len(parseFuncs); i++ {
		node := parseFuncs[i]()
		if node != nil {
			errorCount := len(node.Errors())
			// If this node has no errors, return it immediately
			if errorCount == 0 {
				return node
			}
			// Keep track of node with fewest errors
			if bestNode == nil || errorCount < bestErrorCount {
				bestNode = node
				bestErrorCount = errorCount
				bestMark = ctx.mark() // Save position after parsing this node
			}
		}
		// no use to continue now
		if ctx.is(lexer.TokenEOF) {
			break
		}
		ctx.gotoMark(mark)
	}

	// Restore stream position to match the best node we're returning
	if bestNode != nil {
		ctx.gotoMark(bestMark)
	}

	// Return best node found (may be nil, or may have errors)
	return bestNode
}

//
// Parse entry point
//

// collectErrors recursively collects all errors from nodes in the AST
func collectErrors(node ParserNode, errors []*compiler.Diagnostic) []*compiler.Diagnostic {
	if node == nil {
		return errors
	}

	// Add errors from this node
	errors = append(errors, node.Errors()...)

	// Recursively collect from children
	for _, child := range node.Children() {
		errors = collectErrors(child, errors)
	}

	return errors
}

func Parse(source string, tokens lexer.TokenStream) (ParserNode, []*compiler.Diagnostic) {
	ctx := parserContext{source, tokens, nil, make([]*compiler.Diagnostic, 0, 10)}
	if ctx.next(skipEOL) != nil {
		node := ctx.compilationUnit()

		// Collect all errors from the AST nodes
		allErrors := collectErrors(node, ctx.errors)

		return node, allErrors
	}
	return nil, ctx.errors
}

func DumpAST(ast CompilationUnit) {
	fmt.Println("========== AST ==========")
	fmt.Printf("Compilation Unit with %d declarations\n", len(ast.Declarations()))
	for i, decl := range ast.Declarations() {
		fmt.Printf("  [%d] %T\n", i, decl)
	}
	fmt.Println()
}
