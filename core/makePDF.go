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

type Order struct {
	name  []string
	lines []int
	cpe   []string
	cve   []*CVEInfo
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

func PDFcreationWindow(a *ClipWindow, makePDFFor []*Module, ctx context.Context) {
	filesaveDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			go makePDFFile(a, makePDFFor, writer, err, ctx)
		}, a.Window)
	filesaveDialog.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
	filesaveDialog.Resize(fyne.NewSize(900, 500))
	fyne.Do(func() { filesaveDialog.Show() })
}

func makePDFFile(a *ClipWindow, makePDFFor []*Module, writer fyne.URIWriteCloser, err error, ctx context.Context) {
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

func PDF(a *ClipWindow, makePDFFor []*Module, ctx context.Context, path string) {
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
			fyne.DoAndWait(func() {
				progressBar.SetValue(float64(i+1) / length)
			})
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

	maxGoroutines := 50
	sem := make(chan struct{}, maxGoroutines)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, soft := range softByLine {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			prod := strings.ToLower(strings.ReplaceAll(soft.name[1], " ", "_") + ":" + soft.name[2])
			var err error
			soft.cpe, err = client.FetchCPEName(prod, ctx)
			if err != nil {
				soft.cpe = nil
				return
			}
			client.http.CloseIdleConnections()
		}()
	}

	wg.Wait()

	for _, soft := range softByLine {

		if soft.cpe == nil {
			continue
		}
		seenCVEs := make(map[string]*CVEInfo)
		for _, cpe := range soft.cpe {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				resp, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cpeName=", cpe, ctx)
				if err != nil {
					return
				}
				for _, cve := range resp {
					if seenCVEs[cve.ID] == nil {
						mu.Lock()
						seenCVEs[cve.ID] = cve
						mu.Unlock()
					}
				}
				client.http.CloseIdleConnections()
			}()
		}
		wg.Wait()
		for _, value := range seenCVEs {
			soft.cve = append(soft.cve, value)
		}
	}

	for _, cve := range cvesByLine {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			limiter.Wait(ctx)
			info, err := client.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cveId=", cve.name[0], ctx)
			if err != nil {
				return
			}
			cve.cve = append(cve.cve, info...)
			client.http.CloseIdleConnections()
		}()
	}

	wg.Wait()

	if len(softByLine) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for _, soft := range softByLine {
			if soft.cve != nil {
				outputListed = append(outputListed, fmt.Sprintf("\n%s\n    Found in lines: %s\n    Known CVE(s) related to that:", soft.name[0], func(i []int) string {
					str := ""
					for _, i := range soft.lines {
						str += strconv.Itoa(i) + ", "
					}
					return str[:len(str)-2]
				}(soft.lines)))
				for _, cve := range soft.cve {
					outputListed = append(outputListed, "\n    "+cve.ID)
					outputListed = appendOutput(outputListed, cve)
				}
			}
		}
	}

	if len(cvesByLine) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}
		for _, cveStruct := range cvesByLine {
			if cveStruct.cve != nil {
				outputListed = append(outputListed, fmt.Sprintf("\n%s\nFound in lines: %s", cveStruct.name[0], func(i []int) string {
					str := ""
					for _, i := range cveStruct.lines {
						str += strconv.Itoa(i) + ", "
					}
					return str[:len(str)-2]
				}(cveStruct.lines)))
				outputListed = appendOutput(outputListed, cveStruct.cve[0])
			}
		}
	}
	return strings.Join(outputListed, "\n")
}

func FindCVEs(lines []string) ([]*Order, []*Order) {
	reCve := regexp.MustCompile(`CVE-\d{4}-\d{4,}`)
	reSoft := regexp.MustCompile(`(?i)(?:version)|(?:ver.)|(?:CVE\-\d{4}\-\d{4,})|(?:port[|\s:\\/]*\d{1,5})|(?:ping[\s:]+\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(?:\:\d{1,5})*)|([\w\-]+(?:\s\d+)?)(?:\s*(?:[:\\\/\s\-\|]+|ver\.|v\.|version(?:\s*(?:[\\/:|]*)\s*))\s*)((?:(?:\d+(?:\.\d+[a-z]*\d*)+(?:-\d(?:\.\d)+)?)|\d+[a-z]?\d*)(?:[\-\\\/]?dev|[\-\\\/]?beta|[\-\\\/]?alpha)?)`)
	softMap := map[string]*Order{}
	softKeys := []string{}
	cveMap := map[string]*Order{}
	cveKeys := []string{}
	for i, line := range lines {
		foundCve := reCve.FindAllString(line, -1)
		foundSoft := reSoft.FindAllStringSubmatch(line, -1)

		for _, f := range foundCve {
			cve, ok := cveMap[f]
			if !ok {
				cve = &Order{name: []string{f}}
				cveKeys = append(cveKeys, f)
			}
			cve.lines = append(cve.lines, i+1)
			cveMap[f] = cve
		}

		for _, f := range foundSoft {

			if len(f[1]) <= 2 || f[2] == "" {
				continue
			}

			soft, ok := softMap[f[0]]
			if !ok {
				soft = &Order{name: []string{f[0], f[1], f[2]}}
				softKeys = append(softKeys, f[0])
			}
			soft.lines = append(soft.lines, i+1)
			softMap[f[0]] = soft
		}
	}
	cve := []*Order{}
	for _, key := range cveKeys {
		cve = append(cve, cveMap[key])
	}

	software := []*Order{}
	for _, key := range softKeys {
		software = append(software, softMap[key])
	}
	return software, cve
}

func appendOutput(outputListed []string, cve *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf("\n    Description:\n%s", cve.Description))
	if cve.SeverityV40 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V40 metrics: %s", cve.SeverityV40))
	}
	if cve.SeverityV31 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V31 metrics: %s", cve.SeverityV31))
	}
	if cve.SeverityV30 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V30 metrics: %s", cve.SeverityV30))
	}
	if cve.SeverityV2 != "" {
		outputListed = append(outputListed, fmt.Sprintf("    Severity calculated with V2 metrics: %s", cve.SeverityV2))
	}
	outputListed = append(outputListed, "    Links:")
	outputListed = append(outputListed, cve.Links...)

	return outputListed
}
