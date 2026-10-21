// Command vnsalary converts Vietnamese gross and net salaries from the command
// line, using the github.com/HuynhNhatKhanh/vnsalary library.
//
// Usage:
//
//	vnsalary gross <amount> [flags]   gross to net
//	vnsalary net <amount> [flags]     net to gross
//	vnsalary diff <amount> [flags]    compare net between two payment dates
//	vnsalary flat <amount> [flags]    10% withholding (no labour contract or under 3 months)
//	vnsalary annual <file.json>       annual PIT finalization of a tax year
//	vnsalary version                  print the version
//
// Flags: -d/--dependents, -r/--region (1..4), --date YYYY-MM-DD (payment date,
// default today), --from/--to (for diff), --ins-base, --overtime, --meal,
// --other-exempt, --bonus, --foreigner, --json, --lang vi|en.
//
// Exit codes: 0 success, 1 calculation error, 2 usage error.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/HuynhNhatKhanh/vnsalary"
)

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	return time.Parse("2006-01-02", s)
}

// version is set by release builds with -ldflags "-X main.version=vX.Y.Z".
// Builds from `go install …@vX.Y.Z` leave it empty and use the module version
// recorded in the binary instead.
var version string

func versionString() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

// Default periods compared by "diff" when --from/--to are omitted:
// the last 7-bracket regime vs. the first 5-bracket regime.
const (
	defaultDiffFrom = "2024-07-01"
	defaultDiffTo   = "2026-01-01"
)

// messages holds every user-facing CLI string for one display language.
type messages struct {
	usage         string
	errFlags      string
	errNoAmount   string
	errBadAmount  string
	errBadDate    string
	errBadLang    string
	errCalc       string
	errFromPeriod string
	errToPeriod   string
	errBadCommand string
	errNoFile     string
	errReadFile   string
	errBadInput   string
	errPayment    string
	errFlagUnused string
	errExtraArgs  string
}

