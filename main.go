package main

import (
	"flag"
	"log"

	"fyne.io/fyne/v2/app"

	"easy-alarms/internal/alarm"
	"easy-alarms/internal/audio"
	"easy-alarms/internal/store"
	"easy-alarms/internal/ui"
)

func main() {
	hidden := flag.Bool("hidden", false, "start minimized to the system tray")
	noTray := flag.Bool("no-tray", false, "disable the system tray (diagnostic)")
	flag.Parse()

	st, err := store.Load()
	if err != nil {
		log.Fatalf("loading alarms: %v", err)
	}

	a := app.NewWithID("me.enredando.easy-alarms")
	player := audio.NewPlayer()
	sched := alarm.NewScheduler(func() []*alarm.Alarm { return st.List() })

	mainUI := ui.New(a, st, sched, player)
	mainUI.TrayEnabled = !*noTray
	sched.OnRing = mainUI.Ring
	sched.Start()

	mainUI.Run(*hidden)
}
