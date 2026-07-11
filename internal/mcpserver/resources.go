package mcpserver

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	errNoRinging   = errors.New("no alarm is ringing")
	errManyRinging = errors.New("several alarms are ringing; specify an id")
)

const alarmsURI = "alarms://all"

func (h *handlers) registerResources(s *mcp.Server) {
	s.AddResource(&mcp.Resource{
		URI:         alarmsURI,
		Name:        "All alarms",
		Description: "The full easy-alarms state as JSON: version, ringing alarms, the next alarm, and every alarm and timer.",
		MIMEType:    "application/json",
	}, h.readAlarms)
}

func (h *handlers) readAlarms(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	st, err := h.client.Status()
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}