var msgs = map[string]messages{
	"vi": {
		usage: `vnsalary — Công cụ tính lương Gross ↔ Net Việt Nam

Cách dùng:
  vnsalary gross <số_tiền> [flags]  Tính từ lương Gross sang Net
  vnsalary net <số_tiền> [flags]    Tính từ lương Net sang Gross
  vnsalary diff <số_tiền> [flags]   So sánh lương Net giữa 2 kỳ luật
  vnsalary flat <số_tiền> [flags]   Khấu trừ 10% (không ký HĐLĐ hoặc HĐLĐ dưới 3 tháng)
  vnsalary annual <file.json>       Quyết toán thuế TNCN năm từ danh sách chi trả ('-' = stdin)
  vnsalary version                  In phiên bản

Flags:
  -d, --dependents int      Số người phụ thuộc (mặc định: 0)
  -r, --region int          Vùng lương tối thiểu 1..4 (mặc định: 1)
      --date string         Ngày chi trả lương YYYY-MM-DD (mặc định: hôm nay)
      --from string         Kỳ trước cho lệnh diff (mặc định: ` + defaultDiffFrom + `)
      --to string           Kỳ sau cho lệnh diff (mặc định: ` + defaultDiffTo + `)
      --ins-base int        Mức lương đóng bảo hiểm cố định (tùy chọn)
      --overtime int        Tiền làm thêm giờ, làm đêm nằm trong số tiền (miễn thuế từ 2026)
      --meal int            Tiền ăn giữa ca bằng tiền nằm trong số tiền (miễn thuế tới mức trần)
      --other-exempt int    Phụ cấp miễn thuế khác theo quy chế công ty (điện thoại, trang phục…)
      --bonus int           Tiền thưởng trong tháng nằm trong số tiền (chịu thuế, không đóng BH)
      --foreigner           Người lao động nước ngoài: không đóng BHTN
      --lang string         Ngôn ngữ hiển thị: vi, en (mặc định: vi)
      --json                Xuất kết quả dạng JSON`,
		errFlags:      "Lỗi phân tích cờ: %v",
		errNoAmount:   "Lỗi: thiếu số tiền. Ví dụ: vnsalary %s 35000000",
		errBadAmount:  "Lỗi: số tiền không hợp lệ (%s)",
		errBadDate:    "Lỗi định dạng ngày %s: %v",
		errBadLang:    "Lỗi: --lang chỉ nhận vi hoặc en (nhận được %q)",
		errCalc:       "Lỗi: %v",
		errFromPeriod: "Lỗi kỳ trước: %v",
		errToPeriod:   "Lỗi kỳ sau: %v",
		errBadCommand: "Lệnh không hợp lệ: %s",
		errNoFile:     "Lỗi: thiếu file danh sách chi trả. Ví dụ: vnsalary annual payments.json",
		errReadFile:   "Lỗi đọc file %s: %v",
		errBadInput:   "Lỗi dữ liệu trong %s: %v",
		errPayment:    "Lỗi ở lần chi trả %d (%s): %v",
		errFlagUnused: "Lỗi: lệnh %s không dùng cờ -%s",
		errExtraArgs:  "Lỗi: tham số thừa: %s",
	},
	"en": {
		usage: `vnsalary — Vietnam Gross ↔ Net salary calculator

Usage:
  vnsalary gross <amount> [flags]   Convert gross salary to net
  vnsalary net <amount> [flags]     Convert net salary to gross
  vnsalary diff <amount> [flags]    Compare net salary between two policy periods
  vnsalary flat <amount> [flags]    10% withholding (no labour contract, or one under 3 months)
  vnsalary annual <file.json>       Annual PIT finalization from a list of payments ('-' = stdin)
  vnsalary version                  Print the version

Flags:
  -d, --dependents int      Number of dependents (default: 0)
  -r, --region int          Minimum wage region 1..4 (default: 1)
      --date string         Salary payment date YYYY-MM-DD (default: today)
      --from string         Earlier period for diff (default: ` + defaultDiffFrom + `)
      --to string           Later period for diff (default: ` + defaultDiffTo + `)
      --ins-base int        Fixed insurance contribution salary (optional)
      --overtime int        Overtime and night-work pay included in the amount (tax-exempt from 2026)
      --meal int            Cash meal allowance included in the amount (tax-exempt up to the cap)
      --other-exempt int    Other allowances exempt under the employer's regulation (phone, uniform...)
      --bonus int           Bonus paid this month, included in the amount (taxed, no insurance)
      --foreigner           Foreign national: no unemployment insurance
      --lang string         Display language: vi, en (default: vi)
      --json                Print the result as JSON`,
		errFlags:      "Error parsing flags: %v",
		errNoAmount:   "Error: missing amount. Example: vnsalary %s 35000000",
		errBadAmount:  "Error: invalid amount (%s)",
		errBadDate:    "Error: invalid date %s: %v",
		errBadLang:    "Error: --lang accepts vi or en (got %q)",
		errCalc:       "Error: %v",
		errFromPeriod: "Error in earlier period: %v",
		errToPeriod:   "Error in later period: %v",
		errBadCommand: "Unknown command: %s",
		errNoFile:     "Error: missing payments file. Example: vnsalary annual payments.json",
		errReadFile:   "Error reading %s: %v",
		errBadInput:   "Invalid data in %s: %v",
		errPayment:    "Error in payment %d (%s): %v",
		errFlagUnused: "Error: the %s command does not use -%s",
		errExtraArgs:  "Error: unexpected arguments: %s",
	},
}

// detectLang reads --lang/-lang (also --lang=xx) before flag parsing, so that
// usage and early errors are printed in the requested language.
func detectLang(args []string) string {
	for i, a := range args {
		switch {
		case (a == "--lang" || a == "-lang") && i+1 < len(args):
			return args[i+1]
		case strings.HasPrefix(a, "--lang="):
			return strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-lang="):
			return strings.TrimPrefix(a, "-lang=")
		}
	}
	return "vi"
}

// errorf prints a formatted message to w and returns code, so that run can
// "return errorf(...)" on every failure path.
func errorf(w io.Writer, code int, format string, args ...any) int {
	fmt.Fprintf(w, format+"\n", args...)
	return code
}

func printUsage(w io.Writer, lang string) {
	fmt.Fprintln(w, msgs[lang].usage)
}

