package ui

import (
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"easy-alarms/internal/astro"
)

// dayBar is a full-width bar that paints the sky across a 24h day
// (night → dawn → day → dusk → night, anchored to the real sunrise/sunset)
// with a bright cursor marking the current moment.
type dayBar struct {
	widget.BaseWidget
	loc             astro.Location
	sunrise, sunset time.Time
	haveSun         bool
	frac            float64   // 0..1 through the day
	day             int       // YearDay of the last sun computation
	size            fyne.Size // last laid-out size

	sky    *canvas.Raster
	cursor *canvas.Rectangle
}

func newDayBar(loc astro.Location, now time.Time) *dayBar {
	b := &dayBar{loc: loc, day: now.YearDay(), frac: dayFraction(now)}
	b.sunrise, b.sunset, b.haveSun = astro.SunTimes(loc, now)
	b.sky = canvas.NewRasterWithPixels(b.skyPixel)
	b.cursor = canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 235})
	b.ExtendBaseWidget(b)
	return b
}

// setLocation switches the location and repaints with the new sun times.
func (b *dayBar) setLocation(loc astro.Location, now time.Time) {
	b.loc = loc
	b.day = now.YearDay()
	b.sunrise, b.sunset, b.haveSun = astro.SunTimes(loc, now)
	b.Refresh()
}

// tick advances the cursor every second and recomputes the sun once a day.
func (b *dayBar) tick(now time.Time) {
	b.frac = dayFraction(now)
	if b.size.Width > 0 {
		b.cursor.Move(fyne.NewPos(float32(b.frac)*b.size.Width-1, 0))
		canvas.Refresh(b.cursor)
	}
	if now.YearDay() != b.day {
		b.day = now.YearDay()
		b.sunrise, b.sunset, b.haveSun = astro.SunTimes(b.loc, now)
		b.Refresh() // repaint the sky with the new sun times
	}
}

func (b *dayBar) skyPixel(x, _, w, _ int) color.Color {
	if w <= 0 {
		return skyNight
	}
	return b.skyColor(float64(x) / float64(w))
}

var (
	skyNight = color.NRGBA{R: 44, G: 52, B: 92, A: 255}
	skyDay   = color.NRGBA{R: 96, G: 160, B: 230, A: 255}
	skyTwil  = color.NRGBA{R: 235, G: 138, B: 72, A: 255}
)

// skyColor maps a fraction of the day (0=00:00, 1=24:00) to a sky colour,
// with warm twilight bands around sunrise and sunset. All comparisons are
// circular, so it stays correct when sunset falls past midnight (high
// latitudes) and sunrise/sunset wrap the day boundary.
func (b *dayBar) skyColor(frac float64) color.NRGBA {
	if !b.haveSun {
		return skyNight
	}
	rise, set := dayFraction(b.sunrise), dayFraction(b.sunset)
	const tw = 40.0 / 1440.0 // ~40 min twilight band
	switch {
	case math.Abs(circularDelta(frac, rise)) < tw: // around sunrise
		if s := circularDelta(frac, rise); s < 0 {
			return lerp(skyNight, skyTwil, (s+tw)/tw)
		} else {
			return lerp(skyTwil, skyDay, s/tw)
		}
	case math.Abs(circularDelta(frac, set)) < tw: // around sunset
		if s := circularDelta(frac, set); s < 0 {
			return lerp(skyDay, skyTwil, (s+tw)/tw)
		} else {
			return lerp(skyTwil, skyNight, s/tw)
		}
	case inDaylight(frac, rise, set):
		return skyDay
	default:
		return skyNight
	}
}

// circularDelta is frac-center wrapped to [-0.5, 0.5] (signed distance around
// the 24h circle).
func circularDelta(frac, center float64) float64 {
	d := frac - center
	if d > 0.5 {
		d -= 1
	} else if d < -0.5 {
		d += 1
	}
	return d
}

// inDaylight reports whether frac lies on the daylight arc from rise to set,
// handling the case where the arc wraps past midnight (set < rise).
func inDaylight(frac, rise, set float64) bool {
	if rise <= set {
		return frac >= rise && frac <= set
	}
	return frac >= rise || frac <= set
}

func (b *dayBar) CreateRenderer() fyne.WidgetRenderer {
	return &dayBarRenderer{b: b, objects: []fyne.CanvasObject{b.sky, b.cursor}}
}

type dayBarRenderer struct {
	b       *dayBar
	objects []fyne.CanvasObject
}

func (r *dayBarRenderer) Layout(size fyne.Size) {
	r.b.size = size
	r.b.sky.Resize(size)
	r.b.cursor.Resize(fyne.NewSize(2, size.Height))
	r.b.cursor.Move(fyne.NewPos(float32(r.b.frac)*size.Width-1, 0))
}
func (r *dayBarRenderer) MinSize() fyne.Size           { return fyne.NewSize(0, 16) }
func (r *dayBarRenderer) Refresh()                     { r.b.sky.Refresh(); r.Layout(r.b.Size()) }
func (r *dayBarRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *dayBarRenderer) Destroy()                     {}

// dayFraction is t's position through its local day, 0..1.
func dayFraction(t time.Time) float64 {
	return float64(t.Hour()*3600+t.Minute()*60+t.Second()) / 86400.0
}

func lerp(a, c color.NRGBA, x float64) color.NRGBA {
	if x < 0 {
		x = 0
	} else if x > 1 {
		x = 1
	}
	mix := func(p, q uint8) uint8 { return uint8(float64(p) + (float64(q)-float64(p))*x) }
	return color.NRGBA{R: mix(a.R, c.R), G: mix(a.G, c.G), B: mix(a.B, c.B), A: 255}
}
