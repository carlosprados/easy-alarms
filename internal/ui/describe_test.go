package ui

import (
	"testing"
	"time"

	"easy-alarms/internal/alarm"
)

func TestDescribeNextCoarseHasNoSeconds(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.Local)
	next := now.Add(12*time.Minute + 30*time.Second)
	got := describeNextCoarse(next, now)
	want := "hoy a las 10:12 (en 12m)" // seconds dropped → tray won't churn every second
	if got != want {
		t.Errorf("describeNextCoarse = %q, want %q", got, want)
	}
}

func TestRepeatLabel(t *testing.T) {
	mask := func(days ...time.Weekday) [7]bool {
		var r [7]bool
		for _, d := range days {
			r[d] = true
		}
		return r
	}
	cases := []struct {
		name   string
		repeat [7]bool
		want   string
	}{
		{"one-shot", mask(), ""},
		{"every day", mask(time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday), "Todos los días"},
		{"weekdays", mask(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday), "Días laborables (L–V)"},
		{"weekend", mask(time.Saturday, time.Sunday), "Fines de semana"},
		{"mon wed fri", mask(time.Monday, time.Wednesday, time.Friday), "Lun, Mié, Vie"},
		{"single sunday listed Monday-first", mask(time.Sunday), "Dom"},
		{"sat+sun+mon is not weekend", mask(time.Saturday, time.Sunday, time.Monday), "Lun, Sáb, Dom"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := repeatLabel(c.repeat); got != c.want {
				t.Errorf("repeatLabel = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDayLabel(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 0, 0, 0, time.Local) // Monday
	if got := dayLabel(now.Add(time.Hour), now); got != "hoy" {
		t.Errorf("same day = %q, want hoy", got)
	}
	if got := dayLabel(now.AddDate(0, 0, 1), now); got != "mañana" {
		t.Errorf("next day = %q, want mañana", got)
	}
	if got := dayLabel(now.AddDate(0, 0, 3), now); got != "el jueves" {
		t.Errorf("three days = %q, want el jueves", got)
	}
}

func TestScheduleLabel(t *testing.T) {
	now := time.Date(2026, 10, 30, 9, 0, 0, 0, time.Local) // Friday
	weekdays := [7]bool{false, true, true, true, true, true, false}

	dated := &alarm.Alarm{Kind: alarm.KindClock, Hour: 7, Date: "2026-11-02"}
	if got, want := scheduleLabel(dated, now), "Solo el lunes 2 de noviembre"; got != want {
		t.Errorf("dated: %q, want %q", got, want)
	}

	one := &alarm.Alarm{Kind: alarm.KindClock, Hour: 7, Repeat: weekdays, Enabled: true, SkipUntil: "2026-11-02"}
	if got, want := scheduleLabel(one, now), "Días laborables (L–V) · ⏭ no suena el lunes 2 de noviembre"; got != want {
		t.Errorf("single skip: %q, want %q", got, want)
	}

	vacation := &alarm.Alarm{Kind: alarm.KindClock, Hour: 7, Repeat: weekdays, Enabled: true, SkipUntil: "2026-11-10"}
	if got, want := scheduleLabel(vacation, now), "Días laborables (L–V) · ⏭ en pausa hasta el martes 10 de noviembre incluido"; got != want {
		t.Errorf("vacation: %q, want %q", got, want)
	}
}
