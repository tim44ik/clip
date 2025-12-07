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
		} `json:"cve"`

		Metrics struct {
			CvssMetricV2 []struct {
				CvssData struct {
					BaseScore int `json:"baseScore"`
				} `json:"cvssData"`
				BaseSeverity string `json:"baseSeverity"`
			} `json:"cvssMetricV2"`
			CvssMetricV30 []struct {
				CvssData struct {
					BaseScore    int    `json:"baseScore"`
					BaseSeverity string `json:"baseSeverity"`
				} `json:"cvssData"`
			} `json:"cvssMetricV30"`
			CvssMetricV31 []struct {
				CvssData struct {
					BaseScore    int    `json:"baseScore"`
					BaseSeverity string `json:"baseSeverity"`
				} `json:"cvssData"`
			} `json:"cvssMetricV31"`
			CvssMetricV40 []struct {
				CvssData struct {
					BaseScore    int    `json:"baseScore"`
					BaseSeverity string `json:"baseSeverity"`
				} `json:"cvssData"`
			} `json:"cvssMetricV40"`
		} `json:"metrics"`
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

func (n *NVDClient) FetchCPEName(prod, ver string) ([]string, error) {
	matchStringQuery := fmt.Sprintf("https://services.nvd.nist.gov/rest/json/cpematch/2.0?matchStringSearch=cpe:2.3:*:*:%s:%s", prod, ver)
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
		return nil, fmt.Errorf("No CPE was found")
	}

	success := []string{}
	for _, match := range parsed.MatchStrings {
		if len(match.MatchString.Matches) > 0 {
			for _, cpe := range match.MatchString.Matches {
				if strings.Contains(cpe.CpeName, prod+":"+ver) && !slices.Contains(success, cpe.CpeName) {
					success = append(success, cpe.CpeName)
				}
			}
		}
	}
	return success, nil
}

func (n *NVDClient) FetchCVEByCPE(cpeName string) (*CVEInfo, error) {
	matchStringQuery := fmt.Sprintf("https://services.nvd.nist.gov/rest/json/cves/2.0?cpeName=%s", cpeName)
	resp, err := n.http.Get(matchStringQuery)
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
		return nil, fmt.Errorf("No CVE was found")
	}
	info := &CVEInfo{}

	if len(parsed.Vulnerabilities) > 0 {
		info.ID = parsed.Vulnerabilities[0].CVE.ID
		for _, st := range parsed.Vulnerabilities[0].CVE.Descriptions {
			if st.Language == "en" {
				info.Description = st.Value
			}
		}
		if len(parsed.Vulnerabilities[0].Metrics.CvssMetricV2) > 0 {
			info.SeverityV2 = parsed.Vulnerabilities[0].Metrics.CvssMetricV2[0].BaseSeverity
			info.V2Score = parsed.Vulnerabilities[0].Metrics.CvssMetricV2[0].CvssData.BaseScore
		} else if len(parsed.Vulnerabilities[0].Metrics.CvssMetricV30) > 0 {
			info.SeverityV30 = parsed.Vulnerabilities[0].Metrics.CvssMetricV30[0].CvssData.BaseSeverity
			info.V30Score = parsed.Vulnerabilities[0].Metrics.CvssMetricV30[0].CvssData.BaseScore
		} else if len(parsed.Vulnerabilities[0].Metrics.CvssMetricV31) > 0 {
			info.SeverityV31 = parsed.Vulnerabilities[0].Metrics.CvssMetricV31[0].CvssData.BaseSeverity
			info.V31Score = parsed.Vulnerabilities[0].Metrics.CvssMetricV31[0].CvssData.BaseScore
		} else if len(parsed.Vulnerabilities[0].Metrics.CvssMetricV40) > 0 {
			info.SeverityV40 = parsed.Vulnerabilities[0].Metrics.CvssMetricV40[0].CvssData.BaseSeverity
			info.V40Score = parsed.Vulnerabilities[0].Metrics.CvssMetricV40[0].CvssData.BaseScore
		}
		for _, r := range parsed.Vulnerabilities[0].CVE.References {
			if !slices.Contains(r.Tags, "Broken Link") {
				info.Links = append(info.Links, r.Url)
			}
		}
	}
	return info, nil
}

func (n *NVDClient) Fetch(cve string) (*CVEInfo, error) {
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
	if parsed.TotalResults == 0 {
		return nil, fmt.Errorf("No CVE was found")
	}
	info := &CVEInfo{}

	if len(parsed.Vulnerabilities) > 0 {
		v := parsed.Vulnerabilities[0]
		info.ID = v.CVE.ID
		if len(v.Metrics.CvssMetricV2) > 0 {
			info.SeverityV2 = v.Metrics.CvssMetricV2[0].BaseSeverity
			info.V2Score = v.Metrics.CvssMetricV2[0].CvssData.BaseScore
		} else if len(v.Metrics.CvssMetricV30) > 0 {
			info.SeverityV30 = v.Metrics.CvssMetricV30[0].CvssData.BaseSeverity
			info.V30Score = v.Metrics.CvssMetricV30[0].CvssData.BaseScore
		} else if len(v.Metrics.CvssMetricV31) > 0 {
			info.SeverityV31 = v.Metrics.CvssMetricV31[0].CvssData.BaseSeverity
			info.V31Score = v.Metrics.CvssMetricV31[0].CvssData.BaseScore
		} else if len(v.Metrics.CvssMetricV40) > 0 {
			info.SeverityV40 = v.Metrics.CvssMetricV40[0].CvssData.BaseSeverity
			info.V40Score = v.Metrics.CvssMetricV40[0].CvssData.BaseScore
		}
		for _, r := range v.CVE.References {
			if !slices.Contains(r.Tags, "Broken Link") {
				info.Links = append(info.Links, r.Url)
			}
		}
	}
	return info, nil
}
