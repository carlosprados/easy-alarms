// Package mcpserver exposes easy-alarms over the Model Context Protocol so a
// local AI can create and manage alarms and timers. Every tool delegates to
// the control client, connecting lazily, so the MCP server can be registered
// while the GUI app is down; tools then return a clear "not running" error the
// model can relay.
package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"easy-alarms/internal/control"
)

type handlers struct {
	client *control.Client
}

// Run starts the MCP server on stdio and blocks until ctx is cancelled or the
// client disconnects.
func Run(ctx context.Context, client *control.Client, version string) error {
	h := &handlers{client: client}
	s := mcp.NewServer(&mcp.Implementation{Name: "easy-alarms", Version: version}, nil)

	h.registerTools(s)
	h.registerResources(s)
	h.registerPrompts(s)

	return s.Run(ctx, &mcp.StdioTransport{})
}
