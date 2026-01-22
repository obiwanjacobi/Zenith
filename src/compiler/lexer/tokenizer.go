package lexer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

type CodeReader interface {
	io.RuneScanner
}

type Tokenizer struct {
	reader     CodeReader
	done       bool
	index      int
	line       int
	column     int
	lastColumn int // for unread
}

func TokenizerFromFile(file *os.File) *Tokenizer {
	return TokenizerFromReader(newCodeReader(file))
}
func TokenizerFromReader(reader CodeReader) *Tokenizer {
	return &Tokenizer{
		reader: reader,
		done:   false,
		index:  -1,
		line:   1,
		column: 0,
	}
}

func (t *Tokenizer) Tokens() <-chan Token {
	tokenChan := make(chan Token)

	go func() {
		defer close(tokenChan)

		for {
			token, err := t.parseToken()
			tokenChan <- token

			// if err == io.EOF {
			// 	token = &tokenData{TokenEOF, t.makeLocation(), ""}
			// 	tokenChan <- token
			// 	// we're done
			// 	return
			// }
			if token.Id() == TokenEOF {
				// we're done
				return
			}
			if err != nil {
				// todo: output error
				fmt.Println(err)
				return
			}
		}
	}()

	return tokenChan
}

func (t *Tokenizer) parseToken() (Token, error) {
	r, err := t.read()
	if r == 0 {
		return &tokenData{TokenEOF, t.makeLocation(), ""}, nil
	}

	location := t.makeLocation()

	var token Token
	switch {
	case unicode.IsSpace(r):
		token, err = t.parseWhitespace(r, location)
	case unicode.IsDigit(r):
		token, err = t.parseNumber(r, location)
	case isPunctuation(r):
		token, err = t.parsePunctuation(r, location)
	case unicode.IsLetter(r):
		token, err = t.parseIdentifierOrKeyword(r, location)
	default:
		token, err = t.parseUnknown(r, location)
	}

	return token, err
}

func (t *Tokenizer) parseIdentifierOrKeyword(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	for {
		r, err := t.read()
		if r == 0 {
			break
		}
		if err == nil && (unicode.IsSpace(r) || isPunctuation(r)) {
			t.unread(r)
			break
		}
		if err != nil {
			return &invalidTokenData{location, builder.String(), TokenIdentifier}, err
		}
		builder.WriteRune(r)
	}

	var token Token
	idOrKeyword := builder.String()

	switch idOrKeyword {
	case "and":
		token = &tokenData{TokenAnd, location, idOrKeyword}
	case "or":
		token = &tokenData{TokenOr, location, idOrKeyword}
	case "not":
		token = &tokenData{TokenNot, location, idOrKeyword}
	case "for":
		token = &tokenData{TokenFor, location, idOrKeyword}
	case "if":
		token = &tokenData{TokenIf, location, idOrKeyword}
	case "elsif":
		token = &tokenData{TokenElsif, location, idOrKeyword}
	case "else":
		token = &tokenData{TokenElse, location, idOrKeyword}
	case "select":
		token = &tokenData{TokenSelect, location, idOrKeyword}
	case "case":
		token = &tokenData{TokenCase, location, idOrKeyword}
	case "struct":
		token = &tokenData{TokenStruct, location, idOrKeyword}
	case "const":
		token = &tokenData{TokenConst, location, idOrKeyword}
	case "any":
		token = &tokenData{TokenAny, location, idOrKeyword}
	case "true":
		token = &tokenData{TokenTrue, location, idOrKeyword}
	case "false":
		token = &tokenData{TokenFalse, location, idOrKeyword}
	default:
		token = &tokenData{TokenIdentifier, location, idOrKeyword}
	}

	return token, nil
}

func (t *Tokenizer) parseNumber(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	var isHex = false // allows a-f/A-F

	for {
		r, err := t.read()

		if err != nil || !unicode.IsDigit(r) {
			if err == nil && !unicode.IsDigit(r) {
				// this all is to allow for 0xAA and 0b01101011
				if builder.Len() == 1 && first == '0' && (r == 'x' || r == 'b') {
					isHex = r == 'x'
					builder.WriteRune(r)
					continue
				} else if r == '_' {
					builder.WriteRune(r)
					continue
				} else if isHex && isHexLetter(r) {
					builder.WriteRune(r)
					continue
				}
			}

			t.unread(r)

			token := &tokenData{TokenNumber, location, builder.String()}
			return token, err
		}

		builder.WriteRune(r)
	}
}

