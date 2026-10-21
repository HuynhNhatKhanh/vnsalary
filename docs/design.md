# vnsalary design

This document records how vnsalary works and why: scope, data model,
algorithms, rounding, legal-date semantics, testing and the decisions taken.
It replaces the Vietnamese draft spec written before the first line of code.
User documentation is in the [README](../README.md); the procedure for law
changes is in [CONTRIBUTING.md](../CONTRIBUTING.md).

## 1. Goals

- Compute Vietnamese **gross ↔ net** salary for a resident employee to the
  exact VND: compulsory insurance, family deductions, progressive personal
  income tax (PIT) and employer cost.
- Apply **the law in force on a given date**, with every statutory number
  traceable to a legal document.
- Be a small, dependency-free Go library first; the CLI and any web demo are
  clients of the public API.

When the project started (October 2026) no Go library covered this; existing
tools were web apps or JS/TS/Python scripts, and published 2026 parameters
disagreed between sources. A parameter set with citations is useful in itself.

## 2. Scope

**In v0.1.0**

| Area | Detail |
|---|---|
| Gross → Net, Net → Gross | exact to 1 VND; inverse returns the smallest gross |
| Employee insurance | BHXH 8%, BHYT 1.5%, BHTN 1%, with caps |
| Insurance caps | BHXH/BHYT: 20 × base salary; BHTN: 20 × regional minimum wage |
| Insurance base | gross by default, or a fixed `InsuranceBase` |
| Family deductions | personal + per dependent |
| PIT | progressive by bracket: 7 brackets until 2025, 5 brackets from 2026 |
| Employer cost | BHXH 17.5% (incl. 0.5% occupational accident), BHYT 3%, BHTN 1% |
| Policy by date | `Options.Date` selects the parameter set |
| CLI | `gross`, `net`, `diff` (with `--foreigner`, `--non-resident`), `flat`, `annual` (JSON file of payments), `--json`, `--lang vi\|en` |
| Extras (v0.3.0) | overtime, cash meal allowance, other exempt allowances and bonus inside gross; exempt income, insurance salary and the tax caused by the bonus |
| Non-residents (v0.5.0) | `Options.NonResident`: 20% of taxable income, no deductions; `NetToGross` still exact and minimal; `--non-resident` |
| Foreign employees (v0.5.0) | `Options.Foreigner`: no unemployment insurance for employee or employer; `--foreigner` |
| 10% withholding (v0.5.0) | `FlatWithholding`: 10% of a payment to a resident without a labour contract or with one under 3 months, from the threshold of the payment date (2M, 5M from 2026-07-01); `vnsalary flat` |
| Annual finalization (v0.4.0) | yearly tax on all payments of a tax year vs. the tax withheld; 12 months of personal deduction, dependent months, charity, medical and tuition deductions (§8) |

**Later:** see the roadmap (§13). Known simplifications of v0.1.0 are listed
in §12.

**Out of scope:** other PIT income types (securities, rentals, ESOP),
persistence, user accounts.

## 3. Data model

```go
type VND int64            // whole đồng; never float64

type Options struct {
	InsuranceBase *VND      // nil: insurance on gross
	Dependents    int       // 0..MaxDependents
	Region        Region    // 1..4, regional minimum wage zone
	Date          time.Time // salary payment date; selects the policy
}

func GrossToNet(gross VND, opts Options) (Result, error)
func NetToGross(net VND, opts Options) (Result, error)
func PolicyAt(date time.Time) (Policy, error) // deep copy
```

Both directions take the same `Options`, so no field is silently ignored in
one direction. `Result` carries every intermediate amount (each insurance item,
deductions, assessable income, tax per bracket, employer items) and the
`PolicyRef` with its legal sources, so a caller can show how a number was
obtained.

**Policies** are plain structs in `defaultPolicies` (`policies.go`), newest
first, compiled into the binary: no files are read at run time and there is no
mutable global state. A policy holds the base salary, the social/health cap,
the four regional minimum wages, the two deductions and the brackets.

Contribution rates are constants in the calculation because nothing in the
covered period changes them; a rate change would move them into `Policy`.

