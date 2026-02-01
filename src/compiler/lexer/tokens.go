package lexer

import "fmt"

type Location struct {
	Index  int // stream index
	Line   int // code line
	Column int // column on line
}

type TokenId int

const (
	TokenUnknown         TokenId = iota
	TokenInvalid                 // syntax error
	TokenEOF                     // End of File
	TokenIdentifier              // symbol
	TokenNumber                  // literal number
	TokenString                  // "<string>"
	TokenCharacter               // '<c>'
	TokenComment                 // // comment eol|eof
	TokenPlus                    // +
	TokenMinus                   // -
	TokenAsterisk                // *
	TokenSlash                   // /
	TokenPercent                 // %
	TokenWhitespace              // space, \t, \r
	TokenEOL                     // \n
	TokenBracketOpen             // [
	TokenBracketClose            // ]
	TokenBracesOpen              // {
	TokenBracesClose             // }
	TokenParenOpen               // (
	TokenParenClose              // )
	TokenPeriod                  // .
	TokenComma                   // ,
	TokenSemiColon               // ;
	TokenColon                   // :
	TokenEquals                  // =
	TokenGreater                 // >
	TokenLess                    // <
	TokenGreaterOrEquals         // >=
	TokenLessOrEquals            // <=
	TokenNotEquals               // <>
	TokenAmpersant               // &
	TokenHash                    // #
	TokenAt                      // @
	TokenDollar                  // $
	TokenPipe                    // |
	TokenCaret                   // ^
	TokenTilde                   // ~
	TokenBackTick                // `
	TokenExclamation             // !
	TokenQuestion                // ?
	TokenUnderscore              // _
	TokenIncrement               // ++
	TokenDecrement               // --
	TokenAnd                     // and
	TokenOr                      // or
	TokenNot                     // not
	TokenFor                     // for
	TokenIf                      // if
	TokenElsif                   // elsif
	TokenElse                    // else
	TokenSelect                  // select
	TokenCase                    // case
	TokenStruct                  // struct
	TokenType                    // type
	TokenConst                   // const
	TokenAny                     // any
	TokenTrue                    // true
	TokenFalse                   // false
	TokenReturn                  // ret

	//TokenDoubleQuote            // "
	//TokenSingleQuote            // '
)

/** Token
 *  Lexer token identifying text
 */
type Token interface {
	Id() TokenId
	Location() Location
	Text() string
}

type tokenData struct {
	id       TokenId
	location Location
	text     string
}

func (t *tokenData) Id() TokenId {
	return t.id
}
func (t *tokenData) Location() Location {
	return t.location
}
func (t *tokenData) Text() string {
	return t.text
}

type InvalidToken interface {
	Token
	InitialId() TokenId
}
type invalidTokenData struct {
	location  Location
	text      string
	initialId TokenId
}

func (t *invalidTokenData) Id() TokenId {
	return TokenInvalid
}
func (t *invalidTokenData) Location() Location {
	return t.location
}
func (t *invalidTokenData) Text() string {
	return t.text
}
func (t *invalidTokenData) InitialId() TokenId {
	return t.initialId
}

/** TokenStream
 *	Reads Tokens from a stream and allows buffering and rewinding.
 */
type TokenStream interface {
	Peek() Token
	Read() (Token, error)
	Mark() TokenStreamMark
	GotoMark(mark TokenStreamMark) bool
	FromMark(mark TokenStreamMark) []Token
}

type TokenStreamMark struct {
	streamPosition int
}

type tokenStreamImpl struct {
	stream     <-chan Token
	stream_pos int
	buffer     []Token
	buffer_pos int
}

func (ts *tokenStreamImpl) Peek() Token {
	if ts.buffer_pos < 0 || ts.buffer_pos >= len(ts.buffer) {
		return nil
	}
	return ts.buffer[ts.buffer_pos]
}

func (ts *tokenStreamImpl) Read() (Token, error) {
	if ts.stream_pos == ts.buffer_pos {
		token, ok := <-ts.stream
		if ok {
			ts.stream_pos++
			ts.buffer_pos++
			ts.buffer = append(ts.buffer, token)
			return token, nil
		}
	}

	ts.buffer_pos++
	if len(ts.buffer) > ts.buffer_pos {
		return ts.buffer[ts.buffer_pos], nil
	}

	// todo: error: stream closed
	return nil, nil
}
func (ts *tokenStreamImpl) Mark() TokenStreamMark {
	return TokenStreamMark{ts.buffer_pos}
}
func (ts *tokenStreamImpl) GotoMark(mark TokenStreamMark) bool {
	// can go forward and backward within the buffer
	if mark.streamPosition < len(ts.buffer) {
		ts.buffer_pos = mark.streamPosition
		return true
	}

	fmt.Println("ERR: TokenStream.GotoMark => invalid mark.")
	return false
}
func (ts *tokenStreamImpl) FromMark(mark TokenStreamMark) []Token {
	if mark.streamPosition < 0 || mark.streamPosition >= len(ts.buffer) {
		return make([]Token, 0)
	}
	return ts.buffer[mark.streamPosition:ts.buffer_pos]
}

func NewTokenStream(tokens <-chan Token, bufferSize int) TokenStream {
	return &tokenStreamImpl{
		stream:     tokens,
		stream_pos: -1,
		buffer:     make([]Token, 0, bufferSize),
		buffer_pos: -1,
	}
}

func DumpTokens(tokens TokenStream) {
	fmt.Println("========== TOKENS ==========")
	mark := tokens.Mark()
	tokens.GotoMark(TokenStreamMark{0})
	for {
		tok := tokens.Peek()
		if tok == nil || tok.Id() == TokenEOF {
			break
		}
		tokens.Read()
		fmt.Printf("  %v: %s\n", tok.Id(), tok.Text())
	}
	tokens.GotoMark(mark)
	fmt.Println()
}
