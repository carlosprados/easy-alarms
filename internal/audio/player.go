package audio

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

const sampleRate = beep.SampleRate(44100)

type Player struct {
	initOnce sync.Once
}

func NewPlayer() *Player { return &Player{} }

func (p *Player) init() {
	p.initOnce.Do(func() {
		speaker.Init(sampleRate, sampleRate.N(100*time.Millisecond))
	})
}

// Play loops the given audio file until Stop is called. It is used for
// ringing alarms, so it must always make noise: an empty or undecodable path
// falls back to the built-in beep rather than failing.
func (p *Player) Play(path string) {
	p.init()
	s, err := open(path)
	if err != nil {
		s = fallbackTone()
	}
	speaker.Clear()
	speaker.Play(s)
}

// Preview is like Play but reports decode errors instead of silently falling
// back, so the UI can tell the user a file's format is unsupported. An empty
// path previews the built-in tone.
func (p *Player) Preview(path string) error {
	p.init()
	s, err := open(path)
	if err != nil {
		return err
	}
	speaker.Clear()
	speaker.Play(s)
	return nil
}

func (p *Player) Stop() {
	speaker.Clear()
}

func open(path string) (beep.Streamer, error) {
	if path == "" {
		return fallbackTone(), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var (
		s      beep.StreamSeekCloser
		format beep.Format
	)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		s, format, err = mp3.Decode(f)
	case ".wav":
		s, format, err = wav.Decode(f)
	case ".flac":
		s, format, err = flac.Decode(f)
	default: // .ogg, .oga and friends
		s, format, err = vorbis.Decode(f)
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	loop, err := beep.Loop2(s)
	if err != nil {
		f.Close()
		return nil, err
	}
	if format.SampleRate != sampleRate {
		return beep.Resample(4, format.SampleRate, sampleRate, loop), nil
	}
	return loop, nil
}

func fallbackTone() beep.Streamer {
	return &beepingTone{
		freq: 880,
		on:   sampleRate.N(160 * time.Millisecond),
		off:  sampleRate.N(240 * time.Millisecond),
	}
}

// beepingTone is an endless "beep … beep … beep" pattern: a sine tone gated
// on for `on` samples then silent for `off` samples, repeating. Much less
// grating than a continuous tone for the built-in alarm sound.
type beepingTone struct {
	freq     float64
	on, off  int
	pos      int     // position within the current on+off cycle
	phase    float64 // sine phase in radians, reset between beeps to avoid clicks
}

func (b *beepingTone) Stream(samples [][2]float64) (int, bool) {
	const amp = 0.6
	step := 2 * math.Pi * b.freq / float64(sampleRate)
	period := b.on + b.off
	for i := range samples {
		var v float64
		if b.pos < b.on {
			v = amp * math.Sin(b.phase)
			b.phase += step
		} else {
			b.phase = 0
		}
		samples[i][0] = v
		samples[i][1] = v
		if b.pos++; b.pos >= period {
			b.pos = 0
		}
	}
	return len(samples), true
}

func (b *beepingTone) Err() error { return nil }
