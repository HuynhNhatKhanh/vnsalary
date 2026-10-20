# vnsalary

[![CI](https://github.com/HuynhNhatKhanh/vnsalary/actions/workflows/ci.yml/badge.svg)](https://github.com/HuynhNhatKhanh/vnsalary/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/HuynhNhatKhanh/vnsalary.svg)](https://pkg.go.dev/github.com/HuynhNhatKhanh/vnsalary)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Thư viện Golang tính lương Gross ↔ Net và Thuế thu nhập cá nhân (TNCN) Việt Nam.**  
> *Chính xác tới từng 1 đồng. Giải ngược giải tích từng đoạn tuyến tính. Thuần Go Standard Library (Zero external dependencies).*

📖 [Read the English version here](README.md)

---

## 🌟 Tính Năng Nổi Bật

- **Zero External Dependencies:** Thuần Go 100%, chỉ sử dụng Go Standard Library.
- **Giải Ngược Giải Tích Từng Đoạn Tuyến Tính (Piecewise-Linear Inversion):** Tính Net → Gross bằng cách xác định các điểm gãy chính sách (breakpoints) và tính nghiệm xấp xỉ ban đầu bằng giải tích, kết hợp tinh chỉnh cục bộ biên (±50 VND) và fallback Binary Search trên đoạn đơn điệu để bù trừ sai số làm tròn số nguyên (integer discretization). Đảm bảo tính nhất quán toán học hai chiều: `GrossToNet(NetToGross(net)).Net == net`.
- **Số Học Số Nguyên (`int64` VND):** Tuyệt đối không dùng `float64` cho tiền tệ. Thuật toán làm tròn half-up số nguyên độc lập trên từng khoản bảo hiểm và từng bậc thuế lũy tiến theo công thức `(n + d/2) / d`, loại bỏ hoàn toàn sai số dấu phẩy động IEEE-754.
- **Engine Chính Sách Đa Thời Kỳ & Căn Cứ Pháp Lý Minh Bạch:** Tự động tra cứu luật theo ngày chi trả thực tế (**Cash-basis** theo Luật Thuế TNCN số 109/2025/QH15 Điều 8 khoản 3; trước đó Thông tư 111/2013/TT-BTC Điều 8 khoản 2 điểm b):
  - **Mốc 01/07/2024:** Lương cơ sở 2.340.000đ (NĐ 73/2024/NĐ-CP), lương tối thiểu 4 vùng (NĐ 74/2024/NĐ-CP), biểu thuế 7 bậc (Luật Thuế TNCN 2007), giảm trừ gia cảnh 11M / 4.4M (NQ 954/2020/UBTVQH14).
  - **Mốc 01/01/2026:** Biểu thuế lũy tiến **5 bậc mới** (Luật Thuế TNCN số 109/2025/QH15 Điều 9), giảm trừ gia cảnh nâng lên **15,5M / 6,2M** (Nghị quyết 110/2025/UBTVQH15), lương tối thiểu vùng mới (NĐ 293/2025/NĐ-CP).
  - **Mốc 01/07/2026:** Lương cơ sở tăng lên **2.530.000đ** (NĐ 161/2026/NĐ-CP) nên trần BHXH/BHYT tăng từ 46,8M lên **50,6M**. Các tham số khác giữ nguyên.
- **An Toàn Concurrency & Bất Biến (Immutability):** `PolicyAt` trả về deep copy của struct chính sách, bảo vệ dữ liệu cấu hình toàn cục. Bổ sung kiểm tra chặn tràn số (`int64` overflow guard cho mức lương > 10¹⁴ VND) và chặn âm lương khi bảo hiểm vượt quá lương Gross.
- **Bóc Tách Đầy Đủ Chi Phí Doanh Nghiệp (Employer Cost):** Tính toàn bộ bảo hiểm công ty đóng (BHXH 17.5%, BHYT 3%, BHTN 1%) và tổng ngân sách nhân sự thực tế.
- **Quyết Toán Thuế TNCN Năm:** `Annual` quyết toán một năm từ các lần chi trả (nhiều công ty, tháng có thưởng, tháng không có thu nhập, đăng ký người phụ thuộc muộn) và trả về số còn phải nộp hoặc được hoàn; giảm trừ bản thân đủ 12 tháng, trừ chi phí y tế và giáo dục từ 2026 (NĐ 253/2026/NĐ-CP Điều 48–49).
- **Tích Hợp Sẵn Công Cụ CLI:** Hỗ trợ tính dòng lệnh `gross`, `net`, `diff` so sánh 2024 vs 2026, `annual` quyết toán từ file JSON, hỗ trợ đặt cờ trước hoặc sau số tiền, xuất bảng đẹp và cờ `--json` để tích hợp pipeline.

---

## 📦 Cài Đặt

```bash
go get github.com/HuynhNhatKhanh/vnsalary
```

Yêu cầu môi trường: **Go 1.22+**.

---

## 🚀 Hướng Dẫn Nhanh

### 1. Tính Lương Gross sang Net

```go
package main

import (
	"fmt"
	"time"

	"github.com/HuynhNhatKhanh/vnsalary"
)

func main() {
	opts := vnsalary.Options{
		Dependents: 1,                 // 1 người phụ thuộc
		Region:     vnsalary.Region1,  // Vùng 1 (Hà Nội, TP.HCM,...)
		Date:       time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}

	result, err := vnsalary.GrossToNet(35_000_000, opts)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Gross:                %d VND\n", result.Gross)
	fmt.Printf("Lương Net thực nhận:  %d VND\n", result.Net)
	fmt.Printf("Bảo hiểm NLĐ:         %d VND\n", result.Insurance.Total)
	fmt.Printf("Thuế TNCN:            %d VND\n", result.Tax)
	fmt.Printf("Tổng chi phí công ty: %d VND\n", result.Employer.TotalCost)
}
```

### 2. Tính Ngược Net sang Gross

```go
net := vnsalary.VND(30_843_750)
result, err := vnsalary.NetToGross(net, opts)
if err != nil {
	panic(err)
}

fmt.Printf("Lương Gross tương ứng: %d VND\n", result.Gross) // Kết quả: 35000000 VND
```

### 3. Đóng Bảo Hiểm Theo Mức Lương Cơ Bản Cố Định

Nhiều doanh nghiệp ký hợp đồng với mức lương đóng bảo hiểm khác lương Gross:

```go
insBase := vnsalary.VND(5_000_000) // Đóng bảo hiểm trên mức 5.000.000đ
opts := vnsalary.Options{
	InsuranceBase: &insBase,
	Dependents:    0,
	Region:        vnsalary.Region1,
	Date:          time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
}

result, err := vnsalary.GrossToNet(30_000_000, opts)
```

### 4. Làm Thêm Giờ, Phụ Cấp và Tiền Thưởng

Gross là **tổng số tiền chi trả trong tháng**. Dùng `Options.Extras` để cho thư viện biết phần nào trong đó là làm thêm giờ, phụ cấp hay tiền thưởng:

```go
opts := vnsalary.Options{
	Region: vnsalary.Region1,
	Date:   time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
	Extras: vnsalary.Extras{
		Overtime: 5_000_000,  // miễn thuế từ 2026, không tính vào lương đóng BH
		MealCash: 1_500_000,  // miễn thuế tới 1.200.000đ/tháng từ 01/07/2026
		Bonus:    10_000_000, // chịu thuế trong tháng chi trả, không tính vào lương đóng BH
	},
}
result, _ := vnsalary.GrossToNet(55_000_000, opts) // 55M đã gồm các khoản trên
fmt.Println(result.Exempt.Total, result.BonusTax)   // phần được miễn; thuế phát sinh do khoản thưởng
```

| Khoản | Thuế TNCN | Lương đóng BH (khi không truyền `InsuranceBase`) | Căn cứ |
| :--- | :--- | :--- | :--- |
| `Overtime` (trong giới hạn giờ luật cho phép) | miễn từ kỳ tính thuế 2026; trước 2026 tính thuế, kèm cảnh báo | không tính | Luật 109/2025 Điều 4.8, NĐ 253/2026 Điều 26 |
| `MealCash` | miễn tới 1.200.000đ/tháng từ 01/07/2026; trước đó 730.000đ, kèm cảnh báo (chưa xác nhận) | có tính | NĐ 253/2026 Điều 8.2.g |
| `OtherExempt` (điện thoại, trang phục…) | miễn; mức do người dùng tự chịu trách nhiệm | có tính | NĐ 253/2026 Điều 8.2.đ |
| `Bonus` | chịu thuế trong tháng chi trả; `Result.BonusTax` = thuế khi có thưởng − thuế khi không có | không tính | NĐ 253/2026 Điều 8.2.i, 46.3 |

Làm thêm giờ và tiền thưởng thay đổi theo từng tháng, nên không thuộc khoản "trả thường xuyên, ổn định" làm căn cứ đóng bảo hiểm (Luật 41/2024 Điều 31.1.b). Nếu hợp đồng cũng không tính tiền ăn hay phụ cấp khác vào lương đóng BH, hãy truyền `InsuranceBase`. `NetToGross` giữ nguyên các khoản Extras và trả về tổng gross nhỏ nhất.

### 5. Quyết Toán Thuế TNCN Năm

Thuế khấu trừ hằng tháng lệch với thuế cả năm khi thu nhập không đều, khi đổi chỗ làm hoặc không làm đủ năm (giảm trừ bản thân vẫn tính đủ 12 tháng), hay khi đăng ký người phụ thuộc muộn. `Annual` tính thuế cả năm và số còn phải nộp hoặc được hoàn:

```go
var payments []vnsalary.Payment
for month := 1; month <= 12; month++ {
	paid := time.Date(2026, time.Month(month), 28, 0, 0, 0, 0, time.UTC)
	r, _ := vnsalary.GrossToNet(30_000_000, vnsalary.Options{Region: vnsalary.Region1, Dependents: 1, Date: paid})
	payments = append(payments, r.Payment(paid)) // hoặc điền Payment theo phiếu lương
}
res, _ := vnsalary.Annual(2026, payments, vnsalary.AnnualOptions{
	DependentMonths: 12,         // một người phụ thuộc cả năm
	Healthcare:      10_000_000, // được trừ từ 2026, tối đa 23 triệu/năm
})
fmt.Println(res.Tax, res.Withheld, res.Balance) // Balance < 0: được hoàn
```

| Khoản giảm trừ | Mức | Căn cứ |
| :--- | :--- | :--- |
| Bản thân | 12 × mức tháng, dù làm bao nhiêu tháng | NĐ 253/2026 Điều 48.1.b |
| Người phụ thuộc | `DependentMonths` × mức tháng, tính từ tháng phát sinh nghĩa vụ nuôi dưỡng | NĐ 253/2026 Điều 48.2.a |
| Từ thiện, y tế (≤ 23 triệu), giáo dục (≤ 24 triệu) | theo năm, từ kỳ tính thuế 2026; có khoản này thì phải tự quyết toán | NĐ 253/2026 Điều 49, 51.3 |

Biểu thuế năm bằng biểu tháng × 12 (Luật 109/2025 Điều 9.2). Hỗ trợ kỳ tính thuế từ 2025.

---

## 📊 Bảng Căn Cứ Pháp Lý (2024 vs 2026)

| Thông Số | Từ 01/07/2024 | Từ 01/01/2026 | Từ 01/07/2026 | Căn Cứ Pháp Quy |
| :--- | :--- | :--- | :--- | :--- |
| **Lương cơ sở / Tham chiếu** | 2.340.000 đ | 2.340.000 đ | **2.530.000 đ** | NĐ 73/2024/NĐ-CP; NĐ 161/2026/NĐ-CP |
| **Trần BHXH & BHYT** | 46.800.000 đ (20 × Lương cơ sở) | 46.800.000 đ | **50.600.000 đ** | tới 30/06/2025: Luật BHXH 2014 Điều 89.3, Luật BHYT Điều 14.2; từ 01/07/2025: Luật BHXH 41/2024/QH15 Điều 31.1.đ (mức tham chiếu = lương cơ sở, Điều 141.13), Luật BHYT Điều 14.5 sửa đổi bởi Luật 51/2024/QH15 |
| **Trần BHTN (Vùng I)** | 99.200.000 đ (20 × 4,96M) | 106.200.000 đ (20 × 5,31M) | 106.200.000 đ | Luật Việc làm 2013 Điều 58 (tới hết 2025), Luật Việc làm 74/2025/QH15 Điều 34.2 (từ 01/01/2026); NĐ 74/2024 & NĐ 293/2025 |
| **Giảm trừ bản thân** | 11.000.000 đ / tháng | **15.500.000 đ / tháng** | 15.500.000 đ / tháng | NQ 954/2020 & NQ 110/2025 |
| **Giảm trừ phụ thuộc** | 4.400.000 đ / người / tháng | **6.200.000 đ / người / tháng** | 6.200.000 đ / người / tháng | NQ 954/2020 & NQ 110/2025 |
| **Biểu thuế TNCN** | 7 bậc (5% – 35%) | **5 bậc (5% – 35%)** | 5 bậc | Luật Thuế TNCN 2007 & Luật Thuế TNCN 109/2025 Điều 9 |

### Biểu Thuế Lũy Tiến 5 Bậc Mới 2026 (Luật 109/2025/QH15 Điều 9)

| Bậc | Thu Nhập Tính Thuế / Tháng (TNTT) | Thuế Suất |
| :---: | :--- | :---: |
| **1** | Đến 10 triệu đồng | **5%** |
| **2** | Trên 10 triệu đến 30 triệu đồng | **10%** |
| **3** | Trên 30 triệu đến 60 triệu đồng | **20%** |
| **4** | Trên 60 triệu đến 100 triệu đồng | **30%** |
| **5** | Trên 100 triệu đồng | **35%** |

> **Nguyên tắc thời điểm tính thuế (Cash-basis):**  
> Thời điểm xác định thu nhập chịu thuế từ tiền lương là **thời điểm tổ chức, cá nhân trả thu nhập cho người nộp thuế** (Luật Thuế TNCN số 109/2025/QH15 Điều 8 khoản 3; trước đó Thông tư 111/2013/TT-BTC Điều 8 khoản 2 điểm b, đã được thay bằng Thông tư 87/2026/TT-BTC từ 01/07/2026).  
> Do đó, lương tháng 12/2025 nếu chi trả vào ngày 05/01/2026 sẽ được áp dụng **chính sách năm 2026** (giảm trừ 15,5M và biểu 5 bậc), như cơ quan thuế đã trả lời vào tháng 01/2026.

## 🔢 Quy Ước Làm Tròn

Mọi số tiền là số nguyên đồng (`int64`), không dùng số thực ở bất kỳ bước nào.

| Khoản | Quy tắc |
| :--- | :--- |
| Từng khoản bảo hiểm, người lao động (8%, 1,5%, 1%) và công ty (17,5%, 3%, 1%) | tỉ lệ × lương đóng đã áp trần, **làm tròn nửa lên tới 1 đồng, từng khoản riêng** |
| Thuế TNCN | thuế **từng bậc** làm tròn nửa lên tới 1 đồng, rồi cộng lại |
| Net, các tổng | cộng chính xác các khoản đã làm tròn |
| Làm tròn net tới 1.000 đồng | không áp dụng (chính sách từng công ty); cần thì dùng `RoundToThousand` |

Nửa lên nghĩa là x,5 đồng làm tròn lên: 1,5% của 20.000.300 là 300.004,5, thành 300.005 (có trong golden).

Văn bản luật quy định tỉ lệ và mức trần nhưng không quy định làm tròn ở bước nào, nên các công cụ tính khác nhau (có nơi chỉ làm tròn tổng, có nơi cắt phần lẻ). Chênh lệch tối đa vài đồng. Ví dụ gross 20.000.028 năm 2026: các khoản bảo hiểm là 1.600.002,24 + 300.000,42 + 200.000,28, làm tròn từng khoản ra 2.100.002, làm tròn một lần trên tổng ra 2.100.003, nên net lệch 1 đồng (TopCV ra cùng net với vnsalary ở ví dụ này, kiểm ngày 13/10/2026). Trong một mẫu (gross 20,0M–20,2M, năm 2026, không người phụ thuộc), khoảng 28% số tiền cho net lệch 1 đồng giữa hai cách. **Lệch từ 1.000 đồng trở lên không phải do làm tròn: hãy mở issue.**


---

## ⚠️ Giới Hạn

vnsalary áp dụng cho **người lao động cư trú, hợp đồng lao động từ 3 tháng trở lên**. Hãy đọc các điểm sau trước khi dùng cho bảng lương thật:

- **Các khoản miễn thuế được nhận như người dùng truyền vào.** `Extras.Overtime` phải là tiền làm thêm trong giới hạn giờ luật cho phép (phần vượt giới hạn chỉ để trong gross), và mức của `Extras.OtherExempt` (điện thoại, trang phục…) theo quy chế công ty; thư viện không kiểm các mức này.
- **Chưa hỗ trợ:** hợp đồng dưới 3 tháng (khấu trừ 10%), người không cư trú (20%), người nước ngoài (không đóng BHTN). Khoản đã khấu trừ 10% vẫn đưa được vào `Annual`.
- **Lương đóng bảo hiểm thấp hơn lương tối thiểu vùng vẫn được tính, kèm cảnh báo.** Người làm toàn thời gian phải được trả ít nhất mức này, nên `Result.Warnings` báo mã `insurance_salary_below_regional_minimum` (CLI in dòng `[!]`). Đây không phải lỗi vì hợp đồng bán thời gian có thể thấp hơn hợp pháp.
- **Tỉ lệ TNLĐ-BNN của công ty cố định 0,5%**, nên BHXH phần công ty luôn là 17,5%. Một số doanh nghiệp đóng 0,3%.
- **Kinh phí công đoàn 2% không nằm trong `Employer.TotalCost`.** Đây không phải bảo hiểm, nhưng công ty vẫn phải đóng.
- **Vùng lương tính theo nơi người lao động làm việc**, không theo trụ sở công ty hay nơi ở. Thư viện không tự suy ra vùng; hãy truyền đúng vùng nơi làm việc.

Các tính năng dự kiến được theo dõi trong [milestones](https://github.com/HuynhNhatKhanh/vnsalary/milestones); mọi ý tưởng và yêu cầu đều có thể mở issue.

---

## 🛠️ Công Cụ Dòng Lệnh (CLI)

### Cài Đặt CLI
```bash
go install github.com/HuynhNhatKhanh/vnsalary/cmd/vnsalary@latest
```
*(Hoặc tự build file nhị phân: `go build -o vnsalary ./cmd/vnsalary`)*

**Không cài Go:** tải file chạy sẵn cho Linux, macOS hoặc Windows (amd64/arm64) ở [bản phát hành mới nhất](https://github.com/HuynhNhatKhanh/vnsalary/releases/latest), đối chiếu với `checksums.txt`, rồi chạy `vnsalary version`.

### Các Lệnh Tiêu Biểu

#### 1. Tính Gross sang Net
```bash
vnsalary gross 35000000 -d 1 -r 1 --date 2026-01-15
# Hoặc đặt flag trước số tiền:
vnsalary gross -d 1 35000000 --date 2026-01-15
```

#### 2. Tính Ngược Net sang Gross
```bash
vnsalary net 30843750 -d 1 -r 1 --date 2026-01-15
# Kết quả: đúng 35.000.000đ Gross
```

#### 3. So Sánh Hai Kỳ Luật (Diff)
```bash
vnsalary diff 35000000 -d 1 -r 1                                   # mặc định: 2024-07-01 so với 2026-01-01
vnsalary diff 60000000 --from 2026-06-30 --to 2026-07-01           # hai ngày chi trả bất kỳ
```

#### Làm thêm giờ, phụ cấp và tiền thưởng
Số tiền là tổng chi trả trong tháng; các cờ cho biết phần nào là khoản đặc biệt:
```bash
vnsalary gross 50000000 --bonus 20000000 --date 2026-08-10        # hiện phần thuế do khoản thưởng
vnsalary gross 55000000 --overtime 5000000 --meal 1500000 --other-exempt 800000 --date 2026-08-10
```

#### Quyết toán thuế TNCN năm
`vnsalary annual` đọc các lần chi trả trong năm từ file JSON (`-` để đọc từ stdin). Mỗi lần chi trả ghi `gross` (khoản khấu trừ được tính theo `dependents` và `region` của file; `bonus`, `overtime`, `meal`, `other_exempt`, `ins_base` như lệnh `gross`) hoặc ghi đúng số trên phiếu lương (`taxable_income`, `insurance`, `tax_withheld`):
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
vnsalary annual payments.json            # in CÒN PHẢI NỘP hoặc ĐƯỢC HOÀN
```
`dependent_months` mặc định bằng `dependents` × 12; `charity`, `healthcare`, `education` là số tiền cả năm. Trường lạ bị từ chối, để gõ sai tên trường không làm mất khoản giảm trừ mà không biết.

#### 4. Chọn Ngôn Ngữ Hiển Thị
Giao diện mặc định là tiếng Việt. `--lang en` chuyển toàn bộ CLI sang tiếng Anh (hướng dẫn, thông báo lỗi, bảng kết quả, số tiền dạng `35,000,000 VND`). Kết quả `--json` không phụ thuộc ngôn ngữ:
```bash
vnsalary gross 35000000 -d 1 --lang en
```

#### 5. Xuất Dữ Liệu JSON
Thêm cờ `--json` vào bất kỳ lệnh nào để tích hợp với shell script hoặc backend khác:
```bash
vnsalary gross 35000000 --json
```

---

## ⚡ Hiệu Năng & Benchmark

Ví dụ một lần chạy (`go test -bench=. -benchmem`, linux/amd64, Intel i7-10700K; số đo khác nhau tùy máy và tùy lần chạy):
```text
BenchmarkGrossToNet-16      3850897           329.0 ns/op         336 B/op          3 allocs/op
BenchmarkNetToGross-16       111998         12749   ns/op       12920 B/op        133 allocs/op
```

- `GrossToNet` dưới 1 µs; allocation đến từ danh sách bậc thuế và bản sao nguồn văn bản trong kết quả.
- `NetToGross` nội suy trên đoạn tuyến tính, quét tăng dần ±50 đồng rồi lùi tới gross nhỏ nhất có cùng net, dự phòng bằng tìm nhị phân. Lý do kết quả luôn là gross nhỏ nhất: xem [docs/design.md](docs/design.md#7-net--gross).

---

## 🧪 Kiểm Thử Chất Lượng (Test Suites)

- **Golden Test Dataset (`testdata/golden.json`):** 36 ca, mỗi ca có trường `source` và `verified_at` ghi rõ đã kiểm bằng cách nào (tính tay theo văn bản, bản tính tham chiếu độc lập, hay công cụ bên ngoài). 12 ca đã khớp tới từng đồng với công cụ của TopCV (kiểm ngày 13/10 và 15/10/2026); ca nào chưa đối chiếu công cụ ngoài thì ghi rõ là chưa. 11 ca dùng số lẻ như 23.456.789 để mọi bước làm tròn đều được kiểm, có ghi kỳ vọng từng khoản bảo hiểm.
- **Tính Bất Biến Hai Chiều:** với mọi gross `g`, `NetToGross` trả về gross **nhỏ nhất** cho cùng net, nên `Net(NetToGross(Net(g))) == Net(g)` và gross tính ngược `<= g` (do làm tròn, nhiều gross có thể ra cùng một net).
- **Bất biến của bộ luật:** `TestPoliciesInvariants` báo lỗi nếu bộ luật mới phá giả định của thuật toán tính ngược (thứ tự, trần BHXH thấp hơn trần BHTN, bậc cao nhất 35%, có nguồn văn bản).
- **Fuzz Testing:** Kiểm thử tự động `FuzzGrossToNet_NetToGross` phát hiện ca biên ngẫu nhiên.
- **Bảo Vệ Tính Toàn Vẹn:** Kiểm tra tính bất biến của dữ liệu toàn cục qua `PolicyAt`, chặn tràn số (`> 10^14` VND), và chặn âm lương khi bảo hiểm vượt quá gross.
- **CLI:** mọi lệnh, cách viết cờ, ngôn ngữ và mã thoát đều có test chạy trong tiến trình; bảng kết quả được chốt bằng file snapshot.
- **CI:** Go 1.22 (bản tối thiểu), oldstable và stable, có race detector, `go vet`, golangci-lint, gofmt và chạy fuzz ngắn.

```bash
go test -cover ./...
```
Độ bao phủ code ở v0.4.0: thư viện **97,4%**, CLI **95,9%**.

---

## 🤝 Đóng Góp

Báo thay đổi luật và báo kết quả tính lệch là hai loại đóng góp giá trị nhất; cả hai đều có mẫu issue. Xem [CONTRIBUTING.md](CONTRIBUTING.md) (cách thêm bộ luật mới, quy tắc đánh số phiên bản), [CHANGELOG.md](CHANGELOG.md) (lịch sử phát hành) và [docs/design.md](docs/design.md) (thiết kế và các quyết định).

---

## 📄 Bản Quyền

Phát hành theo giấy phép mã nguồn mở MIT License. Xem file [LICENSE](LICENSE) để biết thêm chi tiết.
