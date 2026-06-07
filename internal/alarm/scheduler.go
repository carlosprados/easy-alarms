package alarm

import (
	"sync"
	"time"
)

// Scheduler fires alarms by polling the wall clock once per second. Polling
// (instead of long time.Timers) survives system suspend: a missed alarm
// fires right after resume.
type Scheduler struct {
	OnRing func(*Alarm)

	mu     sync.Mutex
	nextAt map[string]time.Time
	source func() []*Alarm
	stop   chan struct{}
}

func NewScheduler(source func() []*Alarm) *Scheduler {
	return &Scheduler{
		nextAt: map[string]time.Time{},
		source: source,
		stop:   make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.Reschedule()
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) loop() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case now := <-t.C:
			s.tick(now)
		}
	}
}

func (s *Scheduler) tick(now time.Time) {
	s.mu.Lock()
	var due []*Alarm
	for _, a := range s.source() {
		next, ok := s.nextAt[a.ID]
		if !ok || next.IsZero() || now.Before(next) {
			continue
		}
		due = append(due, a)
		s.nextAt[a.ID] = time.Time{} // disarm until explicitly rescheduled
	}
	s.mu.Unlock()
	for _, a := range due {
		if s.OnRing != nil {
			s.OnRing(a)
		}
	}
}

// Reschedule recomputes all trigger times from alarm state. Call after any
// mutation. Note it clears active snoozes.
func (s *Scheduler) Reschedule() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextAt = map[string]time.Time{}
	for _, a := range s.source() {
		s.nextAt[a.ID] = a.NextTrigger(now)
	}
}

// Snooze re-arms an alarm to fire after d, overriding its normal schedule.
func (s *Scheduler) Snooze(id string, d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextAt[id] = time.Now().Add(d)
}

// NextFor returns the currently scheduled trigger for the alarm, which may
// differ from Alarm.NextTrigger while snoozed.
func (s *Scheduler) NextFor(a *Alarm) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if next, ok := s.nextAt[a.ID]; ok {
		return next
	}
	return a.NextTrigger(time.Now())
}
