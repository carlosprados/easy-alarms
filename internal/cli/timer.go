package cli

import (
	"github.com/spf13/cobra"

	"easy-alarms/internal/control"
)

func (c *cli) newTimerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timer",
		Short: "Manage countdown timers",
	}
	cmd.AddCommand(
		c.newTimerCreateCmd(),
		c.newTimerOpCmd(control.OpStart, "start ID", "Start an idle timer"),
		c.newTimerOpCmd(control.OpPause, "pause ID", "Pause a running timer"),
		c.newTimerOpCmd(control.OpResume, "resume ID", "Resume a paused timer"),
		c.newTimerOpCmd(control.OpStop, "stop ID", "Stop a timer (back to idle)"),
		c.newAlarmDeleteCmd(), // `timer delete` is the same delete as alarms
	)
	return cmd
}

func (c *cli) newTimerCreateCmd() *cobra.Command {
	var (
		req    control.CreateTimerRequest
		paused bool
	)
	cmd := &cobra.Command{
		Use:   "create --duration D [flags]",
		Short: "Create a timer (starts running unless --paused)",
		Example: `  alarmctl timer create --duration 25m --label Pomodoro
  alarmctl timer create --duration 1h30m --paused`,
		RunE: func(*cobra.Command, []string) error {
			start := !paused
			req.Start = &start
			d, err := c.client.CreateTimer(req)
			if err != nil {
				return err
			}
			c.emit(d, func() { printAlarm(d) })
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&req.Duration, "duration", "", "countdown length, e.g. 10m, 1h30m, 90s [required]")
	f.StringVar(&req.Label, "label", "", "optional label")
	f.StringVar(&req.Sound, "sound", "", "path to a sound file (empty = built-in tone)")
	f.BoolVar(&paused, "paused", false, "create the timer without starting it")
	cmd.MarkFlagRequired("duration")
	return cmd
}

func (c *cli) newTimerOpCmd(op control.TimerOp, use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := c.client.TimerOp(args[0], op)
			if err != nil {
				return err
			}
			c.emit(d, func() { printAlarm(d) })
			return nil
		},
	}
}
