package reporter

import (
	"clip/errors"
	outputprocessor "clip/fileprocessors/outputProcessor"
	"clip/modules"
	_ "embed"

	"github.com/phpdave11/gofpdf"
)

type pdf struct {
}

func (p *pdf) GetFileType() string {
	return ".pdf"
}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func (p *pdf) CreateReport(
	db outputprocessor.DB,
	makePDFFor []*modules.Module,
	path string,
	progress chan<- float64,
	errChan chan<- error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "", tnrFont)
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "B", tnrbFont)
	pdf.AddPage()
	pdf.SetFont("TimesNewRoman", "", 22)
	pdf.SetTextColor(0, 0, 0)

	length := float64(len(makePDFFor) * 2)

	cache := make(map[string]*outputprocessor.Order)
	software := make([]*outputprocessor.Order, 0)
	cve := make([]*outputprocessor.Order, 0)

	processor := outputprocessor.NewProcessor(cache, software, cve)

	for i, m := range makePDFFor {
		if m.MakeReport.Process && db != nil {
			processor.ProcessOutput(db, m.Output)
		}
		progress <- float64(i+1)/length - 0.01
	}

	processed := processor.PrintResults()

	for i, m := range makePDFFor {
		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, m.Name)
		pdf.Ln(15)

		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		pdf.MultiCell(0, 10, m.Output, "0", "L", false)
		progress <- float64(i+1)/length - 0.01
	}

	pdf.MultiCell(0, 10, processed, "0", "L", false)

	err := pdf.OutputFileAndClose(path)
	if err != nil {
		errChan <- errors.New(errWritingToFile)
	}
	progress <- 1
}
