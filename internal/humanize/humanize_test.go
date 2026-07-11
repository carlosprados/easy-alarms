package humanize

import (
	"testing"
	"time"
)

func TestCoarse(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "<1m"},
		{90 * time.Second, "1m"},
		{12 * time.Minute, "12m"},
		{2*time.Hour + 5*time.Minute, "2h 05m"},
		{25*time.Hour + 3*time.Hour, "1d 4h"},
		{-time.Minute, "<1m"}, // never negative
	}
	for _, c := range cases {
		if got := Coarse(c.d); got != c.want {
			t.Errorf("Coarse(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestLong(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Second, "45s"},
		{90 * time.Second, "1m 30s"},
		{2*time.Hour + 5*time.Minute, "2h 05m"},
		{25 * time.Hour, "1d 1h"},
		{-time.Second, "0s"}, // never negative
	}
	for _, c := range cases {
		if got := Long(c.d); got != c.want {
			t.Errorf("Long(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestCompact(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Second, "45s"},
		{10 * time.Minute, "10m"},
		{90 * time.Minute, "1h30m"},
		{time.Hour, "1h"},
		{0, "0s"},
	}
	for _, c := range cases {
		if got := Compact(c.d); got != c.want {
			t.Errorf("Compact(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
