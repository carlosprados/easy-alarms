package control

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"easy-alarms/internal/alarm"
)

// fakeBackend records calls and returns canned results, so handlers can be
// tested without Fyne.
type fakeBackend struct {
	alarms       map[string]AlarmState
	ringing      []string
	lastClock    ClockSpec
	lastTimer    TimerSpec
	lastPatch    Patch
	snoozeErr    error
	dismissErr   error
	timerOpErr   error
	lastTimerOp  TimerOp
	lastSnoozeID string
	lastSnoozeD  time.Duration
}

func newFake() *fakeBackend {
	return &fakeBackend{alarms: map[string]AlarmState{
		"clk": {Alarm: alarm.Alarm{ID: "clk", Kind: alarm.KindClock, Enabled: true, Hour: 7, Minute: 30}},
		"tmr": {Alarm: alarm.Alarm{ID: "tmr", Kind: alarm.KindTimer, Enabled: true, Duration: 10 * time.Minute}},
	}}
}

func (f *fakeBackend) Snapshot() ([]AlarmState, []string) {
	out := make([]AlarmState, 0, len(f.alarms))
	for _, s := range f.alarms {
		out = append(out, s)
	}
	return out, f.ringing
}
func (f *fakeBackend) Get(id string) (AlarmState, error) {
	s, ok := f.alarms[id]
	if !ok {
		return AlarmState{}, ErrNotFound
	}
	return s, nil
}
func (f *fakeBackend) CreateAlarm(spec ClockSpec) (AlarmState, error) {
	f.lastClock = spec
	return AlarmState{Alarm: alarm.Alarm{ID: "new", Kind: alarm.KindClock, Hour: spec.Hour, Minute: spec.Minute, Repeat: spec.Repeat, Enabled: true}}, nil
}
func (f *fakeBackend) CreateTimer(spec TimerSpec) (AlarmState, error) {
	f.lastTimer = spec
	return AlarmState{Alarm: alarm.Alarm{ID: "new", Kind: alarm.KindTimer, Duration: spec.Duration, Enabled: true}}, nil
}
func (f *fakeBackend) Update(id string, p Patch) (AlarmState, error) {
	s, ok := f.alarms[id]
	if !ok {
		return AlarmState{}, ErrNotFound
	}
	f.lastPatch = p
	return s, nil
}
func (f *fakeBackend) Delete(id string) error {
	if _, ok := f.alarms[id]; !ok {
		return ErrNotFound
	}
	delete(f.alarms, id)
	return nil
}
func (f *fakeBackend) SetEnabled(id string, on bool) (AlarmState, error) {
	s, ok := f.alarms[id]
	if !ok {
		return AlarmState{}, ErrNotFound
	}
	if s.Alarm.Kind != alarm.KindClock {
		return AlarmState{}, ErrWrongKind
	}
	return s, nil
}
func (f *fakeBackend) TimerOp(id string, op TimerOp) (AlarmState, error) {
	f.lastTimerOp = op
	if f.timerOpErr != nil {
		return AlarmState{}, f.timerOpErr
	}
	s, ok := f.alarms[id]
	if !ok {
		return AlarmState{}, ErrNotFound
	}
	return s, nil
}
func (f *fakeBackend) SnoozeRinging(id string, d time.Duration) error {
	f.lastSnoozeID, f.lastSnoozeD = id, d
	return f.snoozeErr
}
func (f *fakeBackend) DismissRinging(id string) error { return f.dismissErr }
func (f *fakeBackend) Settings() SettingsDTO          { return SettingsDTO{Lat: 40, Lon: -3} }

func do(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	srv.routes().ServeHTTP(w, r)
	return w
}

