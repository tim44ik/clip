package lexer

import (
	"fmt"
)

type Lexer struct {
	input string
	pos   int
	line  int
	col   int
	ch    byte
	kw    map[string]TokenType
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		input: input,
		pos:   0,
		line:  1,
		col:   1,
		kw:    make(map[string]TokenType),
	}
	l.kw["true"] = TOKEN_TRUE
	l.kw["false"] = TOKEN_FALSE
	l.kw["if"] = TOKEN_IF
	l.kw["then"] = TOKEN_THEN
	l.kw["else"] = TOKEN_ELSE
	l.kw["end"] = TOKEN_END
	l.kw["for"] = TOKEN_FOR
	l.kw["do"] = TOKEN_DO
	l.kw["print"] = TOKEN_PRINT
	l.kw["contains"] = TOKEN_CONTAINS
	l.kw["replace"] = TOKEN_REPLACE
	l.kw["split"] = TOKEN_SPLIT
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.pos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.pos]
	}
	l.pos++
}

func (l *Lexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		if l.ch == '\n' {
			l.line++
			l.col = 1
		} else {
			l.col++
		}
		l.readChar()
	}
}

func (l *Lexer) readNumber() string {
	start := l.pos - 1
	for l.ch >= '0' && l.ch <= '9' {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

func (l *Lexer) readString() string {
	l.readChar()
	start := l.pos - 1
	for l.ch != '"' && l.ch != 0 {
		l.readChar()
	}
	if l.ch == 0 {
		panic("незакрытая строка")
	}
	str := l.input[start : l.pos-1]
	l.readChar()
	return str
}

func (l *Lexer) readIdentifier() string {
	start := l.pos - 1
	for (l.ch >= 'a' && l.ch <= 'z') || (l.ch >= 'A' && l.ch <= 'Z') || l.ch == '_' || l.ch == '%' {
		l.readChar()
	}
	return l.input[start : l.pos-1]
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	tok := Token{Line: l.line, Col: l.col}

	switch l.ch {
	case 0:
		tok.Type = TOKEN_EOF
	case '+':
		tok.Type = TOKEN_PLUS
		tok.Value = "+"
		l.readChar()
	case '-':
		tok.Type = TOKEN_MINUS
		tok.Value = "-"
		l.readChar()
	case '*':
		tok.Type = TOKEN_MULT
		tok.Value = "*"
		l.readChar()
	case '/':
		tok.Type = TOKEN_DIV
		tok.Value = "/"
		l.readChar()
	case '%':
		if l.peekChar() >= 'a' && l.peekChar() <= 'z' ||
			l.peekChar() >= 'A' && l.peekChar() <= 'Z' ||
			l.peekChar() == '_' {
			ident := l.readIdentifier()
			name := ident[1:]
			if kw, ok := l.kw[name]; ok {
				tok.Type = kw
				tok.Value = name
			} else {
				tok.Type = TOKEN_IDENT
				tok.Value = name
			}
			return tok
		} else {
			tok.Type = TOKEN_MOD
			tok.Value = "%"
			l.readChar()
		}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_EQ
			tok.Value = "=="
		} else {
			tok.Type = TOKEN_ASSIGN
			tok.Value = "="
		}
		l.readChar()
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_NEQ
			tok.Value = "!="
			l.readChar()
		} else {
			panic(fmt.Sprintf("неожиданный '!' на %d:%d", l.line, l.col))
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_LE
			tok.Value = "<="
		} else {
			tok.Type = TOKEN_LT
			tok.Value = "<"
		}
		l.readChar()
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_GE
			tok.Value = ">="
		} else {
			tok.Type = TOKEN_GT
			tok.Value = ">"
		}
		l.readChar()
	case '(':
		tok.Type = TOKEN_LPAREN
		tok.Value = "("
		l.readChar()
	case ')':
		tok.Type = TOKEN_RPAREN
		tok.Value = ")"
		l.readChar()
	case ',':
		tok.Type = TOKEN_COMMA
		tok.Value = ","
		l.readChar()
	case ';':
		tok.Type = TOKEN_SEMICOLON
		tok.Value = ";"
		l.readChar()
	case '"':
		tok.Type = TOKEN_STRING
		tok.Value = l.readString()
	case '[':
		tok.Type = TOKEN_LBRACKET
		tok.Value = "["
		l.readChar()
	case ']':
		tok.Type = TOKEN_RBRACKET
		tok.Value = "]"
		l.readChar()
	case ':':
		tok.Type = TOKEN_COLON
		tok.Value = ":"
		l.readChar()
	default:
		if l.ch >= '0' && l.ch <= '9' {
			tok.Type = TOKEN_NUMBER
			tok.Value = l.readNumber()
			return tok
		} else if (l.ch >= 'a' && l.ch <= 'z') || (l.ch >= 'A' && l.ch <= 'Z') || l.ch == '_' {
			ident := l.readIdentifier()
			if kw, ok := l.kw[ident]; ok {
				tok.Type = kw
				tok.Value = ident
			} else {
				panic(fmt.Sprintf("неизвестный идентификатор '%s' на %d:%d", ident, l.line, l.col))
			}
			return tok
		} else {
			panic(fmt.Sprintf("неизвестный символ '%c' на %d:%d", l.ch, l.line, l.col))
		}
	}
	return tok
}
