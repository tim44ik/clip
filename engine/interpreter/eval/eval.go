package eval

import (
	ast "clip/engine/interpreter/asts"
	"clip/engine/interpreter/lexer"
	"fmt"
	"strings"
)

type Environment struct {
	vars map[string]interface{}
}

func NewEnvironment() *Environment {
	return &Environment{vars: make(map[string]interface{})}
}

func (e *Environment) Get(name string) interface{} {
	if val, ok := e.vars[name]; ok {
		return val
	}
	panic(fmt.Sprintf("переменная %s не определена", name))
}

func (e *Environment) Set(name string, value interface{}) {
	e.vars[name] = value
}

func (env *Environment) Eval(prog *ast.Program) {
	for _, stmt := range prog.Statements {
		env.evalStmt(stmt)
	}
}

func (env *Environment) evalStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		val := env.evalExpr(s.Expr)
		env.Set(s.Name, val)
	case *ast.PrintStmt:
		val := env.evalExpr(s.Expr)
		switch v := val.(type) {
		case bool:
			fmt.Println(v)
		default:
			fmt.Println(v)
		}
	case *ast.IfStmt:
		cond := env.evalExpr(s.Cond)
		if isTruthy(cond) {
			for _, st := range s.ThenBody {
				env.evalStmt(st)
			}
		} else {
			for _, st := range s.ElseBody {
				env.evalStmt(st)
			}
		}
	case *ast.ForStmt:
		if s.Init != nil {
			env.evalStmt(s.Init)
		}
		for {
			if s.Cond != nil {
				cond := env.evalExpr(s.Cond)
				if !isTruthy(cond) {
					break
				}
			}
			for _, st := range s.Body {
				env.evalStmt(st)
			}
			if s.Post != nil {
				env.evalStmt(s.Post)
			}
		}
	case *ast.AssignIndexStmt:
		arrVal := env.evalExpr(s.Array)
		arr, ok := arrVal.([]interface{})
		if !ok {
			panic("присваивание элементу возможно только для массива")
		}
		idxVal := env.evalExpr(s.Index)
		idx, ok := idxVal.(int)
		if !ok {
			panic("индекс должен быть целым числом")
		}
		if idx < 0 || idx >= len(arr) {
			panic(fmt.Sprintf("индекс %d вне диапазона (длина %d)", idx, len(arr)))
		}
		val := env.evalExpr(s.Value)
		arr[idx] = val
	default:
		panic(fmt.Sprintf("неизвестный оператор: %T", stmt))
	}
}

func isTruthy(v interface{}) bool {
	switch vv := v.(type) {
	case bool:
		return vv
	case int:
		return vv != 0
	case string:
		return vv != ""
	default:
		return false
	}
}

