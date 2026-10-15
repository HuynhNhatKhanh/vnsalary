package vnsalary

import "errors"

const (
	// MaxGross is the maximum supported salary amount (100 trillion VND) to guard against int64 overflow.
	MaxGross VND = 100_000_000_000_000

	// MaxDependents is the maximum allowed number of dependents.
	MaxDependents = 50
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

	// ErrNetUnreachable indicates that no integer gross amount maps to this exact net salary due to half-up discretization.
	ErrNetUnreachable = errors.New("vnsalary: net salary is unreachable due to integer rounding")
)
