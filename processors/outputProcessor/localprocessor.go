package outputprocessor

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

type Processor struct {
	db       DB
	mu       sync.RWMutex
	cache    map[string]*Order
	software []*Order
	cve      []*Order
}

type Order struct {
	name []string
	cve  []*CVEInfo
}

type CVEInfo struct {
	ID          string
	Description string
	Severity    string
	Links       []string
}

func NewProcessor(db DB, cache map[string]*Order, software, cve []*Order) *Processor {
	return &Processor{db: db, cache: cache, software: software, cve: cve}
}

func (p *Processor) ProcessOutput(data string) string {
	outputDivided := strings.Split(data, "\n")

	p.findCVEs(outputDivided)

	sem := make(chan struct{}, len(outputDivided))
	defer close(sem)
	var wg sync.WaitGroup

	for _, soft := range p.software {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			ok := p.Get(soft)
			if ok {
				return
			}

			var err error
			soft.cve, err = p.db.GetPData(soft.name[1], soft.name[2])
			if err != nil {
				soft.cve = nil
				return
			}
			p.Set(soft.name[0], soft.cve)
		}()
	}

	for _, cveData := range p.cve {
		wg.Add(1)

		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			ok := p.Get(cveData)
			if ok {
				return
			}

			var err error
			cveData.cve, err = p.db.GetVulnerabilities(cveData.name[0])
			if err != nil {
				cveData.cve = nil
				return
			}

			p.Set(cveData.name[0], cveData.cve)
		}()
	}
	wg.Wait()
	return p.returnResults(data)
}

func (p *Processor) Get(soft *Order) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val, ok := p.cache[soft.name[0]]
	soft.cve = val.cve
	return ok
}

func (p *Processor) Set(key string, value []*CVEInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[key].cve = value
}

func (p *Processor) returnResults(data string) string {
	output := []string{fmt.Sprintf("\nProcessing results for %s:", data)}
	if len(p.software) != 0 {
		for _, soft := range p.software {
			if soft.cve != nil {
				output = append(output,
					fmt.Sprintf(
						"\n%s\nKnown CVE related to that:",
						soft.name[0]))

				for _, cve := range soft.cve {
					output = append(output, "\n"+cve.ID)

					output = appendOutput(output, cve)
				}
			}
		}
	}

	if len(p.cve) != 0 {
		for _, cveStruct := range p.cve {
			if cveStruct.cve != nil {
				output = append(output, fmt.Sprintf(
					"\n%s",
					cveStruct.name[0]))

				output = appendOutput(output, cveStruct.cve[0])
			}
		}
	}

	return strings.Join(output, "\n")
}

func (p *Processor) findCVEs(lines []string) {
	reCve := regexp.MustCompile(`CVE-\d{4}-\d{4,}`)
	reSoft := regexp.MustCompile(`(?i)(?:version)|(?:ver.)|(?:CVE\-\d{4}\-\d{4,})|(?:port[|\s:\\/]*\d{1,5})|(?:ping[\s:]+\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(?:\:\d{1,5})*)|([\w\-]+(?:\s\d+)?)(?:\s*(?:[:\\\/\s\-\|]+|ver\.|v\.|version(?:\s*(?:[\\/:|]*)\s*))\s*)((?:(?:\d+(?:\.\d+[a-z]*\d*)+(?:-\d(?:\.\d)+)?)|\d+[a-z]?\d*)(?:[\-\\\/]?dev|[\-\\\/]?beta|[\-\\\/]?alpha)?)`)

	softKeys := []string{}
	cveKeys := []string{}

	for _, line := range lines {
		foundCve := reCve.FindAllString(line, -1)
		foundSoft := reSoft.FindAllStringSubmatch(line, -1)

		for _, f := range foundCve {
			cve, ok := p.cache[f]
			if !ok {
				cve = &Order{name: []string{f}}
				cveKeys = append(cveKeys, f)
			}
			p.cache[f] = cve
		}

		for _, f := range foundSoft {
			if len([]rune(f[1])) <= 2 || f[2] == "" {
				continue
			}

			soft, ok := p.cache[f[0]]
			if !ok {
				soft = &Order{name: []string{f[0], f[1], f[2]}}
				softKeys = append(softKeys, f[0])
			}
			p.cache[f[0]] = soft
		}
	}

	for _, key := range cveKeys {
		p.cve = append(p.cve, p.cache[key])
	}

	for _, key := range softKeys {
		p.software = append(p.software, p.cache[key])
	}
}

func appendOutput(outputListed []string, cve *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf(
		"\nDescription:\n%s",
		cve.Description,
	),
	)

	outputListed = append(outputListed, fmt.Sprintf(
		"Severity calculated with V40 metrics: %s",
		cve.Severity))

	outputListed = append(outputListed, "Links:")
	outputListed = append(outputListed, cve.Links...)

	return outputListed
}
