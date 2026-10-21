package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HuynhNhatKhanh/vnsalary"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden with the current CLI output")

// runCLI runs the CLI in-process and returns the exit code and both output streams.
func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRun_ExitCodesAndMessages(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string // substring; "" means stdout must be empty
		wantStderr string // substring; "" means stderr must be empty
	}{
		{"gross ok", []string{"gross", "35000000", "-d", "1", "--date", "2026-01-15"}, 0, "30.843.750 đ", ""},
		{"net ok", []string{"net", "30843750", "-d", "1", "--date", "2026-01-15"}, 0, "35.000.000 đ", ""},
		{"diff ok", []string{"diff", "35000000", "-d", "1"}, 0, "2026-01-01", ""},
		{"help command", []string{"help"}, 0, "Cách dùng:", ""},
		{"help flag", []string{"--help"}, 0, "Cách dùng:", ""},
		{"help after command", []string{"gross", "35000000", "-h"}, 0, "Cách dùng:", ""},
		{"help in english", []string{"help", "--lang", "en"}, 0, "Usage:", ""},

		{"no args", nil, 2, "", "Cách dùng:"},
		{"flags without command", []string{"--lang", "en"}, 2, "", "Usage:"},
		{"unknown command", []string{"foo", "1"}, 2, "", "Lệnh không hợp lệ: foo"},
		{"unknown command en", []string{"foo", "1", "--lang=en"}, 2, "", "Unknown command: foo"},
		{"missing amount", []string{"gross"}, 2, "", "thiếu số tiền"},
		{"missing amount en", []string{"gross", "-lang", "en"}, 2, "", "missing amount"},
		{"bad amount", []string{"gross", "abc"}, 2, "", "số tiền không hợp lệ (abc)"},
		{"negative amount", []string{"gross", "--", "-5"}, 2, "", "số tiền không hợp lệ (-5)"},
		{"unknown flag", []string{"gross", "35000000", "-x"}, 2, "", "flag provided but not defined: -x"},
		{"bad flag value", []string{"gross", "35000000", "-d", "two"}, 2, "", "Lỗi phân tích cờ"},
		{"bad date", []string{"gross", "35000000", "--date", "2026-13-01"}, 2, "", "--date"},
		{"bad diff date", []string{"diff", "35000000", "--from", "yesterday"}, 2, "", "--from"},
		{"bad lang", []string{"gross", "35000000", "--lang", "jp"}, 2, "", `got "jp"`},

		{"region out of range", []string{"gross", "35000000", "-r", "5", "--date", "2026-01-15"}, 1, "", "Lỗi:"},
		{"date before first policy", []string{"gross", "35000000", "--date", "2020-01-01"}, 1, "", "Lỗi:"},
		{"insurance exceeds gross", []string{"gross", "100000", "--ins-base", "5000000", "--date", "2026-01-15"}, 1, "", "Lỗi:"},
		{"diff earlier period invalid", []string{"diff", "35000000", "--from", "2020-01-01"}, 1, "", "Lỗi kỳ trước"},
		{"negative extra", []string{"gross", "35000000", "--bonus", "-1", "--date", "2026-01-15"}, 1, "", "Lỗi:"},
		{"negative insurance base", []string{"gross", "35000000", "--ins-base", "-1", "--date", "2026-01-15"}, 1, "", "non-negative"},
		{"extras exceed gross", []string{"gross", "10000000", "--bonus", "20000000", "--date", "2026-01-15"}, 1, "", "extras exceed gross"},
		{"diff later period invalid en", []string{"diff", "35000000", "--to", "2020-01-01", "--lang", "en"}, 1, "", "Error in later period"},

		{"flat ok", []string{"flat", "8000000", "--date", "2026-08-01"}, 0, "THỰC NHẬN: 7.200.000 đ", ""},
		{"flat below threshold", []string{"flat", "4000000", "--date", "2026-08-01", "--lang", "en"}, 0, "Below the threshold", ""},
		{"foreigner net", []string{"net", "50754400", "--foreigner", "--date", "2026-08-10"}, 0, "LƯƠNG GROSS: 60.000.000 đ", ""},
		{"non-resident net", []string{"net", "43193000", "--foreigner", "--non-resident", "--date", "2026-08-10"}, 0, "LƯƠNG GROSS: 60.000.000 đ", ""},
		{"flat rejects non-resident", []string{"flat", "8000000", "--non-resident"}, 2, "", "lệnh flat không dùng cờ -non-resident"},
		{"flat rejects foreigner", []string{"flat", "8000000", "--foreigner"}, 2, "", "lệnh flat không dùng cờ -foreigner"},
		{"flat rejects unused flag", []string{"flat", "8000000", "--bonus", "1000000"}, 2, "", "lệnh flat không dùng cờ -bonus"},
		{"flat rejects unused flag en", []string{"flat", "8000000", "-d", "1", "--lang", "en"}, 2, "", "the flat command does not use -d"},
		{"extra argument hides a flag", []string{"gross", "35000000", "x", "--bonus", "5000000"}, 2, "", "tham số thừa: x --bonus 5000000"},
		{"flat extra argument hides a flag", []string{"flat", "8000000", "extra", "--bonus", "1000000", "--lang", "en"}, 2, "", "unexpected arguments: extra --bonus 1000000"},
		{"annual extra argument", []string{"annual", "testdata/annual_job_change.json", "other.json"}, 2, "", "tham số thừa: other.json"},
		{"flat negative", []string{"flat", "--", "-1"}, 2, "", "số tiền không hợp lệ"},
		{"flat date before first policy", []string{"flat", "3000000", "--date", "2020-01-01"}, 1, "", "Lỗi:"},
		{"annual ok", []string{"annual", "testdata/annual_job_change.json"}, 0, "ĐƯỢC HOÀN: 2.831.250 đ", ""},
		{"annual file after flags", []string{"annual", "--lang", "en", "testdata/annual_job_change.json"}, 0, "REFUNDABLE: 2,831,250 VND", ""},
		{"annual missing file", []string{"annual"}, 2, "", "thiếu file danh sách chi trả"},
		{"annual unreadable file", []string{"annual", "testdata/does-not-exist.json"}, 2, "", "Lỗi đọc file testdata/does-not-exist.json"},
		{"annual unknown flag", []string{"annual", "testdata/annual_job_change.json", "-x"}, 2, "", "flag provided but not defined: -x"},
		{"annual help", []string{"annual", "-h"}, 0, "vnsalary annual <file.json>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tt.wantCode, stdout, stderr)
			}
			checkStream(t, "stdout", stdout, tt.wantStdout)
			checkStream(t, "stderr", stderr, tt.wantStderr)
		})
	}
}

