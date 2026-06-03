package interpreter_tests

import (
	"clip/engine/interpreter/eval"
	"clip/engine/interpreter/lexer"
	"clip/engine/interpreter/parser"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCompile(t *testing.T) {
	t.Run("compile", func(t *testing.T) {
		source := `
	contains("123","1")
	%s = "tom was walking down the street"
	print(contains(%s,"tom"))
	%splitted = split(%s, " ")
	print(contains(%splitted,"tom"))
	%splitted = append(%splitted, 1, true)
	print(contains(%splitted,"1"))
	print(contains(%splitted,true))
	print(replace(%splitted,"walking","running"))
	print(replace(%splitted,true,false))
	print(replace(%splitted,1,2))
	print(%splitted)
	print(replace(%s,"walking", "running"))
	print(%splitted)
	print(len(%splitted))
	%splitted = append(%splitted, "with an ice cream")
	print(%splitted)
	%t = %splitted[:3]
	%t = append(%t, "up")
	print(contains(%s[:15],"tom"))
	%f = fields(%s)
	for %i = 1; %i<len(%f); %i=%i+1 do
		if not contains(%f[%i], "tom") and contains(%f[%i-1], "walking") then
			print(%f[%i-1], %f[%i])
			break
		end
		print(%i)
	end

	print(%t, %splitted, 123, "alice")
	`
		l := lexer.NewLexer(source)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(nil, context.Background(), nil, "test", printString)
		env.Eval(prog)
	})
}

//result
// true
// true
// false
// true
// [tom was running down the street 1 true]
// [tom was walking down the street 1 false]
// [tom was walking down the street 2 true]
// [tom was walking down the street 1 true]
// tom was running down the street
// [tom was walking down the street 1 true]
// 8
// [tom was walking down the street 1 true with an ice cream]
// true
// 1
// 2
// walking down
// [tom was walking up] [tom was walking down the street 1 true with an ice cream] 123 alice

func TestParallel(t *testing.T) {
	t.Parallel()
	t.Run("p1", func(t *testing.T) {
		t.Parallel()
		s := `
	%s = "ggfdgdg dfghdttghdh tom"
	%f = fields(%s)
	for %i = 1; %i<len(%f); %i=%i+1 do
		if contains(%f[%i], "tom") then
			print(%f[%i-1], %f[%i])
		end
		print(%i)
	end
	`
		l := lexer.NewLexer(s)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(nil, context.Background(), nil, "test", printString)
		env.Eval(prog)
	})
	t.Run("p2", func(t *testing.T) {
		t.Parallel()
		s := `
	%s = "tom ggfdgdg dfghdttghdh "
	%f = fields(%s)
for %i = 1; %i<len(%f); %i=%i+1 do
	if not contains(%f[%i], "tom") and contains(%f[%i-1], "walk") then
		print(%f[%i-1], %f[%i])
		break
	end
	print(%i)
end
	`
		l := lexer.NewLexer(s)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(nil, context.Background(), nil, "test", printString)
		env.Eval(prog)
	})
}

// result
// 1
// dfghdttghdh саша
// 1
// 2
func TestPanic(t *testing.T) {
	t.Parallel()

	t.Run("p1", func(t *testing.T) {
		t.Parallel()

		s := `
	%s = "ggfdgdg dfghdttghdh tom"
	%f = fields(%s)
	for %i = 0; %i<len(%f); %i=%i+1 do
		%a = %f[%i] +" "+ str(%i)
		print(%a)
	end
	`
		l := lexer.NewLexer(s)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(nil, context.Background(), nil, "test", printString)

		env.Eval(prog)
	})

	t.Run("p2", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Unexpected panic: %v", r)
			}
		}()

		s := `
	%s = "tom ggfdgdg dfghdttghdh "
	%f = fields(%s)
	for %i = 1; %i<len(%f); %i=%i+1 do
		if not contains(%f[%i], "tom") and contains(%f[%i-1], "walked") then
			print(%f[%i-1], %f[%i])
			break
		end
		print(%i)
	end
	`
		l := lexer.NewLexer(s)
		p := parser.NewParser(l)
		prog := p.ParseProgram()
		env := eval.NewEnvironment(nil, context.Background(), nil, "test", printString)
		env.Eval(prog)

	})
}

func TestContext(t *testing.T) {
	t.Run("context_cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s := `

	for %i = true; %i==true; do
		print(%i)
	end
	`
		l := lexer.NewLexer(s)
		p := parser.NewParser(l)
		prog := p.ParseProgram()

		env := eval.NewEnvironment(nil, ctx, nil, "test_cancel", printString)

		evalDone := make(chan struct{})
		var evalErr error

		go func() {
			defer close(evalDone)
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						evalErr = err
					} else {
						evalErr = fmt.Errorf("%v", r)
					}
				}
			}()

			env.Eval(prog)
		}()

		time.Sleep(50 * time.Millisecond)

		cancel()

		select {
		case <-evalDone:
			if evalErr != nil && !errors.Is(evalErr, context.Canceled) && !strings.Contains(evalErr.Error(), "context canceled") {
				t.Errorf("Interpreter finished with unexpected error %v", evalErr)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("The test lagged could not be finished")
		}
	})

}

func printString(s any) {
	switch s := s.(type) {
	case []any:
		fmt.Print("\n")
		for i := range s {
			fmt.Printf("%v ", s[i])
		}
	default:
		fmt.Printf("\n%v", s)
	}
}
