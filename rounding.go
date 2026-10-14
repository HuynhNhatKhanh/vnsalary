package vnsalary

// roundDiv performs integer division with half-up rounding for positive values.
// Eliminates floating-point representation and precision errors.
func roundDiv(numerator, denominator int64) VND {
	if numerator <= 0 || denominator <= 0 {
		return 0
	}
	return VND((numerator + denominator/2) / denominator)
}

// RoundToThousand rounds a non-negative VND amount to the nearest thousand (1,000 VND) half-up.
// Convenient for payroll systems with cash-disbursement rounding policies.
func RoundToThousand(v VND) VND {
	if v <= 0 {
		return 0
	}
	return (v + 500) / 1000 * 1000
}
