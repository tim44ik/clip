package interpreter_tests

import (
	"clip/engine/interpreter/eval"
	"clip/engine/interpreter/lexer"
	"clip/engine/interpreter/parser"
	"context"
	"fmt"
	"testing"
)

func TestCompile(t *testing.T) {
	t.Run("compile", func(t *testing.T) {
		source := `
	contains("123","1")
	%s = "шла саша по шоссе"
	print(contains(%s,"шла"))
	%splitted = split(%s, " ")
	print(contains(%splitted,"шла"))
	%splitted = append(%splitted, 1, true)
	print(contains(%splitted,"1"))
	print(contains(%splitted,true))
	print(replace(%splitted,"шла","бежала"))
	print(replace(%splitted,true,false))
	print(replace(%splitted,1,2))
	print(%splitted)
	print(replace(%s,"шла", "бежала"))
	print(%splitted)
	print(len(%splitted))
	%splitted = append(%splitted, "и сосала сушку")
	print(%splitted)
	%t = %splitted[:3]
	%t = append(%t, "дорожке")
	print(contains(%s[:15],"саша"))
	%f = fields(%s)
	for %i = 1; %i<len(%f); %i=%i+1 do
		if not contains(%f[%i], "саша") and contains(%f[%i-1], "шла") then
			print(%f[%i-1], %f[%i])
			break
		end
		print(%i)
	end

	print(%t, %splitted, 123, "маша")
	`
		l := lexer.NewLexer(source)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(context.Background(), nil, "test", printString)
		env.Eval(prog)
	})
}

func TestParallel(t *testing.T) {
	t.Parallel()
	t.Run("p1", func(t *testing.T) {
		s := `
	%s = "ggfdgdg dfghdttghdh саша"
	%f = fields(%s)
	for %i = 1; %i<len(%f); %i=%i+1 do
		if contains(%f[%i], "саша") then
			print(%f[%i-1], %f[%i])
			continue
		end
		print(%i)
	end
	`
		l := lexer.NewLexer(s)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(context.Background(), nil, "test", printString)
		env.Eval(prog)
	})
	t.Run("p2", func(t *testing.T) {
		s := `
	%s = "саша ggfdgdg dfghdttghdh "
	%f = fields(%s)
for %i = 1; %i<len(%f); %i=%i+1 do
	if not contains(%f[%i], "саша") and contains(%f[%i-1], "шла") then
		print(%f[%i-1], %f[%i])
		break
	end
	print(%i)
end
	`
		l := lexer.NewLexer(s)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(context.Background(), nil, "test", printString)
		env.Eval(prog)
	})
}

func printString(s any) {
	switch s := s.(type) {
	case []any:
		fmt.Print("\n")
		for i := range s {
			fmt.Print(fmt.Sprintf("%v ", s[i]))
		}
	default:
		fmt.Println(fmt.Sprintf("%v", s))
	}
}
