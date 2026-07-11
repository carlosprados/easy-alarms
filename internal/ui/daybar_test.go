package ui

import "testing"

func TestInDaylight(t *testing.T) {
	// Normal day: sunrise 06:00 (.25), sunset 21:00 (.875).
	rise, set := 0.25, 0.875
	if !inDaylight(0.5, rise, set) { // noon
		t.Error("noon should be daylight")
	}
	if inDaylight(0.1, rise, set) { // before dawn
		t.Error("pre-dawn should be night")
	}
	if inDaylight(0.95, rise, set) { // after dusk
		t.Error("post-dusk should be night")
	}

	// Sunset past midnight (high latitude): sunrise 05:00 (.208),
	// sunset 01:30 next day (.0625).
	rise, set = 0.208, 0.0625
	if !inDaylight(0.5, rise, set) { // midday: lit
		t.Error("midday should be daylight when the day wraps midnight")
	}
	if !inDaylight(0.99, rise, set) { // just before midnight: still lit
		t.Error("late night should be daylight when sunset is past midnight")
	}
	if inDaylight(0.15, rise, set) { // between sunset and sunrise: dark
		t.Error("the short night should be dark")
	}
}

func TestCircularDelta(t *testing.T) {
	cases := []struct{ frac, center, want float64 }{
		{0.30, 0.25, 0.05},
		{0.20, 0.25, -0.05},
		{0.01, 0.99, 0.02},  // wraps forward across midnight
		{0.99, 0.01, -0.02}, // wraps backward across midnight
	}
	for _, c := range cases {
		if got := circularDelta(c.frac, c.center); got < c.want-1e-9 || got > c.want+1e-9 {
			t.Errorf("circularDelta(%g,%g)=%g want %g", c.frac, c.center, got, c.want)
		}
	}
}
