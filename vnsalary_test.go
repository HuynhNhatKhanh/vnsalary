package vnsalary

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func TestGrossToNet_BasicCases(t *testing.T) {
	opts2024 := Options{
		Dependents: 0,
		Region:     Region1,
		Date:       date(2024, 10, 1),
	}

	// Case 1: Low income below personal deduction (11M), PIT = 0, only mandatory 10.5% insurance deducted
	// 5,000,000 VND:
	// BHXH 8% = 400,000; BHYT 1.5% = 75,000; BHTN 1% = 50,000 -> Total insurance = 525,000 VND
	// Personal deduction = 11,000,000 VND -> Assessable income = 0
	// Net = 5,000,000 - 525,000 = 4,475,000 VND
	res1, err := GrossToNet(5_000_000, opts2024)
	if err != nil {
		t.Fatalf("GrossToNet failed: %v", err)
	}
	if res1.Net != 4_475_000 {
		t.Errorf("expected Net 4,475,000, got %d", res1.Net)
	}
	if res1.Tax != 0 {
		t.Errorf("expected Tax 0, got %d", res1.Tax)
	}
	if res1.Insurance.Total != 525_000 {
		t.Errorf("expected Insurance 525,000, got %d", res1.Insurance.Total)
	}

	// Case 2: 15,000,000 VND (2024 regime, 0 dependents)
	// Insurance 10.5% = 1,575,000 VND
	// Deductions = 11,000,000 VND
	// Assessable income = 15,000,000 - 1,575,000 - 11,000,000 = 2,425,000 VND (Bracket 1: 5%)
	// PIT = 2,425,000 * 5% = 121,250 VND
	// Net = 15,000,000 - 1,575,000 - 121,250 = 13,303,750 VND
	res2, err := GrossToNet(15_000_000, opts2024)
	if err != nil {
		t.Fatalf("GrossToNet failed: %v", err)
	}
	if res2.Tax != 121_250 {
		t.Errorf("expected Tax 121,250, got %d", res2.Tax)
	}
	if res2.Net != 13_303_750 {
		t.Errorf("expected Net 13,303,750, got %d", res2.Net)
	}
}

func TestGrossToNet_InsuranceCaps(t *testing.T) {
	opts2024 := Options{
		Dependents: 0,
		Region:     Region1, // Region 1: Min wage 4,960,000 VND -> Unemployment insurance cap = 99,200,000 VND
		Date:       date(2024, 10, 1),
	}

	// Gross = 60,000,000 VND (exceeds social & health cap of 46.8M, but below unemployment cap of 99.2M)
	res, err := GrossToNet(60_000_000, opts2024)
	if err != nil {
		t.Fatalf("GrossToNet failed: %v", err)
	}

	if res.Insurance.Social != 3_744_000 {
		t.Errorf("expected Social 3,744,000 (8%% of 46.8M), got %d", res.Insurance.Social)
	}
	if res.Insurance.Health != 702_000 {
		t.Errorf("expected Health 702,000 (1.5%% of 46.8M), got %d", res.Insurance.Health)
	}
	if res.Insurance.Unemployment != 600_000 {
		t.Errorf("expected Unemployment 600,000 (1%% of 60M), got %d", res.Insurance.Unemployment)
	}

	// Gross = 120,000,000 VND (exceeds both 46.8M social/health cap and 99.2M unemployment cap)
	res2, err := GrossToNet(120_000_000, opts2024)
	if err != nil {
		t.Fatalf("GrossToNet failed: %v", err)
	}
	if res2.Insurance.Unemployment != 992_000 {
		t.Errorf("expected Unemployment 992,000 (1%% of 99.2M), got %d", res2.Insurance.Unemployment)
	}
}

func TestGrossToNet_FixedInsuranceBase(t *testing.T) {
	insBase := VND(5_000_000)
	opts := Options{
		InsuranceBase: &insBase,
		Dependents:    0,
		Region:        Region1,
		Date:          date(2024, 10, 1),
	}

	res, err := GrossToNet(30_000_000, opts)
	if err != nil {
		t.Fatalf("GrossToNet failed: %v", err)
	}

	// Insurance calculated on contractual base 5,000,000 VND rather than 30,000,000 VND
	// 5M * 10.5% = 525,000 VND
	if res.Insurance.Total != 525_000 {
		t.Errorf("expected Insurance 525,000, got %d", res.Insurance.Total)
	}
}

func TestPolicyDiff_2024_vs_2026(t *testing.T) {
	gross := VND(35_000_000)

	// 2024 Policy
	opts2024 := Options{
		Dependents: 1,
		Region:     Region1,
		Date:       date(2024, 10, 1),
	}
	res2024, err := GrossToNet(gross, opts2024)
	if err != nil {
		t.Fatalf("GrossToNet 2024 failed: %v", err)
	}

	// 2026 Policy
	opts2026 := Options{
		Dependents: 1,
		Region:     Region1,
		Date:       date(2026, 1, 15),
	}
	res2026, err := GrossToNet(gross, opts2026)
	if err != nil {
		t.Fatalf("GrossToNet 2026 failed: %v", err)
	}

	if res2026.Net <= res2024.Net {
		t.Errorf("2026 Net (%d) should be greater than 2024 Net (%d) due to tax reduction",
			res2026.Net, res2024.Net)
	}

	diffNet := res2026.Net - res2024.Net
	if diffNet != 1_157_500 {
		t.Errorf("expected net gain 1,157,500 VND in 2026, got %d", diffNet)
	}
}

