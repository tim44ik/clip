package core

import (
	"context"
	"fmt"
	"smartpentestutility/shell"
	"smartpentestutility/utility"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/zenity"
	"github.com/phpdave11/gofpdf"

	_ "embed"
)

type Scenario struct {
	Main         string
	ThreadNumber int
	pdfChecker   bool
	profileName  string
	Modules      map[string]string
	Outputs      map[string]string
}

func NewScenario(main string, m map[string]string, thread int, p string, b bool) *Scenario {
	return &Scenario{Main: main, ThreadNumber: thread, pdfChecker: b, profileName: p, Modules: m, Outputs: map[string]string{}}
}

func (s *Scenario) BeginScenario(ctx context.Context) {
	s.execute(ctx)
	if s.pdfChecker {
		s.makePDF()
	}
}

func (s *Scenario) execute(ctx context.Context) {
	type execRespond struct {
		moduleName string
		Output     string
	}
	output := make(chan execRespond, len(s.Modules))
	defer close(output)

	var wg sync.WaitGroup

	stopper := make(chan struct{}, s.ThreadNumber)
	defer close(stopper)

	for moduleName, moduleContent := range s.Modules {
		wg.Add(1)
		go func(module, content string) {
			stopper <- struct{}{}
			defer func() { <-stopper }()
			defer wg.Done()

			if utility.IsCanceled(ctx) {
				output <- execRespond{
					moduleName: module,
					Output:     "Отменено",
				}
				return
			}

			if content == "" {
				output <- execRespond{
					moduleName: module,
					Output:     "",
				}
				return
			}

			execution := shell.NewRuntime()
			e := execution.Execute(s.Main, ctx)
			if e != nil {
				output <- execRespond{
					moduleName: module,
					Output:     fmt.Sprintf("Main module error: %s", e.Error()),
				}
				return
			}
			e = execution.Execute(content, ctx)
			if e != nil {
				output <- execRespond{
					moduleName: module,
					Output:     fmt.Sprintf("Module '%s' error: %s", moduleName, e.Error()),
				}
				return
			}

			output <- execRespond{
				moduleName: module,
				Output:     execution.Output.String(),
			}
		}(moduleName, moduleContent)
	}

	wg.Wait()

channelFlush:
	for {
		select {
		case i := <-output:
			s.Outputs[i.moduleName] = i.Output
		default:
			break channelFlush
		}
	}
	for key, value := range s.Outputs {
		s.Modules[key] = value
	}
}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func (s *Scenario) makePDF() {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "", tnrFont)
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "B", tnrbFont)
	pdf.AddPage()
	pdf.SetFont("TimesNewRoman", "", 22)
	pdf.SetTextColor(0, 0, 0)
	for key, value := range s.Modules {
		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, key)
		pdf.Ln(15)
		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		pdf.MultiCell(0, 10, value, "0", "L", false)
	}

	e := pdf.OutputFileAndClose(strings.TrimSuffix(s.profileName, ".json") + time.Now().Format(" 02.01.2006 104-05") + ".pdf")
	if e != nil {
		zenity.Error("Ошибка формирования PDF: " + e.Error())
	}
}
