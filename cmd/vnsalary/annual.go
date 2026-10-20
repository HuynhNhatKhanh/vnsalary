package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/HuynhNhatKhanh/vnsalary"
)

// annualInput is the JSON file read by "vnsalary annual".
type annualInput struct {
	Year            int            `json:"year"`             // default: the year of the first payment
	Region          int            `json:"region"`           // default 1; used by payments given as gross
	Dependents      int            `json:"dependents"`       // used by payments given as gross (monthly withholding)
	DependentMonths *int           `json:"dependent_months"` // default: dependents × 12
	Charity         int64          `json:"charity"`
	Healthcare      int64          `json:"healthcare"`
	Education       int64          `json:"education"`
	Payments        []annualRecord `json:"payments"`
}

// annualRecord is one payment: either gross (withholding is computed with
// GrossToNet) or the amounts printed on a payslip.
type annualRecord struct {
	Date string `json:"date"`

	Gross       *int64 `json:"gross"`
	InsBase     int64  `json:"ins_base"`
	Overtime    int64  `json:"overtime"`
	MealCash    int64  `json:"meal"`
	OtherExempt int64  `json:"other_exempt"`
	Bonus       int64  `json:"bonus"`
	Dependents  *int   `json:"dependents"` // overrides the file's dependents for this payment
	Region      *int   `json:"region"`     // overrides the file's region for this payment

	TaxableIncome *int64 `json:"taxable_income"`
	Insurance     int64  `json:"insurance"`
	TaxWithheld   int64  `json:"tax_withheld"`
}

var (
	errGrossAndPayslip = errors.New("give either gross (with its options) or the payslip amounts taxable_income, insurance, tax_withheld, not both")
	errNoAmount        = errors.New("give gross or taxable_income")
	errTrailingData    = errors.New("data after the JSON object")
)

// runAnnual executes "vnsalary annual <file.json|->".
func runAnnual(args []string, stdout, stderr io.Writer, lang string) int {
	m := msgs[lang]
	flags := flag.NewFlagSet("annual", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	asJSON := flags.Bool("json", false, "")
	flags.String("lang", "vi", "") // already read by detectLang

	// Allow the file anywhere among the flags, as for the amount of other commands.
	var path string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-lang" || a == "--lang":
			flagArgs = append(flagArgs, a)
			if i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		case a != "-" && len(a) > 0 && a[0] == '-':
			flagArgs = append(flagArgs, a)
		case path == "":
			path = a
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
	if flags.NArg() > 0 { // may hide flags after it, see run
		return errorf(stderr, 2, m.errExtraArgs, strings.Join(flags.Args(), " "))
	}
	if path == "" {
		return errorf(stderr, 2, m.errNoFile)
	}

	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return errorf(stderr, 2, m.errReadFile, path, err)
	}
	var in annualInput
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // a misspelled field must not be silently ignored
	if err := dec.Decode(&in); err != nil {
		return errorf(stderr, 2, m.errBadInput, path, err)
	}
	// Anything after the object, such as a second object with more payments,
	// would otherwise be ignored.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errorf(stderr, 2, m.errBadInput, path, errTrailingData)
	}

	payments := make([]vnsalary.Payment, 0, len(in.Payments))
	for i, rec := range in.Payments {
		date, err := rec.check()
		if err != nil {
			return errorf(stderr, 2, m.errPayment, i+1, rec.Date, err)
		}
		p, err := rec.payment(in, date)
		if err != nil {
			return errorf(stderr, 1, m.errPayment, i+1, rec.Date, err)
		}
		payments = append(payments, p)
	}

	year := in.Year
	if year == 0 && len(payments) > 0 {
		year = payments[0].Date.Year()
	}
	dependentMonths := in.Dependents * 12
	if in.DependentMonths != nil {
		dependentMonths = *in.DependentMonths
	}
	res, err := vnsalary.Annual(year, payments, vnsalary.AnnualOptions{
		DependentMonths: dependentMonths,
		Charity:         vnsalary.VND(in.Charity),
		Healthcare:      vnsalary.VND(in.Healthcare),
		Education:       vnsalary.VND(in.Education),
	})
	if err != nil {
		return errorf(stderr, 1, m.errCalc, err)
	}
	if *asJSON {
		data, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return 0
	}
	printAnnual(stdout, res, len(payments), dependentMonths, lang)
	return 0
}