func checkStream(t *testing.T, name, got, want string) {
	t.Helper()
	if want == "" {
		if got != "" {
			t.Errorf("%s should be empty, got:\n%s", name, got)
		}
		return
	}
	if !strings.Contains(got, want) {
		t.Errorf("%s does not contain %q:\n%s", name, want, got)
	}
}

func TestRun_AmountPositionAndFlagSpellings(t *testing.T) {
	variants := [][]string{
		{"gross", "35000000", "-d", "1", "-r", "1", "--date", "2026-01-15"},
		{"gross", "-d", "1", "35000000", "--date", "2026-01-15"},
		{"gross", "--dependents", "1", "--region", "1", "--date=2026-01-15", "35000000"},
		{"gross", "-dependents=1", "-date", "2026-01-15", "35000000"},
	}
	_, want, _ := runCLI(variants[0]...)
	for _, args := range variants[1:] {
		code, got, stderr := runCLI(args...)
		if code != 0 || got != want {
			t.Errorf("%v: exit %d, stderr %q; output differs from %v", args, code, stderr, variants[0])
		}
	}
}

// TestRun_JSONMatchesLibrary checks that --json output is the library result,
// unchanged by the CLI and independent of --lang.
func TestRun_JSONMatchesLibrary(t *testing.T) {
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	opts := vnsalary.Options{Dependents: 1, Region: vnsalary.Region1, Date: date}

	wantGross, err := vnsalary.GrossToNet(35_000_000, opts)
	if err != nil {
		t.Fatal(err)
	}
	wantNet, err := vnsalary.NetToGross(30_843_750, opts)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		args []string
		want vnsalary.Result
	}{
		{[]string{"gross", "35000000", "-d", "1", "--date", "2026-01-15", "--json"}, wantGross},
		{[]string{"gross", "35000000", "-d", "1", "--date", "2026-01-15", "--json", "--lang", "en"}, wantGross},
		{[]string{"net", "30843750", "-d", "1", "--date", "2026-01-15", "--json"}, wantNet},
	}
	for _, tt := range tests {
		code, stdout, stderr := runCLI(tt.args...)
		if code != 0 {
			t.Fatalf("%v: exit %d: %s", tt.args, code, stderr)
		}
		var got vnsalary.Result
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("%v: invalid JSON: %v\n%s", tt.args, err, stdout)
		}
		if got.Gross != tt.want.Gross || got.Net != tt.want.Net || got.Tax != tt.want.Tax ||
			got.Insurance != tt.want.Insurance || got.Employer != tt.want.Employer {
			t.Errorf("%v: got %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestRun_DiffJSON(t *testing.T) {
	code, stdout, stderr := runCLI("diff", "60000000", "--from", "2026-06-30", "--to", "2026-07-01", "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	var got struct {
		Gross      int64           `json:"gross"`
		PeriodFrom vnsalary.Result `json:"period_from"`
		PeriodTo   vnsalary.Result `json:"period_to"`
		DiffNet    int64           `json:"diff_net"`
		DiffTax    int64           `json:"diff_tax"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	// Values pinned by TestBaseSalaryChange_2026_07 in the library tests.
	if got.Gross != 60_000_000 || got.PeriodFrom.Net != 50_563_200 || got.PeriodTo.Net != 50_274_400 {
		t.Errorf("unexpected diff result: %+v", got)
	}
	if got.DiffNet != int64(got.PeriodTo.Net-got.PeriodFrom.Net) || got.DiffTax != int64(got.PeriodTo.Tax-got.PeriodFrom.Tax) {
		t.Errorf("diff fields inconsistent: %+v", got)
	}
}

// TestRun_Golden pins the human-readable output. After an intended change, run
//
//	go test ./cmd/vnsalary -run TestRun_Golden -update
//
// and review the diff of testdata/*.golden before committing.
func TestRun_Golden(t *testing.T) {
	tests := []struct {
		file string
		args []string
	}{
		{"gross_vi.golden", []string{"gross", "35000000", "-d", "1", "--date", "2026-01-15"}},
		{"gross_en.golden", []string{"gross", "35000000", "-d", "1", "--date", "2026-01-15", "--lang", "en"}},
		{"net_old_policy_vi.golden", []string{"net", "40000000", "-r", "2", "--date", "2025-03-10"}},
		{"diff_vi.golden", []string{"diff", "35000000", "-d", "1"}},
		{"diff_en.golden", []string{"diff", "60000000", "--from", "2026-06-30", "--to", "2026-07-01", "--lang", "en"}},
		{"extras_vi.golden", []string{"gross", "55000000", "-d", "1", "--overtime", "5000000", "--meal", "1500000", "--other-exempt", "800000", "--bonus", "10000000", "--date", "2026-08-10"}},
		{"extras_en.golden", []string{"gross", "50000000", "--bonus", "20000000", "--date", "2026-08-10", "--lang", "en"}},
		{"foreigner_vi.golden", []string{"gross", "60000000", "--foreigner", "--date", "2026-08-10"}},
		{"non_resident_en.golden", []string{"gross", "60000000", "--foreigner", "--non-resident", "-d", "1", "--date", "2026-08-10", "--lang", "en"}},
		{"flat_vi.golden", []string{"flat", "12345675", "--date", "2026-08-15"}},
		{"flat_en.golden", []string{"flat", "3000000", "--date", "2026-06-30", "--lang", "en"}},
		{"annual_vi.golden", []string{"annual", "testdata/annual_job_change.json"}},
		{"annual_en.golden", []string{"annual", "testdata/annual_deductions.json", "--lang", "en"}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			code, got, stderr := runCLI(tt.args...)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			path := filepath.Join("testdata", tt.file)
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run with -update to create it)", err)
			}
			if got != string(want) {
				t.Errorf("output differs from %s (run with -update if intended)\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
			}
		})
	}
}

func TestRun_Warnings(t *testing.T) {
	tests := []struct {
		args []string
		want string // "" means no warning line
	}{
		{[]string{"gross", "3000000", "--date", "2026-03-15"}, "[!] Lương đóng bảo hiểm thấp hơn lương tối thiểu vùng"},
		{[]string{"gross", "3000000", "--date", "2026-03-15", "--lang", "en"}, "[!] Insurance salary is below the regional minimum wage"},
		{[]string{"gross", "30000000", "--ins-base", "4000000", "--date", "2026-03-15"}, "[!] Lương đóng bảo hiểm thấp hơn"},
		{[]string{"gross", "35000000", "--date", "2026-03-15"}, ""},
	}
	for _, tt := range tests {
		code, stdout, stderr := runCLI(tt.args...)
		if code != 0 {
			t.Fatalf("%v: exit %d: %s", tt.args, code, stderr)
		}
		if tt.want == "" {
			if strings.Contains(stdout, "[!]") {
				t.Errorf("%v: unexpected warning:\n%s", tt.args, stdout)
			}
			continue
		}
		if !strings.Contains(stdout, tt.want) {
			t.Errorf("%v: missing %q:\n%s", tt.args, tt.want, stdout)
		}
	}

	// JSON carries the code, independent of --lang.
	_, stdout, _ := runCLI("gross", "3000000", "--date", "2026-03-15", "--json", "--lang", "en")
	if !strings.Contains(stdout, `"code": "insurance_salary_below_regional_minimum"`) {
		t.Errorf("JSON missing warning code:\n%s", stdout)
	}
}

func TestRun_Version(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"version", "--lang", "en"}} {
		code, stdout, stderr := runCLI(args...)
		if code != 0 || stderr != "" || !strings.HasPrefix(stdout, "vnsalary ") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", args, code, stdout, stderr)
		}
	}
	old := version
	version = "v9.9.9"
	defer func() { version = old }()
	if _, stdout, _ := runCLI("version"); stdout != "vnsalary v9.9.9\n" {
		t.Errorf("ldflags version not used: %q", stdout)
	}
}

func TestFormatVND(t *testing.T) {
	tests := []struct {
		v      vnsalary.VND
		lang   string
		expect string
	}{
		{0, "vi", "0 đ"},
		{999, "vi", "999 đ"},
		{1_000, "vi", "1.000 đ"},
		{23_456_789, "vi", "23.456.789 đ"},
		{23_456_789, "en", "23,456,789 VND"},
		{-288_800, "vi", "-288.800 đ"},
		{-1_000, "en", "-1,000 VND"},
		{100_000_000_000_000, "en", "100,000,000,000,000 VND"},
	}
	for _, tt := range tests {
		if got := formatVND(tt.v, tt.lang); got != tt.expect {
			t.Errorf("formatVND(%d, %q) = %q, want %q", tt.v, tt.lang, got, tt.expect)
		}
	}
}

func TestFormatSigned(t *testing.T) {
	if got := formatSigned(72_200, "vi"); got != "+72.200 đ" {
		t.Errorf("positive: got %q", got)
	}
	if got := formatSigned(-72_200, "en"); got != "-72,200 VND" {
		t.Errorf("negative: got %q", got)
	}
	if got := formatSigned(0, "vi"); got != "0 đ" {
		t.Errorf("zero: got %q", got)
	}
}

func TestDetectLang(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, "vi"},
		{[]string{"gross", "1"}, "vi"},
		{[]string{"gross", "1", "--lang", "en"}, "en"},
		{[]string{"gross", "-lang", "en", "1"}, "en"},
		{[]string{"--lang=en"}, "en"},
		{[]string{"-lang=vi"}, "vi"},
		{[]string{"gross", "--lang"}, "vi"}, // missing value: fall back, flag parsing reports it
		{[]string{"--lang", "jp"}, "jp"},    // validated by run
	}
	for _, tt := range tests {
		if got := detectLang(tt.args); got != tt.want {
			t.Errorf("detectLang(%q) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

// writeTemp writes content to a JSON file in a test temp directory.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "payments.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun_AnnualInputErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCode int
		want     string
	}{
		{"misspelled field", `{"paymnts": []}`, 2, `unknown field "paymnts"`},
		{"not json", `payments`, 2, "Lỗi dữ liệu"},
		{"two json values", `{"payments": []} {"payments": [{"date": "2026-01-01", "taxable_income": 1}]}`, 2, "data after the JSON object"},
		{"trailing delimiter", `{"payments": []} }`, 2, "data after the JSON object"},
		{"negative ins_base", `{"payments": [{"date": "2026-02-01", "gross": 30000000, "ins_base": -1}]}`, 1, "non-negative"},
		{"bad date", `{"payments": [{"date": "2026-02-30", "gross": 1}]}`, 2, "Lỗi ở lần chi trả 1 (2026-02-30)"},
		{"gross and payslip", `{"payments": [{"date": "2026-02-01", "gross": 30000000, "tax_withheld": 1}]}`, 2, "not both"},
		{"payslip with a gross option", `{"payments": [{"date": "2026-12-20", "taxable_income": 30000000, "bonus": 20000000}]}`, 2, "not both"},
		{"no amount", `{"payments": [{"date": "2026-02-01", "insurance": 1}]}`, 2, "give gross or taxable_income"},
		{"gross option without gross", `{"payments": [{"date": "2026-02-01", "bonus": 1}]}`, 2, "give gross or taxable_income"},
		{"gross error", `{"payments": [{"date": "2026-02-01", "gross": 30000000, "region": 9}]}`, 1, "lần chi trả 1"},
		{"payment outside year", `{"year": 2026, "payments": [{"date": "2025-12-31", "taxable_income": 1}]}`, 1, "outside the tax year"},
		{"year before 2025", `{"payments": [{"date": "2024-12-31", "taxable_income": 1}]}`, 1, "2025 or later"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI("annual", writeTemp(t, tt.input))
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d\nstderr:\n%s", code, tt.wantCode, stderr)
			}
			checkStream(t, "stdout", stdout, "")
			checkStream(t, "stderr", stderr, tt.want)
		})
	}
}

// Payments given as gross are computed with GrossToNet using the file's
// dependents and region, overridable per payment; dependent_months defaults
// to dependents × 12.
func TestRun_AnnualGrossDefaultsAndOverrides(t *testing.T) {
	path := writeTemp(t, `{
		"dependents": 1,
		"region": 2,
		"payments": [
			{"date": "2026-03-10", "gross": 30000000},
			{"date": "2026-04-10", "gross": 30000000, "dependents": 0, "region": 1, "bonus": 5000000, "ins_base": 20000000}
		]
	}`)
	code, stdout, stderr := runCLI("annual", path, "--json")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	var got vnsalary.AnnualResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}

	var payments []vnsalary.Payment
	for _, c := range []struct {
		date        time.Time
		dep, region int
		bonus       vnsalary.VND
		insBase     *vnsalary.VND
	}{
		{time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), 1, 2, 0, nil},
		{time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), 0, 1, 5_000_000, ptr(vnsalary.VND(20_000_000))},
	} {
		r, err := vnsalary.GrossToNet(30_000_000, vnsalary.Options{
			Dependents: c.dep, Region: vnsalary.Region(c.region), Date: c.date,
			InsuranceBase: c.insBase, Extras: vnsalary.Extras{Bonus: c.bonus},
		})
		if err != nil {
			t.Fatal(err)
		}
		payments = append(payments, r.Payment(c.date))
	}
	want, err := vnsalary.Annual(2026, payments, vnsalary.AnnualOptions{DependentMonths: 12})
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(got)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("CLI result differs from the library:\n got  %s\n want %s", gotJSON, wantJSON)
	}
}

func ptr[T any](v T) *T { return &v }
