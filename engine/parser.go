package engine

import (
	"fmt"
	"strings"
)

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser { return &Parser{tokens: tokens} }

func (p *Parser) Parse() (*Program, error) {
	program := &Program{}
	for !p.check(TokenEOF) {
		stmt, err := p.statement()
		if err != nil {
			return nil, err
		}
		program.Statements = append(program.Statements, stmt)
	}
	return program, nil
}

func (p *Parser) statement() (Stmt, error) {
	if p.matchIdent("let") {
		name, err := p.consumeIdent("nama variabel")
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(TokenEqual, "="); err != nil {
			return nil, err
		}
		value, err := p.expression()
		if err != nil {
			return nil, err
		}
		p.optional(TokenSemicolon)
		return LetStmt{Name: name.Text, Value: value}, nil
	}
	if p.matchIdent("function") {
		return p.functionStmt()
	}
	if p.matchIdent("if") {
		return p.ifStmt()
	}
	if p.matchIdent("for") {
		return p.forStmt()
	}
	if p.matchIdent("while") {
		return p.whileStmt()
	}
	if p.matchIdent("break") {
		p.optional(TokenSemicolon)
		return BreakStmt{}, nil
	}
	if p.matchIdent("continue") {
		p.optional(TokenSemicolon)
		return ContinueStmt{}, nil
	}
	if p.matchIdent("return") {
		value, err := p.expression()
		if err != nil {
			return nil, err
		}
		p.optional(TokenSemicolon)
		return ReturnStmt{Value: value}, nil
	}
	if p.matchIdent("video") {
		return p.videoStmt()
	}
	if p.matchIdent("export") {
		return p.exportStmt()
	}

	if p.check(TokenIdent) && p.peekKind(1) == TokenEqual {
		name := p.advance().Text
		p.advance()
		value, err := p.expression()
		if err != nil {
			return nil, err
		}
		p.optional(TokenSemicolon)
		return AssignStmt{Name: name, Value: value}, nil
	}

	expr, err := p.expression()
	if err != nil {
		return nil, err
	}
	p.optional(TokenSemicolon)
	return ExprStmt{Expr: expr}, nil
}

