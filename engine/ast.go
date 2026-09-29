package engine

type Stmt interface{ stmtNode() }
type Expr interface{ exprNode() }

type Program struct{ Statements []Stmt }
type BlockStmt struct{ Statements []Stmt }

func (BlockStmt) stmtNode() {}

type ExprStmt struct{ Expr Expr }

func (ExprStmt) stmtNode() {}

type LetStmt struct {
	Name  string
	Value Expr
}

func (LetStmt) stmtNode() {}

type AssignStmt struct {
	Name  string
	Value Expr
}

func (AssignStmt) stmtNode() {}

type IfStmt struct {
	Cond Expr
	Then *BlockStmt
	Else *BlockStmt
}

func (IfStmt) stmtNode() {}

type ForStmt struct {
	Name  string
	Start Expr
	End   Expr
	Body  *BlockStmt
}

func (ForStmt) stmtNode() {}

type WhileStmt struct {
	Cond Expr
	Body *BlockStmt
}

func (WhileStmt) stmtNode() {}

type BreakStmt struct{}

func (BreakStmt) stmtNode() {}

type ContinueStmt struct{}

func (ContinueStmt) stmtNode() {}

type ReturnStmt struct{ Value Expr }

func (ReturnStmt) stmtNode() {}

type FunctionStmt struct {
	Name   string
	Params []string
	Body   *BlockStmt
}

func (FunctionStmt) stmtNode() {}

type VideoStmt struct {
	Width        Expr
	Height       Expr
	FPS          Expr
	Duration     Expr
	DesignWidth  Expr
	DesignHeight Expr
	Body         *BlockStmt
}

func (VideoStmt) stmtNode() {}

type ExportStmt struct {
	Target  Expr
	Codec   string
	Bitrate Expr
	Audio   Expr
}

func (ExportStmt) stmtNode() {}

type LiteralExpr struct{ Value interface{} }

func (LiteralExpr) exprNode() {}

type VariableExpr struct{ Name string }

func (VariableExpr) exprNode() {}

type ArrayExpr struct{ Items []Expr }

func (ArrayExpr) exprNode() {}

type ObjectExpr struct{ Entries []ObjectEntry }

func (ObjectExpr) exprNode() {}

type ObjectEntry struct {
	Key   string
	Value Expr
}

type IndexExpr struct {
	Object Expr
	Index  Expr
}

func (IndexExpr) exprNode() {}

type UnaryExpr struct {
	Op    TokenKind
	Right Expr
}

func (UnaryExpr) exprNode() {}

type BinaryExpr struct {
	Left  Expr
	Op    TokenKind
	Right Expr
}

func (BinaryExpr) exprNode() {}

type CallExpr struct {
	Callee Expr
	Args   []Expr
}

func (CallExpr) exprNode() {}

type PropertyExpr struct {
	Object Expr
	Name   string
}

func (PropertyExpr) exprNode() {}
