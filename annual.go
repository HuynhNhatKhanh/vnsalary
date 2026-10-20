package vnsalary

import (
	"fmt"
	"time"
)

// Payment is one salary payment of a tax year, as calculated by GrossToNet or
// as printed on a payslip or a withholding certificate.
type Payment struct {
	Date          time.Time // Payment date; the payment belongs to the tax year of this date (cash basis)
	TaxableIncome VND       // Taxable income (thu nhập chịu thuế): gross minus exempt income
	Insurance     VND       // Compulsory insurance paid by the employee (BHXH, BHYT, BHTN)
	TaxWithheld   VND       // PIT withheld by the payer, at the progressive rates or the 10% flat rate
}

// Payment returns r as a payment made on date, for use with Annual.
func (r Result) Payment(date time.Time) Payment {
	return Payment{Date: date, TaxableIncome: r.TaxableIncome, Insurance: r.Insurance.Total, TaxWithheld: r.Tax}
}

// AnnualOptions holds the yearly deductions claimed at finalization.
// The zero value claims only the personal deduction.
type AnnualOptions struct {
	// DependentMonths is the number of months claimed for all dependents
	// together (2 dependents for the whole year = 24). At finalization a
	// dependent counts from the month the duty of support arose, even if
	// registered later in the year (Decree 253/2026/NĐ-CP Art. 48.2.a).
	DependentMonths int
	Charity         VND // Contributions to licensed charity, humanitarian and study-promotion funds (Decree 253/2026/NĐ-CP Art. 49.1)
	Healthcare      VND // Medical expenses of the taxpayer and dependents; capped by Policy.HealthcareDeductionCap
	Education       VND // Tuition of the taxpayer and dependents; capped by Policy.EducationDeductionCap
}

// AnnualDeductions itemizes the deductions applied at finalization.
type AnnualDeductions struct {
	Personal   VND `json:"personal"`   // 12 months of the personal deduction, whatever the months worked
	Dependents VND `json:"dependents"` // DependentMonths × the monthly dependent deduction
	Charity    VND `json:"charity"`
	Healthcare VND `json:"healthcare"` // After the yearly cap
	Education  VND `json:"education"`  // After the yearly cap
	Total      VND `json:"total"`
}

// AnnualResult is the PIT finalization (quyết toán thuế TNCN) of a tax year.
type AnnualResult struct {
	Year             int              `json:"year"`
	TaxableIncome    VND              `json:"taxable_income"` // Sum of the payments' taxable income
	Insurance        VND              `json:"insurance"`      // Sum of the payments' compulsory insurance
	Deductions       AnnualDeductions `json:"deductions"`
	AssessableIncome VND              `json:"assessable_income"` // Taxable income minus insurance and deductions, not below zero
	Tax              VND              `json:"tax"`               // Tax due for the year on the yearly schedule
	Brackets         []BracketLine    `json:"brackets"`
	Withheld         VND              `json:"withheld"` // Sum of the payments' withheld tax
	Balance          VND              `json:"balance"`  // Tax - Withheld: positive = to pay, negative = refundable
	Policy           PolicyRef        `json:"policy"`
	Warnings         []Warning        `json:"warnings,omitempty"`
}

// WarnHealthEducationNotDeductible is reported when Healthcare or Education
// is given for a tax year before 2026, when these expenses were not
// deductible; they are ignored.
const WarnHealthEducationNotDeductible = "health_education_not_deductible"

// Annual computes the PIT finalization of a resident individual's salary
// income for a tax year: the yearly tax on the sum of all payments, compared
// with the tax already withheld.
//
// Monthly withholding and the yearly tax differ when income is uneven (a
// bonus month), when the taxpayer changed employers or did not work the
// whole year (the personal deduction always counts 12 months at
// finalization, Decree 253/2026/NĐ-CP Art. 48.1.b), or when dependents were
// registered late. Pass the payments from every payer in the year, including
// those withheld at the 10% flat rate.
//
// The yearly schedule is the monthly schedule with every limit multiplied by
// 12 (Law on PIT 109/2025/QH15 Art. 9.2). Claiming Charity, Healthcare or
// Education from tax period 2026 means the taxpayer must file the
// finalization personally instead of authorizing the employer
// (Decree 253/2026/NĐ-CP Art. 51.3).
func Annual(year int, payments []Payment, opts AnnualOptions) (AnnualResult, error) {
	if year < FirstAnnualYear || year > 9999 {
		return AnnualResult{}, ErrYearOutOfRange
	}
	if len(payments) > MaxAnnualPayments {
		return AnnualResult{}, ErrTooManyPayments
	}
	if opts.DependentMonths < 0 || opts.DependentMonths > MaxDependents*12 {
		return AnnualResult{}, ErrInvalidDependentMonths
	}
	for _, v := range []VND{opts.Charity, opts.Healthcare, opts.Education} {
		if err := checkAmount(v); err != nil {
			return AnnualResult{}, err
		}
	}

	res := AnnualResult{Year: year}
	for i, p := range payments {
		if p.Date.IsZero() {
			return AnnualResult{}, fmt.Errorf("payment %d: %w", i, ErrDateZero)
		}
		if p.Date.Year() != year {
			return AnnualResult{}, fmt.Errorf("payment %d dated %s: %w", i, p.Date.Format(time.DateOnly), ErrPaymentOutsideYear)
		}
		for _, v := range []VND{p.TaxableIncome, p.Insurance, p.TaxWithheld} {
			if err := checkAmount(v); err != nil {
				return AnnualResult{}, fmt.Errorf("payment %d: %w", i, err)
			}
		}
		res.TaxableIncome += p.TaxableIncome
		res.Insurance += p.Insurance
		res.Withheld += p.TaxWithheld
	}

	// Deductions and brackets are set per tax period, so the policy in force
	// on 1 January holds for the whole year (TestPoliciesInvariants).
	policy, err := policyAt(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return AnnualResult{}, err
	}

	d := AnnualDeductions{
		Personal:   policy.PersonalDeduction * 12,
		Dependents: policy.DependentDeduction * VND(opts.DependentMonths),
		Charity:    opts.Charity,
		Healthcare: minVND(opts.Healthcare, policy.HealthcareDeductionCap),
		Education:  minVND(opts.Education, policy.EducationDeductionCap),
	}
	d.Total = d.Personal + d.Dependents + d.Charity + d.Healthcare + d.Education
	res.Deductions = d

	res.AssessableIncome = max(res.TaxableIncome-res.Insurance-d.Total, 0)
	res.Tax, res.Brackets = scheduleTax(policy.Brackets, 12, res.AssessableIncome)
	res.Balance = res.Tax - res.Withheld
	res.Policy = policy.Ref.clone()

	if (opts.Healthcare > 0 && policy.HealthcareDeductionCap == 0) || (opts.Education > 0 && policy.EducationDeductionCap == 0) {
		res.Warnings = append(res.Warnings, Warning{
			Code:    WarnHealthEducationNotDeductible,
			Message: fmt.Sprintf("medical and tuition expenses are deductible from tax period 2026; ignored for %d", year),
		})
	}
	return res, nil
}

// checkAmount rejects negative amounts and amounts above MaxGross.
func checkAmount(v VND) error {
	if v < 0 {
		return ErrInvalidAmount
	}
	if v > MaxGross {
		return ErrAmountTooLarge
	}
	return nil
}
