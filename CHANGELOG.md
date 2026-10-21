# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/) with the rules in
[CONTRIBUTING.md](CONTRIBUTING.md#versioning). In short:

- **MINOR**: a new policy (law change), a changed statutory parameter, any
  change of computed results, or new API.
- **PATCH**: bug fixes that do not change any computed amount, docs, CLI wording.
- **MAJOR**: breaking API change; from v1 on, the module path gets `/v2`.

Entries that change computed amounts are marked **(results change)**.

## [Unreleased]

## [0.6.0] - 2026-10-21

No change in any computed amount for existing inputs. New: employer options
for the accident insurance rate and the trade union fee.

### Added

- `Options.Employer` (`EmployerOptions`): `AccidentPerMille` overrides the
  employer occupational accident insurance rate (0.5% by default; 0.3% when
  approved, Decree 58/2020/NĐ-CP)
  ([#10](https://github.com/HuynhNhatKhanh/vnsalary/issues/10)), and
  `UnionFee` adds the 2% trade union fee on the social insurance salary to
  `Employer.TotalCost` (Law on Trade Unions 50/2024/QH15 Art. 29.1.b)
  ([#11](https://github.com/HuynhNhatKhanh/vnsalary/issues/11)). Both change
  the employer's cost only. `EmployerCost.UnionFee`;
  `Policy.AccidentPerMille`, `Policy.UnionFeePerMille`;
  `ErrInvalidAccidentRate`. CLI `--accident-rate`, `--union-fee`.

### Changed

- design.md: the open question on Law 09/2026/QH16 is closed; it and
  Resolution 43/2026/QH16 do not touch salary income.

## [0.5.0] - 2026-10-21

No change in any computed amount for existing inputs. New: 10% withholding,
foreign employees, non-residents.

### Added

- `FlatWithholding`: the 10% PIT withheld from a payment to a resident
  without a labour contract or with one under 3 months, from 2,000,000 per
  payment (Circular 111/2013/TT-BTC Art. 25.1.i), and from 5,000,000 for
  payments from 2026-07-01 (Decree 253/2026/NĐ-CP Art. 50.2)
  ([#7](https://github.com/HuynhNhatKhanh/vnsalary/issues/7)).
  `FlatResult.Payment` feeds the withholding into `Annual`.
- `Policy.FlatWithholdingThreshold`.
- `vnsalary flat <amount> [--date]`.
- `Options.Foreigner`: foreign nationals pay no unemployment insurance,
  employee or employer (Law on Employment 74/2025/QH15 Art. 2.1); social and
  health insurance unchanged
  ([#9](https://github.com/HuynhNhatKhanh/vnsalary/issues/9)). CLI
  `--foreigner`; `"foreigner"` in the `annual` input file.
- `Options.NonResident`: 20% of taxable income, without family or insurance
  deductions (Decree 253/2026/NĐ-CP Art. 64; Law on PIT 2007 Art. 26
  before 2026) ([#8](https://github.com/HuynhNhatKhanh/vnsalary/issues/8)).
  `NetToGross` stays exact and returns the smallest gross. Warning
  `dependents_ignored_non_resident`; `Policy.NonResidentBrackets`; CLI
  `--non-resident`.

### Fixed

- CLI: `diff` prints warnings (once per code across both periods); it
  printed none before.
- CLI: an extra argument is now an error. Flags after it, as in
  `gross 35000000 x --bonus 5000000`, were ignored without notice.

## [0.4.0] - 2026-10-20

No change in any monthly computed amount. New API for annual finalization.

### Added

- `Annual` computes the annual PIT finalization of a tax year from its
  payments: the yearly tax on all salary income versus the tax withheld, and
  the balance to pay or refund
  ([#6](https://github.com/HuynhNhatKhanh/vnsalary/issues/6)). The personal
  deduction counts 12 months whatever the months worked; dependents count by
  month; charity, medical (up to 23M) and tuition (up to 24M) deductions apply
  from tax period 2026 (Decree 253/2026/NĐ-CP Art. 48–49). Tax years from 2025.
- `Payment`, `Result.Payment`, `AnnualOptions`, `AnnualDeductions`,
  `AnnualResult`; errors `ErrYearOutOfRange`, `ErrPaymentOutsideYear`,
  `ErrTooManyPayments`, `ErrInvalidDependentMonths`; warning
  `health_education_not_deductible`.
- `Policy.HealthcareDeductionCap` and `Policy.EducationDeductionCap`, and the
  legal sources for finalization in the 2024-07-01 and 2026 policies.
- `vnsalary annual <file.json>`: finalization from a JSON file of payments,
  each given as gross (withholding computed with `GrossToNet`) or as the
  amounts on the payslip; table in Vietnamese or English, or `--json`.
  Unknown fields are rejected.
- Four golden finalization cases (job change, December bonus, a steady 2025
  salary, 2026 deductions), computed by hand and by an independent script.

### Fixed

- CLI: a negative `--ins-base` is now an error instead of being ignored.

## [0.3.1] - 2026-10-16

No change in any computed amount or in the library API.

### Added

- `vnsalary version` (also `--version`): the release version, or the module
  version for builds made with `go install …@vX.Y.Z`.
- Prebuilt CLI binaries for Linux, macOS and Windows (amd64 and arm64) with
  SHA-256 checksums, attached to each GitHub release by the `Release` workflow
  ([#12](https://github.com/HuynhNhatKhanh/vnsalary/issues/12)).
- pkg.go.dev examples for bonuses (`BonusTax`), exempt allowances and warnings.
- Two golden cases for the policy from 2026-07-01 checked against TopCV
  (36 cases, 12 matched to the VND against TopCV).

## [0.3.0] - 2026-10-15

### Added

- `Options.Extras` for parts of gross that the law treats differently
  ([#3](https://github.com/HuynhNhatKhanh/vnsalary/issues/3),
  [#4](https://github.com/HuynhNhatKhanh/vnsalary/issues/4)). Gross stays the
  total paid in the month; the zero value keeps earlier results.
  - `Overtime`: fully tax-exempt from tax period 2026 (Law 109/2025 Art. 4.8,
    Decree 253/2026 Art. 26); taxed before 2026 with a warning.
  - `MealCash`: tax-exempt up to 1,200,000/month from 2026-07-01 (Decree
    253/2026 Art. 8.2.g); 730,000 before, with a warning that it is not
    confirmed.
  - `OtherExempt`: other allowances exempt under the employer's regulation.
  - `Bonus`: taxed in the month paid.
  - Overtime and bonuses are outside the insurance salary when
    `InsuranceBase` is nil.
- `Result.Exempt` (itemized exempt income) and `Result.BonusTax` (tax caused by
  the bonus), both omitted from JSON when no extras are given.
- `Policy.MealCashCap`, `Policy.MealCashCapVerified`, `Policy.OvertimeFullyExempt`.
- `ErrExtrasExceedGross`; warnings `overtime_taxed_before_2026`,
  `meal_cap_unverified`, `insurance_salary_includes_allowances`.
- CLI flags `--overtime`, `--meal`, `--other-exempt`, `--bonus`.
- `NetToGross` with extras: holds them fixed and returns the smallest total gross.

## [0.2.0] - 2026-10-15

### Added

- `Result.Warnings` (`[]Warning` with `Code` and `Message`, omitted from JSON
  when empty) for inputs that are computed as given but worth checking.
  Amounts never change.
- Warning `WarnInsuranceSalaryBelowRegionalMinimum` when the insurance salary
  (`InsuranceBase`, or gross) is positive but below the regional minimum wage
  ([#5](https://github.com/HuynhNhatKhanh/vnsalary/issues/5)). It is a warning,
  not an error, because part-time contracts can legally be lower and the legal
  floor of the social insurance salary is the reference level.
- CLI prints warnings as `[!]` lines in Vietnamese or English; `--json`
  includes them.

## [0.1.0] - 2026-10-14

First public release.

### Added

- `GrossToNet` and `NetToGross` for resident employees on labour contracts of
  3 months or more: compulsory insurance (BHXH 8%, BHYT 1.5%, BHTN 1%) with
  caps, family deductions, progressive PIT, and employer contributions
  (17.5% / 3% / 1%) with total employer cost.
- `NetToGross` returns the smallest gross whose net equals the requested net.
  Net never decreases as gross grows (steps of 0 or 1 VND), so every net up to
  Net(`MaxGross`) is reachable.
- Policies selected by salary payment date (cash basis), each with legal sources:
  - from 2024-07-01: base salary 2,340,000; regional minimums (Decree 74/2024);
    deductions 11M / 4.4M; 7-bracket schedule;
  - from 2026-01-01: deductions 15.5M / 6.2M (Resolution 110/2025);
    5-bracket schedule (Law 109/2025/QH15); regional minimums (Decree 293/2025);
  - from 2026-07-01: base salary 2,530,000 (Decree 161/2026), social/health
    cap 50.6M.

  Sources cite the article and clause, checked against the legal texts on
  2026-10-13 (see `docs/design.md` §14).
- `PolicyAt` (returns a deep copy), `Policy.RegionalMinSalary`,
  `Policy.MaxUnemploymentSalary`, `RoundToThousand`.
- Optional fixed insurance base (`Options.InsuranceBase`).
- Input bounds: `MaxGross` (10^14 VND) and `MaxDependents` (50).
- CLI `vnsalary` with `gross`, `net` and `diff` commands, `--json`, and
  `--lang vi|en`. Exit codes: 0 success, 1 calculation error, 2 usage error.
- Tests: 25 golden cases with recorded sources (11 with odd amounts that pin
  each rounding step), round-trip, minimality and step properties, fuzzing,
  policy invariants, CLI tests with output snapshots.
- CI on Go 1.22, oldstable and stable: vet, race detector, golangci-lint,
  gofmt, fuzz smoke test.

### Known limitations

- No tax-exempt allowances, overtime premium, bonuses or 13th-month salary.
- No annual PIT finalization; no 10% flat withholding (contracts under
  3 months) or 20% non-resident rate.
- Employer accident/occupational disease insurance is fixed at 0.5%
  (some sectors may pay 0.3%); the 2% trade union fee is not included.
- The caller chooses the region; an insurance base below the regional
  minimum wage is not rejected.

[Unreleased]: https://github.com/HuynhNhatKhanh/vnsalary/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/HuynhNhatKhanh/vnsalary/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/HuynhNhatKhanh/vnsalary/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/HuynhNhatKhanh/vnsalary/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/HuynhNhatKhanh/vnsalary/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/HuynhNhatKhanh/vnsalary/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/HuynhNhatKhanh/vnsalary/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/HuynhNhatKhanh/vnsalary/releases/tag/v0.1.0
