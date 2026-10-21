package vnsalary

import (
	"fmt"
	"math"
	"time"
)

// VND is an amount in Vietnamese Dong as a whole number. Salaries are non-negative;
// differences between two results (for example in a policy comparison) may be negative.
type VND int64

// Region represents the statutory minimum wage region (Vùng lương tối thiểu: 1, 2, 3, 4).
type Region int

const (
	Region1 Region = 1
	Region2 Region = 2
	Region3 Region = 3
	Region4 Region = 4
)

// Options specifies optional calculation parameters.
type Options struct {
	InsuranceBase *VND      // nil = based on gross salary; non-nil = contractual insurance salary (lương đóng bảo hiểm)
	Dependents    int       // Qualifying dependents count (0..50) (số người phụ thuộc)
	Region        Region    // Statutory minimum wage region (1..4) (vùng lương tối thiểu)
	Date          time.Time // Salary payment date; selects the policy in force (cash basis, Law on PIT 109/2025/QH15 Art. 8.3)
	Extras        Extras    // Parts of gross treated differently from base salary; zero value = none

	// Foreigner marks a foreign national: no unemployment insurance (BHTN) for
	// the employee or the employer, because that insurance covers Vietnamese
	// citizens only (Law on Employment 74/2025/QH15 Art. 2.1; Law 38/2013/QH13
	// Art. 3.1 before 2026). Social and health insurance stay: a foreign
	// employee on a contract of 12 months or more pays social insurance (Law on
	// Social Insurance 41/2024/QH15 Art. 2.2). Its exemptions (intra-company
	// transfer, retirement age at signing, treaties) are not modelled; set
	// InsuranceBase to 0 in those cases only if health insurance is also not due.
	Foreigner bool
}

// Extras are amounts already included in gross (gross is the total paid in the
// month) that the law treats differently from base salary. The zero value
// reproduces the calculation of earlier versions.
//
// When InsuranceBase is nil, the insurance salary is gross minus Overtime and
// Bonus: both vary from month to month, so they are outside the salary "paid
// regularly and stably" (Law on Social Insurance 41/2024/QH15 Art. 31.1.b).
// MealCash and OtherExempt stay in the insurance salary; set InsuranceBase if
// the employment contract excludes them.
type Extras struct {
	Overtime    VND // Overtime and night-work pay within the legal hour limits; fully tax-exempt from tax period 2026 (pay for hours beyond the limits belongs in gross only)
	MealCash    VND // Cash meal allowance; tax-exempt up to Policy.MealCashCap
	OtherExempt VND // Other allowances exempt under the employer's regulation (phone, uniform, travel...); the caller is responsible for their limits
	Bonus       VND // Bonus paid this month; taxable, outside the insurance salary
}

func (e Extras) total() VND { return e.Overtime + e.MealCash + e.OtherExempt + e.Bonus }

// ExemptIncome itemizes the parts of gross excluded from taxable income
// (thu nhập không tính thuế).
type ExemptIncome struct {
	Overtime VND `json:"overtime"`
	Meal     VND `json:"meal"`
	Other    VND `json:"other"`
	Total    VND `json:"total"`
}

// Insurance itemizes mandatory employee contributions.
type Insurance struct {
	Social       VND `json:"social"`       // Social insurance (BHXH) 8%
	Health       VND `json:"health"`       // Health insurance (BHYT) 1.5%
	Unemployment VND `json:"unemployment"` // Unemployment insurance (BHTN) 1%
	Total        VND `json:"total"`        // Total employee insurance contribution
}

// Deductions itemizes personal and family tax relief deductions (giảm trừ gia cảnh).
type Deductions struct {
	Personal   VND `json:"personal"`   // Taxpayer personal deduction (giảm trừ bản thân)
	Dependents VND `json:"dependents"` // Qualifying dependents deduction (giảm trừ người phụ thuộc)
	Total      VND `json:"total"`      // Total deduction amount
}

// BracketLine details personal income tax paid at a specific progressive bracket.
type BracketLine struct {
	Bracket     int `json:"bracket"`      // Tax bracket index (1, 2, ...)
	TaxablePart VND `json:"taxable_part"` // Income portion assessed within this bracket
	RatePercent int `json:"rate_percent"` // Tax rate percentage (5, 10, 15, ...)
	Tax         VND `json:"tax"`          // Tax amount accrued at this bracket
}

