package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
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
	http *http.Client
}

func NewNVDClient() *NVDClient {
	return &NVDClient{
		http: &http.Client{Timeout: 12 * time.Second},
	}
}

func (n *NVDClient) FetchCPEName(prod string) ([]string, error) {
	matchStringQuery := fmt.Sprintf("https://services.nvd.nist.gov/rest/json/cpematch/2.0?matchStringSearch=cpe:2.3:*:*:%s", prod)
	resp, err := n.http.Get(matchStringQuery)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("NVD: %s", string(body))
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

func (n *NVDClient) Fetch(link, subject string) (*CVEInfo, error) {
	url := link + subject
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
	if parsed.TotalResults == 0 {
		return nil, fmt.Errorf("no CVE was found")
	}
	info := &CVEInfo{}

	if len(parsed.Vulnerabilities) > 0 {
		info.ID = parsed.Vulnerabilities[0].CVE.ID
		if len(parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV2) > 0 {
			info.SeverityV2 = parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV2[0].BaseSeverity
		} else if len(parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV30) > 0 {
			info.SeverityV30 = parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV30[0].CvssData.BaseSeverity
		} else if len(parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV31) > 0 {
			info.SeverityV31 = parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV31[0].CvssData.BaseSeverity
		} else if len(parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV40) > 0 {
			info.SeverityV40 = parsed.Vulnerabilities[0].CVE.Metrics.CvssMetricV40[0].CvssData.BaseSeverity
		}
		for _, r := range parsed.Vulnerabilities[0].CVE.References {
			if !slices.Contains(r.Tags, "Broken Link") {
				info.Links = append(info.Links, r.Url)
			}
		}
		for _, r := range parsed.Vulnerabilities[0].CVE.Descriptions {
			if r.Language == "en" {
				info.Description = r.Value
			}
		}
	}
	return info, nil
}
