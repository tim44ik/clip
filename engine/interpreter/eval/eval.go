package eval

import (
	"bufio"
	"bytes"
	ast "clip/engine/interpreter/asts"
	"clip/engine/interpreter/lexer"
	outputprocessor "clip/processors/outputProcessor"
	"clip/processors/reporter"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Environment struct {
	ctx        context.Context
	proc       *os.Process
	stdOutR    io.ReadCloser
	stdInW     io.WriteCloser
	report     *reporter.Report
	client     *outputprocessor.Processor
	outputter  func(any)
	moduleName string
	database   *gorm.DB
	vars       map[string]interface{}
}

type BreakSignal struct{}
type ContinueSignal struct{}

func NewEnvironment(database *gorm.DB, ctx context.Context, report *reporter.Report, name string, outputter func(any)) *Environment {
	return &Environment{database: database, ctx: ctx, vars: make(map[string]interface{}), report: report, moduleName: name, outputter: outputter}
}

func (env *Environment) get(name string) interface{} {
	if val, ok := env.vars[name]; ok {
		return val
	}
	panic(fmt.Sprintf("Variable %s was not defined", name))
}

func (env *Environment) set(name string, value interface{}) {
	env.vars[name] = value
}

func (env *Environment) Eval(prog *ast.Program) {
	defer env.close()
	for _, stmt := range prog.Statements {
		if env.ctx.Err() != nil {
			panic("Context was cancelled")
		}
		env.evalStmt(stmt, false)
	}
}

func (env *Environment) run() {
	var err error
	var stdInR, stdOutW, stdErrW *os.File
	env.stdOutR, stdOutW, err = os.Pipe()
	if err != nil {
		panic(err)
	}

	stdInR, env.stdInW, err = os.Pipe()
	if err != nil {
		panic(err)
	}

	env.proc, err = os.StartProcess("/bin/bash", nil, &os.ProcAttr{
		Files: []*os.File{stdInR, stdOutW, stdOutW}})
	if err != nil {
		panic(err)
	}

	stdInR.Close()
	stdOutW.Close()
	stdErrW.Close()
}

func (env *Environment) close() {
	if env.proc != nil {
		env.stdOutR.Close()
		env.stdInW.Close()
		env.proc.Wait()
	}
}

func (env *Environment) runCommand(verbose bool, cmd string) string {
	if env.ctx.Err() != nil {
		panic("Context error")
	}
	marker := fmt.Sprintf("__END_OF_CMD_%d_%d__", os.Getpid(), time.Now().UnixNano())
	fullCmd := fmt.Sprintf("%s\necho %s\n", cmd, marker)
	_, err := env.stdInW.Write([]byte(fullCmd))
	if err != nil {
		panic(err)
	}

	var output bytes.Buffer
	reader := bufio.NewReader(env.stdOutR)

	for {
		if env.ctx.Err() != nil {
			panic("Context error")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return output.String() + "\n" + err.Error()
		}
		if strings.Contains(line, marker) {
			return output.String()
		}
		if verbose && env.outputter != nil {
			env.outputter(line)
		}
		output.WriteString(line)
	}
	return output.String()
}

func (env *Environment) execBlock(stmts []ast.Stmt) (isBreak bool, isContinue bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(BreakSignal); ok {
				isBreak = true
				return
			}
			if _, ok := r.(ContinueSignal); ok {
				isContinue = true
				return
			}
			panic(r)
		}
	}()
	for _, stmt := range stmts {
		if env.ctx.Err() != nil {
			panic("Context error")
		}
		env.evalStmt(stmt, true)
	}
	return false, false
}