func TestNetToGross_RoundTripProperty(t *testing.T) {
	testDates := []time.Time{
		date(2024, 8, 1),
		date(2026, 2, 1),
	}

	testGrosses := []VND{
		0,
		4_000_000,
		10_000_000,
		15_000_000,
		20_000_000,
		25_000_000,
		35_000_000,
		46_800_000,
		50_000_000,
		75_000_000,
		99_200_000,
		100_000_000,
		150_000_000,
		250_000_000,
	}

	for _, d := range testDates {
		for _, dep := range []int{0, 1, 2} {
			for _, reg := range []Region{Region1, Region2} {
				opts := Options{
					Dependents: dep,
					Region:     reg,
					Date:       d,
				}

				for _, g := range testGrosses {
					resForward, err := GrossToNet(g, opts)
					if err != nil {
						t.Fatalf("GrossToNet failed for %d: %v", g, err)
					}

					resInverse, err := NetToGross(resForward.Net, opts)
					if err != nil {
						t.Fatalf("NetToGross failed for Net %d (Gross %d): %v", resForward.Net, g, err)
					}

					if resInverse.Net != resForward.Net {
						t.Errorf("Roundtrip mismatch: NetToGross(%d).Net = %d, want %d",
							resForward.Net, resInverse.Net, resForward.Net)
					}
					if resInverse.Gross > g {
						t.Errorf("NetToGross returned gross %d > original %d for net %d",
							resInverse.Gross, g, resForward.Net)
					}
				}
			}
		}
	}
}

func TestPolicyAt_Immutability(t *testing.T) {
	d := date(2024, 8, 1)
	p1, err := PolicyAt(d)
	if err != nil {
		t.Fatalf("PolicyAt failed: %v", err)
	}

	// Attempt to mutate returned data
	origNumerator := p1.Brackets[0].RateNumerator
	p1.Brackets[0].RateNumerator = 99
	p1.Ref.Sources[0] = "MODIFIED_BY_CALLER"

	// Fetch PolicyAt again and ensure global defaults remain untouched
	p2, err := PolicyAt(d)
	if err != nil {
		t.Fatalf("second PolicyAt failed: %v", err)
	}

	if p2.Brackets[0].RateNumerator != origNumerator {
		t.Errorf("PolicyAt leaked mutable global slice! Got %d, want %d",
			p2.Brackets[0].RateNumerator, origNumerator)
	}
	if p2.Ref.Sources[0] == "MODIFIED_BY_CALLER" {
		t.Errorf("PolicyAt leaked mutable sources slice!")
	}
}

func TestOverflow_BoundsChecks(t *testing.T) {
	opts := Options{
		Dependents: 0,
		Region:     Region1,
		Date:       date(2024, 8, 1),
	}

	// 1. Enormous gross amount (2^61)
	hugeGross := VND(int64(1) << 61)
	_, err := GrossToNet(hugeGross, opts)
	if !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("expected ErrAmountTooLarge for gross 2^61, got %v", err)
	}

	// 2. Gross exceeds MaxGross
	_, err = GrossToNet(MaxGross+1, opts)
	if !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("expected ErrAmountTooLarge for MaxGross+1, got %v", err)
	}

	// 3. Net exceeds MaxGross
	_, err = NetToGross(MaxGross+1, opts)
	if !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("expected ErrAmountTooLarge for net MaxGross+1, got %v", err)
	}

	// 4. Dependents exceed MaxDependents
	badOpts := opts
	badOpts.Dependents = MaxDependents + 1
	_, err = GrossToNet(20_000_000, badOpts)
	if !errors.Is(err, ErrInvalidDependents) {
		t.Errorf("expected ErrInvalidDependents for dependents > 50, got %v", err)
	}

	// 5. InsuranceBase exceeds MaxGross
	insBase := MaxGross + 1
	badOpts = opts
	badOpts.InsuranceBase = &insBase
	_, err = GrossToNet(20_000_000, badOpts)
	if !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("expected ErrAmountTooLarge for InsuranceBase > MaxGross, got %v", err)
	}
}

func TestInsuranceExceedsGross(t *testing.T) {
	insBase := VND(10_000_000) // Insurance 10.5% = 1,050,000 VND
	opts := Options{
		InsuranceBase: &insBase,
		Dependents:    0,
		Region:        Region1,
		Date:          date(2024, 8, 1),
	}

	// Gross salary is only 1,000,000 VND, mandatory insurance is 1,050,000 VND -> negative net
	_, err := GrossToNet(1_000_000, opts)
	if !errors.Is(err, ErrInsuranceExceedsGross) {
		t.Errorf("expected ErrInsuranceExceedsGross, got %v", err)
	}
}

func TestBracketBoundaries_PlusMinusOneVND(t *testing.T) {
	opts2024 := Options{
		Dependents: 0,
		Region:     Region1,
		Date:       date(2024, 8, 1),
	}

	// Verify monotonicity: increasing Gross by 1 VND never decreases Net
	for base := VND(10_000_000); base <= 100_000_000; base += 10_000_000 {
		for delta := VND(-5); delta <= 5; delta++ {
			g := base + delta
			r1, _ := GrossToNet(g, opts2024)
			r2, _ := GrossToNet(g+1, opts2024)
			if r2.Net < r1.Net {
				t.Fatalf("Monotonicity violated at gross %d: Net(%d)=%d > Net(%d)=%d",
					g, g, r1.Net, g+1, r2.Net)
			}
		}
	}
}

