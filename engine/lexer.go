package engine

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Lexer struct {
	source []rune
	pos    int
	line   int
	col    int
}

func NewLexer(source string) *Lexer {
	return &Lexer{source: []rune(source), line: 1, col: 1}
}

func (l *Lexer) Lex() ([]Token, error) {
	tokens := make([]Token, 0, 256)
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Kind == TokenEOF {
			return tokens, nil
		}
	}
}

func (l *Lexer) next() (Token, error) {
	for l.pos < len(l.source) {
		r := l.source[l.pos]
		if unicode.IsSpace(r) {
			l.advance()
			continue
		}
		if r == '/' && l.peek(1) == '/' {
			for l.pos < len(l.source) && l.source[l.pos] != '\n' {
				l.advance()
			}
			continue
		}
		break
	}

	if l.pos >= len(l.source) {
		return Token{Kind: TokenEOF, Line: l.line, Col: l.col}, nil
	}

	startLine, startCol := l.line, l.col
	r := l.source[l.pos]

	if unicode.IsLetter(r) || r == '_' {
		start := l.pos
		for l.pos < len(l.source) && (unicode.IsLetter(l.source[l.pos]) || unicode.IsDigit(l.source[l.pos]) || l.source[l.pos] == '_') {
			l.advance()
		}
		return Token{Kind: TokenIdent, Text: string(l.source[start:l.pos]), Line: startLine, Col: startCol}, nil
	}

	if unicode.IsDigit(r) || (r == '.' && unicode.IsDigit(l.peek(1))) {
		start := l.pos
		dots := 0
		for l.pos < len(l.source) {
			c := l.source[l.pos]
			if unicode.IsDigit(c) {
				l.advance()
				continue
			}
			if c == '.' {
				if l.peek(1) == '.' {
					break
				}
				if dots > 0 {
					break
				}
				dots++
				l.advance()
				continue
			}
			break
		}
		if l.pos < len(l.source) && l.source[l.pos] == '.' && l.peek(1) == '.' {
			return Token{Kind: TokenNumber, Text: string(l.source[start:l.pos]), Line: startLine, Col: startCol}, nil
		}
		return Token{Kind: TokenNumber, Text: string(l.source[start:l.pos]), Line: startLine, Col: startCol}, nil
	}

	if r == '"' {
		l.advance()
		var b strings.Builder
		for l.pos < len(l.source) {
			c := l.source[l.pos]
			if c == '"' {
				l.advance()
				return Token{Kind: TokenString, Text: b.String(), Line: startLine, Col: startCol}, nil
			}
			if c == '\\' {
				l.advance()
				if l.pos >= len(l.source) {
					return Token{}, fmt.Errorf("baris %d kolom %d: string tidak selesai", startLine, startCol)
				}
				esc := l.source[l.pos]
				l.advance()
				switch esc {
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				case '\\':
					b.WriteByte('\\')
				case '"':
					b.WriteByte('"')
				default:
					b.WriteRune(esc)
				}
				continue
			}
			b.WriteRune(c)
			l.advance()
		}
		return Token{}, fmt.Errorf("baris %d kolom %d: string tidak selesai", startLine, startCol)
	}

	pairs := map[rune]struct {
		one TokenKind
		two TokenKind
		ch2 rune
	}{
		'+': {one: TokenPlus}, '-': {one: TokenMinus}, '*': {one: TokenStar}, '/': {one: TokenSlash}, '%': {one: TokenPercent},
		'=': {one: TokenEqual, two: TokenEqualEqual, ch2: '='},
		'!': {one: TokenBang, two: TokenBangEqual, ch2: '='},
		'&': {one: TokenAndAnd, two: TokenAndAnd, ch2: '&'},
		'|': {one: TokenOrOr, two: TokenOrOr, ch2: '|'},
		'<': {one: TokenLess, two: TokenLessEqual, ch2: '='},
		'>': {one: TokenGreater, two: TokenGreaterEqual, ch2: '='},
		'(': {one: TokenLeftParen}, ')': {one: TokenRightParen}, '{': {one: TokenLeftBrace}, '}': {one: TokenRightBrace},
		'[': {one: TokenLeftBracket}, ']': {one: TokenRightBracket}, ',': {one: TokenComma}, ';': {one: TokenSemicolon},
		'.': {one: TokenDot}, ':': {one: TokenColon},
	}

	if r == '.' && l.peek(1) == '.' {
		l.advance()
		l.advance()
		return Token{Kind: TokenRange, Text: "..", Line: startLine, Col: startCol}, nil
	}

	if p, ok := pairs[r]; ok {
		l.advance()
		if p.two != 0 && l.pos < len(l.source) && l.source[l.pos] == p.ch2 {
			l.advance()
			return Token{Kind: p.two, Text: string(r) + string(p.ch2), Line: startLine, Col: startCol}, nil
		}
		if r == '&' || r == '|' {
			return Token{}, fmt.Errorf("baris %d kolom %d: operator %q harus ditulis ganda", startLine, startCol, string(r)+string(r))
		}
		return Token{Kind: p.one, Text: string(r), Line: startLine, Col: startCol}, nil
	}

	return Token{}, fmt.Errorf("baris %d kolom %d: karakter tidak dikenal %q", startLine, startCol, r)
}

func (l *Lexer) peek(n int) rune {
	idx := l.pos + n
	if idx >= len(l.source) {
		return 0
	}
	return l.source[idx]
}

func (l *Lexer) advance() {
	if l.pos >= len(l.source) {
		return
	}
	if l.source[l.pos] == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	l.pos++
}

func parseNumber(text string) float64 {
	n, _ := strconv.ParseFloat(text, 64)
	return n
}
