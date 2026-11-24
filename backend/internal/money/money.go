// Package money parses and formats fixed two-decimal amounts without floats.
package money

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var amountPattern = regexp.MustCompile(`^\d{1,9}(\.\d{1,2})?$`)

// Parse converts a decimal string such as "328.00" into cents.
func Parse(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if !amountPattern.MatchString(s) {
		return 0, fmt.Errorf("金额格式必须为十进制字符串，最多两位小数")
	}
	whole, frac, _ := strings.Cut(s, ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("金额超出范围")
	}
	var f int64
	switch len(frac) {
	case 0:
	case 1:
		f = int64(frac[0]-'0') * 10
	default:
		f = int64(frac[0]-'0')*10 + int64(frac[1]-'0')
	}
	return w*100 + f, nil
}

// Format renders cents as a two-decimal string.
func Format(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// FormatRate rounds a ratio to four decimal places (e.g. 0.3484).
func FormatRate(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10000) / 10000
}