// formatVND formats an amount as "35.000.000 đ" (vi) or "35,000,000 VND" (en).
func formatVND(v vnsalary.VND, lang string) string {
	sep, unit := byte('.'), " đ"
	if lang == "en" {
		sep, unit = ',', " VND"
	}

	val := int64(v)
	neg := val < 0
	if neg {
		val = -val
	}
	s := strconv.FormatInt(val, 10)

	var res []byte
	if neg {
		res = append(res, '-')
	}
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			res = append(res, sep)
		}
		res = append(res, s[i])
	}
	return string(res) + unit
}

// formatSigned is formatVND with an explicit "+" for positive amounts.
func formatSigned(v vnsalary.VND, lang string) string {
	if v > 0 {
		return "+" + formatVND(v, lang)
	}
	return formatVND(v, lang)
}

func printTable(w io.Writer, r vnsalary.Result, lang string) {
	if lang == "en" {
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " GROSS SALARY: %s\n", formatVND(r.Gross, lang))
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " [Mandatory Employee Insurance: %s]\n", formatVND(r.Insurance.Total, lang))
		fmt.Fprintf(w, "  • Social Insurance (BHXH 8%%):       %s\n", formatVND(r.Insurance.Social, lang))
		fmt.Fprintf(w, "  • Health Insurance (BHYT 1.5%%):     %s\n", formatVND(r.Insurance.Health, lang))
		fmt.Fprintf(w, "  • Unemployment (BHTN 1%%):           %s\n", formatVND(r.Insurance.Unemployment, lang))
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		if r.Exempt != nil {
			fmt.Fprintf(w, " [Tax-Exempt Income: %s]\n", formatVND(r.Exempt.Total, lang))
			fmt.Fprintf(w, "  • Overtime:                          %s\n", formatVND(r.Exempt.Overtime, lang))
			fmt.Fprintf(w, "  • Meal allowance:                    %s\n", formatVND(r.Exempt.Meal, lang))
			fmt.Fprintf(w, "  • Other allowances:                  %s\n", formatVND(r.Exempt.Other, lang))
			fmt.Fprintln(w, "-----------------------------------------------------------------")
		}
		fmt.Fprintf(w, " [Tax Deductions: %s]\n", formatVND(r.Deductions.Total, lang))
		fmt.Fprintf(w, "  • Personal:                          %s\n", formatVND(r.Deductions.Personal, lang))
		fmt.Fprintf(w, "  • Dependents:                        %s\n", formatVND(r.Deductions.Dependents, lang))
		fmt.Fprintf(w, "  • Assessable Income:                 %s\n", formatVND(r.AssessableIncome, lang))
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " [Personal Income Tax (PIT): %s]\n", formatVND(r.Tax, lang))
		for _, b := range r.Brackets {
			fmt.Fprintf(w, "  • Bracket %d (%d%% on %s): %s\n",
				b.Bracket, b.RatePercent, formatVND(b.TaxablePart, lang), formatVND(b.Tax, lang))
		}
		if r.BonusTax > 0 {
			fmt.Fprintf(w, "  • Of which caused by the bonus:      %s\n", formatVND(r.BonusTax, lang))
		}
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " NET TAKE-HOME PAY: %s\n", formatVND(r.Net, lang))
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " Employer Cost: %s (Gross + Employer Insurance %s)\n",
			formatVND(r.Employer.TotalCost, lang), formatVND(r.Employer.TotalInsurance, lang))
		fmt.Fprintf(w, " Applied Policy: %s\n", r.Policy.Description)
		printWarnings(w, r.Warnings, lang)
		return
	}

	// Default Vietnamese output
	fmt.Fprintln(w, "=================================================================")
	fmt.Fprintf(w, " LƯƠNG GROSS: %s\n", formatVND(r.Gross, lang))
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " [Bảo hiểm người lao động: %s]\n", formatVND(r.Insurance.Total, lang))
	fmt.Fprintf(w, "  • BHXH (8%%):       %s\n", formatVND(r.Insurance.Social, lang))
	fmt.Fprintf(w, "  • BHYT (1.5%%):     %s\n", formatVND(r.Insurance.Health, lang))
	fmt.Fprintf(w, "  • BHTN (1%%):       %s\n", formatVND(r.Insurance.Unemployment, lang))
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	if r.Exempt != nil {
		fmt.Fprintf(w, " [Thu nhập miễn thuế: %s]\n", formatVND(r.Exempt.Total, lang))
		fmt.Fprintf(w, "  • Làm thêm giờ:    %s\n", formatVND(r.Exempt.Overtime, lang))
		fmt.Fprintf(w, "  • Tiền ăn giữa ca: %s\n", formatVND(r.Exempt.Meal, lang))
		fmt.Fprintf(w, "  • Phụ cấp khác:    %s\n", formatVND(r.Exempt.Other, lang))
		fmt.Fprintln(w, "-----------------------------------------------------------------")
	}
	fmt.Fprintf(w, " [Giảm trừ gia cảnh: %s]\n", formatVND(r.Deductions.Total, lang))
	fmt.Fprintf(w, "  • Bản thân:        %s\n", formatVND(r.Deductions.Personal, lang))
	fmt.Fprintf(w, "  • Người phụ thuộc: %s\n", formatVND(r.Deductions.Dependents, lang))
	fmt.Fprintf(w, "  • Thu nhập tính thuế: %s\n", formatVND(r.AssessableIncome, lang))
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " [Thuế TNCN: %s]\n", formatVND(r.Tax, lang))
	for _, b := range r.Brackets {
		fmt.Fprintf(w, "  • Bậc %d (%d%% trên %s): %s\n",
			b.Bracket, b.RatePercent, formatVND(b.TaxablePart, lang), formatVND(b.Tax, lang))
	}
	if r.BonusTax > 0 {
		fmt.Fprintf(w, "  • Trong đó do khoản thưởng: %s\n", formatVND(r.BonusTax, lang))
	}
	fmt.Fprintln(w, "=================================================================")
	fmt.Fprintf(w, " LƯƠNG NET THỰC NHẬN: %s\n", formatVND(r.Net, lang))
	fmt.Fprintln(w, "=================================================================")
	fmt.Fprintf(w, " Chi phí người sử dụng lao động: %s (Gross + Bảo hiểm NSDLĐ %s)\n",
		formatVND(r.Employer.TotalCost, lang), formatVND(r.Employer.TotalInsurance, lang))
	desc := r.Policy.DescriptionVI
	if desc == "" {
		desc = r.Policy.Description
	}
	fmt.Fprintf(w, " Áp dụng: %s\n", desc)
	printWarnings(w, r.Warnings, lang)
}

