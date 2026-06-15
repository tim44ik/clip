package scenario

import (
	"clip/engine/interpreter/eval"
	"clip/engine/interpreter/lexer"
	"clip/engine/interpreter/parser"
	"clip/errors"
	"clip/models/modules"
	"clip/processors/reporter"
	"context"
	"fmt"
	"strings"
	"sync"

	_ "embed"

	"gorm.io/gorm"
)

type Scenario struct {
	main          string
	threadNumber  int
	modulesStruct [][]*modules.Module
	report        *reporter.Report
}

func NewScenario(main string, thread int, module [][]*modules.Module) *Scenario {
	return &Scenario{main: main, threadNumber: thread, modulesStruct: module}
}

func (s *Scenario) Execute(database *gorm.DB, errCh chan<- error, ctx context.Context, outputter func(any, *modules.Module)) *reporter.Report {
	var wg sync.WaitGroup

	s.report = reporter.NewReport()
	semaphore := make(chan struct{}, s.threadNumber)
	defer close(semaphore)

	for i := range s.modulesStruct {
		wg.Add(len(s.modulesStruct[i]))
		for _, m := range s.modulesStruct[i] {
			go func(m *modules.Module) {
				if ctx.Err() != nil {
					outputter("Canceled\n", m)
					return
				}

				localoutputter := func(s any) {
					outputter(s, m)
				}

				defer func() {
					if r := recover(); r != nil {
						switch r.(type) {
						case eval.BreakSignal, eval.ContinueSignal:
							return
						default:
							localoutputter(fmt.Sprintf("Module '%v' error: %v\n", m.Name, r))
							errCh <- errors.NewWithPlace(errWhileExecutingCode, errors.Place(m.Name))
						}
					}
				}()
				m.Output = ""

				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				defer wg.Done()

				startFrom := strings.IndexFunc(m.Content, func(r rune) bool { return r == '\n' })
				l := lexer.NewLexer(s.main + "\n" + m.Content[startFrom:])
				p := parser.NewParser(l)
				prog := p.ParseProgram()
				env := eval.NewEnvironment(database, ctx, s.report, m.Name, localoutputter)
				env.Eval(prog)
			}(m)
		}
		wg.Wait()
	}

	if len(s.report.Content) != 0 {
		return s.report
	}
	return nil
}
