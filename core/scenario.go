package core

import (
	"clip/modules"
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

func (s *Scenario) BeginScenario(ctx context.Context, outputter func(string, *modules.Module)) {
	s.execute(ctx, outputter)
}

func (s *Scenario) execute(ctx context.Context, outputter func(string, *modules.Module)) {
	var wg sync.WaitGroup

	stopper := make(chan struct{}, s.ThreadNumber)
	defer close(stopper)

	for i := range s.ModulesStruct {
		for _, m := range s.ModulesStruct[i] {
			wg.Add(1)
			go func(m *modules.Module) {
				m.Output = ""
				localOutputter := func(s string) {
					go outputter(s, m)
				}
				stopper <- struct{}{}
				defer func() { <-stopper }()
				defer wg.Done()

				if utility.IsCanceled(ctx) {
					localOutputter("Canceled\n")
					return
				}
				execution := NewRuntime()
				startFrom := strings.IndexFunc(m.Content, func(r rune) bool { return r == '\n' })
				e := execution.Execute(s.Main+"\n"+m.Content[startFrom+1:], ctx, localOutputter)
				if e != nil {
					localOutputter(fmt.Sprintf("Module '%s' error: %s\n", m.Name, e.Error()))
					return
				}

			}(m)
		}
		wg.Wait()
	}

}