// warningText holds the localized text of known warning codes; unknown codes
// fall back to the library's English message.
var warningText = map[string]map[string]string{
	vnsalary.WarnInsuranceSalaryBelowRegionalMinimum: {
		"vi": "Lương đóng bảo hiểm thấp hơn lương tối thiểu vùng (thường do hợp đồng bán thời gian hoặc nhập sai)",
		"en": "Insurance salary is below the regional minimum wage (usually a part-time contract or an input mistake)",
	},
	vnsalary.WarnOvertimeTaxedBeforeExemption: {
		"vi": "Trước năm 2026 chỉ phần lương làm thêm giờ cao hơn giờ thường được miễn thuế; đang tính thuế toàn bộ --overtime, hãy đưa phần được miễn vào --other-exempt",
		"en": "Before 2026 only overtime pay above normal-hours pay is exempt; all of --overtime is taxed, put the exempt part in --other-exempt",
	},
	vnsalary.WarnMealCapUnverified: {
		"vi": "Mức trần tiền ăn giữa ca cho ngày chi trả này chưa được xác nhận bằng văn bản luật",
		"en": "The meal allowance cap for this payment date is not confirmed by a legal text",
	},
	vnsalary.WarnHealthEducationNotDeductible: {
		"vi": "Chi phí y tế, giáo dục chỉ được giảm trừ từ kỳ tính thuế 2026; đã bỏ qua",
		"en": "Medical and tuition expenses are deductible from tax period 2026; ignored",
	},
	vnsalary.WarnInsuranceSalaryIncludesAllowances: {
		"vi": "Lương đóng bảo hiểm đang gồm cả --meal và --other-exempt; dùng --ins-base nếu hợp đồng không tính chúng",
		"en": "Insurance salary includes --meal and --other-exempt; use --ins-base if the contract excludes them",
	},
}

func printWarnings(w io.Writer, warnings []vnsalary.Warning, lang string) {
	for _, warn := range warnings {
		text := warningText[warn.Code][lang]
		if text == "" {
			text = warn.Message
		}
		fmt.Fprintf(w, " [!] %s\n", text)
	}
}

