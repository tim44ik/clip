package outputprocessor

import (
	"fmt"
	"log"
	"regexp"
	"slices"
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
			log.Println(soft)
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

func (p *Processor) Get(key *Order) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val, ok := p.cache[key.name[0]]
	if val.cve != nil {
		key.cve = val.cve
		return ok
	}
	return false
}

func (p *Processor) Set(key string, value []*CVEInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[key].cve = value
	for _, v := range value {
		if _, ok := p.cache[v.ID]; !ok {
			p.cache[v.ID] = &Order{name: []string{v.ID}, cve: []*CVEInfo{v}}
		}
	}
}

func (p *Processor) returnResults(data string) string {
	if len(p.software) == 0 && len(p.cve) == 0 {
		return ""
	}

	dr := []rune(data)
	if len(dr) > 20 {
		dr = dr[:20]
		dr = append(dr, []rune("...")...)
	}
	output := []string{fmt.Sprintf("\n\nProcessing results for \"%s\":", strings.TrimSpace(string(dr)))}
	for _, soft := range p.software {
		if soft.cve != nil {
			output = append(output,
				fmt.Sprintf(
					"\n>%s\n\nKnown CVE related to that:",
					soft.name[0]))

			for _, cve := range soft.cve {
				output = append(output, "\n"+cve.ID)

				output = appendOutput(output, cve)
			}
		}
	}

	for _, cveStruct := range p.cve {
		if cveStruct.cve != nil {
			output = append(output, "\n>"+cveStruct.name[0])
			output = appendOutput(output, cveStruct.cve[0])
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
			if !slices.Contains(cveKeys, f) {
				cveKeys = append(cveKeys, f)
			}
			cve, ok := p.cache[f]
			if !ok {
				cve = &Order{name: []string{f}}
				p.cache[f] = cve
			}
		}

		for _, f := range foundSoft {
			if len([]rune(f[1])) < 3 || len([]rune(f[2])) < 3 {
				continue
			}

			if !slices.Contains(softKeys, f[0]) {
				softKeys = append(softKeys, f[0])
			}

			soft, ok := p.cache[f[0]]
			if !ok {
				soft = &Order{name: []string{f[0], f[1], f[2]}}
				p.cache[f[0]] = soft
			}

		}
	}
	p.cve = []*Order{}
	p.software = []*Order{}

	for _, key := range cveKeys {
		p.cve = append(p.cve, p.cache[key])
	}

	for _, key := range softKeys {
		p.software = append(p.software, p.cache[key])
	}

	log.Println(softKeys)
}

func appendOutput(outputListed []string, cve *CVEInfo) []string {
	outputListed = append(outputListed, fmt.Sprintf(
		"\nDescription:\n%s",
		cve.Description,
	),
	)

	outputListed = append(outputListed, fmt.Sprintf(
		"\nSeverity: %s",
		cve.Severity))

	outputListed = append(outputListed, "\nLinks:")
	outputListed = append(outputListed, cve.Links...)

	return outputListed
}
