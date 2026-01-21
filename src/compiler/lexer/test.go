package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func RunTokenizer(code string) []Token {
	tokenizer := TokenizerFromReader(newTestReader(code))

	var tokens []Token
	for token := range tokenizer.Tokens() {
		tokens = append(tokens, token)
	}

	return tokens
}

func OpenTokenStream(code string) TokenStream {
	tokenizer := TokenizerFromReader(newTestReader(code))
	return NewTokenStream(tokenizer.Tokens(), 1024)
}

// ----------------------------------------------------------------------------

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

// ----------------------------------------------------------------------------

func Test_TestReader(t *testing.T) {
	reader := newTestReader("X")

	r, s, err := reader.ReadRune()

	assert.Nil(t, err)
	assert.Equal(t, 1, s)
	assert.Equal(t, 'X', r)
}
