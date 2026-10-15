package vnsalary

import (
	"math"
	"time"
)

// defaultPolicies must be sorted descending by EffectiveFrom (newest policy first).
// Verified by TestPoliciesInvariants in vnsalary_test.go.
var defaultPolicies = []Policy{
	// =========================================================================
	// POLICY 3: EFFECTIVE FROM 2026-07-01 (BASE SALARY 2.53M -> CEILING 50.6M)
	// Other parameters remain identical to the 2026-01-01 regime.
	// =========================================================================
	{
		EffectiveFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Ref: PolicyRef{
			EffectiveFrom: "2026-07-01",
			Description:   "Policy from 2026-07-01: Base salary 2.53M VND (social/health cap 50.6M VND), 5 progressive brackets, deductions 15.5M/6.2M",
			DescriptionVI: "Chính sách từ 01/07/2026: Lương cơ sở 2,53M (trần BHXH/BHYT 50,6M), biểu thuế 5 bậc, giảm trừ 15,5M/6,2M",
			Sources: []string{
				"Nghị định 161/2026/NĐ-CP Điều 3 khoản 2 (Lương cơ sở 2.530.000đ từ 01/07/2026)",
				"Luật BHXH số 41/2024/QH15 Điều 31 khoản 1 điểm đ, Điều 141 khoản 13 (Trần BHXH 20 lần mức tham chiếu; mức tham chiếu bằng lương cơ sở)",
				"Luật BHYT Điều 14 khoản 5, sửa đổi bởi Luật số 51/2024/QH15 (Trần BHYT 20 lần mức tham chiếu)",
				"Luật Việc làm số 74/2025/QH15 Điều 34 khoản 2 (Trần BHTN 20 lần lương tối thiểu vùng)",
				"Nghị định 293/2025/NĐ-CP Điều 3 khoản 1 (Lương tối thiểu vùng từ 01/01/2026)",
				"Nghị quyết 110/2025/UBTVQH15 (Giảm trừ bản thân 15,5M, người phụ thuộc 6,2M)",
				"Luật Thuế TNCN số 109/2025/QH15 Điều 9 khoản 2, Điều 29 khoản 2 (Biểu thuế 5 bậc với tiền lương từ kỳ tính thuế 2026)",
				"Luật Thuế TNCN số 109/2025/QH15 Điều 4 khoản 8 & Nghị định 253/2026/NĐ-CP Điều 26 (Miễn thuế toàn bộ tiền lương làm thêm giờ, làm đêm trong giới hạn luật)",
				"Nghị định 253/2026/NĐ-CP Điều 8 khoản 2 điểm g, Điều 69 khoản 1 điểm b (Tiền ăn giữa ca bằng tiền chỉ tính thuế phần vượt 1.200.000đ/tháng, từ 01/07/2026)",
			},
		},
		BaseSalary:            2_530_000,
		MaxSocialHealthSalary: 2_530_000 * 20, // 50,600,000 VND
		RegionalMinSalaries: [5]VND{
			0,
			5_310_000, // Region I (unemployment cap = 106,200,000 VND)
			4_730_000, // Region II (unemployment cap = 94,600,000 VND)
			4_140_000, // Region III (unemployment cap = 82,800,000 VND)
			3_700_000, // Region IV (unemployment cap = 74,000,000 VND)
		},
		PersonalDeduction:  15_500_000,
		DependentDeduction: 6_200_000,
		Brackets: []TaxBracket{
			{Limit: 10_000_000, RateNumerator: 5, RateDenom: 100},
			{Limit: 30_000_000, RateNumerator: 10, RateDenom: 100},
			{Limit: 60_000_000, RateNumerator: 20, RateDenom: 100},
			{Limit: 100_000_000, RateNumerator: 30, RateDenom: 100},
			{Limit: math.MaxInt64, RateNumerator: 35, RateDenom: 100},
		},
		MealCashCap:         1_200_000,
		MealCashCapVerified: true,
		OvertimeFullyExempt: true,
	},

	// =========================================================================
	// POLICY 2: EFFECTIVE FROM 2026-01-01 TO 2026-06-30 (TAX YEAR 2026)
	// =========================================================================
	{
		EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Ref: PolicyRef{
			EffectiveFrom: "2026-01-01",
			Description:   "Policy for tax year 2026: 5-bracket progressive tax schedule and updated family relief deductions",
			DescriptionVI: "Chính sách từ kỳ tính thuế 2026: Biểu thuế lũy tiến 5 bậc và mức giảm trừ gia cảnh mới",
			Sources: []string{
				"Nghị quyết 110/2025/UBTVQH15 (Giảm trừ bản thân 15,5M, người phụ thuộc 6,2M từ kỳ tính thuế 2026)",
				"Luật Thuế TNCN số 109/2025/QH15 Điều 9 khoản 2, Điều 29 khoản 2 (Biểu thuế 5 bậc với tiền lương từ kỳ tính thuế 2026)",
				"Nghị định 293/2025/NĐ-CP Điều 3 khoản 1 (Lương tối thiểu vùng từ 01/01/2026)",
				"Nghị định 73/2024/NĐ-CP (Lương cơ sở 2.340.000đ)",
				"Luật BHXH số 41/2024/QH15 Điều 31 khoản 1 điểm đ, Điều 141 khoản 13 (Trần BHXH 20 lần mức tham chiếu; mức tham chiếu bằng lương cơ sở)",
				"Luật BHYT Điều 14 khoản 5, sửa đổi bởi Luật số 51/2024/QH15 (Trần BHYT 20 lần mức tham chiếu)",
				"Luật Việc làm số 74/2025/QH15 Điều 34 khoản 2 (Trần BHTN 20 lần lương tối thiểu vùng, hiệu lực 01/01/2026)",
				"Luật Thuế TNCN số 109/2025/QH15 Điều 4 khoản 8 & Nghị định 253/2026/NĐ-CP Điều 26, Điều 69 khoản 1 (Miễn thuế toàn bộ tiền lương làm thêm giờ, làm đêm từ kỳ tính thuế 2026)",
				"Tiền ăn giữa ca 730.000đ/tháng theo mức tham chiếu Thông tư 26/2016/TT-BLĐTBXH — ⚠️ chưa xác nhận cho 01–06/2026 (thông tư đã bị bãi bỏ)",
			},
		},
		BaseSalary:            2_340_000,
		MaxSocialHealthSalary: 2_340_000 * 20, // 46,800,000 VND
		RegionalMinSalaries: [5]VND{
			0,
			5_310_000, // Region I (unemployment cap = 106,200,000 VND)
			4_730_000, // Region II (unemployment cap = 94,600,000 VND)
			4_140_000, // Region III (unemployment cap = 82,800,000 VND)
			3_700_000, // Region IV (unemployment cap = 74,000,000 VND)
		},
		PersonalDeduction:  15_500_000,
		DependentDeduction: 6_200_000,
		Brackets: []TaxBracket{
			{Limit: 10_000_000, RateNumerator: 5, RateDenom: 100},
			{Limit: 30_000_000, RateNumerator: 10, RateDenom: 100},
			{Limit: 60_000_000, RateNumerator: 20, RateDenom: 100},
			{Limit: 100_000_000, RateNumerator: 30, RateDenom: 100},
			{Limit: math.MaxInt64, RateNumerator: 35, RateDenom: 100},
		},
		MealCashCap:         730_000,
		OvertimeFullyExempt: true,
	},

	// =========================================================================
	// POLICY 1: EFFECTIVE FROM 2024-07-01 TO 2025-12-31
	// =========================================================================
	{
		EffectiveFrom: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		Ref: PolicyRef{
			EffectiveFrom: "2024-07-01",
			Description:   "Policy from 2024-07-01: Base salary 2.34M VND, regional minimum wages under Decree 74/2024/ND-CP, 7-bracket progressive tax",
			DescriptionVI: "Chính sách từ 01/07/2024: Lương cơ sở 2,34M, lương tối thiểu vùng theo NĐ 74/2024, biểu thuế 7 bậc",
			Sources: []string{
				"Nghị định 73/2024/NĐ-CP (Lương cơ sở 2.340.000đ từ 01/07/2024)",
				"Nghị định 74/2024/NĐ-CP (Lương tối thiểu vùng từ 01/07/2024)",
				"Nghị quyết 954/2020/UBTVQH14 (Giảm trừ bản thân 11M, người phụ thuộc 4,4M)",
				"Luật BHXH 2014 Điều 89 khoản 3 & Luật BHYT 2008 Điều 14 khoản 2 (Trần BHXH/BHYT 20 lần lương cơ sở, tới 30/06/2025)",
				"Luật BHXH số 41/2024/QH15 Điều 31 khoản 1 điểm đ, Điều 141 khoản 13 & Luật BHYT Điều 14 khoản 5 sửa đổi bởi Luật số 51/2024/QH15 (cùng mức trần, từ 01/07/2025)",
				"Luật Việc làm 2013 Điều 58 (Trần BHTN 20 lần lương tối thiểu vùng)",
				"Luật Thuế TNCN 2007 Điều 22, sửa đổi bổ sung (Biểu thuế lũy tiến 7 bậc)",
				"Thông tư 111/2013/TT-BTC Điều 3 khoản 1 điểm i (Chỉ miễn phần tiền lương làm thêm giờ cao hơn giờ làm việc bình thường)",
				"Tiền ăn giữa ca 730.000đ/tháng theo mức tham chiếu Thông tư 26/2016/TT-BLĐTBXH — ⚠️ chưa xác nhận ngày bãi bỏ",
			},
		},
		BaseSalary:            2_340_000,
		MaxSocialHealthSalary: 2_340_000 * 20, // 46,800,000 VND
		RegionalMinSalaries: [5]VND{
			0,
			4_960_000, // Region I (unemployment cap = 99,200,000 VND)
			4_410_000, // Region II (unemployment cap = 88,200,000 VND)
			3_860_000, // Region III (unemployment cap = 77,200,000 VND)
			3_450_000, // Region IV (unemployment cap = 69,000,000 VND)
		},
		PersonalDeduction:  11_000_000,
		DependentDeduction: 4_400_000,
		Brackets: []TaxBracket{
			{Limit: 5_000_000, RateNumerator: 5, RateDenom: 100},
			{Limit: 10_000_000, RateNumerator: 10, RateDenom: 100},
			{Limit: 18_000_000, RateNumerator: 15, RateDenom: 100},
			{Limit: 32_000_000, RateNumerator: 20, RateDenom: 100},
			{Limit: 52_000_000, RateNumerator: 25, RateDenom: 100},
			{Limit: 80_000_000, RateNumerator: 30, RateDenom: 100},
			{Limit: math.MaxInt64, RateNumerator: 35, RateDenom: 100},
		},
		MealCashCap: 730_000,
	},
}

