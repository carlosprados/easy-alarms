// Package cli implements the alarmctl command tree: a thin, self-documenting
// Cobra client over the easy-alarms control socket.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"easy-alarms/internal/control"
)

type cli struct {
	socket  string
	json    bool
	version string
	client  *control.Client
}

// Execute builds and runs the root command, returning a process exit code.
func Execute(version string) int {
	c := &cli{version: version}
	root := c.newRoot()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return exitCode(err)
	}
	return 0
}

// exitCode maps an error to a documented exit code:
//
//	1 validation / conflict (bad input, wrong kind, invalid state)
//	2 cannot connect (the app is not running)
//	3 not found (unknown alarm ID)
func exitCode(err error) int {
	if errors.Is(err, control.ErrNotRunning) {
		return 2
	}
	var ae *control.APIError
	if errors.As(err, &ae) && ae.Code == 404 {
		return 3
	}
	return 1
}

func (c *cli) newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "alarmctl",
		Short: "Control the running easy-alarms app from the command line",
		Long: `alarmctl controls a running easy-alarms GUI over its local control socket.

The easy-alarms app must be running; alarmctl does not launch it. Every command
supports --json for scripting.

Exit codes:
  0  success
  1  invalid input or conflicting request
  2  the app is not running (cannot connect to the socket)
  3  alarm not found`,
		SilenceErrors:     true,
		SilenceUsage:      true,
		PersistentPreRunE: func(*cobra.Command, []string) error { c.client = control.NewClient(c.socket); return nil },
	}
	root.PersistentFlags().StringVar(&c.socket, "socket", control.SocketPath(), "path to the control socket")
	root.PersistentFlags().BoolVar(&c.json, "json", false, "output raw JSON instead of human-readable text")

	root.AddCommand(
		c.newStatusCmd(),
		c.newListCmd(),
		c.newAlarmCmd(),
		c.newTimerCmd(),
		c.newSnoozeCmd(),
		c.newDismissCmd(),
		c.newMCPCmd(),
		c.newVersionCmd(),
	)
	return root
}

func (c *cli) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the alarmctl version and the running app's version",
		RunE: func(*cobra.Command, []string) error {
			fmt.Printf("alarmctl %s\n", c.version)
			if st, err := c.client.Status(); err == nil {
				fmt.Printf("easy-alarms %s (running)\n", st.Version)
			} else {
				fmt.Println("easy-alarms (not running)")
			}
			return nil
		},
	}
}
