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
	ID       string
	Label    string
	Kind     Kind
	Hour     int           // clock: 0-23
	Minute   int           // clock: 0-59
	Repeat   [7]bool       // clock: indexed by time.Weekday (0 = Sunday)
	Duration  time.Duration // timer: configured length
	FiresAt   time.Time     // timer: zero when not running
	Remaining time.Duration // timer: time left while paused, zero otherwise
	Sound    string        // audio file path; empty = built-in tone
	Enabled  bool
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
		c := time.Date(now.Year(), now.Month(), now.Day(), a.Hour, a.Minute, 0, 0, now.Location())
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

func (a *Alarm) Repeats() bool {
	for _, r := range a.Repeat {
		if r {
			return true
		}
	}
	return false
}