func (env *Environment) evalStmt(stmt ast.Stmt, inLoop bool) {
	if env.ctx.Err() != nil {
		panic("Context error")
	}
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		val := env.evalExpr(s.Expr)
		env.set(s.Name, val)
	case *ast.PrintStmt:
		val := make([]interface{}, 0, len(s.Expr))
		for i := range s.Expr {
			val = append(val, env.evalExpr(s.Expr[i]))
		}

		env.outputter(val)
	case *ast.ExprStmt:
		env.evalExpr(s.Expr)
	case *ast.IfStmt:
		cond := env.evalExpr(s.Cond)
		if isTruthy(cond) {
			for _, st := range s.ThenBody {
				env.evalStmt(st, inLoop)
			}
		} else {
			for _, st := range s.ElseBody {
				env.evalStmt(st, inLoop)
			}
		}
	case *ast.ForStmt:
		if s.Init != nil {
			env.evalStmt(s.Init, false)
		}
		for {
			if s.Cond != nil {
				cond := env.evalExpr(s.Cond)
				if !isTruthy(cond) {
					break
				}
			}

			isBreak, isContinue := env.execBlock(s.Body)

			if isBreak {
				break
			}

			if isContinue {
				if s.Post != nil {
					env.evalStmt(s.Post, false)
				}
				continue
			}

			if s.Post != nil {
				env.evalStmt(s.Post, false)
			}
		}

	case *ast.ContinueStmt:
		if !inLoop {
			panic("continue outside loop")
		}

		panic(ContinueSignal{})
	case *ast.BreakStmt:
		if !inLoop {
			panic("break outside loop")
		}

		panic(BreakSignal{})
	case *ast.AssignIndexStmt:
		arrVal := env.evalExpr(s.Array)
		arr, ok := arrVal.([]interface{})
		if !ok {
			panic("Assignment to an element is only possible for an array")
		}
		idxVal := env.evalExpr(s.Index)
		idx, ok := idxVal.(int)
		if !ok {
			panic("index must be an integer")
		}
		if idx < 0 || idx >= len(arr) {
			panic(fmt.Sprintf("Index %d outside range (length %d)", idx, len(arr)))
		}
		val := env.evalExpr(s.Value)
		arr[idx] = val
	default:
		panic(fmt.Sprintf("Unknown operator: %T", stmt))
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
	if env.ctx.Err() != nil {
		panic("Context error")
	}
	switch e := expr.(type) {
	case *ast.IntLiteral:
		return e.Value
	case *ast.BoolLiteral:
		return e.Value
	case *ast.StringLiteral:
		return e.Value
	case *ast.VarExpr:
		return env.get(e.Name)
	case *ast.UnaryExpr:
		right := env.evalExpr(e.Right)
		switch e.Operator {
		case lexer.TOKEN_MINUS:
			switch v := right.(type) {
			case int:
				return -v
			default:
				panic("Unary minus for numbers only")
			}
		case lexer.TOKEN_NOT:
			return !isTruthy(right)
		default:
			panic("Unknown unary operator")
		}
	case *ast.BinaryExpr:
		left := env.evalExpr(e.Left)
		right := env.evalExpr(e.Right)
		switch e.Operator {
		case lexer.TOKEN_AND:
			return isTruthy(left) && isTruthy(right)
		case lexer.TOKEN_OR:
			leftVal := env.evalExpr(e.Left)
			if isTruthy(leftVal) {
				return true
			}
			rightVal := env.evalExpr(e.Right)
			return isTruthy(rightVal)
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
			panic("Unknown binary operator")
		}
	case *ast.CallExpr:
		switch e.Func {
		case "contains":
			if len(e.Args) != 2 {
				panic("contains requires 2 arguments")
			}
			v := env.evalExpr(e.Args[0])
			sub := env.evalExpr(e.Args[1])
			switch value := v.(type) {
			case string:
				substr := toString(sub)
				return strings.Contains(value, substr)
			case []interface{}:
				return slices.Contains(value, sub)
			default:
				panic("wrong value type")
			}

		case "replace":
			if len(e.Args) != 3 {
				panic("replace requires 3 arguments")
			}
			value := env.evalExpr(e.Args[0])
			old := env.evalExpr(e.Args[1])
			new := env.evalExpr(e.Args[2])
			switch v := value.(type) {
			case string:
				o := toString(old)
				n := toString(new)
				return strings.ReplaceAll(v, o, n)
			case []interface{}:
				return func() []interface{} {
					newSlice := make([]interface{}, len(v))
					for i, val := range v {
						if val == old {
							newSlice[i] = new
							continue
						}
						newSlice[i] = val
					}
					return newSlice
				}()
			default:
				panic("Wrong data type in replace")
			}

		case "split":
			if len(e.Args) != 2 {
				panic("split requires 2 arguments")
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
				panic("len requires 1 argument")
			}
			arg := env.evalExpr(e.Args[0])
			switch v := arg.(type) {
			case []interface{}:
				return len(v)
			case string:
				return len([]rune(v))
			default:
				panic("len only applicable to array or string")
			}
		case "append":
			if len(e.Args) < 2 {
				panic("append requires 2 or more arguments")
			}

			arrVal := env.evalExpr(e.Args[0])
			arr, ok := arrVal.([]interface{})
			if !ok {
				panic("append: first argument must be an array")
			}
			newElems := make([]interface{}, len(arr))
			copy(newElems, arr)
			for i := 1; i < len(e.Args); i++ {
				newElems = append(newElems, env.evalExpr(e.Args[i]))
			}
			return newElems
		case "int":
			if len(e.Args) != 1 {
				panic("int requires exactly 1 argument")
			}
			val := env.evalExpr(e.Args[0])
			return toInt(val)
		case "str":
			if len(e.Args) != 1 {
				panic("str requires exactly 1 argument")
			}
			val := env.evalExpr(e.Args[0])
			return toString(val)
		case "fields":
			if len(e.Args) != 1 {
				panic("fields requires 1 argument")
			}

			s := toString(env.evalExpr(e.Args[0]))
			fields := strings.Fields(s)
			res := make([]interface{}, 0, len(fields))
			for _, f := range fields {
				res = append(res, f)
			}
			return res
		case "run":
			if len(e.Args) < 2 {
				panic("run requires 2 and more arguments")
			}

			if env.proc == nil {
				env.run()
			}

			varg := toString(env.evalExpr(e.Args[0]))
			isv := env.isVerbose(varg)
			var output strings.Builder
			for i := 1; i < len(e.Args); i++ {
				s := toString(env.evalExpr(e.Args[i]))
				output.WriteString(env.runCommand(isv, s))
			}

			return output.String()
		case "runIsolated":
			if len(e.Args) < 2 {
				panic("run requires 2 and more arguments")
			}

			varg := toString(env.evalExpr(e.Args[0]))
			isv := env.isVerbose(varg)
			var output strings.Builder
			for i := 1; i < len(e.Args); i++ {
				s := toString(env.evalExpr(e.Args[i]))
				output.WriteString(env.runIsolated(isv, s))
			}

			return output.String()
		case "report":
			if len(e.Args) < 2 {
				panic("report requires 2 and more arguments")
			}

			body := env.addToReport(e.Args)

			return body
		case "process":
			if len(e.Args) < 2 {
				panic("process requires 2 and more arguments")
			}

			return env.process(e.Args)
		default:
			panic("Unknown function")
		}
	case *ast.ArrayLiteral:
		elems := make([]interface{}, len(e.Elements))
		for i, elem := range e.Elements {
			elems[i] = env.evalExpr(elem)
		}
		return elems
	case *ast.IndexExpr:
		arr := env.evalExpr(e.Array)
		idxVal := env.evalExpr(e.Index)
		idx, ok := idxVal.(int)
		if !ok {
			panic("Index must be an integer")
		}

		if str, ok := arr.(string); ok {
			runes := []rune(str)
			if idx < 0 || idx >= len(runes) {
				panic(fmt.Sprintf("Index %d outside of string range (length %d)", idx, len(runes)))
			}
			return string(runes[idx])
		}

		slice, ok := arr.([]interface{})
		if !ok {
			panic("indexing is only possible for an array or a string")
		}
		if idx < 0 || idx >= len(slice) {
			panic(fmt.Sprintf("Index %d outside of range (length %d)", idx, len(slice)))
		} else if slice[idx] == nil {
			return ""
		}
		return slice[idx]
	case *ast.SliceExpr:
		arr := env.evalExpr(e.Container)

		var length int
		var runes []rune
		var isString bool
		var sliceData []interface{}

		switch v := arr.(type) {
		case string:
			runes = []rune(v)
			length = len(runes)
			isString = true
		case []interface{}:
			sliceData = v
			length = len(sliceData)
		default:
			panic("Wrong data type")
		}

		start := 0
		end := length

		if e.Start != nil {
			startVal := env.evalExpr(e.Start)
			startInt, ok := startVal.(int)
			if !ok {
				panic("The start of the slice must be an integer")
			}
			start = startInt
		}

		if e.End != nil {
			endVal := env.evalExpr(e.End)
			endInt, ok := endVal.(int)
			if !ok {
				panic("The end of the slice must be an integer")
			}
			end = endInt
		}

		if start < 0 {
			start += length
		}

		if end < 0 {
			end += length
		}

		if start < 0 || end < 0 || start > length || end > length || start > end {
			panic(fmt.Sprintf("Invalid slice boundaries:: %d:%d (length %d)", start, end, length))
		}

		if isString {
			return string(runes[start:end])
		} else {
			return sliceData[start:end]
		}
	default:
		panic("Unknown expression node")
	}
}

