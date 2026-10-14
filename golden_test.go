package vnsalary_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/HuynhNhatKhanh/vnsalary"
)

// goldenCase is one entry of testdata/golden.json. The per-item insurance
// fields are optional; when present they pin the rounding of each item.
// Source and VerifiedAt are mandatory: every expected value must say where it
// came from (see CONTRIBUTING.md).
type goldenCase struct {
	Description            string `json:"description"`
	Gross                  int64  `json:"gross"`
	Dependents             int    `json:"dependents"`
	Region                 int    `json:"region"`
	Date                   string `json:"date"`
	InsBase                int64  `json:"ins_base"`
	ExpectedSocial         *int64 `json:"expected_social"`
	ExpectedHealth         *int64 `json:"expected_health"`
	ExpectedUnemployment   *int64 `json:"expected_unemployment"`
	ExpectedInsuranceTotal int64  `json:"expected_insurance_total"`
	ExpectedTax            int64  `json:"expected_tax"`
	ExpectedNet            int64  `json:"expected_net"`
	Source                 string `json:"source"`
	VerifiedAt             string `json:"verified_at"`
}

func TestGoldenData(t *testing.T) {
	data, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatalf("failed to read testdata/golden.json: %v", err)
	}

	var cases []goldenCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed to parse golden.json: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			if tc.Source == "" || tc.VerifiedAt == "" {
				t.Errorf("case must have source and verified_at")
			}
			date, err := time.Parse("2006-01-02", tc.Date)
			if err != nil {
				t.Fatalf("invalid date format in test case: %v", err)
			}

			opts := vnsalary.Options{
				Dependents: tc.Dependents,
				Region:     vnsalary.Region(tc.Region),
				Date:       date,
			}
			if tc.InsBase > 0 {
				insBase := vnsalary.VND(tc.InsBase)
				opts.InsuranceBase = &insBase
			}

			// Forward calculation: Gross -> Net
			res, err := vnsalary.GrossToNet(vnsalary.VND(tc.Gross), opts)
			if err != nil {
				t.Fatalf("GrossToNet failed: %v", err)
			}

			checkOptional(t, "Social", res.Insurance.Social, tc.ExpectedSocial)
			checkOptional(t, "Health", res.Insurance.Health, tc.ExpectedHealth)
			checkOptional(t, "Unemployment", res.Insurance.Unemployment, tc.ExpectedUnemployment)
			if int64(res.Insurance.Total) != tc.ExpectedInsuranceTotal {
				t.Errorf("Insurance total mismatch: got %d, want %d", res.Insurance.Total, tc.ExpectedInsuranceTotal)
			}
			if int64(res.Tax) != tc.ExpectedTax {
				t.Errorf("Tax mismatch: got %d, want %d", res.Tax, tc.ExpectedTax)
			}
			if int64(res.Net) != tc.ExpectedNet {
				t.Errorf("Net mismatch: got %d, want %d", res.Net, tc.ExpectedNet)
			}

			// Reverse calculation: Net -> Gross returns the smallest gross with
			// the same net, which is the case gross itself or a few VND below it.
			inv, err := vnsalary.NetToGross(res.Net, opts)
			if err != nil {
				t.Fatalf("NetToGross failed: %v", err)
			}
			if inv.Net != res.Net {
				t.Errorf("NetToGross inverse Net mismatch: got %d, want %d", inv.Net, res.Net)
			}
			if inv.Gross > res.Gross {
				t.Errorf("NetToGross returned gross %d above case gross %d", inv.Gross, res.Gross)
			}
			if inv.Gross > 0 {
				below, err := vnsalary.GrossToNet(inv.Gross-1, opts)
				if err == nil && below.Net == res.Net {
					t.Errorf("NetToGross gross %d is not minimal: %d gives the same net", inv.Gross, inv.Gross-1)
				}
			}
		})
	}
}

func checkOptional(t *testing.T, name string, got vnsalary.VND, want *int64) {
	t.Helper()
	if want != nil && int64(got) != *want {
		t.Errorf("%s mismatch: got %d, want %d", name, got, *want)
	}
}
