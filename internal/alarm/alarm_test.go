package alarm

import (
	"testing"
	"time"
)

// reference instant: a fixed wall-clock used across cases. Expectations are
// derived from it (via Weekday/AddDate) so the test is self-consistent
// regardless of which weekday the date happens to fall on.
var now = time.Date(2026, 6, 10, 10, 0, 0, 0, time.Local)

func clock(hour, minute int, repeat [7]bool) *Alarm {
	return &Alarm{Kind: KindClock, Hour: hour, Minute: minute, Repeat: repeat, Enabled: true}
}

// onlyOn builds a repeat mask set on a single weekday.
func onlyOn(d time.Weekday) [7]bool {
	var r [7]bool
	r[d] = true
	return r
}

func at(base time.Time, hour, minute int) time.Time {
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, base.Location())
}

func TestNextTrigger(t *testing.T) {
	noRepeat := [7]bool{}

	cases := []struct {
		name string
		a    *Alarm
		want time.Time
	}{
		{
			name: "disabled alarm never triggers",
			a:    &Alarm{Kind: KindClock, Hour: 14, Enabled: false},
			want: time.Time{},
		},
		{
			name: "running timer returns its FiresAt",
			a:    &Alarm{Kind: KindTimer, FiresAt: now.Add(90 * time.Minute), Enabled: true},
			want: now.Add(90 * time.Minute),
		},
		{
			name: "stopped timer never triggers",
			a:    &Alarm{Kind: KindTimer, Enabled: true},
			want: time.Time{},
		},
		{
			name: "one-shot later today fires today",
			a:    clock(14, 30, noRepeat),
			want: at(now, 14, 30),
		},
		{
			name: "one-shot already passed today fires tomorrow",
			a:    clock(8, 0, noRepeat),
			want: at(now.AddDate(0, 0, 1), 8, 0),
		},
		{
			name: "repeating today, time still ahead, fires today",
			a:    clock(14, 0, onlyOn(now.Weekday())),
			want: at(now, 14, 0),
		},
		{
			name: "repeating today, time passed, fires same weekday next week",
			a:    clock(8, 0, onlyOn(now.Weekday())),
			want: at(now.AddDate(0, 0, 7), 8, 0),
		},
		{
			name: "repeating on a weekday three days out",
			a:    clock(9, 15, onlyOn((now.Weekday()+3)%7)),
			want: at(now.AddDate(0, 0, 3), 9, 15),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.a.NextTrigger(now)
			if !got.Equal(tc.want) {
				t.Errorf("NextTrigger() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRepeats(t *testing.T) {
	if clock(8, 0, [7]bool{}).Repeats() {
		t.Error("alarm with no repeat days should not repeat")
	}
	if !clock(8, 0, onlyOn(time.Monday)).Repeats() {
		t.Error("alarm with a repeat day should repeat")
	}
}

func TestNextTriggerDated(t *testing.T) {
	a := clock(7, 0, [7]bool{})
	a.Date = now.AddDate(0, 0, 20).Format(DateLayout)
	if got, want := a.NextTrigger(now), at(now.AddDate(0, 0, 20), 7, 0); !got.Equal(want) {
		t.Errorf("future date: got %v, want %v", got, want)
	}
	a.Date = now.Format(DateLayout) // 07:00 today is before now (10:00)
	if got := a.NextTrigger(now); !got.IsZero() {
		t.Errorf("past date: got %v, want zero", got)
	}
}

func TestSkip(t *testing.T) {
	daily := [7]bool{true, true, true, true, true, true, true}

	a := clock(11, 0, daily)
	first, ok := a.SkipNext(now)
	if !ok || !first.Equal(at(now, 11, 0)) {
		t.Fatalf("SkipNext = %v, %v; want today 11:00", first, ok)
	}
	if got, want := a.NextTrigger(now), at(now.AddDate(0, 0, 1), 11, 0); !got.Equal(want) {
		t.Errorf("after one skip: got %v, want %v", got, want)
	}
	a.SkipNext(now)
	if got, want := a.NextTrigger(now), at(now.AddDate(0, 0, 2), 11, 0); !got.Equal(want) {
		t.Errorf("after two skips: got %v, want %v", got, want)
	}

	// A vacation longer than the 8-day search window still resolves.
	a.SkipUntil = now.AddDate(0, 0, 15).Format(DateLayout)
	if got, want := a.NextTrigger(now), at(now.AddDate(0, 0, 16), 11, 0); !got.Equal(want) {
		t.Errorf("vacation: got %v, want %v", got, want)
	}

	// A skip in the past is inert.
	a.SkipUntil = now.AddDate(0, 0, -1).Format(DateLayout)
	if _, active := a.ActiveSkip(now); active {
		t.Error("past skip reported active")
	}
	if got := a.NextTrigger(now); !got.Equal(at(now, 11, 0)) {
		t.Errorf("past skip: got %v", got)
	}

	oneShot := clock(11, 0, [7]bool{})
	if _, ok := oneShot.SkipNext(now); ok {
		t.Error("one-shot alarm accepted a skip")
	}
}
