package control

import (
	"fmt"
	"time"

	"easy-alarms/internal/alarm"
	"easy-alarms/internal/humanize"
)

// AlarmDTO is the wire representation of an alarm or timer. It uses snake_case
// tags and human-friendly strings instead of the tagless, capitalized on-disk
// alarm.Alarm encoding, so the CLI/MCP surface stays stable and readable.
type AlarmDTO struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"` // "clock" | "timer"
	Label   string `json:"label,omitempty"`
	Enabled bool   `json:"enabled"`

	// Clock fields.
	Time string   `json:"time,omitempty"` // "07:30"
	Days []string `json:"days,omitempty"` // ["mon","tue"]; empty = one-shot

	// Timer fields.
	Duration  string `json:"duration,omitempty"`  // configured length, e.g. "25m"
	State     string `json:"state,omitempty"`     // "idle" | "running" | "paused"
	Remaining string `json:"remaining,omitempty"` // time left while running/paused

	Sound string `json:"sound,omitempty"` // "" = built-in tone

	NextTrigger *time.Time `json:"next_trigger,omitempty"` // includes snooze override
	NextIn      string     `json:"next_in,omitempty"`      // human countdown to NextTrigger
	Snoozed     bool       `json:"snoozed,omitempty"`
	Ringing     bool       `json:"ringing,omitempty"`
}

// StatusDTO is the response of GET /status: the whole observable app state.
type StatusDTO struct {
	Version string     `json:"version"`
	Now     time.Time  `json:"now"`
	Ringing []string   `json:"ringing"`        // IDs of alarms with an open ring dialog
	Next    *AlarmDTO  `json:"next,omitempty"` // soonest scheduled alarm/timer
	Alarms  []AlarmDTO `json:"alarms"`
}

// SettingsDTO is the read-only settings view (GET /settings).
type SettingsDTO struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	ShowSeconds bool    `json:"show_seconds"`
}

// AlarmState is the value snapshot the Backend hands to the wire layer. It is a
// copy taken on the Fyne main thread, so converting it to a DTO is race-free.
type AlarmState struct {
	Alarm   alarm.Alarm
	Next    time.Time // scheduler's next-fire time (snooze-aware); zero if never
	Snoozed bool
	Ringing bool
}

// ToDTO converts a snapshot into its wire form. now is captured once per
// request so all countdowns in a response share a consistent clock.
func ToDTO(s AlarmState, now time.Time) AlarmDTO {
	a := s.Alarm
	d := AlarmDTO{
		ID:      a.ID,
		Kind:    string(a.Kind),
		Label:   a.Label,
		Enabled: a.Enabled,
		Sound:   a.Sound,
		Snoozed: s.Snoozed,
		Ringing: s.Ringing,
	}
	if !s.Next.IsZero() {
		next := s.Next
		d.NextTrigger = &next
		d.NextIn = humanize.Compact(next.Sub(now))
	}
	switch a.Kind {
	case alarm.KindClock:
		d.Time = fmt.Sprintf("%02d:%02d", a.Hour, a.Minute)
		d.Days = DaysToStrings(a.Repeat)
	case alarm.KindTimer:
		d.Duration = humanize.Compact(a.Duration)
		switch {
		case !a.FiresAt.IsZero():
			d.State = "running"
			d.Remaining = humanize.Compact(a.FiresAt.Sub(now))
		case a.Remaining > 0:
			d.State = "paused"
			d.Remaining = humanize.Compact(a.Remaining)
		default:
			d.State = "idle"
		}
	}
	return d
}

// --- request bodies ---

// CreateAlarmRequest creates a clock alarm. It is enabled immediately.
type CreateAlarmRequest struct {
	Label string `json:"label,omitempty"`
	At    string `json:"at"`             // "HH:MM"
	Days  string `json:"days,omitempty"` // "daily"|"weekdays"|"weekend"|"mon,tue"|""
	Sound string `json:"sound,omitempty"`
}

// CreateTimerRequest creates a timer. It starts running unless Start is false.
type CreateTimerRequest struct {
	Label    string `json:"label,omitempty"`
	Duration string `json:"duration"` // Go duration, > 0
	Sound    string `json:"sound,omitempty"`
	Start    *bool  `json:"start,omitempty"` // default true
}

// UpdateAlarmRequest is a partial update: only non-nil fields change.
type UpdateAlarmRequest struct {
	Label    *string `json:"label,omitempty"`
	At       *string `json:"at,omitempty"`       // clock only
	Days     *string `json:"days,omitempty"`     // clock only
	Duration *string `json:"duration,omitempty"` // timer only
	Sound    *string `json:"sound,omitempty"`
}

// SnoozeRequest carries the snooze length; empty means the 5m default.
type SnoozeRequest struct {
	For string `json:"for,omitempty"`
}

// ErrorResponse is the body of any non-2xx response.
type ErrorResponse struct {
	Error string `json:"error"`
}