// EmployerCost itemizes statutory employer contributions and total company expenditure.
type EmployerCost struct {
	Social         VND `json:"social"`          // Social insurance (BHXH) 17.5% (14% pension/death, 3% sickness/maternity, 0.5% occupational accident)
	Health         VND `json:"health"`          // Health insurance (BHYT) 3%
	Unemployment   VND `json:"unemployment"`    // Unemployment insurance (BHTN) 1%
	TotalInsurance VND `json:"total_insurance"` // Total employer contribution (21.5%)
	TotalCost      VND `json:"total_cost"`      // Total employer cost = Gross + TotalInsurance
}

// PolicyRef provides references to statutory decrees and enacted legislation.
type PolicyRef struct {
	EffectiveFrom string   `json:"effective_from"`
	Description   string   `json:"description"`    // Description in English
	DescriptionVI string   `json:"description_vi"` // Description in Vietnamese (mô tả tiếng Việt)
	Sources       []string `json:"sources"`        // Official statutory legal documents in Vietnamese
}

// Result holds the comprehensive breakdown of a salary calculation.
type Result struct {
	Gross            VND           `json:"gross"`
	Net              VND           `json:"net"`
	Insurance        Insurance     `json:"insurance"`
	TaxableIncome    VND           `json:"taxable_income"`    // Total taxable income (thu nhập chịu thuế)
	AssessableIncome VND           `json:"assessable_income"` // Income assessed for PIT after insurance and deductions (thu nhập tính thuế)
	Deductions       Deductions    `json:"deductions"`
	Tax              VND           `json:"tax"`
	Brackets         []BracketLine `json:"brackets"`
	Employer         EmployerCost  `json:"employer"`
	Policy           PolicyRef     `json:"policy"`
	Exempt           *ExemptIncome `json:"exempt,omitempty"`    // Exempt parts of gross; set only when Options.Extras is used
	BonusTax         VND           `json:"bonus_tax,omitempty"` // Tax caused by Extras.Bonus: tax with the bonus minus tax without it
	Warnings         []Warning     `json:"warnings,omitempty"`  // Inputs computed as given but worth checking; never change amounts
}

// Warning flags an input that is computed as given but is probably not what
// the caller intended. Warnings never change any amount.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WarnInsuranceSalaryBelowRegionalMinimum is reported when the salary used for
// insurance (InsuranceBase, or gross when InsuranceBase is nil) is positive but
// below the regional minimum wage. A full-time employee must be paid at least
// that minimum (Labour Code 2019, Art. 90.2), so a lower insurance salary
// usually means a part-time contract or an input mistake. It is a warning, not
// an error: the legal floor for the social insurance salary itself is the
// reference level (Law on Social Insurance 41/2024/QH15, Art. 31.1), which is
// lower than the regional minimum.
const WarnInsuranceSalaryBelowRegionalMinimum = "insurance_salary_below_regional_minimum"

// Warning codes for Options.Extras.
const (
	// WarnOvertimeTaxedBeforeExemption: before tax period 2026 only the part of
	// overtime pay above normal-hours pay was exempt (Circular 111/2013/TT-BTC
	// Art. 3.1.i). The library cannot split Extras.Overtime, so it taxes all of
	// it; put the exempt part in Extras.OtherExempt instead.
	WarnOvertimeTaxedBeforeExemption = "overtime_taxed_before_2026"

	// WarnMealCapUnverified: the meal allowance cap for this payment date could
	// not be confirmed from a legal text; the policy's cap is applied.
	WarnMealCapUnverified = "meal_cap_unverified"

	// WarnInsuranceSalaryIncludesAllowances: InsuranceBase is nil, so MealCash
	// and OtherExempt are included in the insurance salary. Set InsuranceBase if
	// the employment contract excludes them.
	WarnInsuranceSalaryIncludesAllowances = "insurance_salary_includes_allowances"
)

func minVND(a, b VND) VND {
	if a < b {
		return a
	}
	return b
}

// GrossToNet converts Gross salary to Net take-home pay and itemizes all deductions.
func GrossToNet(gross VND, opts Options) (Result, error) {
	if gross < 0 {
		return Result{}, ErrInvalidAmount
	}
	if gross > MaxGross {
		return Result{}, ErrAmountTooLarge
	}
	if err := validateOptions(opts); err != nil {
		return Result{}, err
	}

	policy, err := policyAt(opts.Date)
	if err != nil {
		return Result{}, err
	}

	res, err := computeNet(policy, gross, opts)
	if err != nil {
		return Result{}, err
	}
	return finalize(policy, res, opts), nil
}

