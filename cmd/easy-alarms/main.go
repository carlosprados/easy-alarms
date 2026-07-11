package main

import (
	"flag"
	"log"
	"os"

	"fyne.io/fyne/v2/app"

	"easy-alarms/internal/alarm"
	"easy-alarms/internal/audio"
	"easy-alarms/internal/control"
	"easy-alarms/internal/store"
	"easy-alarms/internal/ui"
)

// version is injected via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	hidden := flag.Bool("hidden", false, "start minimized to the system tray")
	noTray := flag.Bool("no-tray", false, "disable the system tray (diagnostic)")
	noControl := flag.Bool("no-control", false, "disable the alarmctl control socket")
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

	if !*noControl {
		srv := control.NewServer(mainUI.Controller(), version)
		// The listener must only start once the Fyne event loop is running:
		// the control handlers use fyne.DoAndWait, which deadlocks if invoked
		// before the loop can drain the queue. SetOnStarted fires from inside
		// the running loop, so the first accepted request is always safe.
		a.Lifecycle().SetOnStarted(func() {
			go func() {
				ln, err := control.Listen()
				if err != nil {
					log.Printf("control socket disabled: %v", err)
					return
				}
				log.Printf("control socket listening on %s", control.SocketPath())
				if err := srv.Serve(ln); err != nil {
					log.Printf("control server stopped: %v", err)
				}
			}()
		})
		// Close the listener before the loop dies. An in-flight handler still
		// blocked in DoAndWait during shutdown would hang, but only in an
		// already-exiting process, which is acceptable.
		a.Lifecycle().SetOnStopped(func() {
			_ = srv.Close()
			_ = os.Remove(control.SocketPath())
		})
	}

	mainUI.Run(*hidden)
}
