package astro

import (
	"math"
	"time"
)

const (
	synodicMonth = 29.53058867 // mean length of a lunation, days
	newMoonRef   = 2451550.1   // Julian Date of the new moon on 2000-01-06
)

// Moon describes the moon's phase at an instant.
type Moon struct {
	Phase        float64 // 0=new, 0.25=first quarter, 0.5=full, 0.75=last quarter
	Illumination float64 // fraction of the disc lit, 0..1
	Name         string  // Spanish phase name
	Emoji        string
}

// MoonPhase computes the moon phase for t. It depends only on the date/time,
// not on location.
func MoonPhase(t time.Time) Moon {
	age := math.Mod(julianDate(t)-newMoonRef, synodicMonth)
	if age < 0 {
		age += synodicMonth
	}
	phase := age / synodicMonth
	name, emoji := moonName(phase)
	return Moon{
		Phase:        phase,
		Illumination: (1 - math.Cos(2*math.Pi*phase)) / 2,
		Name:         name,
		Emoji:        emoji,
	}
}

func moonName(phase float64) (name, emoji string) {
	switch {
	case phase < 0.03 || phase >= 0.97:
		return "Luna nueva", "🌑"
	case phase < 0.22:
		return "Creciente", "🌒"
	case phase < 0.28:
		return "Cuarto creciente", "🌓"
	case phase < 0.47:
		return "Gibosa creciente", "🌔"
	case phase < 0.53:
		return "Luna llena", "🌕"
	case phase < 0.72:
		return "Gibosa menguante", "🌖"
	case phase < 0.78:
		return "Cuarto menguante", "🌗"
	default:
		return "Menguante", "🌘"
	}
}
