package core

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type NVDResponse struct {
	Vulnerabilities []struct {
		CVE struct {
			ID         string `json:"id"`
			References []struct {
				Url string `json:"url"`
			} `json:"references"`
		} `json:"cve"`

		Metrics struct {
			CvssMetricV2 []struct {
				BaseSeverity string `json:"baseSeverity"`
			} `json:"cvssMetricV2"`
		} `json:"metrics"`
	} `json:"vulnerabilities"`
}

type NVDClient struct {
	http  *http.Client
	mu    sync.Mutex
	cache map[string]*CVEInfo
}

type CVEInfo struct {
	Severity string
	Links    []string
}

type Job struct {
	CVE string
}

type Result struct {
	CVE  string
	Info *CVEInfo
	Err  error
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
			enumed := enumLines(m.Output)
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

	outputListed := enumLines(output)

	cvesByLine := FindCVEs(outputListed)

	var cveList []string
	for cve := range cvesByLine {
		cveList = append(cveList, cve)
	}

	maxGoroutines := 10
	sem := make(chan struct{}, maxGoroutines)

	var wg sync.WaitGroup
	cveData := sync.Map{}

	for _, cve := range cveList {
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
		}(cve)
	}

	wg.Wait()

	outputListed = append(outputListed, "\nProcessing results:\n")
	for cve, lines := range cvesByLine {
		dataAny, _ := cveData.Load(cve)
		info := dataAny.(*CVEInfo)

		builder := strings.Builder{}
		builder.WriteString(fmt.Sprintf("%s found in lines: %s\n", cve, strings.Join(lines, ", ")))
		builder.WriteString(fmt.Sprintf("severity: %s\n", info.Severity))
		builder.WriteString("links:\n")

		for _, l := range info.Links {
			builder.WriteString(l + "\n")
		}

		outputListed = append(outputListed, builder.String())
	}

	return strings.Join(outputListed, "\n")
}

func enumLines(output string) []string {
	divided := strings.Split(output, "\n")
	for i, v := range divided[:len(divided)-2] {
		divided[i] = strconv.Itoa(i+1) + "  " + v
	}
	return divided
}

func NewNVDClient() *NVDClient {
	return &NVDClient{
		http:  &http.Client{Timeout: 10 * time.Second},
		cache: make(map[string]*CVEInfo),
	}
}

func (n *NVDClient) Fetch(cve string) (*CVEInfo, error) {
	n.mu.Lock()
	if v, ok := n.cache[cve]; ok {
		n.mu.Unlock()
		return v, nil
	}
	n.mu.Unlock()

	url := fmt.Sprintf("https://services.nvd.nist.gov/rest/json/cves/2.0?cveId=%s", cve)
	resp, err := n.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("NVD: %s", string(body))
	}

	var parsed NVDResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	info := &CVEInfo{}

	if len(parsed.Vulnerabilities) > 0 {
		v := parsed.Vulnerabilities[0]

		if len(v.Metrics.CvssMetricV2) > 0 {
			info.Severity = v.Metrics.CvssMetricV2[0].BaseSeverity
		} else {
			info.Severity = "UNKNOWN"
		}

		for _, r := range v.CVE.References {
			info.Links = append(info.Links, r.Url)
		}
	}

	n.mu.Lock()
	n.cache[cve] = info
	n.mu.Unlock()

	return info, nil
}

func FindCVEs(lines []string) map[string][]string {
	re := regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)

	result := make(map[string][]string)

	for i, line := range lines {
		found := re.FindAllString(line, -1)
		seen := map[string]bool{}

		for _, cve := range found {
			if !seen[cve] {
				result[cve] = append(result[cve], strconv.Itoa(i+1))
				seen[cve] = true
			}
		}
	}
	return result
}