// PolicyAt returns the statutory policy parameter set effective at the given payment date.
// The returned value is an independent deep copy: modifying it will not affect any subsequent calculations.
// Returns ErrDateOutOfRange if date is prior to July 01, 2024.
func PolicyAt(date time.Time) (Policy, error) {
	p, err := policyAt(date)
	if err != nil {
		return Policy{}, err
	}
	pCopy := *p
	pCopy.Brackets = append([]TaxBracket(nil), p.Brackets...)
	pCopy.Ref = p.Ref.clone()
	return pCopy, nil
}

// policyAt returns a pointer to the shared policy struct (no copy). Internal use only.
// Never expose this pointer or its backing slice outside the package.
func policyAt(date time.Time) (*Policy, error) {
	if date.IsZero() {
		return nil, ErrDateZero
	}

	// Compare by calendar date in the date's timezone (ignoring clock time).
	y, m, d := date.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	for i := range defaultPolicies {
		if !day.Before(defaultPolicies[i].EffectiveFrom) {
			return &defaultPolicies[i], nil
		}
	}
	return nil, ErrDateOutOfRange
}

// clone returns an isolated copy of PolicyRef with its own Sources slice.
func (r PolicyRef) clone() PolicyRef {
	r.Sources = append([]string(nil), r.Sources...)
	return r
}
