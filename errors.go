package vnsalary

import "errors"

const (
	// MaxGross is the maximum supported salary amount (100 trillion VND) to guard against int64 overflow.
	MaxGross VND = 100_000_000_000_000

	// MaxDependents is the maximum allowed number of dependents.
	MaxDependents = 50

	// MaxAnnualPayments is the maximum number of payments Annual accepts for one tax year.
	MaxAnnualPayments = 1000

	// FirstAnnualYear is the first tax year Annual supports: the policy data
	// starts on 2024-07-01, so 2025 is the first year it covers in full.
	FirstAnnualYear = 2025
)

var (
	// ErrInvalidAmount indicates that a negative amount was supplied for gross or net salary.
	ErrInvalidAmount = errors.New("vnsalary: amount must be non-negative")

	// ErrAmountTooLarge indicates that an amount exceeds the supported maximum (100 trillion VND).
	ErrAmountTooLarge = errors.New("vnsalary: amount exceeds supported maximum (100,000,000,000,000 VND)")

	// ErrInvalidDependents indicates that the dependents count is out of range [0, 50].
	ErrInvalidDependents = errors.New("vnsalary: dependents count must be between 0 and 50")

	// ErrInvalidRegion indicates that the minimum wage region is outside [1, 4].
	ErrInvalidRegion = errors.New("vnsalary: region must be between 1 and 4")

	// ErrDateOutOfRange indicates that the effective date is prior to July 01, 2024.
	ErrDateOutOfRange = errors.New("vnsalary: date must be on or after 2024-07-01")

	// ErrDateZero indicates that a zero time value was provided for the payment date.
	ErrDateZero = errors.New("vnsalary: date cannot be zero")

	// ErrExtrasExceedGross indicates that Options.Extras add up to more than gross;
	// gross is the total paid in the month and must include them.
	ErrExtrasExceedGross = errors.New("vnsalary: extras exceed gross (gross must include them)")

	// ErrInsuranceExceedsGross indicates that the mandatory employee insurance contribution exceeds the gross salary.
	ErrInsuranceExceedsGross = errors.New("vnsalary: insurance contribution exceeds gross salary")

	// ErrInvalidAccidentRate indicates that EmployerOptions.AccidentPerMille is outside [0, 10].
	ErrInvalidAccidentRate = errors.New("vnsalary: accident insurance rate must be between 0 and 10 per mille")

	// ErrYearOutOfRange indicates that Annual was asked for a tax year before FirstAnnualYear.
	ErrYearOutOfRange = errors.New("vnsalary: tax year must be 2025 or later")

	// ErrPaymentOutsideYear indicates that a payment passed to Annual is dated outside the tax year.
	ErrPaymentOutsideYear = errors.New("vnsalary: payment date is outside the tax year")

	// ErrTooManyPayments indicates that Annual received more than MaxAnnualPayments payments.
	ErrTooManyPayments = errors.New("vnsalary: too many payments (maximum 1000 per tax year)")

	// ErrInvalidDependentMonths indicates that AnnualOptions.DependentMonths is outside [0, MaxDependents*12].
	ErrInvalidDependentMonths = errors.New("vnsalary: dependent months must be between 0 and 600")

	// ErrNetUnreachable indicates that no integer gross amount maps to this exact net salary due to half-up discretization.
	ErrNetUnreachable = errors.New("vnsalary: net salary is unreachable due to integer rounding")
)