func (env *Environment) evalExpr(expr ast.Expr) interface{} {
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return e.Value
	case *ast.BoolLiteral:
		return e.Value
	case *ast.StringLiteral:
		return e.Value
	case *ast.VarExpr:
		return env.Get(e.Name)
	case *ast.UnaryExpr:
		right := env.evalExpr(e.Right)
		switch e.Operator {
		case lexer.TOKEN_MINUS:
			switch v := right.(type) {
			case int:
				return -v
			default:
				panic("унарный минус только для чисел")
			}
		default:
			panic("неизвестный унарный оператор")
		}
	case *ast.BinaryExpr:
		left := env.evalExpr(e.Left)
		right := env.evalExpr(e.Right)
		switch e.Operator {
		case lexer.TOKEN_PLUS:
			return add(left, right)
		case lexer.TOKEN_MINUS:
			return sub(left, right)
		case lexer.TOKEN_MULT:
			return mul(left, right)
		case lexer.TOKEN_DIV:
			return div(left, right)
		case lexer.TOKEN_MOD:
			return mod(left, right)
		case lexer.TOKEN_EQ:
			return equal(left, right)
		case lexer.TOKEN_NEQ:
			return !equal(left, right)
		case lexer.TOKEN_LT:
			return less(left, right)
		case lexer.TOKEN_GT:
			return greater(left, right)
		case lexer.TOKEN_LE:
			return le(left, right)
		case lexer.TOKEN_GE:
			return ge(left, right)
		default:
			panic("неизвестный бинарный оператор")
		}
	case *ast.CallExpr:
		switch e.Func {
		case "contains":
			if len(e.Args) != 2 {
				panic("contains требует 2 аргумента")
			}
			str := toString(env.evalExpr(e.Args[0]))
			sub := toString(env.evalExpr(e.Args[1]))
			return strings.Contains(str, sub)
		case "replace":
			if len(e.Args) != 3 {
				panic("replace требует 3 аргумента")
			}
			str := toString(env.evalExpr(e.Args[0]))
			old := toString(env.evalExpr(e.Args[1]))
			newStr := toString(env.evalExpr(e.Args[2]))
			return strings.ReplaceAll(str, old, newStr)
		case "split":
			if len(e.Args) != 2 {
				panic("split требует 2 аргумента: строка, разделитель")
			}
			str := toString(env.evalExpr(e.Args[0]))
			sep := toString(env.evalExpr(e.Args[1]))
			parts := strings.Split(str, sep)
			res := make([]interface{}, len(parts))
			for i, p := range parts {
				res[i] = p
			}
			return res
		case "len":
			if len(e.Args) != 1 {
				panic("len требует 1 аргумент")
			}
			arg := env.evalExpr(e.Args[0])
			switch v := arg.(type) {
			case []interface{}:
				return len(v)
			case string:
				return len(v)
			default:
				panic("len применим только к массиву или строке")
			}
		default:
			panic("неизвестная функция")
		}
	case *ast.ArrayLiteral:
		elems := make([]interface{}, len(e.Elements))
		for i, elem := range e.Elements {
			elems[i] = env.evalExpr(elem)
		}
		return elems
	case *ast.IndexExpr:
		arrVal := env.evalExpr(e.Array)
		arr, ok := arrVal.([]interface{})
		if !ok {
			panic("индексирование возможно только для массива")
		}
		idxVal := env.evalExpr(e.Index)
		idx, ok := idxVal.(int)
		if !ok {
			panic("индекс должен быть целым числом")
		}
		if idx < 0 || idx >= len(arr) {
			panic(fmt.Sprintf("индекс %d вне диапазона (длина %d)", idx, len(arr)))
		}
		return arr[idx]
	default:
		panic("неизвестный узел выражения")
	}
}

func add(a, b interface{}) interface{} {
	switch av := a.(type) {
	case int:
		switch bv := b.(type) {
		case int:
			return av + bv
		}
	case string:
		switch bv := b.(type) {
		case string:
			return av + bv
		}
	}
	panic(fmt.Sprintf("неподдерживаемые типы для +: %T и %T", a, b))
}
func sub(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av - bv
	}
	panic(fmt.Sprintf("неподдерживаемые типы для -: %T и %T", a, b))
}
func mul(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av * bv
	}
	panic(fmt.Sprintf("неподдерживаемые типы для *: %T и %T", a, b))
}
func div(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		if bv == 0 {
			panic("деление на ноль")
		}
		return av / bv
	}
	panic(fmt.Sprintf("неподдерживаемые типы для /: %T и %T", a, b))
}
func mod(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		if bv == 0 {
			panic("остаток от деления на ноль")
		}
		return av % bv
	}
	panic(fmt.Sprintf("неподдерживаемые типы для %%: %T и %T", a, b))
}
func equal(a, b interface{}) bool {
	switch av := a.(type) {
	case int:
		if bv, ok := b.(int); ok {
			return av == bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return av == bv
		}
	}
	return false
}
func less(a, b interface{}) bool {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av < bv
	}
	panic(fmt.Sprintf("сравнение < допустимо только для чисел, получены %T и %T", a, b))
}
func greater(a, b interface{}) bool {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av > bv
	}
	panic(fmt.Sprintf("сравнение > допустимо только для чисел, получены %T и %T", a, b))
}
func le(a, b interface{}) bool {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av <= bv
	}
	panic(fmt.Sprintf("сравнение <= допустимо только для чисел, получены %T и %T", a, b))
}
func ge(a, b interface{}) bool {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av >= bv
	}
	panic(fmt.Sprintf("сравнение >= допустимо только для чисел, получены %T и %T", a, b))
}
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	panic(fmt.Sprintf("ожидалась строка, получен %T", v))
}