// finalize prepares a computed result for the caller: it copies the policy
// reference (so the shared policy table never leaks) and attaches warnings.
// computeNet stays free of both, because NetToGross calls it many times.
func finalize(policy *Policy, res Result, opts Options) Result {
	res.Policy = res.Policy.clone()
	ex := opts.Extras

	if ex != (Extras{}) {
		exempt := exemptIncome(policy, ex)
		res.Exempt = &exempt
	}
	if ex.Bonus > 0 {
		// The bonus is outside the insurance salary, so only taxable income changes.
		without := res.TaxableIncome - ex.Bonus - res.Insurance.Total - res.Deductions.Total
		taxWithout, _ := progressiveTax(policy, max(without, 0))
		res.BonusTax = res.Tax - taxWithout
	}

	insSalary := insuranceSalary(res.Gross, opts)
	if regionalMin := policy.RegionalMinSalary(opts.Region); insSalary > 0 && insSalary < regionalMin {
		res.Warnings = append(res.Warnings, Warning{
			Code: WarnInsuranceSalaryBelowRegionalMinimum,
			Message: fmt.Sprintf("insurance salary %d VND is below the regional minimum wage %d VND for region %d",
				insSalary, regionalMin, opts.Region),
		})
	}
	if ex.Overtime > 0 && !policy.OvertimeFullyExempt {
		res.Warnings = append(res.Warnings, Warning{
			Code:    WarnOvertimeTaxedBeforeExemption,
			Message: "before tax period 2026 only overtime pay above normal-hours pay is exempt; all of Extras.Overtime is taxed, put the exempt part in Extras.OtherExempt",
		})
	}
	if ex.MealCash > 0 && !policy.MealCashCapVerified {
		res.Warnings = append(res.Warnings, Warning{
			Code:    WarnMealCapUnverified,
			Message: fmt.Sprintf("meal allowance cap %d VND for this payment date is not confirmed by a legal text", policy.MealCashCap),
		})
	}
	if opts.InsuranceBase == nil && (ex.MealCash > 0 || ex.OtherExempt > 0) {
		res.Warnings = append(res.Warnings, Warning{
			Code:    WarnInsuranceSalaryIncludesAllowances,
			Message: "insurance salary includes MealCash and OtherExempt; set InsuranceBase if the employment contract excludes them",
		})
	}
	return res
}

// validateOptions verifies parameter boundaries shared between GrossToNet and NetToGross.
func validateOptions(opts Options) error {
	if opts.Dependents < 0 || opts.Dependents > MaxDependents {
		return ErrInvalidDependents
	}
	if opts.Region < 1 || opts.Region > 4 {
		return ErrInvalidRegion
	}
	if opts.InsuranceBase != nil {
		if err := checkAmount(*opts.InsuranceBase); err != nil {
			return err
		}
	}
	for _, v := range []VND{opts.Extras.Overtime, opts.Extras.MealCash, opts.Extras.OtherExempt, opts.Extras.Bonus} {
		if err := checkAmount(v); err != nil {
			return err
		}
	}
	return nil
}

// exemptIncome returns the parts of opts.Extras excluded from taxable income
// under the policy.
func exemptIncome(policy *Policy, ex Extras) ExemptIncome {
	var e ExemptIncome
	if policy.OvertimeFullyExempt {
		e.Overtime = ex.Overtime
	}
	e.Meal = minVND(ex.MealCash, policy.MealCashCap)
	e.Other = ex.OtherExempt
	e.Total = e.Overtime + e.Meal + e.Other
	return e
}

// insuranceSalary returns the salary insurance contributions are based on.
func insuranceSalary(gross VND, opts Options) VND {
	if opts.InsuranceBase != nil {
		return *opts.InsuranceBase
	}
	return gross - opts.Extras.Overtime - opts.Extras.Bonus
}

// progressiveTax applies the policy's monthly progressive schedule to
// assessable income, rounding each bracket half-up to 1 VND.
func progressiveTax(policy *Policy, assessable VND) (VND, []BracketLine) {
	return scheduleTax(policy.Brackets, 1, assessable)
}

