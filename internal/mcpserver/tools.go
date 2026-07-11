package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"easy-alarms/internal/control"
)

// okResult is the structured output of tools that don't return an alarm.
type okResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

func (h *handlers) registerTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_status",
		Description: "Get the full state of easy-alarms: version, the currently ringing alarms, the soonest upcoming alarm, and the complete list of alarms and timers.",
	}, h.getStatus)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_alarms",
		Description: "List all alarms and timers, optionally filtered by kind.",
	}, h.listAlarms)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_alarm",
		Description: "Create a clock alarm that rings at a time of day. It is enabled immediately. Use days for repetition; omit it for a one-shot alarm that rings the next time the given time comes around.",
	}, h.createAlarm)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_timer",
		Description: "Create a countdown timer. By default it starts running immediately.",
	}, h.createTimer)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_alarm",
		Description: "Change fields of an existing alarm or timer by ID. Only the fields you provide are modified.",
	}, h.updateAlarm)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_alarm",
		Description: "Delete an alarm or timer by ID.",
	}, h.deleteAlarm)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_enabled",
		Description: "Enable or disable a clock alarm by ID. (Timers use timer_control instead.)",
	}, h.setEnabled)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "timer_control",
		Description: "Control a timer's lifecycle by ID: start an idle timer, pause a running one, resume a paused one, or stop it back to idle.",
	}, h.timerControl)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "snooze",
		Description: "Snooze a ringing alarm. If id is omitted and exactly one alarm is ringing, that one is snoozed.",
	}, h.snooze)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "dismiss",
		Description: "Stop a ringing alarm. If id is omitted, every ringing alarm is dismissed.",
	}, h.dismiss)
}

// --- tool inputs ---

type emptyIn struct{}

type listIn struct {
	Kind string `json:"kind,omitempty" jsonschema:"filter by kind: clock or timer; empty for all"`
}

type createAlarmIn struct {
	At    string `json:"at" jsonschema:"time of day in 24h HH:MM format, e.g. 07:30"`
	Days  string `json:"days,omitempty" jsonschema:"repetition: daily, weekdays, weekend, or a comma list of day names in English or Spanish (mon,tue / lun,mar); empty for a one-shot"`
	Label string `json:"label,omitempty" jsonschema:"optional human label"`
	Sound string `json:"sound,omitempty" jsonschema:"optional path to a sound file; empty uses the built-in tone"`
}

type createTimerIn struct {
	Duration string `json:"duration" jsonschema:"countdown length as a Go duration, e.g. 10m, 1h30m, 90s"`
	Label    string `json:"label,omitempty" jsonschema:"optional human label"`
	Sound    string `json:"sound,omitempty" jsonschema:"optional path to a sound file; empty uses the built-in tone"`
	Start    *bool  `json:"start,omitempty" jsonschema:"whether to start the timer immediately; defaults to true"`
}

type updateAlarmIn struct {
	ID       string `json:"id" jsonschema:"the alarm or timer ID"`
	At       string `json:"at,omitempty" jsonschema:"new time HH:MM (clock alarms only)"`
	Days     string `json:"days,omitempty" jsonschema:"new repetition (clock alarms only)"`
	Duration string `json:"duration,omitempty" jsonschema:"new duration, e.g. 25m (timers only); restarts a running timer"`
	Label    string `json:"label,omitempty" jsonschema:"new label"`
	Sound    string `json:"sound,omitempty" jsonschema:"new sound file path; empty uses the built-in tone"`
}

type idIn struct {
	ID string `json:"id" jsonschema:"the alarm or timer ID"`
}

type setEnabledIn struct {
	ID      string `json:"id" jsonschema:"the clock alarm ID"`
	Enabled bool   `json:"enabled" jsonschema:"true to enable, false to disable"`
}

type timerControlIn struct {
	ID string `json:"id" jsonschema:"the timer ID"`
	Op string `json:"op" jsonschema:"one of: start, pause, resume, stop"`
}

type snoozeIn struct {
	ID  string `json:"id,omitempty" jsonschema:"the ringing alarm ID; optional when exactly one alarm is ringing"`
	For string `json:"for,omitempty" jsonschema:"snooze length, e.g. 5m, 10m; defaults to 5m"`
}

type dismissIn struct {
	ID string `json:"id,omitempty" jsonschema:"the ringing alarm ID; optional to dismiss all ringing alarms"`
}

// --- tool handlers ---