func TestRegionalUnemploymentCaps(t *testing.T) {
	// Gross salary 120M exceeds unemployment cap in all 4 regions
	gross := VND(120_000_000)
	date2024 := date(2024, 8, 1)

	expectedCaps := map[Region]VND{
		Region1: 4_960_000 * 20 / 100, // 992,000 VND
		Region2: 4_410_000 * 20 / 100, // 882,000 VND
		Region3: 3_860_000 * 20 / 100, // 772,000 VND
		Region4: 3_450_000 * 20 / 100, // 690,000 VND
	}

	for reg, expectedCap := range expectedCaps {
		opts := Options{Region: reg, Date: date2024}
		res, err := GrossToNet(gross, opts)
		if err != nil {
			t.Fatalf("Region %d failed: %v", reg, err)
		}
		if res.Insurance.Unemployment != expectedCap {
			t.Errorf("Region %d unemployment mismatch: got %d, want %d",
				reg, res.Insurance.Unemployment, expectedCap)
		}
	}
}

func TestTimezoneAndBoundaryDates(t *testing.T) {
	opts := Options{
		Dependents: 0,
		Region:     Region1,
	}

	// 2025-12-31 23:59:59 (Asia/Ho_Chi_Minh) -> Still governed by 2024 policy
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	d2025 := time.Date(2025, 12, 31, 23, 59, 59, 0, loc)
	opts.Date = d2025

	p2025, err := PolicyAt(opts.Date)
	if err != nil {
		t.Fatalf("PolicyAt 2025 failed: %v", err)
	}
	if p2025.PersonalDeduction != 11_000_000 {
		t.Errorf("expected 11M personal deduction on 2025-12-31, got %d", p2025.PersonalDeduction)
	}

	// 2026-01-01 00:00:00 (Asia/Ho_Chi_Minh) -> Governed by 2026 policy
	d2026 := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	opts.Date = d2026

	p2026, err := PolicyAt(opts.Date)
	if err != nil {
		t.Fatalf("PolicyAt 2026 failed: %v", err)
	}
	if p2026.PersonalDeduction != 15_500_000 {
		t.Errorf("expected 15.5M personal deduction on 2026-01-01, got %d", p2026.PersonalDeduction)
	}
}

func TestDefaultPolicies_Order(t *testing.T) {
	for i := 0; i < len(defaultPolicies)-1; i++ {
		if !defaultPolicies[i].EffectiveFrom.After(defaultPolicies[i+1].EffectiveFrom) {
			t.Errorf("defaultPolicies must be sorted descending by EffectiveFrom: index %d (%v) <= index %d (%v)",
				i, defaultPolicies[i].EffectiveFrom, i+1, defaultPolicies[i+1].EffectiveFrom)
		}
	}
}

func TestBadInputs(t *testing.T) {
	opts := Options{
		Dependents: 0,
		Region:     Region1,
		Date:       date(2024, 10, 1),
	}

	// 1. Negative Gross
	_, err := GrossToNet(-1, opts)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}

	// 2. Negative Net
	_, err = NetToGross(-1, opts)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount, got %v", err)
	}

	// 3. Dependents < 0
	badOpts := opts
	badOpts.Dependents = -1
	_, err = GrossToNet(10_000_000, badOpts)
	if !errors.Is(err, ErrInvalidDependents) {
		t.Errorf("expected ErrInvalidDependents, got %v", err)
	}

	// 4. Region outside [1, 4]
	badOpts = opts
	badOpts.Region = 5
	_, err = GrossToNet(10_000_000, badOpts)
	if !errors.Is(err, ErrInvalidRegion) {
		t.Errorf("expected ErrInvalidRegion, got %v", err)
	}

	// 5. Date prior to 2024-07-01
	badOpts = opts
	badOpts.Date = date(2024, 6, 30)
	_, err = GrossToNet(10_000_000, badOpts)
	if !errors.Is(err, ErrDateOutOfRange) {
		t.Errorf("expected ErrDateOutOfRange, got %v", err)
	}

	// 6. Zero Date
	badOpts = opts
	badOpts.Date = time.Time{}
	_, err = GrossToNet(10_000_000, badOpts)
	if !errors.Is(err, ErrDateZero) {
		t.Errorf("expected ErrDateZero, got %v", err)
	}
}

func TestRoundToThousand(t *testing.T) {
	if RoundToThousand(0) != 0 {
		t.Errorf("expected 0, got %d", RoundToThousand(0))
	}
	if RoundToThousand(14_499) != 14_000 {
		t.Errorf("expected 14,000, got %d", RoundToThousand(14_499))
	}
	if RoundToThousand(14_500) != 15_000 {
		t.Errorf("expected 15,000, got %d", RoundToThousand(14_500))
	}
}

func FuzzGrossToNet_NetToGross(f *testing.F) {
	// gross, dependents, region, policy index, overtime, meal, other exempt, bonus
	f.Add(int64(0), 0, 1, 0, int64(0), int64(0), int64(0), int64(0))
	f.Add(int64(15_000_000), 1, 1, 1, int64(0), int64(0), int64(0), int64(0))
	f.Add(int64(35_000_000), 1, 2, 0, int64(3_000_000), int64(900_000), int64(0), int64(0))
	f.Add(int64(80_000_000), 2, 1, 2, int64(0), int64(1_500_000), int64(500_000), int64(30_000_000))
	f.Add(int64(150_000_000), 0, 3, 1, int64(10_000_000), int64(0), int64(0), int64(80_000_000))

	dates := []time.Time{date(2025, 3, 10), date(2026, 3, 15), date(2026, 8, 5)}

	f.Fuzz(func(t *testing.T, gross int64, dependents, region, policy int, ot, meal, other, bonus int64) {
		if gross < 0 || gross > 500_000_000 || dependents < 0 || dependents > 10 ||
			region < 1 || region > 4 || policy < 0 || policy >= len(dates) {
			return
		}
		for _, v := range []int64{ot, meal, other, bonus} {
			if v < 0 || v > 200_000_000 {
				return
			}
		}
		opts := Options{
			Dependents: dependents,
			Region:     Region(region),
			Date:       dates[policy],
			Extras:     Extras{Overtime: VND(ot), MealCash: VND(meal), OtherExempt: VND(other), Bonus: VND(bonus)},
		}

		// Both insurance sets and both tax regimes, without changing the corpus format.
		opts.Foreigner = gross%2 == 1
		opts.NonResident = gross%4 >= 2

		resForward, err := GrossToNet(VND(gross), opts)
		if errors.Is(err, ErrExtrasExceedGross) {
			return
		}
		if err != nil {
			t.Fatalf("GrossToNet failed: %v", err)
		}

		resInverse, err := NetToGross(resForward.Net, opts)
		if err != nil {
			t.Fatalf("NetToGross failed for net %d: %v", resForward.Net, err)
		}
		if resInverse.Net != resForward.Net || resInverse.Gross > resForward.Gross {
			t.Fatalf("inverse mismatch: NetToGross(%d) = gross %d net %d, from gross %d",
				resForward.Net, resInverse.Gross, resInverse.Net, resForward.Gross)
		}
		// Non-resident nets can step down by 1 VND, so look further than one VND below.
		for g := max(resInverse.Gross-64, 0); g < resInverse.Gross; g++ {
			if below, err := GrossToNet(g, opts); err == nil && below.Net == resForward.Net {
				t.Fatalf("gross %d is not the smallest for net %d: %d gives it", resInverse.Gross, resForward.Net, g)
			}
		}
	})
}

