package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

// ErrNotRunning means the control socket could not be reached, i.e. the GUI app
// is not running.
var ErrNotRunning = errors.New("easy-alarms is not running")

// APIError is a non-2xx response from the control server. Code carries the HTTP
// status so callers can map it to an exit code.
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string { return e.Message }

// Client talks to the control server over its Unix socket.
type Client struct {
	socket string
	http   *http.Client
}

// NewClient builds a client for the given socket path.
func NewClient(socket string) *Client {
	return &Client{
		socket: socket,
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", socket)
				},
			},
		},
	}
}

// SocketPath reports the socket this client dials.
func (c *Client) SocketPath() string { return c.socket }

func (c *Client) do(method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "http://easy-alarms"+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		// A dial failure on the socket means the app is not running.
		return fmt.Errorf("%w (socket %s): start the app first", ErrNotRunning, c.socket)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var er ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&er)
		msg := er.Error
		if msg == "" {
			msg = resp.Status
		}
		return &APIError{Code: resp.StatusCode, Message: msg}
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Status returns the whole observable app state.
func (c *Client) Status() (StatusDTO, error) {
	var s StatusDTO
	err := c.do(http.MethodGet, "/status", nil, &s)
	return s, err
}

// List returns all alarms, optionally filtered by kind ("clock"|"timer"|"").
func (c *Client) List(kind string) ([]AlarmDTO, error) {
	path := "/alarms"
	if kind != "" {
		path += "?kind=" + url.QueryEscape(kind)
	}
	var out []AlarmDTO
	err := c.do(http.MethodGet, path, nil, &out)
	return out, err
}

// Get returns a single alarm by ID.
func (c *Client) Get(id string) (AlarmDTO, error) {
	var d AlarmDTO
	err := c.do(http.MethodGet, "/alarms/"+id, nil, &d)
	return d, err
}

// CreateAlarm creates a clock alarm.
func (c *Client) CreateAlarm(req CreateAlarmRequest) (AlarmDTO, error) {
	var d AlarmDTO
	err := c.do(http.MethodPost, "/alarms", req, &d)
	return d, err
}

// CreateTimer creates a timer.
func (c *Client) CreateTimer(req CreateTimerRequest) (AlarmDTO, error) {
	var d AlarmDTO
	err := c.do(http.MethodPost, "/timers", req, &d)
	return d, err
}

// Update applies a partial update to an alarm or timer.
func (c *Client) Update(id string, req UpdateAlarmRequest) (AlarmDTO, error) {
	var d AlarmDTO
	err := c.do(http.MethodPatch, "/alarms/"+id, req, &d)
	return d, err
}

// Delete removes an alarm or timer.
func (c *Client) Delete(id string) error {
	return c.do(http.MethodDelete, "/alarms/"+id, nil, nil)
}

// SetEnabled enables or disables a clock alarm.
func (c *Client) SetEnabled(id string, on bool) (AlarmDTO, error) {
	action := "disable"
	if on {
		action = "enable"
	}
	var d AlarmDTO
	err := c.do(http.MethodPost, "/alarms/"+id+"/"+action, nil, &d)
	return d, err
}

// TimerOp runs a timer lifecycle action (start|pause|resume|stop).
func (c *Client) TimerOp(id string, op TimerOp) (AlarmDTO, error) {
	var d AlarmDTO
	err := c.do(http.MethodPost, "/timers/"+id+"/"+string(op), nil, &d)
	return d, err
}

// Snooze snoozes a ringing alarm by the given duration ("" = 5m default).
func (c *Client) Snooze(id, forDur string) error {
	return c.do(http.MethodPost, "/alarms/"+id+"/snooze", SnoozeRequest{For: forDur}, nil)
}

// Dismiss stops a ringing alarm.
func (c *Client) Dismiss(id string) error {
	return c.do(http.MethodPost, "/alarms/"+id+"/dismiss", nil, nil)
}

// Settings returns the read-only app settings.
func (c *Client) Settings() (SettingsDTO, error) {
	var s SettingsDTO
	err := c.do(http.MethodGet, "/settings", nil, &s)
	return s, err
}
