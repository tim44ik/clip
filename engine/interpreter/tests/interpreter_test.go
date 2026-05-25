package interpreter_tests

import (
	"clip/engine/interpreter/eval"
	"clip/engine/interpreter/lexer"
	"clip/engine/interpreter/parser"
	"testing"
)

func TestCompile(t *testing.T) {
	t.Run("compile", func(t *testing.T) {
		source := `
	%s = "шла саша по шоссе"
	%splitted = split(%s, " ")
	print(%splitted)
	print(len(%splitted))
	%splitted = append(%splitted, "и сосала сушку")
	print(%splitted)
	%t = %splitted[:3]
	%t = append(%t, "дорожке")
	print(contains(%s[:15],"саша"))
	%f = fields(%s)
	for %i = 0; %i<len(%f); %i=%i+1 do
		if contains(%f[%i], "саша") then
			print(fields(%f[%i]))
			break
		end
		print(%i)
	end
	
	print(%t, %splitted, 123, "маша")
	`
		l := lexer.NewLexer(source)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment()
		env.Eval(prog)
	})
}
