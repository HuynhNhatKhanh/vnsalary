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
- **CLI Included:** Standalone command-line tool with human-readable formatting, policy diff comparison, and `--json` export.

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

vnsalary covers **resident employees on labour contracts of 3 months or more**, with the whole gross salary taxable. Check these before relying on it for a real payroll:

- **No tax-exempt income or bonuses yet.** Overtime premium, meal and phone allowances, and 13th-month or Tết bonuses are not modelled; pass only the taxable salary as gross.
- **Not covered:** annual PIT finalization, contracts under 3 months (10% withholding), non-residents (20%), foreign employees (no unemployment insurance).
- **Insurance base is not checked against the regional minimum wage.** The law requires the contribution salary to be at least that minimum; an `InsuranceBase` below it is still computed.
- **Employer accident insurance is fixed at 0.5%**, so employer social insurance is always 17.5%. Some employers pay 0.3%.
- **The 2% trade union fee is not in `Employer.TotalCost`.** It is not insurance, but the employer pays it.
- **Region is the place of work**, not the head office or the employee's home. The library does not infer it; pass the region where the employee works.

Planned work is listed in the [roadmap](docs/design.md#12-roadmap).

---

## 🛠️ Command-Line Interface (CLI)

### Install CLI
```bash
go install github.com/HuynhNhatKhanh/vnsalary/cmd/vnsalary@latest
```
*(Or compile directly: `go build -o vnsalary ./cmd/vnsalary`)*

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
BenchmarkGrossToNet-16      3694824           325.3 ns/op         320 B/op          4 allocs/op
BenchmarkNetToGross-16       108325         18020   ns/op       14864 B/op        194 allocs/op
```

- **Forward calculation**: sub-microsecond; allocations come from the bracket lines and the copied policy reference in the result.
- **Inverse calculation**: interpolation on the linear segment, then an ascending scan of ±50 VND extended down to the smallest gross with the same net, with a binary-search fallback. See [docs/design.md](docs/design.md#7-net--gross) for why the result is always the smallest gross.

---

## 🧪 Testing

The library is verified with a comprehensive test suite:
- **Golden Test Dataset (`testdata/golden.json`):** 25 cases, each with a `source` and `verified_at` field stating how it was checked (hand calculation, an independent reference implementation, or an external calculator). 10 cases were matched to the VND against the TopCV calculator on 2026-10-13; cases not yet cross-checked with an external tool say so explicitly. 11 cases use odd amounts such as 23,456,789 so that every rounding step is exercised, and pin each insurance item.
- **Round-Trip Invariance Testing:** for every gross `g`, `NetToGross(Net(g))` returns the **smallest** gross with the same net, so `Net(NetToGross(Net(g))) == Net(g)` and `NetToGross(Net(g)).Gross <= g` (several grosses can share one net because of rounding).
- **Policy invariants:** `TestPoliciesInvariants` fails if a new policy breaks an assumption of the inverse algorithm (ordering, social/health cap below unemployment cap, top bracket 35%, sources present).
- **Fuzz Testing:** Includes `FuzzGrossToNet_NetToGross` verifying property invariance on arbitrary inputs.
- **Security & Integrity:** Verifies immutability of `PolicyAt` slices, overflow bounds checking (> 10¹⁴ VND), and negative net rejection when insurance exceeds gross.
- **CLI:** every command, flag spelling, language and exit code is tested in-process; table output is pinned by snapshot files.
- **CI:** Go 1.22 (minimum), oldstable and stable, with the race detector, `go vet`, golangci-lint, gofmt and a fuzz smoke run.

```bash
go test -cover ./...
```
Statement coverage at v0.1.0: library **92.3%**, CLI **96.5%**.

---

## 🤝 Contributing

Law changes and calculation mismatches are the most useful reports; both have issue templates. See [CONTRIBUTING.md](CONTRIBUTING.md) for how a new policy is added and how versions are numbered, [CHANGELOG.md](CHANGELOG.md) for releases, and [docs/design.md](docs/design.md) for the design and its decisions.

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.
