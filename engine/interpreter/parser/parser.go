package main

import (
	ast "clip/engine/interpreter/asts"
	"clip/engine/interpreter/lexer"
	"fmt"
	"strconv"
)

type Parser struct {
	lexer   *lexer.Lexer
	curTok  lexer.Token
	peekTok lexer.Token
}

func NewParser(lexer *lexer.Lexer) *Parser {
	p := &Parser{lexer: lexer}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curTok = p.peekTok
	p.peekTok = p.lexer.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curTok.Type == t
}

func (p *Parser) expect(t lexer.TokenType) {
	if p.curTokenIs(t) {
		p.nextToken()
		return
	}
	panic(fmt.Sprintf("ожидался %v, получен %v ('%s') на %d:%d", t, p.curTok.Type, p.curTok.Value, p.curTok.Line, p.curTok.Col))
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{}
	for !p.curTokenIs(lexer.TOKEN_EOF) {
		prog.Statements = append(prog.Statements, p.parseStatement())
	}
	return prog
}

func (p *Parser) parseStatement() ast.Stmt {
	switch {
	case p.curTokenIs(lexer.TOKEN_IDENT):
		if p.curTokenIs(lexer.TOKEN_LBRACKET) {
			return p.parseAssignIndex()
		}
		return p.parseAssign()
	case p.curTokenIs(lexer.TOKEN_PRINT):
		return p.parsePrint()
	case p.curTokenIs(lexer.TOKEN_IF):
		return p.parseIf()
	case p.curTokenIs(lexer.TOKEN_FOR):
		return p.parseFor()
	default:
		panic(fmt.Sprintf("неожиданный токен: %v ('%s') на %d:%d", p.curTok.Type, p.curTok.Value, p.curTok.Line, p.curTok.Col))
	}
}

func (p *Parser) parseAssignIndex() *ast.AssignIndexStmt {
	array := &ast.VarExpr{Name: p.curTok.Value}
	p.nextToken()
	p.expect(lexer.TOKEN_LBRACKET)
	index := p.parseExpr()
	p.expect(lexer.TOKEN_RBRACKET)
	p.expect(lexer.TOKEN_ASSIGN)
	value := p.parseExpr()
	return &ast.AssignIndexStmt{Array: array, Index: index, Value: value}
}

func (p *Parser) parseAssign() *ast.AssignStmt {
	name := p.curTok.Value
	p.nextToken()
	p.expect(lexer.TOKEN_ASSIGN)
	expr := p.parseExpr()
	return &ast.AssignStmt{Name: name, Expr: expr}
}

func (p *Parser) parsePrint() *ast.PrintStmt {
	p.nextToken()
	p.expect(lexer.TOKEN_LPAREN)
	expr := p.parseExpr()
	p.expect(lexer.TOKEN_RPAREN)
	return &ast.PrintStmt{Expr: expr}
}

func (p *Parser) parseIf() *ast.IfStmt {
	p.nextToken()
	cond := p.parseExpr()
	p.expect(lexer.TOKEN_THEN)
	thenBody := p.parseBlockUntil(lexer.TOKEN_ELSE, lexer.TOKEN_END)
	var elseBody []ast.Stmt
	if p.curTokenIs(lexer.TOKEN_ELSE) {
		p.nextToken()
		elseBody = p.parseBlockUntil(lexer.TOKEN_END)
	}
	p.expect(lexer.TOKEN_END)
	return &ast.IfStmt{Cond: cond, ThenBody: thenBody, ElseBody: elseBody}
}

func (p *Parser) parseFor() *ast.ForStmt {
	p.nextToken()
	var init *ast.AssignStmt
	if !p.curTokenIs(lexer.TOKEN_SEMICOLON) {
		init = p.parseAssign()
	}
	p.expect(lexer.TOKEN_SEMICOLON)
	var cond ast.Expr
	if !p.curTokenIs(lexer.TOKEN_SEMICOLON) && !p.curTokenIs(lexer.TOKEN_DO) {
		cond = p.parseExpr()
	}
	p.expect(lexer.TOKEN_SEMICOLON)
	var post *ast.AssignStmt
	if !p.curTokenIs(lexer.TOKEN_DO) {
		post = p.parseAssign()
	}
	p.expect(lexer.TOKEN_DO)
	body := p.parseBlockUntil(lexer.TOKEN_END)
	p.expect(lexer.TOKEN_END)
	return &ast.ForStmt{Init: init, Cond: cond, Post: post, Body: body}
}

func (p *Parser) parseBlockUntil(terminals ...lexer.TokenType) []ast.Stmt {
	body := []ast.Stmt{}
	for !p.curTokenIs(lexer.TOKEN_EOF) {
		for _, t := range terminals {
			if p.curTokenIs(t) {
				return body
			}
		}
		body = append(body, p.parseStatement())
	}
	return body
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseCompare()
}