func (h *handlers) getStatus(_ context.Context, _ *mcp.CallToolRequest, _ emptyIn) (*mcp.CallToolResult, control.StatusDTO, error) {
	st, err := h.client.Status()
	return nil, st, err
}

func (h *handlers) listAlarms(_ context.Context, _ *mcp.CallToolRequest, in listIn) (*mcp.CallToolResult, control.StatusDTO, error) {
	list, err := h.client.List(in.Kind)
	// Wrap in StatusDTO-free shape via a status-like object is overkill; return
	// a bare status carrying only alarms.
	return nil, control.StatusDTO{Alarms: list}, err
}

func (h *handlers) createAlarm(_ context.Context, _ *mcp.CallToolRequest, in createAlarmIn) (*mcp.CallToolResult, control.AlarmDTO, error) {
	d, err := h.client.CreateAlarm(control.CreateAlarmRequest{At: in.At, Days: in.Days, Label: in.Label, Sound: in.Sound})
	return nil, d, err
}

func (h *handlers) createTimer(_ context.Context, _ *mcp.CallToolRequest, in createTimerIn) (*mcp.CallToolResult, control.AlarmDTO, error) {
	d, err := h.client.CreateTimer(control.CreateTimerRequest{Duration: in.Duration, Label: in.Label, Sound: in.Sound, Start: in.Start})
	return nil, d, err
}

func (h *handlers) updateAlarm(_ context.Context, _ *mcp.CallToolRequest, in updateAlarmIn) (*mcp.CallToolResult, control.AlarmDTO, error) {
	var req control.UpdateAlarmRequest
	if in.At != "" {
		req.At = &in.At
	}
	if in.Days != "" {
		req.Days = &in.Days
	}
	if in.Duration != "" {
		req.Duration = &in.Duration
	}
	if in.Label != "" {
		req.Label = &in.Label
	}
	if in.Sound != "" {
		req.Sound = &in.Sound
	}
	d, err := h.client.Update(in.ID, req)
	return nil, d, err
}

func (h *handlers) deleteAlarm(_ context.Context, _ *mcp.CallToolRequest, in idIn) (*mcp.CallToolResult, okResult, error) {
	if err := h.client.Delete(in.ID); err != nil {
		return nil, okResult{}, err
	}
	return nil, okResult{OK: true, Message: "deleted " + in.ID}, nil
}

func (h *handlers) setEnabled(_ context.Context, _ *mcp.CallToolRequest, in setEnabledIn) (*mcp.CallToolResult, control.AlarmDTO, error) {
	d, err := h.client.SetEnabled(in.ID, in.Enabled)
	return nil, d, err
}

func (h *handlers) timerControl(_ context.Context, _ *mcp.CallToolRequest, in timerControlIn) (*mcp.CallToolResult, control.AlarmDTO, error) {
	d, err := h.client.TimerOp(in.ID, control.TimerOp(in.Op))
	return nil, d, err
}

func (h *handlers) snooze(_ context.Context, _ *mcp.CallToolRequest, in snoozeIn) (*mcp.CallToolResult, okResult, error) {
	id := in.ID
	if id == "" {
		resolved, err := h.soleRinging()
		if err != nil {
			return nil, okResult{}, err
		}
		id = resolved
	}
	if err := h.client.Snooze(id, in.For); err != nil {
		return nil, okResult{}, err
	}
	return nil, okResult{OK: true, Message: "snoozed " + id}, nil
}

func (h *handlers) dismiss(_ context.Context, _ *mcp.CallToolRequest, in dismissIn) (*mcp.CallToolResult, okResult, error) {
	if in.ID != "" {
		if err := h.client.Dismiss(in.ID); err != nil {
			return nil, okResult{}, err
		}
		return nil, okResult{OK: true, Message: "dismissed " + in.ID}, nil
	}
	st, err := h.client.Status()
	if err != nil {
		return nil, okResult{}, err
	}
	for _, id := range st.Ringing {
		if err := h.client.Dismiss(id); err != nil {
			return nil, okResult{}, err
		}
	}
	return nil, okResult{OK: true, Message: "dismissed all ringing alarms"}, nil
}

// soleRinging returns the only ringing alarm ID, or an error if zero or many.
func (h *handlers) soleRinging() (string, error) {
	st, err := h.client.Status()
	if err != nil {
		return "", err
	}
	switch len(st.Ringing) {
	case 1:
		return st.Ringing[0], nil
	case 0:
		return "", errNoRinging
	default:
		return "", errManyRinging
	}
}
