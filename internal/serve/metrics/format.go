package metrics

import "fmt"

// FormatDollars renders a dollar amount per the spec's tile-headline
// rules: thousands shown as "$Nk" or "$N.NK" depending on magnitude.
// Negative numbers prefix the dash before the dollar sign.
func FormatDollars(v float64) string {
	if v < 0 {
		return "-" + FormatDollars(-v)
	}
	if v < 1000 {
		return fmt.Sprintf("$%.0f", v)
	}
	thousands := v / 1000
	if thousands >= 100 {
		return fmt.Sprintf("$%.0fK", thousands)
	}
	return fmt.Sprintf("$%.1fK", thousands)
}

// FormatHours renders an hours scalar as "~Nh" rounded to the nearest
// whole hour.
func FormatHours(h float64) string {
	if h < 0 {
		h = 0
	}
	return fmt.Sprintf("~%.0fh", h)
}

// FormatROIMultiple renders an ROI multiple per the spec's rules:
//   - 0 (sentinel for zero spend) → "—"
//   - <10 → two decimals (e.g. "3.21x")
//   - 10-999 → one decimal (e.g. "44.0x")
//   - >=1000 → capped at "999x"
func FormatROIMultiple(m float64) string {
	if m == 0 {
		return "—"
	}
	if m >= 1000 {
		return "999×"
	}
	if m >= 10 {
		return fmt.Sprintf("%.1f×", m)
	}
	return fmt.Sprintf("%.2f×", m)
}