func BenchmarkGrossToNet(b *testing.B) {
	opts := Options{
		Dependents: 1,
		Region:     Region1,
		Date:       date(2024, 10, 1),
	}
	gross := VND(45_000_000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GrossToNet(gross, opts)
	}
}

func BenchmarkNetToGross(b *testing.B) {
	opts := Options{
		Dependents: 1,
		Region:     Region1,
		Date:       date(2024, 10, 1),
	}
	net := VND(38_000_000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NetToGross(net, opts)
	}
}

// TestPoliciesInvariants verifies fundamental assumptions relied upon by the piecewise inverse algorithm.
func TestPoliciesInvariants(t *testing.T) {
	for i, p := range defaultPolicies {
		// 1. Descending order by effective date (policyAt selects first matching policy).
		if i > 0 && !p.EffectiveFrom.Before(defaultPolicies[i-1].EffectiveFrom) {
			t.Errorf("policy %s must be older than policy %s", p.Ref.EffectiveFrom, defaultPolicies[i-1].Ref.EffectiveFrom)
		}
		// 2. computeBreakpoints assumes social/health cap is below unemployment cap across all regions.
		for r := Region1; r <= Region4; r++ {
			if p.MaxSocialHealthSalary >= p.MaxUnemploymentSalary(r) {
				t.Errorf("policy %s region %d: social/health cap %d must be below unemployment cap %d",
					p.Ref.EffectiveFrom, r, p.MaxSocialHealthSalary, p.MaxUnemploymentSalary(r))
			}
		}
		// 3. Social/health cap = 20 x base salary.
		if p.MaxSocialHealthSalary != p.BaseSalary*20 {
			t.Errorf("policy %s: social/health cap %d != 20 x base %d", p.Ref.EffectiveFrom, p.MaxSocialHealthSalary, p.BaseSalary)
		}
		// 4. Tax bracket limits must increase monotonically, top bracket must be 35%.
		for j := 1; j < len(p.Brackets); j++ {
			if p.Brackets[j].Limit <= p.Brackets[j-1].Limit {
				t.Errorf("policy %s: bracket limits must increase", p.Ref.EffectiveFrom)
			}
		}
		last := p.Brackets[len(p.Brackets)-1]
		if last.RateNumerator*100/last.RateDenom != 35 {
			t.Errorf("policy %s: top bracket must be 35%%", p.Ref.EffectiveFrom)
		}
		// 5. Each policy must declare statutory decree sources.
		if len(p.Ref.Sources) == 0 {
			t.Errorf("policy %s has no sources", p.Ref.EffectiveFrom)
		}
		// 6. Deductions and brackets are set per tax period: Annual uses the
		// policy in force on 1 January for the whole year.
		if year := p.EffectiveFrom.Year(); year >= FirstAnnualYear {
			jan, _ := policyAt(date(year, 1, 1))
			if p.PersonalDeduction != jan.PersonalDeduction || p.DependentDeduction != jan.DependentDeduction ||
				p.HealthcareDeductionCap != jan.HealthcareDeductionCap || p.EducationDeductionCap != jan.EducationDeductionCap ||
				!slices.Equal(p.Brackets, jan.Brackets) {
				t.Errorf("policy %s changes deductions or brackets within tax year %d", p.Ref.EffectiveFrom, year)
			}
		}
	}
}

// TestBaseSalaryChange_2026_07 verifies social/health cap transition from 46.8M to 50.6M on 2026-07-01.
func TestBaseSalaryChange_2026_07(t *testing.T) {
	cases := []struct {
		date       time.Time
		wantSocial VND
		wantNet    VND
	}{
		// 60M, 0 dependents, Region I. Manual calculation:
		// 2026-06-30: Insurance = 46.8M*9.5% + 60M*1% = 4,446,000 + 600,000 = 5,046,000; Assessable = 39,454,000;
		//             Tax = 500,000 + 2,000,000 + 1,890,800 = 4,390,800; Net = 50,563,200 VND.
		{date(2026, 6, 30), 3_744_000, 50_563_200},
		// 2026-07-01: Insurance = 50.6M*9.5% + 600,000 = 4,807,000 + 600,000 = 5,407,000; Assessable = 39,093,000;
		//             Tax = 500,000 + 2,000,000 + 1,818,600 = 4,318,600; Net = 50,274,400 VND.
		{date(2026, 7, 1), 4_048_000, 50_274_400},
	}
	for _, c := range cases {
		res, err := GrossToNet(60_000_000, Options{Region: Region1, Date: c.date})
		if err != nil {
			t.Fatalf("%s: %v", c.date.Format("2006-01-02"), err)
		}
		if res.Insurance.Social != c.wantSocial || res.Net != c.wantNet {
			t.Errorf("%s: social=%d net=%d, want social=%d net=%d",
				c.date.Format("2006-01-02"), res.Insurance.Social, res.Net, c.wantSocial, c.wantNet)
		}
	}
}

