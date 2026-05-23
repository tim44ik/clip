package outputprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"slices"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type NVDResponse struct {
	TotalResults    int `json:"totalResults"`
	Vulnerabilities []struct {
		CVE struct {
			ID           string `json:"id"`
			Descriptions []struct {
				Language string `json:"lang"`
				Value    string `json:"value"`
			} `json:"descriptions"`
			References []struct {
				Url  string   `json:"url"`
				Tags []string `json:"tags"`
			} `json:"references"`
			Metrics struct {
				CvssMetricV2 []struct {
					BaseSeverity string `json:"baseSeverity"`
				} `json:"cvssMetricV2"`
				CvssMetricV30 []struct {
					CvssData struct {
						BaseSeverity string `json:"baseSeverity"`
					} `json:"cvssData"`
				} `json:"cvssMetricV30"`
				CvssMetricV31 []struct {
					CvssData struct {
						BaseSeverity string `json:"baseSeverity"`
					} `json:"cvssData"`
				} `json:"cvssMetricV31"`
				CvssMetricV40 []struct {
					CvssData struct {
						BaseSeverity string `json:"baseSeverity"`
					} `json:"cvssData"`
				} `json:"cvssMetricV40"`
			} `json:"metrics"`
		} `json:"cve"`
	} `json:"vulnerabilities"`
}

type SoftVerLookup struct {
	TotalResults int `json:"totalResults"`
	MatchStrings []struct {
		MatchString struct {
			Matches []struct {
				CpeName string `json:"cpeName"`
			} `json:"matches"`
		} `json:"matchString"`
	} `json:"matchStrings"`
}

type NVDClient struct {
	maxRate int
	ctx     context.Context
	http    *http.Client
}

var NVDlimiter = rate.NewLimiter(rate.Every((time.Duration(rand.Intn(1000)+6000))*time.Millisecond), 1)

func (n *NVDClient) GetMaxRate() int {
	return n.maxRate
}

func (n *NVDClient) Lookup(prod string) ([]string, error) {
	body, err := n.connectAndFetch("https://services.nvd.nist.gov/rest/json/cpematch/2.0?matchStringSearch=cpe:2.3:*:*:" + prod)
	if err != nil {
		return nil, err
	}

	var parsed SoftVerLookup
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	if parsed.TotalResults == 0 {
		return nil, fmt.Errorf("no CPE was found")
	}

	success := []string{}
	for _, match := range parsed.MatchStrings {
		if len(match.MatchString.Matches) > 0 {
			for _, cpe := range match.MatchString.Matches {
				if strings.Contains(cpe.CpeName, prod) && !slices.Contains(success, cpe.CpeName) {
					success = append(success, cpe.CpeName)
				}
			}
		}
	}
	return success, nil
}

func (n *NVDClient) Fetch(link, subject string) ([]*CVEInfo, error) {
	body, err := n.connectAndFetch(link + subject)
	if err != nil {
		return nil, err
	}
	var parsed NVDResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.TotalResults == 0 {
		return nil, fmt.Errorf("no CVE was found")
	}

	infoSlice := []*CVEInfo{}
	for _, vulnerability := range parsed.Vulnerabilities {

		info := &CVEInfo{}
		info.ID = vulnerability.CVE.ID

		if vulnerability.CVE.Metrics.CvssMetricV2 != nil {
			info.SeverityV2 = vulnerability.CVE.Metrics.CvssMetricV2[0].BaseSeverity
		}
		if vulnerability.CVE.Metrics.CvssMetricV30 != nil {
			info.SeverityV30 = vulnerability.CVE.Metrics.CvssMetricV30[0].CvssData.BaseSeverity
		}
		if vulnerability.CVE.Metrics.CvssMetricV31 != nil {
			info.SeverityV31 = vulnerability.CVE.Metrics.CvssMetricV31[0].CvssData.BaseSeverity
		}
		if vulnerability.CVE.Metrics.CvssMetricV40 != nil {
			info.SeverityV40 = vulnerability.CVE.Metrics.CvssMetricV40[0].CvssData.BaseSeverity
		}

		for _, r := range vulnerability.CVE.References {
			if !slices.Contains(info.Links, r.Url) {
				if r.Tags != nil {
					if !slices.Contains(r.Tags, "Broken Link") {
						info.Links = append(info.Links, r.Url)
					}
				} else {
					info.Links = append(info.Links, r.Url)
				}
			}
			if len(info.Links) == 10 {
				break
			}
		}

		for _, r := range vulnerability.CVE.Descriptions {
			if r.Language == "en" {
				info.Description = r.Value
			}
		}
		infoSlice = append(infoSlice, info)
	}
	return infoSlice, nil
}

func (n *NVDClient) connectAndFetch(link string) ([]byte, error) {
	NVDlimiter.Wait(n.ctx)

	req, err := http.NewRequestWithContext(n.ctx, "GET", link, nil)
	if err != nil {
		return nil, err
	}

	resp, err := n.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("NVD: %s", string(body))
	}

	return body, nil
}
