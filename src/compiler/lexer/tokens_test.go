package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TokenNumber0(t *testing.T) {
	code := "0"
	tokens := RunTokenizer(code)

	t1 := tokens[0]
	assert.Equal(t, TokenNumber, t1.Id())
	assert.Equal(t, "0", t1.Text())
	assert.Equal(t, 0, t1.Location().Index)
	assert.Equal(t, 1, t1.Location().Line)
	assert.Equal(t, 1, t1.Location().Column)

	eof := tokens[1]
	assert.Equal(t, TokenEOF, eof.Id())
	assert.Equal(t, len(code), eof.Location().Index)
}

func Test_TokenNumber(t *testing.T) {
	code := "1234"
	tokens := RunTokenizer(code)

	first := tokens[0]
	assert.Equal(t, TokenNumber, first.Id())
	assert.Equal(t, "1234", first.Text())
	assert.Equal(t, 0, first.Location().Index)
	assert.Equal(t, 1, first.Location().Line)
	assert.Equal(t, 1, first.Location().Column)

	eof := tokens[1]
	assert.Equal(t, TokenEOF, eof.Id())
	assert.Equal(t, len(code), eof.Location().Index)
}

func Test_TokenNumberAndWS(t *testing.T) {
	code := "1234 4321 "
	tokens := RunTokenizer(code)

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

func Test_TokenHexNumber(t *testing.T) {
	code := "0xA5_0F"
	tokens := RunTokenizer(code)

	first := tokens[0]
	assert.Equal(t, TokenNumber, first.Id())
	assert.Equal(t, "0xA5_0F", first.Text())
	assert.Equal(t, 0, first.Location().Index)
	assert.Equal(t, 1, first.Location().Line)
	assert.Equal(t, 1, first.Location().Column)

	eof := tokens[1]
	assert.Equal(t, TokenEOF, eof.Id())
	assert.Equal(t, len(code), eof.Location().Index)
}

func Test_TokenBinNumber(t *testing.T) {
	code := "0b0110_0011"
	tokens := RunTokenizer(code)

	first := tokens[0]
	assert.Equal(t, TokenNumber, first.Id())
	assert.Equal(t, "0b0110_0011", first.Text())
	assert.Equal(t, 0, first.Location().Index)
	assert.Equal(t, 1, first.Location().Line)
	assert.Equal(t, 1, first.Location().Column)

	eof := tokens[1]
	assert.Equal(t, TokenEOF, eof.Id())
	assert.Equal(t, len(code), eof.Location().Index)
}

func Test_TokenPunctuation(t *testing.T) {
	code := "!@#$%^&*()[]{};:,./?`~=+_-"
	tokens := RunTokenizer(code)

	expected := []TokenId{
		TokenExclamation, TokenAtSign, TokenHash, TokenDollar, TokenPercent, TokenCaret, TokenAmpersant, TokenAsterisk,
		TokenParenOpen, TokenParenClose, TokenBracketOpen, TokenBracketClose, TokenBracesOpen, TokenBracesClose,
		TokenSemiColon, TokenColon, TokenComma, TokenPeriod, TokenSlash,
		TokenQuestion, TokenBackTick, TokenTilde, TokenEquals, TokenPlus, TokenUnderscore, TokenMinus,
		TokenEOF,
	}

	for i := 0; i < len(tokens); i++ {
		assert.Equal(t, expected[i], tokens[i].Id(), tokens[i].Text())
	}
}

func Test_TokenKeywords(t *testing.T) {
	code := "and or not for if elsif else select case struct const any"
	tokens := RunTokenizer(code)

	expected := []TokenId{
		TokenAnd, TokenOr, TokenNot, TokenFor, TokenIf, TokenElsif, TokenElse, TokenSelect,
		TokenCase, TokenStruct, TokenConst, TokenAny,
	}

	// i += 2 => we skip all the TokenWhitespace between the keywords
	for i := 0; i < len(expected); i++ {
		assert.Equal(t, expected[i], tokens[i*2].Id(), tokens[i*2].Text())
	}
}

func Test_TokenIdentifier(t *testing.T) {
	code := "andorfor"
	tokens := RunTokenizer(code)

	id1 := tokens[0]
	assert.Equal(t, TokenIdentifier, id1.Id())
	assert.Equal(t, code, id1.Text())
}

func Test_TokenIdentifierMulitple(t *testing.T) {
	code := "andorfor ifelsifelse"
	tokens := RunTokenizer(code)

	id1 := tokens[0]
	assert.Equal(t, TokenIdentifier, id1.Id())
	assert.Equal(t, "andorfor", id1.Text())

	// tokens[1] - whitespace token skipped

	id2 := tokens[2]
	assert.Equal(t, TokenIdentifier, id2.Id())
	assert.Equal(t, "ifelsifelse", id2.Text())
}

func Test_TokenString(t *testing.T) {
	code := "\"string\""
	tokens := RunTokenizer(code)

	str1 := tokens[0]
	assert.Equal(t, TokenString, str1.Id())
	assert.Equal(t, code, str1.Text())
}

func Test_TokenChar(t *testing.T) {
	code := "'c'"
	tokens := RunTokenizer(code)

	char1 := tokens[0]
	assert.Equal(t, TokenCharacter, char1.Id())
	assert.Equal(t, code, char1.Text())
}

func Test_TokenPublicLabel(t *testing.T) {
	code := "label:"
	tokens := RunTokenizer(code)

	id1 := tokens[0]
	assert.Equal(t, TokenIdentifier, id1.Id())
	assert.Equal(t, "label", id1.Text())

	t2 := tokens[1]
	assert.Equal(t, TokenColon, t2.Id())
	assert.Equal(t, ":", t2.Text())
}

func Test_TokenPrivateLabel(t *testing.T) {
	code := ".label"
	tokens := RunTokenizer(code)

	t1 := tokens[0]
	assert.Equal(t, TokenPeriod, t1.Id())
	assert.Equal(t, ".", t1.Text())

	id2 := tokens[1]
	assert.Equal(t, TokenIdentifier, id2.Id())
	assert.Equal(t, "label", id2.Text())
}

func Test_TokenComment(t *testing.T) {
	code := "// comment \n"
	tokens := RunTokenizer(code)

	t1 := tokens[0]
	assert.Equal(t, TokenComment, t1.Id())
	assert.Equal(t, "// comment ", t1.Text())

	t2 := tokens[1]
	assert.Equal(t, TokenEOL, t2.Id())

	t3 := tokens[2]
	assert.Equal(t, TokenEOF, t3.Id())
}