// check parses the date and rejects a record that mixes the gross and the
// payslip form, or gives neither. Its errors are input errors (exit code 2).
func (r annualRecord) check() (time.Time, error) {
	date, err := time.Parse("2006-01-02", r.Date)
	if err != nil {
		return time.Time{}, err
	}
	grossOptions := r.InsBase != 0 || r.Overtime != 0 || r.MealCash != 0 || r.OtherExempt != 0 ||
		r.Bonus != 0 || r.Dependents != nil || r.Region != nil
	payslip := r.TaxableIncome != nil || r.Insurance != 0 || r.TaxWithheld != 0
	switch {
	case (r.Gross != nil || grossOptions) && payslip:
		return time.Time{}, errGrossAndPayslip
	case r.Gross == nil && r.TaxableIncome == nil:
		return time.Time{}, errNoAmount
	}
	return date, nil
}

// payment converts a checked record into a vnsalary.Payment, computing the
// withholding with GrossToNet when the record gives gross.
func (r annualRecord) payment(in annualInput, date time.Time) (vnsalary.Payment, error) {
	if r.Gross == nil {
		return vnsalary.Payment{
			Date:          date,
			TaxableIncome: vnsalary.VND(*r.TaxableIncome),
			Insurance:     vnsalary.VND(r.Insurance),
			TaxWithheld:   vnsalary.VND(r.TaxWithheld),
		}, nil
	}
	region, dependents := in.Region, in.Dependents
	if region == 0 {
		region = 1
	}
	if r.Region != nil {
		region = *r.Region
	}
	if r.Dependents != nil {
		dependents = *r.Dependents
	}
	opts := vnsalary.Options{
		Dependents: dependents,
		Region:     vnsalary.Region(region),
		Date:       date,
		Extras: vnsalary.Extras{
			Overtime:    vnsalary.VND(r.Overtime),
			MealCash:    vnsalary.VND(r.MealCash),
			OtherExempt: vnsalary.VND(r.OtherExempt),
			Bonus:       vnsalary.VND(r.Bonus),
		},
	}
	if r.InsBase != 0 { // a negative value is rejected by GrossToNet, not ignored
		v := vnsalary.VND(r.InsBase)
		opts.InsuranceBase = &v
	}
	res, err := vnsalary.GrossToNet(vnsalary.VND(*r.Gross), opts)
	if err != nil {
		return vnsalary.Payment{}, err
	}
	return res.Payment(date), nil
}

