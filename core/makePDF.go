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
	SeverityV31 string
	SeverityV30 string
	SeverityV2  string
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

	softByLine, cvesByLine := FindCVEs(outputListed)

	maxGoroutines := 2
	sem := make(chan struct{}, maxGoroutines)

	// var wg sync.WaitGroup
	cveData := make(map[string]*CVEInfo)

	for key := range cvesByLine {
		// wg.Add(1)
		sem <- struct{}{}

		func(cve string) {
			// defer wg.Done()
			defer func() { <-sem }()
			client := NewNVDClient()
			info, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cveId=", cve)
			if err != nil {
				return
			}
			cveData[cve] = info
			client.http.CloseIdleConnections()
		}(key)
	}

	// wg.Wait()

	cpeNameData := make(map[string][]string, len(softByLine))
	for soft := range softByLine {
		// wg.Add(1)
		sem <- struct{}{}
		func(prod string) {
			// defer wg.Done()
			defer func() { <-sem }()
			client := NewNVDClient()
			cpeNameList, err := client.FetchCPEName(prod)
			if err != nil {
				return
			}
			cpeNameData[soft] = cpeNameList
			client.http.CloseIdleConnections()
		}(soft)
	}

	// wg.Wait()

	cpeData := make(map[string][]*CVEInfo, len(softByLine))
	for soft, cpeName := range cpeNameData {
		// wg.Add(1)
		sem <- struct{}{}
		func(cpeName []string) {
			// defer wg.Done()
			defer func() { <-sem }()
			client := NewNVDClient()
			var respSlice []*CVEInfo
			for _, cpe := range cpeName {
				resp, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cpeName=", cpe)
				if err != nil {
					return
				}
				respSlice = append(respSlice, resp)
			}
			cpeData[soft] = respSlice
			client.http.CloseIdleConnections()
		}(cpeName)
	}

	// wg.Wait()

	if len(cpeData) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for cpe, info := range cpeData {
			if cpeData[cpe] != nil {
				outputListed = append(outputListed, fmt.Sprintf("\n%s\nfound in lines: %s\nKnown CVEs related to that:", cpe, softByLine[cpe][:len(softByLine[cpe])-2]))
				for _, cve := range info {
					outputListed = append(outputListed, cve.ID)
					outputListed = appendOutput(outputListed, cve)
				}
			}
		}
	}

	if len(cveData) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for cve, info := range cveData {
			outputListed = append(outputListed, fmt.Sprintf("\n%s\nfound in lines: %s", cve, cvesByLine[cve][:len(cvesByLine[cve])-2]))
			outputListed = appendOutput(outputListed, info)
		}
	}

	return strings.Join(outputListed, "\n")
}

func FindCVEs(lines []string) (software map[string]string, cve map[string]string) {
	reCve := regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)
	reSoft := regexp.MustCompile(`([\w\-]+(?:\s\d+)?)(?:\s*(?:[:\/\s\-\|]+|ver\.|v\.|version(?:\s*(?:[\\/:|]*)\s*))\s*)((?:(?:[\w\d]+(?:\.[\w\d]+)+(?:-[\w\d](?:\.[\w\d])+)?)|\d+H\d+|\d+|j[gk]\d+)(?:[-\\\/]*(?:dev|beta|alpha)?))`)

	software = make(map[string]string)
	cve = make(map[string]string)

	for i, line := range lines {
		foundCve := reCve.FindAllString(line, -1)
		foundSoft := reSoft.FindAllStringSubmatch(line, -1)

		for _, f := range foundCve {
			cve[f] = strconv.Itoa(i+1) + ", "
		}

		for _, f := range foundSoft {

			software[strings.ReplaceAll(f[1]+":"+f[2], " ", "_")] = strconv.Itoa(i+1) + ", "
		}
	}

	return software, cve
}

func appendOutput(outputListed []string, cveStruct *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf("Description: %s\n", strings.ToLower(cveStruct.Description[:1])+cveStruct.Description[1:]))
	if cveStruct.SeverityV40 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V40 metrics: %s", cveStruct.SeverityV40))
	}
	if cveStruct.SeverityV31 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V31 metrics: %s", cveStruct.SeverityV31))
	}
	if cveStruct.SeverityV30 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V30 metrics: %s", cveStruct.SeverityV30))
	}
	if cveStruct.SeverityV2 != "" {
		outputListed = append(outputListed, fmt.Sprintf("Severity calculated with V2 metrics: %s", cveStruct.SeverityV2))
	}
	outputListed = append(outputListed, "\nLinks:")
	outputListed = append(outputListed, cveStruct.Links...)
	return outputListed
}
