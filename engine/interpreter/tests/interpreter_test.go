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
	Contains("123","1")
	Run("Verbose","echo 8.8.8.8", "cd /D C:\Users\w\Desktop\clip", "dir")
	RunIsolated("Verbose", "dir")
	Print(Run("", "dir"))
	`
		l := lexer.NewLexer(source)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(nil, "test")
		env.Eval(prog)
	})
}

// %s = "шла саша по шоссе"
// Print(Contains(%s,"шла"))
// %splitted = Split(%s, " ")
// Print(Contains(%splitted,"шла"))
// %splitted = Append(%splitted, 1, true)
// Print(Contains(%splitted,"1"))
// print(contains(%splitted,true))
// print(replace(%splitted,"шла","бежала"))
// print(replace(%splitted,true,false))
// print(replace(%splitted,1,2))
// print(%splitted)
// print(replace(%s,"шла", "бежала"))
// print(%splitted)
// print(len(%splitted))
// %splitted = append(%splitted, "и сосала сушку")
// print(%splitted)
// %t = %splitted[:3]
// %t = append(%t, "дорожке")
// print(contains(%s[:15],"саша"))
// %f = fields(%s)
// for %i = 1; %i<len(%f); %i=%i+1 do
// 	if contains(%f[%i], "саша") and contains(%f[%i-1], "шла") then
// 		print(%f[%i-1], %f[%i])
// 		break
// 	end
// 	print(%i)
// end

// print(%t, %splitted, 123, "маша")
