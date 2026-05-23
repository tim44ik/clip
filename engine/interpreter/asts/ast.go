package ast

import "clip/engine/interpreter/lexer"

type Node interface {
	node()
}
type Expr interface {
	Node
	exprNode()
}
type Stmt interface {
	Node
	stmtNode()
}

type Program struct {
	Statements []Stmt
}

func (p *Program) node() {}

type AssignStmt struct {
	Name string
	Expr Expr
}

func (a *AssignStmt) node()     {}
func (a *AssignStmt) stmtNode() {}

type PrintStmt struct {
	Expr Expr
}

func (p *PrintStmt) node()     {}
func (p *PrintStmt) stmtNode() {}

type IfStmt struct {
	Cond     Expr
	ThenBody []Stmt
	ElseBody []Stmt
}

func (i *IfStmt) node()     {}
func (i *IfStmt) stmtNode() {}

type ForStmt struct {
	Init *AssignStmt
	Cond Expr
	Post *AssignStmt
	Body []Stmt
}

func (f *ForStmt) node()     {}
func (f *ForStmt) stmtNode() {}

type IntLiteral struct {
	Value int
}

func (i *IntLiteral) node()     {}
func (i *IntLiteral) exprNode() {}

type BoolLiteral struct {
	Value bool
}

func (b *BoolLiteral) node()     {}
func (b *BoolLiteral) exprNode() {}

type StringLiteral struct {
	Value string
}

func (s *StringLiteral) node()     {}
func (s *StringLiteral) exprNode() {}

type VarExpr struct {
	Name string
}

func (v *VarExpr) node()     {}
func (v *VarExpr) exprNode() {}

type BinaryExpr struct {
	Left     Expr
	Operator lexer.TokenType
	Right    Expr
}

func (b *BinaryExpr) node()     {}
func (b *BinaryExpr) exprNode() {}

type UnaryExpr struct {
	Operator lexer.TokenType
	Right    Expr
}

func (u *UnaryExpr) node()     {}
func (u *UnaryExpr) exprNode() {}

type CallExpr struct {
	Func string
	Args []Expr
}

func (c *CallExpr) node()     {}
func (c *CallExpr) exprNode() {}

type ArrayLiteral struct {
	Elements []Expr
}

func (a *ArrayLiteral) node()     {}
func (a *ArrayLiteral) exprNode() {}

type IndexExpr struct {
	Array Expr
	Index Expr
}

func (i *IndexExpr) node()     {}
func (i *IndexExpr) exprNode() {}

type AssignIndexStmt struct {
	Array Expr
	Index Expr
	Value Expr
}

func (a *AssignIndexStmt) node()     {}
func (a *AssignIndexStmt) stmtNode() {}
