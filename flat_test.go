package vnsalary

import (
	"errors"
	"testing"
	"time"
)

func TestFlatWithholding_ThresholdByPaymentDate(t *testing.T) {
	cases := []struct {
		name         string
		amount       VND
		y, m, d      int
		wantRequired bool
		wantWithheld VND
	}{
		// Circular 111/2013 Art. 25.1.i: from 2,000,000 per payment.
		{"2025 below 2M", 1_999_999, 2025, 5, 1, false, 0},
		{"2025 at 2M", 2_000_000, 2025, 5, 1, true, 200_000},
		{"last day of 2M threshold", 4_999_999, 2026, 6, 30, true, 500_000},
		// Decree 253/2026 Art. 50.2: from 5,000,000 per payment, from 2026-07-01.
		{"first day of 5M threshold, below", 4_999_999, 2026, 7, 1, false, 0},
		{"at 5M", 5_000_000, 2026, 7, 1, true, 500_000},
		{"odd amount rounds half-up", 12_345_675, 2026, 8, 15, true, 1_234_568},
		{"zero", 0, 2026, 8, 15, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := FlatWithholding(tc.amount, date(tc.y, tc.m, tc.d))
			if err != nil {
				t.Fatal(err)
			}
			if res.Required != tc.wantRequired || res.Withheld != tc.wantWithheld {
				t.Errorf("got required %v, withheld %d; want %v, %d", res.Required, res.Withheld, tc.wantRequired, tc.wantWithheld)
			}
			if res.Net != tc.amount-tc.wantWithheld {
				t.Errorf("net %d != amount - withheld", res.Net)
			}
		})
	}
}

func TestFlatWithholding_Errors(t *testing.T) {
	cases := []struct {
		amount VND
		y      int
		want   error
	}{
		{-1, 2026, ErrInvalidAmount},
		{MaxGross + 1, 2026, ErrAmountTooLarge},
		{1_000_000, 2020, ErrDateOutOfRange},
	}
	for _, tc := range cases {
		if _, err := FlatWithholding(tc.amount, date(tc.y, 1, 1)); !errors.Is(err, tc.want) {
			t.Errorf("amount %d year %d: got %v, want %v", tc.amount, tc.y, err, tc.want)
		}
	}
	if _, err := FlatWithholding(1, time.Time{}); !errors.Is(err, ErrDateZero) {
		t.Errorf("zero date: got %v, want ErrDateZero", err)
	}
}

// A student paid 8,000,000 once in 2026 has 800,000 withheld; with no other
// income, finalization refunds all of it.
func TestFlatWithholding_RefundedAtFinalization(t *testing.T) {
	paid := date(2026, 9, 10)
	res, err := FlatWithholding(8_000_000, paid)
	if err != nil {
		t.Fatal(err)
	}
	annual, err := Annual(2026, []Payment{res.Payment(paid)}, AnnualOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if annual.Tax != 0 || annual.Balance != -800_000 {
		t.Errorf("got tax %d, balance %d; want 0, -800,000", annual.Tax, annual.Balance)
	}
}
