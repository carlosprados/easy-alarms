// Package astro computes sunrise/sunset and moon phase from date and location,
// using only the standard library (no external dependencies).
package astro

import (
	"math"
	"time"
)

// Location is a geographic point. Lat is north-positive, Lon east-positive.
type Location struct {
	Lat float64
	Lon float64
}

// Madrid is the default location, used when none is configured.
var Madrid = Location{Lat: 40.4168, Lon: -3.7038}

const deg = math.Pi / 180

// SunTimes returns sunrise and sunset for the calendar date of t at loc,
// expressed in t's location. ok is false on polar day/night, when the sun
// never crosses the horizon and the times would be meaningless.
//
// Implements the NOAA sunrise equation; accuracy is ~1 minute, plenty for a
// clock decoration.
func SunTimes(loc Location, t time.Time) (sunrise, sunset time.Time, ok bool) {
	dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	jdate := julianDate(dayStart)

	lw := -loc.Lon // west longitude, positive
	n := math.Round(jdate - 2451545.0 - 0.0009 + lw/360.0)
	jStar := 2451545.0 + 0.0009 + lw/360.0 + n

	m := math.Mod(357.5291+0.98560028*(jStar-2451545.0), 360)
	c := 1.9148*math.Sin(m*deg) + 0.0200*math.Sin(2*m*deg) + 0.0003*math.Sin(3*m*deg)
	lambda := math.Mod(m+c+180+102.9372, 360)

	jTransit := jStar + 0.0053*math.Sin(m*deg) - 0.0069*math.Sin(2*lambda*deg)

	sinDelta := math.Sin(lambda*deg) * math.Sin(23.4397*deg)
	delta := math.Asin(sinDelta)

	cosOmega := (math.Sin(-0.833*deg) - math.Sin(loc.Lat*deg)*math.Sin(delta)) /
		(math.Cos(loc.Lat*deg) * math.Cos(delta))
	if cosOmega > 1 || cosOmega < -1 {
		return time.Time{}, time.Time{}, false
	}
	omega := math.Acos(cosOmega) / deg

	jRise := jTransit - omega/360
	jSet := jTransit + omega/360
	return fromJulian(jRise).In(t.Location()), fromJulian(jSet).In(t.Location()), true
}

// julianDate is the Julian Date of the instant t (UTC).
func julianDate(t time.Time) float64 {
	return float64(t.UTC().Unix())/86400.0 + 2440587.5
}

func fromJulian(jd float64) time.Time {
	return time.Unix(int64(math.Round((jd-2440587.5)*86400.0)), 0).UTC()
}
