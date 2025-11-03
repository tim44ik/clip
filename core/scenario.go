package core

import (
	"clip/utility"
	"context"
	"fmt"
	"sync"

	_ "embed"
)

type Scenario struct {
	Main          string
	ThreadNumber  int
	ModulesStruct []*Module
}

func NewScenario(main string, thread int, module []*Module) *Scenario {
	return &Scenario{Main: main, ThreadNumber: thread, ModulesStruct: module}
}

func (s *Scenario) BeginScenario(ctx context.Context, outputter func(string, *Module)) {
	s.execute(ctx, outputter)
}

func (s *Scenario) execute(ctx context.Context, outputter func(string, *Module)) {
	var wg sync.WaitGroup

	stopper := make(chan struct{}, s.ThreadNumber)
	defer close(stopper)

	for _, m := range s.ModulesStruct {
		wg.Add(1)
		go func(m *Module) {
			m.Output = ""
			localOutputter := func(s string) {
				go outputter(s, m)
			}
			stopper <- struct{}{}
			defer func() { <-stopper }()
			defer wg.Done()

			if utility.IsCanceled(ctx) {
				localOutputter("Отменено\n")
				return
			}
			m.Output = ""
			execution := NewRuntime()
			e := execution.Execute(s.Main+"\n"+m.Content, ctx, localOutputter)
			if e != nil {
				localOutputter(fmt.Sprintf("Module '%s' error: %s\n", m.Name, e.Error()))
				return
			}

		}(m)
	}

	wg.Wait()
}
