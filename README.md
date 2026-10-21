# vnsalary

[![CI](https://github.com/HuynhNhatKhanh/vnsalary/actions/workflows/ci.yml/badge.svg)](https://github.com/HuynhNhatKhanh/vnsalary/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/HuynhNhatKhanh/vnsalary.svg)](https://pkg.go.dev/github.com/HuynhNhatKhanh/vnsalary)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Vietnam Gross ↔ Net Salary & PIT Calculator in pure Go.**  
> *Exact to the single VND. Analytical piecewise-linear inversion. Zero external dependencies.*

📖 [Đọc bản tiếng Việt tại đây (Vietnamese README)](README.vi.md)

---

## 🌟 Highlights

- **Zero External Dependencies:** 100% built with the Go Standard Library.
- **Analytical Piecewise-Linear Inversion:** Solves Net → Gross by identifying policy breakpoints and computing initial analytical approximations, combined with local boundary refinement (±50 VND) and monotonic binary search fallback to handle integer half-up discretization. Guarantees round-trip mathematical consistency: `GrossToNet(NetToGross(net)).Net == net`.
- **Exact Integer Arithmetic (`int64` VND):** Zero `float64` for currency. Rounding is performed half-up independently on each mandatory insurance item and progressive tax bracket using integer arithmetic `(n + d/2) / d`, eliminating IEEE-754 floating-point errors.
- **Multi-Era Policy Engine & Legal Traceability:** Automatically selects policy parameters based on the salary payment date using the **Cash-basis** principle (Law on PIT 109/2025/QH15 Art. 8.3; previously Circular 111/2013/TT-BTC Art. 8.2.b):
  - **July 01, 2024:** Base salary 2,340,000 VND (Decree 73/2024/ND-CP), regional minimums (Decree 74/2024/ND-CP), 7-bracket progressive tax schedule (Law on PIT), personal deductions of 11M / 4.4M (Resolution 954/2020/UBTVQH14).
  - **January 01, 2026:** New 5-bracket progressive tax schedule (Law on PIT No. 109/2025/QH15 Article 9), upgraded personal deductions of 15.5M / 6.2M (Resolution 110/2025/UBTVQH15), regional minimums (Decree 293/2025/ND-CP).
  - **July 01, 2026:** Base salary raised to 2,530,000 VND (Decree 161/2026/ND-CP), so the social & health insurance cap rises from 46.8M to 50.6M. Other parameters unchanged.
- **Concurrency-Safe & Immutable:** `PolicyAt` returns deep copies of policies, preventing external modification of global slices. Input validation enforces bounded amounts (up to 10¹⁴ VND) and dependent limits (≤ 50) to prevent `int64` overflow.
- **Comprehensive Employer Cost:** Computes mandatory employer contributions (Social 17.5%, Health 3%, Unemployment 1%) and total company cost.
- **Annual PIT Finalization:** `Annual` settles a tax year from its payments (several employers, bonus months, months without income, late dependents) and returns the amount to pay or refund; personal deduction for 12 months, medical and tuition deductions from 2026 (Decree 253/2026/ND-CP Art. 48–49).
- **CLI Included:** Standalone command-line tool with human-readable formatting, policy diff comparison, annual finalization from a JSON file, and `--json` export.

---

## 📦 Installation

```bash
go get github.com/HuynhNhatKhanh/vnsalary
```

Requirements: **Go 1.22+**.

---

## 🚀 Quick Start

### 1. Gross to Net Calculation

```go
package main

import (
	"fmt"
	"time"

	"github.com/HuynhNhatKhanh/vnsalary"
)

func main() {
	opts := vnsalary.Options{
		Dependents: 1,
		Region:     vnsalary.Region1,
		Date:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}

	result, err := vnsalary.GrossToNet(35_000_000, opts)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Gross:      %d VND\n", result.Gross)
	fmt.Printf("Net:        %d VND\n", result.Net)
	fmt.Printf("Insurance:  %d VND\n", result.Insurance.Total)
	fmt.Printf("PIT Tax:    %d VND\n", result.Tax)
	fmt.Printf("Employer:   %d VND\n", result.Employer.TotalCost)
}
```

### 2. Net to Gross Calculation (Exact Inversion)

```go
net := vnsalary.VND(30_843_750)
result, err := vnsalary.NetToGross(net, opts)
if err != nil {
	panic(err)
}

fmt.Printf("Required Gross: %d VND\n", result.Gross) // Output: 35000000 VND
```

### 3. Custom Insurance Base

Many employers register a contractual basic salary for insurance contributions:

```go
insBase := vnsalary.VND(5_000_000)
opts := vnsalary.Options{
	InsuranceBase: &insBase,
	Dependents:    0,
	Region:        vnsalary.Region1,
	Date:          time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
}

result, err := vnsalary.GrossToNet(30_000_000, opts)
```

### 4. Overtime, Allowances and Bonuses

Gross is the **total paid in the month**. Tell the library which parts of it are overtime, allowances or a bonus with `Options.Extras`:

```go
opts := vnsalary.Options{
	Region: vnsalary.Region1,
	Date:   time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
	Extras: vnsalary.Extras{
		Overtime: 5_000_000,  // tax-exempt from 2026, outside the insurance salary
		MealCash: 1_500_000,  // tax-exempt up to 1,200,000/month from 2026-07-01
		Bonus:    10_000_000, // taxed in the month paid, outside the insurance salary
	},
}
result, _ := vnsalary.GrossToNet(55_000_000, opts) // 55M includes the extras
fmt.Println(result.Exempt.Total, result.BonusTax)   // exempt amount; tax caused by the bonus
```

| Extra | PIT | Insurance salary (when `InsuranceBase` is nil) | Basis |
| :--- | :--- | :--- | :--- |
| `Overtime` (within legal hour limits) | exempt from tax period 2026; before 2026 taxed, with a warning | excluded | Law 109/2025 Art. 4.8, Decree 253/2026 Art. 26 |
| `MealCash` | exempt up to 1,200,000/month from 2026-07-01; 730,000 before, with a warning (not confirmed) | included | Decree 253/2026 Art. 8.2.g |
| `OtherExempt` (phone, uniform, …) | exempt; limits are the caller's responsibility | included | Decree 253/2026 Art. 8.2.đ |
| `Bonus` | taxed in the month paid; `Result.BonusTax` = tax with it − tax without it | excluded | Decree 253/2026 Art. 8.2.i, 46.3 |

Overtime and bonuses vary month to month, so they are outside the salary "paid regularly and stably" that insurance is based on (Law 41/2024 Art. 31.1.b). If your contract also excludes the meal or other allowances, set `InsuranceBase`. `NetToGross` holds the extras fixed and returns the smallest total gross.

### 5. Annual PIT Finalization

Monthly withholding and the yearly tax differ when income is uneven, when you changed employers or did not work the whole year (the personal deduction still counts 12 months), or when dependents were registered late. `Annual` computes the yearly tax and what is left to pay or refund:

```go
var payments []vnsalary.Payment
for month := 1; month <= 12; month++ {
	paid := time.Date(2026, time.Month(month), 28, 0, 0, 0, 0, time.UTC)
	r, _ := vnsalary.GrossToNet(30_000_000, vnsalary.Options{Region: vnsalary.Region1, Dependents: 1, Date: paid})
	payments = append(payments, r.Payment(paid)) // or fill Payment from a payslip
}
res, _ := vnsalary.Annual(2026, payments, vnsalary.AnnualOptions{
	DependentMonths: 12,         // one dependent for the whole year
	Healthcare:      10_000_000, // deductible from 2026, up to 23M/year
})
fmt.Println(res.Tax, res.Withheld, res.Balance) // Balance < 0: refundable
```

| Deduction | Amount | Basis |
| :--- | :--- | :--- |
| Personal | 12 × monthly, whatever the months worked | Decree 253/2026 Art. 48.1.b |
| Dependents | `DependentMonths` × monthly, from the month the duty of support arose | Decree 253/2026 Art. 48.2.a |
| Charity, medical (≤ 23M), tuition (≤ 24M) | per year, from tax period 2026; claiming them means filing yourself | Decree 253/2026 Art. 49, 51.3 |

The yearly schedule is the monthly one × 12 (Law 109/2025 Art. 9.2). Supported tax years: 2025 onwards.

### 6. 10% Withholding (No Labour Contract, or Under 3 Months)

Payments to a resident without a labour contract or with one under 3 months are withheld at 10% of the whole payment, without family deductions, once the payment reaches the threshold of its date:

```go
res, _ := vnsalary.FlatWithholding(8_000_000, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC))
fmt.Println(res.Withheld, res.Net) // 800000 7200000
```

| Payment date | Threshold per payment | Basis |
| :--- | :--- | :--- |
| before 2026-07-01 | 2,000,000 | Circular 111/2013 Art. 25.1.i |
| from 2026-07-01 | 5,000,000 (below it, 10% only at the individual's request) | Decree 253/2026 Art. 50.2 |

The withholding is a prepayment: `res.Payment(date)` goes into `Annual`, which refunds it when the year's income is below the deductions. Not modelled: the commitment that lets an individual defer withholding, and compulsory insurance, which depends on the contract.

### 7. Foreign Employees

Foreign nationals pay no unemployment insurance (it covers Vietnamese citizens only); social and health insurance stay:

```go
res, _ := vnsalary.GrossToNet(60_000_000, vnsalary.Options{
	Region: vnsalary.Region1, Date: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), Foreigner: true,
})
// res.Insurance.Unemployment == 0, res.Employer.Unemployment == 0
```

A foreign employee pays social insurance on a contract of 12 months or more (Law 41/2024 Art. 2.2); the exemptions (intra-company transfer, retirement age at signing, treaties) are not modelled. `Foreigner` changes insurance only: tax still follows residence (183 days in Vietnam, or a regular place to stay).

### 8. Non-Residents

A non-resident pays 20% of taxable income from work in Vietnam, with no family or insurance deductions (Decree 253/2026 Art. 64; Law on PIT 2007 Art. 26 before). Residence is the caller's call: 183 days or more in Vietnam in the calendar year or in 12 consecutive months, or a regular place to stay (Decree 253/2026 Art. 4).

```go
res, _ := vnsalary.GrossToNet(60_000_000, vnsalary.Options{
	Region: vnsalary.Region1, Date: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
	Foreigner: true, NonResident: true,
})
fmt.Println(res.Tax, res.Net) // 12000000 43193000
```

`NonResident` changes tax only; combine it with `Foreigner` for a foreign national. `NetToGross` still returns the smallest gross: with extras, a non-resident's net can fall by 1 VND as gross rises (the flat tax does not shrink when insurance grows), so it searches a 32 VND window below the first match.

### 9. Employer Cost Options

`Options.Employer` changes the employer's cost only, never the employee's net:

```go
rate := int64(3) // 0.3%, if your accident insurance rate was approved
res, _ := vnsalary.GrossToNet(30_000_000, vnsalary.Options{
	Region: vnsalary.Region1, Date: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
	Employer: vnsalary.EmployerOptions{AccidentPerMille: &rate, UnionFee: true},
})
fmt.Println(res.Employer.Social, res.Employer.UnionFee) // 5190000 600000
```

| Option | Default | Basis |
| :--- | :--- | :--- |
| `AccidentPerMille` | 5 (0.5%) occupational accident insurance, part of employer social insurance | Decree 58/2020 Art. 4.1.a; 0.3% when approved |
| `UnionFee` | off; when set, 2% of the social insurance salary is added to `TotalCost` | Law on Trade Unions 50/2024 Art. 29.1.b; due with or without a grassroots union |

---

## 📊 Legal Reference & Policy Comparison

| Parameter | From 01/07/2024 | From 01/01/2026 | From 01/07/2026 | Legal Source |
| :--- | :--- | :--- | :--- | :--- |
| **Base Salary / Reference** | 2,340,000 VND | 2,340,000 VND | **2,530,000 VND** | Decree 73/2024/ND-CP; Decree 161/2026/ND-CP |
| **Social & Health Cap** | 46,800,000 VND (20 × Base) | 46,800,000 VND | **50,600,000 VND** | until 30/06/2025: Law on Social Insurance 2014 Art. 89.3, Law on Health Insurance Art. 14.2; from 01/07/2025: Law on Social Insurance No. 41/2024/QH15 Art. 31.1.đ (reference level = base salary, Art. 141.13), Law on Health Insurance Art. 14.5 as amended by Law No. 51/2024/QH15 |
| **Unemployment Cap (Region I)** | 99,200,000 VND (20 × 4.96M) | 106,200,000 VND (20 × 5.31M) | 106,200,000 VND | Law on Employment 2013 Art. 58 (until 2025), Law on Employment No. 74/2025/QH15 Art. 34.2 (from 01/01/2026); Decree 74/2024 & Decree 293/2025 |
| **Personal Deduction** | 11,000,000 VND / month | 15,500,000 VND / month | 15,500,000 VND / month | Resolution 954/2020 (2024) & Resolution 110/2025 (2026) |
| **Dependent Deduction** | 4,400,000 VND / person / month | 6,200,000 VND / person / month | 6,200,000 VND / person / month | Resolution 954/2020 (2024) & Resolution 110/2025 (2026) |
| **PIT Schedule** | 7 brackets (5% – 35%) | **5 brackets (5% – 35%)** | 5 brackets | Law on PIT 2007 (2024) & Law on PIT No. 109/2025/QH15 Art. 9 (2026) |

### 2026 Progressive PIT Tax Schedule (Law 109/2025/QH15 Art. 9)

| Bracket | Monthly Assessable Income | Tax Rate |
| :---: | :--- | :---: |
| **1** | Up to 10,000,000 VND | **5%** |
| **2** | Over 10,000,000 to 30,000,000 VND | **10%** |
| **3** | Over 30,000,000 to 60,000,000 VND | **20%** |
| **4** | Over 60,000,000 to 100,000,000 VND | **30%** |
| **5** | Over 100,000,000 VND | **35%** |

> **Cash Basis Principle:** taxable income from salaries is determined at the **time the income is paid** (Law on PIT No. 109/2025/QH15 Art. 8.3; before that Circular 111/2013/TT-BTC Art. 8.2.b, replaced by Circular 87/2026/TT-BTC from 01/07/2026). Salary for December 2025 paid on January 05, 2026 is subject to the **2026 policy**, as the tax authority confirmed in January 2026.

## 🔢 Rounding Convention

All amounts are whole VND (`int64`); no floating point is used anywhere.

| Amount | Rule |
| :--- | :--- |
| Each insurance item, employee (8%, 1.5%, 1%) and employer (17.5%, 3%, 1%) | rate × capped salary, rounded **half-up to 1 VND separately** |
| PIT | tax of **each bracket** rounded half-up to 1 VND, then summed |
| Net, totals | exact sums of the rounded items |
| Net rounded to 1,000 VND | not applied (company policy); use `RoundToThousand` if you need it |

Half-up means x.5 VND goes up: 1.5% of 20,000,300 is 300,004.5, so 300,005 (a golden case).

The legal texts set rates and caps but no rounding step, and calculators differ (some round only the totals, some truncate). The effect is a few VND at most. Example, gross 20,000,028 in 2026: the insurance items are 1,600,002.24 + 300,000.42 + 200,000.28, so per item 2,100,002 but once on the total 2,100,003, and the net differs by 1 VND (TopCV gives the same net as vnsalary here, checked on 2026-10-13). In one sample (gross 20.0M–20.2M, 2026, no dependents) the two conventions gave nets 1 VND apart for about 28% of amounts. **A difference of 1,000 VND or more is not rounding: please open an issue.**


---

## ⚠️ Limitations

vnsalary covers **employees on labour contracts, resident or not, Vietnamese or foreign**; payments without a contract or under one shorter than 3 months go through `FlatWithholding`. Check these before relying on it for a real payroll:

- **Exempt allowances are taken as given.** `Extras.Overtime` must be pay for hours within the legal limits (pay for extra hours belongs in gross only), and the limits of `Extras.OtherExempt` (phone, uniform, …) follow your employer's regulation; the library does not check them.
- **Not covered:** for foreign employees, the social insurance exemptions (intra-company transfer, retirement age, treaties); for non-residents working in and outside Vietnam, splitting income by days in Vietnam (Decree 253/2026 Art. 64.3): pass the Vietnam part.
- **An insurance salary below the regional minimum wage is computed as given, with a warning.** A full-time employee must be paid at least that minimum, so `Result.Warnings` reports code `insurance_salary_below_regional_minimum` (the CLI prints it with `[!]`). It is not an error because part-time contracts can legally be lower.
- **Employer accident insurance defaults to 0.5%** and the **2% trade union fee is off by default**; set `Options.Employer` (CLI `--accident-rate`, `--union-fee`) to match your employer.
- **Region is the place of work**, not the head office or the employee's home. The library does not infer it; pass the region where the employee works.

Planned work is tracked in the [milestones](https://github.com/HuynhNhatKhanh/vnsalary/milestones); ideas and requests are welcome as issues.

---

## 🛠️ Command-Line Interface (CLI)

### Install CLI
```bash
go install github.com/HuynhNhatKhanh/vnsalary/cmd/vnsalary@latest
```
*(Or compile directly: `go build -o vnsalary ./cmd/vnsalary`)*

**Homebrew** (macOS, Linux) and **Scoop** (Windows):
```bash
brew install HuynhNhatKhanh/tap/vnsalary
```
```powershell
scoop bucket add huynhnhatkhanh https://github.com/HuynhNhatKhanh/homebrew-tap
scoop install huynhnhatkhanh/vnsalary
```

**Without a package manager:** download a prebuilt binary for Linux, macOS or Windows (amd64/arm64) from the [latest release](https://github.com/HuynhNhatKhanh/vnsalary/releases/latest), check it against `checksums.txt`, and run `vnsalary version`.

### Commands

#### 1. Calculate Gross to Net
```bash
vnsalary gross 35000000 -d 1 -r 1 --date 2026-01-15
# Also supports flags before positional args:
vnsalary gross -d 1 35000000 --date 2026-01-15
```

#### 2. Calculate Net to Gross
```bash
vnsalary net 30843750 -d 1 -r 1 --date 2026-01-15
# Outputs exactly 35,000,000 VND Gross
```

#### 3. Compare Policy Diff
```bash
vnsalary diff 35000000 -d 1 -r 1                                   # default: 2024-07-01 vs 2026-01-01
vnsalary diff 60000000 --from 2026-06-30 --to 2026-07-01           # any two payment dates
```

#### Overtime, allowances and bonuses
The amount is the total paid in the month; flags say which parts are extras:
```bash
vnsalary gross 50000000 --bonus 20000000 --date 2026-08-10        # shows the tax caused by the bonus
vnsalary gross 55000000 --overtime 5000000 --meal 1500000 --other-exempt 800000 --date 2026-08-10
```

#### Foreign employees
```bash
vnsalary gross 60000000 --foreigner --date 2026-08-10                # no unemployment insurance
vnsalary gross 60000000 --foreigner --non-resident --date 2026-08-10 # 20% flat, no deductions
vnsalary gross 30000000 --accident-rate 0.3 --union-fee              # employer cost with 0.3% accident insurance and the 2% union fee
```

#### 10% withholding
```bash
vnsalary flat 8000000 --date 2026-09-10                         # 800,000 withheld; below the threshold it says so
```

#### Annual PIT finalization
`vnsalary annual` reads the payments of a tax year from a JSON file (`-` reads stdin). Give each payment either its `gross` (withholding is computed with the file's `dependents` and `region`; `bonus`, `overtime`, `meal`, `other_exempt`, `ins_base` as in `gross`) or the amounts on the payslip (`taxable_income`, `insurance`, `tax_withheld`):
```json
{
  "year": 2026,
  "dependents": 1,
  "healthcare": 10000000,
  "payments": [
    {"date": "2026-01-28", "gross": 40000000},
    {"date": "2026-08-28", "taxable_income": 25000000, "insurance": 2625000, "tax_withheld": 343750}
  ]
}
```
```bash
vnsalary annual payments.json            # prints TO PAY or REFUNDABLE
```
`dependent_months` defaults to `dependents` × 12; `charity`, `healthcare` and `education` are yearly amounts. Unknown fields are rejected, so a typo cannot silently drop a deduction.

#### 4. Language Selection
Default interface is in Vietnamese; `--lang en` switches the whole CLI to English (help, errors, tables, and amounts formatted as `35,000,000 VND`). JSON output is language-independent:
```bash
vnsalary gross 35000000 -d 1 --lang en
```

#### 5. JSON Output
Append `--json` to any command for programmatic integration:
```bash
vnsalary gross 35000000 --json
```

---

## ⚡ Performance & Benchmarks

Example run (`go test -bench=. -benchmem`, linux/amd64, Intel i7-10700K; numbers vary by machine and run):
```text
BenchmarkGrossToNet-16      3850897           329.0 ns/op         336 B/op          3 allocs/op
BenchmarkNetToGross-16       111998         12749   ns/op       12920 B/op        133 allocs/op
```

- **Forward calculation**: sub-microsecond; allocations come from the bracket lines and the copied policy reference in the result.
- **Inverse calculation**: interpolation on the linear segment, then an ascending scan of ±50 VND extended down to the smallest gross with the same net, with a binary-search fallback. See [docs/design.md](docs/design.md#7-net--gross) for why the result is always the smallest gross.

---

## 🧪 Testing

The library is verified with a comprehensive test suite:
- **Golden Test Dataset (`testdata/golden.json`):** 40 cases, each with a `source` and `verified_at` field stating how it was checked (hand calculation, an independent reference implementation, or an external calculator). 12 cases were matched to the VND against the TopCV calculator (2026-10-13 and 2026-10-15); cases not yet cross-checked with an external tool say so explicitly. 14 cases use odd amounts such as 23,456,789 so that every rounding step is exercised; 14 pin each insurance item. 4 cases cover foreign employees and non-residents.
- **Round-Trip Invariance Testing:** for every gross `g`, `NetToGross(Net(g))` returns the **smallest** gross with the same net, so `Net(NetToGross(Net(g))) == Net(g)` and `NetToGross(Net(g)).Gross <= g` (several grosses can share one net because of rounding).
- **Policy invariants:** `TestPoliciesInvariants` fails if a new policy breaks an assumption of the inverse algorithm (ordering, social/health cap below unemployment cap, top bracket 35%, sources present).
- **Fuzz Testing:** Includes `FuzzGrossToNet_NetToGross` verifying property invariance on arbitrary inputs.
- **Security & Integrity:** Verifies immutability of `PolicyAt` slices, overflow bounds checking (> 10¹⁴ VND), and negative net rejection when insurance exceeds gross.
- **CLI:** every command, flag spelling, language and exit code is tested in-process; table output is pinned by snapshot files.
- **CI:** Go 1.22 (minimum), oldstable and stable, with the race detector, `go vet`, golangci-lint, gofmt and a fuzz smoke run.

```bash
go test -cover ./...
```
Statement coverage at v0.6.0: library **97.7%**, CLI **96.4%**.

---

## 🤝 Contributing

Law changes and calculation mismatches are the most useful reports; both have issue templates. See [CONTRIBUTING.md](CONTRIBUTING.md) for how a new policy is added and how versions are numbered, [CHANGELOG.md](CHANGELOG.md) for releases, and [docs/design.md](docs/design.md) for the design and its decisions.

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.
