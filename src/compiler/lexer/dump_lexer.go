package lexer

import (
	"fmt"
	"io"
)

// DumpTokens prints all tokens in the stream to w, then restores the stream position.
func DumpTokens(w io.Writer, tokens TokenStream) {
	fmt.Fprintln(w, "========== TOKENS ==========")
	mark := tokens.Mark()
	tokens.GotoMark(TokenStreamMark{0})
	for {
		tok := tokens.Peek()
		if tok == nil || tok.Id() == TokenEOF {
			break
		}
		tokens.Read()
		fmt.Fprintf(w, "  %v: %s\n", tok.Id(), tok.Text())
	}
	tokens.GotoMark(mark)
	fmt.Fprintln(w)
}
