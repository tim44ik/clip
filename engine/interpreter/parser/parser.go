package parser

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

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekTok.Type == t
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
		if p.peekTokenIs(lexer.TOKEN_LBRACKET) {
			name := p.curTok.Value
			p.nextToken()
			p.nextToken()
			p.expect(lexer.TOKEN_LBRACKET)
			index := p.parseExpr()
			p.expect(lexer.TOKEN_RBRACKET)
			p.expect(lexer.TOKEN_ASSIGN)
			value := p.parseExpr()
			return &ast.AssignIndexStmt{
				Array: &ast.VarExpr{Name: name},
				Index: index,
				Value: value,
			}
		}
		return p.parseAssign()
	case p.curTokenIs(lexer.TOKEN_PRINT):
		return p.parsePrint()
	case p.curTokenIs(lexer.TOKEN_IF):
		return p.parseIf()
	case p.curTokenIs(lexer.TOKEN_FOR):
		return p.parseFor()
	case p.curTokenIs(lexer.TOKEN_BREAK):
		return p.parseBreak()
	case p.curTokenIs(lexer.TOKEN_CONTINUE):
		return p.parseContinue()
	default:
		if p.isExpressionStart(p.curTok.Type) {
			expr := p.parseExpr()
			return &ast.ExprStmt{Expr: expr}
		}
		panic(fmt.Sprintf("неожиданный токен: %v ('%s') на %d:%d", p.curTok.Type, p.curTok.Value, p.curTok.Line, p.curTok.Col))
	}
}

func (p *Parser) parseContinue() ast.Stmt {
	p.nextToken()
	return &ast.ContinueStmt{}
}

func (p *Parser) parseBreak() ast.Stmt {
	p.nextToken()
	return &ast.BreakStmt{}
}

func (p *Parser) parsePrint() ast.Stmt {
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
	return &ast.PrintStmt{Expr: args}
}

func (p *Parser) parseAssign() *ast.AssignStmt {
	name := p.curTok.Value
	p.nextToken()
	p.expect(lexer.TOKEN_ASSIGN)
	expr := p.parseExpr()
	return &ast.AssignStmt{Name: name, Expr: expr}
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

func (p *Parser) parseLogical() ast.Expr {
	left := p.parseCompare()
	for {
		if p.curTokenIs(lexer.TOKEN_AND) || p.curTokenIs(lexer.TOKEN_OR) {
			op := p.curTok.Type
			p.nextToken()
			right := p.parseCompare()
			left = &ast.BinaryExpr{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}
	return left
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
	return p.parseLogical()
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
	if p.curTokenIs(lexer.TOKEN_MINUS) || p.curTokenIs(lexer.TOKEN_NOT) {
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
		if p.curTokenIs(lexer.TOKEN_LBRACKET) {
			return p.parseIndexOrSlice(&ast.VarExpr{Name: name})
		}
		return &ast.VarExpr{Name: name}
	case p.curTokenIs(lexer.TOKEN_CONTAINS) ||
		p.curTokenIs(lexer.TOKEN_REPLACE) ||
		p.curTokenIs(lexer.TOKEN_SPLIT) ||
		p.curTokenIs(lexer.TOKEN_LEN) ||
		p.curTokenIs(lexer.TOKEN_APPEND) ||
		p.curTokenIs(lexer.TOKEN_FIELDS) ||
		p.curTokenIs(lexer.TOKEN_RUN) ||
		p.curTokenIs(lexer.TOKEN_RUNISOLATED) ||
		p.curTokenIs(lexer.TOKEN_PROCESS) ||
		p.curTokenIs(lexer.TOKEN_REPORT):
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

func (p *Parser) parseIndexOrSlice(array ast.Expr) ast.Expr {
	p.nextToken()
	var start, end ast.Expr
	if !p.curTokenIs(lexer.TOKEN_COLON) && !p.curTokenIs(lexer.TOKEN_RBRACKET) {
		start = p.parseExpr()
	}
	if p.curTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		if !p.curTokenIs(lexer.TOKEN_RBRACKET) {
			end = p.parseExpr()
		}
	} else {
		p.expect(lexer.TOKEN_RBRACKET)
		return &ast.IndexExpr{Array: array, Index: start}
	}
	p.expect(lexer.TOKEN_RBRACKET)
	return &ast.SliceExpr{Container: array, Start: start, End: end}
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

func (p *Parser) isExpressionStart(typ lexer.TokenType) bool {
	switch typ {
	case lexer.TOKEN_NUMBER, lexer.TOKEN_STRING, lexer.TOKEN_TRUE, lexer.TOKEN_FALSE,
		lexer.TOKEN_IDENT, lexer.TOKEN_LPAREN, lexer.TOKEN_LBRACKET,
		lexer.TOKEN_MINUS, lexer.TOKEN_NOT,
		lexer.TOKEN_CONTAINS, lexer.TOKEN_REPLACE, lexer.TOKEN_SPLIT, lexer.TOKEN_LEN,
		lexer.TOKEN_APPEND, lexer.TOKEN_FIELDS, lexer.TOKEN_RUN, lexer.TOKEN_RUNISOLATED,
		lexer.TOKEN_PROCESS, lexer.TOKEN_REPORT:
		return true
	default:
		return false
	}
}
