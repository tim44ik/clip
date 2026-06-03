package processor_test

import (
	"clip/engine/interpreter/eval"
	"clip/engine/interpreter/lexer"
	"clip/engine/interpreter/parser"
	"clip/models/nvd"
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProcessCaching(t *testing.T) {
	defer func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Unexpected panic")
			}
		}()
	}()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&nvd.CVE{}, &nvd.CPE{}, &nvd.CPE_CVE{})
	cpe1 := nvd.CPE{CPE_name: "cpe:2.3:o:test:test_product:1.0:*:*:*:*:*:*", Vendor: "test", Product: "test_product", Version: "1.0"}
	cpe2 := nvd.CPE{CPE_name: "cpe:2.3:o:test:test_product:2.0:*:*:*:*:*:*", Vendor: "test", Product: "test_product", Version: "2.0"}
	cve1 := nvd.CVE{ID: "CVE-2021-1234", Description: "test desc", Severity: "HIGH", References: ""}
	cve2 := nvd.CVE{ID: "CVE-2021-5678", Description: "another", Severity: "MEDIUM", References: ""}
	db.Create(&cpe1)
	db.Create(&cpe2)
	db.Create(&cve1)
	db.Create(&cve2)
	db.Create(&nvd.CPE_CVE{CPE_name: cpe1.CPE_name, CVE_id: cve1.ID})
	db.Create(&nvd.CPE_CVE{CPE_name: cpe2.CPE_name, CVE_id: cve2.ID})

	s := `
	%s = process("NVD", "test_product 1.0","test_product 1.0", "test_product 2.0","CVE-2021-1234", "CVE-2021-5678")
	%full = ""
	for %i = 0; %i<len(%s); %i=%i+1 do
		%full = %full +%s[%i]
	end
	print(%full)
	`
	l := lexer.NewLexer(s)
	p := parser.NewParser(l)
	prog := p.ParseProgram()
	env := eval.NewEnvironment(db, context.Background(), nil, "test", printString)
	env.Eval(prog)
}

//result
// Processing results for test_product 1.0:

// Known CVE related to that:

// CVE-2021-1234

// Description:
// test desc

// Severity calculated with V40 metrics: HIGH

// Links:

// Processing results for test_product 1.0:

// Known CVE related to that:

// CVE-2021-1234

// Description:
// test desc

// Severity calculated with V40 metrics: HIGH

// Links:

// Processing results for test_product 2.0:

// Known CVE related to that:

// CVE-2021-5678

// Description:
// another

// Severity calculated with V40 metrics: MEDIUM

// Links:

// Processing results for CVE-2021-1234:

// Description:
// test desc

// Severity calculated with V40 metrics: HIGH

// Links:

// Processing results for CVE-2021-5678:

// Description:
// another

// Severity calculated with V40 metrics: MEDIUM

// Links:

func printString(s any) {
	switch s := s.(type) {
	case []any:
		fmt.Print("\n")
		for i := range s {
			fmt.Printf("%v ", s[i])
		}
	default:
		fmt.Printf("\n%v", s)
	}
}
