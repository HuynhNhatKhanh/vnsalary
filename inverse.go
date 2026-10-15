package vnsalary

import (
	"sort"
)

// breakpoint represents a point where the slope changes (insurance cap reached or tax bracket transitions).
type breakpoint struct {
	gross VND
	net   VND
}

// computeBreakpoints calculates the set of breakpoint thresholds of Gross -> Net in gross space.
func computeBreakpoints(policy *Policy, opts Options) []breakpoint {
	var grossPoints []VND
	grossPoints = append(grossPoints, 0)

	// Statutory insurance caps when contributions are based on gross salary
	if opts.InsuranceBase == nil {
		grossPoints = append(grossPoints, policy.MaxSocialHealthSalary)
		grossPoints = append(grossPoints, policy.MaxUnemploymentSalary(opts.Region))
	}

	totalDeductions := policy.PersonalDeduction + policy.DependentDeduction*VND(opts.Dependents)

	// Threshold 1: Point where personal income tax begins (income after insurance exceeds family relief deductions)
	// AssessableIncome = 0 => PreTax = totalDeductions
	if totalDeductions > 0 {
		var gZeroTax VND
		if opts.InsuranceBase != nil {
			insSalary := *opts.InsuranceBase
			baseSH := minVND(insSalary, policy.MaxSocialHealthSalary)
			baseU := minVND(insSalary, policy.MaxUnemploymentSalary(opts.Region))
			totalIns := roundDiv(int64(baseSH)*95, 1000) + roundDiv(int64(baseU)*1, 100)
			gZeroTax = totalDeductions + totalIns
		} else {
			capSH := policy.MaxSocialHealthSalary
			capU := policy.MaxUnemploymentSalary(opts.Region)
			g1 := roundDiv(int64(totalDeductions)*1000, 895)
			if g1 <= capSH && g1 <= capU {
				gZeroTax = g1
			} else {
				insSH := roundDiv(int64(capSH)*95, 1000)
				g2 := roundDiv(int64(totalDeductions+insSH)*100, 99)
				if g2 <= capU {
					gZeroTax = g2
				} else {
					insU := roundDiv(int64(capU)*1, 100)
					gZeroTax = totalDeductions + insSH + insU
				}
			}
		}
		if gZeroTax > 0 {
			grossPoints = append(grossPoints, gZeroTax)
		}
	}

	// Thresholds 2+: Progressive tax bracket transitions
	for _, b := range policy.Brackets {
		if b.Limit > 1_000_000_000_000 { // Ignore MaxInt64
			continue
		}

		targetPreTax := b.Limit + totalDeductions

		var gEstimate VND
		if opts.InsuranceBase != nil {
			insSalary := *opts.InsuranceBase
			baseSH := minVND(insSalary, policy.MaxSocialHealthSalary)
			baseU := minVND(insSalary, policy.MaxUnemploymentSalary(opts.Region))
			totalIns := roundDiv(int64(baseSH)*95, 1000) + roundDiv(int64(baseU)*1, 100)
			gEstimate = targetPreTax + totalIns
		} else {
			capSH := policy.MaxSocialHealthSalary
			capU := policy.MaxUnemploymentSalary(opts.Region)

			g1 := roundDiv(int64(targetPreTax)*1000, 895)
			if g1 <= capSH && g1 <= capU {
				gEstimate = g1
			} else {
				insSH := roundDiv(int64(capSH)*95, 1000)
				g2 := roundDiv(int64(targetPreTax+insSH)*100, 99)
				if g2 <= capU {
					gEstimate = g2
				} else {
					insU := roundDiv(int64(capU)*1, 100)
					gEstimate = targetPreTax + insSH + insU
				}
			}
		}

		if gEstimate > 0 {
			grossPoints = append(grossPoints, gEstimate)
		}
	}

	// Deduplicate and sort in ascending order
	sort.Slice(grossPoints, func(i, j int) bool {
		return grossPoints[i] < grossPoints[j]
	})

	var grossList []VND
	for i, g := range grossPoints {
		if i == 0 || g != grossPoints[i-1] {
			grossList = append(grossList, g)
		}
	}

	// Evaluate corresponding Net at each breakpoint
	bps := make([]breakpoint, 0, len(grossList))
	for _, g := range grossList {
		res, err := computeNet(policy, g, opts)
		if err != nil {
			// For instance, gross is too low compared to fixed InsuranceBase -> skip point.
			continue
		}
		bps = append(bps, breakpoint{gross: g, net: res.Net})
	}

	return bps
}