| Insurance | Employee | Employer | Basis in force in 2026 |
|---|---|---|---|
| Social (BHXH) | 8% | 17.5% = 14% + 3% + 0.5% occupational accident | Law on Social Insurance 41/2024/QH15 Art. 33–34; Decree 58/2020 (0.5%) |
| Health (BHYT) | 1.5% | 3% | 4.5% split 1/3 : 2/3, Decree 188/2025 |
| Unemployment (BHTN) | 1% | 1% | Decree 374/2025 (the law sets "at most 1%") |

**Immutability.** The public `PolicyAt` returns a deep copy (brackets and
sources are copied); the internal `policyAt` returns a pointer for speed and
never escapes the package. `Result.Policy` is cloned at the API boundary.
`TestPolicyAt_Immutability` and `TestResultDoesNotLeakSharedData` guard this.

**Bounds.** Amounts are limited to `MaxGross` = 10^14 VND and dependents to
`MaxDependents` = 50, so every intermediate product fits in `int64`.

## 4. Choosing the policy: payment date

Taxable income from salaries is determined at the time the income is **paid**
(cash basis): Law on PIT 109/2025/QH15 Art. 8.3, and before it Circular
111/2013/TT-BTC Art. 8.2.b (replaced by Circular 87/2026/TT-BTC from
2026-07-01). `Options.Date` is therefore the payment date, compared by
calendar date in the caller's time zone: December 2025 salary paid on
5 January 2026 uses the 2026 deductions and brackets, which the tax authority
confirmed in January 2026.

The same date also selects the insurance parameters. This is a simplification:
insurance is legally owed for the month worked, so a salary paid after a base
salary change can, in rare cases, use the newer insurance cap one month early.
Callers who need the exact split can compute insurance and tax with two dates.

Covered policies:

| From | Change |
|---|---|
| 2024-07-01 | base salary 2,340,000; regional minimums (Decree 74/2024); deductions 11M / 4.4M; 7 brackets |
| 2026-01-01 | deductions 15.5M / 6.2M; 5 brackets; regional minimums (Decree 293/2025) |
| 2026-07-01 | base salary 2,530,000 → social/health cap 50.6M |

Dates before 2024-07-01 return `ErrDateOutOfRange`. The full source list per
policy is in `policies.go` and the README.

## 5. Gross → Net

1. Select the policy by date.
2. Insurance salary = `InsuranceBase`, or gross − `Extras.Overtime` −
   `Extras.Bonus` (both vary month to month, so they are outside the salary
   "paid regularly and stably", Law 41/2024 Art. 31.1.b). Social and health use
   `min(salary, 20 × base salary)`; unemployment uses
   `min(salary, 20 × regional minimum)`.
3. Each insurance item = rate × capped salary, **rounded half-up to 1 VND**.
   If the total exceeds gross (possible only with a fixed `InsuranceBase`),
   return `ErrInsuranceExceedsGross` instead of a negative net.
4. Taxable income = gross − exempt income, where exempt income is overtime
   (from tax period 2026, Law 109/2025 Art. 4.8) + min(meal cash, policy cap)
   + other exempt allowances. Assessable income = max(0, taxable income −
   insurance − deductions).
5. Tax = Σ over brackets of rate × part of assessable income in the bracket,
   each **rounded half-up to 1 VND**.
6. Net = gross − insurance − tax. Employer items are computed the same way.
7. With a bonus, `BonusTax` = tax − tax on (assessable income without the
   bonus). The bonus does not change insurance, so this is exactly the tax the
   bonus adds in that month.

## 6. Rounding convention

All arithmetic is integer: `roundDiv(n, d) = (n + d/2) / d` for n ≥ 0.

| Amount | Rule |
|---|---|
| each employee and employer insurance item | half-up to 1 VND, separately |
| tax of each bracket | half-up to 1 VND, then summed |
| net, totals | exact sums of rounded items |
| net to 1,000 VND | not applied (company policy); `RoundToThousand` is provided |

Other calculators round differently (once on the total, or by truncation).
The difference is a few VND at most. Example, gross 20,000,028 in 2026: the
insurance items are 1,600,002.24 + 300,000.42 + 200,000.28; per item this gives
2,100,002, once on the total 2,100,003, so the net differs by 1 VND. In one
sample (gross 20.0M–20.2M, 2026, no dependents) about 28% of amounts differ by
1 VND between the two conventions. The golden case
`2026 - 20,000,028 …` pins this example.

