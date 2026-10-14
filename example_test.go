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
