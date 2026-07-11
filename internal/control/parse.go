package control

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// timePattern mirrors the GUI's clock-time validation (internal/ui/dialogs.go).
var timePattern = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)

// ParseClockTime parses "HH:MM" (24h) into hour/minute.
func ParseClockTime(s string) (hour, minute int, err error) {
	s = strings.TrimSpace(s)
	if !timePattern.MatchString(s) {
		return 0, 0, fmt.Errorf("invalid time %q, expected HH:MM (24h)", s)
	}
	fmt.Sscanf(s, "%d:%d", &hour, &minute)
	return hour, minute, nil
}

// dayToken maps English and Spanish day names (long and short, accented or
// not) to a time.Weekday.
var dayToken = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday, "dom": time.Sunday, "domingo": time.Sunday,
	"mon": time.Monday, "monday": time.Monday, "lun": time.Monday, "lunes": time.Monday,
	"tue": time.Tuesday, "tuesday": time.Tuesday, "mar": time.Tuesday, "martes": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday, "mie": time.Wednesday, "mié": time.Wednesday,
	"miercoles": time.Wednesday, "miércoles": time.Wednesday,
	"thu": time.Thursday, "thursday": time.Thursday, "jue": time.Thursday, "jueves": time.Thursday,
	"fri": time.Friday, "friday": time.Friday, "vie": time.Friday, "viernes": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday, "sab": time.Saturday, "sáb": time.Saturday,
	"sabado": time.Saturday, "sábado": time.Saturday,
}

// ParseDays turns a --days value into a repeat mask indexed by time.Weekday
// (0 = Sunday). It accepts the empty string (a one-shot alarm), the keywords
// daily/diario, weekdays/laborables and weekend/finde, or a comma-separated
// list of English or Spanish day names.
func ParseDays(s string) ([7]bool, error) {
	var r [7]bool
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "":
		return r, nil // one-shot
	case "daily", "diario", "everyday":
		for i := range r {
			r[i] = true
		}
		return r, nil
	case "weekdays", "laborables", "workdays":
		r[time.Monday], r[time.Tuesday], r[time.Wednesday], r[time.Thursday], r[time.Friday] = true, true, true, true, true
		return r, nil
	case "weekend", "weekends", "finde":
		r[time.Saturday], r[time.Sunday] = true, true
		return r, nil
	}
	for tok := range strings.SplitSeq(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		d, ok := dayToken[tok]
		if !ok {
			return [7]bool{}, fmt.Errorf("unknown day %q (use mon..sun, lun..dom, or daily/weekdays/weekend)", tok)
		}
		r[d] = true
	}
	return r, nil
}

// ParseTimerDuration parses a Go duration string and requires it to be positive.
func ParseTimerDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q, e.g. 10m, 1h30m, 90s", s)
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be positive, got %q", s)
	}
	return d, nil
}

// englishShort are canonical Monday-first day abbreviations for the wire,
// indexed by time.Weekday.
var englishShort = [7]string{
	time.Sunday: "sun", time.Monday: "mon", time.Tuesday: "tue", time.Wednesday: "wed",
	time.Thursday: "thu", time.Friday: "fri", time.Saturday: "sat",
}

var mondayFirst = []time.Weekday{
	time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday,
}

// DaysToStrings renders a repeat mask as canonical English short names,
// Monday-first. A one-shot (no days set) returns nil.
func DaysToStrings(r [7]bool) []string {
	var out []string
	for _, d := range mondayFirst {
		if r[d] {
			out = append(out, englishShort[d])
		}
	}
	return out
}
