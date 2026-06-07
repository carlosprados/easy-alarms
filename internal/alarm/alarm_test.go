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
