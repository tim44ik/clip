package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
	http *http.Client
}

func NewNVDClient() *NVDClient {
	return &NVDClient{
		http: &http.Client{Timeout: 12 * time.Second},
	}
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
	return info, nil
}
