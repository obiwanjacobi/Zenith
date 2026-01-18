package compiler

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

type Tokenizer struct {
	reader     CodeReader
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

			if err == io.EOF {
				token = TokenData{TokenEOF, t.makeLocation(), ""}
				tokenChan <- token
				// we're done
				return
			}
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
	if err == io.EOF {
		return TokenData{TokenEOF, t.makeLocation(), ""}, nil
	}

	location := t.makeLocation()

	var token Token
	switch {
	case unicode.IsSpace(r):
		token, err = t.parseWhitespace(r, location)
	case unicode.IsDigit(r):
		token, err = t.parseNumber(r, location)
	case t.isPunctuation(r):
		token, err = t.parsePunctuation(r, location)
	case unicode.IsLetter(r):
		token, err = t.parseIdentifierOrKeyword(r, location)
	default:
		// todo: parse unknown: token ends on whitespace
		token = TokenData{TokenUnknown, location, string(r)}
	}

	return token, err
}

func (t *Tokenizer) parseIdentifierOrKeyword(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	for {
		r, err := t.read()
		if err == nil && unicode.IsSpace(r) {
			t.unread(r)
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return TokenData{TokenInvalid, location, builder.String()}, err
		}
		builder.WriteRune(r)
	}

	var token Token
	idOrKeyword := builder.String()

	switch idOrKeyword {
	case "and":
		token = TokenData{TokenAnd, location, idOrKeyword}
	case "or":
		token = TokenData{TokenOr, location, idOrKeyword}
	case "for":
		token = TokenData{TokenFor, location, idOrKeyword}
	case "if":
		token = TokenData{TokenIf, location, idOrKeyword}
	case "elsif":
		token = TokenData{TokenElsif, location, idOrKeyword}
	case "else":
		token = TokenData{TokenElse, location, idOrKeyword}
	case "switch":
		token = TokenData{TokenSwitch, location, idOrKeyword}
	case "case":
		token = TokenData{TokenCase, location, idOrKeyword}
	case "struct":
		token = TokenData{TokenStruct, location, idOrKeyword}
	case "const":
		token = TokenData{TokenConst, location, idOrKeyword}
	case "any":
		token = TokenData{TokenAny, location, idOrKeyword}
	default:
		token = TokenData{TokenIdentifier, location, idOrKeyword}
	}

	return token, nil
}

func (t *Tokenizer) parseNumber(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	var invalid = false
	var isHex = false // allows a-f/A-F

	for {
		r, err := t.read()

		if err == nil && !unicode.IsSpace(r) && !unicode.IsDigit(r) {
			if builder.Len() == 1 && first == '0' && r == 'x' || r == 'b' {
				isHex = r == 'x'
				builder.WriteRune(r)
			} else if r == '_' {
				builder.WriteRune(r)
			} else if isHex && isHexLetter(r) {
				builder.WriteRune(r)
			} else {
				invalid = true
			}
		} else if err != nil || !unicode.IsDigit(r) {
			if err != io.EOF {
				t.unread(r)
			}

			var token Token
			if invalid {
				token = InvalidTokenData{TokenInvalid, location, builder.String(), TokenNumber}
			} else {
				token = TokenData{TokenNumber, location, builder.String()}
			}
			return token, err
		} else {
			builder.WriteRune(r)
		}
	}
}

func (t *Tokenizer) isPunctuation(r rune) bool {
	return unicode.IsPunct(r) || r == '$' || r == '^' || r == '=' || r == '+' || r == '`' || r == '~'
}

func (t *Tokenizer) parsePunctuation(first rune, location Location) (Token, error) {
	var token Token
	var text = string(first)

	switch first {
	case '+':
		token = TokenData{TokenPlus, location, text}
	case '-':
		token = TokenData{TokenMinus, location, text}
	case '*':
		token = TokenData{TokenAsterisk, location, text}
	case '/':
		token = TokenData{TokenSlash, location, text}
	case '%':
		token = TokenData{TokenPercent, location, text}
	case '[':
		token = TokenData{TokenBracketOpen, location, text}
	case ']':
		token = TokenData{TokenBracketClose, location, text}
	case '{':
		token = TokenData{TokenBracesOpen, location, text}
	case '}':
		token = TokenData{TokenBracesClose, location, text}
	case '(':
		token = TokenData{TokenParenOpen, location, text}
	case ')':
		token = TokenData{TokenParenClose, location, text}
	case '.':
		token = TokenData{TokenDot, location, text}
	case ',':
		token = TokenData{TokenComma, location, text}
	case ';':
		token = TokenData{TokenSemiColon, location, text}
	case ':':
		token = TokenData{TokenColon, location, text}
	case '=':
		token = TokenData{TokenEquals, location, text}
	case '&':
		token = TokenData{TokenAmpersant, location, text}
	case '#':
		token = TokenData{TokenHash, location, text}
	case '@':
		token = TokenData{TokenAt, location, text}
	case '$':
		token = TokenData{TokenDollar, location, text}
	case '|':
		token = TokenData{TokenPipe, location, text}
	case '^':
		token = TokenData{TokenCaret, location, text}
	case '~':
		token = TokenData{TokenTilde, location, text}
	case '`':
		token = TokenData{TokenBackTick, location, text}
	case '!':
		token = TokenData{TokenExclamation, location, text}
	case '?':
		token = TokenData{TokenQuestion, location, text}
	case '"':
		token = TokenData{TokenDoubleQuote, location, text}
	case '\'':
		token = TokenData{TokenSingleQuote, location, text}
	case '_':
		token = TokenData{TokenUnderscore, location, text}
	default:
		token = TokenData{TokenInvalid, location, text}
	}

	return token, nil
}

func (t *Tokenizer) parseWhitespace(first rune, location Location) (Token, error) {
	var builder strings.Builder
	builder.WriteRune(first)

	for {
		r, err := t.read()
		if err != nil || !unicode.IsSpace(r) {
			if err != io.EOF {
				t.unread(r)
			}
			return TokenData{TokenWhitespace, location, builder.String()}, err
		}

		builder.WriteRune(r)
	}
}

func (t *Tokenizer) read() (rune, error) {
	r, _, err := t.reader.ReadRune()
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
