package control

import (
	"errors"
	"time"
)

// Sentinel errors the Backend returns; the server maps them to HTTP codes.
var (
	ErrNotFound   = errors.New("alarm not found")                         // -> 404
	ErrNotRinging = errors.New("alarm is not ringing")                    // -> 409
	ErrWrongKind  = errors.New("operation not valid for this alarm kind") // -> 409
	ErrBadState   = errors.New("invalid state transition")                // -> 409
)

// TimerOp is a timer lifecycle action.
type TimerOp string

const (
	OpStart  TimerOp = "start"
	OpPause  TimerOp = "pause"
	OpResume TimerOp = "resume"
	OpStop   TimerOp = "stop"
)

// ClockSpec is a validated clock-alarm creation request.
type ClockSpec struct {
	Label  string
	Hour   int
	Minute int
	Repeat [7]bool
	Sound  string
}

// TimerSpec is a validated timer creation request.
type TimerSpec struct {
	Label    string
	Duration time.Duration
	Sound    string
	Start    bool
}

// Patch is a validated partial update. Only non-nil fields are applied.
type Patch struct {
	Label    *string
	Hour     *int
	Minute   *int
	Repeat   *[7]bool
	Duration *time.Duration
	Sound    *string
}

// Backend is the app-side surface the control server drives. Implementations
// must be safe to call from any goroutine (the GUI implementation hops each
// call onto the Fyne main thread).
type Backend interface {
	Snapshot() ([]AlarmState, []string) // all alarms + IDs currently ringing
	Get(id string) (AlarmState, error)
	CreateAlarm(spec ClockSpec) (AlarmState, error)
	CreateTimer(spec TimerSpec) (AlarmState, error)
	Update(id string, p Patch) (AlarmState, error)
	Delete(id string) error
	SetEnabled(id string, on bool) (AlarmState, error) // clock alarms only
	TimerOp(id string, op TimerOp) (AlarmState, error) // timers only
	SnoozeRinging(id string, d time.Duration) error
	DismissRinging(id string) error
	Settings() SettingsDTO
}