// NetToGross solves the minimum integer Gross salary that yields the exact Net salary.
//
// Step 1: Analytical piecewise linear interpolation on the interval [bps[i], bps[i+1]] containing net to derive gApprox.
// Step 2: Ascending scan within +/- searchRadius (50 VND) around gApprox; the first match is then
// extended downward to the smallest gross with the same net.
// Step 3 (Fallback): Monotonic binary search on [segmentLow, segmentHigh] to guarantee complete coverage.
func NetToGross(net VND, opts Options) (Result, error) {
	if net < 0 {
		return Result{}, ErrInvalidAmount
	}
	if net > MaxGross {
		return Result{}, ErrAmountTooLarge
	}
	if err := validateOptions(opts); err != nil {
		return Result{}, err
	}

	policy, err := policyAt(opts.Date)
	if err != nil {
		return Result{}, err
	}

	finish := func(g VND) (Result, error) {
		res, err := computeNet(policy, g, opts)
		if err != nil {
			return Result{}, err
		}
		return finalize(policy, res, opts), nil
	}

	if net == 0 {
		return finish(0)
	}

	// netAt evaluates Net for a candidate Gross g; ok=false if g causes invalid state (e.g. insurance exceeds gross).
	netAt := func(g VND) (VND, bool) {
		res, err := computeNet(policy, g, opts)
		if err != nil {
			return 0, false
		}
		return res.Net, true
	}

	// leftmost walks down from a gross g with Net(g) == net to the smallest such
	// gross. Net is non-decreasing in gross (TestNetSteps_ZeroOrOne), so the
	// grosses sharing one net form a single interval of a few VND.
	leftmost := func(g VND) VND {
		for g > 0 {
			if n, ok := netAt(g - 1); !ok || n != net {
				break
			}
			g--
		}
		return g
	}

	bps := computeBreakpoints(policy, opts)

	// 1. Interpolate across the interval [bps[i], bps[i+1]] containing net.
	var gApprox, segmentLow, segmentHigh VND
	switch {
	case len(bps) == 0:
		// No valid breakpoints (e.g. very high fixed InsuranceBase): fallback to full range binary search.
		gApprox, segmentLow, segmentHigh = 0, 0, MaxGross
	case net <= bps[0].net:
		gApprox, segmentLow, segmentHigh = 0, 0, bps[0].gross
	case net >= bps[len(bps)-1].net:
		last := bps[len(bps)-1]
		// Beyond the final breakpoint: insurance capped, top tax bracket at 35% -> slope is 65%.
		gApprox = last.gross + roundDiv(int64(net-last.net)*100, 65)
		segmentLow, segmentHigh = last.gross, MaxGross
	default:
		for i := 0; i < len(bps)-1; i++ {
			if net >= bps[i].net && net <= bps[i+1].net {
				spanNet := bps[i+1].net - bps[i].net
				spanGross := bps[i+1].gross - bps[i].gross
				gApprox = bps[i].gross
				if spanNet != 0 {
					gApprox += roundDiv(int64(net-bps[i].net)*int64(spanGross), int64(spanNet))
				}
				segmentLow, segmentHigh = bps[i].gross, bps[i+1].gross
				break
			}
		}
	}

	// 2. Ascending scan around gApprox; leftmost extends the first match below the window if needed.
	const searchRadius = VND(50)
	start := max(gApprox-searchRadius, 0)
	end := min(gApprox+searchRadius, MaxGross)
	for g := start; g <= end; g++ {
		if n, ok := netAt(g); ok && n == net {
			return finish(leftmost(g))
		}
	}

	// 3. Fallback: Monotonic binary search for the smallest gross with net == target.
	low, high := segmentLow, min(segmentHigh, MaxGross)
	found := VND(-1)
	for low <= high {
		mid := low + (high-low)/2
		n, ok := netAt(mid)
		switch {
		case !ok || n < net:
			low = mid + 1
		case n == net:
			found = mid
			high = mid - 1
		default:
			high = mid - 1
		}
	}
	if found != -1 {
		return finish(leftmost(found))
	}

	return Result{}, ErrNetUnreachable
}
