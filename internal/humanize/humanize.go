// Package humanize renders time.Duration values as short, language-neutral
// strings ("1h30m", "2h 05m", "45s"). The output uses only digits and unit
// letters, so it is shared by the Spanish GUI and the English CLI alike.
package humanize

import (
	"fmt"
	"time"
)

// Long renders a second-resolution duration: "2d 3h", "2h 05m", "12m 30s",
// "45s". Negative durations clamp to zero.
func Long(d time.Duration) string {
	d = max(d.Round(time.Second), 0)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h >= 24:
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	case h > 0:
		return fmt.Sprintf("%dh %02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// Coarse drops seconds: "10h 25m", "12m", "<1m". It is used where a live
// display should change at most once a minute instead of once a second.
func Coarse(d time.Duration) string {
	d = max(d, 0)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h >= 24:
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	case h > 0:
		return fmt.Sprintf("%dh %02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return "<1m"
	}
}

// Compact renders "1h30m", "10m", "45s" — no separators, no zero-padded noise.
func Compact(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	out := ""
	if h > 0 {
		out += fmt.Sprintf("%dh", h)
	}
	if m > 0 {
		out += fmt.Sprintf("%dm", m)
	}
	if s > 0 || out == "" {
		out += fmt.Sprintf("%ds", s)
	}
	return out
}
