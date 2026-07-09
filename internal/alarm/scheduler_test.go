package alarm

import (
	"testing"
	"time"
)

// A snoozed alarm must ring even if unrelated mutations trigger reschedules
// in between. Regression test: Reschedule used to wipe all snoozes, so a
// snoozed one-shot (disabled after ringing) silently never fired.
func TestSnoozeSurvivesReschedule(t *testing.T) {
	a := &Alarm{ID: "a", Kind: KindClock, Hour: 8, Enabled: false} // one-shot that already rang
	s := NewScheduler(func() []*Alarm { return []*Alarm{a} })

	s.Reschedule()
	s.Snooze(a.ID, 5*time.Minute)
	s.Reschedule() // e.g. the user toggled some other alarm

	next := s.NextFor(a)
	if next.IsZero() {
		t.Fatal("snooze was lost across Reschedule")
	}

	var rang []*Alarm
	s.OnRing = func(al *Alarm) { rang = append(rang, al) }
	s.tick(next.Add(time.Second))
	if len(rang) != 1 || rang[0] != a {
		t.Fatalf("snoozed alarm did not fire, rang = %v", rang)
	}
	if got := s.NextFor(a); !got.IsZero() {
		t.Errorf("disabled one-shot should be disarmed after firing, got %v", got)
	}
}

func TestClearSnooze(t *testing.T) {
	a := &Alarm{ID: "a", Kind: KindClock, Hour: 8, Enabled: false}
	s := NewScheduler(func() []*Alarm { return []*Alarm{a} })

	s.Reschedule()
	s.Snooze(a.ID, 5*time.Minute)
	s.ClearSnooze(a.ID)
	if got := s.NextFor(a); !got.IsZero() {
		t.Errorf("cleared snooze should leave the disabled alarm disarmed, got %v", got)
	}
}

func TestSnoozePrunedWhenAlarmRemoved(t *testing.T) {
	alarms := []*Alarm{{ID: "a", Kind: KindTimer, Enabled: true}}
	s := NewScheduler(func() []*Alarm { return alarms })

	s.Snooze("a", 5*time.Minute)
	alarms = nil
	s.Reschedule()

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.snoozes) != 0 {
		t.Errorf("snooze for a removed alarm should be pruned, got %v", s.snoozes)
	}
}
