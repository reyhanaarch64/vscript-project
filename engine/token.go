package engine

type TokenKind int

const (
	TokenEOF TokenKind = iota
	TokenIdent
	TokenNumber
	TokenString
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenPercent
	TokenEqual
	TokenEqualEqual
	TokenBang
	TokenBangEqual
	TokenLess
	TokenLessEqual
	TokenGreater
	TokenGreaterEqual
	TokenAndAnd
	TokenOrOr
	TokenLeftParen
	TokenRightParen
	TokenLeftBrace
	TokenRightBrace
	TokenLeftBracket
	TokenRightBracket
	TokenComma
	TokenSemicolon
	TokenDot
	TokenRange
	TokenColon
)

type Token struct {
	Kind TokenKind
	Text string
	Line int
	Col  int
}

func (t Token) String() string {
	return t.Text
}
