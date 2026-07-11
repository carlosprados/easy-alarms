package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *handlers) registerPrompts(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "wake-me-up",
		Description: "Set a morning wake-up alarm and confirm when it will ring.",
		Arguments: []*mcp.PromptArgument{
			{Name: "time", Description: "wake-up time, HH:MM (24h)", Required: true},
			{Name: "days", Description: "daily, weekdays, weekend, or a day list; empty for a one-shot", Required: false},
			{Name: "label", Description: "optional label", Required: false},
		},
	}, h.wakeMeUp)

	s.AddPrompt(&mcp.Prompt{
		Name:        "pomodoro",
		Description: "Run a Pomodoro focus session using timers.",
		Arguments: []*mcp.PromptArgument{
			{Name: "work", Description: "work interval, e.g. 25m (default 25m)", Required: false},
			{Name: "break", Description: "break interval, e.g. 5m (default 5m)", Required: false},
			{Name: "cycles", Description: "number of work/break cycles (default 4)", Required: false},
		},
	}, h.pomodoro)
}

func userText(text string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{{
			Role:    mcp.Role("user"),
			Content: &mcp.TextContent{Text: text},
		}},
	}
}

func arg(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

func (h *handlers) wakeMeUp(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	a := req.Params.Arguments
	t := arg(a, "time", "07:00")
	days := arg(a, "days", "weekdays")
	label := arg(a, "label", "Wake up")
	return userText(fmt.Sprintf(
		"Create a clock alarm with the create_alarm tool: time %q, days %q, label %q. "+
			"Then call get_status and tell me exactly when it will next ring, in plain language.",
		t, days, label)), nil
}

func (h *handlers) pomodoro(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	a := req.Params.Arguments
	work := arg(a, "work", "25m")
	brk := arg(a, "break", "5m")
	cycles := arg(a, "cycles", "4")
	return userText(fmt.Sprintf(
		"Run a Pomodoro session of %s cycles: each cycle is a %s work timer followed by a %s break timer. "+
			"Use the create_timer tool to start each interval, and poll get_status until a timer is no longer running "+
			"before starting the next one. Label the timers clearly (e.g. \"Pomodoro work 1/%s\"). "+
			"Tell me when the whole session is done.",
		cycles, work, brk, cycles)), nil
}
