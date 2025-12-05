package core

import (
	"clip/utility"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/phpdave11/gofpdf"
)

type CVEInfo struct {
	Severity string
	Links    []string
}

func PDFcreationWindow(a *SpuWindow) {
	addmoduleDialog := dialog.NewCustomConfirm(
		a.langmap[a.Modules.CurrentLang][35],
		a.langmap[a.Modules.CurrentLang][23],
		a.langmap[a.Modules.CurrentLang][24],
		container.NewPadded(
			container.NewBorder(widget.NewCheck(a.langmap[a.Modules.CurrentLang][34], func(b bool) { a.makePDF.process = b }),
				nil, nil, nil)),
		func(b bool) {
			if b {
				filesaveDialog := dialog.NewFileSave(
					func(writer fyne.URIWriteCloser, err error) {
						makePDFFile(a, writer, err)
					}, a.Window)
				filesaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
				filesaveDialog.Resize(fyne.NewSize(900, 500))
				filesaveDialog.Show()
			}
		}, a.Window)
	addmoduleDialog.Resize(fyne.NewSize(300, 200))
	addmoduleDialog.Show()
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
		if a.makePDF.process {
			processedString := processOutput(m.Output)
			pdf.MultiCell(0, 10, processedString, "0", "L", false)
		} else {
			enumed := utility.EnumLines(m.Output)
			pdf.MultiCell(0, 10, strings.Join(enumed, "\n"), "0", "L", false)
		}

	}
	e := pdf.OutputFileAndClose(a.makePDF.pdfPath)
	if e != nil {
		dialog.ShowError(fmt.Errorf("%s:\n%s", a.langmap[a.Modules.CurrentLang][28], e), a.Window)
	}
}

func processOutput(output string) string {

	client := NewNVDClient()

	outputListed := utility.EnumLines(output)

	cvesByLine := FindCVEs(outputListed)

	maxGoroutines := 10
	sem := make(chan struct{}, maxGoroutines)

	var wg sync.WaitGroup
	cveData := sync.Map{}

	for key := range cvesByLine {
		wg.Add(1)
		sem <- struct{}{}

		go func(cve string) {
			defer wg.Done()
			defer func() { <-sem }()

			info, err := client.Fetch(cve)
			if err != nil {
				info = &CVEInfo{Severity: "UNKNOWN", Links: []string{}}
			}

			cveData.Store(cve, info)
		}(key)
	}

	wg.Wait()
	if len(cvesByLine) != 0 {
		outputListed = append(outputListed, "\nProcessing results:")
		for cve, lines := range cvesByLine {
			dataAny, _ := cveData.Load(cve)
			info := dataAny.(*CVEInfo)

			outputListed = append(outputListed, fmt.Sprintf("\n%s found in lines: %s", cve, lines[:len(lines)-2]))
			outputListed = append(outputListed, fmt.Sprintf("Severity: %s\n", info.Severity))
			outputListed = append(outputListed, "Links:")
			outputListed = append(outputListed, info.Links...)
		}
	}
	return strings.Join(outputListed, "\n")
}

func FindCVEs(lines []string) map[string]string {
	re := regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)

	result := make(map[string]string)

	for i, line := range lines {
		found := re.FindAllString(line, -1)

		for _, cve := range found {
			result[cve] += strconv.Itoa(i+1) + ", "
		}
	}
	return result
}