// TestResultDoesNotLeakSharedData verifies mutating Result.Policy.Sources does not corrupt internal package state.
func TestResultDoesNotLeakSharedData(t *testing.T) {
	opts := Options{Region: Region1, Date: date(2026, 8, 1)}
	r1, _ := GrossToNet(20_000_000, opts)
	r1.Policy.Sources[0] = "tampered"
	r2, _ := NetToGross(r1.Net, opts)
	r2.Policy.Sources[0] = "tampered"
	r3, _ := GrossToNet(20_000_000, opts)
	if r3.Policy.Sources[0] == "tampered" {
		t.Fatal("Result.Policy.Sources shares memory with package data")
	}
}

// TestNetSteps_ZeroOrOne checks the property NetToGross relies on: raising
// gross by 1 VND changes net by 0 or +1 VND, so net never decreases and every
// value up to Net(MaxGross) is reachable. With the current rates (8%, 1.5%,
// 1%) no two insurance items round up at the same gross; a rate change in a
// new policy can break this and must be caught here.
func TestNetSteps_ZeroOrOne(t *testing.T) {
	const window = 3_000
	insBase := VND(5_310_000)
	for i := range defaultPolicies {
		pol := &defaultPolicies[i]
		for _, opts := range []Options{
			{Dependents: 0, Region: Region1, Date: pol.EffectiveFrom},
			{Dependents: 2, Region: Region4, Date: pol.EffectiveFrom},
			{Dependents: 1, Region: Region1, Date: pol.EffectiveFrom, InsuranceBase: &insBase},
			{Dependents: 1, Region: Region1, Date: pol.EffectiveFrom, Extras: Extras{Overtime: 3_333_333, MealCash: 1_500_000, OtherExempt: 777_777, Bonus: 12_345_678}},
			{Dependents: 0, Region: Region1, Date: pol.EffectiveFrom, Foreigner: true},
			{Dependents: 1, Region: Region2, Date: pol.EffectiveFrom, Foreigner: true, Extras: Extras{Overtime: 3_333_333, MealCash: 1_500_000, OtherExempt: 777_777, Bonus: 12_345_678}},
		} {
			// Walk a window around every breakpoint, where slopes and rounding change.
			for _, bp := range computeBreakpoints(pol, opts) {
				start := max(bp.gross-window/2, 0)
				prev, prevErr := computeNet(pol, start, opts)
				for g := start + 1; g < start+window; g++ {
					res, err := computeNet(pol, g, opts)
					if err != nil || prevErr != nil {
						prev, prevErr = res, err
						continue
					}
					if step := res.Net - prev.Net; step != 0 && step != 1 {
						t.Fatalf("policy %s, opts %+v: Net(%d)=%d, Net(%d)=%d (step %d)",
							pol.Ref.EffectiveFrom, opts, g-1, prev.Net, g, res.Net, step)
					}
					prev = res
				}
			}
		}
	}
}

// TestNetToGross_Unreachable: since net grows by at most 1 VND per VND of
// gross, the only unreachable nets are those above Net(MaxGross).
func TestNetToGross_Unreachable(t *testing.T) {
	opts := Options{Region: Region1, Date: date(2026, 8, 1)}
	top, err := GrossToNet(MaxGross, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := NetToGross(top.Net, opts); err != nil || res.Net != top.Net {
		t.Errorf("Net(MaxGross)=%d should be reachable: res=%d err=%v", top.Net, res.Net, err)
	}
	if _, err := NetToGross(top.Net+1, opts); !errors.Is(err, ErrNetUnreachable) {
		t.Errorf("net above Net(MaxGross): want ErrNetUnreachable, got %v", err)
	}
}

func TestWarnings_InsuranceSalaryBelowRegionalMinimum(t *testing.T) {
	low, atMin := VND(4_000_000), VND(5_310_000)
	tests := []struct {
		name    string
		gross   VND
		region  Region
		insBase *VND
		warn    bool
	}{
		{"gross below region I minimum", 3_000_000, Region1, nil, true},
		{"gross equal to minimum", 5_310_000, Region1, nil, false},
		{"gross above minimum", 35_000_000, Region1, nil, false},
		{"zero gross", 0, Region1, nil, false},
		{"region IV minimum is lower", 3_800_000, Region4, nil, false},
		{"insurance base below minimum, gross high", 30_000_000, Region1, &low, true},
		{"insurance base equal to minimum", 30_000_000, Region1, &atMin, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{Region: tt.region, Date: date(2026, 3, 15), InsuranceBase: tt.insBase}
			res, err := GrossToNet(tt.gross, opts)
			if err != nil {
				t.Fatal(err)
			}
			got := len(res.Warnings) == 1 && res.Warnings[0].Code == WarnInsuranceSalaryBelowRegionalMinimum
			if got != tt.warn || (!tt.warn && len(res.Warnings) != 0) {
				t.Errorf("warnings = %+v, want warning: %v", res.Warnings, tt.warn)
			}

			// NetToGross reports the same warning for the gross it finds.
			inv, err := NetToGross(res.Net, opts)
			if err != nil {
				t.Fatal(err)
			}
			if (len(inv.Warnings) > 0) != (len(res.Warnings) > 0) && inv.Gross == res.Gross {
				t.Errorf("NetToGross warnings = %+v, GrossToNet warnings = %+v", inv.Warnings, res.Warnings)
			}
		})
	}
}