## 7. Net → Gross

**Shape of the function.** Ignoring rounding, net = f(gross) is continuous,
increasing and piecewise linear. Its slope changes at the breakpoints: the
social/health cap, the unemployment cap, the gross where tax starts, and each
bracket boundary (about 10 points). Between two breakpoints the slope is
`(1 − insurance rate) × (1 − tax rate)`.

**Steps of 0 or 1 VND.** With rounding, raising gross by 1 VND changes net by
0 or +1 VND, never more and never negative. The reason: each item's exact value
grows by less than 1 VND per VND, so each rounded item grows by 0 or 1; and
for the rates 8%, 1.5% and 1% no two insurance items cross a half at the same
gross (checking gross mod 200 shows the three crossing sets are disjoint).
So insurance grows by at most 1 per step, assessable income by 0 or 1, and
tax by 0 or 1. Consequences:

- net is non-decreasing in gross, so "the smallest gross with this net" is
  well defined and found by a left-most search;
- every net from 0 to Net(`MaxGross`) is reachable; `ErrNetUnreachable` occurs
  only above Net(`MaxGross`).

`TestNetSteps_ZeroOrOne` walks every breakpoint of every policy and fails if a
step is not 0 or 1 (a mutation giving health insurance the same rate as social
insurance makes it fail). A future rate change must pass this test.

**Algorithm.**

1. Compute the breakpoints (gross, net) for the policy and options. With a
   fixed `InsuranceBase`, insurance is constant and only the tax breakpoints
   remain. Extras are held fixed: the lowest valid gross is their sum, the
   insurance caps move up by overtime + bonus, and the tax thresholds move by
   the exempt income. The insurance salary still grows 1 VND per VND of gross,
   so the step property above holds (`TestNetSteps_ZeroOrOne` includes extras).
2. Find the segment containing the target net and interpolate the gross with
   integer arithmetic. Above the last breakpoint the slope is 65% (insurance
   capped, top rate 35%).
3. Scan gross upward from `approx − 50` to `approx + 50` VND and stop at the
   first gross with the target net.
4. If the window misses, binary-search the segment for a gross with the
   target net.
5. From the gross found, step down while the net stays the same.

Because net is non-decreasing, the grosses sharing one net form a single
interval (a few VND wide), so step 5 always ends at its smallest element:
the result is minimal by construction, not only within the scan window.
`TestNetToGross_RoundTripProperty`, the golden tests (the gross 1 VND below
the result gives a smaller net) and fuzzing check it. The window keeps the
common case at roughly 13 µs on the author's machine.

**Non-residents.** A flat 20% on taxable income does not shrink when
insurance grows, and with extras the tax base and the insurance base are
offset, so net can fall by 1 VND as gross rises. The grosses sharing one net
then need not be contiguous. For non-residents the walk to the smallest gross
checks a 32 VND window below each match: net grows by at least 0.58 VND per
VND of gross and the varying roundings move it by at most 5 VND, so nothing
further below can match (`TestSmallestGross_CrossesStepDown`,
`TestNonResident_NetStepsAndSmallestGross`).

The breakpoint estimates assume the social/health cap is below the
unemployment cap in every region (50.6M vs. at least 74M today);
`TestPoliciesInvariants` fails if a new policy breaks that, or the ordering,
the 20× cap, increasing brackets, the 35% top rate, or missing sources.

## 8. Annual finalization

`Annual(year, payments, opts)` settles the salary PIT of a resident individual
for one tax year (quyết toán thuế TNCN).

1. **Payments.** Each `Payment` is what a payslip or a withholding
   certificate shows: taxable income, compulsory insurance and the tax
   withheld. `Result.Payment(date)` converts a `GrossToNet` result. A payment
   belongs to the tax year of its payment date (cash basis, §4); a payment
   dated in another year is an error, not silently dropped. Payments from all
   payers go in, including those withheld at the 10% flat rate.
