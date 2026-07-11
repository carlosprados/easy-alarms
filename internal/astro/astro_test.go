package astro

import (
	"math"
	"testing"
	"time"
)

func TestSunTimesMadridSummer(t *testing.T) {
	cest := time.FixedZone("CEST", 2*3600)
	day := time.Date(2024, 6, 21, 12, 0, 0, 0, cest)
	rise, set, ok := SunTimes(Madrid, day)
	if !ok {
		t.Fatal("expected the sun to rise on the summer solstice in Madrid")
	}
	if !rise.Before(set) {
		t.Fatalf("sunrise %v not before sunset %v", rise, set)
	}
	// Around the solstice Madrid has ~15h of daylight.
	daylight := set.Sub(rise)
	if daylight < 14*time.Hour || daylight > 15*time.Hour+30*time.Minute {
		t.Errorf("implausible daylight length: %v", daylight)
	}
	if rise.Hour() < 5 || rise.Hour() > 8 {
		t.Errorf("sunrise hour out of range: %v", rise)
	}
	if set.Hour() < 20 || set.Hour() > 22 {
		t.Errorf("sunset hour out of range: %v", set)
	}
}

func TestSunTimesPolarDay(t *testing.T) {
	// Far north in midsummer: the sun never sets.
	arctic := Location{Lat: 78, Lon: 15}
	day := time.Date(2024, 6, 21, 12, 0, 0, 0, time.UTC)
	if _, _, ok := SunTimes(arctic, day); ok {
		t.Error("expected ok=false during polar day")
	}
}

func TestMoonPhaseKnownDates(t *testing.T) {
	// Reference new moon.
	newMoon := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)
	if got := MoonPhase(newMoon); got.Illumination > 0.02 {
		t.Errorf("new moon illumination = %.3f, want ~0", got.Illumination)
	}
	// Half a lunation later → full moon.
	full := newMoon.Add(time.Duration(synodicMonth / 2 * float64(24*time.Hour)))
	if got := MoonPhase(full); got.Illumination < 0.98 {
		t.Errorf("full moon illumination = %.3f, want ~1", got.Illumination)
	}
	// A quarter lunation → first quarter, ~half lit.
	quarter := newMoon.Add(time.Duration(synodicMonth / 4 * float64(24*time.Hour)))
	if got := MoonPhase(quarter); math.Abs(got.Illumination-0.5) > 0.05 {
		t.Errorf("first quarter illumination = %.3f, want ~0.5", got.Illumination)
	}
}