func (env *Environment) process(args []ast.Expr) []interface{} {
	dbTypeVal := env.evalExpr(args[0])
	dbType := toString(dbTypeVal)
	db := outputprocessor.NewDB(env.database, dbType, env.ctx)

	cache := make(map[string]*outputprocessor.Order)
	software := make([]*outputprocessor.Order, 0)
	cve := make([]*outputprocessor.Order, 0)

	processor := outputprocessor.NewProcessor(db, cache, software, cve)

	data := args[1:]
	output := make([]interface{}, len(data))
	for i, expr := range data {
		val := env.evalExpr(expr)
		output[i] = processor.ProcessOutput(toString(val))
	}
	return output
}

func (env *Environment) addToReport(args []ast.Expr) string {
	rtype := toString(env.evalExpr(args[0]))
	env.checkReporter(rtype)
	if env.report.Reporter.GetFileType() != rtype {
		panic("Unknown report format")
	}
	var content *reporter.ReportContent
	for _, c := range env.report.Content {
		if c.Mname == env.moduleName {
			content = c
			break
		}
	}

	if content == nil {
		env.report.Content = append(env.report.Content, env.report.NewReportContent(env.moduleName))
		content = env.report.Content[len(env.report.Content)-1]
	}

	return env.fillReport(content, args[1:])
}