2. **Deductions.**
   - Personal: 12 × the monthly deduction, whatever the months worked
     (Decree 253/2026/NĐ-CP Art. 48.1.b; Circular 111/2013/TT-BTC Art. 9.1.c.1
     before 2026).
   - Dependents: `DependentMonths` × the monthly deduction. The caller counts
     each dependent from the month the duty of support arose, which may be
     earlier than the registration (Art. 48.2.a).
   - Charity (Art. 49.1), medical expenses up to 23,000,000 and tuition up to
     24,000,000 per year (Art. 49.2), from tax period 2026 (Art. 69.1.a).
     Before 2026 medical and tuition expenses are ignored with the warning
     `health_education_not_deductible`.
3. **Tax.** Assessable income = taxable income − insurance − deductions, not
   below zero. The yearly schedule is the monthly one with every limit × 12
   (Law 109/2025/QH15 Art. 9.2: 120 / 360 / 720 / 1,200 million; before 2026
   60 / 120 / 216 / 384 / 624 / 960 million), rounded as in §6.
4. **Balance** = tax − withheld: positive is still to pay, negative is
   refundable.

**One policy per year.** Deductions and brackets are set per tax period, so
the policy in force on 1 January applies to the whole year.
`TestPoliciesInvariants` fails if a policy changes them within a year. The
base salary change on 2026-07-01 only changes insurance amounts, which arrive
inside the payments.

**Why the balance is not zero.** With the same salary every month the yearly
tax equals the withholding up to rounding (at most 6 VND per bracket; tested).
It differs when income is uneven (a bonus month is withheld at its monthly
bracket), when the taxpayer changed employers or did not work the whole year
(the personal deduction still counts 12 months), when dependents were
registered late, or when charity, medical and tuition deductions are claimed.

**Not covered:** voluntary pension and life insurance deductions
(Art. 46.2.a), tax reductions for disasters and serious illness (Art. 40),
the month-based personal deduction of foreigners arriving or leaving during
the year (Art. 48.1.c), and who must or may file (Art. 51). Claiming the
Art. 49 deductions means the taxpayer files personally instead of
authorizing the employer (Art. 51.3); the library computes amounts and does
not decide filing obligations.

## 9. Testing

| Layer | What it proves |
|---|---|
| table tests | brackets, caps, regions, dependents, ±1 VND around every bracket limit, dates before/on/after each policy change |
| golden data (`testdata/golden.json`) | 36 cases with `source`, `verified_at`, `notes`; 12 matched against TopCV; 11 odd amounts pin each insurance item and the tax rounding; 9 cover overtime, allowances and bonuses |
| annual golden data (`testdata/annual_golden.json`) | 4 finalization cases (job change, December bonus, steady 2025 salary, 2026 deductions) computed by hand and by an independent script; `ExampleAnnual` reproduces the job-change case through `GrossToNet` |
| independent reference | expected values for odd amounts come from a separate implementation with exact fractions, not from vnsalary |
| properties | `Net(NetToGross(n)) == n`; `NetToGross(Net(g)).Gross ≤ g` and minimal; net steps of 0 or 1; a steady salary's yearly tax equals its withholding up to rounding |
| invariants | `TestPoliciesInvariants` on the policy table |
| fuzzing | `FuzzGrossToNet_NetToGross` on arbitrary amounts and options |
| CLI | every command, flag spelling, language and exit code, with output snapshots |
| CI | Go 1.22, oldstable, stable; `-race`, vet, golangci-lint, gofmt, fuzz smoke |

Golden cases are labelled honestly: cases not yet compared with an external
calculator say so in `source`. Comparing them with two public calculators is
a release task, recorded by updating `source` and `verified_at`.

## 10. Repository layout

