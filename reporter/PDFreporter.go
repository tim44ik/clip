package reporter

import (
	"clip/errors"
	"clip/modules"
	outputprocessor "clip/outputProcessor"
	"clip/utility"
	_ "embed"
	"strings"

	"github.com/phpdave11/gofpdf"
)

type PDF struct {
}

func NewPDF() *PDF {
	return &PDF{}
}

func (p *PDF) GetFileType() string {
	return ".pdf"
}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func (p *PDF) CreateReport(
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

	length := float64(len(makePDFFor))

	for i, m := range makePDFFor {

		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, m.Name)
		pdf.Ln(15)

		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		if m.MakeReport.Process && db != nil {
			processedString := outputprocessor.ProcessOutput(db, m.Output)
			pdf.MultiCell(0, 10, processedString, "0", "L", false)
		} else {
			enumed := utility.EnumLines(m.Output)
			pdf.MultiCell(0, 10, strings.Join(enumed, "\n"), "0", "L", false)
		}
		progress <- float64(i+1)/length - 0.01
	}

	err := pdf.OutputFileAndClose(path)
	if err != nil {
		errChan <- errors.New(errWritingToFile)
	}
	progress <- 1
}