func (p *Parser) parseCompare() ast.Expr {
	left := p.parseAdditive()
	if p.curTokenIs(lexer.TOKEN_EQ) || p.curTokenIs(lexer.TOKEN_NEQ) ||
		p.curTokenIs(lexer.TOKEN_LT) || p.curTokenIs(lexer.TOKEN_GT) ||
		p.curTokenIs(lexer.TOKEN_LE) || p.curTokenIs(lexer.TOKEN_GE) {
		op := p.curTok.Type
		p.nextToken()
		right := p.parseAdditive()
		return &ast.BinaryExpr{Left: left, Operator: op, Right: right}
	}
	return left
}

func (p *Parser) parseAdditive() ast.Expr {
	left := p.parseMultiplicative()
	for p.curTokenIs(lexer.TOKEN_PLUS) || p.curTokenIs(lexer.TOKEN_MINUS) {
		op := p.curTok.Type
		p.nextToken()
		right := p.parseMultiplicative()
		left = &ast.BinaryExpr{Left: left, Operator: op, Right: right}
	}
	return left
}

func (p *Parser) parseMultiplicative() ast.Expr {
	left := p.parseUnary()
	for p.curTokenIs(lexer.TOKEN_MULT) ||
		p.curTokenIs(lexer.TOKEN_DIV) ||
		p.curTokenIs(lexer.TOKEN_MOD) {
		op := p.curTok.Type
		p.nextToken()
		right := p.parseUnary()
		left = &ast.BinaryExpr{Left: left, Operator: op, Right: right}
	}
	return left
}

func (p *Parser) parseUnary() ast.Expr {
	if p.curTokenIs(lexer.TOKEN_MINUS) {
		op := p.curTok.Type
		p.nextToken()
		expr := p.parsePrimary()
		return &ast.UnaryExpr{Operator: op, Right: expr}
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() ast.Expr {
	switch {
	case p.curTokenIs(lexer.TOKEN_NUMBER):
		val, _ := strconv.Atoi(p.curTok.Value)
		p.nextToken()
		return &ast.IntLiteral{Value: val}
	case p.curTokenIs(lexer.TOKEN_TRUE):
		p.nextToken()
		return &ast.BoolLiteral{Value: true}
	case p.curTokenIs(lexer.TOKEN_FALSE):
		p.nextToken()
		return &ast.BoolLiteral{Value: false}
	case p.curTokenIs(lexer.TOKEN_STRING):
		val := p.curTok.Value
		p.nextToken()
		return &ast.StringLiteral{Value: val}
	case p.curTokenIs(lexer.TOKEN_IDENT):
		name := p.curTok.Value
		p.nextToken()
		return &ast.VarExpr{Name: name}
	case p.curTokenIs(lexer.TOKEN_IDENT):
		name := p.curTok.Value
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_LBRACKET) {
			return p.parseIndex(&ast.VarExpr{Name: name})
		}
		return &ast.VarExpr{Name: name}
	case p.curTokenIs(lexer.TOKEN_CONTAINS) ||
		p.curTokenIs(lexer.TOKEN_REPLACE):
		return p.parseCall()
	case p.curTokenIs(lexer.TOKEN_LPAREN):
		p.nextToken()
		expr := p.parseExpr()
		p.expect(lexer.TOKEN_RPAREN)
		return expr
	case p.curTokenIs(lexer.TOKEN_LBRACKET):
		return p.parseArrayLiteral()
	default:
		panic(fmt.Sprintf("ожидалось выражение, получен %v ('%s') на %d:%d", p.curTok.Type, p.curTok.Value, p.curTok.Line, p.curTok.Col))
	}
}

func (p *Parser) parseArrayLiteral() *ast.ArrayLiteral {
	p.nextToken()
	elements := []ast.Expr{}
	if !p.curTokenIs(lexer.TOKEN_RBRACKET) {
		elements = append(elements, p.parseExpr())
		for p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			elements = append(elements, p.parseExpr())
		}
	}
	p.expect(lexer.TOKEN_RBRACKET)
	return &ast.ArrayLiteral{Elements: elements}
}

func (p *Parser) parseIndex(array ast.Expr) ast.Expr {
	p.nextToken()
	index := p.parseExpr()
	p.expect(lexer.TOKEN_RBRACKET)
	return &ast.IndexExpr{Array: array, Index: index}
}

func (p *Parser) parseCall() *ast.CallExpr {
	funcName := p.curTok.Value
	p.nextToken()
	p.expect(lexer.TOKEN_LPAREN)
	args := []ast.Expr{}
	if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		args = append(args, p.parseExpr())
		for p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			args = append(args, p.parseExpr())
		}
	}
	p.expect(lexer.TOKEN_RPAREN)
	return &ast.CallExpr{Func: funcName, Args: args}
}
