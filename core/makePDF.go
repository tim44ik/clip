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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
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

func PDFcreationWindow(Window fyne.Window, Profile string, langmap []string, makePDFFor []*Module) {
	filesaveDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			makePDFFile(Window, Profile, langmap, makePDFFor, writer, err)
		}, Window)
	filesaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	filesaveDialog.Resize(fyne.NewSize(900, 500))
	fyne.Do(func() { filesaveDialog.Show() })
}

func makePDFFile(Window fyne.Window, Profile string, langmap []string, makePDFFor []*Module, writer fyne.URIWriteCloser, err error) {
	if err != nil || writer == nil {
		return
	}

	path := writer.URI().Path()
	if filepath.Ext(path) != ".pdf" {
		defer os.Remove(path)
	}

	path = strings.TrimSuffix(path, filepath.Ext(path))
	path += ".pdf"

	if filepath.Base(path) == ".pdf" {
		path = strings.TrimSuffix(Profile, ".json") + time.Now().Format(" 02.01.2006 15-04-05") + ".pdf"
	}
	PDF(Window, langmap, makePDFFor, path)
}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func PDF(Window fyne.Window, langmap []string, makePDFFor []*Module, path string) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "", tnrFont)
	pdf.AddUTF8FontFromBytes("TimesNewRoman", "B", tnrbFont)
	pdf.AddPage()
	pdf.SetFont("TimesNewRoman", "", 22)
	pdf.SetTextColor(0, 0, 0)
	for _, m := range makePDFFor {
		pdf.SetFontSize(22)
		pdf.SetFontStyle("B")
		pdf.Cell(0, 10, m.Name)
		pdf.Ln(15)
		pdf.SetFontSize(14)
		pdf.SetFontStyle("")
		if m.MakePDF.Process {
			processedString := processOutput(m.output)
			pdf.MultiCell(0, 10, processedString, "0", "L", false)
			time.Sleep(30 * time.Second)
		} else {
			enumed := utility.EnumLines(m.output)
			pdf.MultiCell(0, 10, strings.Join(enumed, "\n"), "0", "L", false)
		}

	}
	e := pdf.OutputFileAndClose(path)
	if e != nil {
		dialog.ShowError(fmt.Errorf("%s:\n%s", langmap[28], e), Window)
	}
}

func processOutput(output string) string {

	client := NewNVDClient()

	outputListed := utility.EnumLines(output)

	softByLine, cvesByLine := FindCVEs(outputListed)

	maxGoroutines := 5
	sem := make(chan struct{}, maxGoroutines)
	var wg sync.WaitGroup
	var mu sync.Mutex

	cpeNameData := make(map[string][]string, len(softByLine))
	for soft := range softByLine {
		wg.Add(1)
		sem <- struct{}{}
		go func(prod string) {
			defer wg.Done()
			defer func() { <-sem }()
			cpeNameList, err := client.FetchCPEName(prod)
			if err != nil {
				return
			}
			mu.Lock()
			cpeNameData[soft] = cpeNameList
			mu.Unlock()
			client.http.CloseIdleConnections()
		}(soft)
	}

	wg.Wait()

	cpeData := make(map[string][]*CVEInfo, len(softByLine))
	for soft, cpeName := range cpeNameData {
		wg.Add(1)
		sem <- struct{}{}
		go func(cpeName []string) {
			defer wg.Done()
			defer func() { <-sem }()
			var respSlice []*CVEInfo
			for _, cpe := range cpeName {
				resp, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cpeName=", cpe)
				if err != nil {
					return
				}
				respSlice = append(respSlice, resp)
			}
			mu.Lock()
			cpeData[soft] = respSlice
			mu.Unlock()
			client.http.CloseIdleConnections()
		}(cpeName)
	}

	wg.Wait()

	cveData := make(map[string]*CVEInfo)

	for key := range cvesByLine {
		wg.Add(1)
		sem <- struct{}{}
		go func(cve string) {
			defer wg.Done()
			defer func() { <-sem }()
			info, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cveId=", cve)
			if err != nil {
				return
			}
			mu.Lock()
			cveData[cve] = info
			mu.Unlock()
			client.http.CloseIdleConnections()
		}(key)
	}

	wg.Wait()

	if len(cpeData) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for cpe, lines := range softByLine {
			if cpeData[cpe] != nil {
				outputListed = append(outputListed, fmt.Sprintf("\n%s\n    found in lines: %s\n    Known CVEs related to that:", cpe, lines[:len(lines)-2]))
				for _, cve := range cpeData[cpe] {
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
			outputListed = append(outputListed, fmt.Sprintf("\n%s\nFound in lines: %s", cve, cvesByLine[cve][:len(cvesByLine[cve])-2]))
			outputListed = appendOutput(outputListed, info)
		}
	}

	return strings.Join(outputListed, "\n")
}

func FindCVEs(lines []string) (software map[string]string, cve map[string]string) {
	reCve := regexp.MustCompile(`CVE-\d{4}-\d{4,}`)
	reSoft := regexp.MustCompile(`(?:CVE\-\d{4}\-\d{4,})|(?:[Pp]ort[|\s:\\/]*\d{1,5})|(?:[Pp]ing[\s:]+\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(?:\:\d{1,5})*)|([\w\-]+(?:\s\d+)?)(?:\s*(?:[:\/\s\-\|]+|ver\.|v\.|version(?:\s*(?:[\\/:|]*)\s*))\s*)((?:(?:[\w\d]+(?:\.[\w\d]+)+(?:-[\w\d](?:\.[\w\d])+)?)|\d+H\d+|\d+|j[gk]\d+)(?:[\-\\\/]*(?:dev|beta|alpha)?))`)

	software = make(map[string]string)
	cve = make(map[string]string)

	for i, line := range lines {
		foundCve := reCve.FindAllString(line, -1)
		foundSoft := reSoft.FindAllStringSubmatch(line, -1)

		for _, f := range foundCve {
			cve[f] = strconv.Itoa(i+1) + ", "
		}

		for _, f := range foundSoft {
			f[1] = strings.ToLower(f[1])
			f[2] = strings.ToLower(f[2])
			if (f[1] != "" && f[1] != "version" && f[1] != "ver." && len(f[1]) > 2) && f[2] != "" {
				software[strings.ReplaceAll(f[1]+":"+f[2], " ", "_")] = strconv.Itoa(i+1) + ", "
			}

		}
	}

	return software, cve
}

func appendOutput(outputListed []string, cveStruct *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf("    \nDescription: %s\n", strings.ToLower(cveStruct.Description[:1])+cveStruct.Description[1:]))
	if cveStruct.SeverityV40 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V40 metrics: %s", cveStruct.SeverityV40))
	}
	if cveStruct.SeverityV31 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V31 metrics: %s", cveStruct.SeverityV31))
	}
	if cveStruct.SeverityV30 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V30 metrics: %s", cveStruct.SeverityV30))
	}
	if cveStruct.SeverityV2 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V2 metrics: %s", cveStruct.SeverityV2))
	}
	outputListed = append(outputListed, "\n    Links:")
	outputListed = append(outputListed, cveStruct.Links...)
	return outputListed
}