func (p *Parser) functionStmt() (Stmt, error) {
	name, err := p.consumeIdent("nama fungsi")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenLeftParen, "("); err != nil {
		return nil, err
	}
	params := []string{}
	if !p.check(TokenRightParen) {
		for {
			param, e := p.consumeIdent("nama parameter")
			if e != nil {
				return nil, e
			}
			params = append(params, param.Text)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if _, err := p.consume(TokenRightParen, ")"); err != nil {
		return nil, err
	}
	body, err := p.block()
	if err != nil {
		return nil, err
	}
	return FunctionStmt{Name: name.Text, Params: params, Body: body}, nil
}

func (p *Parser) ifStmt() (Stmt, error) {
	cond, err := p.expression()
	if err != nil {
		return nil, err
	}
	thenBlock, err := p.block()
	if err != nil {
		return nil, err
	}
	var elseBlock *BlockStmt
	if p.matchIdent("else") {
		// dukung else if dengan membungkus IfStmt berikutnya ke dalam block else.
		if p.matchIdent("if") {
			nested, nestedErr := p.ifStmt()
			if nestedErr != nil {
				return nil, nestedErr
			}
			elseBlock = &BlockStmt{Statements: []Stmt{nested}}
		} else {
			elseBlock, err = p.block()
			if err != nil {
				return nil, err
			}
		}
	}
	return IfStmt{Cond: cond, Then: thenBlock, Else: elseBlock}, nil
}

func (p *Parser) forStmt() (Stmt, error) {
	name, err := p.consumeIdent("nama iterator")
	if err != nil {
		return nil, err
	}
	if !p.matchIdent("in") {
		return nil, p.errorf("diharapkan 'in'")
	}
	start, err := p.expression()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(TokenRange, ".."); err != nil {
		return nil, err
	}
	end, err := p.expression()
	if err != nil {
		return nil, err
	}
	body, err := p.block()
	if err != nil {
		return nil, err
	}
	return ForStmt{Name: name.Text, Start: start, End: end, Body: body}, nil
}

func (p *Parser) whileStmt() (Stmt, error) {
	cond, err := p.expression()
	if err != nil {
		return nil, err
	}
	body, err := p.block()
	if err != nil {
		return nil, err
	}
	return WhileStmt{Cond: cond, Body: body}, nil
}

func (p *Parser) videoStmt() (Stmt, error) {
	width, err := p.expression()
	if err != nil {
		return nil, err
	}
	height, err := p.expression()
	if err != nil {
		return nil, err
	}
	fps, err := p.expression()
	if err != nil {
		return nil, err
	}
	if p.check(TokenIdent) && strings.ToLower(p.peek().Text) == "fps" {
		p.advance()
	}
	duration, err := p.durationExpression()
	if err != nil {
		return nil, err
	}
	var designWidth, designHeight Expr
	if p.matchIdent("design") {
		designWidth, err = p.expression()
		if err != nil {
			return nil, err
		}
		designHeight, err = p.expression()
		if err != nil {
			return nil, err
		}
	}
	body, err := p.block()
	if err != nil {
		return nil, err
	}
	return VideoStmt{Width: width, Height: height, FPS: fps, Duration: duration, DesignWidth: designWidth, DesignHeight: designHeight, Body: body}, nil
}

func (p *Parser) durationExpression() (Expr, error) {
	if p.check(TokenNumber) {
		n := p.advance()
		suffix := ""
		if p.check(TokenIdent) {
			s := strings.ToLower(p.peek().Text)
			if s == "s" || s == "ms" {
				p.advance()
				suffix = s
			}
		}
		return LiteralExpr{Value: n.Text + suffix}, nil
	}
	return p.expression()
}

func (p *Parser) exportStmt() (Stmt, error) {
	p.matchIdent("video")
	target, err := p.expression()
	if err != nil {
		return nil, err
	}
	codec := "libx264"
	var bitrate Expr
	var audio Expr
	if p.matchIdent("codec") {
		c, e := p.consume(TokenString, "nama codec")
		if e != nil {
			return nil, e
		}
		codec = c.Text
	}
	if p.matchIdent("bitrate") {
		bitrate, err = p.expression()
		if err != nil {
			return nil, err
		}
	}
	if p.matchIdent("audio") {
		audio, err = p.expression()
		if err != nil {
			return nil, err
		}
	}
	p.optional(TokenSemicolon)
	return ExportStmt{Target: target, Codec: codec, Bitrate: bitrate, Audio: audio}, nil
}

func (p *Parser) block() (*BlockStmt, error) {
	if _, err := p.consume(TokenLeftBrace, "{"); err != nil {
		return nil, err
	}
	b := &BlockStmt{}
	for !p.check(TokenRightBrace) && !p.check(TokenEOF) {
		stmt, err := p.statement()
		if err != nil {
			return nil, err
		}
		b.Statements = append(b.Statements, stmt)
	}
	if _, err := p.consume(TokenRightBrace, "}"); err != nil {
		return nil, err
	}
	return b, nil
}

func (p *Parser) expression() (Expr, error) { return p.logicalOr() }
func (p *Parser) logicalOr() (Expr, error) {
	left, err := p.logicalAnd()
	if err != nil {
		return nil, err
	}
	for p.match(TokenOrOr) {
		right, e := p.logicalAnd()
		if e != nil {
			return nil, e
		}
		left = BinaryExpr{Left: left, Op: TokenOrOr, Right: right}
	}
	return left, nil
}
func (p *Parser) logicalAnd() (Expr, error) {
	left, err := p.equality()
	if err != nil {
		return nil, err
	}
	for p.match(TokenAndAnd) {
		right, e := p.equality()
		if e != nil {
			return nil, e
		}
		left = BinaryExpr{Left: left, Op: TokenAndAnd, Right: right}
	}
	return left, nil
}
func (p *Parser) equality() (Expr, error) {
	left, err := p.comparison()
	if err != nil {
		return nil, err
	}
	for p.match(TokenEqualEqual, TokenBangEqual) {
		op := p.prev().Kind
		right, e := p.comparison()
		if e != nil {
			return nil, e
		}
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}
func (p *Parser) comparison() (Expr, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for p.match(TokenLess, TokenLessEqual, TokenGreater, TokenGreaterEqual) {
		op := p.prev().Kind
		right, e := p.term()
		if e != nil {
			return nil, e
		}
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}
func (p *Parser) term() (Expr, error) {
	left, err := p.factor()
	if err != nil {
		return nil, err
	}
	for p.match(TokenPlus, TokenMinus) {
		op := p.prev().Kind
		right, e := p.factor()
		if e != nil {
			return nil, e
		}
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}
func (p *Parser) factor() (Expr, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for p.match(TokenStar, TokenSlash, TokenPercent) {
		op := p.prev().Kind
		right, e := p.unary()
		if e != nil {
			return nil, e
		}
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left, nil
}
func (p *Parser) unary() (Expr, error) {
	if p.match(TokenBang, TokenMinus, TokenPlus) {
		op := p.prev().Kind
		r, err := p.unary()
		if err != nil {
			return nil, err
		}
		return UnaryExpr{Op: op, Right: r}, nil
	}
	return p.call()
}
func (p *Parser) call() (Expr, error) {
	expr, err := p.primary()
	if err != nil {
		return nil, err
	}
	for {
		if p.match(TokenLeftParen) {
			args := []Expr{}
			if !p.check(TokenRightParen) {
				for {
					a, e := p.expression()
					if e != nil {
						return nil, e
					}
					args = append(args, a)
					if !p.match(TokenComma) {
						break
					}
				}
			}
			if _, e := p.consume(TokenRightParen, ")"); e != nil {
				return nil, e
			}
			expr = CallExpr{Callee: expr, Args: args}
			continue
		}
		if p.match(TokenDot) {
			name, e := p.consumeIdent("nama properti")
			if e != nil {
				return nil, e
			}
			expr = PropertyExpr{Object: expr, Name: name.Text}
			continue
		}
		if p.match(TokenLeftBracket) {
			idx, e := p.expression()
			if e != nil {
				return nil, e
			}
			if _, e = p.consume(TokenRightBracket, "]"); e != nil {
				return nil, e
			}
			expr = IndexExpr{Object: expr, Index: idx}
			continue
		}
		break
	}
	return expr, nil
}
func (p *Parser) primary() (Expr, error) {
	if p.match(TokenNumber) {
		return LiteralExpr{Value: parseNumber(p.prev().Text)}, nil
	}
	if p.match(TokenString) {
		return LiteralExpr{Value: p.prev().Text}, nil
	}
	if p.check(TokenIdent) {
		t := p.advance()
		switch t.Text {
		case "true":
			return LiteralExpr{Value: true}, nil
		case "false":
			return LiteralExpr{Value: false}, nil
		case "null":
			return LiteralExpr{Value: nil}, nil
		default:
			return VariableExpr{Name: t.Text}, nil
		}
	}
	if p.match(TokenLeftBracket) {
		items := []Expr{}
		if !p.check(TokenRightBracket) {
			for {
				e, err := p.expression()
				if err != nil {
					return nil, err
				}
				items = append(items, e)
				if !p.match(TokenComma) {
					break
				}
			}
		}
		if _, err := p.consume(TokenRightBracket, "]"); err != nil {
			return nil, err
		}
		return ArrayExpr{Items: items}, nil
	}
	if p.match(TokenLeftBrace) {
		entries := []ObjectEntry{}
		if !p.check(TokenRightBrace) {
			for {
				key, err := p.consumeObjectKey()
				if err != nil {
					return nil, err
				}
				if _, err = p.consume(TokenColon, ":"); err != nil {
					return nil, err
				}
				value, err := p.expression()
				if err != nil {
					return nil, err
				}
				entries = append(entries, ObjectEntry{Key: key, Value: value})
				if !p.match(TokenComma) {
					break
				}
			}
		}
		if _, err := p.consume(TokenRightBrace, "}"); err != nil {
			return nil, err
		}
		return ObjectExpr{Entries: entries}, nil
	}
	if p.match(TokenLeftParen) {
		e, err := p.expression()
		if err != nil {
			return nil, err
		}
		if _, err = p.consume(TokenRightParen, ")"); err != nil {
			return nil, err
		}
		return e, nil
	}
	return nil, p.errorf("ekspresi tidak valid")
}
func (p *Parser) consumeObjectKey() (string, error) {
	if p.match(TokenString) {
		return p.prev().Text, nil
	}
	if p.match(TokenIdent) {
		return p.prev().Text, nil
	}
	return "", p.errorf("diharapkan key object")
}
func (p *Parser) match(kinds ...TokenKind) bool {
	for _, k := range kinds {
		if p.check(k) {
			p.advance()
			return true
		}
	}
	return false
}
func (p *Parser) matchIdent(s string) bool {
	if p.check(TokenIdent) && p.peek().Text == s {
		p.advance()
		return true
	}
	return false
}
func (p *Parser) optional(kind TokenKind) {
	if p.check(kind) {
		p.advance()
	}
}
func (p *Parser) consume(kind TokenKind, what string) (Token, error) {
	if p.check(kind) {
		return p.advance(), nil
	}
	return Token{}, p.errorf("diharapkan %s", what)
}
func (p *Parser) consumeIdent(what string) (Token, error) { return p.consume(TokenIdent, what) }
func (p *Parser) check(kind TokenKind) bool               { return p.peek().Kind == kind }
func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Kind: TokenEOF}
	}
	return p.tokens[p.pos]
}
func (p *Parser) prev() Token { return p.tokens[p.pos-1] }
func (p *Parser) advance() Token {
	t := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}
func (p *Parser) peekKind(n int) TokenKind {
	idx := p.pos + n
	if idx >= len(p.tokens) {
		return TokenEOF
	}
	return p.tokens[idx].Kind
}
func (p *Parser) errorf(format string, args ...interface{}) error {
	t := p.peek()
	return fmt.Errorf("baris %d kolom %d: %s", t.Line, t.Col, fmt.Sprintf(format, args...))
}
