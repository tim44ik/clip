package outputprocessor

import (
	"clip/utility"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
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

type DB interface {
	Lookup(string) ([]string, error)
	Fetch(string, string) ([]*CVEInfo, error)
}

func ProcessOutput(db DB, output string) string {
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
			soft.cpe, err = db.Lookup(prod)
			if err != nil {
				soft.cpe = nil
				return
			}
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

				resp, err := db.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cpeName=", cpe)
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

			info, err := db.Fetch("https://services.nvd.nist.gov/rest/json/cves/2.0?cveId=", cve.name[0])
			if err != nil {
				return
			}

			cve.cve = append(cve.cve, info...)
		}()
	}

	wg.Wait()

	if len(softByLine) != 0 {
		if !slices.Contains(outputListed, "\nProcessing results:") {
			outputListed = append(outputListed, "\nProcessing results:")
		}

		for _, soft := range softByLine {
			if soft.cve != nil {
				outputListed = append(outputListed,
					fmt.Sprintf(
						"\n%s\n    Found in lines: %s\n    Known CVE(s) related to that:",
						soft.name[0],
						func(i []int) string {
							str := ""
							for _, i := range soft.lines {
								str += strconv.Itoa(i) + ", "
							}
							return str[:len(str)-2]
						}(soft.lines)))

				for _, cve := range soft.cve {
					outputListed = append(outputListed, "\n    "+cve.ID)

					outputListed = AppendOutput(outputListed, cve)
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
				outputListed = append(outputListed, fmt.Sprintf(
					"\n%s\nFound in lines: %s",
					cveStruct.name[0],
					func(i []int) string {
						str := ""
						for _, i := range cveStruct.lines {
							str += strconv.Itoa(i) + ", "
						}
						return str[:len(str)-2]
					}(cveStruct.lines)))

				outputListed = AppendOutput(outputListed, cveStruct.cve[0])
			}
		}
	}

	return strings.Join(outputListed, "\n")
}

func FindCVEs(lines []string) (cve, software []*Order) {
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

	for _, key := range cveKeys {
		cve = append(cve, cveMap[key])
	}

	for _, key := range softKeys {
		software = append(software, softMap[key])
	}

	return software, cve
}

func AppendOutput(outputListed []string, cve *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf(
		"\n    Description:\n%s",
		cve.Description,
	),
	)

	if cve.SeverityV40 != "" {
		outputListed = append(outputListed, fmt.Sprintf(
			"    Severity calculated with V40 metrics: %s",
			cve.SeverityV40))
	}
	if cve.SeverityV31 != "" {
		outputListed = append(outputListed, fmt.Sprintf(
			"    Severity calculated with V31 metrics: %s",
			cve.SeverityV31))
	}
	if cve.SeverityV30 != "" {
		outputListed = append(outputListed, fmt.Sprintf(
			"    Severity calculated with V30 metrics: %s",
			cve.SeverityV30))
	}
	if cve.SeverityV2 != "" {
		outputListed = append(outputListed, fmt.Sprintf(
			"    Severity calculated with V2 metrics: %s",
			cve.SeverityV2))
	}

	outputListed = append(outputListed, "    Links:")
	outputListed = append(outputListed, cve.Links...)

	return outputListed
}
