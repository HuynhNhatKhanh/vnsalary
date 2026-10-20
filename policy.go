package vnsalary

import (
	"time"
)

// TaxBracket represents a single bracket in the progressive personal income tax schedule.
type TaxBracket struct {
	Limit         VND   // Upper limit of assessable income in this bracket (e.g. 5,000,000 VND; math.MaxInt64 for the top bracket)
	RateNumerator int64 // Tax rate numerator (e.g. 5)
	RateDenom     int64 // Tax rate denominator (e.g. 100)
}

// Policy contains all statutory parameters effective for a given period.
type Policy struct {
	EffectiveFrom time.Time
	Ref           PolicyRef

	// Base salary (lương cơ sở / mức tham chiếu) and statutory ceiling for social & health insurance (20 x base salary)
	BaseSalary            VND
	MaxSocialHealthSalary VND

	// Regional minimum wages for Regions I, II, III, IV (index 1..4) used to cap unemployment insurance (20 x regional min)
	RegionalMinSalaries [5]VND

	// Monthly personal family deductions (giảm trừ gia cảnh)
	PersonalDeduction  VND // Personal taxpayer deduction (giảm trừ bản thân)
	DependentDeduction VND // Deduction per qualifying dependent (giảm trừ người phụ thuộc)

	// Progressive PIT schedule (biểu thuế lũy tiến từng phần)
	Brackets []TaxBracket

	// Income excluded from PIT (thu nhập không tính thuế), see Extras.
	MealCashCap         VND  // Cash meal allowance excluded from taxable income up to this amount per month
	MealCashCapVerified bool // False when the cap for this period could not be confirmed from a legal text
	OvertimeFullyExempt bool // Overtime and night-work pay within the legal limits is fully excluded (from tax period 2026)

	// Yearly caps on medical and tuition expenses of the taxpayer and dependents
	// deductible at annual finalization (from tax period 2026); zero = not deductible.
	HealthcareDeductionCap VND
	EducationDeductionCap  VND

	// Payments of this amount or more to a resident without a labour contract
	// or with one under 3 months are withheld at 10% (see FlatWithholding).
	FlatWithholdingThreshold VND
}

// MaxUnemploymentSalary returns the statutory cap for unemployment insurance (20 x regional minimum wage).
func (p Policy) MaxUnemploymentSalary(region Region) VND {
	if region < 1 || region > 4 {
		return 0
	}
	return p.RegionalMinSalaries[region] * 20
}

// RegionalMinSalary returns the statutory minimum wage for the specified region.
func (p Policy) RegionalMinSalary(region Region) VND {
	if region < 1 || region > 4 {
		return 0
	}
	return p.RegionalMinSalaries[region]
}
