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
		{"diff later period invalid en", []string{"diff", "35000000", "--to", "2020-01-01", "--lang", "en"}, 1, "", "Error in later period"},
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