func TestWarnings_DoNotChangeAmounts(t *testing.T) {
	opts := Options{Region: Region1, Date: date(2026, 3, 15)}
	res, err := GrossToNet(3_000_000, opts)
	if err != nil {
		t.Fatal(err)
	}
	// 3,000,000 × 10.5% insurance, no tax.
	if res.Insurance.Total != 315_000 || res.Tax != 0 || res.Net != 2_685_000 {
		t.Errorf("amounts changed: insurance=%d tax=%d net=%d", res.Insurance.Total, res.Tax, res.Net)
	}
}

func TestWarnings_OmittedFromJSONWhenEmpty(t *testing.T) {
	opts := Options{Region: Region1, Date: date(2026, 3, 15)}
	for _, tc := range []struct {
		gross VND
		want  bool
	}{{35_000_000, false}, {3_000_000, true}} {
		res, err := GrossToNet(tc.gross, opts)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(string(data), `"warnings"`); got != tc.want {
			t.Errorf("gross %d: JSON has warnings key = %v, want %v", tc.gross, got, tc.want)
		}
	}
}

func hasWarning(r Result, code string) bool {
	for _, w := range r.Warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestExtras_ZeroValueKeepsEarlierResults(t *testing.T) {
	for _, d := range []time.Time{date(2025, 3, 10), date(2026, 3, 15), date(2026, 8, 5)} {
		for _, g := range []VND{0, 3_000_000, 23_456_789, 60_000_000, 133_456_789} {
			opts := Options{Dependents: 1, Region: Region1, Date: d}
			res, err := GrossToNet(g, opts)
			if err != nil {
				t.Fatal(err)
			}
			if res.Exempt != nil || res.BonusTax != 0 || res.TaxableIncome != g {
				t.Errorf("%s gross %d: extras fields set without extras: %+v", d.Format("2006-01-02"), g, res)
			}
			data, _ := json.Marshal(res)
			if strings.Contains(string(data), `"exempt"`) || strings.Contains(string(data), `"bonus_tax"`) {
				t.Errorf("JSON gained extras keys without extras: %s", data)
			}
		}
	}
}

func TestExtras_Validation(t *testing.T) {
	base := Options{Region: Region1, Date: date(2026, 8, 5)}
	tests := []struct {
		name  string
		gross VND
		ex    Extras
		want  error
	}{
		{"negative overtime", 20_000_000, Extras{Overtime: -1}, ErrInvalidAmount},
		{"negative bonus", 20_000_000, Extras{Bonus: -1}, ErrInvalidAmount},
		{"too large", 20_000_000, Extras{MealCash: MaxGross + 1}, ErrAmountTooLarge},
		{"extras exceed gross", 20_000_000, Extras{Bonus: 15_000_000, Overtime: 5_000_001}, ErrExtrasExceedGross},
	}
	for _, tt := range tests {
		opts := base
		opts.Extras = tt.ex
		if _, err := GrossToNet(tt.gross, opts); !errors.Is(err, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, err, tt.want)
		}
	}
	// Extras equal to gross is valid (the whole payment is extras).
	opts := base
	opts.Extras = Extras{Bonus: 20_000_000}
	if _, err := GrossToNet(20_000_000, opts); err != nil {
		t.Errorf("extras == gross: %v", err)
	}
}

func TestExtras_InsuranceSalaryAndExemptions(t *testing.T) {
	opts := Options{Region: Region1, Date: date(2026, 8, 5),
		Extras: Extras{Overtime: 5_000_000, MealCash: 1_500_000, OtherExempt: 800_000, Bonus: 2_000_000}}
	res, err := GrossToNet(40_000_000, opts)
	if err != nil {
		t.Fatal(err)
	}
	// Insurance on 40M - 5M overtime - 2M bonus = 33M: 33M × 10.5%.
	if res.Insurance.Total != 3_465_000 {
		t.Errorf("insurance = %d, want 3,465,000", res.Insurance.Total)
	}
	// Exempt: overtime 5M (2026) + meal capped at 1.2M + other 0.8M = 7M.
	want := ExemptIncome{Overtime: 5_000_000, Meal: 1_200_000, Other: 800_000, Total: 7_000_000}
	if res.Exempt == nil || *res.Exempt != want {
		t.Errorf("exempt = %+v, want %+v", res.Exempt, want)
	}
	if res.TaxableIncome != 33_000_000 {
		t.Errorf("taxable = %d, want 33,000,000", res.TaxableIncome)
	}
}

func TestExtras_BonusTax(t *testing.T) {
	d := date(2026, 8, 5)
	withBonus, err := GrossToNet(50_000_000, Options{Region: Region1, Date: d, Extras: Extras{Bonus: 20_000_000}})
	if err != nil {
		t.Fatal(err)
	}
	salaryOnly, err := GrossToNet(30_000_000, Options{Region: Region1, Date: d})
	if err != nil {
		t.Fatal(err)
	}
	// The bonus does not change insurance, so BonusTax is the difference in tax.
	if withBonus.Insurance != salaryOnly.Insurance {
		t.Errorf("bonus changed insurance: %+v vs %+v", withBonus.Insurance, salaryOnly.Insurance)
	}
	if withBonus.BonusTax != withBonus.Tax-salaryOnly.Tax || withBonus.BonusTax != 2_135_000 {
		t.Errorf("BonusTax = %d, want %d (= 2,135,000)", withBonus.BonusTax, withBonus.Tax-salaryOnly.Tax)
	}
}

