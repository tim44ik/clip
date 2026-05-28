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
)

type Scenario struct {
	Main          string
	ThreadNumber  int
	ModulesStruct [][]*modules.Module
	report        *reporter.Report
}

func NewScenario(main string, thread int, module [][]*modules.Module) *Scenario {
	return &Scenario{Main: main, ThreadNumber: thread, ModulesStruct: module}
}

func (s *Scenario) Execute(errCh chan<- error, ctx context.Context, outputter func(string, *modules.Module)) *reporter.Report {
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, s.ThreadNumber)
	defer close(semaphore)

	for i := range s.ModulesStruct {
		wg.Add(len(s.ModulesStruct[i]))
		for _, m := range s.ModulesStruct[i] {
			go func(m *modules.Module) {
				if ctx.Err() != nil {
					outputter("Canceled\n", m)
					return
				}

				localoutputter := func(s string) {
					outputter(s, m)
				}

				defer func() {
					if r := recover(); r != nil {
						switch r.(type) {
						case eval.BreakSignal, eval.ContinueSignal:
							return
						default:
							localoutputter(fmt.Sprintf("Module '%s' error: %v\n", m.Name, r))
							errCh <- errors.NewWithPlace(errWhileExecutingCode, errors.Place(m.Name))
						}
					}
				}()
				m.Output = ""

				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				defer wg.Done()

				startFrom := strings.IndexFunc(m.Content, func(r rune) bool { return r == '\n' })

				l := lexer.NewLexer(s.Main + "\n" + m.Content[startFrom+1:])
				p := parser.NewParser(l)
				prog := p.ParseProgram()
				env := eval.NewEnvironment(ctx, s.report, m.Name, localoutputter)
				env.Eval(prog)
			}(m)
		}
		wg.Wait()
	}

	if s.report != nil {
		return s.report
	}
	return nil
}
