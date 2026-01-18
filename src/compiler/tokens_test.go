package compiler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TestReader(t *testing.T) {
	reader := newTestReader("X")

	r, s, err := reader.ReadRune()

	assert.Nil(t, err)
	assert.Equal(t, 1, s)
	assert.Equal(t, 'X', r)
}

func Test_TokenNumber(t *testing.T) {
	code := "1234"
	tokens := runTokenizer(code)

	first := tokens[0]
	assert.Equal(t, TokenNumber, first.Id())
	assert.Equal(t, "1234", first.Text())
	assert.Equal(t, 0, first.Location().Index)
	assert.Equal(t, 1, first.Location().Line)
	assert.Equal(t, 1, first.Location().Column)

	eof := tokens[1]
	assert.Equal(t, TokenEOF, eof.Id())
	assert.Equal(t, 4, eof.Location().Index)
}

func Test_TokenNumberAndWS(t *testing.T) {
	code := "1234 4321 "
	tokens := runTokenizer(code)

	n1 := tokens[0]
	assert.Equal(t, TokenNumber, n1.Id())

	ws1 := tokens[1]
	assert.Equal(t, TokenWhitespace, ws1.Id())
	assert.Equal(t, " ", ws1.Text())
	assert.Equal(t, 4, ws1.Location().Index)
	assert.Equal(t, 1, ws1.Location().Line)
	assert.Equal(t, 5, ws1.Location().Column)

	n2 := tokens[2]
	assert.Equal(t, TokenNumber, n2.Id())

	ws2 := tokens[3]
	assert.Equal(t, TokenWhitespace, ws2.Id())

	eof := tokens[4]
	assert.Equal(t, TokenEOF, eof.Id())
}

func Test_TokenPunctuation(t *testing.T) {
	code := "!@#$%^&*()[]{};':\",./?`~=+_-"
	tokens := runTokenizer(code)

	expected := []TokenType{
		TokenExclamation, TokenAt, TokenHash, TokenDollar, TokenPercent, TokenCaret, TokenAmpersant, TokenAsterisk,
		TokenParenOpen, TokenParenClose, TokenBracketOpen, TokenBracketClose, TokenBracesOpen, TokenBracesClose,
		TokenSemiColon, TokenSingleQuote, TokenColon, TokenDoubleQuote, TokenComma, TokenDot, TokenSlash,
		TokenQuestion, TokenBackTick, TokenTilde, TokenEquals, TokenPlus, TokenUnderscore, TokenMinus,
		TokenEOF,
	}

	for i := 0; i < len(tokens); i++ {
		assert.Equal(t, expected[i], tokens[i].Id(), tokens[i].Text())
	}
}

func runTokenizer(code string) []Token {
	tokenizer := TokenizerFromReader(newTestReader(code))

	var tokens []Token
	for token := range tokenizer.Tokens() {
		tokens = append(tokens, token)
	}

	return tokens
}

// -----------------------------------------------------------------------------

type testReaderImpl struct {
	reader *strings.Reader
}

func (tr *testReaderImpl) ReadRune() (r rune, size int, err error) {
	return tr.reader.ReadRune()
}
func (tr *testReaderImpl) UnreadRune() error {
	return tr.reader.UnreadRune()
}
func newTestReader(code string) CodeReader {
	return &testReaderImpl{
		reader: strings.NewReader(code),
	}
}
