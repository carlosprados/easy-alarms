package control

import (
	"testing"
	"time"
)

func TestParseClockTime(t *testing.T) {
	ok := []struct {
		in   string
		h, m int
	}{
		{"07:30", 7, 30},
		{"7:05", 7, 5},
		{"00:00", 0, 0},
		{"23:59", 23, 59},
		{" 9:15 ", 9, 15},
	}
	for _, c := range ok {
		h, m, err := ParseClockTime(c.in)
		if err != nil || h != c.h || m != c.m {
			t.Errorf("ParseClockTime(%q) = %d:%d, %v; want %d:%d", c.in, h, m, err, c.h, c.m)
		}
	}
	bad := []string{"", "24:00", "12:60", "7", "07-30", "aa:bb", "25:10"}
	for _, in := range bad {
		if _, _, err := ParseClockTime(in); err == nil {
			t.Errorf("ParseClockTime(%q) expected error", in)
		}
	}
}

func mask(days ...time.Weekday) [7]bool {
	var r [7]bool
	for _, d := range days {
		r[d] = true
	}
	return r
}

func TestParseDays(t *testing.T) {
	cases := []struct {
		in   string
		want [7]bool
	}{
		{"", mask()},
		{"daily", mask(time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday)},
		{"diario", mask(time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday)},
		{"weekdays", mask(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday)},
		{"laborables", mask(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday)},
		{"weekend", mask(time.Saturday, time.Sunday)},
		{"finde", mask(time.Saturday, time.Sunday)},
		{"mon,wed,fri", mask(time.Monday, time.Wednesday, time.Friday)},
		{"lun,mié,vie", mask(time.Monday, time.Wednesday, time.Friday)},
		{"lun, mie , vie", mask(time.Monday, time.Wednesday, time.Friday)}, // accent-less + spaces
		{"SÁB,DOM", mask(time.Saturday, time.Sunday)},                      // case-insensitive
	}
	for _, c := range cases {
		got, err := ParseDays(c.in)
		if err != nil {
			t.Errorf("ParseDays(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseDays(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	if _, err := ParseDays("funday"); err == nil {
		t.Error("ParseDays(funday) expected error")
	}
}

func TestParseTimerDuration(t *testing.T) {
	if d, err := ParseTimerDuration("1h30m"); err != nil || d != 90*time.Minute {
		t.Errorf("ParseTimerDuration(1h30m) = %v, %v", d, err)
	}
	for _, bad := range []string{"", "0s", "-5m", "abc", "10"} {
		if _, err := ParseTimerDuration(bad); err == nil {
			t.Errorf("ParseTimerDuration(%q) expected error", bad)
		}
	}
}

func TestDaysRoundTrip(t *testing.T) {
	in := "mon,wed,sun"
	r, err := ParseDays(in)
	if err != nil {
		t.Fatal(err)
	}
	got := DaysToStrings(r)
	want := []string{"mon", "wed", "sun"} // Monday-first
	if len(got) != len(want) {
		t.Fatalf("DaysToStrings = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DaysToStrings = %v, want %v", got, want)
		}
	}
	if DaysToStrings(mask()) != nil {
		t.Error("one-shot should produce nil days")
	}
}