func TestExtras_Warnings(t *testing.T) {
	tests := []struct {
		name    string
		date    time.Time
		ex      Extras
		insBase *VND
		want    []string
		notWant []string
	}{
		{"overtime before 2026 is taxed", date(2025, 11, 10), Extras{Overtime: 4_000_000}, nil,
			[]string{WarnOvertimeTaxedBeforeExemption}, nil},
		{"overtime in 2026 is exempt", date(2026, 3, 15), Extras{Overtime: 4_000_000}, nil,
			nil, []string{WarnOvertimeTaxedBeforeExemption}},
		{"meal cap unverified before 2026-07", date(2026, 6, 30), Extras{MealCash: 900_000}, nil,
			[]string{WarnMealCapUnverified, WarnInsuranceSalaryIncludesAllowances}, nil},
		{"meal cap verified from 2026-07", date(2026, 7, 1), Extras{MealCash: 900_000}, nil,
			[]string{WarnInsuranceSalaryIncludesAllowances}, []string{WarnMealCapUnverified}},
		{"explicit insurance base silences inference warning", date(2026, 7, 1), Extras{OtherExempt: 500_000}, ptrVND(20_000_000),
			nil, []string{WarnInsuranceSalaryIncludesAllowances}},
	}
	for _, tt := range tests {
		res, err := GrossToNet(30_000_000, Options{Region: Region1, Date: tt.date, Extras: tt.ex, InsuranceBase: tt.insBase})
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		for _, c := range tt.want {
			if !hasWarning(res, c) {
				t.Errorf("%s: missing warning %s in %+v", tt.name, c, res.Warnings)
			}
		}
		for _, c := range tt.notWant {
			if hasWarning(res, c) {
				t.Errorf("%s: unexpected warning %s", tt.name, c)
			}
		}
	}
	// Overtime before 2026 is taxed: same tax as plain salary of the same gross.
	ot, _ := GrossToNet(30_000_000, Options{Region: Region1, Date: date(2025, 11, 10), Extras: Extras{Overtime: 4_000_000}})
	if ot.Exempt.Overtime != 0 || ot.TaxableIncome != 30_000_000 {
		t.Errorf("overtime before 2026 should not be exempt: %+v", ot.Exempt)
	}
}

func ptrVND(v VND) *VND { return &v }

// TestNetToGross_WithExtras: the inverse holds the extras fixed and returns
// the smallest total gross with the requested net.
func TestNetToGross_WithExtras(t *testing.T) {
	ib := VND(10_000_000)
	for _, opts := range []Options{
		{Region: Region1, Date: date(2026, 8, 5), Extras: Extras{Overtime: 5_000_000, MealCash: 1_500_000, OtherExempt: 800_000}},
		{Dependents: 2, Region: Region1, Date: date(2026, 12, 25), Extras: Extras{Bonus: 80_000_000}},
		{Region: Region4, Date: date(2025, 11, 10), Extras: Extras{Overtime: 4_000_000, MealCash: 900_000}},
		{Dependents: 1, Region: Region2, Date: date(2026, 10, 5), InsuranceBase: &ib, Extras: Extras{MealCash: 1_000_000, Bonus: 5_000_000}},
	} {
		minGross := opts.Extras.total()
		for g := minGross; g < minGross+300_000_000; g += 7_777_777 {
			want, err := GrossToNet(g, opts)
			if err != nil {
				continue // e.g. insurance on a fixed base exceeds a small gross
			}
			got, err := NetToGross(want.Net, opts)
			if err != nil {
				t.Fatalf("%+v net %d: %v", opts.Extras, want.Net, err)
			}
			if got.Net != want.Net || got.Gross > g {
				t.Fatalf("%+v: NetToGross(%d) = gross %d net %d, gross %d gives that net", opts.Extras, want.Net, got.Gross, got.Net, g)
			}
			if below, err := GrossToNet(got.Gross-1, opts); err == nil && below.Net == want.Net {
				t.Fatalf("%+v: gross %d not minimal", opts.Extras, got.Gross)
			}
		}
	}
}

// Foreign nationals pay no unemployment insurance (employee or employer);
// social and health insurance are unchanged.
func TestForeigner_NoUnemploymentInsurance(t *testing.T) {
	for _, d := range []time.Time{date(2025, 3, 10), date(2026, 3, 10), date(2026, 8, 10)} {
		vn, err := GrossToNet(60_000_000, Options{Region: Region1, Date: d})
		if err != nil {
			t.Fatal(err)
		}
		fn, err := GrossToNet(60_000_000, Options{Region: Region1, Date: d, Foreigner: true})
		if err != nil {
			t.Fatal(err)
		}
		if fn.Insurance.Unemployment != 0 || fn.Employer.Unemployment != 0 {
			t.Errorf("%s: unemployment insurance %d / employer %d, want 0", d.Format(time.DateOnly), fn.Insurance.Unemployment, fn.Employer.Unemployment)
		}
		if fn.Insurance.Social != vn.Insurance.Social || fn.Insurance.Health != vn.Insurance.Health ||
			fn.Employer.Social != vn.Employer.Social || fn.Employer.Health != vn.Employer.Health {
			t.Errorf("%s: social or health insurance differs for a foreigner", d.Format(time.DateOnly))
		}
		if fn.Insurance.Total != vn.Insurance.Total-vn.Insurance.Unemployment {
			t.Errorf("%s: employee total %d, want %d", d.Format(time.DateOnly), fn.Insurance.Total, vn.Insurance.Total-vn.Insurance.Unemployment)
		}
		inv, err := NetToGross(fn.Net, Options{Region: Region1, Date: d, Foreigner: true})
		if err != nil || inv.Gross != 60_000_000 {
			t.Errorf("%s: NetToGross(%d) = %d, %v; want 60,000,000", d.Format(time.DateOnly), fn.Net, inv.Gross, err)
		}
	}
}

