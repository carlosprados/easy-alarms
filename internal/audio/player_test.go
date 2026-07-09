package audio

import "testing"

// TestBeepingToneEnvelope verifies the built-in tone is intermittent
// (beep … silence … beep) rather than a continuous drone.
func TestBeepingToneEnvelope(t *testing.T) {
	b := &beepingTone{freq: 880, on: 100, off: 150}
	buf := make([][2]float64, b.on+b.off)
	n, ok := b.Stream(buf)
	if !ok || n != len(buf) {
		t.Fatalf("Stream returned n=%d ok=%v", n, ok)
	}

	// During the "on" window there must be audible signal...
	var onEnergy float64
	for i := range b.on {
		onEnergy += buf[i][0] * buf[i][0]
	}
	if onEnergy == 0 {
		t.Fatal("on window is silent")
	}
	// ...and during the "off" window pure silence.
	for i := b.on; i < b.on+b.off; i++ {
		if buf[i][0] != 0 || buf[i][1] != 0 {
			t.Fatalf("off window not silent at sample %d: %v", i, buf[i])
		}
	}
}

// constant emits a fixed sample value forever, to probe gain wrappers.
type constant float64

func (c constant) Stream(samples [][2]float64) (int, bool) {
	for i := range samples {
		samples[i][0], samples[i][1] = float64(c), float64(c)
	}
	return len(samples), true
}

func (c constant) Err() error { return nil }

// TestFadeInRamp verifies the ring fade-in starts silent, grows monotonically
// and leaves the signal untouched once the ramp is over.
func TestFadeInRamp(t *testing.T) {
	f := &fadeIn{s: constant(1), ramp: 100}
	buf := make([][2]float64, 150)
	if n, ok := f.Stream(buf); !ok || n != len(buf) {
		t.Fatalf("Stream returned n=%d ok=%v", n, ok)
	}
	if buf[0][0] != 0 {
		t.Errorf("fade should start silent, got %v", buf[0][0])
	}
	for i := 1; i < 100; i++ {
		if buf[i][0] < buf[i-1][0] {
			t.Fatalf("gain not monotonic at sample %d: %v < %v", i, buf[i][0], buf[i-1][0])
		}
	}
	for i := 100; i < 150; i++ {
		if buf[i][0] != 1 {
			t.Fatalf("sample %d after ramp should be full volume, got %v", i, buf[i][0])
		}
	}
}

// TestBeepingToneLoops checks the pattern repeats and never reports EOF, so
// the alarm keeps ringing until stopped.
func TestBeepingToneLoops(t *testing.T) {
	b := &beepingTone{freq: 880, on: 10, off: 10}
	buf := make([][2]float64, 5*(b.on+b.off)) // five full cycles
	if _, ok := b.Stream(buf); !ok {
		t.Fatal("stream ended early")
	}
	period := b.on + b.off
	for cycle := range 5 {
		base := cycle * period
		if buf[base+b.on][0] != 0 { // first silent sample of each cycle
			t.Fatalf("cycle %d: expected silence at off boundary", cycle)
		}
	}
}
