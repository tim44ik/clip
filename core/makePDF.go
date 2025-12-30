package core

import (
	"clip/utility"
	"context"
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
	"fyne.io/fyne/v2/widget"
	"github.com/phpdave11/gofpdf"
)

type order struct {
	name  string
	lines []int
	cpe   string
	cve   string
}

type CVEInfo struct {
	ID          string
	Description string
	SeverityV40 string
	SeverityV31 string
	SeverityV30 string
	SeverityV2  string
	Links       []string
}

func PDFcreationWindow(a *SpuWindow, makePDFFor []*Module, ctx context.Context) {
	filesaveDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			go makePDFFile(a, makePDFFor, writer, err, ctx)
		}, a.Window)
	filesaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	filesaveDialog.Resize(fyne.NewSize(900, 500))
	fyne.Do(func() { filesaveDialog.Show() })
}

func makePDFFile(a *SpuWindow, makePDFFor []*Module, writer fyne.URIWriteCloser, err error, ctx context.Context) {
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
		path = strings.TrimSuffix(a.profiles.path, ".json") + time.Now().Format(" 02.01.2006 15-04-05") + ".pdf"
	}
	PDF(a, makePDFFor, ctx, path)
}

//go:embed TimesNewRoman.ttf
var tnrFont []byte

//go:embed TimesNewRomanB.ttf
var tnrbFont []byte

func PDF(a *SpuWindow, makePDFFor []*Module, ctx context.Context, path string) {
	progressBar := widget.NewProgressBar()
	progressWindow := dialog.NewCustomWithoutButtons("Creating PDF", progressBar, a.Window)
	fyne.Do(func() { progressWindow.Show() })
	go func() {
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
			if m.MakePDF.Process {
				processedString := processOutput(ctx, m.output)
				pdf.MultiCell(0, 10, processedString, "0", "L", false)
			} else {
				enumed := utility.EnumLines(m.output)
				pdf.MultiCell(0, 10, strings.Join(enumed, "\n"), "0", "L", false)
			}
			fyne.Do(func() {
				progressBar.SetValue(float64(i+1) / length)
				progressBar.Refresh()
				progressWindow.Refresh()
			})
			fmt.Print(float64(i+1) / length)
		}
		e := pdf.OutputFileAndClose(path)
		if e != nil {
			dialog.ShowError(fmt.Errorf("%s:\n%s", a.langmap[a.Modules.CurrentLang][28], e), a.Window)
		}
		fyne.Do(func() { progressWindow.Hide() })
	}()
}

