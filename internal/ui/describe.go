package ui

import (
	"fmt"
	"time"
)

var weekdayNames = [7]string{
	"el domingo", "el lunes", "el martes", "el miércoles",
	"el jueves", "el viernes", "el sábado",
}

// describeNext renders the headline feature: when exactly an alarm will
// ring, in human terms with a live second-resolution countdown.
func describeNext(next, now time.Time) string {
	if next.IsZero() {
		return "💤 Inactiva"
	}
	return fmt.Sprintf("🔔 Suena %s a las %s (en %s)",
		dayLabel(next, now), next.Format("15:04"), humanDuration(next.Sub(now)))
}

// describeNextCoarse is the tray variant: minute resolution, so the tray menu
// changes at most once a minute instead of once a second.
func describeNextCoarse(next, now time.Time) string {
	return fmt.Sprintf("%s a las %s (en %s)",
		dayLabel(next, now), next.Format("15:04"), coarseDuration(next.Sub(now)))
}

func dayLabel(next, now time.Time) string {
	switch {
	case sameDay(next, now):
		return "hoy"
	case sameDay(next, now.AddDate(0, 0, 1)):
		return "mañana"
	default:
		return weekdayNames[next.Weekday()]
	}
}

// coarseDuration drops seconds: "10h 25m", "12m", "<1m".
func coarseDuration(d time.Duration) string {
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

func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func humanDuration(d time.Duration) string {
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

// compactDuration renders "1h30m", "10m", "45s" — no zero-padded noise.
func compactDuration(d time.Duration) string {
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
