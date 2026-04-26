package scenario

import (
	"clip/modules"
	r "clip/runtime"
	"clip/utility"
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
}

func NewScenario(main string, thread int, module [][]*modules.Module) *Scenario {
	return &Scenario{Main: main, ThreadNumber: thread, ModulesStruct: module}
}

func (s *Scenario) Execute(ctx context.Context, outputter func(string, *modules.Module)) {
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, s.ThreadNumber)
	defer close(semaphore)

	for i := range s.ModulesStruct {
		wg.Add(len(s.ModulesStruct[i]))
		for _, m := range s.ModulesStruct[i] {
			go func(m *modules.Module) {
				m.Output = ""

				localOutputter := func(s string) {
					go outputter(s, m)
				}

				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				defer wg.Done()

				if utility.IsCanceled(ctx) {
					localOutputter("Canceled\n")
					return
				}

				execution := r.NewRuntime()

				startFrom := strings.IndexFunc(m.Content,
					func(r rune) bool { return r == '\n' })

				err := execution.Execute(s.Main+"\n"+m.Content[startFrom+1:],
					ctx,
					localOutputter)
				if err != nil {
					localOutputter(fmt.Sprintf("Module '%s' error: %s\n", m.Name, err.Error()))
					return
				}

			}(m)
		}
		wg.Wait()
	}
}