func printAnnual(w io.Writer, r vnsalary.AnnualResult, payments, dependentMonths int, lang string) {
	d := r.Deductions
	if lang == "en" {
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " ANNUAL PIT FINALIZATION %d (%d payments)\n", r.Year, payments)
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " Taxable income:          %s\n", formatVND(r.TaxableIncome, lang))
		fmt.Fprintf(w, " Compulsory insurance:    %s\n", formatVND(r.Insurance, lang))
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " [Deductions: %s]\n", formatVND(d.Total, lang))
		fmt.Fprintf(w, "  • Personal (12 months):  %s\n", formatVND(d.Personal, lang))
		fmt.Fprintf(w, "  • Dependents (%d months): %s\n", dependentMonths, formatVND(d.Dependents, lang))
		printOptional(w, "  • Charity:               %s\n", d.Charity, lang)
		printOptional(w, "  • Medical:               %s\n", d.Healthcare, lang)
		printOptional(w, "  • Tuition:               %s\n", d.Education, lang)
		fmt.Fprintf(w, "  • Assessable income:     %s\n", formatVND(r.AssessableIncome, lang))
		fmt.Fprintln(w, "-----------------------------------------------------------------")
		fmt.Fprintf(w, " [PIT for the year: %s]\n", formatVND(r.Tax, lang))
		for _, b := range r.Brackets {
			fmt.Fprintf(w, "  • Bracket %d (%d%% on %s): %s\n",
				b.Bracket, b.RatePercent, formatVND(b.TaxablePart, lang), formatVND(b.Tax, lang))
		}
		fmt.Fprintf(w, " Withheld during the year: %s\n", formatVND(r.Withheld, lang))
		fmt.Fprintln(w, "=================================================================")
		switch {
		case r.Balance > 0:
			fmt.Fprintf(w, " TO PAY: %s\n", formatVND(r.Balance, lang))
		case r.Balance < 0:
			fmt.Fprintf(w, " REFUNDABLE: %s\n", formatVND(-r.Balance, lang))
		default:
			fmt.Fprintln(w, " NOTHING TO PAY OR REFUND")
		}
		fmt.Fprintln(w, "=================================================================")
		fmt.Fprintf(w, " Applied Policy: %s\n", r.Policy.Description)
		printWarnings(w, r.Warnings, lang)
		return
	}

	// Default Vietnamese output
	fmt.Fprintln(w, "=================================================================")
	fmt.Fprintf(w, " QUYẾT TOÁN THUẾ TNCN NĂM %d (%d lần chi trả)\n", r.Year, payments)
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " Tổng thu nhập chịu thuế: %s\n", formatVND(r.TaxableIncome, lang))
	fmt.Fprintf(w, " Bảo hiểm bắt buộc:       %s\n", formatVND(r.Insurance, lang))
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " [Giảm trừ: %s]\n", formatVND(d.Total, lang))
	fmt.Fprintf(w, "  • Bản thân (12 tháng):  %s\n", formatVND(d.Personal, lang))
	fmt.Fprintf(w, "  • Người phụ thuộc (%d tháng): %s\n", dependentMonths, formatVND(d.Dependents, lang))
	printOptional(w, "  • Từ thiện, nhân đạo:   %s\n", d.Charity, lang)
	printOptional(w, "  • Y tế:                 %s\n", d.Healthcare, lang)
	printOptional(w, "  • Giáo dục - đào tạo:   %s\n", d.Education, lang)
	fmt.Fprintf(w, "  • Thu nhập tính thuế:   %s\n", formatVND(r.AssessableIncome, lang))
	fmt.Fprintln(w, "-----------------------------------------------------------------")
	fmt.Fprintf(w, " [Thuế TNCN cả năm: %s]\n", formatVND(r.Tax, lang))
	for _, b := range r.Brackets {
		fmt.Fprintf(w, "  • Bậc %d (%d%% trên %s): %s\n",
			b.Bracket, b.RatePercent, formatVND(b.TaxablePart, lang), formatVND(b.Tax, lang))
	}
	fmt.Fprintf(w, " Đã khấu trừ trong năm:   %s\n", formatVND(r.Withheld, lang))
	fmt.Fprintln(w, "=================================================================")
	switch {
	case r.Balance > 0:
		fmt.Fprintf(w, " CÒN PHẢI NỘP: %s\n", formatVND(r.Balance, lang))
	case r.Balance < 0:
		fmt.Fprintf(w, " ĐƯỢC HOÀN: %s\n", formatVND(-r.Balance, lang))
	default:
		fmt.Fprintln(w, " KHÔNG CÒN PHẢI NỘP, KHÔNG ĐƯỢC HOÀN")
	}
	fmt.Fprintln(w, "=================================================================")
	desc := r.Policy.DescriptionVI
	if desc == "" {
		desc = r.Policy.Description
	}
	fmt.Fprintf(w, " Áp dụng: %s\n", desc)
	printWarnings(w, r.Warnings, lang)
}

// printOptional prints a deduction line only when the amount is claimed.
func printOptional(w io.Writer, format string, v vnsalary.VND, lang string) {
	if v > 0 {
		fmt.Fprintf(w, format, formatVND(v, lang))
	}
}