func (env *Environment) checkReporter(rtype string) {
	var err error
	if env.report.Reporter == nil {
		env.report.Reporter, err = env.report.NewReporter(rtype)
		if err != nil {
			panic(err)
		}
	}

}

func (env *Environment) fillReport(r *reporter.ReportContent, args []ast.Expr) string {
	for i := range args {
		val := env.evalExpr(args[i])
		r.Body += toString(val) + "\n"
	}
	return r.Body
}

func (env *Environment) runIsolated(verbose bool, s string) string {
	cmd := exec.Command("/bin/bash", "-c", s)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	err := cmd.Run()
	if err != nil {
		return stdout.String() + "\n" + err.Error()
	}
	if verbose && env.outputter != nil {
		env.outputter(stdout.String())
	}
	return stdout.String()
}

func (env *Environment) writeStdIn(s string) {
	env.stdInW.Write([]byte(s))
}

func (env *Environment) isVerbose(varg string) bool {
	switch varg {
	case "Verbose":
		return true
	case "":
		return false
	default:
		panic("Unknown argument")
	}
}

func add(a, b interface{}) interface{} {
	if isNil(a) {
		a = ""
	}
	if isNil(b) {
		b = ""
	}
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
	panic(fmt.Sprintf("Unknown types for + operator: %T and %T", a, b))
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

func sub(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av - bv
	}
	panic(fmt.Sprintf("Unknown types for - operator: %T and %T", a, b))
}
func mul(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av * bv
	}
	panic(fmt.Sprintf("Unknown types for * operator: %T and %T", a, b))
}
func div(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		if bv == 0 {
			panic("division by 0")
		}
		return av / bv
	}
	panic(fmt.Sprintf("Unknown types for / operator: %T and %T", a, b))
}
func mod(a, b interface{}) interface{} {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		if bv == 0 {
			panic("Remainder after division by zero")
		}
		return av % bv
	}
	panic(fmt.Sprintf("Unknown types for %% operator : %T and %T", a, b))
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
	panic(fmt.Sprintf("Comparison < is only valid for numbers obtained %T and %T", a, b))
}
func greater(a, b interface{}) bool {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av > bv
	}
	panic(fmt.Sprintf("Comparison > is only valid for numbers obtained %T and %T", a, b))
}
func le(a, b interface{}) bool {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av <= bv
	}
	panic(fmt.Sprintf("Comparison <= is only valid for numbers obtained %T and %T", a, b))
}
func ge(a, b interface{}) bool {
	av, ok1 := a.(int)
	bv, ok2 := b.(int)
	if ok1 && ok2 {
		return av >= bv
	}
	panic(fmt.Sprintf("Comparison >= is only valid for numbers obtained %T and %T", a, b))
}
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	default:
		panic(fmt.Sprintf("Expected string, got %T", v))
	}
}

func toInt(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			panic(fmt.Sprintf("cannot convert '%s' to integer", v))
		}
		return i
	default:
		panic(fmt.Sprintf("cannot convert %T to integer", v))
	}
}
