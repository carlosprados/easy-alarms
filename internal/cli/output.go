package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"easy-alarms/internal/control"
)

// emit prints v as indented JSON when --json is set, otherwise runs human.
func (c *cli) emit(v any, human func()) {
	if c.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
		return
	}
	human()
}

// daysHuman renders a day list as a keyword when possible, else a comma list.
func daysHuman(days []string) string {
	switch len(days) {
	case 0:
		return "one-shot"
	case 7:
		return "daily"
	}
	set := map[string]bool{}
	for _, d := range days {
		set[d] = true
	}
	if len(days) == 5 && set["mon"] && set["tue"] && set["wed"] && set["thu"] && set["fri"] {
		return "weekdays"
	}
	if len(days) == 2 && set["sat"] && set["sun"] {
		return "weekend"
	}
	return strings.Join(days, ",")
}

// alarmLine renders one alarm/timer as a single human-readable line.
func alarmLine(d control.AlarmDTO) string {
	label := d.Label
	if label == "" {
		label = "(no label)"
	}
	var b strings.Builder
	if d.Kind == "timer" {
		fmt.Fprintf(&b, "⏱  %-24s %-8s %s", label, d.Duration, d.State)
		if d.Remaining != "" && d.State != "idle" {
			fmt.Fprintf(&b, " (%s left)", d.Remaining)
		}
	} else {
		enabled := "off"
		if d.Enabled {
			enabled = "on "
		}
		fmt.Fprintf(&b, "⏰  %-24s %s  [%-8s] %s", label, d.Time, daysHuman(d.Days), enabled)
	}
	switch {
	case d.Ringing:
		b.WriteString("  🔔 RINGING")
	case d.Snoozed && d.NextIn != "":
		fmt.Fprintf(&b, "  💤 snoozed, rings in %s", d.NextIn)
	case d.NextIn != "":
		fmt.Fprintf(&b, "  → in %s", d.NextIn)
	}
	fmt.Fprintf(&b, "   #%s", d.ID)
	return b.String()
}

func printfln(format string, a ...any) { fmt.Printf(format+"\n", a...) }

func printAlarm(d control.AlarmDTO) { fmt.Println(alarmLine(d)) }

func printList(list []control.AlarmDTO) {
	if len(list) == 0 {
		fmt.Println("No alarms.")
		return
	}
	for _, d := range list {
		fmt.Println(alarmLine(d))
	}
}
