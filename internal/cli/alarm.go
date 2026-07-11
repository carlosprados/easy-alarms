package cli

import (
	"github.com/spf13/cobra"

	"easy-alarms/internal/control"
)

func (c *cli) newAlarmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alarm",
		Short: "Manage clock alarms",
	}
	cmd.AddCommand(
		c.newAlarmCreateCmd(),
		c.newAlarmShowCmd(),
		c.newAlarmEditCmd(),
		c.newAlarmDeleteCmd(),
		c.newAlarmEnableCmd(true),
		c.newAlarmEnableCmd(false),
	)
	return cmd
}

func (c *cli) newAlarmCreateCmd() *cobra.Command {
	var req control.CreateAlarmRequest
	cmd := &cobra.Command{
		Use:   "create --at HH:MM [flags]",
		Short: "Create a clock alarm (enabled immediately)",
		Long: `Create a clock alarm.

--days accepts a keyword (daily, weekdays, weekend) or a comma-separated list of
day names in English or Spanish (mon,tue,... or lun,mar,...). Omit it for a
one-shot alarm that rings the next time HH:MM comes around.`,
		Example: `  alarmctl alarm create --at 07:30 --days weekdays --label "Work"
  alarmctl alarm create --at 09:00 --days lun,mié,vie
  alarmctl alarm create --at 22:15   # one-shot`,
		RunE: func(*cobra.Command, []string) error {
			d, err := c.client.CreateAlarm(req)
			if err != nil {
				return err
			}
			c.emit(d, func() { printAlarm(d) })
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&req.At, "at", "", "time of day, HH:MM (24h) [required]")
	f.StringVar(&req.Days, "days", "", "daily|weekdays|weekend or a day list (mon,tue / lun,mar)")
	f.StringVar(&req.Label, "label", "", "optional label")
	f.StringVar(&req.Sound, "sound", "", "path to a sound file (empty = built-in tone)")
	cmd.MarkFlagRequired("at")
	return cmd
}

func (c *cli) newAlarmShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show ID",
		Short:   "Show one alarm or timer by ID",
		Args:    cobra.ExactArgs(1),
		Example: "  alarmctl alarm show 1783789071827385956",
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := c.client.Get(args[0])
			if err != nil {
				return err
			}
			c.emit(d, func() { printAlarm(d) })
			return nil
		},
	}
}

func (c *cli) newAlarmEditCmd() *cobra.Command {
	var (
		req                         control.UpdateAlarmRequest
		label, at, days, dur, sound string
	)
	cmd := &cobra.Command{
		Use:   "edit ID [flags]",
		Short: "Change fields of an existing alarm or timer",
		Long: `Change one or more fields of an alarm or timer. Only the flags you pass are
modified. Changing a running timer's duration restarts its countdown.`,
		Args: cobra.ExactArgs(1),
		Example: `  alarmctl alarm edit 178... --label "Gym"
  alarmctl alarm edit 178... --at 08:00 --days weekend
  alarmctl alarm edit 178... --duration 30m`,
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Flags()
			if f.Changed("label") {
				req.Label = &label
			}
			if f.Changed("at") {
				req.At = &at
			}
			if f.Changed("days") {
				req.Days = &days
			}
			if f.Changed("duration") {
				req.Duration = &dur
			}
			if f.Changed("sound") {
				req.Sound = &sound
			}
			d, err := c.client.Update(args[0], req)
			if err != nil {
				return err
			}
			c.emit(d, func() { printAlarm(d) })
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&label, "label", "", "new label")
	f.StringVar(&at, "at", "", "new time, HH:MM (clock alarms)")
	f.StringVar(&days, "days", "", "new repeat days (clock alarms)")
	f.StringVar(&dur, "duration", "", "new duration, e.g. 25m (timers)")
	f.StringVar(&sound, "sound", "", "new sound file path (empty = built-in tone)")
	return cmd
}

func (c *cli) newAlarmDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete ID",
		Aliases: []string{"rm"},
		Short:   "Delete an alarm or timer",
		Args:    cobra.ExactArgs(1),
		Example: "  alarmctl alarm delete 1783789071827385956",
		RunE: func(_ *cobra.Command, args []string) error {
			if err := c.client.Delete(args[0]); err != nil {
				return err
			}
			if !c.json {
				printfln("Deleted %s", args[0])
			}
			return nil
		},
	}
}

func (c *cli) newAlarmEnableCmd(on bool) *cobra.Command {
	use, short := "disable ID", "Disable a clock alarm"
	if on {
		use, short = "enable ID", "Enable a clock alarm"
	}
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			d, err := c.client.SetEnabled(args[0], on)
			if err != nil {
				return err
			}
			c.emit(d, func() { printAlarm(d) })
			return nil
		},
	}
}
