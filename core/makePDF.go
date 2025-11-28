package core

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"github.com/phpdave11/gofpdf"
)

func PDFcreationWindow(a *SpuWindow) {
	filesavedialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			makePDFFile(a, writer, err)
		}, a.Window)
	filesavedialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	filesavedialog.Resize(fyne.NewSize(900, 500))
	filesavedialog.Show()
}

func makePDFFile(a *SpuWindow, writer fyne.URIWriteCloser, err error) {
	if err != nil || writer == nil {
		return
	}

	path := writer.URI().Path()
	if filepath.Ext(path) != ".pdf" {
		defer os.Remove(path)
	}

	a.makePDF.pdfPath = path
	a.makePDF.pdfPath = strings.TrimSuffix(a.makePDF.pdfPath, filepath.Ext(a.makePDF.pdfPath))
	a.makePDF.pdfPath += ".pdf"

	if filepath.Base(a.makePDF.pdfPath) == ".pdf" {
		a.makePDF.pdfPath = strings.TrimSuffix(a.Profiles.Path, ".json") + time.Now().Format(" 02.01.2006 15-04-05") + a.makePDF.pdfPath
	}
	PDF(a)

}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func PDF(a *SpuWindow) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "", tnrFont)
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "B", tnrbFont)
	pdf.AddPage()
	pdf.SetFont("TimesNewRoman", "", 22)
	pdf.SetTextColor(0, 0, 0)
	for _, m := range a.Modules.ChildModules {
		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, m.Name)
		pdf.Ln(15)
		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		pdf.MultiCell(0, 10, m.Output, "0", "L", false)
	}
	e := pdf.OutputFileAndClose(a.makePDF.pdfPath)
	if e != nil {
		dialog.ShowError(fmt.Errorf("%s:\n%s", a.langmap[a.Modules.CurrentLang][28], e), a.Window)
	}
}
