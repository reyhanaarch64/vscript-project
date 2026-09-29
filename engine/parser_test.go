package engine

import "testing"

func TestElseIfParsesAsNestedIf(t *testing.T) {
	source := `
let x = 2;
if x == 1 { print("one"); }
else if x == 2 { print("two"); }
else { print("other"); }
`
	tokens, err := NewLexer(source).Lex()
	if err != nil {
		t.Fatal(err)
	}
	program, err := NewParser(tokens).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Statements) != 2 {
		t.Fatalf("got %d statements", len(program.Statements))
	}
	outer, ok := program.Statements[1].(IfStmt)
	if !ok || outer.Else == nil || len(outer.Else.Statements) != 1 {
		t.Fatal("else-if tidak diparse sebagai nested if")
	}
	if _, ok := outer.Else.Statements[0].(IfStmt); !ok {
		t.Fatal("nested statement bukan IfStmt")
	}
}

func TestVideoDesignClause(t *testing.T) {
	source := `video 1920 1080 30fps 2s design 480 270 { background("#000000"); }`
	tokens, err := NewLexer(source).Lex()
	if err != nil {
		t.Fatal(err)
	}
	program, err := NewParser(tokens).Parse()
	if err != nil {
		t.Fatal(err)
	}
	v := program.Statements[0].(VideoStmt)
	if v.DesignWidth == nil || v.DesignHeight == nil {
		t.Fatal("design clause hilang")
	}
}

func TestInvalidNumberFailsParsing(t *testing.T) {
	source := `let x = 1.2.3;`
	tokens, err := NewLexer(source).Lex()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewParser(tokens).Parse(); err == nil {
		t.Fatal("angka malformed seharusnya menghasilkan error")
	}
}
