package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (c *cli) newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Show the next alarm, anything ringing, and a summary",
		Example: "  alarmctl status\n  alarmctl status --json",
		RunE: func(*cobra.Command, []string) error {
			st, err := c.client.Status()
			if err != nil {
				return err
			}
			c.emit(st, func() {
				fmt.Printf("easy-alarms %s — %d alarm(s)\n", st.Version, len(st.Alarms))
				if len(st.Ringing) > 0 {
					fmt.Printf("🔔 Ringing now: %d\n", len(st.Ringing))
					for _, d := range st.Alarms {
						if d.Ringing {
							printAlarm(d)
						}
					}
				}
				if st.Next != nil {
					fmt.Print("Next: ")
					printAlarm(*st.Next)
				} else {
					fmt.Println("Next: nothing scheduled")
				}
			})
			return nil
		},
	}
}

func (c *cli) newListCmd() *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all alarms and timers",
		Example: "  alarmctl list\n  alarmctl list --kind timer\n  alarmctl list --json",
		RunE: func(*cobra.Command, []string) error {
			if kind != "" && kind != "clock" && kind != "timer" {
				return fmt.Errorf("--kind must be clock or timer")
			}
			list, err := c.client.List(kind)
			if err != nil {
				return err
			}
			c.emit(list, func() { printList(list) })
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "filter by kind: clock or timer")
	return cmd
}
