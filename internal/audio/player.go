package audio

import (
	"io"
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

const (
	sampleRate = beep.SampleRate(44100)
	ringFadeIn = 10 * time.Second // ramp so early-morning alarms don't blast
)

type Player struct {
	initOnce sync.Once

	mu  sync.Mutex
	src io.Closer // file backing the currently playing stream, nil for the tone
}

func NewPlayer() *Player { return &Player{} }

func (p *Player) init() {
	p.initOnce.Do(func() {
		speaker.Init(sampleRate, sampleRate.N(100*time.Millisecond))
	})
}

// Play loops the given audio file until Stop is called, fading the volume in
// gradually. It is used for ringing alarms, so it must always make noise: an
// empty or undecodable path falls back to the built-in beep rather than
// failing.
func (p *Player) Play(path string) {
	p.init()
	s, src, err := open(path)
	if err != nil {
		s, src = fallbackTone(), nil
	}
	p.swap(&fadeIn{s: s, ramp: sampleRate.N(ringFadeIn)}, src)
}

// Preview is like Play but reports decode errors instead of silently falling
// back, so the UI can tell the user a file's format is unsupported. An empty
// path previews the built-in tone.
func (p *Player) Preview(path string) error {
	p.init()
	s, src, err := open(path)
	if err != nil {
		return err
	}
	p.swap(s, src)
	return nil
}

func (p *Player) Stop() {
	p.swap(nil, nil)
}

// swap silences the speaker, closes the file backing the previous stream and
// starts the new one (nil s just stops). Clear runs before Close so the mixer
// never pulls from a closed file.
func (p *Player) swap(s beep.Streamer, src io.Closer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	speaker.Clear()
	if p.src != nil {
		p.src.Close()
	}
	p.src = src
	if s != nil {
		speaker.Play(s)
	}
}

func open(path string) (beep.Streamer, io.Closer, error) {
	if path == "" {
		return fallbackTone(), nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	loop, err := beep.Loop2(s)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	if format.SampleRate != sampleRate {
		return beep.Resample(4, format.SampleRate, sampleRate, loop), f, nil
	}
	return loop, f, nil
}

// fadeIn ramps the wrapped streamer from silence to full volume over ramp
// samples. The quadratic curve sounds smoother to the ear than a linear one.
type fadeIn struct {
	s         beep.Streamer
	pos, ramp int
}

func (f *fadeIn) Stream(samples [][2]float64) (int, bool) {
	n, ok := f.s.Stream(samples)
	for i := 0; i < n && f.pos < f.ramp; i++ {
		g := float64(f.pos) / float64(f.ramp)
		g *= g
		samples[i][0] *= g
		samples[i][1] *= g
		f.pos++
	}
	return n, ok
}

func (f *fadeIn) Err() error { return f.s.Err() }

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
	freq    float64
	on, off int
	pos     int     // position within the current on+off cycle
	phase   float64 // sine phase in radians, reset between beeps to avoid clicks
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