func (t *Tokenizer) parsePunctuation(first rune, location Location) (Token, error) {
	var token Token
	var err error
	var text = string(first)

	switch first {
	case '+':
		token, err = t.parsePlusOrIncrement(first, location)
	case '-':
		token, err = t.parseMinusOrDecrement(first, location)
	case '*':
		token = &tokenData{TokenAsterisk, location, text}
	case '/':
		token, err = t.parseCommentOrSlash(first, location)
	case '%':
		token = &tokenData{TokenPercent, location, text}
	case '[':
		token = &tokenData{TokenBracketOpen, location, text}
	case ']':
		token = &tokenData{TokenBracketClose, location, text}
	case '{':
		token = &tokenData{TokenBracesOpen, location, text}
	case '}':
		token = &tokenData{TokenBracesClose, location, text}
	case '(':
		token = &tokenData{TokenParenOpen, location, text}
	case ')':
		token = &tokenData{TokenParenClose, location, text}
	case '.':
		token = &tokenData{TokenPeriod, location, text}
	case ',':
		token = &tokenData{TokenComma, location, text}
	case ';':
		token = &tokenData{TokenSemiColon, location, text}
	case ':':
		token = &tokenData{TokenColon, location, text}
	case '=':
		token = &tokenData{TokenEquals, location, text}
	case '>':
		token, err = t.parseGreaterOrGreaterEquals(first, location)
	case '<':
		token, err = t.parseLessOrLessEqualsOrNotEquals(first, location)
	case '&':
		token = &tokenData{TokenAmpersant, location, text}
	case '#':
		token = &tokenData{TokenHash, location, text}
	case '@':
		token = &tokenData{TokenAt, location, text}
	case '$':
		token = &tokenData{TokenDollar, location, text}
	case '|':
		token = &tokenData{TokenPipe, location, text}
	case '^':
		token = &tokenData{TokenCaret, location, text}
	case '~':
		token = &tokenData{TokenTilde, location, text}
	case '`':
		token = &tokenData{TokenBackTick, location, text}
	case '!':
		token = &tokenData{TokenExclamation, location, text}
	case '?':
		token = &tokenData{TokenQuestion, location, text}
	case '"':
		token, err = t.parseString(first, location)
	case '\'':
		token, err = t.parseChar(first, location)
	case '_':
		token = &tokenData{TokenUnderscore, location, text}
	default:
		token = &invalidTokenData{location, text, TokenUnknown}
	}

	return token, err
}

func (t *Tokenizer) parsePlusOrIncrement(first rune, location Location) (Token, error) {
	return t.parseSingeOrDouble(first, location, TokenPlus, TokenIncrement)
}

func (t *Tokenizer) parseMinusOrDecrement(first rune, location Location) (Token, error) {
	return t.parseSingeOrDouble(first, location, TokenMinus, TokenDecrement)
}

func (t *Tokenizer) parseSingeOrDouble(first rune, location Location, singleId TokenId, doubleId TokenId) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	r, err := t.read()
	if err != nil {
		return &invalidTokenData{location, builder.String(), singleId}, err
	}
	if r != first {
		t.unread(r)
		return &tokenData{singleId, location, builder.String()}, nil
	}
	builder.WriteRune(r)
	return &tokenData{doubleId, location, builder.String()}, nil
}

func (t *Tokenizer) parseCommentOrSlash(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	r, err := t.read()
	if err != nil && err != io.EOF {
		return &invalidTokenData{location, builder.String(), TokenSlash}, err
	}
	if r != '/' {
		t.unread(r)
		return &tokenData{TokenSlash, location, builder.String()}, nil
	}

	// it's a comment
	builder.WriteRune(r)
	for {
		r, err := t.read()
		if err != nil || r == '\n' {
			if err != io.EOF {
				t.unread(r)
			}
			break
		}
		builder.WriteRune(r)
	}
	return &tokenData{TokenComment, location, builder.String()}, err
}

