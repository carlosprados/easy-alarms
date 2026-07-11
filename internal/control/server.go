package control

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"
)

// Server exposes a Backend over HTTP/JSON on a Unix socket.
type Server struct {
	backend Backend
	version string
	http    *http.Server
}

// NewServer builds a control server for the given backend.
func NewServer(backend Backend, version string) *Server {
	s := &Server{backend: backend, version: version}
	s.http = &http.Server{
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// Serve accepts connections on ln until Close is called. It returns nil on a
// clean shutdown.
func (s *Server) Serve(ln net.Listener) error {
	if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Close stops the server, closing the listener.
func (s *Server) Close() error {
	if s.http == nil {
		return nil
	}
	return s.http.Close()
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("GET /settings", s.handleSettings)
	mux.HandleFunc("GET /alarms", s.handleList)
	mux.HandleFunc("GET /alarms/{id}", s.handleGet)
	mux.HandleFunc("POST /alarms", s.handleCreateAlarm)
	mux.HandleFunc("PATCH /alarms/{id}", s.handleUpdate)
	mux.HandleFunc("DELETE /alarms/{id}", s.handleDelete)
	mux.HandleFunc("POST /alarms/{id}/enable", s.handleEnable(true))
	mux.HandleFunc("POST /alarms/{id}/disable", s.handleEnable(false))
	mux.HandleFunc("POST /alarms/{id}/snooze", s.handleSnooze)
	mux.HandleFunc("POST /alarms/{id}/dismiss", s.handleDismiss)
	mux.HandleFunc("POST /timers", s.handleCreateTimer)
	mux.HandleFunc("POST /timers/{id}/{op}", s.handleTimerOp)
	return mux
}

// --- handlers ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	states, ringing := s.backend.Snapshot()
	status := StatusDTO{
		Version: s.version,
		Now:     now,
		Ringing: ringing,
		Alarms:  make([]AlarmDTO, 0, len(states)),
	}
	var next *AlarmDTO
	var nextAt time.Time
	for _, st := range states {
		d := ToDTO(st, now)
		status.Alarms = append(status.Alarms, d)
		if !st.Next.IsZero() && (next == nil || st.Next.Before(nextAt)) {
			cp := d
			next, nextAt = &cp, st.Next
		}
	}
	status.Next = next
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.backend.Settings())
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	kind := r.URL.Query().Get("kind")
	states, _ := s.backend.Snapshot()
	out := make([]AlarmDTO, 0, len(states))
	for _, st := range states {
		if kind != "" && string(st.Alarm.Kind) != kind {
			continue
		}
		out = append(out, ToDTO(st, now))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	st, err := s.backend.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ToDTO(st, time.Now()))
}

func (s *Server) handleCreateAlarm(w http.ResponseWriter, r *http.Request) {
	var req CreateAlarmRequest
	if !decode(w, r, &req) {
		return
	}
	hour, minute, err := ParseClockTime(req.At)
	if err != nil {
		writeStatus(w, http.StatusBadRequest, err)
		return
	}
	repeat, err := ParseDays(req.Days)
	if err != nil {
		writeStatus(w, http.StatusBadRequest, err)
		return
	}
	st, err := s.backend.CreateAlarm(ClockSpec{
		Label: req.Label, Hour: hour, Minute: minute, Repeat: repeat, Sound: req.Sound,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ToDTO(st, time.Now()))
}

func (s *Server) handleCreateTimer(w http.ResponseWriter, r *http.Request) {
	var req CreateTimerRequest
	if !decode(w, r, &req) {
		return
	}
	dur, err := ParseTimerDuration(req.Duration)
	if err != nil {
		writeStatus(w, http.StatusBadRequest, err)
		return
	}
	start := req.Start == nil || *req.Start
	st, err := s.backend.CreateTimer(TimerSpec{
		Label: req.Label, Duration: dur, Sound: req.Sound, Start: start,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ToDTO(st, time.Now()))
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var req UpdateAlarmRequest
	if !decode(w, r, &req) {
		return
	}
	var p Patch
	p.Label = req.Label
	p.Sound = req.Sound
	if req.At != nil {
		hour, minute, err := ParseClockTime(*req.At)
		if err != nil {
			writeStatus(w, http.StatusBadRequest, err)
			return
		}
		p.Hour, p.Minute = &hour, &minute
	}
	if req.Days != nil {
		repeat, err := ParseDays(*req.Days)
		if err != nil {
			writeStatus(w, http.StatusBadRequest, err)
			return
		}
		p.Repeat = &repeat
	}
	if req.Duration != nil {
		dur, err := ParseTimerDuration(*req.Duration)
		if err != nil {
			writeStatus(w, http.StatusBadRequest, err)
			return
		}
		p.Duration = &dur
	}
	st, err := s.backend.Update(r.PathValue("id"), p)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ToDTO(st, time.Now()))
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.backend.Delete(r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleEnable(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := s.backend.SetEnabled(r.PathValue("id"), on)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ToDTO(st, time.Now()))
	}
}

func (s *Server) handleTimerOp(w http.ResponseWriter, r *http.Request) {
	op := TimerOp(r.PathValue("op"))
	switch op {
	case OpStart, OpPause, OpResume, OpStop:
	default:
		writeStatus(w, http.StatusBadRequest, errors.New("unknown timer op (start|pause|resume|stop)"))
		return
	}
	st, err := s.backend.TimerOp(r.PathValue("id"), op)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ToDTO(st, time.Now()))
}

func (s *Server) handleSnooze(w http.ResponseWriter, r *http.Request) {
	var req SnoozeRequest
	if r.ContentLength != 0 && !decode(w, r, &req) {
		return
	}
	d := 5 * time.Minute
	if req.For != "" {
		parsed, err := ParseTimerDuration(req.For)
		if err != nil {
			writeStatus(w, http.StatusBadRequest, err)
			return
		}
		d = parsed
	}
	if err := s.backend.SnoozeRinging(r.PathValue("id"), d); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDismiss(w http.ResponseWriter, r *http.Request) {
	if err := s.backend.DismissRinging(r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeStatus(w, http.StatusBadRequest, errors.New("invalid JSON body: "+err.Error()))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeStatus(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, ErrorResponse{Error: err.Error()})
}

// writeErr maps a Backend sentinel error to its HTTP status.
func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeStatus(w, http.StatusNotFound, err)
	case errors.Is(err, ErrNotRinging), errors.Is(err, ErrWrongKind), errors.Is(err, ErrBadState):
		writeStatus(w, http.StatusConflict, err)
	default:
		writeStatus(w, http.StatusInternalServerError, err)
	}
}
