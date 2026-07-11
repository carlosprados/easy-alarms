package cli

import (
	"github.com/spf13/cobra"

	"easy-alarms/internal/mcpserver"
)

func (c *cli) newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP server (stdio) exposing easy-alarms to a local AI",
		Long: `Run a Model Context Protocol server on stdin/stdout. It exposes tools
(create_alarm, create_timer, list_alarms, timer_control, snooze, dismiss, ...),
a resource (alarms://all) and prompts (wake-me-up, pomodoro) that let a local AI
manage alarms and timers.

Register it with Claude Code:

  claude mcp add easy-alarms -- alarmctl mcp

The easy-alarms app must be running for the tools to succeed; if it is not, they
return a clear "not running" error.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpserver.Run(cmd.Context(), c.client, c.version)
		},
	}
}
