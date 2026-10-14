// Package vnsalary provides an exact Gross-to-Net and Net-to-Gross salary and
// personal income tax (PIT / Thuế TNCN) calculator according to Vietnamese labor
// and taxation regulations.
//
// The library determines statutory parameters based on the income payment date
// (Date): taxable income from salaries is determined when the income is paid
// (Law on PIT No. 109/2025/QH15, Article 8.3; previously Circular
// 111/2013/TT-BTC, Article 8.2.b).
//
// All monetary values are represented as integer Vietnamese Dong (VND int64)
// with half-up integer rounding independently applied to mandatory insurance
// items and progressive tax brackets. No floating-point arithmetic (zero float64)
// and zero external dependencies (Go Standard Library only).
package vnsalary
