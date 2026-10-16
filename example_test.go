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
