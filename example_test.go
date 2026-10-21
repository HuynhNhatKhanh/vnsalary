package vnsalary_test

import (
	"fmt"
	"time"

	"github.com/HuynhNhatKhanh/vnsalary"
)

func ExampleGrossToNet() {
	opts := vnsalary.Options{
		Dependents: 1,
		Region:     vnsalary.Region1,
		Date:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}

	res, err := vnsalary.GrossToNet(35_000_000, opts)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("Gross: %d VND\n", res.Gross)
	fmt.Printf("BHXH (8%%): %d VND\n", res.Insurance.Social)
	fmt.Printf("BHYT (1.5%%): %d VND\n", res.Insurance.Health)
	fmt.Printf("BHTN (1%%): %d VND\n", res.Insurance.Unemployment)
	fmt.Printf("Total Insurance: %d VND\n", res.Insurance.Total)
	fmt.Printf("Tax: %d VND\n", res.Tax)
	fmt.Printf("Net: %d VND\n", res.Net)

	// Output:
	// Gross: 35000000 VND
	// BHXH (8%): 2800000 VND
	// BHYT (1.5%): 525000 VND
	// BHTN (1%): 350000 VND
	// Total Insurance: 3675000 VND
	// Tax: 481250 VND
	// Net: 30843750 VND
}

func ExampleNetToGross() {
	opts := vnsalary.Options{
		Dependents: 1,
		Region:     vnsalary.Region1,
		Date:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}

	res, err := vnsalary.NetToGross(30_843_750, opts)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("Net: %d VND -> Gross: %d VND\n", res.Net, res.Gross)

	// Output:
	// Net: 30843750 VND -> Gross: 35000000 VND
}

// A bonus is part of the gross paid in the month. It is taxed in that month but
// is not part of the insurance salary; BonusTax is the tax the bonus adds.
func ExampleGrossToNet_bonus() {
	opts := vnsalary.Options{
		Region: vnsalary.Region1,
		Date:   time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		Extras: vnsalary.Extras{Bonus: 20_000_000}, // included in the 50M gross
	}

	res, err := vnsalary.GrossToNet(50_000_000, opts)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("Insurance: %d VND (on 30M salary, not the bonus)\n", res.Insurance.Total)
	fmt.Printf("Tax: %d VND, of which caused by the bonus: %d VND\n", res.Tax, res.BonusTax)
	fmt.Printf("Net: %d VND\n", res.Net)

	// Output:
	// Insurance: 3150000 VND (on 30M salary, not the bonus)
	// Tax: 2770000 VND, of which caused by the bonus: 2135000 VND
	// Net: 44080000 VND
}

// Overtime pay and allowances inside gross are excluded from taxable income
// under the rules in force on the payment date.
func ExampleGrossToNet_allowances() {
	opts := vnsalary.Options{
		Dependents: 1,
		Region:     vnsalary.Region1,
		Date:       time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		Extras: vnsalary.Extras{
			Overtime:    5_000_000, // fully exempt from tax period 2026
			MealCash:    1_500_000, // exempt up to 1,200,000 from 2026-07-01
			OtherExempt: 800_000,   // phone, uniform... under the employer's regulation
		},
	}

	res, err := vnsalary.GrossToNet(55_000_000, opts)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("Exempt: overtime %d + meal %d + other %d = %d VND\n",
		res.Exempt.Overtime, res.Exempt.Meal, res.Exempt.Other, res.Exempt.Total)
	fmt.Printf("Taxable income: %d VND\n", res.TaxableIncome)
	fmt.Printf("Net: %d VND\n", res.Net)

	// Output:
	// Exempt: overtime 5000000 + meal 1200000 + other 800000 = 7000000 VND
	// Taxable income: 48000000 VND
	// Net: 48145000 VND
}

// Warnings flag inputs that are computed as given but worth checking; they
// never change the amounts.
func ExampleResult_warnings() {
	opts := vnsalary.Options{
		Region: vnsalary.Region1,
		Date:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
	}

	res, err := vnsalary.GrossToNet(3_000_000, opts)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	for _, w := range res.Warnings {
		fmt.Println(w.Code)
	}
	fmt.Printf("Net: %d VND\n", res.Net)

	// Output:
	// insurance_salary_below_regional_minimum
	// Net: 2685000 VND
}

// Changed employer in 2026: 40M a month from January to June, no income in
// July, 25M a month from August. Each employer withheld with a monthly personal
// deduction; at finalization the personal deduction counts all 12 months, so
// part of the tax is refunded.
func ExampleAnnual() {
	var payments []vnsalary.Payment
	for month := 1; month <= 12; month++ {
		gross := vnsalary.VND(40_000_000)
		switch {
		case month == 7:
			continue
		case month > 7:
			gross = 25_000_000
		}
		paid := time.Date(2026, time.Month(month), 28, 0, 0, 0, 0, time.UTC)
		res, err := vnsalary.GrossToNet(gross, vnsalary.Options{Region: vnsalary.Region1, Date: paid})
		if err != nil {
			fmt.Println(err)
			return
		}
		payments = append(payments, res.Payment(paid))
	}

	res, err := vnsalary.Annual(2026, payments, vnsalary.AnnualOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Personal deduction:", res.Deductions.Personal)
	fmt.Println("Tax for the year:  ", res.Tax)
	fmt.Println("Tax withheld:      ", res.Withheld)
	fmt.Println("Balance:           ", res.Balance)
	// Output:
	// Personal deduction: 186000000
	// Tax for the year:   8067500
	// Tax withheld:       10898750
	// Balance:            -2831250
}

// A one-off payment of 8,000,000 in September 2026 to someone without a labour
// contract: 10% is withheld because the payment reaches the 5,000,000
// threshold. With no other income in the year, finalization refunds it.
func ExampleFlatWithholding() {
	paid := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	res, err := vnsalary.FlatWithholding(8_000_000, paid)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Withheld:", res.Withheld, "Net:", res.Net)

	annual, err := vnsalary.Annual(2026, []vnsalary.Payment{res.Payment(paid)}, vnsalary.AnnualOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Balance at finalization:", annual.Balance)
	// Output:
	// Withheld: 800000 Net: 7200000
	// Balance at finalization: -800000
}

// An employer approved for the 0.3% accident insurance rate, including the 2%
// trade union fee in its cost. The employee's net is unchanged.
func ExampleEmployerOptions() {
	rate := int64(3) // 0.3%
	res, err := vnsalary.GrossToNet(30_000_000, vnsalary.Options{
		Region:   vnsalary.Region1,
		Date:     time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		Employer: vnsalary.EmployerOptions{AccidentPerMille: &rate, UnionFee: true},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Employer social insurance:", res.Employer.Social)
	fmt.Println("Trade union fee:          ", res.Employer.UnionFee)
	fmt.Println("Total employer cost:      ", res.Employer.TotalCost)
	// Output:
	// Employer social insurance: 5190000
	// Trade union fee:           600000
	// Total employer cost:       36990000
}