func TestCreateAlarmHappy(t *testing.T) {
	f := newFake()
	srv := NewServer(f, "test")
	w := do(t, srv, "POST", "/alarms", `{"at":"07:30","days":"weekdays","label":"Work"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("code = %d, body %s", w.Code, w.Body)
	}
	if f.lastClock.Hour != 7 || f.lastClock.Minute != 30 {
		t.Errorf("spec = %+v", f.lastClock)
	}
	if !f.lastClock.Repeat[time.Monday] || f.lastClock.Repeat[time.Sunday] {
		t.Errorf("weekdays not parsed: %v", f.lastClock.Repeat)
	}
}

func TestCreateAlarmBadTime(t *testing.T) {
	srv := NewServer(newFake(), "test")
	w := do(t, srv, "POST", "/alarms", `{"at":"25:00"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, body %s", w.Code, w.Body)
	}
}

func TestGetUnknown(t *testing.T) {
	srv := NewServer(newFake(), "test")
	w := do(t, srv, "GET", "/alarms/nope", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("code = %d", w.Code)
	}
}

func TestDismissNotRinging(t *testing.T) {
	f := newFake()
	f.dismissErr = ErrNotRinging
	srv := NewServer(f, "test")
	w := do(t, srv, "POST", "/alarms/clk/dismiss", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("code = %d, body %s", w.Code, w.Body)
	}
}

func TestTimerOpBadState(t *testing.T) {
	f := newFake()
	f.timerOpErr = ErrBadState
	srv := NewServer(f, "test")
	w := do(t, srv, "POST", "/timers/tmr/pause", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("code = %d", w.Code)
	}
	if f.lastTimerOp != OpPause {
		t.Errorf("op = %q", f.lastTimerOp)
	}
}

func TestTimerOpUnknown(t *testing.T) {
	srv := NewServer(newFake(), "test")
	w := do(t, srv, "POST", "/timers/tmr/frobnicate", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", w.Code)
	}
}

func TestUpdatePartial(t *testing.T) {
	f := newFake()
	srv := NewServer(f, "test")
	w := do(t, srv, "PATCH", "/alarms/clk", `{"label":"Renamed"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, body %s", w.Code, w.Body)
	}
	if f.lastPatch.Label == nil || *f.lastPatch.Label != "Renamed" {
		t.Errorf("label patch missing: %+v", f.lastPatch)
	}
	if f.lastPatch.Hour != nil || f.lastPatch.Repeat != nil {
		t.Errorf("only label should be set: %+v", f.lastPatch)
	}
}

func TestSnoozeDefaultAndCustom(t *testing.T) {
	f := newFake()
	srv := NewServer(f, "test")
	// default 5m (empty body)
	if w := do(t, srv, "POST", "/alarms/clk/snooze", ""); w.Code != http.StatusNoContent {
		t.Fatalf("default snooze code = %d", w.Code)
	}
	if f.lastSnoozeD != 5*time.Minute {
		t.Errorf("default snooze = %v", f.lastSnoozeD)
	}
	// custom
	if w := do(t, srv, "POST", "/alarms/clk/snooze", `{"for":"1m"}`); w.Code != http.StatusNoContent {
		t.Fatalf("custom snooze code = %d", w.Code)
	}
	if f.lastSnoozeD != time.Minute {
		t.Errorf("custom snooze = %v", f.lastSnoozeD)
	}
}

func TestStatus(t *testing.T) {
	f := newFake()
	next := time.Now().Add(time.Hour)
	f.alarms["clk"] = AlarmState{Alarm: f.alarms["clk"].Alarm, Next: next}
	f.ringing = []string{"tmr"}
	srv := NewServer(f, "v1.2.3")
	w := do(t, srv, "GET", "/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d", w.Code)
	}
	var got StatusDTO
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1.2.3" {
		t.Errorf("version = %q", got.Version)
	}
	if len(got.Alarms) != 2 {
		t.Errorf("alarms = %d", len(got.Alarms))
	}
	if got.Next == nil || got.Next.ID != "clk" {
		t.Errorf("next = %+v", got.Next)
	}
	if len(got.Ringing) != 1 || got.Ringing[0] != "tmr" {
		t.Errorf("ringing = %v", got.Ringing)
	}
}

func TestListKindFilter(t *testing.T) {
	srv := NewServer(newFake(), "test")
	w := do(t, srv, "GET", "/alarms?kind=timer", "")
	var got []AlarmDTO
	json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0].Kind != "timer" {
		t.Errorf("filtered list = %+v", got)
	}
}