// scheduleTax applies a progressive schedule whose bracket limits are
// multiplied by months (12 for the yearly schedule, Law on PIT 109/2025/QH15
// Art. 9.2), rounding each bracket half-up to 1 VND.
func scheduleTax(brackets []TaxBracket, months VND, assessable VND) (VND, []BracketLine) {
	var total VND
	var lines []BracketLine
	rem := assessable
	var prevLimit VND
	for i, b := range brackets {
		if rem <= 0 {
			break
		}
		if b.Limit <= math.MaxInt64/months {
			b.Limit *= months
		} else {
			b.Limit = math.MaxInt64 // top bracket: no upper limit
		}
		part := minVND(rem, b.Limit-prevLimit)
		if part > 0 {
			tax := roundDiv(int64(part)*b.RateNumerator, b.RateDenom)
			total += tax
			lines = append(lines, BracketLine{
				Bracket:     i + 1,
				TaxablePart: part,
				RatePercent: int(b.RateNumerator * 100 / b.RateDenom),
				Tax:         tax,
			})
			rem -= part
		}
		prevLimit = b.Limit
	}
	return total, lines
}

// computeNet evaluates the pure calculation logic of GrossToNet: no validation, no policy cloning.
// Callers must have pre-validated gross and opts. Result.Policy.Sources references shared data,
// so public exported functions must clone it before returning.
func computeNet(policy *Policy, gross VND, opts Options) (Result, error) {
	if opts.Extras.total() > gross {
		return Result{}, ErrExtrasExceedGross
	}

	// 1. Determine base salary for mandatory insurance contributions
	insSalary := insuranceSalary(gross, opts)

	// 2. Mandatory employee insurance (Bảo hiểm người lao động)
	baseSH := minVND(insSalary, policy.MaxSocialHealthSalary)
	baseU := minVND(insSalary, policy.MaxUnemploymentSalary(opts.Region))
	if opts.Foreigner {
		baseU = 0 // no unemployment insurance for foreign nationals
	}

	empSocial := roundDiv(int64(baseSH)*8, 100)      // Social insurance (BHXH): 8%
	empHealth := roundDiv(int64(baseSH)*15, 1000)    // Health insurance (BHYT): 1.5%
	empUnemployment := roundDiv(int64(baseU)*1, 100) // Unemployment insurance (BHTN): 1%
	totalEmpInsurance := empSocial + empHealth + empUnemployment

	// Reject cases where mandatory insurance exceeds gross salary, causing negative net
	if totalEmpInsurance > gross {
		return Result{}, ErrInsuranceExceedsGross
	}

	// 3. Mandatory employer contributions (Chi phí người sử dụng lao động)
	emprSocial := roundDiv(int64(baseSH)*175, 1000)   // Employer social insurance (BHXH): 17.5%
	emprHealth := roundDiv(int64(baseSH)*3, 100)      // Employer health insurance (BHYT): 3%
	emprUnemployment := roundDiv(int64(baseU)*1, 100) // Employer unemployment insurance (BHTN): 1%
	totalEmprInsurance := emprSocial + emprHealth + emprUnemployment

	// 4. Family tax relief deductions (Giảm trừ gia cảnh)
	personalDeduction := policy.PersonalDeduction
	dependentDeduction := policy.DependentDeduction * VND(opts.Dependents)
	totalDeductions := personalDeduction + dependentDeduction

	// 5. Taxable and assessable income (Thu nhập chịu thuế & Thu nhập tính thuế)
	taxableIncome := gross - exemptIncome(policy, opts.Extras).Total
	assessableIncome := taxableIncome - totalEmpInsurance - totalDeductions
	if assessableIncome < 0 {
		assessableIncome = 0
	}

	// 6. Progressive personal income tax calculation (Thuế TNCN lũy tiến từng phần)
	totalTax, brackets := progressiveTax(policy, assessableIncome)

	// 7. Net take-home pay (Lương Net thực nhận)
	net := gross - totalEmpInsurance - totalTax

	return Result{
		Gross:            gross,
		Net:              net,
		Insurance:        Insurance{Social: empSocial, Health: empHealth, Unemployment: empUnemployment, Total: totalEmpInsurance},
		TaxableIncome:    taxableIncome,
		AssessableIncome: assessableIncome,
		Deductions:       Deductions{Personal: personalDeduction, Dependents: dependentDeduction, Total: totalDeductions},
		Tax:              totalTax,
		Brackets:         brackets,
		Employer: EmployerCost{
			Social:         emprSocial,
			Health:         emprHealth,
			Unemployment:   emprUnemployment,
			TotalInsurance: totalEmprInsurance,
			TotalCost:      gross + totalEmprInsurance,
		},
		Policy: policy.Ref,
	}, nil
}