func (t *Tokenizer) parseGreaterOrGreaterEquals(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)
	r, err := t.read()
	if err != nil && err != io.EOF {
		return &invalidTokenData{location, builder.String(), TokenGreater}, err
	}
	if r != '=' {
		t.unread(r)
		return &tokenData{TokenGreater, location, builder.String()}, nil
	}
	builder.WriteRune(r)
	return &tokenData{TokenGreaterOrEquals, location, builder.String()}, nil
}

func (t *Tokenizer) parseLessOrLessEqualsOrNotEquals(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	r, err := t.read()
	if err != nil {
		return &invalidTokenData{location, builder.String(), TokenLess}, err
	}
	if r != '=' && r != '>' {
		t.unread(r)
		return &tokenData{TokenLess, location, builder.String()}, nil
	}
	builder.WriteRune(r)
	if r == '=' {
		return &tokenData{TokenLessOrEquals, location, builder.String()}, nil
	}
	return &tokenData{TokenNotEquals, location, builder.String()}, nil
}

func (t *Tokenizer) parseString(first rune, location Location) (Token, error) {
	return t.parseEnclosed(first, location, TokenString)
}

func (t *Tokenizer) parseChar(first rune, location Location) (Token, error) {
	return t.parseEnclosed(first, location, TokenCharacter)
}

func (t *Tokenizer) parseEnclosed(first rune, location Location, tokenId TokenId) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	for {
		r, err := t.read()
		if err != nil {
			return &invalidTokenData{location, builder.String(), tokenId}, err
		}

		if r == first {
			builder.WriteRune(r)
			return &tokenData{tokenId, location, builder.String()}, err
		}

		builder.WriteRune(r)
	}
}

func (t *Tokenizer) parseWhitespace(first rune, location Location) (Token, error) {

	if first == '\n' {
		return &tokenData{TokenEOL, location, string(first)}, nil
	}

	var builder strings.Builder
	builder.WriteRune(first)

	for {
		r, err := t.read()
		if err != nil || !unicode.IsSpace(r) || r == '\n' {
			if err != io.EOF {
				t.unread(r)
			}

			return &tokenData{TokenWhitespace, location, builder.String()}, err
		}

		builder.WriteRune(r)
	}
}

func (t *Tokenizer) parseUnknown(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	for {
		r, err := t.read()
		if err != nil || unicode.IsSpace(r) {
			if err != io.EOF {
				t.unread(r)
			}
			return &tokenData{TokenUnknown, location, builder.String()}, err
		}

		builder.WriteRune(r)
	}
}

func (t *Tokenizer) read() (rune, error) {
	r, _, err := t.reader.ReadRune()
	if err == io.EOF {
		if !t.done {
			t.done = true
			t.index++ // past end
		}
		return 0, nil
	}

	t.index++
	t.lastColumn = t.column
	if r == '\n' {
		t.line++
		t.column = 0
	} else {
		t.column++
	}
	return r, err
}
func (t *Tokenizer) unread(r rune) error {
	if r == 0 {
		return nil
	}
	err := t.reader.UnreadRune()
	t.index--
	t.column = t.lastColumn
	if r == '\n' {
		t.line--
	}
	return err
}
func (t *Tokenizer) makeLocation() Location {
	return Location{
		Index:  t.index,
		Line:   t.line,
		Column: t.column,
	}
}

// -----------------------------------------------------------------------------

type codeReaderImpl struct {
	CodeReader
	file   *os.File
	reader *bufio.Reader
}

func (cr *codeReaderImpl) ReadRune() (r rune, size int, err error) {
	return cr.reader.ReadRune()
}
func (cr *codeReaderImpl) UnreadRune() error {
	return cr.reader.UnreadRune()
}
func newCodeReader(file *os.File) CodeReader {
	return &codeReaderImpl{
		file:   file,
		reader: bufio.NewReader(file),
	}
}

func isHexLetter(r rune) bool {
	return r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
}
func isPunctuation(r rune) bool {
	return unicode.IsPunct(r) || r == '$' || r == '^' || r == '=' || r == '+' || r == '`' || r == '~' || r == '<' || r == '>' || r == '|' || r == '&'
}
func isWhitespace(r rune) bool {
	return unicode.IsSpace(r) && r != '\n'
}
