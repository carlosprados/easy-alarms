package ui

import (
	"fmt"
	"strings"
	"time"

	"easy-alarms/internal/humanize"
)

var weekdayNames = [7]string{
	"el domingo", "el lunes", "el martes", "el miércoles",
	"el jueves", "el viernes", "el sábado",
}

var longWeekdays = [7]string{
	"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado",
}

var longMonths = [12]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

// longDate renders a header-style Spanish date: "Sábado, 11 de julio".
func longDate(t time.Time) string {
	return fmt.Sprintf("%s, %d de %s", longWeekdays[t.Weekday()], t.Day(), longMonths[t.Month()-1])
}

// describeNext renders the headline feature: when exactly an alarm will
// ring, in human terms with a live second-resolution countdown.
func describeNext(next, now time.Time) string {
	if next.IsZero() {
		return "💤 Inactiva"
	}
	return fmt.Sprintf("🔔 Suena %s a las %s (en %s)",
		dayLabel(next, now), next.Format("15:04"), humanize.Long(next.Sub(now)))
}

// describeNextCoarse is the tray variant: minute resolution, so the tray menu
// changes at most once a minute instead of once a second.
func describeNextCoarse(next, now time.Time) string {
	return fmt.Sprintf("%s a las %s (en %s)",
		dayLabel(next, now), next.Format("15:04"), humanize.Coarse(next.Sub(now)))
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

// shortDays are Monday-first abbreviations, indexed by time.Weekday.
var shortDays = [7]string{"Dom", "Lun", "Mar", "Mié", "Jue", "Vie", "Sáb"}

// repeatLabel summarises which days a clock alarm repeats on. It returns ""
// for a one-shot (no days set), so callers can omit the line entirely.
func repeatLabel(repeat [7]bool) string {
	var days []time.Weekday
	for d := time.Sunday; d <= time.Saturday; d++ {
		if repeat[d] {
			days = append(days, d)
		}
	}
	switch len(days) {
	case 0:
		return ""
	case 7:
		return "Todos los días"
	}
	if onlyWeekdays(repeat) {
		return "Días laborables (L–V)"
	}
	if onlyWeekend(repeat) {
		return "Fines de semana"
	}
	// Otherwise list them Monday-first.
	order := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}
	var names []string
	for _, d := range order {
		if repeat[d] {
			names = append(names, shortDays[d])
		}
	}
	return strings.Join(names, ", ")
}

func onlyWeekdays(r [7]bool) bool {
	return r[time.Monday] && r[time.Tuesday] && r[time.Wednesday] && r[time.Thursday] && r[time.Friday] &&
		!r[time.Saturday] && !r[time.Sunday]
}

func onlyWeekend(r [7]bool) bool {
	return r[time.Saturday] && r[time.Sunday] &&
		!r[time.Monday] && !r[time.Tuesday] && !r[time.Wednesday] && !r[time.Thursday] && !r[time.Friday]
}

func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
