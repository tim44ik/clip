package outputprocessor_test

import (
	"clip/models/nvd"
	outputprocessor "clip/processors/outputProcessor"
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mockDB struct {
	getPDataFunc           func(product, version string) ([]*outputprocessor.CVEInfo, error)
	getVulnerabilitiesFunc func(cveID string) ([]*outputprocessor.CVEInfo, error)
}

func (m *mockDB) GetPData(product, version string) ([]*outputprocessor.CVEInfo, error) {
	return m.getPDataFunc(product, version)
}

func (m *mockDB) GetVulnerabilities(cveID string) ([]*outputprocessor.CVEInfo, error) {
	return m.getVulnerabilitiesFunc(cveID)
}

func TestProcessOutput(t *testing.T) {
	mock := &mockDB{
		getPDataFunc: func(product, version string) ([]*outputprocessor.CVEInfo, error) {
			if product == "test_product" && version == "1.0" {
				return []*outputprocessor.CVEInfo{
					{ID: "CVE-2021-1234", Description: "test desc", Severity: "HIGH", Links: []string{"http://example.com"}},
				}, nil
			}
			return nil, nil
		},
		getVulnerabilitiesFunc: func(cveID string) ([]*outputprocessor.CVEInfo, error) {
			if cveID == "CVE-2021-5678" {
				return []*outputprocessor.CVEInfo{
					{ID: "CVE-2021-5678", Description: "another desc", Severity: "MEDIUM", Links: []string{}},
				}, nil
			}
			return nil, nil
		},
	}
	proc := outputprocessor.NewProcessor(mock, make(map[string]*outputprocessor.Order), []*outputprocessor.Order{}, []*outputprocessor.Order{})

	input := `test_product version 1.0
CVE-2021-5678
`
	result := proc.ProcessOutput(input)
	print(result)
}

// result
// Processing results for "test_product version 1.0
// CVE-2021-5678":

// test_product version 1.0
// Known CVE related to that:

// CVE-2021-1234

// Description:
// test desc

// Severity calculated with V40 metrics: HIGH

// Links:
// http://example.com

// CVE-2021-5678

// Description:
// another desc

// Severity calculated with V40 metrics: MEDIUM

func TestNVDClient_GetPData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(&nvd.CVE{}, &nvd.CPE{}, &nvd.CPE_CVE{})
	if err != nil {
		t.Fatal(err)
	}

	cpe := nvd.CPE{CPE_name: "cpe:2.3:o:test:test_product:1.0:*:*:*:*:*:*", Vendor: "test", Product: "test_product", Version: "1.0"}
	cve := nvd.CVE{ID: "CVE-2021-1234", Description: "test desc", Severity: "HIGH", References: "http://example.com"}
	relation := nvd.CPE_CVE{CPE_name: cpe.CPE_name, CVE_id: cve.ID}
	db.Create(&cpe)
	db.Create(&cve)
	db.Create(&relation)

	client := outputprocessor.NewDB(db, "NVD", context.Background())
	result, err := client.GetPData("test_product", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("Unexpected result: %d", len(result))
	}
	if result[0].ID != "CVE-2021-1234" {
		t.Errorf("Wrong ID: %s", result[0].ID)
	}
	if result[0].Severity != "HIGH" {
		t.Errorf("Wrong Severity data: %s", result[0].Severity)
	}
	if len(result[0].Links) != 1 || result[0].Links[0] != "http://example.com" {
		t.Errorf("Wrong links: %v", result[0].Links)
	}
}

func TestNVDClient_GetVulnerabilities(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&nvd.CVE{})
	cve := nvd.CVE{ID: "CVE-2021-5678", Description: "another desc", Severity: "MEDIUM", References: ""}
	db.Create(&cve)

	client := outputprocessor.NewDB(db, "NVD", context.Background())
	result, err := client.GetVulnerabilities("CVE-2021-5678")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("Unexpected result: %d", len(result))
	}
	if result[0].ID != "CVE-2021-5678" {
		t.Errorf("Wrong ID: %s", result[0].ID)
	}
}
