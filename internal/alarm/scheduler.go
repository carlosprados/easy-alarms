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

	mu      sync.Mutex
	nextAt  map[string]time.Time
	snoozes map[string]time.Time // per-alarm override; survives Reschedule
	source  func() []*Alarm
	stop    chan struct{}
}

func NewScheduler(source func() []*Alarm) *Scheduler {
	return &Scheduler{
		nextAt:  map[string]time.Time{},
		snoozes: map[string]time.Time{},
		source:  source,
		stop:    make(chan struct{}),
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
		next, snoozed := s.snoozes[a.ID]
		if !snoozed {
			next = s.nextAt[a.ID]
		}
		if next.IsZero() || now.Before(next) {
			continue
		}
		due = append(due, a)
		delete(s.snoozes, a.ID)
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
// mutation. Active snoozes survive; they are only dropped when their alarm
// fires, is explicitly cleared (ClearSnooze) or no longer exists.
func (s *Scheduler) Reschedule() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextAt = map[string]time.Time{}
	for _, a := range s.source() {
		s.nextAt[a.ID] = a.NextTrigger(now)
	}
	for id := range s.snoozes {
		if _, ok := s.nextAt[id]; !ok {
			delete(s.snoozes, id)
		}
	}
}

// Snooze re-arms an alarm to fire after d, overriding its normal schedule.
func (s *Scheduler) Snooze(id string, d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snoozes[id] = time.Now().Add(d)
}

// ClearSnooze cancels a pending snooze, reverting the alarm to its normal
// schedule. Call when the user edits or disables the alarm.
func (s *Scheduler) ClearSnooze(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.snoozes, id)
}

// NextFor returns the currently scheduled trigger for the alarm, which may
// differ from Alarm.NextTrigger while snoozed.
func (s *Scheduler) NextFor(a *Alarm) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if next, ok := s.snoozes[a.ID]; ok {
		return next
	}
	if next, ok := s.nextAt[a.ID]; ok {
		return next
	}
	return a.NextTrigger(time.Now())
}
