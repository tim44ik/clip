package reporter

import (
	"clip/errors"
	_ "embed"

	"github.com/phpdave11/gofpdf"
)

type pdf struct{}

func (p *pdf) GetFileType() string {
	return ".pdf"
}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func (p *pdf) CreateReport(
	path string,
	content []*ReportContent,
	errChan chan<- error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "", tnrFont)
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "B", tnrbFont)
	pdf.AddPage()
	pdf.SetFont("TimesNewRoman", "", 22)
	pdf.SetTextColor(0, 0, 0)

	for _, c := range content {
		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, c.Mname)
		pdf.Ln(15)

		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		pdf.MultiCell(0, 10, c.Body, "0", "L", false)
	}

	err := pdf.OutputFileAndClose(path)
	if err != nil {
		errChan <- errors.New(errWritingToFile)
	}
}
