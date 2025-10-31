package core

import (
	"context"
	"fmt"
	"smartpentestutility/utility"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/zenity"
	"github.com/phpdave11/gofpdf"

	_ "embed"
)

type Scenario struct {
	Main          string
	ThreadNumber  int
	pdfChecker    bool
	profileName   string
	Outputs       map[string]string
	ModulesStruct []*Module
}

func NewScenario(main string, thread int, p string, b bool, module []*Module) *Scenario {
	return &Scenario{Main: main, ThreadNumber: thread, pdfChecker: b, profileName: p, Outputs: map[string]string{}, ModulesStruct: module}
}

func (s *Scenario) BeginScenario(ctx context.Context, outputter func(string, *Module)) {
	s.execute(ctx, outputter)
	if s.pdfChecker {
		s.makePDF()
	}
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

			execution := NewRuntime(m)
			e := execution.Execute(s.Main, ctx, localOutputter)
			if e != nil {
				localOutputter(fmt.Sprintf("Main module error: %s\n", e.Error()))
				return
			}
			e = execution.Execute(m.Content, ctx, localOutputter)
			if e != nil {
				localOutputter(fmt.Sprintf("Module '%s' error: %s\n", m.Name, e.Error()))
				return
			}

		}(m)
	}

	wg.Wait()
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
	for _, m := range s.ModulesStruct {
		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, m.Name)
		pdf.Ln(15)
		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		pdf.MultiCell(0, 10, m.Output, "0", "L", false)
	}

	e := pdf.OutputFileAndClose(strings.TrimSuffix(s.profileName, ".json") + time.Now().Format(" 02.01.2006 15-04-05") + ".pdf")
	if e != nil {
		zenity.Error("Ошибка формирования PDF: " + e.Error())
	}
}
