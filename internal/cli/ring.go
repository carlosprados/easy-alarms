package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *cli) newSnoozeCmd() *cobra.Command {
	var forDur string
	cmd := &cobra.Command{
		Use:   "snooze [ID]",
		Short: "Snooze a ringing alarm",
		Long: `Snooze a ringing alarm by --for (default 5m). The ID is optional when
exactly one alarm is ringing.`,
		Args:    cobra.MaximumNArgs(1),
		Example: "  alarmctl snooze\n  alarmctl snooze --for 10m\n  alarmctl snooze 178... --for 15m",
		RunE: func(_ *cobra.Command, args []string) error {
			id, err := c.resolveRinging(args)
			if err != nil {
				return err
			}
			if err := c.client.Snooze(id, forDur); err != nil {
				return err
			}
			if !c.json {
				printfln("Snoozed %s", id)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&forDur, "for", "", "snooze length, e.g. 5m, 10m (default 5m)")
	return cmd
}

func (c *cli) newDismissCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dismiss [ID]",
		Short: "Stop a ringing alarm",
		Long: `Stop a ringing alarm. With no ID, dismisses every alarm currently
ringing.`,
		Args:    cobra.MaximumNArgs(1),
		Example: "  alarmctl dismiss\n  alarmctl dismiss 1783789071837342389",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				if err := c.client.Dismiss(args[0]); err != nil {
					return err
				}
				if !c.json {
					printfln("Dismissed %s", args[0])
				}
				return nil
			}
			// No ID: dismiss all ringing.
			st, err := c.client.Status()
			if err != nil {
				return err
			}
			if len(st.Ringing) == 0 {
				return fmt.Errorf("no alarm is ringing")
			}
			for _, id := range st.Ringing {
				if err := c.client.Dismiss(id); err != nil {
					return err
				}
				if !c.json {
					printfln("Dismissed %s", id)
				}
			}
			return nil
		},
	}
}

// resolveRinging returns the target alarm ID: the given arg, or the sole
// ringing alarm when no arg is passed.
func (c *cli) resolveRinging(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	st, err := c.client.Status()
	if err != nil {
		return "", err
	}
	switch len(st.Ringing) {
	case 0:
		return "", fmt.Errorf("no alarm is ringing")
	case 1:
		return st.Ringing[0], nil
	default:
		return "", fmt.Errorf("%d alarms are ringing; specify an ID: %v", len(st.Ringing), st.Ringing)
	}
}
