package ui

import (
	"time"

	"fyne.io/fyne/v2"

	"easy-alarms/internal/alarm"
	"easy-alarms/internal/control"
)

// Controller returns a control.Backend that drives this UI. Every method hops
// onto the Fyne main thread with fyne.DoAndWait before touching alarm state,
// honoring the app-wide invariant that alarms are mutated only there, and
// reuses commit() so persistence, rescheduling and redraw stay in lockstep.
func (u *UI) Controller() control.Backend {
	return controller{u}
}

type controller struct{ u *UI }

// findByID returns the shared alarm pointer for id, or nil. Main thread only.
func (u *UI) findByID(id string) *alarm.Alarm {
	for _, a := range u.store.List() {
		if a.ID == id {
			return a
		}
	}
	return nil
}

// snapshotOne captures a race-free value copy of an alarm plus its scheduler
// and ring state. Main thread only.
func (u *UI) snapshotOne(a *alarm.Alarm) control.AlarmState {
	_, ringing := u.ringing[a.ID]
	return control.AlarmState{
		Alarm:   *a,
		Next:    u.sched.NextFor(a),
		Snoozed: u.sched.Snoozed(a.ID),
		Ringing: ringing,
	}
}

func (c controller) Snapshot() (states []control.AlarmState, ringing []string) {
	fyne.DoAndWait(func() {
		for _, a := range c.u.store.List() {
			states = append(states, c.u.snapshotOne(a))
		}
		ringing = c.u.ringingIDs()
	})
	return
}

func (c controller) Get(id string) (st control.AlarmState, err error) {
	fyne.DoAndWait(func() {
		a := c.u.findByID(id)
		if a == nil {
			err = control.ErrNotFound
			return
		}
		st = c.u.snapshotOne(a)
	})
	return
}

func (c controller) CreateAlarm(spec control.ClockSpec) (st control.AlarmState, err error) {
	fyne.DoAndWait(func() {
		a := alarm.New(alarm.KindClock)
		a.Label = spec.Label
		a.Hour = spec.Hour
		a.Minute = spec.Minute
		a.Repeat = spec.Repeat
		a.Sound = spec.Sound
		c.u.store.Add(a)
		c.u.commit()
		st = c.u.snapshotOne(a)
	})
	return
}

func (c controller) CreateTimer(spec control.TimerSpec) (st control.AlarmState, err error) {
	fyne.DoAndWait(func() {
		a := alarm.New(alarm.KindTimer)
		a.Label = spec.Label
		a.Duration = spec.Duration
		a.Sound = spec.Sound
		if spec.Start {
			a.FiresAt = time.Now().Add(spec.Duration)
		}
		c.u.store.Add(a)
		c.u.commit()
		st = c.u.snapshotOne(a)
	})
	return
}

func (c controller) Update(id string, p control.Patch) (st control.AlarmState, err error) {
	fyne.DoAndWait(func() {
		a := c.u.findByID(id)
		if a == nil {
			err = control.ErrNotFound
			return
		}
		clockFields := p.Hour != nil || p.Minute != nil || p.Repeat != nil
		if clockFields && a.Kind != alarm.KindClock {
			err = control.ErrWrongKind
			return
		}
		if p.Duration != nil && a.Kind != alarm.KindTimer {
			err = control.ErrWrongKind
			return
		}
		if p.Label != nil {
			a.Label = *p.Label
		}
		if p.Sound != nil {
			a.Sound = *p.Sound
		}
		if p.Hour != nil {
			a.Hour = *p.Hour
		}
		if p.Minute != nil {
			a.Minute = *p.Minute
		}
		if p.Repeat != nil {
			a.Repeat = *p.Repeat
		}
		if p.Duration != nil {
			a.Duration = *p.Duration
			// Changing a running timer's duration restarts its countdown.
			if !a.FiresAt.IsZero() {
				a.FiresAt = time.Now().Add(*p.Duration)
			}
		}
		// A pending snooze belongs to the pre-edit schedule; drop it (mirrors
		// the GUI editor).
		c.u.sched.ClearSnooze(id)
		c.u.commit()
		st = c.u.snapshotOne(a)
	})
	return
}

func (c controller) Delete(id string) (err error) {
	fyne.DoAndWait(func() {
		a := c.u.findByID(id)
		if a == nil {
			err = control.ErrNotFound
			return
		}
		c.u.dismissRinging(id) // close its ring dialog if open
		c.u.store.Remove(id)
		c.u.sched.ClearSnooze(id)
		c.u.commit()
	})
	return
}

func (c controller) SetEnabled(id string, on bool) (st control.AlarmState, err error) {
	fyne.DoAndWait(func() {
		a := c.u.findByID(id)
		if a == nil {
			err = control.ErrNotFound
			return
		}
		if a.Kind != alarm.KindClock {
			err = control.ErrWrongKind // timers use start/stop, not enable/disable
			return
		}
		if a.Enabled != on {
			a.Enabled = on
			if !on {
				c.u.sched.ClearSnooze(id)
			}
			c.u.commit()
		}
		st = c.u.snapshotOne(a)
	})
	return
}

func (c controller) TimerOp(id string, op control.TimerOp) (st control.AlarmState, err error) {
	fyne.DoAndWait(func() {
		a := c.u.findByID(id)
		if a == nil {
			err = control.ErrNotFound
			return
		}
		if a.Kind != alarm.KindTimer {
			err = control.ErrWrongKind
			return
		}
		running := !a.FiresAt.IsZero()
		paused := a.Remaining > 0
		switch op {
		case control.OpStart:
			if running || paused {
				err = control.ErrBadState // use resume for a paused timer
				return
			}
			a.Enabled = true
			a.FiresAt = time.Now().Add(a.Duration)
		case control.OpPause:
			if !running {
				err = control.ErrBadState
				return
			}
			if left := time.Until(a.FiresAt); left > 0 {
				a.Remaining = left
			}
			a.FiresAt = time.Time{}
		case control.OpResume:
			if !paused {
				err = control.ErrBadState
				return
			}
			a.Enabled = true
			a.FiresAt = time.Now().Add(a.Remaining)
			a.Remaining = 0
		case control.OpStop:
			// Idempotent: stopping an idle timer just leaves it idle.
			a.FiresAt = time.Time{}
			a.Remaining = 0
			c.u.sched.ClearSnooze(id)
		default:
			err = control.ErrBadState
			return
		}
		c.u.commit()
		st = c.u.snapshotOne(a)
	})
	return
}

func (c controller) SnoozeRinging(id string, d time.Duration) (err error) {
	fyne.DoAndWait(func() {
		if !c.u.snoozeRinging(id, d) {
			err = control.ErrNotRinging
		}
	})
	return
}

func (c controller) DismissRinging(id string) (err error) {
	fyne.DoAndWait(func() {
		if !c.u.dismissRinging(id) {
			err = control.ErrNotRinging
		}
	})
	return
}

func (c controller) Settings() (out control.SettingsDTO) {
	fyne.DoAndWait(func() {
		s := c.u.settings
		out = control.SettingsDTO{Lat: s.Lat, Lon: s.Lon, ShowSeconds: s.ShowSeconds}
	})
	return
}