The layout follows the Go team's
[Organizing a Go module](https://go.dev/doc/modules/layout) and common
libraries (`google/uuid`, `shopspring/decimal`, `golang-jwt/jwt`): the package
at the root, the CLI in `cmd/`, no `pkg/`.

```
vnsalary/
├── doc.go, vnsalary.go, annual.go, policy.go, policies.go, inverse.go, rounding.go, errors.go
├── *_test.go, example_test.go, testdata/golden.json, testdata/annual_golden.json
├── cmd/vnsalary/            CLI (stdlib flag), tests and output snapshots
├── docs/design.md           this document
├── .github/                 CI workflow, issue templates
└── README.md, README.vi.md, CHANGELOG.md, CONTRIBUTING.md, LICENSE
```

The web demo will be a separate repository, `vnsalary-web`, importing the
library like any user: no sub-module tags, no AWS dependencies in the library,
and a real test of how pleasant the API is.

## 11. Decisions

| Date | Decision | Reason |
|---|---|---|
| 2026-10-13 | Money is `int64` VND | exact; `float64` cannot represent 0.1 |
| 2026-10-13 | Round half-up per item and per bracket | matches line items on a payslip; documented with its effect (§6) |
| 2026-10-13 | No rounding of net to 1,000 VND in the core | a company policy, not law; `RoundToThousand` offered |
| 2026-10-13 | Policy selected by payment date | PIT cash basis (Law 109/2025/QH15 Art. 8.3; formerly TT 111/2013 Art. 8.2.b) |
| 2026-10-13 | Inverse steps down to the interval's left edge | minimality holds by construction, not by the size of the scan window |
| 2026-10-13 | First policy 2024-07-01 | current base salary period when the project started |
| 2026-10-13 | Only issued documents become policies | drafts change; no "draft" flag in the API |
| 2026-10-13 | Policy for 2026-07-01 added | base salary 2,530,000 (Decree 161/2026) raises the social/health cap |
| 2026-10-13 | `go 1.22` minimum, CI on 1.22/oldstable/stable | a library should not force a new toolchain; CI proves the floor |
| 2026-10-13 | English godoc and comments, Vietnamese legal names | pkg.go.dev audience; legal names are not translated |
| 2026-10-13 | CLI defaults to Vietnamese, `--lang en` | primary users read Vietnamese |
| 2026-10-13 | Versioning: law or result change → MINOR | users can pin a minor version for stable numbers |
| 2026-10-15 | Gross is the total paid in the month; `Extras` itemize parts of it ([#3](https://github.com/HuynhNhatKhanh/vnsalary/issues/3), [#4](https://github.com/HuynhNhatKhanh/vnsalary/issues/4)) | one number users already know (the payslip total); `Extras` exceeding gross is an error |
| 2026-10-15 | Default insurance salary excludes overtime and bonus only | the only parts with a clear legal basis (not regular and stable); meal and other allowances depend on the contract, so a warning points to `InsuranceBase` |
| 2026-10-15 | Overtime before 2026 is taxed with a warning, not an error | before 2026 only the premium over normal pay was exempt and the library cannot split it; callers put the exempt part in `OtherExempt` |
| 2026-10-15 | Meal cap 730,000 before 2026-07-01, flagged unverified | the reference circular was repealed and no replacement figure was found; computing with a warning beats rejecting historical payrolls |
| 2026-10-15 | `Result.Exempt` is a pointer, `BonusTax` omitempty | JSON stays identical to v0.2.0 when no extras are given |
| 2026-10-15 | Insurance salary below the regional minimum is a warning, not an error ([#5](https://github.com/HuynhNhatKhanh/vnsalary/issues/5)) | the legal floor of the social insurance salary is the reference level (Law 41/2024/QH15 Art. 31.1); the regional minimum binds full-time pay (Labour Code 2019 Art. 90.2), so part-time contracts can legally be lower. Warnings never change amounts |
| 2026-10-20 | `Annual` takes payments (taxable income, insurance, tax withheld), not gross ([#6](https://github.com/HuynhNhatKhanh/vnsalary/issues/6)) | that is what payslips and withholding certificates show; several employers and 10% withholding need no extra options |
| 2026-10-20 | Yearly schedule = monthly limits × 12, same rounding as monthly | the law's yearly column; the same convention keeps a steady salary's balance within a few VND |
| 2026-10-20 | First tax year 2025 | the policy data starts on 2024-07-01, so 2024 is not covered in full |
| 2026-10-20 | 10% threshold 5M from 2026-07-01, 2M before ([#7](https://github.com/HuynhNhatKhanh/vnsalary/issues/7)) | withholding is done at payment time under the rules then in force; Decree 253/2026 took effect on 2026-07-01 (see §14 for the reading of Art. 69.1.a) |
| 2026-10-21 | `Options.Foreigner` drops unemployment insurance only ([#9](https://github.com/HuynhNhatKhanh/vnsalary/issues/9)) | the Law on Employment defines the employee as a Vietnamese citizen (74/2025 Art. 2.1; 38/2013 Art. 3.1); social insurance exemptions depend on facts the library does not have (contract length, transfer status, age at signing), so they are documented, not modelled |
| 2026-10-21 | Non-resident rate as a one-bracket schedule, `Policy.NonResidentBrackets` ([#8](https://github.com/HuynhNhatKhanh/vnsalary/issues/8)) | reuses the bracket code and its rounding; a future change of rate is a policy edit |
| 2026-10-21 | `NetToGross` searches a 32 VND window below the first match for non-residents only | with extras their net can fall by 1 VND, so grosses with one net need not be contiguous; residents keep the one-step walk, proven by `TestNetSteps_ZeroOrOne`, and its speed |
| 2026-10-21 | Residence and nationality are separate options | a foreigner can be a tax resident and a Vietnamese citizen a non-resident; insurance follows nationality, tax follows residence |
| 2026-10-20 | `FlatWithholding` computes tax only; below the threshold `Withheld` is 0 | withholding on request and the deferral commitment are the individual's choice; insurance depends on the contract, not on the tax rule |

## 12. Known gaps

Simplifications in v0.1.0 that affect real payrolls are listed under
"Limitations" in the README. Those with a planned fix are tracked as issues:
configurable employer accident insurance rate
([#10](https://github.com/HuynhNhatKhanh/vnsalary/issues/10)) and the optional
trade union fee ([#11](https://github.com/HuynhNhatKhanh/vnsalary/issues/11)).
The region is the place of work; that is a documentation point, not a gap in
the calculation.

## 13. Roadmap

Planned work is tracked as GitHub issues grouped by
[milestone](https://github.com/HuynhNhatKhanh/vnsalary/milestones), ordered by
how often users are likely to need it:

- **v0.2.0** (released): warnings for inputs worth checking, starting with an
  insurance salary below the regional minimum wage.
- **v0.3.0** (released): tax-exempt income and bonuses, so real payslips can be computed.
- **v0.4.0** (released): annual PIT finalization
  ([#6](https://github.com/HuynhNhatKhanh/vnsalary/issues/6)): uneven income,
  job changes and late dependents make monthly withholding differ from the
  yearly tax.
- **v0.5.0**: 10% withholding for contracts under 3 months
  ([#7](https://github.com/HuynhNhatKhanh/vnsalary/issues/7)), foreign
  employees ([#9](https://github.com/HuynhNhatKhanh/vnsalary/issues/9)),
  non-residents ([#8](https://github.com/HuynhNhatKhanh/vnsalary/issues/8)).

Law changes are released as soon as the document is issued, outside this
order (for example the 2027 regional minimum wage,
[#16](https://github.com/HuynhNhatKhanh/vnsalary/issues/16)).

A web demo (`vnsalary-web`, a separate repository) and CLI binaries are planned
outside these milestones.

## 14. Open questions

- **Insurance month vs. payment date.** §4 uses one date for both. If users
  need it, `Options` could take a separate insurance month.
- **Meal allowance cap from 2026-01-01 to 2026-06-30.** Decree 253/2026 sets
  1,200,000 from 2026-07-01; the earlier 730,000 reference (Circular 26/2016)
  was repealed and no replacement was found. The library applies 730,000 with
  a warning.
- **Overtime beyond the legal hour limits** is taxable (Decree 253/2026
  Art. 26.3). How the excess is measured (monthly vs. yearly limits) is not
  settled in the sources found; the library leaves it to the caller.
- **Start of the 5,000,000 threshold.** Decree 253/2026 took effect on
  2026-07-01, but Art. 69.1.a applies its rules on residents' salary income
  from tax period 2026. The library uses 2026-07-01 for withholding, which
  happens at payment time; the choice only moves tax between withholding and
  finalization, not the year's total.
- **Amending law 09/2026/QH16** (April 2026) changes the PIT revenue threshold
  for business income; whether it touches Articles 9–10 of Law 109/2025 (salary
  brackets and deductions) has not been confirmed from the full text.

## 15. Legal sources checked

Checked against the full texts or official publications on 2026-10-13; the rows on overtime, allowances, bonuses and the insurance salary on 2026-10-15; the rows on annual finalization and 10% withholding on 2026-10-20 against Decree 253/2026/NĐ-CP in the Official Gazette (Công báo) No. 402 of 2026-07-16.

| Fact | Source |
|---|---|
| Base salary 2,530,000 from 2026-07-01 | Decree 161/2026/NĐ-CP Art. 3.2 (signed 2026-05-15) |
| Social insurance cap 20 × reference level | Law 41/2024/QH15 Art. 31.1.đ |
| Reference level = base salary until it is abolished | Law 41/2024/QH15 Art. 141.13 |
| Health insurance cap 20 × reference level | Law on Health Insurance Art. 14.5, as amended by Law 51/2024/QH15 |
| Unemployment cap 20 × regional minimum from 2026-01-01 | Law on Employment 74/2025/QH15 Art. 34.2 (Law 2013 Art. 58 before) |
| Regional minimums 5.31 / 4.73 / 4.14 / 3.70M from 2026-01-01 | Decree 293/2025/NĐ-CP Art. 3.1 |
| Deductions 15.5M / 6.2M from tax period 2026 | Resolution 110/2025/UBTVQH15; Law 109/2025/QH15 Art. 10.1 |
| 5 brackets for salaries from tax period 2026 | Law 109/2025/QH15 Art. 9.2 and Art. 29.2 |
| Taxable income at payment time | Law 109/2025/QH15 Art. 8.3; Circular 111/2013 Art. 8.2.b (replaced by Circular 87/2026/TT-BTC) |
| No legal rule on rounding insurance or PIT to the đồng | none found; see §6 |
| Overtime and night-work pay fully exempt from tax period 2026 (within legal limits) | Law 109/2025/QH15 Art. 4.8; Decree 253/2026/NĐ-CP Art. 26, Art. 69.1 |
| Cash meal allowance taxable only above 1,200,000/month from 2026-07-01 | Decree 253/2026/NĐ-CP Art. 8.2.g, Art. 69.1.b |
| Phone, uniform and similar allowances taxable only above the CIT-deductible level | Decree 253/2026/NĐ-CP Art. 8.2.đ |
| Bonuses taxable in the month paid, progressive withholding | Decree 253/2026/NĐ-CP Art. 8.2.i, 46.3, 50.1–2 |
| Insurance salary = wage + allowances + regular, stable additions | Law 41/2024/QH15 Art. 31.1.b; Decree 158/2025/NĐ-CP Art. 7.1 |
| Personal deduction counts 12 months at finalization | Decree 253/2026/NĐ-CP Art. 48.1.b; Circular 111/2013/TT-BTC Art. 9.1.c.1 (amended by Circular 92/2015/TT-BTC Art. 25.6) before 2026 |
| Dependents count from the month the duty of support arose | Decree 253/2026/NĐ-CP Art. 48.2.a |
| Medical up to 23M and tuition up to 24M per year deductible from tax period 2026 | Decree 253/2026/NĐ-CP Art. 49.2, Art. 69.1.a |
| Insurance and charity deducted in the year paid | Decree 253/2026/NĐ-CP Art. 46.2.a, Art. 49.5 |
| Yearly schedule: monthly limits × 12 | Law 109/2025/QH15 Art. 9.2 |
| 10% withholding for contracts under 3 months from 5,000,000 per payment; below it on request | Decree 253/2026/NĐ-CP Art. 50.2 |
| 10% withholding from 2,000,000 per payment before 2026-07-01 | Circular 111/2013/TT-BTC Art. 25.1.i |
| Non-residents: 20% of salary income from work in Vietnam, determined as for residents, no deductions | Decree 253/2026/NĐ-CP Art. 64.1–2 (Công báo No. 402); Law on PIT 2007 Art. 26.1, Circular 111/2013/TT-BTC Art. 18 before 2026 |
| Resident: 183 days in the calendar year or 12 consecutive months, or a regular place to stay | Decree 253/2026/NĐ-CP Art. 4, 5 |
| Unemployment insurance covers Vietnamese citizens only | Law on Employment 74/2025/QH15 Art. 2.1, Art. 31 (full text, xaydungchinhsach.chinhphu.vn); Law 38/2013/QH13 Art. 3.1, Art. 43 before 2026 |
| Foreign employees on contracts of 12 months or more pay social insurance, except intra-company transfers, retirement age at signing, treaties | Law 41/2024/QH15 Art. 2.2 (Công báo No. 987+988, 2024-08-25); Decree 143/2018/NĐ-CP before 2025-07-01 |
