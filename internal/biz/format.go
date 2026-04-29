package biz

import (
	"fmt"
	"math"
)

// FormatUSD formats a dollar amount with commas.
// Returns "TBD" if the amount is zero or negative.
func FormatUSD(v float64) string {
	if v <= 0 {
		return "TBD"
	}
	s := fmt.Sprintf("%.0f", math.Round(v))
	n := len(s)
	out := make([]byte, 0, n+n/3)
	for i, c := range s {
		pos := n - i
		if i > 0 && pos%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return "$" + string(out)
}
