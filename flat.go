package vnsalary

import "time"

// FlatResult is the 10% withholding on one payment to a resident without a
// labour contract or with one under 3 months.
type FlatResult struct {
	Amount    VND       `json:"amount"`    // Payment before tax
	Threshold VND       `json:"threshold"` // Per-payment amount from which withholding is required
	Required  bool      `json:"required"`  // Amount >= Threshold
	Withheld  VND       `json:"withheld"`  // 10% of Amount when Required, otherwise 0
	Net       VND       `json:"net"`       // Amount - Withheld
	Policy    PolicyRef `json:"policy"`
}

// FlatWithholding computes the PIT a payer withholds from one payment of
// salary, wages, fees or other income to a resident individual without a
// labour contract or with one under 3 months, including payments after the
// contract ended: 10% of the whole payment when it reaches the threshold of
// the payment date (Decree 253/2026/NĐ-CP Art. 50.2 from 2026-07-01: 5,000,000;
// Circular 111/2013/TT-BTC Art. 25.1.i before: 2,000,000), without family
// deductions.
//
// Below the threshold withholding is not required; from 2026-07-01 the payer
// may still withhold 10% when the individual asks (Art. 50.2). An individual
// whose only income is of this kind and who expects no tax after deductions
// may give the payer a commitment to defer withholding; that case is not
// modelled. Compulsory insurance, which depends on the contract, is not
// computed. The tax withheld is a prepayment: pass FlatResult.Payment to
// Annual to settle the year.
func FlatWithholding(amount VND, date time.Time) (FlatResult, error) {
	if err := checkAmount(amount); err != nil {
		return FlatResult{}, err
	}
	policy, err := policyAt(date)
	if err != nil {
		return FlatResult{}, err
	}
	res := FlatResult{
		Amount:    amount,
		Threshold: policy.FlatWithholdingThreshold,
		Required:  amount >= policy.FlatWithholdingThreshold,
		Policy:    policy.Ref.clone(),
	}
	if res.Required {
		res.Withheld = roundDiv(int64(amount), 10)
	}
	res.Net = amount - res.Withheld
	return res, nil
}

// Payment returns r as a payment made on date, for use with Annual.
func (r FlatResult) Payment(date time.Time) Payment {
	return Payment{Date: date, TaxableIncome: r.Amount, TaxWithheld: r.Withheld}
}