func processOutput(ctx context.Context, output string) string {

	client := NewNVDClient(ctx)
	outputListed := utility.EnumLines(output)
	softByLine, cvesByLine := FindCVEs(outputListed)

	maxGoroutines := 5
	sem := make(chan struct{}, maxGoroutines)
	var wg sync.WaitGroup
	var mu sync.Mutex

	cpeNameData := make(map[string][]string, len(softByLine))
	for _, soft := range softByLine {
		wg.Add(1)
		sem <- struct{}{}
		go func(prod string) {
			defer wg.Done()
			defer func() { <-sem }()
			limiter.Wait(ctx)
			cpeNameList, err := client.FetchCPEName(prod, ctx)
			if err != nil {
				return
			}
			mu.Lock()
			cpeNameData[soft.name] = cpeNameList
			mu.Unlock()
			client.http.CloseIdleConnections()
		}(soft.name)
	}

	wg.Wait()

	cpeData := make(map[string][]*CVEInfo, len(softByLine))
	for soft, cpeName := range cpeNameData {
		wg.Add(1)
		sem <- struct{}{}
		go func(cpeName []string) {
			defer wg.Done()
			defer func() { <-sem }()
			limiter.Wait(ctx)
			for _, cpe := range cpeName {
				resp, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cpeName=", cpe, ctx)
				if err != nil {
					return
				}
				mu.Lock()
				seen := false
				for _, st := range cpeData[soft] {
					if st.ID == resp.ID {
						seen = true
						break
					}
				}
				if !seen {
					cpeData[soft] = append(cpeData[soft], resp)
				}
				mu.Unlock()
			}
			client.http.CloseIdleConnections()
		}(cpeName)
	}

	wg.Wait()

	cveData := make(map[string]*CVEInfo)

	for _, cves := range cvesByLine {
		wg.Add(1)
		sem <- struct{}{}
		go func(cve string) {
			defer wg.Done()
			defer func() { <-sem }()
			limiter.Wait(ctx)
			info, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cveId=", cve, ctx)
			if err != nil {
				return
			}
			mu.Lock()
			cveData[cve] = info
			mu.Unlock()
			client.http.CloseIdleConnections()
		}(cves.name)
	}

	wg.Wait()

	if len(cpeData) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for _, soft := range softByLine {
			if cpeData[soft.name] != nil {
				outputListed = append(outputListed, fmt.Sprintf("\n%s\n    Found in lines: %s\n    Known CVEs related to that:", soft.name, func(i []int) string {
					str := ""
					for _, i := range soft.lines {
						str += strconv.Itoa(i) + ", "
					}
					return str[:len(str)-2]
				}(soft.lines)))
				for _, cve := range cpeData[soft.name] {
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
		for _, cves := range cvesByLine {
			if cveData[cves.name] != nil {
				outputListed = append(outputListed, fmt.Sprintf("\n%s\nFound in lines: %s", cves.name, func(i []int) string {
					str := ""
					for _, i := range cves.lines {
						str += strconv.Itoa(i) + ", "
					}
					return str[:len(str)-2]
				}(cves.lines)))
				outputListed = appendOutput(outputListed, cveData[cves.name])
			}
		}
	}

	return strings.Join(outputListed, "\n")
}

func FindCVEs(lines []string) ([]order, []order) {
	reCve := regexp.MustCompile(`CVE-\d{4}-\d{4,}`)
	reSoft := regexp.MustCompile(`(?i)(?:CVE\-\d{4}\-\d{4,})|(?:[Pp]ort[|\s:\\/]*\d{1,5})|(?:[Pp]ing[\s:]+\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(?:\:\d{1,5})*)|([\w\-]+(?:\s\d+)?)(?:\s*(?:[:\\\/\s\-\|]+|ver\.|v\.|version(?:\s*(?:[\\/:|]*)\s*))\s*)((?:(?:\d+(?:\.\d+[a-z]*\d*)+(?:-\d(?:\.\d)+)?)|\d+[a-z]?\d*)(?:[\-\\\/]?dev|[\-\\\/]?beta|[\-\\\/]?alpha)?)`)
	software := []order{}
	cve := []order{}
	for i, line := range lines {
		foundCve := reCve.FindAllString(line, -1)
		foundSoft := reSoft.FindAllStringSubmatch(line, -1)

		for _, f := range foundCve {
			seen := false
			for _, s := range cve {
				if s.name == f {
					s.lines = append(s.lines, i+1)
					seen = true
					break
				}
			}
			if !seen {
				cve = append(cve, order{name: f, lines: []int{i + 1}})
			}
		}

		for _, f := range foundSoft {
			f[1] = strings.ToLower(f[1])
			f[2] = strings.ToLower(f[2])
			if (f[1] != "" && f[1] != "version" && f[1] != "ver." && len(f[1]) > 2) && f[2] != "" {
				seen := false
				name := strings.ReplaceAll(f[1]+":"+f[2], " ", "_")
				for _, s := range software {
					if s.name == name {
						s.lines = append(s.lines, i+1)
						seen = true
						break
					}
				}
				if !seen {
					software = append(software, order{name: name, lines: []int{i + 1}})
				}
			}

		}
	}
	return software, cve
}

func appendOutput(outputListed []string, cveStruct *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf("    \nDescription: %s\n", cveStruct.Description))
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