// Non-residents pay 20% of taxable income, without deductions; insurance is
// unchanged and not deducted (Decree 253/2026/NĐ-CP Art. 64).
func TestNonResident_FlatRate(t *testing.T) {
	opts := Options{Region: Region1, Date: date(2026, 8, 10), Foreigner: true, NonResident: true, Dependents: 2}
	res, err := GrossToNet(60_000_000, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Insurance.Total != 4_807_000 || res.Tax != 12_000_000 || res.Net != 43_193_000 {
		t.Errorf("got insurance %d, tax %d, net %d; want 4,807,000, 12,000,000, 43,193,000", res.Insurance.Total, res.Tax, res.Net)
	}
	if res.Deductions != (Deductions{}) || res.AssessableIncome != 60_000_000 {
		t.Errorf("got deductions %+v, assessable %d", res.Deductions, res.AssessableIncome)
	}
	if len(res.Brackets) != 1 || res.Brackets[0].RatePercent != 20 {
		t.Errorf("got brackets %+v, want one line at 20%%", res.Brackets)
	}
	if !hasWarning(res, WarnDependentsIgnoredNonResident) {
		t.Errorf("want %s warning, got %v", WarnDependentsIgnoredNonResident, res.Warnings)
	}

	// A bonus costs exactly 20% of itself.
	opts.Extras = Extras{Bonus: 20_000_000}
	res, err = GrossToNet(80_000_000, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.BonusTax != 4_000_000 || res.Insurance.Total != 4_807_000 {
		t.Errorf("got bonus tax %d, insurance %d; want 4,000,000, 4,807,000", res.BonusTax, res.Insurance.Total)
	}

	// Before 2026 the rate is also 20% (Law on PIT 2007 Art. 26).
	old, err := GrossToNet(30_000_000, Options{Region: Region1, Date: date(2025, 5, 5), NonResident: true})
	if err != nil {
		t.Fatal(err)
	}
	if old.Tax != 6_000_000 {
		t.Errorf("2025: tax %d, want 6,000,000", old.Tax)
	}
}

// With extras, the tax base (gross minus exempt income) and the insurance base
// (gross minus overtime and bonus) are offset, and a flat rate on the first
// does not shrink when insurance grows: net can step down by 1 VND. NetToGross
// must still return the smallest gross with the requested net.
func TestNonResident_NetStepsAndSmallestGross(t *testing.T) {
	for _, ex := range []Extras{{}, {Bonus: 1}, {MealCash: 3}, {OtherExempt: 4, Overtime: 7}, {Overtime: 3_333_333, MealCash: 1_500_000, OtherExempt: 777_777, Bonus: 12_345_678}} {
		opts := Options{Region: Region1, Date: date(2026, 8, 10), NonResident: true, Extras: ex}
		base := ex.total() + 20_000_000
		prev, err := computeNet(&defaultPolicies[0], base, opts)
		if err != nil {
			t.Fatal(err)
		}
		for g := base + 1; g < base+2_000; g++ {
			res, err := computeNet(&defaultPolicies[0], g, opts)
			if err != nil {
				t.Fatal(err)
			}
			if step := res.Net - prev.Net; step < -1 || step > 1 {
				t.Fatalf("extras %+v: Net(%d)=%d, Net(%d)=%d (step %d)", ex, g-1, prev.Net, g, res.Net, step)
			}
			prev = res

			inv, err := NetToGross(res.Net, opts)
			if err != nil {
				t.Fatalf("extras %+v: NetToGross(%d): %v", ex, res.Net, err)
			}
			if inv.Net != res.Net || inv.Gross > g {
				t.Fatalf("extras %+v: NetToGross(%d) = gross %d net %d, from gross %d", ex, res.Net, inv.Gross, inv.Net, g)
			}
			for below := inv.Gross - 64; below < inv.Gross; below++ {
				if r, err := computeNet(&defaultPolicies[0], below, opts); err == nil && r.Net == res.Net {
					t.Fatalf("extras %+v: gross %d is not the smallest for net %d: %d gives it", ex, inv.Gross, res.Net, below)
				}
			}
		}
	}
}

// smallestGross must cross a 1 VND step down of net, which a walk that stops
// at the first different net does not.
func TestSmallestGross_CrossesStepDown(t *testing.T) {
	opts := Options{Region: Region1, Date: date(2026, 8, 10), NonResident: true, Extras: Extras{Bonus: 1}}
	netAt := func(g VND) VND {
		res, err := computeNet(&defaultPolicies[0], g, opts)
		if err != nil {
			t.Fatal(err)
		}
		return res.Net
	}
	for g := VND(30_000_000); g < 30_002_000; g++ {
		n := netAt(g)
		if netAt(g-1) != n+1 || netAt(g-2) != n {
			continue // not a step down that a contiguous walk would stop at
		}
		same := func(x VND) bool { return netAt(x) == n }
		if got := smallestGross(g, 1, same); got != g {
			t.Fatalf("window 1 crossed the step: got %d", got)
		}
		got := smallestGross(g, nonResidentWindow, same)
		if got > g-2 {
			t.Fatalf("Net(%d) = Net(%d) = %d but smallestGross returned %d", g-2, g, n, got)
		}
		inv, err := NetToGross(n, opts)
		if err != nil || inv.Gross != got {
			t.Fatalf("NetToGross(%d) = %d, %v; want %d", n, inv.Gross, err, got)
		}
		return
	}
	t.Fatal("no 1 VND step down found; the test no longer exercises the case")
}