func printFlat(w io.Writer, r vnsalary.FlatResult, lang string) {
	if lang == "en" {
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintln(w, " 10% PIT WITHHOLDING (no labour contract, or one under 3 months)")
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " Payment:            %s\n", formatVND(r.Amount, lang))
		fmt.Fprintf(w, " Threshold:          %s per payment\n", formatVND(r.Threshold, lang))
		fmt.Fprintf(w, " Tax withheld (10%%): %s\n", formatVND(r.Withheld, lang))
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " AMOUNT PAID OUT: %s\n", formatVND(r.Net, lang))
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " Applied Policy: %s\n", r.Policy.Description)
		if !r.Required {
			fmt.Fprintln(w, " [i] Below the threshold: withholding is not required (from 2026-07-01 the payer may withhold 10% at the individual's request)")
		}
		return
	}

	// Default Vietnamese output
	fmt.Fprintln(w, "=================================================================")
	fmt.Fprintln(w, " KHẤU TRỪ THUẾ TNCN 10% (không ký HĐLĐ hoặc HĐLĐ dưới 3 tháng)")
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " Số tiền chi trả:        %s\n", formatVND(r.Amount, lang))
	fmt.Fprintf(w, " Ngưỡng khấu trừ:        %s/lần\n", formatVND(r.Threshold, lang))
	fmt.Fprintf(w, " Thuế khấu trừ (10%%):    %s\n", formatVND(r.Withheld, lang))
	fmt.Fprintln(w, "=================================================================")
	fmt.Fprintf(w, " THỰC NHẬN: %s\n", formatVND(r.Net, lang))
	fmt.Fprintln(w, "=================================================================")
	desc := r.Policy.DescriptionVI
	if desc == "" {
		desc = r.Policy.Description
	}
	fmt.Fprintf(w, " Áp dụng: %s\n", desc)
	if !r.Required {
		fmt.Fprintln(w, " [i] Dưới ngưỡng: không bắt buộc khấu trừ (từ 01/07/2026 vẫn được khấu trừ 10% nếu cá nhân yêu cầu)")
	}
}

