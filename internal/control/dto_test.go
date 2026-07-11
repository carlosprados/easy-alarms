package control

import (
	"testing"
	"time"

	"easy-alarms/internal/alarm"
)

func TestToDTOClock(t *testing.T) {
	now := time.Date(2026, 7, 11, 10, 0, 0, 0, time.Local)
	next := now.Add(2 * time.Hour)
	st := AlarmState{
		Alarm: alarm.Alarm{
			ID: "1", Kind: alarm.KindClock, Label: "Work", Enabled: true,
			Hour: 12, Minute: 0, Repeat: mask(time.Monday, time.Friday),
		},
		Next:    next,
		Snoozed: false,
	}
	d := ToDTO(st, now)
	if d.Time != "12:00" {
		t.Errorf("Time = %q", d.Time)
	}
	if len(d.Days) != 2 || d.Days[0] != "mon" || d.Days[1] != "fri" {
		t.Errorf("Days = %v", d.Days)
	}
	if d.NextTrigger == nil || !d.NextTrigger.Equal(next) {
		t.Errorf("NextTrigger = %v", d.NextTrigger)
	}
	if d.NextIn != "2h" {
		t.Errorf("NextIn = %q, want 2h", d.NextIn)
	}
	if d.State != "" || d.Duration != "" {
		t.Errorf("clock alarm should not carry timer fields: %+v", d)
	}
}

func TestToDTOTimerStates(t *testing.T) {
	now := time.Date(2026, 7, 11, 10, 0, 0, 0, time.Local)
	base := alarm.Alarm{ID: "t", Kind: alarm.KindTimer, Duration: 25 * time.Minute, Enabled: true}

	// running
	run := base
	run.FiresAt = now.Add(10 * time.Minute)
	d := ToDTO(AlarmState{Alarm: run, Next: run.FiresAt}, now)
	if d.State != "running" || d.Remaining != "10m" || d.Duration != "25m" {
		t.Errorf("running: %+v", d)
	}

	// paused
	pause := base
	pause.Remaining = 7 * time.Minute
	d = ToDTO(AlarmState{Alarm: pause}, now)
	if d.State != "paused" || d.Remaining != "7m" {
		t.Errorf("paused: %+v", d)
	}
	if d.NextTrigger != nil {
		t.Errorf("paused timer has no next trigger, got %v", d.NextTrigger)
	}

	// idle
	d = ToDTO(AlarmState{Alarm: base}, now)
	if d.State != "idle" || d.Remaining != "" {
		t.Errorf("idle: %+v", d)
	}
}

func TestToDTOSnoozed(t *testing.T) {
	now := time.Date(2026, 7, 11, 10, 0, 0, 0, time.Local)
	snoozeUntil := now.Add(5 * time.Minute)
	st := AlarmState{
		Alarm:   alarm.Alarm{ID: "1", Kind: alarm.KindClock, Enabled: true, Hour: 9, Minute: 0},
		Next:    snoozeUntil, // scheduler reports the snooze override as the next fire
		Snoozed: true,
	}
	d := ToDTO(st, now)
	if !d.Snoozed {
		t.Error("expected snoozed=true")
	}
	if d.NextIn != "5m" {
		t.Errorf("NextIn = %q, want 5m", d.NextIn)
	}
}
