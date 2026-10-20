package vnsalary

import (
	"errors"
	"testing"
	"time"
)

func monthlyPayments(t *testing.T, year int, gross VND, opts Options) []Payment {
	t.Helper()
	var payments []Payment
	for m := 1; m <= 12; m++ {
		opts.Date = date(year, m, 28)
		res, err := GrossToNet(gross, opts)
		if err != nil {
			t.Fatalf("GrossToNet: %v", err)
		}
		payments = append(payments, res.Payment(opts.Date))
	}
	return payments
}

func TestAnnual_Validation(t *testing.T) {
	ok := []Payment{{Date: date(2026, 3, 1), TaxableIncome: 20_000_000}}
	cases := []struct {
		name     string
		year     int
		payments []Payment
		opts     AnnualOptions
		want     error
	}{
		{"year before 2025", 2024, nil, AnnualOptions{}, ErrYearOutOfRange},
		{"year too large", 10_000, nil, AnnualOptions{}, ErrYearOutOfRange},
		{"payment in another year", 2026, []Payment{{Date: date(2027, 1, 1)}}, AnnualOptions{}, ErrPaymentOutsideYear},
		{"payment without date", 2026, []Payment{{TaxableIncome: 1}}, AnnualOptions{}, ErrDateZero},
		{"negative income", 2026, []Payment{{Date: date(2026, 1, 1), TaxableIncome: -1}}, AnnualOptions{}, ErrInvalidAmount},
		{"withheld too large", 2026, []Payment{{Date: date(2026, 1, 1), TaxWithheld: MaxGross + 1}}, AnnualOptions{}, ErrAmountTooLarge},
		{"too many payments", 2026, make([]Payment, MaxAnnualPayments+1), AnnualOptions{}, ErrTooManyPayments},
		{"negative dependent months", 2026, ok, AnnualOptions{DependentMonths: -1}, ErrInvalidDependentMonths},
		{"dependent months too large", 2026, ok, AnnualOptions{DependentMonths: MaxDependents*12 + 1}, ErrInvalidDependentMonths},
		{"negative charity", 2026, ok, AnnualOptions{Charity: -1}, ErrInvalidAmount},
		{"education too large", 2026, ok, AnnualOptions{Education: MaxGross + 1}, ErrAmountTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Annual(tc.year, tc.payments, tc.opts); !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// With the same salary every month, the yearly schedule (monthly limits × 12)
// gives the tax withheld, up to the half-up rounding of each monthly bracket.
func TestAnnual_SteadySalaryMatchesWithholding(t *testing.T) {
	for _, year := range []int{2025, 2026} {
		for _, gross := range []VND{12_345_678, 35_000_000, 87_654_321, 250_000_000} {
			payments := monthlyPayments(t, year, gross, Options{Region: Region1, Dependents: 1})
			res, err := Annual(year, payments, AnnualOptions{DependentMonths: 12})
			if err != nil {
				t.Fatalf("Annual: %v", err)
			}
			if limit := VND(6 * len(res.Brackets)); res.Balance > limit || res.Balance < -limit {
				t.Errorf("%d gross %d: balance %d beyond rounding (tax %d, withheld %d)", year, gross, res.Balance, res.Tax, res.Withheld)
			}
		}
	}
}

// The personal deduction counts 12 months even with fewer payments
// (Decree 253/2026/NĐ-CP Art. 48.1.b).
func TestAnnual_PersonalDeductionAlwaysTwelveMonths(t *testing.T) {
	payments := []Payment{{Date: date(2026, 12, 31), TaxableIncome: 200_000_000}}
	res, err := Annual(2026, payments, AnnualOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Deductions.Personal != 186_000_000 {
		t.Errorf("personal deduction: got %d, want 186,000,000", res.Deductions.Personal)
	}
	if res.AssessableIncome != 14_000_000 || res.Tax != 700_000 || res.Balance != 700_000 {
		t.Errorf("got assessable %d, tax %d, balance %d", res.AssessableIncome, res.Tax, res.Balance)
	}
}

func TestAnnual_HealthEducationCapsAndWarning(t *testing.T) {
	payments := []Payment{{Date: date(2026, 6, 30), TaxableIncome: 500_000_000}}
	res, err := Annual(2026, payments, AnnualOptions{Healthcare: 30_000_000, Education: 30_000_000, Charity: 1})
	if err != nil {
		t.Fatal(err)
	}
	d := res.Deductions
	if d.Healthcare != 23_000_000 || d.Education != 24_000_000 || d.Charity != 1 {
		t.Errorf("2026 deductions: got %+v", d)
	}
	if d.Total != d.Personal+d.Dependents+d.Charity+d.Healthcare+d.Education {
		t.Errorf("total %d is not the sum of its items", d.Total)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("2026: unexpected warnings %v", res.Warnings)
	}

	payments[0].Date = date(2025, 6, 30)
	res, err = Annual(2025, payments, AnnualOptions{Healthcare: 30_000_000})
	if err != nil {
		t.Fatal(err)
	}
	if res.Deductions.Healthcare != 0 {
		t.Errorf("2025: medical expenses must not be deducted, got %d", res.Deductions.Healthcare)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Code != WarnHealthEducationNotDeductible {
		t.Errorf("2025: want %s warning, got %v", WarnHealthEducationNotDeductible, res.Warnings)
	}
}

func TestAnnual_NoPaymentsAndRefundOnly(t *testing.T) {
	res, err := Annual(2026, nil, AnnualOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Tax != 0 || res.Balance != 0 || res.Brackets != nil {
		t.Errorf("no payments: got %+v", res)
	}
	// Withheld at 10% on income below the deductions: everything is refundable.
	res, err = Annual(2026, []Payment{{Date: date(2026, 5, 5), TaxableIncome: 50_000_000, TaxWithheld: 5_000_000}}, AnnualOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.AssessableIncome != 0 || res.Balance != -5_000_000 {
		t.Errorf("got assessable %d, balance %d", res.AssessableIncome, res.Balance)
	}
}

func TestAnnual_PaymentDateUsesItsOwnTimezone(t *testing.T) {
	hcm := time.FixedZone("ICT", 7*3600)
	// 2027-01-01 00:30 in Vietnam is still 2026 in UTC; the payment belongs to 2027.
	p := Payment{Date: time.Date(2027, 1, 1, 0, 30, 0, 0, hcm), TaxableIncome: 1}
	if _, err := Annual(2026, []Payment{p}, AnnualOptions{}); !errors.Is(err, ErrPaymentOutsideYear) {
		t.Errorf("got %v, want ErrPaymentOutsideYear", err)
	}
}

func TestAnnual_ResultDoesNotLeakPolicySources(t *testing.T) {
	res, err := Annual(2026, nil, AnnualOptions{})
	if err != nil {
		t.Fatal(err)
	}
	res.Policy.Sources[0] = "changed"
	again, _ := Annual(2026, nil, AnnualOptions{})
	if again.Policy.Sources[0] == "changed" {
		t.Error("AnnualResult shares the policy's Sources slice")
	}
}

func TestScheduleTax_TopBracketDoesNotOverflow(t *testing.T) {
	p, _ := policyAt(date(2026, 1, 1))
	tax, lines := scheduleTax(p.Brackets, 12, 1_000_000_000_000)
	if len(lines) != len(p.Brackets) || tax <= 0 {
		t.Errorf("got tax %d with %d brackets", tax, len(lines))
	}
}