func printDiffTable(w io.Writer, amount int64, dFrom, dTo time.Time, resFrom, resTo vnsalary.Result, dependents, region int, lang string) {
	diffNet := resTo.Net - resFrom.Net
	diffTax := resTo.Tax - resFrom.Tax

	if lang == "en" {
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " POLICY COMPARISON FOR GROSS: %s (Dependents: %d, Region: %d)\n",
			formatVND(vnsalary.VND(amount), lang), dependents, region)
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " Previous Period (%s):\n", dFrom.Format("2006-01-02"))
		fmt.Fprintf(w, "  • PIT Tax:  %s\n", formatVND(resFrom.Tax, lang))
		fmt.Fprintf(w, "  • Net Pay:  %s\n", formatVND(resFrom.Net, lang))
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " New Period (%s):\n", dTo.Format("2006-01-02"))
		fmt.Fprintf(w, "  • PIT Tax:  %s\n", formatVND(resTo.Tax, lang))
		fmt.Fprintf(w, "  • Net Pay:  %s\n", formatVND(resTo.Net, lang))
		fmt.Fprintln(w, "=================================================================")
		switch {
		case diffNet > 0:
			fmt.Fprintf(w, " [+] NET INCREASE: %s/month\n", formatSigned(diffNet, lang))
		case diffNet < 0:
			fmt.Fprintf(w, " [-] NET DECREASE: %s/month\n", formatSigned(diffNet, lang))
		default:
			fmt.Fprintln(w, " [=] No change in Net salary between periods.")
		}
		fmt.Fprintf(w, " PIT change: %s/month\n", formatSigned(diffTax, lang))
		fmt.Fprintln(w, "=================================================================")
		return
	}

	// Default Vietnamese diff output
	fmt.Fprintln(w, "=================================================================")
	fmt.Fprintf(w, " SO SÁNH LƯƠNG GROSS: %s (Người phụ thuộc: %d, Vùng %d)\n",
		formatVND(vnsalary.VND(amount), lang), dependents, region)
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " Kỳ trước (%s):\n", dFrom.Format("2006-01-02"))
	fmt.Fprintf(w, "  • Thuế TNCN: %s\n", formatVND(resFrom.Tax, lang))
	fmt.Fprintf(w, "  • Lương Net: %s\n", formatVND(resFrom.Net, lang))
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " Kỳ mới (%s):\n", dTo.Format("2006-01-02"))
	fmt.Fprintf(w, "  • Thuế TNCN: %s\n", formatVND(resTo.Tax, lang))
	fmt.Fprintf(w, "  • Lương Net: %s\n", formatVND(resTo.Net, lang))
	fmt.Fprintln(w, "=================================================================")
	switch {
	case diffNet > 0:
		fmt.Fprintf(w, " [+] LƯƠNG NET TĂNG THÊM: %s/tháng\n", formatSigned(diffNet, lang))
	case diffNet < 0:
		fmt.Fprintf(w, " [-] LƯƠNG NET GIẢM: %s/tháng\n", formatSigned(diffNet, lang))
	default:
		fmt.Fprintln(w, " [=] Không có sự thay đổi về lương Net giữa 2 kỳ.")
	}
	fmt.Fprintf(w, " Thuế TNCN thay đổi: %s/tháng\n", formatSigned(diffTax, lang))
	fmt.Fprintln(w, "=================================================================")
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI with args (without the program name) and returns the
// process exit code: 0 success, 1 calculation error, 2 usage error.
func run(args []string, stdout, stderr io.Writer) int {
	lang := detectLang(args)
	if _, ok := msgs[lang]; !ok {
		return errorf(stderr, 2, msgs["vi"].errBadLang+"\n"+msgs["en"].errBadLang, lang, lang)
	}
	m := msgs[lang]

	if len(args) == 0 {
		printUsage(stderr, lang)
		return 2
	}

	cmd := args[0]
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		printUsage(stdout, lang)
		return 0
	}
	if cmd == "version" || cmd == "--version" {
		fmt.Fprintf(stdout, "vnsalary %s\n", versionString())
		return 0
	}
	if strings.HasPrefix(cmd, "-") { // flags given without a command, e.g. "vnsalary --lang en"
		printUsage(stderr, lang)
		return 2
	}
	switch cmd {
	case "gross", "net", "diff", "flat":
	case "annual":
		return runAnnual(args[1:], stdout, stderr, lang)
	default:
		fmt.Fprintf(stderr, m.errBadCommand+"\n", cmd)
		printUsage(stderr, lang)
		return 2
	}

	// Errors are reported by run itself (localized), not by the flag package.
	flags := flag.NewFlagSet(cmd, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dependents := flags.Int("d", 0, "")
	flags.IntVar(dependents, "dependents", 0, "")
	region := flags.Int("r", 1, "")
	flags.IntVar(region, "region", 1, "")
	dateStr := flags.String("date", "", "")
	fromStr := flags.String("from", defaultDiffFrom, "")
	toStr := flags.String("to", defaultDiffTo, "")
	insBaseFlag := flags.Int64("ins-base", 0, "")
	overtime := flags.Int64("overtime", 0, "")
	meal := flags.Int64("meal", 0, "")
	otherExempt := flags.Int64("other-exempt", 0, "")
	bonus := flags.Int64("bonus", 0, "")
	foreigner := flags.Bool("foreigner", false, "")
	flags.String("lang", "vi", "") // already read by detectLang
	asJSON := flags.Bool("json", false, "")

	// Allow the amount anywhere among the flags: "gross 35000000 -d 1" or "gross -d 1 35000000".
	flagsWithValue := map[string]bool{
		"-d": true, "-dependents": true, "--dependents": true,
		"-r": true, "-region": true, "--region": true,
		"-date": true, "--date": true,
		"-from": true, "--from": true,
		"-to": true, "--to": true,
		"-ins-base": true, "--ins-base": true,
		"-overtime": true, "--overtime": true,
		"-meal": true, "--meal": true,
		"-other-exempt": true, "--other-exempt": true,
		"-bonus": true, "--bonus": true,
		"-lang": true, "--lang": true,
	}
	rawArgs := args[1:]
	var amountStr string
	var flagArgs []string
	for i := 0; i < len(rawArgs); i++ {
		a := rawArgs[i]
		switch {
		case flagsWithValue[a]:
			flagArgs = append(flagArgs, a)
			if i+1 < len(rawArgs) {
				i++
				flagArgs = append(flagArgs, rawArgs[i])
			}
		case strings.HasPrefix(a, "-"):
			flagArgs = append(flagArgs, a)
		case amountStr == "":
			amountStr = a
		default:
			flagArgs = append(flagArgs, a)
		}
	}
	if err := flags.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout, lang)
			return 0
		}
		fmt.Fprintf(stderr, m.errFlags+"\n", err)
		printUsage(stderr, lang)
		return 2
	}
	// flag stops at the first non-flag argument, so anything left over may
	// hide flags after it ("gross 1 x --bonus 2"): reject it.
	rest := flags.Args()
	if amountStr == "" && len(rest) > 0 {
		amountStr, rest = rest[0], rest[1:]
	}
	if len(rest) > 0 {
		return errorf(stderr, 2, m.errExtraArgs, strings.Join(rest, " "))
	}
	if amountStr == "" {
		return errorf(stderr, 2, m.errNoAmount, cmd)
	}
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount < 0 {
		return errorf(stderr, 2, m.errBadAmount, amountStr)
	}

	var insBase *vnsalary.VND
	if *insBaseFlag != 0 { // a negative value is rejected by the library, not ignored
		v := vnsalary.VND(*insBaseFlag)
		insBase = &v
	}
	optsAt := func(d time.Time) vnsalary.Options {
		return vnsalary.Options{
			InsuranceBase: insBase,
			Dependents:    *dependents,
			Region:        vnsalary.Region(*region),
			Date:          d,
			Foreigner:     *foreigner,
			Extras: vnsalary.Extras{
				Overtime:    vnsalary.VND(*overtime),
				MealCash:    vnsalary.VND(*meal),
				OtherExempt: vnsalary.VND(*otherExempt),
				Bonus:       vnsalary.VND(*bonus),
			},
		}
	}
	if cmd == "diff" {
		dFrom, err := parseDate(*fromStr)
		if err != nil {
			return errorf(stderr, 2, m.errBadDate, "--from", err)
		}
		dTo, err := parseDate(*toStr)
		if err != nil {
			return errorf(stderr, 2, m.errBadDate, "--to", err)
		}
		resFrom, err := vnsalary.GrossToNet(vnsalary.VND(amount), optsAt(dFrom))
		if err != nil {
			return errorf(stderr, 1, m.errFromPeriod, err)
		}
		resTo, err := vnsalary.GrossToNet(vnsalary.VND(amount), optsAt(dTo))
		if err != nil {
			return errorf(stderr, 1, m.errToPeriod, err)
		}
		if *asJSON {
			data, _ := json.MarshalIndent(map[string]any{
				"gross":       amount,
				"period_from": resFrom,
				"period_to":   resTo,
				"diff_net":    resTo.Net - resFrom.Net,
				"diff_tax":    resTo.Tax - resFrom.Tax,
			}, "", "  ")
			fmt.Fprintln(stdout, string(data))
			return 0
		}
		printDiffTable(stdout, amount, dFrom, dTo, resFrom, resTo, *dependents, *region, lang)
		return 0
	}

	date, err := parseDate(*dateStr)
	if err != nil {
		return errorf(stderr, 2, m.errBadDate, "--date", err)
	}
	if cmd == "flat" {
		// 10% applies to the whole payment: other flags would be silently ignored.
		var unused string
		flags.Visit(func(f *flag.Flag) {
			if unused == "" && f.Name != "date" && f.Name != "json" && f.Name != "lang" {
				unused = f.Name
			}
		})
		if unused != "" {
			return errorf(stderr, 2, m.errFlagUnused, cmd, unused)
		}
		res, err := vnsalary.FlatWithholding(vnsalary.VND(amount), date)
		if err != nil {
			return errorf(stderr, 1, m.errCalc, err)
		}
		if *asJSON {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(stdout, string(data))
			return 0
		}
		printFlat(stdout, res, lang)
		return 0
	}
	var res vnsalary.Result
	if cmd == "gross" {
		res, err = vnsalary.GrossToNet(vnsalary.VND(amount), optsAt(date))
	} else {
		res, err = vnsalary.NetToGross(vnsalary.VND(amount), optsAt(date))
	}
	if err != nil {
		return errorf(stderr, 1, m.errCalc, err)
	}
	if *asJSON {
		data, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	printTable(stdout, res, lang)
	return 0
}
