package alarm

import (
	"fmt"
	"time"
)

type Kind string

const (
	KindClock Kind = "clock"
	KindTimer Kind = "timer"
)

type Alarm struct {
	ID        string
	Label     string
	Kind      Kind
	Hour      int           // clock: 0-23
	Minute    int           // clock: 0-59
	Repeat    [7]bool       // clock: indexed by time.Weekday (0 = Sunday)
	Duration  time.Duration // timer: configured length
	FiresAt   time.Time     // timer: zero when not running
	Remaining time.Duration // timer: time left while paused, zero otherwise
	Sound     string        // audio file path; empty = built-in tone
	Enabled   bool

	Date      string // clock: "2006-01-02"; rings once on that day (Repeat unused)
	Note      string // free text shown in the editor and when ringing
	SkipUntil string // clock: "2006-01-02"; occurrences up to that day are skipped
}

// DateLayout is the civil-date format of Date and SkipUntil.
const DateLayout = "2006-01-02"

// ParseDate parses a civil date in loc. An empty string yields ok=false.
func ParseDate(s string, loc *time.Location) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	d, err := time.ParseInLocation(DateLayout, s, loc)
	return d, err == nil
}

func New(kind Kind) *Alarm {
	return &Alarm{
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:    kind,
		Enabled: true,
	}
}

// NextTrigger returns the next time the alarm should fire, or zero if it
// never will (disabled, or a timer that is not running).
func (a *Alarm) NextTrigger(now time.Time) time.Time {
	if !a.Enabled {
		return time.Time{}
	}
	switch a.Kind {
	case KindTimer:
		return a.FiresAt
	case KindClock:
		if d, ok := ParseDate(a.Date, now.Location()); ok {
			c := time.Date(d.Year(), d.Month(), d.Day(), a.Hour, a.Minute, 0, 0, now.Location())
			if c.After(now) {
				return c
			}
			return time.Time{}
		}
		day := now
		if skip, ok := a.ActiveSkip(now); ok {
			day = skip.AddDate(0, 0, 1)
		}
		c := time.Date(day.Year(), day.Month(), day.Day(), a.Hour, a.Minute, 0, 0, now.Location())
		for range 8 {
			if c.After(now) && a.firesOn(c.Weekday()) {
				return c
			}
			c = c.AddDate(0, 0, 1)
		}
	}
	return time.Time{}
}

// firesOn reports whether the alarm rings on the given weekday. An alarm
// with no repeat days set is a one-shot and fires on any day.
func (a *Alarm) firesOn(d time.Weekday) bool {
	return !a.Repeats() || a.Repeat[d]
}

// ActiveSkip returns the SkipUntil day when it is today or later; a past skip
// is inert and reported as absent.
func (a *Alarm) ActiveSkip(now time.Time) (time.Time, bool) {
	d, ok := ParseDate(a.SkipUntil, now.Location())
	if !ok {
		return time.Time{}, false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return d, !d.Before(today)
}

// Skippable reports whether the alarm has recurring occurrences to skip.
func (a *Alarm) Skippable() bool {
	return a.Kind == KindClock && a.Date == "" && a.Repeats()
}

// SkipNext skips the next occurrence that would ring and returns it; calling
// it again skips the one after. ok is false when there is nothing to skip.
func (a *Alarm) SkipNext(now time.Time) (time.Time, bool) {
	if !a.Skippable() {
		return time.Time{}, false
	}
	next := a.NextTrigger(now)
	if next.IsZero() {
		return time.Time{}, false
	}
	a.SkipUntil = next.Format(DateLayout)
	return next, true
}

func (a *Alarm) Repeats() bool {
	for _, r := range a.Repeat {
		if r {
			return true
		}
	}
	return false
}
