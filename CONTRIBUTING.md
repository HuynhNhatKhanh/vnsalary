# Contributing to vnsalary

Thanks for helping. Issues and pull requests may be written in English or
Vietnamese.

The most valuable contributions are **law changes** (a new decree, resolution
or law that changes a number) and **calculation mismatches** (vnsalary
disagrees with your payslip or another calculator). Both have issue templates.

## Ground rules

1. **Every number has a source.** A statutory value enters `policies.go` only
   with the document number that sets it. Drafts (*dự thảo*) do not count; wait
   until the document is signed and published.
2. **Money is `int64` VND.** No `float64` anywhere in the calculation.
3. **Standard library only** in the library and the CLI.
4. **Golden values are never produced by vnsalary itself.** An expected value in
   `testdata/golden.json` comes from a hand calculation, an independent
   implementation, or an external calculator, and its `source` field says which
   one. If nothing external has checked it yet, `source` says so.

## Development

```bash
make test     # go test -v -cover ./...
make race     # race detector (needs linux/amd64 or similar 64-bit platform)
make lint     # golangci-lint v2, config in .golangci.yml
make fuzz     # 60 s of FuzzGrossToNet_NetToGross
make golden   # rewrite CLI output snapshots after an intended change
make check    # fmt + vet + lint + test
```

CI runs on Go 1.22 (the minimum in `go.mod`), `oldstable` and `stable`. Do not
use language or library features newer than Go 1.22 unless the minimum is
raised in a MINOR release and noted in the CHANGELOG.

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):
`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`.

## Adding a policy (law change)

A *policy* is the full set of parameters in force from one date. Policies live
in `defaultPolicies` in [`policies.go`](policies.go), **newest first**.

1. **Open a "Law change" issue** with the document number, a link to the
   official text, the signing date, the effective date and the old → new values.
2. **Find the effective date that applies to salaries.** The library picks the
   policy by the salary **payment date** (cash basis). For changes tied to a
   tax period (*kỳ tính thuế*), use the first day of that period, e.g. a change
   "from tax period 2026" starts on `2026-01-01`.
3. **Add the policy.** Copy the newest entry to the top of `defaultPolicies`,
   then change only what the new document changes:
   - `EffectiveFrom`: midnight UTC of the effective date; `Ref.EffectiveFrom`
     the same date as `YYYY-MM-DD`;
   - `Ref.Description` (English) and `Ref.DescriptionVI` (Vietnamese);
   - `Ref.Sources`: one line per parameter, with the document name in
     Vietnamese and what it sets, for example
     `"Nghị định 161/2026/NĐ-CP (Lương cơ sở 2.530.000đ từ 01/07/2026)"`.
     Keep the sources of parameters carried over from the previous policy.
4. **Run `go test ./...`.** `TestPoliciesInvariants` checks the assumptions of
   the inverse algorithm: order, social/health cap below the unemployment cap in
   every region, cap = 20 × base salary, increasing brackets, top rate 35%,
   sources present. `TestNetSteps_ZeroOrOne` checks that net grows by 0 or
   1 VND per VND of gross, which the inverse relies on; a change of
   contribution rates can break it. If the new law breaks one of these
   assumptions, review the inverse in `inverse.go` and
   [docs/design.md §7](docs/design.md#7-net--gross), not just the test.
5. **Add golden cases** to `testdata/golden.json`, with `source`,
   `verified_at` and `notes`:
   - the same input on the day before and on the effective date, at an amount
     the change affects;
   - at least one odd amount (not a multiple of 1,000), so rounding is covered.
6. **Add a focused test** if the change moves a cap or threshold, like
   `TestBaseSalaryChange_2026_07`.
7. **Update the documentation:** the policy tables in `README.md` and
   `README.vi.md`, [`docs/design.md`](docs/design.md), and the CLI's default
   `diff` dates in `cmd/vnsalary/main.go` if the comparison should move.
   Run `make golden` if CLI output changes and review the snapshot diff.
8. **Add a CHANGELOG entry** under `[Unreleased]`, marked **(results change)**.

## Reporting a calculation mismatch

Include the input (gross or net, dependents, region, payment date, insurance
base), the result you expected, and where it comes from (payslip, calculator
URL). A difference of a few VND is usually a rounding convention; see
"Rounding convention" in the README. A difference of 1,000 VND or more is
a bug.

## Versioning

The project follows [Semantic Versioning](https://semver.org/).

| Change | Version |
|---|---|
| New policy, changed statutory parameter, or any change of computed amounts (including a bug fix that changes amounts) | **MINOR** |
| New backward-compatible API | **MINOR** |
| Bug fix that changes no computed amount, docs, CLI wording, performance | **PATCH** |
| Breaking API change | **MAJOR**: from v1.0.0 on, the module path becomes `github.com/HuynhNhatKhanh/vnsalary/v2` |

Fixes that change amounts are released as MINOR so that users who pin a minor
version keep stable results. While the version is `v0.x`, breaking API changes
are released as MINOR and listed under **Breaking** in the CHANGELOG.

### Releasing

1. Move the `[Unreleased]` entries to a new version heading with today's date
   and update the compare links at the bottom of `CHANGELOG.md`.
2. Make sure CI is green on `main`.
3. Tag and push: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`.
   The `Release` workflow then tests, builds the CLI for Linux, macOS and
   Windows (`scripts/build-release.sh`), creates the GitHub release from the
   CHANGELOG section and attaches the archives and `checksums.txt`.
4. Request the version on the module proxy so pkg.go.dev picks it up:
   `GOPROXY=https://proxy.golang.org go list -m github.com/HuynhNhatKhanh/vnsalary@vX.Y.Z`.
5. Check the release page: six archives, `checksums.txt`, and the notes.
