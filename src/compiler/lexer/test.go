package lexer

func RunTokenizer(code string) []Token {
	tokenizer := TokenizerFromString(code)

	var tokens []Token
	for token := range tokenizer.Tokens() {
		tokens = append(tokens, token)
	}

	return tokens
}

func OpenTokenStream(code string) TokenStream {
	tokenizer := TokenizerFromString(code)
	return NewTokenStream(tokenizer.Tokens(), 1024)
}
