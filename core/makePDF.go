package core

import (
	"clip/utility"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	ID          string
	Description string
	SeverityV40 string
	V40Score    int
	SeverityV31 string
	V31Score    int
	SeverityV30 string
	V30Score    int
	SeverityV2  string
	V2Score     int
	Links       []string
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

	outputListed := utility.EnumLines(output)

	cvesByLine, softByLine := FindCVEs(outputListed)

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
			client := NewNVDClient()
			info, err := client.Fetch(cve)
			if err != nil {
				return
			}

			cveData.Store(cve, info)
		}(key)
	}

	wg.Wait()
	cpeData := make(map[string][]string)
	for soft := range softByLine {
		wg.Add(1)
		sem <- struct{}{}
		prod := "mod_ssl"
		ver := "2.8.4"
		go func(prod, ver string) {
			defer wg.Done()
			defer func() { <-sem }()
			client := NewNVDClient()
			cpeNameList, err := client.FetchCPEName(prod, ver)
			if err != nil {
				return
			}
			cpeData[soft] = cpeNameList
		}(prod, ver)
	}

	wg.Wait()
	cpeResponse := make(map[string][]*CVEInfo)
	for soft, cpeName := range cpeData {
		wg.Add(1)
		sem <- struct{}{}
		prod := "mod_ssl"
		ver := "2.8.4"
		go func(prod, ver string) {
			defer wg.Done()
			defer func() { <-sem }()
			client := NewNVDClient()
			for _, cpe := range cpeName {
				resp, err := client.FetchCVEByCPE(cpe)
				cpeResponse[soft] = append(cpeResponse[soft], resp)
				if err != nil {
					return
				}

			}

		}(prod, ver)
	}

	wg.Wait()

	if len(cvesByLine) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for cve, lines := range cvesByLine {
			dataAny, _ := cveData.Load(cve)
			info := dataAny.(*CVEInfo)
			outputListed = append(outputListed, fmt.Sprintf("\n%s found in lines: %s", cve, lines[:len(lines)-2]))
			outputListed = appendOutput(outputListed, info)

		}
	}

	if len(softByLine) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for cpe, lines := range softByLine {
			info := cpeResponse[cpe]
			if info != nil {
				outputListed = append(outputListed, fmt.Sprintf("%s found in lines: %s\nKnown CVEs related to that:", cpe, lines[:len(lines)-2]))
				for _, cve := range info {
					outputListed = appendOutput(outputListed, cve)
				}
			}
		}
	}
	return strings.Join(outputListed, "\n")
}

func FindCVEs(lines []string) (map[string]string, map[string]string) {
	re := regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)
	resoft := regexp.MustCompile(`(?i)\b([a-z][a-z0-9_\-]+(?:[\s\-_]+[0-9]+)?)\s*(?:[|\s\\\/\-_]+)\s*([a-z0-9][0-9.\-]*[a-z0-9])\b`)

	result := make(map[string]string)
	softVers := make(map[string]string)

	for i, line := range lines {
		found := re.FindAllString(line, -1)
		foundSoft := resoft.FindAllString(line, -1)

		for _, cve := range found {
			result[cve] += strconv.Itoa(i+1) + ", "
		}
		for _, soft := range foundSoft {
			softVers[soft] += strconv.Itoa(i+1) + ", "
		}
	}
	return result, softVers
}

func appendOutput(outputListed []string, cveStruct *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf("Description:\n%s", cveStruct.Description))
	if cveStruct.SeverityV40 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V40 metrics: %s\nV40 Score:%d", cveStruct.SeverityV40, cveStruct.V40Score))
	}
	if cveStruct.SeverityV31 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V31 metrics: %s\nV31 Score:%d", cveStruct.SeverityV31, cveStruct.V31Score))
	}
	if cveStruct.SeverityV30 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V30 metrics: %s\nV30 Score:%d", cveStruct.SeverityV30, cveStruct.V30Score))
	}
	if cveStruct.SeverityV2 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V2 metrics2: %s\nV2 Score%d", cveStruct.SeverityV2, cveStruct.V2Score))
	}
	outputListed = append(outputListed, "Links:")
	outputListed = append(outputListed, cveStruct.Links...)
	return outputListed
}
