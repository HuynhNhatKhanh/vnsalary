package vnsalary

import (
	"errors"
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
	f.Add(int64(0), 0, 1)
	f.Add(int64(15_000_000), 1, 1)
	f.Add(int64(35_000_000), 1, 2)
	f.Add(int64(80_000_000), 2, 1)
	f.Add(int64(150_000_000), 0, 3)

	opts := Options{
		Region: Region1,
		Date:   date(2026, 1, 15),
	}

	f.Fuzz(func(t *testing.T, gross int64, dependents int, region int) {
		if gross < 0 || gross > 500_000_000 {
			return
		}
		if dependents < 0 || dependents > 10 {
			return
		}
		if region < 1 || region > 4 {
			return
		}

		fOpts := opts
		fOpts.Dependents = dependents
		fOpts.Region = Region(region)

		resForward, err := GrossToNet(VND(gross), fOpts)
		if err != nil {
			t.Fatalf("GrossToNet failed: %v", err)
		}

		resInverse, err := NetToGross(resForward.Net, fOpts)
		if err != nil {
			t.Fatalf("NetToGross failed for net %d: %v", resForward.Net, err)
		}

		if resInverse.Net != resForward.Net {
			t.Fatalf("Fuzz Net mismatch: got %d, want %d", resInverse.Net, resForward.Net)
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
