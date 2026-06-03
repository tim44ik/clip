package scenario_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"clip/engine/scenario"
	"clip/models/modules"
	"clip/utility"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func collectErrors(errCh chan error) []error {
	var errs []error
	for {
		select {
		case e, ok := <-errCh:
			if !ok {
				return errs
			}
			errs = append(errs, e)
		default:
			return errs
		}
	}
}

func containsOutput(outputs []string, substr string) bool {
	for _, out := range outputs {
		if strings.Contains(out, substr) {
			return true
		}
	}
	return false
}

func TestScenario_Execute(t *testing.T) {

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 10)
	ctx := context.Background()

	var outputs []string
	outputter := func(s any, m *modules.Module) {
		outputs = append(outputs, fmt.Sprintf("%v", s))
		fmt.Println(s)
	}

	t.Run("break inside loop", func(t *testing.T) {
		outputs = nil
		code := `
		queue(1)
		%s = "hi"
		for %i = 0; %i < 10; %i = %i + 1 do
			if %i == 5 then
				break
			end
			print(%i)
		end
		%p = process("NVD", %s)
		%r = report(".pdf", %s)
		%r = report(".pdf", %s)`
		mod := modules.CreateModule("break_inside", code)
		q, err := utility.GetQueue([]*modules.Module{mod})
		if err != nil {
			t.Errorf("unexpected error while making queue: %v", err)
		}
		scenario := scenario.NewScenario("", 1, q)
		report := scenario.Execute(db, errCh, ctx, outputter)
		if report == nil {
			t.Error("report is nil")
		} else {
			report.Reporter.CreateReport("C:\\Users\\w\\Desktop\\1.pdf", report.Content, errCh)
		}

		errs := collectErrors(errCh)
		if len(errs) > 0 {
			t.Errorf("unexpected errors for break inside: %v", errs)
		}
		expected := []string{"0", "1", "2", "3", "4"}
		for _, exp := range expected {
			if !containsOutput(outputs, exp) {
				t.Errorf("missing output %s, got %v", exp, outputs)
			}
		}
		if containsOutput(outputs, "5") {
			t.Errorf("should not print 5 after break")
		}
	})

	t.Run("panic test", func(t *testing.T) {
		defer func() {
			e := <-errCh
			t.Logf("Captured error: %v", e)
			if !strings.Contains(e.Error(), "break outside") {
				t.Fatalf("expected error for break outside loop %v", e)
			}
		}()
		outputs = nil
		code := `
		for %i = 0; %i < 10; %i = %i + 1 do
			if %i == 5 then
				break
			end
			print(%i)
		end
		break`
		mod := modules.CreateModule("break_outside", code)
		scenario := scenario.NewScenario("", 1, [][]*modules.Module{{mod}})
		scenario.Execute(db, errCh, ctx, outputter)
	})

	t.Run("runtime panic", func(t *testing.T) {
		defer func() {
			e := <-errCh
			t.Logf("Captured error: %v", e)
			if !strings.Contains(e.Error(), "division by 0") {
				t.Fatalf("expected error for division by 0 %v", e)
			}
		}()
		outputs = nil
		mod := modules.CreateModule("panic", `%x = 1 / 0`)
		scenario := scenario.NewScenario("", 1, [][]*modules.Module{{mod}})
		scenario.Execute(db, errCh, ctx, outputter)
	})

	t.Run("context cancellation", func(t *testing.T) {
		defer func() {
			e := <-errCh
			t.Logf("Captured error: %v", e)
			if !strings.Contains(e.Error(), "context error") {
				t.Fatalf("expected error for context error %v", e)
			}
		}()
		outputs = nil
		code := `for %i = 0; %i < 1000000; %i = %i + 1 do
		            print(%i)
		        end`
		mod := modules.CreateModule("cancel", code)
		scenario := scenario.NewScenario("", 1, [][]*modules.Module{{mod}})
		cancelCtx, cancel := context.WithCancel(ctx)
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		scenario.Execute(db, errCh, cancelCtx, outputter)

	})
}
