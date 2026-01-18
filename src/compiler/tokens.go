package compiler

type TokenType int

const (
	TokenUnknown      TokenType = iota
	TokenInvalid                // syntax error
	TokenEOF                    // End of File
	TokenIdentifier             // symbol
	TokenNumber                 // literal number
	TokenString                 // "<string>"
	TokenPlus                   // +
	TokenMinus                  // -
	TokenAsterisk               // *
	TokenSlash                  // /
	TokenPercent                // %
	TokenWhitespace             // space, \t, \n, \r
	TokenBracketOpen            // [
	TokenBracketClose           // ]
	TokenBracesOpen             // {
	TokenBracesClose            // }
	TokenParenOpen              // (
	TokenParenClose             // )
	TokenDot                    // .
	TokenComma                  // ,
	TokenSemiColon              // ;
	TokenColon                  // :
	TokenEquals                 // =
	TokenAmpersant              // &
	TokenHash                   // #
	TokenAt                     // @
	TokenDollar                 // $
	TokenPipe                   // |
	TokenCaret                  // ^
	TokenTilde                  // ~
	TokenBackTick               // `
	TokenExclamation            // !
	TokenQuestion               // ?
	TokenDoubleQuote            // "
	TokenSingleQuote            // '
	TokenUnderscore             // _
	TokenAnd                    // and
	TokenOr                     // or
	TokenFor                    // for
	TokenElsif                  // elsif
	TokenElse                   // else
	TokenSwitch                 // switch
	TokenCase                   // case
	TokenStruct                 // struct
	TokenConst                  // const
	TokenAny                    // any
)

type Token interface {
	Id() TokenType
	Location() Location
	Text() string
}

type TokenData struct {
	id       TokenType
	location Location
	text     string
}

func (t TokenData) Id() TokenType {
	return t.id
}
func (t TokenData) Location() Location {
	return t.location
}
func (t TokenData) Text() string {
	return t.text
}

type InvalidToken interface {
	Token
	InitialId() TokenType
}
type InvalidTokenData struct {
	id        TokenType
	location  Location
	text      string
	initialId TokenType
}

func (t InvalidTokenData) Id() TokenType {
	return t.id
}
func (t InvalidTokenData) Location() Location {
	return t.location
}
func (t InvalidTokenData) Text() string {
	return t.text
}
func (t InvalidTokenData) InitailId() TokenType {
	return t.initialId
}
